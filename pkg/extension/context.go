package extension

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-getter"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	extensionv1 "github.com/argoproj/argocd-extensions/api/v1alpha1"
	"github.com/argoproj/argocd-extensions/pkg/git"
)

const (
	ResourcesDir              = "resources"
	ResourceOverrideConfigMap = "argocd-resource-override-cm"
	fileTrackerFileName       = ".fileTracker"
)

type extensionName string

type extensionContext struct {
	client.Client
	name            extensionName
	outputPath      string
	snapshotPath    string
	fileTrackerPath string
	extension       *extensionv1.ArgoCDExtension
}

type sourcesSnapshot struct {
	Revisions []string `json:"revisions"`
	Files     []string `json:"files"`
}

func (s *sourcesSnapshot) shouldDownload(revisions []string) string {
	if len(s.Revisions) == 0 {
		return "Sources has not been downloaded yet"
	}
	if len(s.Revisions) != len(revisions) {
		return fmt.Sprintf("Sources number has changed from %d to %d", len(s.Revisions), len(revisions))
	}
	for i := range revisions {
		if s.Revisions[i] != revisions[i] {
			return fmt.Sprintf("Source #%d has changed from %s to %s", i, s.Revisions[i], revisions[i])
		}
	}

	return ""
}

func (c *extensionContext) deleteFiles(tracker *fileTracker, files []string) error {
	for _, file := range files {
		if tracker.isTracked(file) && !tracker.isOwner(file, c.name) {
			return fmt.Errorf("cannot delete file \"%s\" since it is owned by \"%s\"", file, c.name)
		}
		if err := os.Remove(file); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		tracker.clearMetadata(file)
	}
	return nil
}

func NewExtensionContext(extension *extensionv1.ArgoCDExtension, client client.Client, outputPath string) *extensionContext {
	return &extensionContext{
		Client:          client,
		name:            extensionName(extension.Name),
		extension:       extension,
		outputPath:      outputPath,
		snapshotPath:    path.Join(outputPath, fmt.Sprintf(".%s.snapshot", extension.Name)),
		fileTrackerPath: path.Join(outputPath, fileTrackerFileName),
	}
}

func (c *extensionContext) buildResourceOverrideConfigMap(resourceOverrides map[string]*v1alpha1.ResourceOverride) (*v1.ConfigMap, error) {
	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceOverrideConfigMap,
			Namespace: c.extension.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "argocd",
			},
		},
		Data: make(map[string]string),
	}

	bytes, err := yaml.Marshal(resourceOverrides)
	if err != nil {
		return nil, err
	}
	configMap.Data["resources"] = string(bytes)

	return &configMap, nil
}

func (c *extensionContext) setResourceOverrideConfigMap(ctx context.Context, cm *v1.ConfigMap) error {
	err := c.Update(ctx, cm)
	if apierrors.IsNotFound(err) {
		return c.Create(ctx, cm)
	}
	return err
}

func (c *extensionContext) rebuildResourceOverrideConfigMap(ctx context.Context) error {
	// get resource overrides from the output directory
	// the output directory is shared by *all* extensions and is considered the source of truth
	resourceOverrides, err := c.getExtensionResourceOverrides()
	if err != nil {
		return fmt.Errorf("failed to get resource overrides from output directory: %v", err)
	}
	// builds a ConfigMap based on the given resource overrides
	configMap, err := c.buildResourceOverrideConfigMap(resourceOverrides)
	if err != nil {
		return fmt.Errorf("failed to build resource override ConfigMap: %v", err)
	}
	// creates or updates the argocd-resource-override-cm ConfigMap
	// argocd will pull resource customizations from this ConfigMap in additional to the ones defined in argocd-cm
	err = c.setResourceOverrideConfigMap(ctx, configMap)
	if err != nil {
		return fmt.Errorf("failed to create/update resource override ConfigMap: %v", err)
	}
	return nil
}

func (c *extensionContext) GetSecret(ctx context.Context, key extensionv1.NamespacedName) (v1.Secret, error) {
	var secret v1.Secret
	err := c.Get(ctx, types.NamespacedName{
		Namespace: key.Namespace,
		Name:      key.Name,
	}, &secret)
	return secret, err
}

