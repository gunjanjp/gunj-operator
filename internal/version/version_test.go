// Package version provides tests for version management
package version_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gunjanjp/gunj-operator/internal/helm"
	"github.com/gunjanjp/gunj-operator/internal/version"
)

// Mock Helm client for testing
type mockHelmClient struct {
	charts map[string][]string
}

func (m *mockHelmClient) Install(release, chart string, values map[string]interface{}) error {
	return nil
}

func (m *mockHelmClient) Upgrade(release, chart string, values map[string]interface{}) error {
	return nil
}

func (m *mockHelmClient) Uninstall(release string) error {
	return nil
}

func (m *mockHelmClient) GetValues(release string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockHelmClient) List() ([]helm.Release, error) {
	return nil, nil
}

func (m *mockHelmClient) AddRepository(name, url string) error {
	return nil
}

func (m *mockHelmClient) UpdateRepositories() error {
	return nil
}

func (m *mockHelmClient) Search(repo, chart, version string, maxResults int) ([]helm.ChartVersion, error) {
	versions, exists := m.charts[chart]
	if !exists {
		return nil, nil
	}

	var results []helm.ChartVersion
	for _, v := range versions {
		results = append(results, helm.ChartVersion{
			Chart: &helm.Chart{
				Metadata: &helm.Metadata{
					Name:    chart,
					Version: v,
				},
			},
		})
	}
	return results, nil
}

func (m *mockHelmClient) GetChart(repo, chart, version string) (*helm.Chart, error) {
	return &helm.Chart{
		Metadata: &helm.Metadata{
			Name:    chart,
			Version: version,
		},
	}, nil
}

// Test Registry
func TestVersionRegistry(t *testing.T) {
	// Create mock Helm client
	mockHelm := &mockHelmClient{
		charts: map[string][]string{
			"prometheus": {"2.48.0", "2.47.0", "2.46.0", "2.45.0"},
			"grafana":    {"10.2.0", "10.1.0", "10.0.0", "9.5.0"},
		},
	}

	// Create registry
	registry, err := version.NewRegistry(version.RegistryOptions{
		HelmClient:       mockHelm,
		EnableAutoUpdate: false,
	})
	require.NoError(t, err)

	// Test chart registration
	ctx := context.Background()
	err = registry.RegisterChart(ctx, "prometheus", "bitnami")
	assert.NoError(t, err)

	// Test getting chart info
	info, err := registry.GetChartInfo("prometheus", "bitnami")
	assert.NoError(t, err)
	assert.Equal(t, "prometheus", info.ChartName)
	assert.Equal(t, "2.48.0", info.LatestVersion)
	assert.Len(t, info.AvailableVersions, 4)

	// Test setting current version
	err = registry.SetCurrentVersion("prometheus", "bitnami", "2.46.0")
	assert.NoError(t, err)

	info, err = registry.GetChartInfo("prometheus", "bitnami")
	assert.NoError(t, err)
	assert.Equal(t, "2.46.0", info.CurrentVersion)
	assert.True(t, info.UpdateAvailable)
}

// Test Constraints
func TestVersionConstraints(t *testing.T) {
	// Test basic constraint
	constraint, err := version.NewConstraint(">=2.45.0 <3.0.0")
	require.NoError(t, err)

	// Test valid versions
	valid, err := constraint.Check("2.48.0")
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = constraint.Check("2.45.0")
	assert.NoError(t, err)
	assert.True(t, valid)

	// Test invalid versions
	valid, err = constraint.Check("2.44.0")
	assert.NoError(t, err)
	assert.False(t, valid)

	valid, err = constraint.Check("3.0.0")
	assert.NoError(t, err)
	assert.False(t, valid)

	// Test validator
	validator := version.NewValidator()
	err = validator.AddConstraint("prometheus", ">=2.45.0")
	assert.NoError(t, err)

	err = validator.ValidateVersion("prometheus", "2.48.0")
	assert.NoError(t, err)

	err = validator.ValidateVersion("prometheus", "2.44.0")
	assert.Error(t, err)
}

