/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestObservabilityPlatformConversion(t *testing.T) {
	tests := []struct {
		name        string
		v1alpha1    *ObservabilityPlatform
		expectedErr bool
		validate    func(t *testing.T, v1alpha1 *ObservabilityPlatform, v1beta1 *v1beta1.ObservabilityPlatform)
	}{
		{
			name: "basic conversion with all components",
			v1alpha1: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: ObservabilityPlatformSpec{
					Paused: true,
					Components: Components{
						Prometheus: &PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 3,
							Resources: ResourceRequirements{
								Requests: ResourceList{
									Memory: "4Gi",
									CPU:    "1",
								},
								Limits: ResourceList{
									Memory: "8Gi",
									CPU:    "2",
								},
							},
							Storage: &StorageConfig{
								Size:             "100Gi",
								StorageClassName: "fast-ssd",
							},
							Retention: "30d",
						},
					},
				},
			},
			validate: func(t *testing.T, v1alpha1 *ObservabilityPlatform, v1beta1 *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, v1alpha1.Name, v1beta1.Name)
				assert.Equal(t, v1alpha1.Namespace, v1beta1.Namespace)
				assert.Equal(t, v1alpha1.Spec.Paused, v1beta1.Spec.Paused)
				
				// Validate Prometheus conversion
				require.NotNil(t, v1beta1.Spec.Components.Prometheus)
				assert.Equal(t, v1alpha1.Spec.Components.Prometheus.Enabled, v1beta1.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, v1alpha1.Spec.Components.Prometheus.Version, v1beta1.Spec.Components.Prometheus.Version)
				assert.Equal(t, v1alpha1.Spec.Components.Prometheus.Replicas, v1beta1.Spec.Components.Prometheus.Replicas)
				assert.Equal(t, v1alpha1.Spec.Components.Prometheus.Retention, v1beta1.Spec.Components.Prometheus.Retention)
				
				// Validate resources conversion
				assert.Equal(t, "4Gi", v1beta1.Spec.Components.Prometheus.Resources.Requests[corev1.ResourceMemory].String())
				assert.Equal(t, "1", v1beta1.Spec.Components.Prometheus.Resources.Requests[corev1.ResourceCPU].String())
				assert.Equal(t, "8Gi", v1beta1.Spec.Components.Prometheus.Resources.Limits[corev1.ResourceMemory].String())
				assert.Equal(t, "2", v1beta1.Spec.Components.Prometheus.Resources.Limits[corev1.ResourceCPU].String())
				
				// Validate storage conversion
				assert.Equal(t, "100Gi", v1beta1.Spec.Components.Prometheus.Storage.Size.String())
				assert.Equal(t, "fast-ssd", v1beta1.Spec.Components.Prometheus.Storage.StorageClassName)
			},
		},
		{
			name: "conversion with remote write configuration",
			v1alpha1: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-rw",
					Namespace: "test-namespace",
				},
				Spec: ObservabilityPlatformSpec{
					Components: Components{
						Prometheus: &PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
							RemoteWrite: []RemoteWriteSpec{
								{
									URL:           "https://prometheus.example.com/api/v1/write",
									RemoteTimeout: "30s",
									Headers: map[string]string{
										"Authorization": "Bearer token",
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, v1alpha1 *ObservabilityPlatform, v1beta1 *v1beta1.ObservabilityPlatform) {
				require.NotNil(t, v1beta1.Spec.Components.Prometheus)
				require.Len(t, v1beta1.Spec.Components.Prometheus.RemoteWrite, 1)
				
				rw := v1beta1.Spec.Components.Prometheus.RemoteWrite[0]
				assert.Equal(t, "https://prometheus.example.com/api/v1/write", rw.URL)
				assert.Equal(t, "30s", rw.RemoteTimeout)
				assert.Equal(t, "Bearer token", rw.Headers["Authorization"])
			},
		},
		{
			name: "conversion with grafana configuration",
			v1alpha1: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-grafana",
					Namespace: "test-namespace",
				},
				Spec: ObservabilityPlatformSpec{
					Components: Components{
						Grafana: &GrafanaSpec{
							Enabled:       true,
							Version:       "10.2.0",
							Replicas:      2,
							AdminPassword: "admin123",
							Ingress: &IngressConfig{
								Enabled:   true,
								ClassName: "nginx",
								Host:      "grafana.example.com",
								TLS: &IngressTLS{
									Enabled:    true,
									SecretName: "grafana-tls",
								},
							},
							DataSources: []DataSourceConfig{
								{
									Name:      "Prometheus",
									Type:      "prometheus",
									URL:       "http://prometheus:9090",
									IsDefault: true,
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, v1alpha1 *ObservabilityPlatform, v1beta1 *v1beta1.ObservabilityPlatform) {
				require.NotNil(t, v1beta1.Spec.Components.Grafana)
				assert.Equal(t, v1alpha1.Spec.Components.Grafana.Enabled, v1beta1.Spec.Components.Grafana.Enabled)
				assert.Equal(t, v1alpha1.Spec.Components.Grafana.Version, v1beta1.Spec.Components.Grafana.Version)
				assert.Equal(t, v1alpha1.Spec.Components.Grafana.Replicas, v1beta1.Spec.Components.Grafana.Replicas)
				assert.Equal(t, v1alpha1.Spec.Components.Grafana.AdminPassword, v1beta1.Spec.Components.Grafana.AdminPassword)
				
				// Validate ingress conversion
				require.NotNil(t, v1beta1.Spec.Components.Grafana.Ingress)
				assert.Equal(t, v1alpha1.Spec.Components.Grafana.Ingress.Enabled, v1beta1.Spec.Components.Grafana.Ingress.Enabled)
				assert.Equal(t, v1alpha1.Spec.Components.Grafana.Ingress.ClassName, v1beta1.Spec.Components.Grafana.Ingress.ClassName)
				assert.Equal(t, v1alpha1.Spec.Components.Grafana.Ingress.Host, v1beta1.Spec.Components.Grafana.Ingress.Host)
				
				// Validate datasources conversion
				require.Len(t, v1beta1.Spec.Components.Grafana.DataSources, 1)
				ds := v1beta1.Spec.Components.Grafana.DataSources[0]
				assert.Equal(t, "Prometheus", ds.Name)
				assert.Equal(t, "prometheus", ds.Type)
				assert.Equal(t, "http://prometheus:9090", ds.URL)
				assert.True(t, ds.IsDefault)
			},
		},
		{
			name: "conversion with alerting configuration",
			v1alpha1: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-alerting",
					Namespace: "test-namespace",
				},
				Spec: ObservabilityPlatformSpec{
					Alerting: &AlertingConfig{
						AlertManager: &AlertManagerSpec{
							Enabled:  true,
							Replicas: 3,
							Config:   "global:\n  resolve_timeout: 5m",
						},
						Rules: []AlertingRule{
							{
								Name: "platform-alerts",
								Groups: []AlertRuleGroup{
									{
										Name:     "critical",
										Interval: "1m",
										Rules: []AlertRule{
											{
												Alert: "HighMemoryUsage",
												Expr:  "memory_usage > 90",
												For:   "5m",
												Labels: map[string]string{
													"severity": "critical",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, v1alpha1 *ObservabilityPlatform, v1beta1 *v1beta1.ObservabilityPlatform) {
				require.NotNil(t, v1beta1.Spec.Alerting.Alertmanager)
				assert.Equal(t, v1alpha1.Spec.Alerting.AlertManager.Enabled, v1beta1.Spec.Alerting.Alertmanager.Enabled)
				assert.Equal(t, v1alpha1.Spec.Alerting.AlertManager.Replicas, v1beta1.Spec.Alerting.Alertmanager.Replicas)
				assert.Equal(t, v1alpha1.Spec.Alerting.AlertManager.Config, v1beta1.Spec.Alerting.Alertmanager.Config)
				
				// Validate alerting rules conversion
				require.Len(t, v1beta1.Spec.Alerting.Rules, 1)
				rule := v1beta1.Spec.Alerting.Rules[0]
				assert.Equal(t, "platform-alerts", rule.Name)
				require.Len(t, rule.Groups, 1)
				group := rule.Groups[0]
				assert.Equal(t, "critical", group.Name)
				assert.Equal(t, "1m", group.Interval)
				require.Len(t, group.Rules, 1)
				alert := group.Rules[0]
				assert.Equal(t, "HighMemoryUsage", alert.Alert)
				assert.Equal(t, "memory_usage > 90", alert.Expr)
				assert.Equal(t, "5m", alert.For)
				assert.Equal(t, "critical", alert.Labels["severity"])
			},
		},
		{
			name: "round-trip conversion",
			v1alpha1: &ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-roundtrip",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: ObservabilityPlatformSpec{
					Paused: true,
					Components: Components{
						Prometheus: &PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 2,
							Retention: "7d",
						},
						Loki: &LokiSpec{
							Enabled:   true,
							Version:   "2.9.0",
							Retention: "72h",
							S3: &S3Config{
								Enabled:    true,
								BucketName: "loki-logs",
								Region:     "us-east-1",
							},
						},
					},
					Global: GlobalConfig{
						ExternalLabels: map[string]string{
							"cluster": "production",
						},
						LogLevel: "info",
					},
				},
			},
			validate: func(t *testing.T, original *ObservabilityPlatform, _ *v1beta1.ObservabilityPlatform) {
				// Convert to v1beta1
				hub := &v1beta1.ObservabilityPlatform{}
				err := original.ConvertTo(hub)
				require.NoError(t, err)
				
				// Convert back to v1alpha1
				roundtrip := &ObservabilityPlatform{}
				err = roundtrip.ConvertFrom(hub)
				require.NoError(t, err)
				
				// Validate core fields are preserved
				assert.Equal(t, original.Name, roundtrip.Name)
				assert.Equal(t, original.Namespace, roundtrip.Namespace)
				assert.Equal(t, original.Labels, roundtrip.Labels)
				assert.Equal(t, original.Spec.Paused, roundtrip.Spec.Paused)
				
				// Validate components
				assert.Equal(t, original.Spec.Components.Prometheus.Enabled, roundtrip.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, original.Spec.Components.Prometheus.Version, roundtrip.Spec.Components.Prometheus.Version)
				assert.Equal(t, original.Spec.Components.Prometheus.Replicas, roundtrip.Spec.Components.Prometheus.Replicas)
				assert.Equal(t, original.Spec.Components.Prometheus.Retention, roundtrip.Spec.Components.Prometheus.Retention)
				
				assert.Equal(t, original.Spec.Components.Loki.Enabled, roundtrip.Spec.Components.Loki.Enabled)
				assert.Equal(t, original.Spec.Components.Loki.Version, roundtrip.Spec.Components.Loki.Version)
				assert.Equal(t, original.Spec.Components.Loki.Retention, roundtrip.Spec.Components.Loki.Retention)
				assert.Equal(t, original.Spec.Components.Loki.S3.Enabled, roundtrip.Spec.Components.Loki.S3.Enabled)
				assert.Equal(t, original.Spec.Components.Loki.S3.BucketName, roundtrip.Spec.Components.Loki.S3.BucketName)
				assert.Equal(t, original.Spec.Components.Loki.S3.Region, roundtrip.Spec.Components.Loki.S3.Region)
				
				// Validate global config
				assert.Equal(t, original.Spec.Global.ExternalLabels, roundtrip.Spec.Global.ExternalLabels)
				assert.Equal(t, original.Spec.Global.LogLevel, roundtrip.Spec.Global.LogLevel)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to v1beta1
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tt.v1alpha1.ConvertTo(v1beta1Platform)
			
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			tt.validate(t, tt.v1alpha1, v1beta1Platform)
		})
	}
}

func TestV1Beta1ToV1Alpha1Conversion(t *testing.T) {
	tests := []struct {
		name        string
		v1beta1     *v1beta1.ObservabilityPlatform
		expectedErr bool
		validate    func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform, v1alpha1 *ObservabilityPlatform)
	}{
		{
			name: "conversion with fields that will be lost",
			v1beta1: &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-lossy",
					Namespace: "test-namespace",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled:                 true,
							Version:                 "v2.48.0",
							ExternalLabels:          map[string]string{"env": "prod"},
							AdditionalScrapeConfigs: "- job_name: custom",
						},
						Grafana: &v1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
							Plugins: []string{"piechart-panel", "worldmap-panel"},
							SMTP: &v1beta1.SMTPConfig{
								Host: "smtp.example.com",
								Port: 587,
								From: "grafana@example.com",
							},
						},
						Loki: &v1beta1.LokiSpec{
							Enabled:          true,
							Version:          "2.9.0",
							CompactorEnabled: true,
						},
						Tempo: &v1beta1.TempoSpec{
							Enabled:       true,
							Version:       "2.3.0",
							SearchEnabled: true,
						},
					},
					Security: &v1beta1.SecurityConfig{
						TLS: v1beta1.TLSConfig{
							Enabled: true,
						},
						PodSecurityPolicy: true,
						NetworkPolicy:     true,
					},
					Global: v1beta1.GlobalConfig{
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{
										{
											MatchExpressions: []corev1.NodeSelectorRequirement{
												{
													Key:      "node-type",
													Operator: corev1.NodeSelectorOpIn,
													Values:   []string{"observability"},
												},
											},
										},
									},
								},
							},
						},
						ImagePullSecrets: []corev1.LocalObjectReference{
							{Name: "registry-secret"},
						},
					},
				},
			},
			validate: func(t *testing.T, v1beta1Platform *v1beta1.ObservabilityPlatform, v1alpha1Platform *ObservabilityPlatform) {
				// Validate that basic fields are preserved
				assert.Equal(t, v1beta1Platform.Name, v1alpha1Platform.Name)
				assert.Equal(t, v1beta1Platform.Namespace, v1alpha1Platform.Namespace)
				
				// Validate Prometheus - external labels and additional scrape configs are lost
				require.NotNil(t, v1alpha1Platform.Spec.Components.Prometheus)
				assert.Equal(t, v1beta1Platform.Spec.Components.Prometheus.Enabled, v1alpha1Platform.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, v1beta1Platform.Spec.Components.Prometheus.Version, v1alpha1Platform.Spec.Components.Prometheus.Version)
				
				// Validate Grafana - plugins and SMTP are lost
				require.NotNil(t, v1alpha1Platform.Spec.Components.Grafana)
				assert.Equal(t, v1beta1Platform.Spec.Components.Grafana.Enabled, v1alpha1Platform.Spec.Components.Grafana.Enabled)
				assert.Equal(t, v1beta1Platform.Spec.Components.Grafana.Version, v1alpha1Platform.Spec.Components.Grafana.Version)
				
				// Validate Loki - compactor setting is lost
				require.NotNil(t, v1alpha1Platform.Spec.Components.Loki)
				assert.Equal(t, v1beta1Platform.Spec.Components.Loki.Enabled, v1alpha1Platform.Spec.Components.Loki.Enabled)
				assert.Equal(t, v1beta1Platform.Spec.Components.Loki.Version, v1alpha1Platform.Spec.Components.Loki.Version)
				
				// Validate Tempo - search setting is lost
				require.NotNil(t, v1alpha1Platform.Spec.Components.Tempo)
				assert.Equal(t, v1beta1Platform.Spec.Components.Tempo.Enabled, v1alpha1Platform.Spec.Components.Tempo.Enabled)
				assert.Equal(t, v1beta1Platform.Spec.Components.Tempo.Version, v1alpha1Platform.Spec.Components.Tempo.Version)
				
				// Security configuration is completely lost - v1alpha1 doesn't have this field
				// Affinity and ImagePullSecrets are also lost
			},
		},
		{
			name: "conversion with resource requirements",
			v1beta1: &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-resources",
					Namespace: "test-namespace",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
							},
							Storage: v1beta1.StorageSpec{
								Size:             resource.MustParse("50Gi"),
								StorageClassName: "standard",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, v1beta1Platform *v1beta1.ObservabilityPlatform, v1alpha1Platform *ObservabilityPlatform) {
				require.NotNil(t, v1alpha1Platform.Spec.Components.Prometheus)
				
				// Validate resource conversion
				assert.Equal(t, "500m", v1alpha1Platform.Spec.Components.Prometheus.Resources.Requests.CPU)
				assert.Equal(t, "2Gi", v1alpha1Platform.Spec.Components.Prometheus.Resources.Requests.Memory)
				assert.Equal(t, "2", v1alpha1Platform.Spec.Components.Prometheus.Resources.Limits.CPU)
				assert.Equal(t, "4Gi", v1alpha1Platform.Spec.Components.Prometheus.Resources.Limits.Memory)
				
				// Validate storage conversion
				require.NotNil(t, v1alpha1Platform.Spec.Components.Prometheus.Storage)
				assert.Equal(t, "50Gi", v1alpha1Platform.Spec.Components.Prometheus.Storage.Size)
				assert.Equal(t, "standard", v1alpha1Platform.Spec.Components.Prometheus.Storage.StorageClassName)
			},
		},
		{
			name: "status conversion",
			v1beta1: &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-status",
					Namespace: "test-namespace",
				},
				Status: v1beta1.ObservabilityPlatformStatus{
					Phase:              "Ready",
					ObservedGeneration: 5,
					Message:            "All components are running",
					Version:            "v2.0.0",
					ComponentStatus: map[string]v1beta1.ComponentStatus{
						"prometheus": {
							Phase:    "Ready",
							Version:  "v2.48.0",
							Replicas: 3,
							Ready:    3,
							Message:  "Prometheus is running",
						},
					},
					Health: v1beta1.HealthStatus{
						Status:       "Healthy",
						HealthyCount: 4,
						TotalCount:   4,
					},
				},
			},
			validate: func(t *testing.T, v1beta1Platform *v1beta1.ObservabilityPlatform, v1alpha1Platform *ObservabilityPlatform) {
				// Validate status conversion
				assert.Equal(t, v1beta1Platform.Status.Phase, v1alpha1Platform.Status.Phase)
				assert.Equal(t, v1beta1Platform.Status.ObservedGeneration, v1alpha1Platform.Status.ObservedGeneration)
				assert.Equal(t, v1beta1Platform.Status.Message, v1alpha1Platform.Status.Message)
				
				// Validate component status conversion
				require.Contains(t, v1alpha1Platform.Status.ComponentStatus, "prometheus")
				promStatus := v1alpha1Platform.Status.ComponentStatus["prometheus"]
				assert.Equal(t, "Ready", promStatus.Phase)
				assert.Equal(t, "v2.48.0", promStatus.Version)
				assert.Equal(t, int32(3), promStatus.ReadyReplicas)
				
				// Note: Version and Health fields are lost in conversion to v1alpha1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to v1alpha1
			v1alpha1Platform := &ObservabilityPlatform{}
			err := v1alpha1Platform.ConvertFrom(tt.v1beta1)
			
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			tt.validate(t, tt.v1beta1, v1alpha1Platform)
		})
	}
}
