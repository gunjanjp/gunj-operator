/*
Copyright 2025.

Licensed under the MIT License.
*/

package prometheus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestPrometheusManager_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = policyv1.AddToScheme(scheme)

	tests := []struct {
		name          string
		platform      *observabilityv1beta1.ObservabilityPlatform
		existingObjs  []client.Object
		expectedError bool
		validate      func(t *testing.T, c client.Client)
	}{
		{
			name: "should create all resources when Prometheus is enabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 2,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("4Gi"),
									corev1.ResourceCPU:    resource.MustParse("1"),
								},
							},
							Storage: observabilityv1beta1.StorageSpec{
								Size: resource.MustParse("100Gi"),
							},
							Retention: "30d",
						},
					},
				},
			},
			expectedError: false,
			validate: func(t *testing.T, c client.Client) {
				// Check ConfigMap
				cm := &corev1.ConfigMap{}
				err := c.Get(context.TODO(), types.NamespacedName{
					Name:      "prometheus-test-platform-config",
					Namespace: "test-namespace",
				}, cm)
				assert.NoError(t, err)
				assert.Contains(t, cm.Data, "prometheus.yml")
				assert.Contains(t, cm.Data["prometheus.yml"], "scrape_interval: 15s")

				// Check Service
				svc := &corev1.Service{}
				err = c.Get(context.TODO(), types.NamespacedName{
					Name:      "prometheus-test-platform",
					Namespace: "test-namespace",
				}, svc)
				assert.NoError(t, err)
				assert.Equal(t, int32(9090), svc.Spec.Ports[0].Port)

				// Check StatefulSet
				sts := &appsv1.StatefulSet{}
				err = c.Get(context.TODO(), types.NamespacedName{
					Name:      "prometheus-test-platform",
					Namespace: "test-namespace",
				}, sts)
				assert.NoError(t, err)
				assert.Equal(t, int32(2), *sts.Spec.Replicas)
				assert.Equal(t, "prom/prometheus:v2.48.0", sts.Spec.Template.Spec.Containers[0].Image)

				// Check PodDisruptionBudget
				pdb := &policyv1.PodDisruptionBudget{}
				err = c.Get(context.TODO(), types.NamespacedName{
					Name:      "prometheus-test-platform",
					Namespace: "test-namespace",
				}, pdb)
				assert.NoError(t, err)
				assert.Equal(t, int32(1), pdb.Spec.MinAvailable.IntVal)
			},
		},
		{
			name: "should skip reconciliation when Prometheus is disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: false,
							Version: "v2.48.0",
						},
					},
				},
			},
			expectedError: false,
			validate: func(t *testing.T, c client.Client) {
				// Check that no resources were created
				cm := &corev1.ConfigMap{}
				err := c.Get(context.TODO(), types.NamespacedName{
					Name:      "prometheus-test-platform-config",
					Namespace: "test-namespace",
				}, cm)
				assert.Error(t, err)
			},
		},
		{
			name: "should handle remote write configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1,
							RemoteWrite: []observabilityv1beta1.RemoteWriteSpec{
								{
									URL:           "https://remote-write.example.com/api/v1/write",
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
			expectedError: false,
			validate: func(t *testing.T, c client.Client) {
				// Check ConfigMap contains remote write config
				cm := &corev1.ConfigMap{}
				err := c.Get(context.TODO(), types.NamespacedName{
					Name:      "prometheus-test-platform-config",
					Namespace: "test-namespace",
				}, cm)
				assert.NoError(t, err)
				assert.Contains(t, cm.Data["prometheus.yml"], "remote_write:")
				assert.Contains(t, cm.Data["prometheus.yml"], "https://remote-write.example.com/api/v1/write")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			objs := append(tt.existingObjs, tt.platform)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			// Create manager
			manager := NewPrometheusManager(fakeClient, scheme)

			// Reconcile
			err := manager.Reconcile(context.TODO(), tt.platform)

			// Check error
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Validate
			if tt.validate != nil {
				tt.validate(t, fakeClient)
			}
		})
	}
}

