/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-logr/logr"
	ssh2 "golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
)

// RepositoryManager manages Git repository operations
type RepositoryManager struct {
	Client    client.Client
	Log       logr.Logger
	TempDir   string
	AuthCache map[string]transport.AuthMethod
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(client client.Client, log logr.Logger) (*RepositoryManager, error) {
	tempDir, err := os.MkdirTemp("", "gunj-gitops-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &RepositoryManager{
		Client:    client,
		Log:       log.WithName("git-repository-manager"),
		TempDir:   tempDir,
		AuthCache: make(map[string]transport.AuthMethod),
	}, nil
}

// Cleanup cleans up temporary resources
func (m *RepositoryManager) Cleanup() error {
	return os.RemoveAll(m.TempDir)
}

// CloneRepository clones a Git repository
func (m *RepositoryManager) CloneRepository(
	ctx context.Context,
	repoSpec gitopsv1beta1.GitRepositorySpec,
) (*git.Repository, string, error) {
	log := m.Log.WithValues("url", repoSpec.URL)
	log.Info("Cloning repository")

	// Create unique directory for this clone
	cloneDir := filepath.Join(m.TempDir, fmt.Sprintf("repo-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create clone directory: %w", err)
	}

	// Get authentication
	auth, err := m.getAuth(ctx, repoSpec)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get authentication: %w", err)
	}

	// Clone options
	cloneOptions := &git.CloneOptions{
		URL:           repoSpec.URL,
		Auth:          auth,
		ReferenceName: plumbing.NewBranchReferenceName(repoSpec.Branch),
		SingleBranch:  true,
		Depth:         1,
		Progress:      nil, // Could add progress reporting
	}

	// Clone the repository
	repo, err := git.PlainCloneContext(ctx, cloneDir, false, cloneOptions)
	if err != nil {
		return nil, "", fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Info("Repository cloned successfully", "path", cloneDir)
	return repo, cloneDir, nil
}

// PullRepository pulls latest changes from remote
func (m *RepositoryManager) PullRepository(
	ctx context.Context,
	repo *git.Repository,
	repoSpec gitopsv1beta1.GitRepositorySpec,
) error {
	log := m.Log.WithValues("url", repoSpec.URL)
	log.Info("Pulling repository")

	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get authentication
	auth, err := m.getAuth(ctx, repoSpec)
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	// Pull options
	pullOptions := &git.PullOptions{
		RemoteName:    "origin",
		Auth:          auth,
		ReferenceName: plumbing.NewBranchReferenceName(repoSpec.Branch),
		SingleBranch:  true,
		Force:         true,
	}

	// Pull latest changes
	err = worktree.Pull(pullOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull repository: %w", err)
	}

	if err == git.NoErrAlreadyUpToDate {
		log.Info("Repository already up to date")
	} else {
		log.Info("Repository pulled successfully")
	}

	return nil
}

// GetCurrentRevision gets the current HEAD revision
func (m *RepositoryManager) GetCurrentRevision(repo *git.Repository) (string, error) {
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return ref.Hash().String(), nil
}

// GetFilesAtPath gets files at a specific path in the repository
func (m *RepositoryManager) GetFilesAtPath(repo *git.Repository, path string) (map[string][]byte, error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	files := make(map[string][]byte)
	rootPath := worktree.Filesystem.Root()
	searchPath := filepath.Join(rootPath, path)

	// Walk the directory
	err = filepath.Walk(searchPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		// Store with relative path
		relPath, err := filepath.Rel(searchPath, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		files[relPath] = content
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// CommitAndPush commits changes and pushes to remote
func (m *RepositoryManager) CommitAndPush(
	ctx context.Context,
	repo *git.Repository,
	repoSpec gitopsv1beta1.GitRepositorySpec,
	message string,
	author string,
	email string,
) error {
	log := m.Log.WithValues("url", repoSpec.URL)
	log.Info("Committing and pushing changes")

	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Check if there are changes to commit
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsClean() {
		log.Info("No changes to commit")
		return nil
	}

	// Commit changes
	commit, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  author,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Info("Changes committed", "commit", commit.String())

	// Get authentication
	auth, err := m.getAuth(ctx, repoSpec)
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	// Push changes
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   nil,
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	log.Info("Changes pushed successfully")
	return nil
}

// CreateBranch creates a new branch
func (m *RepositoryManager) CreateBranch(
	repo *git.Repository,
	branchName string,
	baseBranch string,
) error {
	log := m.Log.WithValues("branch", branchName, "base", baseBranch)
	log.Info("Creating branch")

	// Get base branch reference
	baseRef, err := repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		return fmt.Errorf("failed to get base branch reference: %w", err)
	}

	// Create new branch
	branchRef := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(branchRef, baseRef.Hash())

	err = repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Checkout new branch
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: false,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	log.Info("Branch created successfully")
	return nil
}

// getAuth gets authentication method for the repository
func (m *RepositoryManager) getAuth(ctx context.Context, repoSpec gitopsv1beta1.GitRepositorySpec) (transport.AuthMethod, error) {
	// Check cache
	cacheKey := fmt.Sprintf("%s:%s", repoSpec.URL, repoSpec.CredentialsSecret.Name)
	if auth, ok := m.AuthCache[cacheKey]; ok {
		return auth, nil
	}

	// No credentials needed for public repositories
	if repoSpec.CredentialsSecret == nil {
		return nil, nil
	}

	// Get secret
	secret := &corev1.Secret{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      repoSpec.CredentialsSecret.Name,
		Namespace: repoSpec.CredentialsSecret.Namespace,
	}, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials secret: %w", err)
	}

	// Determine auth type based on URL
	var auth transport.AuthMethod
	if strings.HasPrefix(repoSpec.URL, "http") {
		// HTTP/HTTPS authentication
		username := string(secret.Data["username"])
		password := string(secret.Data["password"])
		
		if username == "" {
			username = "git" // Default for token-based auth
		}

		auth = &http.BasicAuth{
			Username: username,
			Password: password,
		}
	} else if strings.HasPrefix(repoSpec.URL, "git@") || strings.HasPrefix(repoSpec.URL, "ssh://") {
		// SSH authentication
		privateKey := secret.Data["ssh-privatekey"]
		if len(privateKey) == 0 {
			privateKey = secret.Data["identity"]
		}

		if len(privateKey) == 0 {
			return nil, fmt.Errorf("no SSH private key found in secret")
		}

		sshAuth, err := ssh.NewPublicKeys("git", privateKey, "")
		if err != nil {
			return nil, fmt.Errorf("failed to create SSH auth: %w", err)
		}

		// Configure host key callback
		knownHosts := secret.Data["known_hosts"]
		if len(knownHosts) > 0 {
			// Parse known hosts
			// This is simplified - in production, you'd properly parse known_hosts
			sshAuth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
		} else {
			// Insecure - accept any host key
			sshAuth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
		}

		auth = sshAuth
	} else {
		return nil, fmt.Errorf("unsupported repository URL scheme: %s", repoSpec.URL)
	}

	// Cache the auth
	m.AuthCache[cacheKey] = auth

	return auth, nil
}

// WriteFiles writes files to the repository
func (m *RepositoryManager) WriteFiles(
	repo *git.Repository,
	basePath string,
	files map[string][]byte,
) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	rootPath := worktree.Filesystem.Root()

	for filename, content := range files {
		fullPath := filepath.Join(rootPath, basePath, filename)
		
		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// DiffFiles compares files between two commits
func (m *RepositoryManager) DiffFiles(
	repo *git.Repository,
	fromRevision string,
	toRevision string,
) ([]FileDiff, error) {
	// Get commits
	fromCommit, err := repo.CommitObject(plumbing.NewHash(fromRevision))
	if err != nil {
		return nil, fmt.Errorf("failed to get from commit: %w", err)
	}

	toCommit, err := repo.CommitObject(plumbing.NewHash(toRevision))
	if err != nil {
		return nil, fmt.Errorf("failed to get to commit: %w", err)
	}

	// Get trees
	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get from tree: %w", err)
	}

	toTree, err := toCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get to tree: %w", err)
	}

	// Calculate diff
	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}

	// Convert to FileDiff
	var diffs []FileDiff
	for _, change := range changes {
		diff := FileDiff{
			Path: change.To.Name,
		}

		switch {
		case change.From.Name == "" && change.To.Name != "":
			diff.Type = "added"
		case change.From.Name != "" && change.To.Name == "":
			diff.Type = "deleted"
			diff.Path = change.From.Name
		default:
			diff.Type = "modified"
		}

		// Get patch
		patch, err := change.Patch()
		if err == nil {
			diff.Patch = patch.String()
		}

		diffs = append(diffs, diff)
	}

	return diffs, nil
}

