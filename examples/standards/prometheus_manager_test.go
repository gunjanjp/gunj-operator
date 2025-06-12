package example_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/client/fake"
    logf "sigs.k8s.io/controller-runtime/pkg/log"
    
    observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
    "github.com/gunjanjp/gunj-operator/examples/standards"
)

// TestPrometheusManagerDeploy demonstrates table-driven testing pattern
func TestPrometheusManagerDeploy(t *testing.T) {
    // Define test cases using table-driven pattern
    tests := []struct {
        name      string
        platform  *observabilityv1.ObservabilityPlatform
        objects   []runtime.Object
        wantErr   bool
        errMsg    string
        validate  func(t *testing.T, client client.Client)
    }{
        {
            name: "successful deployment with defaults",
            platform: &observabilityv1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-platform",
                    Namespace: "monitoring",
                },
                Spec: observabilityv1.ObservabilityPlatformSpec{
                    Components: observabilityv1.Components{
                        Prometheus: &observabilityv1.PrometheusSpec{
                            Enabled: true,
                            Version: "v2.45.0",
                        },
                    },
                },
            },
            wantErr: false,
            validate: func(t *testing.T, c client.Client) {
                // Verify deployment was created
                deployment := &appsv1.Deployment{}
                err := c.Get(context.Background(), types.NamespacedName{
                    Name:      "test-platform-prometheus",
                    Namespace: "monitoring",
                }, deployment)
                require.NoError(t, err, "deployment should exist")
                assert.Equal(t, int32(1), *deployment.Spec.Replicas, "should have 1 replica by default")
                
                // Verify service was created
                service := &corev1.Service{}
                err = c.Get(context.Background(), types.NamespacedName{
                    Name:      "test-platform-prometheus",
                    Namespace: "monitoring",
                }, service)
                require.NoError(t, err, "service should exist")
                assert.Equal(t, "web", service.Spec.Ports[0].Name)
                
                // Verify configmap was created
                configMap := &corev1.ConfigMap{}
                err = c.Get(context.Background(), types.NamespacedName{
                    Name:      "test-platform-prometheus-config",
                    Namespace: "monitoring",
                }, configMap)
                require.NoError(t, err, "configmap should exist")
                assert.Contains(t, configMap.Data["prometheus.yml"], "scrape_interval: 15s")
            },
        },
        {
            name: "deployment with custom replicas and resources",
            platform: &observabilityv1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "ha-platform",
                    Namespace: "monitoring",
                },
                Spec: observabilityv1.ObservabilityPlatformSpec{
                    Components: observabilityv1.Components{
                        Prometheus: &observabilityv1.PrometheusSpec{
                            Enabled:  true,
                            Version:  "v2.45.0",
                            Replicas: ptr(int32(3)),
                            Resources: corev1.ResourceRequirements{
                                Requests: corev1.ResourceList{
                                    corev1.ResourceMemory: resource.MustParse("4Gi"),
                                    corev1.ResourceCPU:    resource.MustParse("1"),
                                },
                                Limits: corev1.ResourceList{
                                    corev1.ResourceMemory: resource.MustParse("8Gi"),
                                    corev1.ResourceCPU:    resource.MustParse("2"),
                                },
                            },
                        },
                    },
                },
            },
            wantErr: false,
            validate: func(t *testing.T, c client.Client) {
                deployment := &appsv1.Deployment{}
                err := c.Get(context.Background(), types.NamespacedName{
                    Name:      "ha-platform-prometheus",
                    Namespace: "monitoring",
                }, deployment)
                require.NoError(t, err)
                
                assert.Equal(t, int32(3), *deployment.Spec.Replicas, "should have 3 replicas")
                
                container := deployment.Spec.Template.Spec.Containers[0]
                assert.Equal(t, resource.MustParse("4Gi"), container.Resources.Requests[corev1.ResourceMemory])
                assert.Equal(t, resource.MustParse("1"), container.Resources.Requests[corev1.ResourceCPU])
            },
        },
        {
            name: "skip deployment when prometheus disabled",
            platform: &observabilityv1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "disabled-platform",
                    Namespace: "monitoring",
                },
                Spec: observabilityv1.ObservabilityPlatformSpec{
                    Components: observabilityv1.Components{
                        Prometheus: &observabilityv1.PrometheusSpec{
                            Enabled: false,
                            Version: "v2.45.0",
                        },
                    },
                },
            },
            wantErr: false,
            validate: func(t *testing.T, c client.Client) {
                // Verify no resources were created
                deployment := &appsv1.Deployment{}
                err := c.Get(context.Background(), types.NamespacedName{
                    Name:      "disabled-platform-prometheus",
                    Namespace: "monitoring",
                }, deployment)
                assert.True(t, errors.IsNotFound(err), "deployment should not exist")
            },
        },
        {
            name:     "error when platform is nil",
            platform: nil,
            wantErr:  true,
            errMsg:   "platform cannot be nil",
        },
    }
    
    // Run test cases
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            scheme := runtime.NewScheme()
            _ = observabilityv1.AddToScheme(scheme)
            _ = appsv1.AddToScheme(scheme)
            _ = corev1.AddToScheme(scheme)
            
            // Create fake client with initial objects
            builder := fake.NewClientBuilder().WithScheme(scheme)
            if tt.platform != nil {
                builder = builder.WithRuntimeObjects(tt.platform)
            }
            if len(tt.objects) > 0 {
                builder = builder.WithRuntimeObjects(tt.objects...)
            }
            client := builder.Build()
            
            // Create manager
            log := logf.Log.WithName("test")
            manager := standards.NewPrometheusManager(client, scheme, log)
            
            // Execute
            ctx := context.Background()
            err := manager.Deploy(ctx, tt.platform)
            
            // Assert error
            if tt.wantErr {
                assert.Error(t, err)
                if tt.errMsg != "" {
                    assert.Contains(t, err.Error(), tt.errMsg)
                }
                return
            }
            
            assert.NoError(t, err)
            
            // Run additional validations
            if tt.validate != nil {
                tt.validate(t, client)
            }
        })
    }
}

