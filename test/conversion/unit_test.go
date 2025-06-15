/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

// Package conversion contains comprehensive conversion tests for the Gunj Operator
package conversion

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	"github.com/gunjanjp/gunj-operator/api/v1alpha1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// TestConversionFieldMapping tests that all fields are properly mapped during conversion
func TestConversionFieldMapping(t *testing.T) {
	tests := []struct {
		name               string
		v1alpha1           *v1alpha1.ObservabilityPlatform
		validateConversion func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform)
	}{
		{
			name: "metadata fields mapping",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "monitoring",
					Labels: map[string]string{
						"app":         "observability",
						"environment": "production",
						"team":        "platform",
					},
					Annotations: map[string]string{
						"description":           "Production observability platform",
						"kubectl.kubernetes.io/last-applied-configuration": "{}",
					},
					UID:             "12345-67890",
					ResourceVersion: "12345",
					Generation:      3,
					Finalizers:      []string{"observability.io/finalizer"},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "parent-config",
							UID:        "parent-uid",
						},
					},
				},
			},
			validateConversion: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				assert.Equal(t, "test-platform", v1beta1.Name)
				assert.Equal(t, "monitoring", v1beta1.Namespace)
				assert.Equal(t, "12345-67890", string(v1beta1.UID))
				assert.Equal(t, "12345", v1beta1.ResourceVersion)
				assert.Equal(t, int64(3), v1beta1.Generation)
				assert.ElementsMatch(t, []string{"observability.io/finalizer"}, v1beta1.Finalizers)
				assert.Len(t, v1beta1.OwnerReferences, 1)
				assert.Equal(t, "parent-config", v1beta1.OwnerReferences[0].Name)
				
				// Verify all labels are preserved
				assert.Len(t, v1beta1.Labels, 3)
				assert.Equal(t, "observability", v1beta1.Labels["app"])
				assert.Equal(t, "production", v1beta1.Labels["environment"])
				assert.Equal(t, "platform", v1beta1.Labels["team"])
				
				// Verify annotations and that conversion annotation is added
				assert.Contains(t, v1beta1.Annotations, "observability.io/converted-from")
				assert.Equal(t, "v1alpha1", v1beta1.Annotations["observability.io/converted-from"])
				assert.Equal(t, "Production observability platform", v1beta1.Annotations["description"])
			},
		},
		{
			name: "resource requirements detailed conversion",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "resource-test",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU:    "100m",
									Memory: "128Mi",
								},
								Limits: v1alpha1.ResourceList{
									CPU:    "2000m",
									Memory: "8Gi",
								},
							},
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU:    "50m",
									Memory: "64Mi",
								},
								Limits: v1alpha1.ResourceList{
									CPU:    "500m",
									Memory: "1Gi",
								},
							},
						},
						Loki: &v1alpha1.LokiSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU:    "1",
									Memory: "2Gi",
								},
								Limits: v1alpha1.ResourceList{
									CPU:    "4",
									Memory: "16Gi",
								},
							},
						},
						Tempo: &v1alpha1.TempoSpec{
							Enabled: true,
							Resources: v1alpha1.ResourceRequirements{
								Requests: v1alpha1.ResourceList{
									CPU:    "250m",
									Memory: "512Mi",
								},
								Limits: v1alpha1.ResourceList{
									CPU:    "1000m",
									Memory: "2Gi",
								},
							},
						},
					},
				},
			},
			validateConversion: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				// Validate Prometheus resources
				promRes := v1beta1.Spec.Components.Prometheus.Resources
				assert.Equal(t, "100m", promRes.Requests.Cpu().String())
				assert.Equal(t, "128Mi", promRes.Requests.Memory().String())
				assert.Equal(t, "2", promRes.Limits.Cpu().String())
				assert.Equal(t, "8Gi", promRes.Limits.Memory().String())
				
				// Validate Grafana resources
				grafanaRes := v1beta1.Spec.Components.Grafana.Resources
				assert.Equal(t, "50m", grafanaRes.Requests.Cpu().String())
				assert.Equal(t, "64Mi", grafanaRes.Requests.Memory().String())
				assert.Equal(t, "500m", grafanaRes.Limits.Cpu().String())
				assert.Equal(t, "1Gi", grafanaRes.Limits.Memory().String())
				
				// Validate Loki resources
				lokiRes := v1beta1.Spec.Components.Loki.Resources
				assert.Equal(t, "1", lokiRes.Requests.Cpu().String())
				assert.Equal(t, "2Gi", lokiRes.Requests.Memory().String())
				assert.Equal(t, "4", lokiRes.Limits.Cpu().String())
				assert.Equal(t, "16Gi", lokiRes.Limits.Memory().String())
				
				// Validate Tempo resources
				tempoRes := v1beta1.Spec.Components.Tempo.Resources
				assert.Equal(t, "250m", tempoRes.Requests.Cpu().String())
				assert.Equal(t, "512Mi", tempoRes.Requests.Memory().String())
				assert.Equal(t, "1", tempoRes.Limits.Cpu().String())
				assert.Equal(t, "2Gi", tempoRes.Limits.Memory().String())
			},
		},
		{
			name: "storage configuration conversion",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "storage-test",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Storage: &v1alpha1.StorageConfig{
								Size:             "100Gi",
								StorageClassName: "fast-ssd",
								VolumeMode:       "Filesystem",
								AccessModes:      []string{"ReadWriteOnce"},
							},
						},
						Loki: &v1alpha1.LokiSpec{
							Enabled: true,
							Storage: &v1alpha1.StorageConfig{
								Size:             "500Gi",
								StorageClassName: "standard",
								VolumeMode:       "Block",
								AccessModes:      []string{"ReadWriteMany"},
							},
						},
						Tempo: &v1alpha1.TempoSpec{
							Enabled: true,
							Storage: &v1alpha1.StorageConfig{
								Size:             "50Gi",
								StorageClassName: "local-path",
							},
						},
					},
				},
			},
			validateConversion: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				// Validate Prometheus storage
				promStorage := v1beta1.Spec.Components.Prometheus.Storage
				assert.Equal(t, "100Gi", promStorage.Size.String())
				assert.Equal(t, "fast-ssd", promStorage.StorageClassName)
				assert.Equal(t, corev1.PersistentVolumeFilesystem, *promStorage.VolumeMode)
				assert.ElementsMatch(t, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, promStorage.AccessModes)
				
				// Validate Loki storage
				lokiStorage := v1beta1.Spec.Components.Loki.Storage
				assert.Equal(t, "500Gi", lokiStorage.Size.String())
				assert.Equal(t, "standard", lokiStorage.StorageClassName)
				assert.Equal(t, corev1.PersistentVolumeBlock, *lokiStorage.VolumeMode)
				assert.ElementsMatch(t, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}, lokiStorage.AccessModes)
				
				// Validate Tempo storage
				tempoStorage := v1beta1.Spec.Components.Tempo.Storage
				assert.Equal(t, "50Gi", tempoStorage.Size.String())
				assert.Equal(t, "local-path", tempoStorage.StorageClassName)
			},
		},
		{
			name: "ingress configuration conversion",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress-test",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Ingress: &v1alpha1.IngressConfig{
								Enabled:   true,
								ClassName: "nginx",
								Host:      "grafana.example.com",
								Path:      "/grafana",
								PathType:  "Prefix",
								TLS: &v1alpha1.IngressTLS{
									Enabled:    true,
									SecretName: "grafana-tls",
									Hosts:      []string{"grafana.example.com", "grafana.local"},
								},
								Annotations: map[string]string{
									"nginx.ingress.kubernetes.io/ssl-redirect":      "true",
									"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
									"cert-manager.io/cluster-issuer":                 "letsencrypt-prod",
								},
							},
						},
					},
				},
			},
			validateConversion: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				ingress := v1beta1.Spec.Components.Grafana.Ingress
				assert.True(t, ingress.Enabled)
				assert.Equal(t, "nginx", ingress.ClassName)
				assert.Equal(t, "grafana.example.com", ingress.Host)
				assert.Equal(t, "/grafana", ingress.Path)
				assert.Equal(t, "Prefix", ingress.PathType)
				
				// Validate TLS
				assert.NotNil(t, ingress.TLS)
				assert.True(t, ingress.TLS.Enabled)
				assert.Equal(t, "grafana-tls", ingress.TLS.SecretName)
				assert.ElementsMatch(t, []string{"grafana.example.com", "grafana.local"}, ingress.TLS.Hosts)
				
				// Validate annotations
				assert.Len(t, ingress.Annotations, 3)
				assert.Equal(t, "true", ingress.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"])
				assert.Equal(t, "true", ingress.Annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"])
				assert.Equal(t, "letsencrypt-prod", ingress.Annotations["cert-manager.io/cluster-issuer"])
			},
		},
		{
			name: "global configuration conversion",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "global-test",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Global: v1alpha1.GlobalConfig{
						ExternalLabels: map[string]string{
							"cluster":     "production",
							"region":      "us-east-1",
							"environment": "prod",
							"team":        "platform",
						},
						LogLevel: "debug",
						NodeSelector: map[string]string{
							"node-role.kubernetes.io/observability": "true",
							"zone":                                   "us-east-1a",
						},
						Tolerations: []v1alpha1.Toleration{
							{
								Key:      "observability",
								Operator: "Equal",
								Value:    "true",
								Effect:   "NoSchedule",
							},
							{
								Key:      "dedicated",
								Operator: "Exists",
								Effect:   "NoExecute",
							},
						},
					},
				},
			},
			validateConversion: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				global := v1beta1.Spec.Global
				
				// Validate external labels
				assert.Len(t, global.ExternalLabels, 4)
				assert.Equal(t, "production", global.ExternalLabels["cluster"])
				assert.Equal(t, "us-east-1", global.ExternalLabels["region"])
				assert.Equal(t, "prod", global.ExternalLabels["environment"])
				assert.Equal(t, "platform", global.ExternalLabels["team"])
				
				// Validate log level
				assert.Equal(t, "debug", global.LogLevel)
				
				// Validate node selector
				assert.Len(t, global.NodeSelector, 2)
				assert.Equal(t, "true", global.NodeSelector["node-role.kubernetes.io/observability"])
				assert.Equal(t, "us-east-1a", global.NodeSelector["zone"])
				
				// Validate tolerations
				assert.Len(t, global.Tolerations, 2)
				assert.Equal(t, "observability", global.Tolerations[0].Key)
				assert.Equal(t, corev1.TolerationOpEqual, global.Tolerations[0].Operator)
				assert.Equal(t, "true", global.Tolerations[0].Value)
				assert.Equal(t, corev1.TaintEffectNoSchedule, global.Tolerations[0].Effect)
				
				assert.Equal(t, "dedicated", global.Tolerations[1].Key)
				assert.Equal(t, corev1.TolerationOpExists, global.Tolerations[1].Operator)
				assert.Equal(t, corev1.TaintEffectNoExecute, global.Tolerations[1].Effect)
			},
		},
		{
			name: "status conversion with all fields",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "status-test",
					Namespace: "default",
				},
				Status: v1alpha1.ObservabilityPlatformStatus{
					Phase:              "Ready",
					ObservedGeneration: 10,
					Message:            "All components are healthy and running",
					LastTransitionTime: &metav1.Time{},
					ComponentStatus: map[string]v1alpha1.ComponentStatus{
						"prometheus": {
							Phase:          "Ready",
							Version:        "v2.48.0",
							ReadyReplicas:  3,
							TotalReplicas:  3,
							LastUpdateTime: &metav1.Time{},
							Message:        "Prometheus is healthy",
						},
						"grafana": {
							Phase:          "Installing",
							Version:        "10.2.0",
							ReadyReplicas:  1,
							TotalReplicas:  2,
							LastUpdateTime: &metav1.Time{},
							Message:        "Waiting for replicas to be ready",
						},
						"loki": {
							Phase:          "Failed",
							Version:        "2.9.0",
							ReadyReplicas:  0,
							TotalReplicas:  1,
							LastUpdateTime: &metav1.Time{},
							Message:        "Failed to create PVC",
						},
					},
					Conditions: []metav1.Condition{
						{
							Type:               "Progressing",
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.Now(),
							Reason:             "NewReplicaSetAvailable",
							Message:            "ReplicaSet has successfully progressed",
						},
						{
							Type:               "Available",
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.Now(),
							Reason:             "MinimumReplicasAvailable",
							Message:            "Deployment has minimum availability",
						},
					},
				},
			},
			validateConversion: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				status := v1beta1.Status
				
				// Validate basic status fields
				assert.Equal(t, "Ready", status.Phase)
				assert.Equal(t, int64(10), status.ObservedGeneration)
				assert.Equal(t, "All components are healthy and running", status.Message)
				assert.NotNil(t, status.LastTransitionTime)
				
				// Validate component status
				assert.Len(t, status.ComponentStatus, 3)
				
				promStatus := status.ComponentStatus["prometheus"]
				assert.Equal(t, "Ready", promStatus.Phase)
				assert.Equal(t, "v2.48.0", promStatus.Version)
				assert.Equal(t, int32(3), promStatus.Ready)
				assert.Equal(t, int32(3), promStatus.Replicas)
				assert.NotNil(t, promStatus.LastUpdateTime)
				assert.Equal(t, "Prometheus is healthy", promStatus.Message)
				
				grafanaStatus := status.ComponentStatus["grafana"]
				assert.Equal(t, "Installing", grafanaStatus.Phase)
				assert.Equal(t, "10.2.0", grafanaStatus.Version)
				assert.Equal(t, int32(1), grafanaStatus.Ready)
				assert.Equal(t, int32(2), grafanaStatus.Replicas)
				assert.Equal(t, "Waiting for replicas to be ready", grafanaStatus.Message)
				
				lokiStatus := status.ComponentStatus["loki"]
				assert.Equal(t, "Failed", lokiStatus.Phase)
				assert.Equal(t, "2.9.0", lokiStatus.Version)
				assert.Equal(t, int32(0), lokiStatus.Ready)
				assert.Equal(t, int32(1), lokiStatus.Replicas)
				assert.Equal(t, "Failed to create PVC", lokiStatus.Message)
				
				// Validate conditions
				assert.Len(t, status.Conditions, 2)
				assert.Equal(t, "Progressing", status.Conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, status.Conditions[0].Status)
				assert.Equal(t, "Available", status.Conditions[1].Type)
				assert.Equal(t, metav1.ConditionTrue, status.Conditions[1].Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to v1beta1
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tt.v1alpha1.ConvertTo(v1beta1Platform)
			require.NoError(t, err)

			// Validate conversion
			tt.validateConversion(t, v1beta1Platform)
		})
	}
}

