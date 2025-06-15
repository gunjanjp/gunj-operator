/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// TestMultiVersionConversionChain tests conversion through multiple versions
func TestMultiVersionConversionChain(t *testing.T) {
	// Simulate a chain: v1alpha1 -> v1beta1 -> v1 (future) -> v1beta1 -> v1alpha1
	tests := []struct {
		name     string
		original *v1alpha1.ObservabilityPlatform
		validate func(t *testing.T, original, final *v1alpha1.ObservabilityPlatform)
	}{
		{
			name:     "simple chain conversion",
			original: createMinimalV1Alpha1Platform(),
			validate: func(t *testing.T, original, final *v1alpha1.ObservabilityPlatform) {
				assert.Equal(t, original.Name, final.Name)
				assert.Equal(t, original.Namespace, final.Namespace)
				assert.Equal(t, original.Spec.Components.Prometheus.Enabled, final.Spec.Components.Prometheus.Enabled)
			},
		},
		{
			name:     "complex chain conversion",
			original: createComplexV1Alpha1Platform(),
			validate: func(t *testing.T, original, final *v1alpha1.ObservabilityPlatform) {
				// Basic metadata
				assert.Equal(t, original.Name, final.Name)
				assert.Equal(t, original.Namespace, final.Namespace)
				
				// Components should be preserved
				assert.Equal(t, original.Spec.Components.Prometheus.Version, final.Spec.Components.Prometheus.Version)
				assert.Equal(t, original.Spec.Components.Grafana.Version, final.Spec.Components.Grafana.Version)
				assert.Equal(t, original.Spec.Components.Loki.Version, final.Spec.Components.Loki.Version)
				assert.Equal(t, original.Spec.Components.Tempo.Version, final.Spec.Components.Tempo.Version)
				
				// Global config
				assert.Equal(t, original.Spec.Global.LogLevel, final.Spec.Global.LogLevel)
				assert.Equal(t, original.Spec.Global.ExternalLabels, final.Spec.Global.ExternalLabels)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: v1alpha1 -> v1beta1
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tt.original.ConvertTo(v1beta1Platform)
			require.NoError(t, err)

			// Step 2: Simulate v1beta1 -> v1 (future version)
			// For now, we'll just deep copy and modify
			v1Platform := v1beta1Platform.DeepCopy()
			v1Platform.APIVersion = "observability.io/v1"
			
			// Step 3: Simulate v1 -> v1beta1
			v1beta1Again := v1Platform.DeepCopy()
			v1beta1Again.APIVersion = "observability.io/v1beta1"

			// Step 4: v1beta1 -> v1alpha1
			final := &v1alpha1.ObservabilityPlatform{}
			err = final.ConvertFrom(v1beta1Again)
			require.NoError(t, err)

			// Validate
			tt.validate(t, tt.original, final)
		})
	}
}

// TestCrossVersionCompatibility tests compatibility between versions
func TestCrossVersionCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		v1alpha1Obj string
		v1beta1Obj  string
		compatible  bool
	}{
		{
			name: "compatible basic objects",
			v1alpha1Obj: `
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: test-platform
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
`,
			v1beta1Obj: `
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: test-platform
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
`,
			compatible: true,
		},
		{
			name: "v1beta1 with extra fields",
			v1alpha1Obj: `
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: test-platform
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
`,
			v1beta1Obj: `
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: test-platform
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
      externalLabels:
        cluster: production
      walCompression: true
`,
			compatible: true, // Should be compatible, extra fields are ignored in v1alpha1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse v1alpha1
			var v1alpha1Platform v1alpha1.ObservabilityPlatform
			err := yaml.Unmarshal([]byte(tt.v1alpha1Obj), &v1alpha1Platform)
			require.NoError(t, err)

			// Parse v1beta1
			var v1beta1Platform v1beta1.ObservabilityPlatform
			err = yaml.Unmarshal([]byte(tt.v1beta1Obj), &v1beta1Platform)
			require.NoError(t, err)

			// Convert v1alpha1 to v1beta1
			converted := &v1beta1.ObservabilityPlatform{}
			err = v1alpha1Platform.ConvertTo(converted)
			require.NoError(t, err)

			if tt.compatible {
				// Basic fields should match
				assert.Equal(t, v1beta1Platform.Name, converted.Name)
				assert.Equal(t, v1beta1Platform.Namespace, converted.Namespace)
				assert.Equal(t, v1beta1Platform.Spec.Components.Prometheus.Enabled, converted.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, v1beta1Platform.Spec.Components.Prometheus.Version, converted.Spec.Components.Prometheus.Version)
			}
		})
	}
}