func TestPrometheusManager_GetStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	tests := []struct {
		name           string
		platform       *observabilityv1beta1.ObservabilityPlatform
		existingObjs   []client.Object
		expectedPhase  string
		expectedReady  int32
		expectedError  bool
	}{
		{
			name: "should return Ready when all replicas are ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 3,
						},
					},
				},
			},
			existingObjs: []client.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-test-platform",
						Namespace: "test-namespace",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &[]int32{3}[0],
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      3,
						ReadyReplicas: 3,
					},
				},
			},
			expectedPhase: "Ready",
			expectedReady: 3,
			expectedError: false,
		},
		{
			name: "should return Degraded when some replicas are not ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 3,
						},
					},
				},
			},
			existingObjs: []client.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-test-platform",
						Namespace: "test-namespace",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &[]int32{3}[0],
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      3,
						ReadyReplicas: 1,
					},
				},
			},
			expectedPhase: "Degraded",
			expectedReady: 1,
			expectedError: false,
		},
		{
			name: "should return Disabled when Prometheus is disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: false,
						},
					},
				},
			},
			expectedPhase: "Disabled",
			expectedReady: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			objs := append(tt.existingObjs, tt.platform)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			// Create manager
			manager := NewPrometheusManager(fakeClient, scheme)

			// Get status
			status, err := manager.GetStatus(context.TODO(), tt.platform)

			// Check error
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, status)
				assert.Equal(t, tt.expectedPhase, status.Phase)
				assert.Equal(t, tt.expectedReady, status.Ready)
			}
		})
	}
}

func TestPrometheusManager_Validate(t *testing.T) {
	tests := []struct {
		name          string
		platform      *observabilityv1beta1.ObservabilityPlatform
		expectedError bool
		errorContains string
	}{
		{
			name: "should pass validation with valid configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Version:   "v2.48.0",
							Retention: "30d",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("4Gi"),
									corev1.ResourceCPU:    resource.MustParse("1"),
								},
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "should fail validation with invalid version format",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Version: "2.48.0", // Missing 'v' prefix
						},
					},
				},
			},
			expectedError: true,
			errorContains: "version must start with 'v'",
		},
		{
			name: "should fail validation with invalid retention format",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Version:   "v2.48.0",
							Retention: "30days", // Invalid format
						},
					},
				},
			},
			expectedError: true,
			errorContains: "invalid retention format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &PrometheusManager{}
			err := manager.Validate(tt.platform)

			if tt.expectedError {
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

func TestPrometheusManager_GetServiceURL(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "monitoring",
		},
	}

	manager := &PrometheusManager{}
	url := manager.GetServiceURL(platform)

	expected := "http://prometheus-test-platform.monitoring.svc.cluster.local:9090"
	assert.Equal(t, expected, url)
}

func TestPrometheusManager_ScaleReplicas(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-test-platform",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &[]int32{1}[0],
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, sts).
		Build()

	manager := NewPrometheusManager(fakeClient, scheme)

	// Scale to 3 replicas
	err := manager.ScaleReplicas(context.TODO(), platform, 3)
	assert.NoError(t, err)

	// Verify the StatefulSet was updated
	updatedSts := &appsv1.StatefulSet{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{
		Name:      "prometheus-test-platform",
		Namespace: "test-namespace",
	}, updatedSts)
	assert.NoError(t, err)
	assert.Equal(t, int32(3), *updatedSts.Spec.Replicas)
}

func TestPrometheusManager_generatePrometheusConfig(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Global: observabilityv1beta1.GlobalConfig{
				ExternalLabels: map[string]string{
					"cluster": "production",
					"region":  "us-east-1",
				},
			},
		},
	}

	prometheusSpec := &observabilityv1beta1.PrometheusSpec{
		ExternalLabels: map[string]string{
			"prometheus": "main",
		},
		RemoteWrite: []observabilityv1beta1.RemoteWriteSpec{
			{
				URL:           "https://remote.example.com/write",
				RemoteTimeout: "30s",
				Headers: map[string]string{
					"X-API-Key": "secret",
				},
			},
		},
	}

	manager := &PrometheusManager{}
	config := manager.generatePrometheusConfig(platform, prometheusSpec)

	// Check global configuration
	assert.Contains(t, config, "scrape_interval: 15s")
	assert.Contains(t, config, "evaluation_interval: 15s")

	// Check external labels
	assert.Contains(t, config, "external_labels:")
	assert.Contains(t, config, "cluster: production")
	assert.Contains(t, config, "region: us-east-1")
	assert.Contains(t, config, "prometheus: main")

	// Check scrape configs
	assert.Contains(t, config, "job_name: 'prometheus'")
	assert.Contains(t, config, "job_name: 'kubernetes-apiservers'")
	assert.Contains(t, config, "job_name: 'kubernetes-nodes'")
	assert.Contains(t, config, "job_name: 'kubernetes-pods'")

	// Check remote write
	assert.Contains(t, config, "remote_write:")
	assert.Contains(t, config, "url: https://remote.example.com/write")
	assert.Contains(t, config, "remote_timeout: 30s")
	assert.Contains(t, config, "X-API-Key: secret")
}