// TestRoundTripConversion tests converting from v1alpha1 to v1beta1 and back
func TestRoundTripConversion(t *testing.T) {
	tests := []struct {
		name                 string
		original             *v1alpha1.ObservabilityPlatform
		expectDataPreserved  bool
		validateRoundTrip    func(t *testing.T, original, roundtrip *v1alpha1.ObservabilityPlatform)
	}{
		{
			name: "complete round-trip with all preserved fields",
			original: createCompleteV1Alpha1Platform(),
			expectDataPreserved: true,
			validateRoundTrip: func(t *testing.T, original, roundtrip *v1alpha1.ObservabilityPlatform) {
				// Use go-cmp for deep comparison, excluding fields that might change
				opts := []cmp.Option{
					// Ignore the conversion annotation that gets added
					cmp.FilterPath(func(p cmp.Path) bool {
						return p.String() == "ObjectMeta.Annotations[\"observability.io/converted-from\"]"
					}, cmp.Ignore()),
				}
				
				if diff := cmp.Diff(original, roundtrip, opts...); diff != "" {
					t.Errorf("Round-trip conversion produced differences (-original +roundtrip):\n%s", diff)
				}
			},
		},
		{
			name: "round-trip with minimal fields",
			original: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "minimal",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			expectDataPreserved: true,
			validateRoundTrip: func(t *testing.T, original, roundtrip *v1alpha1.ObservabilityPlatform) {
				assert.Equal(t, original.Name, roundtrip.Name)
				assert.Equal(t, original.Namespace, roundtrip.Namespace)
				assert.NotNil(t, roundtrip.Spec.Components.Prometheus)
				assert.True(t, roundtrip.Spec.Components.Prometheus.Enabled)
			},
		},
		{
			name: "round-trip with empty components",
			original: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-components",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{},
					Global: v1alpha1.GlobalConfig{
						LogLevel: "info",
					},
				},
			},
			expectDataPreserved: true,
			validateRoundTrip: func(t *testing.T, original, roundtrip *v1alpha1.ObservabilityPlatform) {
				assert.Equal(t, original.Name, roundtrip.Name)
				assert.Equal(t, "info", roundtrip.Spec.Global.LogLevel)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Convert v1alpha1 to v1beta1
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tt.original.ConvertTo(v1beta1Platform)
			require.NoError(t, err)

			// Step 2: Convert v1beta1 back to v1alpha1
			roundtrip := &v1alpha1.ObservabilityPlatform{}
			err = roundtrip.ConvertFrom(v1beta1Platform)
			require.NoError(t, err)

			// Step 3: Validate round-trip
			tt.validateRoundTrip(t, tt.original, roundtrip)
		})
	}
}