// TestVersionMigrationScenarios tests real-world version migration scenarios
func TestVersionMigrationScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		description string
		setup       func() *v1alpha1.ObservabilityPlatform
		migrate     func(*v1alpha1.ObservabilityPlatform) (*v1beta1.ObservabilityPlatform, error)
		validate    func(t *testing.T, original *v1alpha1.ObservabilityPlatform, migrated *v1beta1.ObservabilityPlatform)
	}{
		{
			name:        "production cluster migration",
			description: "Migrating a production cluster from v1alpha1 to v1beta1",
			setup: func() *v1alpha1.ObservabilityPlatform {
				return &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "production-observability",
						Namespace: "monitoring",
						Labels: map[string]string{
							"environment": "production",
							"criticality": "high",
						},
						Annotations: map[string]string{
							"migration-notes": "Production cluster - handle with care",
						},
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled:  true,
								Version:  "v2.47.0",
								Replicas: 3,
								Resources: v1alpha1.ResourceRequirements{
									Requests: v1alpha1.ResourceList{
										CPU:    "2",
										Memory: "8Gi",
									},
									Limits: v1alpha1.ResourceList{
										CPU:    "4",
										Memory: "16Gi",
									},
								},
								Storage: &v1alpha1.StorageConfig{
									Size:             "1Ti",
									StorageClassName: "fast-ssd",
								},
								Retention: "90d",
								RemoteWrite: []v1alpha1.RemoteWriteSpec{
									{
										URL: "https://long-term-storage.example.com/write",
										Headers: map[string]string{
											"X-Auth-Token": "production-token",
										},
									},
								},
							},
							Grafana: &v1alpha1.GrafanaSpec{
								Enabled:       true,
								Version:       "9.5.0",
								Replicas:      2,
								AdminPassword: "encrypted-password",
								Ingress: &v1alpha1.IngressConfig{
									Enabled:   true,
									ClassName: "nginx",
									Host:      "grafana.production.example.com",
									TLS: &v1alpha1.IngressTLS{
										Enabled:    true,
										SecretName: "grafana-tls",
									},
								},
							},
						},
						HighAvailability: &v1alpha1.HighAvailabilityConfig{
							Enabled:     true,
							MinReplicas: 3,
						},
						Backup: &v1alpha1.BackupConfig{
							Enabled:       true,
							Schedule:      "0 2 * * *",
							RetentionDays: 30,
							Destination: v1alpha1.BackupDestination{
								Type: "s3",
								S3: &v1alpha1.S3Config{
									BucketName: "observability-backups-prod",
									Region:     "us-east-1",
								},
							},
						},
					},
					Status: v1alpha1.ObservabilityPlatformStatus{
						Phase:              "Ready",
						ObservedGeneration: 42,
						Message:            "All components healthy",
						ComponentStatus: map[string]v1alpha1.ComponentStatus{
							"prometheus": {
								Phase:         "Ready",
								Version:       "v2.47.0",
								ReadyReplicas: 3,
								TotalReplicas: 3,
							},
							"grafana": {
								Phase:         "Ready",
								Version:       "9.5.0",
								ReadyReplicas: 2,
								TotalReplicas: 2,
							},
						},
					},
				}
			},
			migrate: func(v1alpha1Platform *v1alpha1.ObservabilityPlatform) (*v1beta1.ObservabilityPlatform, error) {
				// Simulate migration process
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				
				// First, do standard conversion
				if err := v1alpha1Platform.ConvertTo(v1beta1Platform); err != nil {
					return nil, err
				}
				
				// Then apply migration-specific changes
				// For example, upgrade versions
				if v1beta1Platform.Spec.Components.Prometheus != nil {
					v1beta1Platform.Spec.Components.Prometheus.Version = "v2.48.0"
				}
				if v1beta1Platform.Spec.Components.Grafana != nil {
					v1beta1Platform.Spec.Components.Grafana.Version = "10.2.0"
				}
				
				// Add new v1beta1 features
				if v1beta1Platform.Spec.Components.Prometheus != nil {
					v1beta1Platform.Spec.Components.Prometheus.ExternalLabels = map[string]string{
						"cluster":     "production",
						"environment": "prod",
					}
				}
				
				return v1beta1Platform, nil
			},
			validate: func(t *testing.T, original *v1alpha1.ObservabilityPlatform, migrated *v1beta1.ObservabilityPlatform) {
				// Verify critical production settings are preserved
				assert.Equal(t, "production-observability", migrated.Name)
				assert.Equal(t, "monitoring", migrated.Namespace)
				assert.Equal(t, "production", migrated.Labels["environment"])
				assert.Equal(t, "high", migrated.Labels["criticality"])
				
				// Verify resources are preserved
				assert.Equal(t, "2", migrated.Spec.Components.Prometheus.Resources.Requests.Cpu().String())
				assert.Equal(t, "8Gi", migrated.Spec.Components.Prometheus.Resources.Requests.Memory().String())
				
				// Verify HA settings
				assert.True(t, migrated.Spec.HighAvailability.Enabled)
				assert.Equal(t, int32(3), migrated.Spec.HighAvailability.MinReplicas)
				
				// Verify backup configuration
				assert.True(t, migrated.Spec.Backup.Enabled)
				assert.Equal(t, "0 2 * * *", migrated.Spec.Backup.Schedule)
				
				// Verify versions were upgraded
				assert.Equal(t, "v2.48.0", migrated.Spec.Components.Prometheus.Version)
				assert.Equal(t, "10.2.0", migrated.Spec.Components.Grafana.Version)
			},
		},
		{
			name:        "development to staging promotion",
			description: "Promoting configuration from dev to staging with version upgrade",
			setup: func() *v1alpha1.ObservabilityPlatform {
				return &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dev-observability",
						Namespace: "dev-monitoring",
						Labels: map[string]string{
							"environment": "development",
						},
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled:  true,
								Version:  "v2.45.0",
								Replicas: 1,
								Resources: v1alpha1.ResourceRequirements{
									Requests: v1alpha1.ResourceList{
										CPU:    "500m",
										Memory: "2Gi",
									},
								},
							},
							Grafana: &v1alpha1.GrafanaSpec{
								Enabled:  true,
								Version:  "9.0.0",
								Replicas: 1,
							},
						},
					},
				}
			},
			migrate: func(v1alpha1Platform *v1alpha1.ObservabilityPlatform) (*v1beta1.ObservabilityPlatform, error) {
				// Convert to v1beta1
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				if err := v1alpha1Platform.ConvertTo(v1beta1Platform); err != nil {
					return nil, err
				}
				
				// Update for staging environment
				v1beta1Platform.Name = "staging-observability"
				v1beta1Platform.Namespace = "staging-monitoring"
				v1beta1Platform.Labels["environment"] = "staging"
				
				// Scale up for staging
				if v1beta1Platform.Spec.Components.Prometheus != nil {
					v1beta1Platform.Spec.Components.Prometheus.Replicas = 2
					v1beta1Platform.Spec.Components.Prometheus.Resources.Requests["cpu"] = resource.MustParse("1")
					v1beta1Platform.Spec.Components.Prometheus.Resources.Requests["memory"] = resource.MustParse("4Gi")
				}
				
				if v1beta1Platform.Spec.Components.Grafana != nil {
					v1beta1Platform.Spec.Components.Grafana.Replicas = 2
				}
				
				// Enable HA for staging
				v1beta1Platform.Spec.HighAvailability = &v1beta1.HighAvailabilityConfig{
					Enabled:     true,
					MinReplicas: 2,
				}
				
				return v1beta1Platform, nil
			},
			validate: func(t *testing.T, original *v1alpha1.ObservabilityPlatform, migrated *v1beta1.ObservabilityPlatform) {
				// Verify environment promotion
				assert.Equal(t, "staging-observability", migrated.Name)
				assert.Equal(t, "staging-monitoring", migrated.Namespace)
				assert.Equal(t, "staging", migrated.Labels["environment"])
				
				// Verify scaling
				assert.Equal(t, int32(2), migrated.Spec.Components.Prometheus.Replicas)
				assert.Equal(t, int32(2), migrated.Spec.Components.Grafana.Replicas)
				
				// Verify resource increases
				assert.Equal(t, "1", migrated.Spec.Components.Prometheus.Resources.Requests.Cpu().String())
				assert.Equal(t, "4Gi", migrated.Spec.Components.Prometheus.Resources.Requests.Memory().String())
				
				// Verify HA enabled
				assert.True(t, migrated.Spec.HighAvailability.Enabled)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Scenario: %s", scenario.description)
			
			// Setup
			original := scenario.setup()
			
			// Migrate
			migrated, err := scenario.migrate(original)
			require.NoError(t, err)
			
			// Validate
			scenario.validate(t, original, migrated)
			
			// Test reverse migration
			reversed := &v1alpha1.ObservabilityPlatform{}
			err = reversed.ConvertFrom(migrated)
			require.NoError(t, err)
			
			// Basic validation of reverse migration
			assert.Equal(t, migrated.Name, reversed.Name)
			assert.Equal(t, migrated.Namespace, reversed.Namespace)
		})
	}
}

