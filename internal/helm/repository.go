/*
Copyright 2025.

Licensed under the MIT License.
*/

package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

// repositoryManager implements the Repository interface
type repositoryManager struct {
	settings     *cli.EnvSettings
	repoFile     string
	repoCache    string
	repositories map[string]*repo.Entry
	mu           sync.RWMutex
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager() (Repository, error) {
	settings := cli.New()
	
	// Ensure repository directories exist
	if err := os.MkdirAll(filepath.Dir(settings.RepositoryConfig), 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create repository config directory")
	}
	
	if err := os.MkdirAll(settings.RepositoryCache, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create repository cache directory")
	}
	
	rm := &repositoryManager{
		settings:     settings,
		repoFile:     settings.RepositoryConfig,
		repoCache:    settings.RepositoryCache,
		repositories: make(map[string]*repo.Entry),
	}
	
	// Load existing repositories
	if err := rm.loadRepositories(); err != nil {
		return nil, err
	}
	
	// Add default repositories if not exist
	if err := rm.addDefaultRepositories(); err != nil {
		return nil, err
	}
	
	return rm, nil
}

// loadRepositories loads repository configuration from file
func (rm *repositoryManager) loadRepositories() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Check if repository file exists
	if _, err := os.Stat(rm.repoFile); os.IsNotExist(err) {
		// Create empty repository file
		repoFile := &repo.File{}
		if err := repoFile.WriteFile(rm.repoFile, 0644); err != nil {
			return errors.Wrap(err, "failed to create repository file")
		}
		return nil
	}
	
	// Load repository file
	repoFile, err := repo.LoadFile(rm.repoFile)
	if err != nil {
		return errors.Wrap(err, "failed to load repository file")
	}
	
	// Store repositories in memory
	for _, entry := range repoFile.Repositories {
		rm.repositories[entry.Name] = entry
	}
	
	return nil
}

// addDefaultRepositories adds default Helm repositories
func (rm *repositoryManager) addDefaultRepositories() error {
	defaultRepos := map[string]string{
		"prometheus-community": "https://prometheus-community.github.io/helm-charts",
		"grafana":             "https://grafana.github.io/helm-charts",
		"bitnami":             "https://charts.bitnami.com/bitnami",
		"stable":              "https://charts.helm.sh/stable",
	}
	
	for name, url := range defaultRepos {
		if _, exists := rm.repositories[name]; !exists {
			if err := rm.AddRepository(context.Background(), name, url, nil); err != nil {
				// Log error but don't fail
				fmt.Printf("Warning: failed to add default repository %s: %v\n", name, err)
			}
		}
	}
	
	return nil
}

// AddRepository adds a new Helm repository
func (rm *repositoryManager) AddRepository(ctx context.Context, name, url string, opts *RepositoryOptions) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Create repository entry
	entry := &repo.Entry{
		Name: name,
		URL:  url,
	}
	
	// Apply options if provided
	if opts != nil {
		entry.Username = opts.Username
		entry.Password = opts.Password
		entry.CertFile = opts.CertFile
		entry.KeyFile = opts.KeyFile
		entry.CAFile = opts.CAFile
		entry.InsecureSkipTLSverify = opts.InsecureSkipTLSVerify
		entry.PassCredentialsAll = opts.PassCredentialsAll
	}
	
	// Create chart repository
	chartRepo, err := repo.NewChartRepository(entry, getter.All(rm.settings))
	if err != nil {
		return errors.Wrap(err, "failed to create chart repository")
	}
	
	// Set cache path
	chartRepo.CachePath = rm.repoCache
	
	// Download index file
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return errors.Wrap(err, "failed to download repository index")
	}
	
	// Load repository file
	repoFile, err := repo.LoadFile(rm.repoFile)
	if err != nil {
		return errors.Wrap(err, "failed to load repository file")
	}
	
	// Check if repository already exists
	if repoFile.Has(name) {
		return fmt.Errorf("repository %s already exists", name)
	}
	
	// Add repository to file
	repoFile.Update(entry)
	
	// Save repository file
	if err := repoFile.WriteFile(rm.repoFile, 0644); err != nil {
		return errors.Wrap(err, "failed to save repository file")
	}
	
	// Update memory cache
	rm.repositories[name] = entry
	
	return nil
}

