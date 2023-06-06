package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/argoproj/argo-cd/common"
	certutil "github.com/argoproj/argo-cd/v2/util/cert"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	commitSHARegex          = regexp.MustCompile("^[0-9A-Fa-f]{40}$")
	truncatedCommitSHARegex = regexp.MustCompile("^[0-9A-Fa-f]{7,}$")
	sshURLRegex             = regexp.MustCompile("^(ssh://)?([^/:]*?)@[^@]+$")
	httpsURLRegex           = regexp.MustCompile("^(https://).*")
	httpURLRegex            = regexp.MustCompile("^(http://).*")
)

type Creds struct {
	SSHPrivateKey string
	Username      string
	Password      string
	Insecure      bool
}

// IsCommitSHA returns whether or not a string is a 40 character SHA-1
func IsCommitSHA(sha string) bool {
	return commitSHARegex.MatchString(sha)
}

// IsTruncatedCommitSHA returns whether or not a string is a truncated  SHA-1
func IsTruncatedCommitSHA(sha string) bool {
	return truncatedCommitSHARegex.MatchString(sha)
}

// LsRemote resolves commit sha for given Git repo and revision
func LsRemote(repoURL string, revision string, auth transport.AuthMethod, insecure bool) (string, error) {
	if IsCommitSHA(revision) || IsTruncatedCommitSHA(revision) {
		return revision, nil
	}

	repo, err := git.Init(memory.NewStorage(), nil)
	if err != nil {
		return "", err
	}
	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: git.DefaultRemoteName,
		URLs: []string{repoURL},
	})
	if err != nil {
		return "", err
	}
	refs, err := remote.List(&git.ListOptions{
		Auth:            auth,
		InsecureSkipTLS: insecure,
	})

	if err != nil {
		return "", err
	}

	if revision == "" {
		revision = "HEAD"
	}
	// refToHash keeps a maps of remote refs to their hash
	// (e.g. refs/heads/master -> a67038ae2e9cb9b9b16423702f98b41e36601001)
	refToHash := make(map[string]string)
	// refToResolve remembers ref name of the supplied revision if we determine the revision is a
	// symbolic reference (like HEAD), in which case we will resolve it from the refToHash map
	refToResolve := ""
	for _, ref := range refs {
		refName := ref.Name().String()
		hash := ref.Hash().String()
		if ref.Type() == plumbing.HashReference {
			refToHash[refName] = hash
		}
		if ref.Name().Short() == revision || refName == revision {
			if ref.Type() == plumbing.HashReference {
				return hash, nil
			}
			if ref.Type() == plumbing.SymbolicReference {
				refToResolve = ref.Target().String()
			}
		}
	}
	if refToResolve != "" {
		// If refToResolve is non-empty, we are resolving symbolic reference (e.g. HEAD).
		// It should exist in our refToHash map
		if hash, ok := refToHash[refToResolve]; ok {
			return hash, nil
		}
	}
	// If we get here, revision string had non hexadecimal characters (indicating its a branch, tag,
	// or symbolic ref) and we were unable to resolve it to a commit SHA.
	return "", fmt.Errorf("Unable to resolve '%s' to a commit SHA", revision)
}

func NewAuth(repoURL string, creds Creds) (transport.AuthMethod, error) {
	if isSSH, user := IsSSHURL(repoURL); isSSH {
		sshUser := user
		signer, err := ssh.ParsePrivateKey([]byte(creds.SSHPrivateKey))
		if err != nil {
			return nil, err
		}
		auth := &gitssh.PublicKeys{}
		auth.User = sshUser
		auth.Signer = signer
		if creds.Insecure {
			auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		} else {
			// Set up validation of SSH known hosts for using our ssh_known_hosts
			// file.
			auth.HostKeyCallback, err = knownhosts.New(certutil.GetSSHKnownHostsDataPath())
			if err != nil {
				log.Errorf("Could not set-up SSH known hosts callback: %v", err)
			}
		}
		return auth, nil
	}
	if IsHTTPURL(repoURL) || IsHTTPSURL(repoURL) && len(creds.Password) > 0 {
		auth := githttp.BasicAuth{Username: creds.Username, Password: creds.Password}
		if auth.Username == "" {
			auth.Username = "x-access-token"
		}
		return &auth, nil
	}
	return nil, nil
}

func IsSSHURL(url string) (bool, string) {
	matches := sshURLRegex.FindStringSubmatch(url)
	if len(matches) > 2 {
		return true, matches[2]
	}
	return false, ""
}

func IsHTTPSURL(url string) bool {
	return httpsURLRegex.MatchString(url)
}

func IsHTTPURL(url string) bool {
	return httpURLRegex.MatchString(url)
}

func GetSSHKnownHostsDataPath() string {
	if envPath := os.Getenv(common.EnvVarSSHDataPath); envPath != "" {
		return filepath.Join(envPath, common.DefaultSSHKnownHostsName)
	} else {
		return filepath.Join(common.DefaultPathSSHConfig, common.DefaultSSHKnownHostsName)
	}
}
