// Package sync provides Git synchronization functionality
package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/config"
	"github.com/gunjanjp/gunj-operator/internal/gitops"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultCloneTimeout is the default timeout for clone operations
	DefaultCloneTimeout = 5 * time.Minute
	
	// DefaultFetchTimeout is the default timeout for fetch operations
	DefaultFetchTimeout = 2 * time.Minute
	
	// CacheTTL is how long to cache repository data
	CacheTTL = 5 * time.Minute
)

// Synchronizer implements Git synchronization operations
type Synchronizer struct {
	client     client.Client
	config     *config.Config
	log        logr.Logger
	
	// Repository cache
	cacheMutex sync.RWMutex
	repoCache  map[string]*cachedRepo
	
	// Temporary directory for clones
	workDir    string
}

// cachedRepo represents a cached repository
type cachedRepo struct {
	path       string
	repo       *git.Repository
	lastUpdate time.Time
	auth       transport.AuthMethod
}

// NewSynchronizer creates a new Git synchronizer
func NewSynchronizer(cfg *config.Config) (*Synchronizer, error) {
	// Create work directory
	workDir := filepath.Join(os.TempDir(), "gunj-operator", "git-sync")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("creating work directory: %w", err)
	}
	
	return &Synchronizer{
		config:    cfg,
		log:       ctrl.Log.WithName("git-synchronizer"),
		repoCache: make(map[string]*cachedRepo),
		workDir:   workDir,
	}, nil
}

// SetClient sets the Kubernetes client for secret access
func (s *Synchronizer) SetClient(client client.Client) {
	s.client = client
}

// Clone clones a Git repository
func (s *Synchronizer) Clone(ctx context.Context, repo gitops.GitRepository) (string, error) {
	log := s.log.WithValues("repo", repo.URL, "branch", repo.Branch)
	
	// Generate cache key
	cacheKey := s.getCacheKey(repo)
	
	// Check cache
	s.cacheMutex.RLock()
	cached, exists := s.repoCache[cacheKey]
	s.cacheMutex.RUnlock()
	
	if exists && time.Since(cached.lastUpdate) < CacheTTL {
		// Use cached repository, but update it
		if err := s.updateRepository(ctx, cached); err != nil {
			log.Error(err, "Failed to update cached repository")
			// Continue with stale cache rather than failing
		}
		return cached.path, nil
	}
	
	// Get authentication
	auth, err := s.getAuth(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("getting authentication: %w", err)
	}
	
	// Create repository directory
	repoPath := filepath.Join(s.workDir, cacheKey)
	
	// Clone or open existing repository
	var gitRepo *git.Repository
	if exists {
		// Open existing repository
		gitRepo, err = git.PlainOpen(repoPath)
		if err != nil {
			// Repository corrupted, remove and re-clone
			os.RemoveAll(repoPath)
			exists = false
		}
	}
	
	if !gitRepo {
		// Clone new repository
		log.Info("Cloning repository")
		
		cloneOpts := &git.CloneOptions{
			URL:           repo.URL,
			Auth:          auth,
			Progress:      os.Stdout,
			ReferenceName: plumbing.NewBranchReferenceName(repo.Branch),
			SingleBranch:  true,
			Depth:         1, // Shallow clone for performance
		}
		
		// Clone with timeout
		cloneCtx, cancel := context.WithTimeout(ctx, DefaultCloneTimeout)
		defer cancel()
		
		gitRepo, err = git.PlainCloneContext(cloneCtx, repoPath, false, cloneOpts)
		if err != nil {
			return "", fmt.Errorf("cloning repository: %w", err)
		}
	}
	
	// Cache the repository
	s.cacheMutex.Lock()
	s.repoCache[cacheKey] = &cachedRepo{
		path:       repoPath,
		repo:       gitRepo,
		lastUpdate: time.Now(),
		auth:       auth,
	}
	s.cacheMutex.Unlock()
	
	log.Info("Repository cloned successfully", "path", repoPath)
	return repoPath, nil
}

