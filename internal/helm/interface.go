/*
Copyright 2025.

Licensed under the MIT License.
*/

package helm

import (
	"context"
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
)

// ChartInfo contains information about a Helm chart
type ChartInfo struct {
	Name        string
	Version     string
	Repository  string
	Description string
}

// ReleaseInfo contains information about a Helm release
type ReleaseInfo struct {
	Name         string
	Namespace    string
	Version      int
	Status       release.Status
	LastDeployed time.Time
	Chart        *ChartInfo
	Values       map[string]interface{}
}

// HelmClient defines the interface for Helm operations
type HelmClient interface {
	// InstallChart installs a Helm chart
	InstallChart(ctx context.Context, opts *InstallOptions) (*ReleaseInfo, error)

	// UpgradeChart upgrades an existing Helm release
	UpgradeChart(ctx context.Context, opts *UpgradeOptions) (*ReleaseInfo, error)

	// UninstallChart uninstalls a Helm release
	UninstallChart(ctx context.Context, releaseName, namespace string) error

	// GetRelease retrieves information about a Helm release
	GetRelease(ctx context.Context, releaseName, namespace string) (*ReleaseInfo, error)

	// ListReleases lists all Helm releases in a namespace
	ListReleases(ctx context.Context, namespace string) ([]*ReleaseInfo, error)

	// RollbackRelease rolls back a release to a previous version
	RollbackRelease(ctx context.Context, releaseName, namespace string, version int) error

	// GetValues retrieves the values for a release
	GetValues(ctx context.Context, releaseName, namespace string) (map[string]interface{}, error)

	// TestRelease runs Helm tests for a release
	TestRelease(ctx context.Context, releaseName, namespace string) error

	// RenderChart renders chart templates without installing
	RenderChart(ctx context.Context, opts *RenderOptions) (string, error)
}

// InstallOptions contains options for installing a Helm chart
type InstallOptions struct {
	ReleaseName      string
	Namespace        string
	ChartRef         string
	Version          string
	Values           map[string]interface{}
	ValuesFiles      []string
	CreateNamespace  bool
	Wait             bool
	WaitForJobs      bool
	Timeout          time.Duration
	DisableHooks     bool
	DisableOpenAPI   bool
	SkipCRDs         bool
	SubNotes         bool
	Description      string
	Atomic           bool
	Force            bool
	DependencyUpdate bool
}

// UpgradeOptions contains options for upgrading a Helm release
type UpgradeOptions struct {
	ReleaseName     string
	Namespace       string
	ChartRef        string
	Version         string
	Values          map[string]interface{}
	ValuesFiles     []string
	ReuseValues     bool
	ResetValues     bool
	Wait            bool
	WaitForJobs     bool
	Timeout         time.Duration
	DisableHooks    bool
	DisableOpenAPI  bool
	SkipCRDs        bool
	SubNotes        bool
	Description     string
	Atomic          bool
	CleanupOnFail   bool
	Force           bool
	Install         bool // Create if doesn't exist
	MaxHistory      int
}

// RenderOptions contains options for rendering a Helm chart
type RenderOptions struct {
	ChartRef     string
	Version      string
	Values       map[string]interface{}
	ValuesFiles  []string
	Namespace    string
	ReleaseName  string
	ShowNotes    bool
	IncludeCRDs  bool
	Validate     bool
	KubeVersion  string
}

// Repository defines the interface for Helm repository operations
type Repository interface {
	// AddRepository adds a new Helm repository
	AddRepository(ctx context.Context, name, url string, opts *RepositoryOptions) error

	// UpdateRepository updates repository index
	UpdateRepository(ctx context.Context, name string) error

	// RemoveRepository removes a Helm repository
	RemoveRepository(ctx context.Context, name string) error

	// ListRepositories lists all configured repositories
	ListRepositories(ctx context.Context) ([]*RepositoryInfo, error)

	// SearchCharts searches for charts in repositories
	SearchCharts(ctx context.Context, keyword string) ([]*ChartInfo, error)

	// GetChartVersions gets available versions for a chart
	GetChartVersions(ctx context.Context, chartName string) ([]string, error)
}