// Test Compatibility Matrix
func TestCompatibilityMatrix(t *testing.T) {
	matrix := version.NewCompatibilityMatrix()
	matrix.LoadDefault()

	// Test compatible versions
	components := []version.ComponentVersion{
		{Component: "prometheus", Version: "2.48.0"},
		{Component: "grafana", Version: "10.2.0"},
		{Component: "loki", Version: "2.9.0"},
	}

	compatible, issues := matrix.CheckCompatibility(components)
	assert.True(t, compatible)
	assert.Empty(t, issues)

	// Test incompatible versions
	components = []version.ComponentVersion{
		{Component: "prometheus", Version: "2.48.0"},
		{Component: "grafana", Version: "8.0.0"}, // Too old
	}

	compatible, issues = matrix.CheckCompatibility(components)
	assert.False(t, compatible)
	assert.NotEmpty(t, issues)

	// Test getting compatible versions
	otherComponents := []version.ComponentVersion{
		{Component: "prometheus", Version: "2.48.0"},
	}
	availableVersions := []string{"10.2.0", "10.1.0", "9.5.0", "8.0.0"}

	compatibleVersions := matrix.GetCompatibleVersions("grafana", otherComponents, availableVersions)
	assert.Contains(t, compatibleVersions, "10.2.0")
	assert.Contains(t, compatibleVersions, "10.1.0")
	assert.NotContains(t, compatibleVersions, "8.0.0")
}

// Test Version Pinning
func TestVersionPinning(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	pinning := version.NewPinningManager(client, "")

	// Test pinning a version
	ctx := context.Background()
	pin := &version.VersionPin{
		Component: "prometheus",
		Version:   "2.48.0",
		Reason:    "Stability requirement for production",
		PinnedBy:  "test@example.com",
	}

	err := pinning.PinVersion(ctx, pin)
	assert.NoError(t, err)

	// Test getting pinned version
	pinnedVersion, exists := pinning.GetPin("prometheus")
	assert.True(t, exists)
	assert.Equal(t, "2.48.0", pinnedVersion.Version)

	// Test applying pin
	effectiveVersion, decision := pinning.ApplyToVersion("prometheus", "2.49.0")
	assert.Equal(t, "2.48.0", effectiveVersion)
	assert.NotNil(t, decision.Pin)

	// Test unpinning
	err = pinning.UnpinVersion(ctx, "prometheus", "test@example.com")
	assert.NoError(t, err)

	_, exists = pinning.GetPin("prometheus")
	assert.False(t, exists)
}

// Test Version History
func TestVersionHistory(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	storage := version.NewInMemoryHistoryStorage()

	history := version.NewHistoryTracker(client, storage, 100)

	// Record version change
	ctx := context.Background()
	entry := version.VersionHistoryEntry{
		Platform: version.PlatformReference{
			Name:      "test-platform",
			Namespace: "default",
			UID:       "12345",
		},
		Component:   "prometheus",
		FromVersion: "2.47.0",
		ToVersion:   "2.48.0",
		Action:      version.VersionActionUpgrade,
		Status:      version.VersionStatusSucceeded,
	}

	err := history.RecordChange(ctx, entry)
	assert.NoError(t, err)

	// Get history
	filter := version.HistoryFilter{
		Component: "prometheus",
		Limit:     10,
	}
	entries := history.GetHistory(filter)
	assert.Len(t, entries, 1)
	assert.Equal(t, "2.48.0", entries[0].ToVersion)

	// Get latest version
	latestVersion, exists := history.GetLatestVersion(entry.Platform, "prometheus")
	assert.True(t, exists)
	assert.Equal(t, "2.48.0", latestVersion)

	// Generate report
	report, err := history.GenerateReport(filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, report.TotalCount)
	assert.Equal(t, 1, report.Statistics.TotalChanges)
	assert.Equal(t, 100.0, report.Statistics.SuccessRate)
}