// Process downloads extension files
func (c *extensionContext) Process(ctx context.Context) error {
	log := k8slog.FromContext(ctx)

	revisions, err := c.resolveRevisions(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve sources revisions: %v", err)
	}

	// load file tracker
	tracker, err := c.loadFileTracker()
	if err != nil {
		return fmt.Errorf("failed to load file tracker: %v", err)
	}

	// try to load previous snapshot and check most recent revisions of all sources
	prev := c.loadSnapshot()

	reason := prev.shouldDownload(revisions)
	if reason == "" {
		log.Info("Sources already downloaded.")
		return nil
	} else {
		log.Info(fmt.Sprintf("%s, redownloading...", reason))
	}

	// download all extension files into temp directory
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("failed to create temp dir %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Error(err, "Failed to delete temp directory")
		}
	}()

	if err := c.downloadTo(ctx, tempDir); err != nil {
		return fmt.Errorf("failed to download sources: %v", err)
	}

	// walk files and verify ownership if they are already tracked
	files, err := c.walkFiles(tempDir)
	if err != nil {
		return fmt.Errorf("failed to walk through files: %v", err)
	}
	for _, file := range files {
		// if the file is not tracked, then we don't need to check the owner
		if tracker.isTracked(file) {
			// if the file is tracked, then we need to make sure this file isn't already owned by a different extension
			if !tracker.isOwner(file, c.name) {
				return fmt.Errorf("file \"%s\" is already owned by \"%s\"", file, tracker.getOwner(file))
			}
		}
	}

	// delete all previously downloaded extension files
	if err := c.deleteFiles(tracker, files); err != nil {
		return fmt.Errorf("failed to clean %s: %v", c.outputPath, err)
	}

	// move downloaded files to the persistent extensions files location
	// track all files as being owned by this extension
	// and store list of files in the snapshot
	snapshot, err := c.moveSourceFiles(tracker, revisions, tempDir)
	if err != nil {
		return fmt.Errorf("failed to move source files: %v", err)
	}

	// rebuilds the ConfigMap since the contents of the output directory have changed
	err = c.rebuildResourceOverrideConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to rebuild resource override ConfigMap: %v", err)
	}

	// stores the latest file tracker
	if err := c.saveFileTracker(tracker); err != nil {
		return fmt.Errorf("failed to persist file tracker: %v", err)
	}
	// store snapshot in extensions directory
	if err := c.saveSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to persist snapshot: %v", err)
	}

	log.Info("Successfully downloaded all sources.")
	return nil
}

// ProcessDeletion deletes all previously downloaded files for the extension
func (c *extensionContext) ProcessDeletion(ctx context.Context) error {
	tracker, err := c.loadFileTracker()
	if err != nil {
		return fmt.Errorf("failed to load file tracker: %v", err)
	}

	snapshot := c.loadSnapshot()
	err = c.deleteFiles(tracker, snapshot.Files)
	if err != nil {
		return fmt.Errorf("failed to delete files: %v", err)
	}

	// rebuilds the ConfigMap since the contents of the output directory have changed
	err = c.rebuildResourceOverrideConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to rebuild resource override ConfigMap: %v", err)
	}

	return c.deleteSnapshot()
}

func getResourceOverrideForResourceDirectory(basePath, groupDirName string, resourceDirName string) (*v1alpha1.ResourceOverride, error) {
	dirPath := path.Join(basePath, ResourcesDir, groupDirName, resourceDirName)
	healthLua := path.Join(dirPath, "health.lua")

	healthScript := ""
	rawScript, err := os.ReadFile(healthLua)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		healthScript = string(rawScript)
	}

	return &v1alpha1.ResourceOverride{
		HealthLua: healthScript,
	}, nil
}

func getResourceOverridesForGroupDirectory(basePath string, groupDirName string) (map[string]*v1alpha1.ResourceOverride, error) {
	dirPath := path.Join(basePath, ResourcesDir, groupDirName)
	resourceDirs, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	resourceOverrideMap := make(map[string]*v1alpha1.ResourceOverride)
	for _, resourceDir := range resourceDirs {
		if !resourceDir.IsDir() {
			return nil, errors.New(fmt.Sprintf("extension path \"%s\" is not a directory", dirPath))
		}
		resourceOverride, err := getResourceOverrideForResourceDirectory(basePath, groupDirName, resourceDir.Name())
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s/%s", groupDirName, resourceDir.Name())
		if err != nil {
			return nil, err
		}
		resourceOverrideMap[key] = resourceOverride
	}
	return resourceOverrideMap, nil
}

func (c *extensionContext) getExtensionResourceOverrides() (map[string]*v1alpha1.ResourceOverride, error) {
	resourcesPath := path.Join(c.outputPath, ResourcesDir)
	groupDirs, err := os.ReadDir(resourcesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*v1alpha1.ResourceOverride), nil
		}
		return nil, err
	}

	resourceOverrideMap := make(map[string]*v1alpha1.ResourceOverride)
	for _, groupDir := range groupDirs {
		if !groupDir.IsDir() {
			return nil, errors.New(fmt.Sprintf("extension resource group \"%s\" is not a directory", groupDir.Name()))
		}
		groupResourceOverrideMap, err := getResourceOverridesForGroupDirectory(c.outputPath, groupDir.Name())
		if err != nil {
			return nil, err
		}
		for key, resourceOverride := range groupResourceOverrideMap {
			if _, exists := resourceOverrideMap[key]; exists {
				return nil, errors.New(fmt.Sprintf("resource override already defined for key \"%s\"", key))
			}
			resourceOverrideMap[key] = resourceOverride
		}
	}

	return resourceOverrideMap, nil
}