// TestV1Beta1FieldLossConversion tests conversion from v1beta1 to v1alpha1 where fields are lost
func TestV1Beta1FieldLossConversion(t *testing.T) {
	tests := []struct {
		name              string
		v1beta1           *v1beta1.ObservabilityPlatform
		expectedLostFields []string
		validate          func(t *testing.T, v1alpha1 *v1alpha1.ObservabilityPlatform)
	}{
		{
			name: "v1beta1 fields not in v1alpha1",
			v1beta1: &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "field-loss-test",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled:                 true,
							Version:                 "v2.48.0",
							ExternalLabels:          map[string]string{"env": "prod"},
							AdditionalScrapeConfigs: "- job_name: custom",
							WALCompression:          pointer.Bool(true),
							EnableFeatures:          []string{"exemplar-storage"},
						},
						Grafana: &v1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.2.0",
							Plugins: []string{"piechart-panel", "worldmap-panel"},
							SMTP: &v1beta1.SMTPConfig{
								Host:     "smtp.example.com",
								Port:     587,
								From:     "grafana@example.com",
								User:     "smtp-user",
								Password: "smtp-pass",
							},
							OAuth: &v1beta1.OAuthConfig{
								Enabled:      true,
								ClientID:     "grafana-client",
								ClientSecret: "grafana-secret",
								Scopes:       []string{"openid", "profile", "email"},
							},
						},
						Loki: &v1beta1.LokiSpec{
							Enabled:          true,
							Version:          "2.9.0",
							CompactorEnabled: true,
							QueryTimeout:     "5m",
							IndexGateway: &v1beta1.IndexGatewayConfig{
								Enabled:  true,
								Replicas: 2,
							},
						},
						Tempo: &v1beta1.TempoSpec{
							Enabled:       true,
							Version:       "2.3.0",
							SearchEnabled: true,
							MetricsGenerator: &v1beta1.MetricsGeneratorConfig{
								Enabled:           true,
								RemoteWriteURL:    "http://prometheus:9090/api/v1/write",
								ServiceGraphsEnabled: true,
							},
						},
					},
					Security: &v1beta1.SecurityConfig{
						TLS: v1beta1.TLSConfig{
							Enabled:    true,
							MinVersion: "1.2",
							CipherSuites: []string{
								"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
								"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
							},
						},
						PodSecurityPolicy: true,
						NetworkPolicy:     true,
						ServiceMesh: &v1beta1.ServiceMeshConfig{
							Enabled: true,
							Istio: &v1beta1.IstioConfig{
								MutualTLS: true,
								Gateway:   "istio-system/main-gateway",
							},
						},
					},
					Global: v1beta1.GlobalConfig{
						ImagePullSecrets: []corev1.LocalObjectReference{
							{Name: "registry-secret"},
						},
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
						PriorityClassName: "high-priority",
					},
				},
			},
			expectedLostFields: []string{
				"Prometheus.ExternalLabels",
				"Prometheus.AdditionalScrapeConfigs",
				"Prometheus.WALCompression",
				"Prometheus.EnableFeatures",
				"Grafana.Plugins",
				"Grafana.SMTP",
				"Grafana.OAuth",
				"Loki.CompactorEnabled",
				"Loki.QueryTimeout",
				"Loki.IndexGateway",
				"Tempo.SearchEnabled",
				"Tempo.MetricsGenerator",
				"Security",
				"Global.ImagePullSecrets",
				"Global.Affinity",
				"Global.PriorityClassName",
			},
			validate: func(t *testing.T, v1alpha1 *v1alpha1.ObservabilityPlatform) {
				// Verify basic fields are preserved
				assert.Equal(t, "field-loss-test", v1alpha1.Name)
				assert.Equal(t, "default", v1alpha1.Namespace)
				
				// Verify components basic fields are preserved
				assert.True(t, v1alpha1.Spec.Components.Prometheus.Enabled)
				assert.Equal(t, "v2.48.0", v1alpha1.Spec.Components.Prometheus.Version)
				
				assert.True(t, v1alpha1.Spec.Components.Grafana.Enabled)
				assert.Equal(t, "10.2.0", v1alpha1.Spec.Components.Grafana.Version)
				
				assert.True(t, v1alpha1.Spec.Components.Loki.Enabled)
				assert.Equal(t, "2.9.0", v1alpha1.Spec.Components.Loki.Version)
				
				assert.True(t, v1alpha1.Spec.Components.Tempo.Enabled)
				assert.Equal(t, "2.3.0", v1alpha1.Spec.Components.Tempo.Version)
				
				// Verify annotation about lost fields
				assert.Contains(t, v1alpha1.Annotations, "observability.io/conversion-lost-fields")
			},
		},
		{
			name: "status fields loss",
			v1beta1: &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "status-loss-test",
					Namespace: "default",
				},
				Status: v1beta1.ObservabilityPlatformStatus{
					Phase:              "Ready",
					Version:            "v2.0.0",
					ObservedGeneration: 5,
					Health: v1beta1.HealthStatus{
						Status:       "Healthy",
						HealthyCount: 4,
						TotalCount:   4,
						LastCheckTime: &metav1.Time{},
					},
					ComponentStatus: map[string]v1beta1.ComponentStatus{
						"prometheus": {
							Phase:          "Ready",
							Version:        "v2.48.0",
							Replicas:       3,
							Ready:          3,
							UpdatedReplicas: 3,
							AvailableReplicas: 3,
							Endpoints: []string{
								"http://prometheus-0:9090",
								"http://prometheus-1:9090",
								"http://prometheus-2:9090",
							},
						},
					},
					Backup: &v1beta1.BackupStatus{
						LastBackupTime: &metav1.Time{},
						LastBackupSize: "10Gi",
						BackupLocation: "s3://backups/2025-01-01",
					},
				},
			},
			expectedLostFields: []string{
				"Status.Version",
				"Status.Health",
				"Status.ComponentStatus[prometheus].UpdatedReplicas",
				"Status.ComponentStatus[prometheus].AvailableReplicas",
				"Status.ComponentStatus[prometheus].Endpoints",
				"Status.Backup",
			},
			validate: func(t *testing.T, v1alpha1 *v1alpha1.ObservabilityPlatform) {
				// Basic status fields should be preserved
				assert.Equal(t, "Ready", v1alpha1.Status.Phase)
				assert.Equal(t, int64(5), v1alpha1.Status.ObservedGeneration)
				
				// Component status basic fields preserved
				promStatus := v1alpha1.Status.ComponentStatus["prometheus"]
				assert.Equal(t, "Ready", promStatus.Phase)
				assert.Equal(t, "v2.48.0", promStatus.Version)
				assert.Equal(t, int32(3), promStatus.ReadyReplicas)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to v1alpha1
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
			err := v1alpha1Platform.ConvertFrom(tt.v1beta1)
			require.NoError(t, err)

			// Validate
			tt.validate(t, v1alpha1Platform)

			// If we expect lost fields, verify the annotation
			if len(tt.expectedLostFields) > 0 {
				lostFieldsAnnotation := v1alpha1Platform.Annotations["observability.io/conversion-lost-fields"]
				assert.NotEmpty(t, lostFieldsAnnotation, "Expected lost fields annotation")
			}
		})
	}
}

