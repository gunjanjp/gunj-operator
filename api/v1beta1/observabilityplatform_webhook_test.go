/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestObservabilityPlatformDefaulting(t *testing.T) {
	tests := []struct {
		name     string
		input    *ObservabilityPlatform
		validate func(t *testing.T, platform *ObservabilityPlatform)
	}{
		{
			name: "should set default metadata labels and annotations",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				assert.Equal(t, "gunj-operator", platform.Labels["app.kubernetes.io/managed-by"])
				assert.Equal(t, "observability-platform", platform.Labels["app.kubernetes.io/name"])
				assert.Equal(t, "test-platform", platform.Labels["app.kubernetes.io/instance"])
				assert.Equal(t, "v2.0.0", platform.Labels["app.kubernetes.io/version"])
				assert.Equal(t, "gunj-operator", platform.Annotations["observability.io/created-by"])
			},
		},
		{
			name: "should set Prometheus defaults",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{
						Prometheus: &PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				prom := platform.Spec.Components.Prometheus
				assert.Equal(t, "v2.48.0", prom.Version)
				assert.Equal(t, int32(1), prom.Replicas)
				assert.NotNil(t, prom.Resources)
				assert.Equal(t, "250m", prom.Resources.Requests.CPU)
				assert.Equal(t, "512Mi", prom.Resources.Requests.Memory)
				assert.Equal(t, "1", prom.Resources.Limits.CPU)
				assert.Equal(t, "2Gi", prom.Resources.Limits.Memory)
				assert.Equal(t, "10Gi", prom.Storage.Size)
				assert.Equal(t, "15d", prom.Storage.Retention)
				assert.Equal(t, "default", prom.ExternalLabels["cluster"])
			},
		},
		{
			name: "should set Grafana defaults and generate password",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{
						Grafana: &GrafanaSpec{
							Enabled: true,
						},
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				grafana := platform.Spec.Components.Grafana
				assert.Equal(t, "10.2.0", grafana.Version)
				assert.Equal(t, int32(1), grafana.Replicas)
				assert.NotEmpty(t, grafana.AdminPassword, "Admin password should be generated")
				assert.Equal(t, "admin", grafana.AdminUser)
				assert.NotNil(t, grafana.Resources)
				assert.NotNil(t, grafana.Persistence)
				assert.True(t, grafana.Persistence.Enabled)
				assert.Equal(t, "1Gi", grafana.Persistence.Size)
			},
		},
		{
			name: "should set Loki defaults",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{
						Loki: &LokiSpec{
							Enabled: true,
						},
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				loki := platform.Spec.Components.Loki
				assert.Equal(t, "2.9.0", loki.Version)
				assert.Equal(t, int32(1), loki.Replicas)
				assert.NotNil(t, loki.Resources)
				assert.Equal(t, "10Gi", loki.Storage.Size)
				assert.NotNil(t, loki.Retention)
				assert.Equal(t, int32(7), loki.Retention.Days)
				assert.True(t, loki.Retention.DeletesEnabled)
			},
		},
		{
			name: "should set Tempo defaults",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{
						Tempo: &TempoSpec{
							Enabled: true,
						},
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				tempo := platform.Spec.Components.Tempo
				assert.Equal(t, "2.3.0", tempo.Version)
				assert.Equal(t, int32(1), tempo.Replicas)
				assert.NotNil(t, tempo.Resources)
				assert.Equal(t, "5Gi", tempo.Storage.Size)
			},
		},
		{
			name: "should set global settings defaults",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				assert.NotNil(t, platform.Spec.Global)
				assert.Equal(t, "info", platform.Spec.Global.LogLevel)
				assert.Equal(t, "default", platform.Spec.Global.ExternalLabels["organization"])
				assert.Equal(t, "default", platform.Spec.Global.ExternalLabels["environment"])
				assert.NotNil(t, platform.Spec.Global.RetentionPolicies)
				assert.Equal(t, "30d", platform.Spec.Global.RetentionPolicies.Metrics)
				assert.Equal(t, "7d", platform.Spec.Global.RetentionPolicies.Logs)
				assert.Equal(t, "3d", platform.Spec.Global.RetentionPolicies.Traces)
			},
		},
		{
			name: "should infer environment from namespace",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "production",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				assert.Equal(t, "production", platform.Spec.Global.ExternalLabels["environment"])
			},
		},
		{
			name: "should set HA defaults when enabled",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{
						Prometheus: &PrometheusSpec{
							Enabled: true,
						},
						Grafana: &GrafanaSpec{
							Enabled: true,
						},
					},
					HighAvailability: &HighAvailabilitySpec{
						Enabled: true,
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				assert.Equal(t, int32(3), platform.Spec.Components.Prometheus.Replicas)
				assert.Equal(t, int32(2), platform.Spec.Components.Grafana.Replicas)
				assert.NotNil(t, platform.Spec.HighAvailability.AntiAffinity)
				assert.Equal(t, "soft", platform.Spec.HighAvailability.AntiAffinity.Type)
			},
		},
		{
			name: "should set backup defaults when enabled",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{},
					Backup: &BackupSpec{
						Enabled: true,
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				backup := platform.Spec.Backup
				assert.Equal(t, "0 2 * * *", backup.Schedule)
				assert.Equal(t, int32(7), backup.Retention)
				assert.Equal(t, "s3", backup.StorageType)
				assert.Equal(t, "observability-backups/default/test-platform", backup.StorageLocation)
			},
		},
		{
			name: "should set alerting defaults when enabled",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{},
					Alerting: &AlertingSpec{
						Alertmanager: &AlertmanagerSpec{
							Enabled: true,
						},
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				am := platform.Spec.Alerting.Alertmanager
				assert.Equal(t, "v0.26.0", am.Version)
				assert.Equal(t, int32(1), am.Replicas)
				assert.NotNil(t, am.Resources)
				assert.NotNil(t, am.Storage)
				assert.Equal(t, "1Gi", am.Storage.Size)
				assert.NotNil(t, am.Config)
				assert.NotNil(t, am.Config.Route)
				assert.Equal(t, "default-receiver", am.Config.Route.Receiver)
			},
		},
		{
			name: "should not override existing values",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "custom-operator",
						"custom-label":                 "custom-value",
					},
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{
						Prometheus: &PrometheusSpec{
							Enabled:  true,
							Version:  "v2.45.0",
							Replicas: 5,
						},
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				// Should not override existing values
				assert.Equal(t, "custom-operator", platform.Labels["app.kubernetes.io/managed-by"])
				assert.Equal(t, "custom-value", platform.Labels["custom-label"])
				assert.Equal(t, "v2.45.0", platform.Spec.Components.Prometheus.Version)
				assert.Equal(t, int32(5), platform.Spec.Components.Prometheus.Replicas)
				// Should add missing defaults
				assert.Equal(t, "observability-platform", platform.Labels["app.kubernetes.io/name"])
			},
		},
		{
			name: "should normalize resource specifications",
			input: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: ObservabilityPlatformSpec{
					Components: &Components{
						Prometheus: &PrometheusSpec{
							Enabled: true,
							Resources: &ResourceRequirements{
								Requests: &ResourceList{
									CPU: "500m",
								},
								// Missing memory request and all limits
							},
						},
					},
				},
			},
			validate: func(t *testing.T, platform *ObservabilityPlatform) {
				prom := platform.Spec.Components.Prometheus
				// Should have added missing memory request
				assert.Equal(t, "512Mi", prom.Resources.Requests.Memory)
				// Should have added limits
				assert.NotNil(t, prom.Resources.Limits)
				assert.Equal(t, "1", prom.Resources.Limits.CPU)
				assert.Equal(t, "2Gi", prom.Resources.Limits.Memory)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy to avoid modifying the original
			platform := tt.input.DeepCopy()
			
			// Apply defaults
			platform.Default()
			
			// Validate the results
			tt.validate(t, platform)
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	// Test password generation
	password := generateSecurePassword(16)
	
	assert.Equal(t, 16, len(password), "Password should be 16 characters long")
	
	// Generate multiple passwords and ensure they're different
	passwords := make([]string, 10)
	for i := 0; i < 10; i++ {
		passwords[i] = generateSecurePassword(16)
	}
	
	// Check that not all passwords are the same
	unique := make(map[string]bool)
	for _, p := range passwords {
		unique[p] = true
	}
	
	assert.Greater(t, len(unique), 1, "Generated passwords should be unique")
	
	// Test password complexity
	for _, p := range passwords {
		hasLower := false
		hasUpper := false
		hasDigit := false
		hasSpecial := false
		
		for _, char := range p {
			switch {
			case char >= 'a' && char <= 'z':
				hasLower = true
			case char >= 'A' && char <= 'Z':
				hasUpper = true
			case char >= '0' && char <= '9':
				hasDigit = true
			default:
				hasSpecial = true
			}
		}
		
		// Password should have a mix of character types
		complexity := 0
		if hasLower {
			complexity++
		}
		if hasUpper {
			complexity++
		}
		if hasDigit {
			complexity++
		}
		if hasSpecial {
			complexity++
		}
		
		assert.GreaterOrEqual(t, complexity, 2, "Password should have at least 2 different character types")
	}
}