// RepositoryOptions contains options for adding a repository
type RepositoryOptions struct {
	Username             string
	Password             string
	CertFile             string
	KeyFile              string
	CAFile               string
	InsecureSkipTLSVerify bool
	PassCredentialsAll   bool
}

// RepositoryInfo contains information about a Helm repository
type RepositoryInfo struct {
	Name   string
	URL    string
	Status string
}

// ValueBuilder provides an interface for building Helm values
type ValueBuilder interface {
	// BuildValues builds Helm values for a component
	BuildValues(componentType string, spec interface{}) (map[string]interface{}, error)

	// MergeValues merges multiple value sets
	MergeValues(base, override map[string]interface{}) map[string]interface{}

	// ValidateValues validates Helm values against a schema
	ValidateValues(componentType string, values map[string]interface{}) error

	// GetDefaultValues returns default values for a component
	GetDefaultValues(componentType string) map[string]interface{}
}

// ChartManager manages Helm charts lifecycle
type ChartManager interface {
	// DownloadChart downloads a chart to local cache
	DownloadChart(ctx context.Context, chartRef, version string) (*chart.Chart, error)

	// LoadChart loads a chart from local path
	LoadChart(path string) (*chart.Chart, error)

	// PackageChart packages a chart directory
	PackageChart(path, destination string) (string, error)

	// ExtractChart extracts a chart archive
	ExtractChart(archive, destination string) error

	// ValidateChart validates a chart
	ValidateChart(chart *chart.Chart) error
}

// VersionManager manages chart versions
type VersionManager interface {
	// GetLatestVersion gets the latest stable version of a chart
	GetLatestVersion(ctx context.Context, chartName string) (string, error)

	// CompareVersions compares two versions
	CompareVersions(v1, v2 string) (int, error)

	// IsVersionCompatible checks if a version is compatible
	IsVersionCompatible(current, target string) (bool, error)

	// GetUpgradePath returns the upgrade path between versions
	GetUpgradePath(current, target string) ([]string, error)
}

// HooksManager manages Helm hooks
type HooksManager interface {
	// EnableHooks enables specific hooks
	EnableHooks(hooks ...string)

	// DisableHooks disables specific hooks
	DisableHooks(hooks ...string)

	// GetHookStatus gets the status of hook execution
	GetHookStatus(ctx context.Context, releaseName, namespace string) (map[string]string, error)

	// RunHook manually runs a specific hook
	RunHook(ctx context.Context, releaseName, namespace, hookName string) error
}

// TestManager manages Helm test operations
type TestManager interface {
	// RunTests runs all tests for a release
	RunTests(ctx context.Context, releaseName, namespace string) (*TestResults, error)

	// GetTestStatus gets the status of test execution
	GetTestStatus(ctx context.Context, releaseName, namespace string) (*TestResults, error)

	// CleanupTests cleans up test resources
	CleanupTests(ctx context.Context, releaseName, namespace string) error
}

// TestResults contains Helm test results
type TestResults struct {
	ReleaseName string
	Namespace   string
	Tests       []TestResult
	Status      string
	StartTime   time.Time
	EndTime     time.Time
}

// TestResult contains a single test result
type TestResult struct {
	Name      string
	Status    string
	Info      string
	Started   time.Time
	Completed time.Time
}

// SecretManager manages Helm secrets
type SecretManager interface {
	// GetReleaseSecrets gets secrets for a release
	GetReleaseSecrets(ctx context.Context, releaseName, namespace string) ([]*corev1.Secret, error)

	// BackupSecrets backs up release secrets
	BackupSecrets(ctx context.Context, releaseName, namespace string) (map[string][]byte, error)

	// RestoreSecrets restores release secrets
	RestoreSecrets(ctx context.Context, releaseName, namespace string, data map[string][]byte) error
}