// TestConversionWithNilAndEmptyFields tests conversion handles nil and empty fields correctly
func TestConversionWithNilAndEmptyFields(t *testing.T) {
	tests := []struct {
		name     string
		v1alpha1 *v1alpha1.ObservabilityPlatform
		validate func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform)
	}{
		{
			name: "nil component specs",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nil-test",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: nil,
						Grafana:    nil,
						Loki:       nil,
						Tempo:      nil,
					},
				},
			},
			validate: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				assert.Nil(t, v1beta1.Spec.Components.Prometheus)
				assert.Nil(t, v1beta1.Spec.Components.Grafana)
				assert.Nil(t, v1beta1.Spec.Components.Loki)
				assert.Nil(t, v1beta1.Spec.Components.Tempo)
			},
		},
		{
			name: "empty maps and slices",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "empty-test",
					Namespace:   "default",
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:     true,
							RemoteWrite: []v1alpha1.RemoteWriteSpec{},
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled:     true,
							DataSources: []v1alpha1.DataSourceConfig{},
						},
					},
					Global: v1alpha1.GlobalConfig{
						ExternalLabels: map[string]string{},
						NodeSelector:   map[string]string{},
						Tolerations:    []v1alpha1.Toleration{},
					},
				},
			},
			validate: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				// Empty maps should be preserved as empty, not nil
				assert.NotNil(t, v1beta1.Labels)
				assert.Len(t, v1beta1.Labels, 0)
				
				// Annotations will have the conversion annotation added
				assert.Contains(t, v1beta1.Annotations, "observability.io/converted-from")
				
				// Empty slices should be preserved
				assert.NotNil(t, v1beta1.Spec.Components.Prometheus.RemoteWrite)
				assert.Len(t, v1beta1.Spec.Components.Prometheus.RemoteWrite, 0)
				
				assert.NotNil(t, v1beta1.Spec.Components.Grafana.DataSources)
				assert.Len(t, v1beta1.Spec.Components.Grafana.DataSources, 0)
				
				// Global config empty collections
				assert.NotNil(t, v1beta1.Spec.Global.ExternalLabels)
				assert.Len(t, v1beta1.Spec.Global.ExternalLabels, 0)
				
				assert.NotNil(t, v1beta1.Spec.Global.NodeSelector)
				assert.Len(t, v1beta1.Spec.Global.NodeSelector, 0)
				
				assert.NotNil(t, v1beta1.Spec.Global.Tolerations)
				assert.Len(t, v1beta1.Spec.Global.Tolerations, 0)
			},
		},
		{
			name: "nil optional fields",
			v1alpha1: &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nil-optional",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Storage: nil,
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Ingress: nil,
						},
						Loki: &v1alpha1.LokiSpec{
							Enabled: true,
							S3:      nil,
						},
					},
					HighAvailability: nil,
					Backup:           nil,
					Alerting:         nil,
				},
			},
			validate: func(t *testing.T, v1beta1 *v1beta1.ObservabilityPlatform) {
				assert.Nil(t, v1beta1.Spec.Components.Prometheus.Storage)
				assert.Nil(t, v1beta1.Spec.Components.Grafana.Ingress)
				assert.Nil(t, v1beta1.Spec.Components.Loki.S3)
				assert.Nil(t, v1beta1.Spec.HighAvailability)
				assert.Nil(t, v1beta1.Spec.Backup)
				assert.Nil(t, v1beta1.Spec.Alerting)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to v1beta1
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			err := tt.v1alpha1.ConvertTo(v1beta1Platform)
			require.NoError(t, err)

			// Validate
			tt.validate(t, v1beta1Platform)
		})
	}
}