func TestDefaultingWithPartialSpec(t *testing.T) {
	// Test with partially filled spec
	platform := &ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "partial-platform",
			Namespace: "staging",
		},
		Spec: ObservabilityPlatformSpec{
			Components: &Components{
				Prometheus: &PrometheusSpec{
					Enabled: true,
					Storage: &StorageSpec{
						Size: "50Gi",
						// Retention not set - should get default
					},
				},
				Grafana: &GrafanaSpec{
					Enabled:       true,
					AdminUser:     "superadmin",
					// AdminPassword not set - should be generated
				},
			},
			Global: &GlobalSettings{
				ExternalLabels: map[string]string{
					"custom": "label",
					// organization not set - should get default
				},
			},
		},
	}
	
	platform.Default()
	
	// Check that existing values are preserved
	assert.Equal(t, "50Gi", platform.Spec.Components.Prometheus.Storage.Size)
	assert.Equal(t, "superadmin", platform.Spec.Components.Grafana.AdminUser)
	assert.Equal(t, "label", platform.Spec.Global.ExternalLabels["custom"])
	
	// Check that defaults are added
	assert.Equal(t, "15d", platform.Spec.Components.Prometheus.Storage.Retention)
	assert.NotEmpty(t, platform.Spec.Components.Grafana.AdminPassword)
	assert.Equal(t, "default", platform.Spec.Global.ExternalLabels["organization"])
	assert.Equal(t, "staging", platform.Spec.Global.ExternalLabels["environment"])
}

func TestDefaultingIdempotency(t *testing.T) {
	// Create a platform with some settings
	platform := &ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "idempotent-test",
			Namespace: "default",
		},
		Spec: ObservabilityPlatformSpec{
			Components: &Components{
				Prometheus: &PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
	
	// Apply defaults once
	platform.Default()
	
	// Capture the state
	firstPass := platform.DeepCopy()
	
	// Apply defaults again
	platform.Default()
	
	// The result should be identical
	assert.Equal(t, firstPass, platform, "Defaulting should be idempotent")
}