// TestPrometheusManagerGetStatus demonstrates status checking tests
func TestPrometheusManagerGetStatus(t *testing.T) {
    tests := []struct {
        name           string
        platform       *observabilityv1.ObservabilityPlatform
        deployment     *appsv1.Deployment
        expectedStatus standards.ComponentStatus
        wantErr        bool
    }{
        {
            name: "ready deployment",
            platform: &observabilityv1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-platform",
                    Namespace: "monitoring",
                },
            },
            deployment: &appsv1.Deployment{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-platform-prometheus",
                    Namespace: "monitoring",
                },
                Spec: appsv1.DeploymentSpec{
                    Replicas: ptr(int32(3)),
                },
                Status: appsv1.DeploymentStatus{
                    ReadyReplicas: 3,
                },
            },
            expectedStatus: standards.ComponentStatus{
                Ready:   true,
                Message: "Prometheus is ready with 3/3 replicas",
            },
            wantErr: false,
        },
        {
            name: "not ready deployment",
            platform: &observabilityv1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-platform",
                    Namespace: "monitoring",
                },
            },
            deployment: &appsv1.Deployment{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-platform-prometheus",
                    Namespace: "monitoring",
                },
                Spec: appsv1.DeploymentSpec{
                    Replicas: ptr(int32(3)),
                },
                Status: appsv1.DeploymentStatus{
                    ReadyReplicas: 1,
                },
            },
            expectedStatus: standards.ComponentStatus{
                Ready:   false,
                Message: "Prometheus is not ready: 1/3 replicas ready",
            },
            wantErr: false,
        },
        {
            name: "deployment not found",
            platform: &observabilityv1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-platform",
                    Namespace: "monitoring",
                },
            },
            deployment: nil,
            expectedStatus: standards.ComponentStatus{
                Ready:   false,
                Message: "Prometheus deployment not found",
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            scheme := runtime.NewScheme()
            _ = observabilityv1.AddToScheme(scheme)
            _ = appsv1.AddToScheme(scheme)
            
            builder := fake.NewClientBuilder().WithScheme(scheme)
            if tt.deployment != nil {
                builder = builder.WithRuntimeObjects(tt.deployment)
            }
            client := builder.Build()
            
            log := logf.Log.WithName("test")
            manager := standards.NewPrometheusManager(client, scheme, log)
            
            // Execute
            ctx := context.Background()
            status, err := manager.GetStatus(ctx, tt.platform)
            
            // Assert
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedStatus.Ready, status.Ready)
            assert.Equal(t, tt.expectedStatus.Message, status.Message)
            assert.WithinDuration(t, time.Now(), status.LastUpdateTime, 5*time.Second)
        })
    }
}