// Helper function to create a complete v1alpha1 platform for testing
func createCompleteV1Alpha1Platform() *v1alpha1.ObservabilityPlatform {
	return &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complete-platform",
			Namespace: "monitoring",
			Labels: map[string]string{
				"app":         "observability",
				"environment": "production",
			},
			Annotations: map[string]string{
				"description": "Complete platform for testing",
			},
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Paused: false,
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled:  true,
					Version:  "v2.48.0",
					Replicas: 3,
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "1",
							Memory: "4Gi",
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "2",
							Memory: "8Gi",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size:             "100Gi",
						StorageClassName: "fast-ssd",
					},
					Retention: "30d",
					RemoteWrite: []v1alpha1.RemoteWriteSpec{
						{
							URL:           "https://remote.example.com/write",
							RemoteTimeout: "30s",
							Headers: map[string]string{
								"X-Auth": "secret",
							},
						},
					},
				},
				Grafana: &v1alpha1.GrafanaSpec{
					Enabled:       true,
					Version:       "10.2.0",
					Replicas:      2,
					AdminPassword: "admin123",
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "500m",
							Memory: "1Gi",
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "1",
							Memory: "2Gi",
						},
					},
					Ingress: &v1alpha1.IngressConfig{
						Enabled:   true,
						ClassName: "nginx",
						Host:      "grafana.example.com",
						Path:      "/",
						PathType:  "Prefix",
						TLS: &v1alpha1.IngressTLS{
							Enabled:    true,
							SecretName: "grafana-tls",
						},
					},
					DataSources: []v1alpha1.DataSourceConfig{
						{
							Name:      "Prometheus",
							Type:      "prometheus",
							URL:       "http://prometheus:9090",
							IsDefault: true,
						},
					},
				},
				Loki: &v1alpha1.LokiSpec{
					Enabled:   true,
					Version:   "2.9.0",
					Retention: "7d",
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "1",
							Memory: "2Gi",
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "2",
							Memory: "4Gi",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size:             "200Gi",
						StorageClassName: "standard",
					},
					S3: &v1alpha1.S3Config{
						Enabled:    true,
						BucketName: "loki-logs",
						Region:     "us-east-1",
						Endpoint:   "s3.amazonaws.com",
					},
				},
				Tempo: &v1alpha1.TempoSpec{
					Enabled:   true,
					Version:   "2.3.0",
					Retention: "24h",
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "500m",
							Memory: "1Gi",
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "1",
							Memory: "2Gi",
						},
					},
					Storage: &v1alpha1.StorageConfig{
						Size:             "50Gi",
						StorageClassName: "fast-ssd",
					},
				},
			},
			Global: v1alpha1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster":     "production",
					"region":      "us-east-1",
					"environment": "prod",
				},
				LogLevel: "info",
				NodeSelector: map[string]string{
					"node-role": "observability",
				},
				Tolerations: []v1alpha1.Toleration{
					{
						Key:      "observability",
						Operator: "Equal",
						Value:    "true",
						Effect:   "NoSchedule",
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
				RetentionDays: 7,
				Destination: v1alpha1.BackupDestination{
					Type: "s3",
					S3: &v1alpha1.S3Config{
						BucketName: "backups",
						Region:     "us-east-1",
					},
				},
			},
			Alerting: &v1alpha1.AlertingConfig{
				AlertManager: &v1alpha1.AlertManagerSpec{
					Enabled:  true,
					Replicas: 3,
					Config:   "global:\n  resolve_timeout: 5m",
					Resources: v1alpha1.ResourceRequirements{
						Requests: v1alpha1.ResourceList{
							CPU:    "100m",
							Memory: "256Mi",
						},
						Limits: v1alpha1.ResourceList{
							CPU:    "500m",
							Memory: "512Mi",
						},
					},
				},
				Rules: []v1alpha1.AlertingRule{
					{
						Name: "platform-alerts",
						Groups: []v1alpha1.AlertRuleGroup{
							{
								Name:     "critical",
								Interval: "1m",
								Rules: []v1alpha1.AlertRule{
									{
										Alert: "HighMemoryUsage",
										Expr:  "memory_usage > 90",
										For:   "5m",
										Labels: map[string]string{
											"severity": "critical",
										},
										Annotations: map[string]string{
											"summary": "High memory usage detected",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Status: v1alpha1.ObservabilityPlatformStatus{
			Phase:              "Ready",
			ObservedGeneration: 5,
			Message:            "All components are running",
			ComponentStatus: map[string]v1alpha1.ComponentStatus{
				"prometheus": {
					Phase:         "Ready",
					Version:       "v2.48.0",
					ReadyReplicas: 3,
					TotalReplicas: 3,
				},
			},
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "AllComponentsReady",
					Message:            "All observability components are ready",
				},
			},
		},
	}
}

// TestConversionAnnotations tests that proper annotations are added during conversion
func TestConversionAnnotations(t *testing.T) {
	tests := []struct {
		name                   string
		setupPlatform          func() runtime.Object
		targetVersion          string
		expectedAnnotations    map[string]string
	}{
		{
			name: "v1alpha1 to v1beta1 adds conversion annotation",
			setupPlatform: func() runtime.Object {
				return &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Annotations: map[string]string{
							"existing": "annotation",
						},
					},
				}
			},
			targetVersion: "v1beta1",
			expectedAnnotations: map[string]string{
				"existing":                          "annotation",
				"observability.io/converted-from":   "v1alpha1",
			},
		},
		{
			name: "v1beta1 to v1alpha1 adds conversion and lost fields annotation",
			setupPlatform: func() runtime.Object {
				return &v1beta1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
					},
					Spec: v1beta1.ObservabilityPlatformSpec{
						Components: v1beta1.Components{
							Prometheus: &v1beta1.PrometheusSpec{
								Enabled:        true,
								ExternalLabels: map[string]string{"env": "prod"}, // This will be lost
							},
						},
						Security: &v1beta1.SecurityConfig{ // This entire section will be lost
							TLS: v1beta1.TLSConfig{
								Enabled: true,
							},
						},
					},
				}
			},
			targetVersion: "v1alpha1",
			expectedAnnotations: map[string]string{
				"observability.io/converted-from":        "v1beta1",
				"observability.io/conversion-lost-fields": "Security,Prometheus.ExternalLabels",
			},
		},
		{
			name: "preserve existing conversion annotations",
			setupPlatform: func() runtime.Object {
				return &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						Annotations: map[string]string{
							"observability.io/converted-from":     "v1",
							"observability.io/conversion-history": "v1->v1alpha1",
						},
					},
				}
			},
			targetVersion: "v1beta1",
			expectedAnnotations: map[string]string{
				"observability.io/converted-from":     "v1alpha1",
				"observability.io/conversion-history": "v1->v1alpha1->v1beta1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := tt.setupPlatform()

			var result runtime.Object
			var err error

			switch v := source.(type) {
			case *v1alpha1.ObservabilityPlatform:
				if tt.targetVersion == "v1beta1" {
					target := &v1beta1.ObservabilityPlatform{}
					err = v.ConvertTo(target)
					result = target
				}
			case *v1beta1.ObservabilityPlatform:
				if tt.targetVersion == "v1alpha1" {
					target := &v1alpha1.ObservabilityPlatform{}
					err = target.ConvertFrom(v)
					result = target
				}
			}

			require.NoError(t, err)

			// Check annotations
			var actualAnnotations map[string]string
			switch v := result.(type) {
			case *v1alpha1.ObservabilityPlatform:
				actualAnnotations = v.Annotations
			case *v1beta1.ObservabilityPlatform:
				actualAnnotations = v.Annotations
			}

			for key, expectedValue := range tt.expectedAnnotations {
				assert.Equal(t, expectedValue, actualAnnotations[key], "Annotation %s mismatch", key)
			}
		})
	}
}