// Test Notification Manager
func TestNotificationManager(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create components
	mockHelm := &mockHelmClient{
		charts: map[string][]string{
			"prometheus": {"2.49.0", "2.48.0"},
		},
	}

	registry, _ := version.NewRegistry(version.RegistryOptions{
		HelmClient: mockHelm,
	})

	matrix := version.NewCompatibilityMatrix()
	matrix.LoadDefault()

	// Create notification manager
	nm, err := version.NewNotificationManager(version.NotificationManagerOptions{
		Client:        client,
		Registry:      registry,
		Compatibility: matrix,
	})
	require.NoError(t, err)

	// Create test notification
	notification := &version.UpgradeNotification{
		Component:        "prometheus",
		CurrentVersion:   "2.48.0",
		AvailableVersion: "2.49.0",
		Type:             version.UpgradeTypeMinor,
		Priority:         version.UpgradePriorityMedium,
		Compatibility: version.CompatibilityInfo{
			Compatible: true,
		},
	}

	// Process notification
	ctx := context.Background()
	err = nm.ProcessNotification(ctx, notification)
	assert.NoError(t, err)

	// Check history
	history := nm.GetHistory(1)
	assert.Len(t, history, 1)
}

// Test Version Manager Integration
func TestVersionManagerIntegration(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create mock Helm client
	mockHelm := &mockHelmClient{
		charts: map[string][]string{
			"prometheus": {"2.49.0", "2.48.0", "2.47.0"},
			"grafana":    {"10.3.0", "10.2.0", "10.1.0"},
		},
	}

	// Create version manager
	config := &version.ManagerConfig{
		HelmClient:          mockHelm,
		KubernetesClient:    client,
		Scheme:              scheme,
		EnableNotifications: true,
		EnableHistory:       true,
		EnableTesting:       false,
	}

	manager, err := version.NewManager(config)
	require.NoError(t, err)
	defer manager.Stop()

	// Register charts
	ctx := context.Background()
	err = manager.RegisterChart(ctx, "prometheus", "bitnami")
	assert.NoError(t, err)

	// Get recommended version
	recommended, err := manager.GetRecommendedVersion(ctx, "prometheus", map[string]string{
		"prometheus": ">=2.48.0",
	})
	assert.NoError(t, err)
	assert.Equal(t, "2.49.0", recommended)

	// Validate upgrade
	platform := version.PlatformReference{
		Name:      "test",
		Namespace: "default",
		UID:       "123",
	}

	validation, err := manager.ValidateUpgrade(ctx, platform, "prometheus", "2.49.0")
	assert.NoError(t, err)
	assert.True(t, validation.Valid)
}

// Test Recommended Versions
func TestRecommendedVersions(t *testing.T) {
	versions := []string{"2.49.0", "2.48.1", "2.48.0", "2.48.0-rc.1", "2.47.0"}
	
	recommended := version.RecommendedVersions(versions, 3)
	assert.Len(t, recommended, 3)
	assert.Equal(t, "2.49.0", recommended[0])
	assert.Equal(t, "2.48.1", recommended[1])
	assert.Equal(t, "2.48.0", recommended[2])
	
	// Should not include pre-release
	assert.NotContains(t, recommended, "2.48.0-rc.1")
}

// Benchmark Registry Performance
func BenchmarkVersionRegistry(b *testing.B) {
	mockHelm := &mockHelmClient{
		charts: map[string][]string{
			"prometheus": generateVersions(100),
		},
	}

	registry, _ := version.NewRegistry(version.RegistryOptions{
		HelmClient: mockHelm,
	})

	ctx := context.Background()
	registry.RegisterChart(ctx, "prometheus", "bitnami")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetChartInfo("prometheus", "bitnami")
	}
}

// Helper to generate test versions
func generateVersions(count int) []string {
	versions := make([]string, count)
	for i := 0; i < count; i++ {
		versions[i] = fmt.Sprintf("2.%d.0", count-i)
	}
	return versions
}