// BenchmarkPrometheusManagerDeploy demonstrates benchmark testing
func BenchmarkPrometheusManagerDeploy(b *testing.B) {
    // Setup
    scheme := runtime.NewScheme()
    _ = observabilityv1.AddToScheme(scheme)
    _ = appsv1.AddToScheme(scheme)
    _ = corev1.AddToScheme(scheme)
    
    platform := &observabilityv1.ObservabilityPlatform{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "bench-platform",
            Namespace: "monitoring",
        },
        Spec: observabilityv1.ObservabilityPlatformSpec{
            Components: observabilityv1.Components{
                Prometheus: &observabilityv1.PrometheusSpec{
                    Enabled: true,
                    Version: "v2.45.0",
                },
            },
        },
    }
    
    client := fake.NewClientBuilder().
        WithScheme(scheme).
        WithRuntimeObjects(platform).
        Build()
    
    log := logf.Log.WithName("benchmark")
    manager := standards.NewPrometheusManager(client, scheme, log)
    ctx := context.Background()
    
    // Reset timer to exclude setup
    b.ResetTimer()
    
    // Run benchmark
    for i := 0; i < b.N; i++ {
        _ = manager.Deploy(ctx, platform)
    }
}

// Helper function to create pointer - demonstrates test helpers
func ptr[T any](v T) *T {
    return &v
}

// TestHelper demonstrates the use of test helper functions
type testHelper struct {
    t      *testing.T
    client client.Client
}

func newTestHelper(t *testing.T) *testHelper {
    scheme := runtime.NewScheme()
    _ = observabilityv1.AddToScheme(scheme)
    _ = appsv1.AddToScheme(scheme)
    _ = corev1.AddToScheme(scheme)
    
    client := fake.NewClientBuilder().WithScheme(scheme).Build()
    
    return &testHelper{
        t:      t,
        client: client,
    }
}

func (h *testHelper) createPlatform(name, namespace string) *observabilityv1.ObservabilityPlatform {
    platform := &observabilityv1.ObservabilityPlatform{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: namespace,
        },
        Spec: observabilityv1.ObservabilityPlatformSpec{
            Components: observabilityv1.Components{
                Prometheus: &observabilityv1.PrometheusSpec{
                    Enabled: true,
                    Version: "v2.45.0",
                },
            },
        },
    }
    
    err := h.client.Create(context.Background(), platform)
    require.NoError(h.t, err)
    
    return platform
}

// TestWithHelper demonstrates using test helpers
func TestWithHelper(t *testing.T) {
    helper := newTestHelper(t)
    
    // Create test platform using helper
    platform := helper.createPlatform("test-platform", "monitoring")
    
    // Verify it was created
    retrieved := &observabilityv1.ObservabilityPlatform{}
    err := helper.client.Get(context.Background(), client.ObjectKeyFromObject(platform), retrieved)
    assert.NoError(t, err)
    assert.Equal(t, platform.Name, retrieved.Name)
}