// TestConversionErrorHandling tests error scenarios in conversion
func TestConversionErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupTest     func() (runtime.Object, runtime.Object, error)
		expectError   bool
		errorContains string
	}{
		{
			name: "invalid resource quantity conversion",
			setupTest: func() (runtime.Object, runtime.Object, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-resource",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Resources: v1alpha1.ResourceRequirements{
									Requests: v1alpha1.ResourceList{
										CPU:    "invalid-cpu-value",
										Memory: "4Gi",
									},
								},
							},
						},
					},
				}
				
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				err := v1alpha1Platform.ConvertTo(v1beta1Platform)
				return v1alpha1Platform, v1beta1Platform, err
			},
			expectError:   true,
			errorContains: "failed to parse CPU resource",
		},
		{
			name: "nil hub object",
			setupTest: func() (runtime.Object, runtime.Object, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				}
				
				err := v1alpha1Platform.ConvertTo(nil)
				return v1alpha1Platform, nil, err
			},
			expectError:   true,
			errorContains: "hub cannot be nil",
		},
		{
			name: "wrong hub type",
			setupTest: func() (runtime.Object, runtime.Object, error) {
				v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
				}
				
				// Try to convert to wrong type
				wrongType := &corev1.Pod{}
				err := v1alpha1Platform.ConvertTo(wrongType)
				return v1alpha1Platform, wrongType, err
			},
			expectError:   true,
			errorContains: "expected *v1beta1.ObservabilityPlatform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := tt.setupTest()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestJSONSerialization tests that objects can be properly serialized and deserialized