// TestVersionedAPIResponses tests handling of different API versions in responses
func TestVersionedAPIResponses(t *testing.T) {
	// Create platforms in different versions
	v1alpha1JSON := `{
		"apiVersion": "observability.io/v1alpha1",
		"kind": "ObservabilityPlatform",
		"metadata": {
			"name": "test-platform",
			"namespace": "default"
		},
		"spec": {
			"components": {
				"prometheus": {
					"enabled": true,
					"version": "v2.48.0"
				}
			}
		}
	}`

	v1beta1JSON := `{
		"apiVersion": "observability.io/v1beta1",
		"kind": "ObservabilityPlatform",
		"metadata": {
			"name": "test-platform",
			"namespace": "default"
		},
		"spec": {
			"components": {
				"prometheus": {
					"enabled": true,
					"version": "v2.48.0",
					"externalLabels": {
						"cluster": "test"
					}
				}
			}
		}
	}`

	// Test unmarshaling with type detection
	t.Run("unmarshal v1alpha1", func(t *testing.T) {
		var obj runtime.Object
		
		// First, detect version
		var typeMeta metav1.TypeMeta
		err := json.Unmarshal([]byte(v1alpha1JSON), &typeMeta)
		require.NoError(t, err)
		
		assert.Equal(t, "observability.io/v1alpha1", typeMeta.APIVersion)
		
		// Then unmarshal to correct type
		var v1alpha1Platform v1alpha1.ObservabilityPlatform
		err = json.Unmarshal([]byte(v1alpha1JSON), &v1alpha1Platform)
		require.NoError(t, err)
		
		obj = &v1alpha1Platform
		assert.NotNil(t, obj)
	})

	t.Run("unmarshal v1beta1", func(t *testing.T) {
		var obj runtime.Object
		
		// First, detect version
		var typeMeta metav1.TypeMeta
		err := json.Unmarshal([]byte(v1beta1JSON), &typeMeta)
		require.NoError(t, err)
		
		assert.Equal(t, "observability.io/v1beta1", typeMeta.APIVersion)
		
		// Then unmarshal to correct type
		var v1beta1Platform v1beta1.ObservabilityPlatform
		err = json.Unmarshal([]byte(v1beta1JSON), &v1beta1Platform)
		require.NoError(t, err)
		
		obj = &v1beta1Platform
		assert.NotNil(t, obj)
	})
}