// UpdateRepository updates repository index
func (rm *repositoryManager) UpdateRepository(ctx context.Context, name string) error {
	rm.mu.RLock()
	entry, exists := rm.repositories[name]
	rm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("repository %s not found", name)
	}
	
	// Create chart repository
	chartRepo, err := repo.NewChartRepository(entry, getter.All(rm.settings))
	if err != nil {
		return errors.Wrap(err, "failed to create chart repository")
	}
	
	// Set cache path
	chartRepo.CachePath = rm.repoCache
	
	// Download index file
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return errors.Wrap(err, "failed to update repository index")
	}
	
	return nil
}

// RemoveRepository removes a Helm repository
func (rm *repositoryManager) RemoveRepository(ctx context.Context, name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Load repository file
	repoFile, err := repo.LoadFile(rm.repoFile)
	if err != nil {
		return errors.Wrap(err, "failed to load repository file")
	}
	
	// Check if repository exists
	if !repoFile.Has(name) {
		return fmt.Errorf("repository %s not found", name)
	}
	
	// Remove repository from file
	repoFile.Remove(name)
	
	// Save repository file
	if err := repoFile.WriteFile(rm.repoFile, 0644); err != nil {
		return errors.Wrap(err, "failed to save repository file")
	}
	
	// Remove from memory cache
	delete(rm.repositories, name)
	
	// Remove index file from cache
	indexFile := filepath.Join(rm.repoCache, fmt.Sprintf("%s-index.yaml", name))
	os.Remove(indexFile)
	
	return nil
}

// ListRepositories lists all configured repositories
func (rm *repositoryManager) ListRepositories(ctx context.Context) ([]*RepositoryInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	repos := make([]*RepositoryInfo, 0, len(rm.repositories))
	
	for name, entry := range rm.repositories {
		status := "OK"
		
		// Check if index file exists
		indexFile := filepath.Join(rm.repoCache, fmt.Sprintf("%s-index.yaml", name))
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			status = "No index"
		}
		
		repos = append(repos, &RepositoryInfo{
			Name:   name,
			URL:    entry.URL,
			Status: status,
		})
	}
	
	return repos, nil
}

// SearchCharts searches for charts in repositories
func (rm *repositoryManager) SearchCharts(ctx context.Context, keyword string) ([]*ChartInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	results := make([]*ChartInfo, 0)
	
	for repoName, entry := range rm.repositories {
		// Load repository index
		indexFile := filepath.Join(rm.repoCache, fmt.Sprintf("%s-index.yaml", repoName))
		
		index, err := repo.LoadIndexFile(indexFile)
		if err != nil {
			// Skip repositories without index
			continue
		}
		
		// Search in repository
		for chartName, versions := range index.Entries {
			// Check if chart name or description contains keyword
			if strings.Contains(strings.ToLower(chartName), strings.ToLower(keyword)) {
				// Get latest version
				if len(versions) > 0 {
					latest := versions[0]
					results = append(results, &ChartInfo{
						Name:        fmt.Sprintf("%s/%s", repoName, chartName),
						Version:     latest.Version,
						Repository:  repoName,
						Description: latest.Description,
					})
				}
			} else {
				// Check in descriptions
				for _, version := range versions {
					if strings.Contains(strings.ToLower(version.Description), strings.ToLower(keyword)) {
						results = append(results, &ChartInfo{
							Name:        fmt.Sprintf("%s/%s", repoName, chartName),
							Version:     version.Version,
							Repository:  repoName,
							Description: version.Description,
						})
						break
					}
				}
			}
		}
	}
	
	return results, nil
}

// GetChartVersions gets available versions for a chart
func (rm *repositoryManager) GetChartVersions(ctx context.Context, chartName string) ([]string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	// Parse chart name to get repository and chart
	parts := strings.Split(chartName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid chart name format, expected repo/chart")
	}
	
	repoName := parts[0]
	chart := parts[1]
	
	// Check if repository exists
	entry, exists := rm.repositories[repoName]
	if !exists {
		return nil, fmt.Errorf("repository %s not found", repoName)
	}
	
	// Load repository index
	indexFile := filepath.Join(rm.repoCache, fmt.Sprintf("%s-index.yaml", repoName))
	
	index, err := repo.LoadIndexFile(indexFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load repository index")
	}
	
	// Get chart versions
	chartVersions, ok := index.Entries[chart]
	if !ok {
		return nil, fmt.Errorf("chart %s not found in repository %s", chart, repoName)
	}
	
	// Extract version strings
	versions := make([]string, len(chartVersions))
	for i, cv := range chartVersions {
		versions[i] = cv.Version
	}
	
	return versions, nil
}