func TestJSONSerialization(t *testing.T) {
	// Create a complex v1alpha1 object
	original := createCompleteV1Alpha1Platform()

	// Convert to v1beta1
	v1beta1Platform := &v1beta1.ObservabilityPlatform{}
	err := original.ConvertTo(v1beta1Platform)
	require.NoError(t, err)

	// Serialize to JSON
	jsonData, err := json.Marshal(v1beta1Platform)
	require.NoError(t, err)

	// Deserialize back
	deserialized := &v1beta1.ObservabilityPlatform{}
	err = json.Unmarshal(jsonData, deserialized)
	require.NoError(t, err)

	// Verify key fields survived serialization
	assert.Equal(t, v1beta1Platform.Name, deserialized.Name)
	assert.Equal(t, v1beta1Platform.Namespace, deserialized.Namespace)
	assert.Equal(t, v1beta1Platform.Spec.Components.Prometheus.Version, deserialized.Spec.Components.Prometheus.Version)
	
	// Convert back to v1alpha1
	final := &v1alpha1.ObservabilityPlatform{}
	err = final.ConvertFrom(deserialized)
	require.NoError(t, err)

	// Verify round-trip through JSON preserved data
	assert.Equal(t, original.Name, final.Name)
	assert.Equal(t, original.Spec.Components.Prometheus.Version, final.Spec.Components.Prometheus.Version)
}