// Pull pulls latest changes from the repository
func (s *Synchronizer) Pull(ctx context.Context, repoPath string) error {
	log := s.log.WithValues("path", repoPath)
	
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}
	
	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}
	
	// Find cached repo for auth
	var auth transport.AuthMethod
	s.cacheMutex.RLock()
	for _, cached := range s.repoCache {
		if cached.path == repoPath {
			auth = cached.auth
			break
		}
	}
	s.cacheMutex.RUnlock()
	
	// Pull with timeout
	pullCtx, cancel := context.WithTimeout(ctx, DefaultFetchTimeout)
	defer cancel()
	
	pullOpts := &git.PullOptions{
		Auth:     auth,
		Progress: os.Stdout,
		Force:    true,
	}
	
	err = worktree.PullContext(pullCtx, pullOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("pulling changes: %w", err)
	}
	
	// Update cache timestamp
	s.cacheMutex.Lock()
	for key, cached := range s.repoCache {
		if cached.path == repoPath {
			cached.lastUpdate = time.Now()
			s.repoCache[key] = cached
			break
		}
	}
	s.cacheMutex.Unlock()
	
	log.Info("Repository updated successfully")
	return nil
}

// GetRevision gets the current revision of the repository
func (s *Synchronizer) GetRevision(ctx context.Context, repoPath string) (string, error) {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("opening repository: %w", err)
	}
	
	// Get HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD: %w", err)
	}
	
	return ref.Hash().String(), nil
}

// GetFiles gets files matching a pattern from the repository
func (s *Synchronizer) GetFiles(ctx context.Context, repoPath string, pattern string) ([]string, error) {
	var files []string
	
	// Walk the repository directory
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip .git directory
		if strings.Contains(path, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Get relative path
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}
		
		// Match pattern
		matched, err := filepath.Match(pattern, filepath.Base(relPath))
		if err != nil {
			return err
		}
		
		if matched {
			files = append(files, relPath)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("walking repository: %w", err)
	}
	
	return files, nil
}

// Cleanup cleans up cloned repositories
func (s *Synchronizer) Cleanup(ctx context.Context, repoPath string) error {
	// Remove from cache
	s.cacheMutex.Lock()
	for key, cached := range s.repoCache {
		if cached.path == repoPath {
			delete(s.repoCache, key)
			break
		}
	}
	s.cacheMutex.Unlock()
	
	// Remove directory
	if err := os.RemoveAll(repoPath); err != nil {
		return fmt.Errorf("removing repository: %w", err)
	}
	
	return nil
}

// CleanupAll cleans up all cached repositories
func (s *Synchronizer) CleanupAll() error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	
	// Clear cache
	s.repoCache = make(map[string]*cachedRepo)
	
	// Remove work directory
	if err := os.RemoveAll(s.workDir); err != nil {
		return fmt.Errorf("removing work directory: %w", err)
	}
	
	// Recreate work directory
	if err := os.MkdirAll(s.workDir, 0755); err != nil {
		return fmt.Errorf("creating work directory: %w", err)
	}
	
	return nil
}

// updateRepository updates an existing repository
func (s *Synchronizer) updateRepository(ctx context.Context, cached *cachedRepo) error {
	// Get worktree
	worktree, err := cached.repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}
	
	// Pull with timeout
	pullCtx, cancel := context.WithTimeout(ctx, DefaultFetchTimeout)
	defer cancel()
	
	pullOpts := &git.PullOptions{
		Auth:     cached.auth,
		Progress: os.Stdout,
		Force:    true,
	}
	
	err = worktree.PullContext(pullCtx, pullOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("pulling changes: %w", err)
	}
	
	cached.lastUpdate = time.Now()
	return nil
}

// getAuth gets authentication for the repository
func (s *Synchronizer) getAuth(ctx context.Context, repo gitops.GitRepository) (transport.AuthMethod, error) {
	// No auth needed for public repositories
	if repo.SecretRef == nil {
		return nil, nil
	}
	
	// Get secret
	secret := &corev1.Secret{}
	err := s.client.Get(ctx, types.NamespacedName{
		Name:      repo.SecretRef.Name,
		Namespace: repo.SecretRef.Namespace,
	}, secret)
	if err != nil {
		return nil, fmt.Errorf("getting secret: %w", err)
	}
	
	// Check for SSH key
	if sshKey, exists := secret.Data["ssh-privatekey"]; exists {
		auth, err := ssh.NewPublicKeys("git", sshKey, "")
		if err != nil {
			return nil, fmt.Errorf("creating SSH auth: %w", err)
		}
		
		// Skip host key verification (not recommended for production)
		// TODO: Implement proper host key verification
		auth.HostKeyCallback = ssh.InsecureIgnoreHostKey()
		
		return auth, nil
	}
	
	// Check for username/password or token
	username := string(secret.Data["username"])
	password := string(secret.Data["password"])
	token := string(secret.Data["token"])
	
	if token != "" {
		// Use token as password with any username
		if username == "" {
			username = "token"
		}
		password = token
	}
	
	if username != "" && password != "" {
		return &http.BasicAuth{
			Username: username,
			Password: password,
		}, nil
	}
	
	return nil, fmt.Errorf("no valid authentication found in secret")
}

