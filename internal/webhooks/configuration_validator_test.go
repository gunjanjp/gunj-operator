/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package webhooks

import (
	"context"
	"testing"

	"github.com/go-logr/logr/testr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestConfigurationValidator_ValidateConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErrs int
	}{
		{
			name: "valid configuration with all components",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
								Limits: &observabilityv1beta1.ResourceList{
									CPU:    "500m",
									Memory: "1Gi",
								},
							},
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:   true,
							Version:   "10.2.0",
							Replicas:  1,
							AdminUser: "admin",
							DataSources: []observabilityv1beta1.DataSourceSpec{
								{
									Name:   "prometheus",
									Type:   "prometheus",
									URL:    "http://prometheus:9090",
									Access: "proxy",
								},
							},
						},
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Version:  "2.9.0",
							Replicas: 1,
						},
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "2.3.0",
							Replicas: 1,
						},
						OpenTelemetryCollector: &observabilityv1beta1.OpenTelemetryCollectorSpec{
							Enabled: true,
							Version: "0.91.0",
							Mode:    "deployment",
						},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid version formats",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "invalid-version",
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:   true,
							Version:   "10.2",
							AdminUser: "admin",
						},
					},
				},
			},
			wantErrs: 2, // Invalid Prometheus and Grafana versions
		},
		{
			name: "incompatible component versions",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1,
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:   true,
							Version:   "9.0.0", // Incompatible with Prometheus 2.48
							Replicas:  1,
							AdminUser: "admin",
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "missing required configurations",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:  true,
							Version:  "10.2.0",
							Replicas: 1,
							// Missing AdminUser
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "no components enabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: false,
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: false,
						},
					},
				},
			},
			wantErrs: 1, // At least one component must be enabled
		},
		{
			name: "HA mode with insufficient replicas",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1, // Should be at least 2 for HA
						},
					},
					HighAvailability: &observabilityv1beta1.HighAvailabilitySettings{
						Enabled: true,
					},
				},
			},
			wantErrs: 2, // Insufficient replicas and missing storage
		},
		{
			name: "resource requests exceed limits",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "2000m",
									Memory: "4Gi",
								},
								Limits: &observabilityv1beta1.ResourceList{
									CPU:    "1000m",
									Memory: "2Gi",
								},
							},
						},
					},
				},
			},
			wantErrs: 2, // CPU and memory requests exceed limits
		},
		{
			name: "invalid storage configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1,
							Storage: &observabilityv1beta1.StorageSpec{
								Size:      "invalid-size",
								Retention: "invalid-retention",
							},
						},
					},
				},
			},
			wantErrs: 2, // Invalid size and retention
		},
		{
			name: "grafana data source without backend",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:   true,
							Version:   "10.2.0",
							Replicas:  1,
							AdminUser: "admin",
							DataSources: []observabilityv1beta1.DataSourceSpec{
								{
									Name:   "prometheus",
									Type:   "prometheus",
									URL:    "http://prometheus:9090",
									Access: "proxy",
								},
							},
						},
						// Prometheus is not enabled
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "otel collector without backend",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						OpenTelemetryCollector: &observabilityv1beta1.OpenTelemetryCollectorSpec{
							Enabled: true,
							Version: "0.91.0",
							Mode:    "deployment",
						},
						// No backends enabled
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "backup configuration incomplete",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
					},
					Backup: &observabilityv1beta1.BackupSettings{
						Enabled: true,
						// Missing schedule and storage type
					},
				},
			},
			wantErrs: 2,
		},
		{
			name: "s3 backup without required fields",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
					},
					Backup: &observabilityv1beta1.BackupSettings{
						Enabled:     true,
						Schedule:    "0 2 * * *",
						StorageType: "s3",
						// Missing storage config
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "alertmanager without prometheus",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:   true,
							Version:   "10.2.0",
							AdminUser: "admin",
						},
					},
					Alerting: &observabilityv1beta1.AlertingSettings{
						Alertmanager: &observabilityv1beta1.AlertmanagerSpec{
							Enabled: true,
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "loki with s3 storage missing bucket",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Version:  "2.9.0",
							Replicas: 1,
							Storage: &observabilityv1beta1.LokiStorageSpec{
								StorageSpec: observabilityv1beta1.StorageSpec{
									Size: "10Gi",
								},
								S3: &observabilityv1beta1.S3StorageSpec{
									Enabled: true,
									// Missing bucket name and region
								},
							},
						},
					},
				},
			},
			wantErrs: 2,
		},
		{
			name: "ingress without host",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:   true,
							Version:   "10.2.0",
							Replicas:  1,
							AdminUser: "admin",
							Ingress: &observabilityv1beta1.IngressSpec{
								Enabled: true,
								// Missing host
							},
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "remote write with multiple auth methods",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1,
							RemoteWrite: []observabilityv1beta1.RemoteWriteSpec{
								{
									URL: "http://remote-prometheus:9090/api/v1/write",
									BasicAuth: &observabilityv1beta1.BasicAuthSpec{
										Username: "user",
										Password: "pass",
									},
									BearerToken: "token123", // Multiple auth methods
								},
							},
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "invalid otel collector mode",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
						OpenTelemetryCollector: &observabilityv1beta1.OpenTelemetryCollectorSpec{
							Enabled: true,
							Version: "0.91.0",
							Mode:    "invalid-mode",
						},
					},
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigurationValidator(testr.New(t))
			errs := validator.ValidateConfiguration(context.Background(), tt.platform)
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateConfiguration() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, err := range errs {
					t.Logf("Error: %v", err)
				}
			}
		})
	}
}

