// Package servicemesh_test provides comprehensive tests for service mesh integration
package servicemesh_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/pkg/servicemesh"
	istioprovider "github.com/gunjanjp/gunj-operator/pkg/servicemesh/istio"
	linkerdprovider "github.com/gunjanjp/gunj-operator/pkg/servicemesh/linkerd"
)

func TestServiceMeshManager(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = observabilityv1.AddToScheme(scheme)

	tests := []struct {
		name     string
		platform *observabilityv1.ObservabilityPlatform
		objects  []client.Object
		wantErr  bool
	}{
		{
			name: "configure service mesh with istio",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: true,
						Type:    "istio",
						MTLS: &observabilityv1.MTLSConfig{
							Enabled: true,
							Mode:    "STRICT",
						},
					},
					Components: &observabilityv1.Components{
						Prometheus: &observabilityv1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "istio-system",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "configure service mesh with linkerd",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: true,
						Type:    "linkerd",
						MTLS: &observabilityv1.MTLSConfig{
							Enabled: true,
						},
					},
					Components: &observabilityv1.Components{
						Prometheus: &observabilityv1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "linkerd",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "service mesh not enabled",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: false,
					},
					Components: &observabilityv1.Components{
						Prometheus: &observabilityv1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			objects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objects...).
				Build()

			// Create manager
			manager := servicemesh.NewExtendedManager(fakeClient, scheme)

			// Execute
			err := manager.ConfigureServiceMesh(ctx, tt.platform)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceMeshHealthCheck(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = observabilityv1.AddToScheme(scheme)

	platform := &observabilityv1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
		Spec: observabilityv1.ObservabilityPlatformSpec{
			ServiceMesh: &observabilityv1.ServiceMeshSpec{
				Enabled: true,
				Type:    "istio",
				HealthCheck: &observabilityv1.ServiceMeshHealthCheck{
					Enabled:  true,
					Interval: "10s",
				},
			},
			Components: &observabilityv1.Components{
				Prometheus: &observabilityv1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}

	istioNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-system",
		},
	}

	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				"istio-injection": "enabled",
			},
		},
	}

	istioPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istiod-123",
			Namespace: "istio-system",
			Labels: map[string]string{
				"app": "istiod",
			},
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	platformPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-0",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"observability.io/platform": "test-platform",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "prometheus"},
				{Name: "istio-proxy"},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "prometheus",
					Ready: true,
				},
				{
					Name:  "istio-proxy",
					Ready: true,
				},
			},
		},
	}

	ctx := context.Background()
	
	// Create fake client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(istioNamespace, testNamespace, istioPod, platformPod).
		Build()

	// Create manager
	manager := servicemesh.NewExtendedManager(fakeClient, scheme)

	// Perform health check
	result, err := manager.PerformHealthCheck(ctx, platform)
	
	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.OverallHealth)
	assert.NotEmpty(t, result.Components)
	assert.True(t, result.Components["control-plane"].Healthy)
	assert.True(t, result.Components["data-plane"].Healthy)
	assert.True(t, result.Components["policies"].Healthy)
}

func TestServiceMeshValidation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1.AddToScheme(scheme)

	tests := []struct {
		name     string
		platform *observabilityv1.ObservabilityPlatform
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid configuration",
			platform: &observabilityv1.ObservabilityPlatform{
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: true,
						MTLS: &observabilityv1.MTLSConfig{
							Mode: "STRICT",
						},
						TrafficManagement: &observabilityv1.TrafficManagementConfig{
							LoadBalancing: &observabilityv1.LoadBalancingConfig{
								Algorithm: "ROUND_ROBIN",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mTLS mode",
			platform: &observabilityv1.ObservabilityPlatform{
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: true,
						MTLS: &observabilityv1.MTLSConfig{
							Mode: "INVALID",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid mTLS mode",
		},
		{
			name: "invalid load balancing algorithm",
			platform: &observabilityv1.ObservabilityPlatform{
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: true,
						TrafficManagement: &observabilityv1.TrafficManagementConfig{
							LoadBalancing: &observabilityv1.LoadBalancingConfig{
								Algorithm: "INVALID_ALGO",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid load balancing algorithm",
		},
		{
			name: "invalid circuit breaker configuration",
			platform: &observabilityv1.ObservabilityPlatform{
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: true,
						CircuitBreaker: &observabilityv1.CircuitBreakerConfig{
							Enabled:           true,
							ConsecutiveErrors: 0,
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "consecutive errors must be at least 1",
		},
		{
			name: "invalid rate limiting configuration",
			platform: &observabilityv1.ObservabilityPlatform{
				Spec: observabilityv1.ObservabilityPlatformSpec{
					ServiceMesh: &observabilityv1.ServiceMeshSpec{
						Enabled: true,
						RateLimiting: &observabilityv1.RateLimitingConfig{
							Enabled: true,
							Rules: []observabilityv1.RateLimitRule{
								{
									Name: "test-rule",
									Limit: &observabilityv1.RateLimit{
										RequestsPerUnit: 0,
										Unit:            "SECOND",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "rate limit requests per unit must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			// Create manager
			manager := servicemesh.NewExtendedManager(fakeClient, scheme)

			// Execute
			err := manager.ValidateServiceMeshConfig(tt.platform)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceMeshProviders(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	t.Run("istio provider", func(t *testing.T) {
		ctx := context.Background()
		
		// Create fake client
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "istio-system",
				},
			}).
			Build()

		// Create provider
		provider := istioprovider.NewProvider(fakeClient)

		// Test IsInstalled
		installed, err := provider.IsInstalled(ctx)
		assert.NoError(t, err)
		assert.False(t, installed) // No pods, so not fully installed
	})

	t.Run("linkerd provider", func(t *testing.T) {
		ctx := context.Background()
		
		// Create fake client
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "linkerd",
				},
			}).
			Build()

		// Create provider
		provider := linkerdprovider.NewProvider(fakeClient)

		// Test IsInstalled
		installed, err := provider.IsInstalled(ctx)
		assert.NoError(t, err)
		assert.False(t, installed) // No pods, so not fully installed
	})
}

func TestServiceMeshMetrics(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = observabilityv1.AddToScheme(scheme)

	platform := &observabilityv1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
		Spec: observabilityv1.ObservabilityPlatformSpec{
			ServiceMesh: &observabilityv1.ServiceMeshSpec{
				Enabled: true,
				Type:    "istio",
			},
		},
	}

	ctx := context.Background()
	
	// Create fake client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "istio-system",
			},
		}).
		Build()

	// Create manager
	manager := servicemesh.NewExtendedManager(fakeClient, scheme)

	// Get metrics
	metrics, err := manager.GetServiceMeshMetrics(ctx, platform)
	
	// Assert - will fail because no mesh is actually installed
	assert.Error(t, err)
	assert.Nil(t, metrics)
}

// BenchmarkServiceMeshConfiguration benchmarks service mesh configuration
func BenchmarkServiceMeshConfiguration(b *testing.B) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = observabilityv1.AddToScheme(scheme)

	platform := &observabilityv1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bench-platform",
			Namespace: "bench-namespace",
		},
		Spec: observabilityv1.ObservabilityPlatformSpec{
			ServiceMesh: &observabilityv1.ServiceMeshSpec{
				Enabled: true,
				Type:    "auto",
			},
			Components: &observabilityv1.Components{
				Prometheus: &observabilityv1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}

	ctx := context.Background()
	
	// Create fake client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bench-namespace",
			},
		}).
		Build()

	// Create manager
	manager := servicemesh.NewExtendedManager(fakeClient, scheme)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ConfigureServiceMesh(ctx, platform)
	}
}