// FileDiff represents a file difference
type FileDiff struct {
	Path  string
	Type  string // added, modified, deleted
	Patch string
}

// GetBranches gets all branches in the repository
func (m *RepositoryManager) GetBranches(repo *git.Repository) ([]string, error) {
	iter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	var branches []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		branches = append(branches, name)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return branches, nil
}

// GetTags gets all tags in the repository
func (m *RepositoryManager) GetTags(repo *git.Repository) ([]string, error) {
	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	var tags []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		tags = append(tags, name)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate tags: %w", err)
	}

	return tags, nil
}

// FetchRemote fetches updates from remote repository
func (m *RepositoryManager) FetchRemote(
	ctx context.Context,
	repo *git.Repository,
	repoSpec gitopsv1beta1.GitRepositorySpec,
) error {
	log := m.Log.WithValues("url", repoSpec.URL)
	log.Info("Fetching from remote")

	// Get authentication
	auth, err := m.getAuth(ctx, repoSpec)
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	// Fetch options
	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/remotes/origin/*"),
			config.RefSpec("+refs/tags/*:refs/tags/*"),
		},
		Force:    true,
		Progress: nil,
	}

	// Fetch from remote
	err = repo.Fetch(fetchOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch from remote: %w", err)
	}

	if err == git.NoErrAlreadyUpToDate {
		log.Info("Remote already up to date")
	} else {
		log.Info("Fetched from remote successfully")
	}

	return nil
}

// CheckoutRevision checks out a specific revision
func (m *RepositoryManager) CheckoutRevision(
	repo *git.Repository,
	revision string,
) error {
	log := m.Log.WithValues("revision", revision)
	log.Info("Checking out revision")

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Try as commit hash first
	hash := plumbing.NewHash(revision)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err == nil {
		log.Info("Checked out commit successfully")
		return nil
	}

	// Try as branch
	branchRef := plumbing.NewBranchReferenceName(revision)
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
	})
	if err == nil {
		log.Info("Checked out branch successfully")
		return nil
	}

	// Try as tag
	tagRef := plumbing.NewTagReferenceName(revision)
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: tagRef,
	})
	if err == nil {
		log.Info("Checked out tag successfully")
		return nil
	}

	return fmt.Errorf("failed to checkout revision %s", revision)
}
