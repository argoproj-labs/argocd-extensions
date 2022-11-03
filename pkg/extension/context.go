package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"

	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	extensionv1 "github.com/argoproj/argocd-extensions/api/v1alpha1"
	"github.com/argoproj/argocd-extensions/pkg/git"
	"github.com/hashicorp/go-getter"
)

type extensionContext struct {
	name         string
	outputPath   string
	snapshotPath string
	sources      []extensionv1.ExtensionSource
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

func (s sourcesSnapshot) deleteFiles() error {
	for i := range s.Files {
		if err := os.Remove(s.Files[i]); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func NewExtensionContext(extension *extensionv1.ArgoCDExtension, outputPath string) *extensionContext {
	return &extensionContext{
		name:         extension.Name,
		sources:      extension.Spec.Sources,
		outputPath:   outputPath,
		snapshotPath: path.Join(outputPath, fmt.Sprintf(".%s.snapshot", extension.Name)),
	}
}

// Process downloads extension files
func (c *extensionContext) Process(ctx context.Context) error {
	log := k8slog.FromContext(ctx)

	revisions, err := c.resolveRevisions()
	if err != nil {
		return fmt.Errorf("failed to resolve sources revisions: %v", err)
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

	// delete all previously downloaded extension files
	if err := prev.deleteFiles(); err != nil {
		return fmt.Errorf("failed to clean %s: %v", c.outputPath, err)
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

	if err := c.downloadTo(tempDir); err != nil {
		return fmt.Errorf("failed to download sources: %v", err)
	}

	// move downloaded files to the persistent extensions files location
	// and store list of files in the snapshot
	snapshot, err := c.moveSourceFiles(revisions, tempDir)
	if err != nil {
		return fmt.Errorf("failed to move source files: %v", err)
	}

	// store snapshot in extensions directory
	if err := c.saveSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to persist snapshot: %v", err)
	}

	log.Info("Successfully downloaded all sources.")
	return nil
}

// ProcessDeletion deletes all previously downloaded files for the extension
func (c *extensionContext) ProcessDeletion() error {
	err := c.loadSnapshot().deleteFiles()
	if err != nil {
		return err
	}
	return os.Remove(c.snapshotPath)
}

func (c *extensionContext) moveSourceFiles(revisions []string, tempDir string) (sourcesSnapshot, error) {
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

func (c *extensionContext) downloadTo(out string) error {
	for _, s := range c.sources {
		switch {
		case s.Git != nil:
			parsedUrl, err := url.Parse(s.Git.Url)
			if err != nil {
				return err
			}
			gitURL := fmt.Sprintf("git::%s%s//resources?ref=%s", parsedUrl.Host, parsedUrl.Path, s.Git.Revision)
			if err := getter.Get(filepath.Join(out, "resources"), gitURL); err != nil {
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

func (c *extensionContext) resolveRevisions() ([]string, error) {
	var res []string
	for _, s := range c.sources {
		switch {
		case s.Git != nil:
			sha, err := git.LsRemote(s.Git.Url, s.Git.Revision)
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