func TestConfigurationValidator_ValidateVersionCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErrs int
	}{
		{
			name: "compatible versions",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
						},
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled: true,
							Version: "2.9.0",
						},
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled: true,
							Version: "2.3.0",
						},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "incompatible grafana version",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: true,
							Version: "9.0.0", // Not compatible with Prometheus 2.48
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "unknown prometheus version",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.99.0", // Unknown version
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
						},
					},
				},
			},
			wantErrs: 0, // No error for unknown versions (forward compatibility)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigurationValidator(testr.New(t))
			errs := validator.validateVersionCompatibility(tt.platform, field.NewPath("spec", "components"))
			if len(errs) != tt.wantErrs {
				t.Errorf("validateVersionCompatibility() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, err := range errs {
					t.Logf("Error: %v", err)
				}
			}
		})
	}
}

func TestConfigurationValidator_ValidateResourceRequirements(t *testing.T) {
	tests := []struct {
		name      string
		resources *observabilityv1beta1.ResourceRequirements
		wantErrs  int
	}{
		{
			name: "valid resources",
			resources: &observabilityv1beta1.ResourceRequirements{
				Requests: &observabilityv1beta1.ResourceList{
					CPU:    "100m",
					Memory: "256Mi",
				},
				Limits: &observabilityv1beta1.ResourceList{
					CPU:    "500m",
					Memory: "1Gi",
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid resource format",
			resources: &observabilityv1beta1.ResourceRequirements{
				Requests: &observabilityv1beta1.ResourceList{
					CPU:    "invalid",
					Memory: "256",
				},
			},
			wantErrs: 2,
		},
		{
			name: "requests exceed limits",
			resources: &observabilityv1beta1.ResourceRequirements{
				Requests: &observabilityv1beta1.ResourceList{
					CPU:    "2",
					Memory: "4Gi",
				},
				Limits: &observabilityv1beta1.ResourceList{
					CPU:    "1",
					Memory: "2Gi",
				},
			},
			wantErrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigurationValidator(testr.New(t))
			errs := validator.validateResourceRequirements(tt.resources, field.NewPath("resources"))
			if len(errs) != tt.wantErrs {
				t.Errorf("validateResourceRequirements() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, err := range errs {
					t.Logf("Error: %v", err)
				}
			}
		})
	}
}

func TestConfigurationValidator_ValidateStorageSpec(t *testing.T) {
	tests := []struct {
		name     string
		storage  *observabilityv1beta1.StorageSpec
		wantErrs int
	}{
		{
			name: "valid storage",
			storage: &observabilityv1beta1.StorageSpec{
				Size:      "10Gi",
				Retention: "30d",
			},
			wantErrs: 0,
		},
		{
			name: "invalid size",
			storage: &observabilityv1beta1.StorageSpec{
				Size: "invalid",
			},
			wantErrs: 1,
		},
		{
			name: "invalid retention format",
			storage: &observabilityv1beta1.StorageSpec{
				Size:      "10Gi",
				Retention: "30days",
			},
			wantErrs: 1,
		},
		{
			name: "valid retention units",
			storage: &observabilityv1beta1.StorageSpec{
				Size:      "10Gi",
				Retention: "24h",
			},
			wantErrs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigurationValidator(testr.New(t))
			errs := validator.validateStorageSpec(tt.storage, field.NewPath("storage"))
			if len(errs) != tt.wantErrs {
				t.Errorf("validateStorageSpec() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, err := range errs {
					t.Logf("Error: %v", err)
				}
			}
		})
	}
}

