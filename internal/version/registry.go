// Package version provides chart version management capabilities
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/helm"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Registry manages chart version information
type Registry struct {
	mu           sync.RWMutex
	versions     map[string]*ChartVersionInfo
	repositories map[string]*repo.ChartRepository
	helmClient   helm.Interface
	log          logr.Logger
	updateTicker *time.Ticker
	stopCh       chan struct{}
}

// ChartVersionInfo contains version information for a chart
type ChartVersionInfo struct {
	ChartName       string                 `json:"chartName"`
	Repository      string                 `json:"repository"`
	CurrentVersion  string                 `json:"currentVersion"`
	LatestVersion   string                 `json:"latestVersion"`
	AvailableVersions []string             `json:"availableVersions"`
	LastUpdated     time.Time              `json:"lastUpdated"`
	UpdateAvailable bool                   `json:"updateAvailable"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// RegistryOptions configures the registry
type RegistryOptions struct {
	HelmClient        helm.Interface
	UpdateInterval    time.Duration
	EnableAutoUpdate  bool
	CacheDir          string
}

// NewRegistry creates a new version registry
func NewRegistry(opts RegistryOptions) (*Registry, error) {
	if opts.HelmClient == nil {
		return nil, fmt.Errorf("helm client is required")
	}

	if opts.UpdateInterval == 0 {
		opts.UpdateInterval = 6 * time.Hour
	}

	r := &Registry{
		versions:     make(map[string]*ChartVersionInfo),
		repositories: make(map[string]*repo.ChartRepository),
		helmClient:   opts.HelmClient,
		log:          log.Log.WithName("version-registry"),
		stopCh:       make(chan struct{}),
	}

	if opts.EnableAutoUpdate {
		r.startAutoUpdate(opts.UpdateInterval)
	}

	return r, nil
}

// RegisterChart registers a chart for version tracking
func (r *Registry) RegisterChart(ctx context.Context, chartName, repository string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.getChartKey(chartName, repository)
	if _, exists := r.versions[key]; exists {
		return nil // Already registered
	}

	info := &ChartVersionInfo{
		ChartName:  chartName,
		Repository: repository,
		LastUpdated: time.Now(),
	}

	// Fetch initial version information
	if err := r.updateChartVersions(ctx, info); err != nil {
		return fmt.Errorf("failed to fetch initial versions: %w", err)
	}

	r.versions[key] = info
	r.log.Info("Chart registered", "chart", chartName, "repository", repository)

	return nil
}

// GetChartInfo retrieves version information for a chart
func (r *Registry) GetChartInfo(chartName, repository string) (*ChartVersionInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := r.getChartKey(chartName, repository)
	info, exists := r.versions[key]
	if !exists {
		return nil, fmt.Errorf("chart %s not found in registry", chartName)
	}

	// Return a copy to prevent external modifications
	return r.copyChartInfo(info), nil
}

// CheckForUpdates checks all registered charts for updates
func (r *Registry) CheckForUpdates(ctx context.Context) ([]UpdateNotification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var notifications []UpdateNotification

	for key, info := range r.versions {
		oldVersion := info.LatestVersion
		
		if err := r.updateChartVersions(ctx, info); err != nil {
			r.log.Error(err, "Failed to update chart versions", "chart", info.ChartName)
			continue
		}

		// Check if update is available
		if oldVersion != "" && info.LatestVersion != oldVersion {
			notifications = append(notifications, UpdateNotification{
				ChartName:      info.ChartName,
				Repository:     info.Repository,
				CurrentVersion: info.CurrentVersion,
				NewVersion:     info.LatestVersion,
				Timestamp:      time.Now(),
			})
		}

		r.versions[key] = info
	}

	return notifications, nil
}

// SetCurrentVersion sets the currently deployed version of a chart
func (r *Registry) SetCurrentVersion(chartName, repository, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := r.getChartKey(chartName, repository)
	info, exists := r.versions[key]
	if !exists {
		return fmt.Errorf("chart %s not found in registry", chartName)
	}

	info.CurrentVersion = version
	info.UpdateAvailable = r.isUpdateAvailable(info.CurrentVersion, info.LatestVersion)
	
	return nil
}

// GetAvailableVersions returns all available versions for a chart
func (r *Registry) GetAvailableVersions(chartName, repository string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := r.getChartKey(chartName, repository)
	info, exists := r.versions[key]
	if !exists {
		return nil, fmt.Errorf("chart %s not found in registry", chartName)
	}

	return append([]string{}, info.AvailableVersions...), nil
}

// Private helper methods

func (r *Registry) updateChartVersions(ctx context.Context, info *ChartVersionInfo) error {
	// Search for chart in repository
	results, err := r.helmClient.Search(info.Repository, info.ChartName, "", 0)
	if err != nil {
		return fmt.Errorf("failed to search chart: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("chart %s not found in repository %s", info.ChartName, info.Repository)
	}

	// Extract versions
	var versions []string
	for _, result := range results {
		if result.Chart != nil && result.Chart.Metadata != nil {
			versions = append(versions, result.Chart.Metadata.Version)
		}
	}

	if len(versions) == 0 {
		return fmt.Errorf("no versions found for chart %s", info.ChartName)
	}

	// Sort versions using semver
	sortedVersions, err := r.sortVersions(versions)
	if err != nil {
		return fmt.Errorf("failed to sort versions: %w", err)
	}

	info.AvailableVersions = sortedVersions
	info.LatestVersion = sortedVersions[0] // First is the latest after sorting
	info.LastUpdated = time.Now()
	info.UpdateAvailable = r.isUpdateAvailable(info.CurrentVersion, info.LatestVersion)

	return nil
}

func (r *Registry) sortVersions(versions []string) ([]string, error) {
	var semverVersions []*semver.Version
	
	for _, v := range versions {
		sv, err := semver.NewVersion(v)
		if err != nil {
			r.log.V(1).Info("Skipping invalid version", "version", v, "error", err)
			continue
		}
		semverVersions = append(semverVersions, sv)
	}

	// Sort in descending order (latest first)
	semver.Sort(semverVersions)
	
	// Reverse to get descending order
	for i := 0; i < len(semverVersions)/2; i++ {
		j := len(semverVersions) - 1 - i
		semverVersions[i], semverVersions[j] = semverVersions[j], semverVersions[i]
	}

	sorted := make([]string, 0, len(semverVersions))
	for _, sv := range semverVersions {
		sorted = append(sorted, sv.Original())
	}

	return sorted, nil
}

func (r *Registry) isUpdateAvailable(current, latest string) bool {
	if current == "" || latest == "" {
		return false
	}

	currentVer, err1 := semver.NewVersion(current)
	latestVer, err2 := semver.NewVersion(latest)
	
	if err1 != nil || err2 != nil {
		return false
	}

	return latestVer.GreaterThan(currentVer)
}

func (r *Registry) getChartKey(chartName, repository string) string {
	return fmt.Sprintf("%s/%s", repository, chartName)
}

func (r *Registry) copyChartInfo(info *ChartVersionInfo) *ChartVersionInfo {
	copy := &ChartVersionInfo{
		ChartName:         info.ChartName,
		Repository:        info.Repository,
		CurrentVersion:    info.CurrentVersion,
		LatestVersion:     info.LatestVersion,
		AvailableVersions: append([]string{}, info.AvailableVersions...),
		LastUpdated:       info.LastUpdated,
		UpdateAvailable:   info.UpdateAvailable,
	}

	if info.Metadata != nil {
		copy.Metadata = make(map[string]interface{})
		for k, v := range info.Metadata {
			copy.Metadata[k] = v
		}
	}

	return copy
}

func (r *Registry) startAutoUpdate(interval time.Duration) {
	r.updateTicker = time.NewTicker(interval)
	
	go func() {
		for {
			select {
			case <-r.updateTicker.C:
				ctx := context.Background()
				if _, err := r.CheckForUpdates(ctx); err != nil {
					r.log.Error(err, "Auto-update check failed")
				}
			case <-r.stopCh:
				r.updateTicker.Stop()
				return
			}
		}
	}()
}

// Stop stops the registry and its background tasks
func (r *Registry) Stop() {
	close(r.stopCh)
}

// UpdateNotification represents a version update notification
type UpdateNotification struct {
	ChartName      string    `json:"chartName"`
	Repository     string    `json:"repository"`
	CurrentVersion string    `json:"currentVersion"`
	NewVersion     string    `json:"newVersion"`
	Timestamp      time.Time `json:"timestamp"`
}

// ToJSON converts the notification to JSON
func (n UpdateNotification) ToJSON() (string, error) {
	data, err := json.Marshal(n)
	return string(data), err
}