func (c *extensionContext) walkFiles(tempDir string) ([]string, error) {
	files := make([]string, 0)
	if err := filepath.Walk(tempDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(c.outputPath, relPath)
		files = append(files, targetPath)
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *extensionContext) moveSourceFiles(tracker *fileTracker, revisions []string, tempDir string) (sourcesSnapshot, error) {
	snapshot := sourcesSnapshot{Revisions: revisions}
	if err := filepath.Walk(tempDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(c.outputPath, relPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		if err := moveFile(path, targetPath); err != nil {
			return err
		}
		tracker.setMetadata(targetPath, fileMetadata{
			Owner: c.name,
		})
		snapshot.Files = append(snapshot.Files, targetPath)
		return nil
	}); err != nil {
		return sourcesSnapshot{}, err
	}
	return snapshot, nil
}

func (c *extensionContext) saveSnapshot(snapshot sourcesSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.snapshotPath, data, 0755); err != nil {
		return fmt.Errorf("failed to persist download sources revisions: %v", err)
	}
	return nil
}

func (c *extensionContext) loadSnapshot() sourcesSnapshot {
	var prev sourcesSnapshot
	if data, err := os.ReadFile(c.snapshotPath); err == nil {
		_ = json.Unmarshal(data, &prev)
	}
	return prev
}

func (c *extensionContext) deleteSnapshot() error {
	err := os.Remove(c.snapshotPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (c *extensionContext) downloadTo(ctx context.Context, out string) error {
	for _, s := range c.extension.Spec.Sources {
		switch {
		case s.Git != nil:
			parsedUrl, err := url.Parse(s.Git.Url)
			if err != nil {
				return err
			}
			secret, err := c.GetSecret(ctx, *s.Git.Secret)
			if err != nil {
				return err
			}
			var gitURL string
			baseDir := "resources"
			if c.extension.Spec.BaseDirectory != "" {
				baseDir = c.extension.Spec.BaseDirectory
			}
			if strings.HasPrefix(s.Git.Url, "ssh://") {
				sshKey := base64.StdEncoding.EncodeToString(secret.Data["sshkey"])
				gitURL = fmt.Sprintf("git::ssh://git@%s%s//%s?ref=%s&sshkey=%s", parsedUrl.Host, parsedUrl.Path, baseDir, s.Git.Revision, sshKey)
			}
			if strings.HasPrefix(s.Git.Url, "http://") || strings.HasPrefix(s.Git.Url, "https://") {
				git_user, _ := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(secret.Data["git_user"]))
				git_token, _ := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(secret.Data["git_token"]))
				gitURL = fmt.Sprintf("git::https://%s:%s@%s%s//%s?ref=%s", git_user, git_token, parsedUrl.Host, parsedUrl.Path, baseDir, s.Git.Revision)
			}
			if err := getter.Get(filepath.Clean(filepath.Join(out, "resources")), gitURL); err != nil {
				return err
			}
		case s.Web != nil:
			if err := getter.Get(out, "http::"+s.Web.Url); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *extensionContext) resolveRevisions(ctx context.Context) ([]string, error) {
	var res []string
	for _, s := range c.extension.Spec.Sources {
		switch {
		case s.Git != nil:
			secret, err := c.GetSecret(ctx, *s.Git.Secret)
			if err != nil {
				return nil, err
			}
			creds, err := getGitCred(s, secret)
			if err != nil {
				return nil, err
			}
			auth, err := git.NewAuth(s.Git.Url, *creds)
			if err != nil {
				return nil, err
			}
			sha, err := git.LsRemote(s.Git.Url, s.Git.Revision, auth, creds.Insecure)
			if err != nil {
				return nil, err
			}
			res = append(res, fmt.Sprintf("%s#%s", s.Git.Url, sha))
		case s.Web != nil:
			res = append(res, s.Web.Url)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})
	return res, nil
}

func moveFile(src string, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	output, err := os.Create(dst)
	if err != nil {
		_ = input.Close()
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	_ = input.Close()
	if err != nil {
		return err
	}

	return os.Remove(src)
}

func getGitCred(s extensionv1.ExtensionSource, secret v1.Secret) (*git.Creds, error) {
	var sshPrivateKey string
	var git_user string
	var git_token string
	var insecure bool
	if strings.HasPrefix(s.Git.Url, "ssh://") {
		if sshkey, ok := secret.Data["sshkey"]; ok {
			sshkey, err := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(sshkey))
			if err != nil {
				return nil, err
			}
			sshPrivateKey = string(sshkey)
		} else {
			return nil, fmt.Errorf("missing sshkey in the provided secret")
		}
	}
	if strings.HasPrefix(s.Git.Url, "http://") || strings.HasPrefix(s.Git.Url, "https://") {
		username, err := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(secret.Data["git_user"]))
		if err != nil {
			return nil, err
		}
		git_user = string(username)
		password, err := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(secret.Data["git_token"]))
		if err != nil {
			return nil, err
		}
		git_token = string(password)
	}
	if is_insecure, ok := secret.Data["insecure"]; ok {
		is_insecure, err := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(is_insecure))
		if err != nil {
			return nil, err
		}
		insecure, err = strconv.ParseBool(string(is_insecure))
		if err != nil {
			return nil, err
		}
	}
	creds := &git.Creds{
		SSHPrivateKey: sshPrivateKey,
		Username:      git_user,
		Password:      git_token,
		Insecure:      insecure,
	}
	return creds, nil
}