func TestConfigurationValidator_CheckDeprecatedConfigurations(t *testing.T) {
	tests := []struct {
		name         string
		platform     *observabilityv1beta1.ObservabilityPlatform
		wantWarnings int
	}{
		{
			name: "no deprecated configs",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
					},
				},
			},
			wantWarnings: 0,
		},
		// Add tests for deprecated configurations when they are defined
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigurationValidator(testr.New(t))
			warnings := validator.checkDeprecatedConfigurations(tt.platform)
			if len(warnings) != tt.wantWarnings {
				t.Errorf("checkDeprecatedConfigurations() returned %d warnings, want %d", len(warnings), tt.wantWarnings)
				for _, warning := range warnings {
					t.Logf("Warning: %s", warning)
				}
			}
		})
	}
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("isValidVersion", func(t *testing.T) {
		tests := []struct {
			version string
			want    bool
		}{
			{"v2.48.0", true},
			{"2.48.0", true},
			{"10.2.0", true},
			{"2.48", false},
			{"v2.48.0.1", false},
			{"invalid", false},
			{"", false},
		}

		for _, tt := range tests {
			t.Run(tt.version, func(t *testing.T) {
				if got := isValidVersion(tt.version); got != tt.want {
					t.Errorf("isValidVersion(%q) = %v, want %v", tt.version, got, tt.want)
				}
			})
		}
	})

	t.Run("normalizeVersion", func(t *testing.T) {
		tests := []struct {
			version string
			want    string
		}{
			{"v2.48.0", "2.48"},
			{"2.48.0", "2.48"},
			{"10.2.3", "10.2"},
			{"2", "2"},
		}

		for _, tt := range tests {
			t.Run(tt.version, func(t *testing.T) {
				if got := normalizeVersion(tt.version); got != tt.want {
					t.Errorf("normalizeVersion(%q) = %v, want %v", tt.version, got, tt.want)
				}
			})
		}
	})

	t.Run("isValidRetention", func(t *testing.T) {
		tests := []struct {
			retention string
			want      bool
		}{
			{"30d", true},
			{"24h", true},
			{"52w", true},
			{"12m", true},
			{"5y", true},
			{"30days", false},
			{"24", false},
			{"d30", false},
			{"", false},
		}

		for _, tt := range tests {
			t.Run(tt.retention, func(t *testing.T) {
				if got := isValidRetention(tt.retention); got != tt.want {
					t.Errorf("isValidRetention(%q) = %v, want %v", tt.retention, got, tt.want)
				}
			})
		}
	})

	t.Run("hasEnabledComponent", func(t *testing.T) {
		tests := []struct {
			name       string
			components *observabilityv1beta1.Components
			want       bool
		}{
			{
				name: "prometheus enabled",
				components: &observabilityv1beta1.Components{
					Prometheus: &observabilityv1beta1.PrometheusSpec{Enabled: true},
				},
				want: true,
			},
			{
				name: "no components enabled",
				components: &observabilityv1beta1.Components{
					Prometheus: &observabilityv1beta1.PrometheusSpec{Enabled: false},
					Grafana:    &observabilityv1beta1.GrafanaSpec{Enabled: false},
				},
				want: false,
			},
			{
				name:       "nil components",
				components: nil,
				want:       false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := hasEnabledComponent(tt.components); got != tt.want {
					t.Errorf("hasEnabledComponent() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("contains", func(t *testing.T) {
		tests := []struct {
			slice []string
			value string
			want  bool
		}{
			{[]string{"a", "b", "c"}, "b", true},
			{[]string{"a", "b", "c"}, "d", false},
			{[]string{}, "a", false},
			{nil, "a", false},
		}

		for _, tt := range tests {
			t.Run(tt.value, func(t *testing.T) {
				if got := contains(tt.slice, tt.value); got != tt.want {
					t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.value, got, tt.want)
				}
			})
		}
	})
}

func TestConfigurationValidator_ValidateDataSourceSpec(t *testing.T) {
	tests := []struct {
		name     string
		ds       observabilityv1beta1.DataSourceSpec
		wantErrs int
	}{
		{
			name: "valid data source",
			ds: observabilityv1beta1.DataSourceSpec{
				Name:   "prometheus",
				Type:   "prometheus",
				URL:    "http://prometheus:9090",
				Access: "proxy",
			},
			wantErrs: 0,
		},
		{
			name: "missing required fields",
			ds: observabilityv1beta1.DataSourceSpec{
				Type: "prometheus",
			},
			wantErrs: 2, // Missing name and URL
		},
		{
			name: "invalid type",
			ds: observabilityv1beta1.DataSourceSpec{
				Name:   "custom",
				Type:   "custom-type",
				URL:    "http://custom:9090",
				Access: "proxy",
			},
			wantErrs: 1,
		},
		{
			name: "invalid access mode",
			ds: observabilityv1beta1.DataSourceSpec{
				Name:   "prometheus",
				Type:   "prometheus",
				URL:    "http://prometheus:9090",
				Access: "invalid",
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConfigurationValidator(testr.New(t))
			errs := validator.validateDataSourceSpec(tt.ds, field.NewPath("dataSource"))
			if len(errs) != tt.wantErrs {
				t.Errorf("validateDataSourceSpec() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, err := range errs {
					t.Logf("Error: %v", err)
				}
			}
		})
	}
}