// getCacheKey generates a cache key for a repository
func (s *Synchronizer) getCacheKey(repo gitops.GitRepository) string {
	// Parse URL to remove credentials
	u, err := url.Parse(repo.URL)
	if err != nil {
		// Fallback to hashing the whole URL
		h := sha256.Sum256([]byte(repo.URL + repo.Branch))
		return hex.EncodeToString(h[:])
	}
	
	// Remove user info
	u.User = nil
	
	// Create cache key from clean URL and branch
	key := fmt.Sprintf("%s-%s", u.String(), repo.Branch)
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])[:16] // Use first 16 chars for shorter directory names
}

// GetCommitInfo gets information about a specific commit
func (s *Synchronizer) GetCommitInfo(ctx context.Context, repoPath string, revision string) (*CommitInfo, error) {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}
	
	// Get commit
	hash := plumbing.NewHash(revision)
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("getting commit: %w", err)
	}
	
	return &CommitInfo{
		Hash:    commit.Hash.String(),
		Author:  commit.Author.Name,
		Email:   commit.Author.Email,
		Time:    commit.Author.When,
		Message: commit.Message,
	}, nil
}

// GetDiff gets the diff between two revisions
func (s *Synchronizer) GetDiff(ctx context.Context, repoPath string, fromRevision, toRevision string) (string, error) {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("opening repository: %w", err)
	}
	
	// Get commits
	fromHash := plumbing.NewHash(fromRevision)
	fromCommit, err := repo.CommitObject(fromHash)
	if err != nil {
		return "", fmt.Errorf("getting from commit: %w", err)
	}
	
	toHash := plumbing.NewHash(toRevision)
	toCommit, err := repo.CommitObject(toHash)
	if err != nil {
		return "", fmt.Errorf("getting to commit: %w", err)
	}
	
	// Get trees
	fromTree, err := fromCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("getting from tree: %w", err)
	}
	
	toTree, err := toCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("getting to tree: %w", err)
	}
	
	// Calculate diff
	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return "", fmt.Errorf("calculating diff: %w", err)
	}
	
	// Format diff
	var diff strings.Builder
	for _, change := range changes {
		patch, err := change.Patch()
		if err != nil {
			continue
		}
		diff.WriteString(patch.String())
		diff.WriteString("\n")
	}
	
	return diff.String(), nil
}

// ListBranches lists all branches in the repository
func (s *Synchronizer) ListBranches(ctx context.Context, repoPath string) ([]string, error) {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}
	
	// Get remote
	remote, err := repo.Remote("origin")
	if err != nil {
		return nil, fmt.Errorf("getting remote: %w", err)
	}
	
	// Find auth for this repo
	var auth transport.AuthMethod
	s.cacheMutex.RLock()
	for _, cached := range s.repoCache {
		if cached.path == repoPath {
			auth = cached.auth
			break
		}
	}
	s.cacheMutex.RUnlock()
	
	// List remote references
	refs, err := remote.List(&git.ListOptions{
		Auth: auth,
	})
	if err != nil {
		return nil, fmt.Errorf("listing references: %w", err)
	}
	
	// Extract branch names
	var branches []string
	for _, ref := range refs {
		if ref.Name().IsBranch() {
			branches = append(branches, ref.Name().Short())
		}
	}
	
	return branches, nil
}

// ListTags lists all tags in the repository
func (s *Synchronizer) ListTags(ctx context.Context, repoPath string) ([]string, error) {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}
	
	// Get tags
	tags, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("getting tags: %w", err)
	}
	
	var tagList []string
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagList = append(tagList, ref.Name().Short())
		return nil
	})
	
	return tagList, err
}

// CheckoutRevision checks out a specific revision
func (s *Synchronizer) CheckoutRevision(ctx context.Context, repoPath string, revision string) error {
	// Open repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}
	
	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}
	
	// Checkout revision
	hash := plumbing.NewHash(revision)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash:  hash,
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("checking out revision: %w", err)
	}
	
	return nil
}

// CommitInfo represents information about a commit
type CommitInfo struct {
	Hash    string
	Author  string
	Email   string
	Time    time.Time
	Message string
}