// TestConversionWithVersionAnnotations tests version tracking through annotations
func TestConversionWithVersionAnnotations(t *testing.T) {
	platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "version-tracking",
			Namespace: "default",
			Annotations: map[string]string{
				"observability.io/original-version": "v1alpha1",
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}

	// Convert to v1beta1
	v1beta1Platform := &v1beta1.ObservabilityPlatform{}
	err := platform.ConvertTo(v1beta1Platform)
	require.NoError(t, err)

	// Check version annotations
	assert.Equal(t, "v1alpha1", v1beta1Platform.Annotations["observability.io/original-version"])
	assert.Equal(t, "v1alpha1", v1beta1Platform.Annotations["observability.io/converted-from"])

	// Convert back
	v1alpha1Again := &v1alpha1.ObservabilityPlatform{}
	err = v1alpha1Again.ConvertFrom(v1beta1Platform)
	require.NoError(t, err)

	// Check version history
	assert.Equal(t, "v1alpha1", v1alpha1Again.Annotations["observability.io/original-version"])
	assert.Equal(t, "v1beta1", v1alpha1Again.Annotations["observability.io/converted-from"])
}

// TestParallelVersionSupport tests supporting multiple versions simultaneously
func TestParallelVersionSupport(t *testing.T) {
	// Create the same logical platform in different versions
	platforms := []runtime.Object{
		&v1alpha1.ObservabilityPlatform{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "observability.io/v1alpha1",
				Kind:       "ObservabilityPlatform",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multi-version",
				Namespace: "default",
			},
			Spec: v1alpha1.ObservabilityPlatformSpec{
				Components: v1alpha1.Components{
					Prometheus: &v1alpha1.PrometheusSpec{
						Enabled:  true,
						Version:  "v2.48.0",
						Replicas: 3,
					},
				},
			},
		},
		&v1beta1.ObservabilityPlatform{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "observability.io/v1beta1",
				Kind:       "ObservabilityPlatform",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multi-version",
				Namespace: "default",
			},
			Spec: v1beta1.ObservabilityPlatformSpec{
				Components: v1beta1.Components{
					Prometheus: &v1beta1.PrometheusSpec{
						Enabled:  true,
						Version:  "v2.48.0",
						Replicas: 3,
					},
				},
			},
		},
	}

	// Verify each can be marshaled/unmarshaled independently
	for i, platform := range platforms {
		t.Run(fmt.Sprintf("platform-%d", i), func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(platform)
			require.NoError(t, err)

			// Unmarshal back
			var typeMeta metav1.TypeMeta
			err = json.Unmarshal(data, &typeMeta)
			require.NoError(t, err)

			// Verify API version is preserved
			switch p := platform.(type) {
			case *v1alpha1.ObservabilityPlatform:
				assert.Equal(t, "observability.io/v1alpha1", typeMeta.APIVersion)
			case *v1beta1.ObservabilityPlatform:
				assert.Equal(t, "observability.io/v1beta1", typeMeta.APIVersion)
			default:
				t.Fatalf("Unknown platform type: %T", p)
			}
		})
	}
}

// TestDeprecationHandling tests handling of deprecated fields across versions
func TestDeprecationHandling(t *testing.T) {
	// Create v1alpha1 with fields that might be deprecated in v1beta1
	v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deprecated-fields",
			Namespace: "default",
			Annotations: map[string]string{
				"observability.io/deprecated-field": "old-value",
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
					// Imagine these fields are deprecated in v1beta1
					Version:  "v2.48.0",
					Replicas: 3,
				},
			},
		},
	}

	// Convert to v1beta1
	v1beta1Platform := &v1beta1.ObservabilityPlatform{}
	err := v1alpha1Platform.ConvertTo(v1beta1Platform)
	require.NoError(t, err)

	// Verify deprecated fields are handled
	assert.Equal(t, "v2.48.0", v1beta1Platform.Spec.Components.Prometheus.Version)
	assert.Equal(t, int32(3), v1beta1Platform.Spec.Components.Prometheus.Replicas)

	// Check if deprecation warnings are added
	// In a real implementation, this might add warning annotations
	if _, exists := v1beta1Platform.Annotations["observability.io/deprecation-warnings"]; exists {
		t.Log("Deprecation warnings found in annotations")
	}
}
