/*
Copyright 2025.

Licensed under the MIT License.
*/

package helm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/helm"
)

func TestValueBuilder(t *testing.T) {
	vb := helm.NewValueBuilder()

	t.Run("BuildPrometheusValues", func(t *testing.T) {
		prometheusSpec := &observabilityv1beta1.PrometheusSpec{
			Enabled: true,
			Version: "v2.48.0",
			Replicas: ptr(int32(3)),
			Resources: &observabilityv1beta1.ResourceRequirements{
				Requests: map[string]string{
					"cpu":    "250m",
					"memory": "512Mi",
				},
				Limits: map[string]string{
					"cpu":    "1000m",
					"memory": "2Gi",
				},
			},
			Storage: &observabilityv1beta1.StorageSpec{
				Size: "100Gi",
				StorageClass: "fast-ssd",
			},
			Retention: "30d",
		}

		values, err := vb.BuildValues("prometheus", prometheusSpec)
		require.NoError(t, err)
		require.NotNil(t, values)

		// Check server configuration
		server, ok := values["server"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, int32(3), server["replicaCount"])
		assert.Equal(t, "30d", server["retention"])

		// Check resources
		resources, ok := server["resources"].(map[string]interface{})
		require.True(t, ok)
		assert.NotNil(t, resources["requests"])
		assert.NotNil(t, resources["limits"])

		// Check persistence
		persistence, ok := server["persistentVolume"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "100Gi", persistence["size"])
		assert.Equal(t, "fast-ssd", persistence["storageClass"])
	})

	t.Run("BuildGrafanaValues", func(t *testing.T) {
		grafanaSpec := &observabilityv1beta1.GrafanaSpec{
			Enabled: true,
			Version: "10.2.0",
			Replicas: ptr(int32(2)),
			AdminPassword: "secure-password",
			Ingress: &observabilityv1beta1.IngressSpec{
				Enabled: true,
				Host: "grafana.example.com",
				TLSSecret: "grafana-tls",
			},
		}

		values, err := vb.BuildValues("grafana", grafanaSpec)
		require.NoError(t, err)
		require.NotNil(t, values)

		assert.Equal(t, int32(2), values["replicaCount"])
		assert.Equal(t, "secure-password", values["adminPassword"])

		// Check ingress
		ingress, ok := values["ingress"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, true, ingress["enabled"])
	})

	t.Run("MergeValues", func(t *testing.T) {
		base := map[string]interface{}{
			"replicas": 1,
			"image": map[string]interface{}{
				"repository": "prometheus",
				"tag":        "v2.45.0",
			},
		}

		override := map[string]interface{}{
			"replicas": 3,
			"image": map[string]interface{}{
				"tag": "v2.48.0",
			},
		}

		merged := vb.MergeValues(base, override)
		assert.Equal(t, 3, merged["replicas"])
		
		image, ok := merged["image"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "prometheus", image["repository"])
		assert.Equal(t, "v2.48.0", image["tag"])
	})

	t.Run("GetDefaultValues", func(t *testing.T) {
		defaults := vb.GetDefaultValues("prometheus")
		require.NotNil(t, defaults)
		
		// Check some default values exist
		assert.Contains(t, defaults, "replicaCount")
		assert.Contains(t, defaults, "image")
		assert.Contains(t, defaults, "resources")
		assert.Contains(t, defaults, "persistence")
	})
}

func TestVersionManager(t *testing.T) {
	// Create a mock repository
	mockRepo := &mockRepository{
		versions: map[string][]string{
			"prometheus-community/prometheus": {"2.48.0", "2.47.2", "2.47.1", "2.46.0", "3.0.0-alpha.1"},
			"grafana/grafana": {"10.2.0", "10.1.5", "10.1.4", "10.0.0", "9.5.3"},
		},
	}

	vm := helm.NewVersionManager(mockRepo)

	t.Run("GetLatestVersion", func(t *testing.T) {
		ctx := context.Background()
		
		version, err := vm.GetLatestVersion(ctx, "prometheus-community/prometheus")
		require.NoError(t, err)
		assert.Equal(t, "2.48.0", version)

		version, err = vm.GetLatestVersion(ctx, "grafana/grafana")
		require.NoError(t, err)
		assert.Equal(t, "10.2.0", version)
	})

	t.Run("CompareVersions", func(t *testing.T) {
		result, err := vm.CompareVersions("2.47.0", "2.48.0")
		require.NoError(t, err)
		assert.Equal(t, -1, result)

		result, err = vm.CompareVersions("2.48.0", "2.48.0")
		require.NoError(t, err)
		assert.Equal(t, 0, result)

		result, err = vm.CompareVersions("2.49.0", "2.48.0")
		require.NoError(t, err)
		assert.Equal(t, 1, result)
	})

	t.Run("IsVersionCompatible", func(t *testing.T) {
		// Same major version - compatible
		compatible, err := vm.IsVersionCompatible("2.47.0", "2.48.0")
		require.NoError(t, err)
		assert.True(t, compatible)

		// Different major version - not compatible
		compatible, err = vm.IsVersionCompatible("2.48.0", "3.0.0")
		require.NoError(t, err)
		assert.False(t, compatible)
	})

	t.Run("GetUpgradePath", func(t *testing.T) {
		// Direct upgrade within same major
		path, err := vm.GetUpgradePath("2.46.0", "2.48.0")
		require.NoError(t, err)
		assert.Equal(t, []string{"2.48.0"}, path)

		// Major version upgrade
		path, err = vm.GetUpgradePath("2.48.0", "4.0.0")
		require.NoError(t, err)
		assert.Equal(t, []string{"3.0.0", "4.0.0"}, path)

		// Same version
		path, err = vm.GetUpgradePath("2.48.0", "2.48.0")
		require.NoError(t, err)
		assert.Empty(t, path)
	})
}

// Helper function to create pointers
func ptr[T any](v T) *T {
	return &v
}

// Mock repository for testing
type mockRepository struct {
	versions map[string][]string
}

func (m *mockRepository) AddRepository(ctx context.Context, name, url string, opts *helm.RepositoryOptions) error {
	return nil
}

func (m *mockRepository) UpdateRepository(ctx context.Context, name string) error {
	return nil
}

func (m *mockRepository) RemoveRepository(ctx context.Context, name string) error {
	return nil
}

func (m *mockRepository) ListRepositories(ctx context.Context) ([]*helm.RepositoryInfo, error) {
	return nil, nil
}

func (m *mockRepository) SearchCharts(ctx context.Context, keyword string) ([]*helm.ChartInfo, error) {
	return nil, nil
}

func (m *mockRepository) GetChartVersions(ctx context.Context, chartName string) ([]string, error) {
	if versions, ok := m.versions[chartName]; ok {
		return versions, nil
	}
	return nil, fmt.Errorf("chart not found")
}
