/*
Copyright 2025.

Licensed under the MIT License.
*/

package tempo

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestNewTempoManager(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	
	manager := NewTempoManager(client, scheme)
	assert.NotNil(t, manager)
	assert.IsType(t, &TempoManager{}, manager)
}

func TestTempoManager_Reconcile(t *testing.T) {
	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErr  bool
		checkFn  func(t *testing.T, c client.Client)
	}{
		{
			name: "successfully reconcile tempo with all resources",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 3,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							Storage: observabilityv1beta1.StorageSpec{
								Size:             "10Gi",
								StorageClassName: ptrString("fast-ssd"),
							},
							Retention: "168h",
						},
					},
					Global: observabilityv1beta1.GlobalConfig{
						ExternalLabels: map[string]string{
							"cluster": "test",
							"env":     "dev",
						},
					},
				},
			},
			wantErr: false,
			checkFn: func(t *testing.T, c client.Client) {
				// Check ConfigMap
				cm := &corev1.ConfigMap{}
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo-config",
					Namespace: "test-namespace",
				}, cm)
				assert.NoError(t, err)
				assert.Contains(t, cm.Data["tempo.yaml"], "http_listen_port: 3200")
				assert.Contains(t, cm.Data["tempo.yaml"], "grpc_listen_port: 9095")
				assert.Contains(t, cm.Data["tempo.yaml"], "max_search_duration: 168h")
				
				// Check Services
				svc := &corev1.Service{}
				err = c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo",
					Namespace: "test-namespace",
				}, svc)
				assert.NoError(t, err)
				assert.Len(t, svc.Spec.Ports, 9) // All receiver ports
				
				// Check headless service
				headlessSvc := &corev1.Service{}
				err = c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo-headless",
					Namespace: "test-namespace",
				}, headlessSvc)
				assert.NoError(t, err)
				assert.Equal(t, "None", headlessSvc.Spec.ClusterIP)
				
				// Check StatefulSet
				sts := &appsv1.StatefulSet{}
				err = c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo",
					Namespace: "test-namespace",
				}, sts)
				assert.NoError(t, err)
				assert.Equal(t, int32(3), *sts.Spec.Replicas)
				assert.Equal(t, "test-platform-tempo-headless", sts.Spec.ServiceName)
				assert.Len(t, sts.Spec.Template.Spec.Containers, 1)
				assert.Equal(t, "grafana/tempo:2.3.0", sts.Spec.Template.Spec.Containers[0].Image)
				
				// Check PodDisruptionBudget
				pdb := &policyv1.PodDisruptionBudget{}
				err = c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo-pdb",
					Namespace: "test-namespace",
				}, pdb)
				assert.NoError(t, err)
				assert.Equal(t, intstr.FromString("50%"), *pdb.Spec.MinAvailable)
			},
		},
		{
			name: "successfully reconcile tempo with S3 storage",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 1,
							Storage: observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
							S3: &observabilityv1beta1.S3Config{
								BucketName:      "tempo-traces",
								Region:          "us-east-1",
								Endpoint:        "s3.amazonaws.com",
								AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
								SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
							},
							Retention: "336h",
						},
					},
					Global: observabilityv1beta1.GlobalConfig{
						ExternalLabels: map[string]string{
							"cluster": "prod",
						},
					},
				},
			},
			wantErr: false,
			checkFn: func(t *testing.T, c client.Client) {
				// Check ConfigMap for S3 configuration
				cm := &corev1.ConfigMap{}
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo-config",
					Namespace: "test-namespace",
				}, cm)
				assert.NoError(t, err)
				assert.Contains(t, cm.Data["tempo.yaml"], "backend: s3")
				assert.Contains(t, cm.Data["tempo.yaml"], "bucket: tempo-traces")
				assert.Contains(t, cm.Data["tempo.yaml"], "region: us-east-1")
				assert.Contains(t, cm.Data["tempo.yaml"], "endpoint: s3.amazonaws.com")
				assert.Contains(t, cm.Data["tempo.yaml"], "access_key: AKIAIOSFODNN7EXAMPLE")
				assert.Contains(t, cm.Data["tempo.yaml"], "secret_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
			},
		},
		{
			name: "skip reconciliation when tempo is disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled: false,
							Version: "v2.3.0",
						},
					},
				},
			},
			wantErr: false,
			checkFn: func(t *testing.T, c client.Client) {
				// Check that no resources are created
				cm := &corev1.ConfigMap{}
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo-config",
					Namespace: "test-namespace",
				}, cm)
				assert.True(t, errors.IsNotFound(err))
			},
		},
		{
			name: "skip reconciliation when tempo spec is nil",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{},
				},
			},
			wantErr: false,
			checkFn: func(t *testing.T, c client.Client) {
				// Check that no resources are created
				sts := &appsv1.StatefulSet{}
				err := c.Get(context.Background(), types.NamespacedName{
					Name:      "test-platform-tempo",
					Namespace: "test-namespace",
				}, sts)
				assert.True(t, errors.IsNotFound(err))
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			scheme := runtime.NewScheme()
			_ = observabilityv1beta1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			_ = policyv1.AddToScheme(scheme)
			
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.platform).
				Build()
			
			manager := &TempoManager{
				Client: c,
				Scheme: scheme,
			}
			
			// Execute
			ctx := context.Background()
			ctx = log.IntoContext(ctx, log.Log)
			err := manager.Reconcile(ctx, tt.platform)
			
			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			if tt.checkFn != nil {
				tt.checkFn(t, c)
			}
		})
	}
}

func TestTempoManager_Delete(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = policyv1.AddToScheme(scheme)
	
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}
	
	// Create resources that should be deleted
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo-config",
			Namespace: "test-namespace",
		},
	}
	
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo",
			Namespace: "test-namespace",
		},
	}
	
	headlessSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo-headless",
			Namespace: "test-namespace",
		},
	}
	
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo",
			Namespace: "test-namespace",
		},
	}
	
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo-pdb",
			Namespace: "test-namespace",
		},
	}
	
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, cm, svc, headlessSvc, sts, pdb).
		Build()
	
	manager := &TempoManager{
		Client: c,
		Scheme: scheme,
	}
	
	// Execute
	ctx := context.Background()
	ctx = log.IntoContext(ctx, log.Log)
	err := manager.Delete(ctx, platform)
	
	// Assert
	assert.NoError(t, err)
	
	// Verify all resources are deleted
	err = c.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, cm)
	assert.True(t, errors.IsNotFound(err), "ConfigMap should be deleted")
	
	err = c.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, svc)
	assert.True(t, errors.IsNotFound(err), "Service should be deleted")
	
	err = c.Get(ctx, types.NamespacedName{Name: headlessSvc.Name, Namespace: headlessSvc.Namespace}, headlessSvc)
	assert.True(t, errors.IsNotFound(err), "Headless service should be deleted")
	
	err = c.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, sts)
	assert.True(t, errors.IsNotFound(err), "StatefulSet should be deleted")
	
	err = c.Get(ctx, types.NamespacedName{Name: pdb.Name, Namespace: pdb.Namespace}, pdb)
	assert.True(t, errors.IsNotFound(err), "PodDisruptionBudget should be deleted")
}

func TestTempoManager_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		platform       *observabilityv1beta1.ObservabilityPlatform
		statefulSet    *appsv1.StatefulSet
		expectedStatus observabilityv1beta1.ComponentStatus
		wantErr        bool
	}{
		{
			name: "tempo is disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled: false,
						},
					},
				},
			},
			expectedStatus: observabilityv1beta1.ComponentStatus{
				Name:    "tempo",
				Status:  observabilityv1beta1.ComponentStatusDisabled,
				Message: "Tempo is disabled",
			},
			wantErr: false,
		},
		{
			name: "all replicas ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Replicas: 3,
						},
					},
				},
			},
			statefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-tempo",
					Namespace: "test-namespace",
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: ptrInt32(3),
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      3,
					ReadyReplicas: 3,
				},
			},
			expectedStatus: observabilityv1beta1.ComponentStatus{
				Name:    "tempo",
				Status:  observabilityv1beta1.ComponentStatusReady,
				Message: "All 3 replicas are ready",
				Ready:   true,
			},
			wantErr: false,
		},
		{
			name: "partial replicas ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Replicas: 3,
						},
					},
				},
			},
			statefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-tempo",
					Namespace: "test-namespace",
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: ptrInt32(3),
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:      3,
					ReadyReplicas: 1,
				},
			},
			expectedStatus: observabilityv1beta1.ComponentStatus{
				Name:    "tempo",
				Status:  observabilityv1beta1.ComponentStatusPending,
				Message: "Waiting for replicas: 1/3 ready",
				Ready:   false,
			},
			wantErr: false,
		},
		{
			name: "statefulset not found",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled: true,
						},
					},
				},
			},
			statefulSet: nil,
			expectedStatus: observabilityv1beta1.ComponentStatus{
				Name:    "tempo",
				Status:  observabilityv1beta1.ComponentStatusFailed,
				Message: "Failed to get StatefulSet: statefulsets.apps \"test-platform-tempo\" not found",
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			scheme := runtime.NewScheme()
			_ = observabilityv1beta1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			
			objs := []client.Object{tt.platform}
			if tt.statefulSet != nil {
				objs = append(objs, tt.statefulSet)
			}
			
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()
			
			manager := &TempoManager{
				Client: c,
				Scheme: scheme,
			}
			
			// Execute
			ctx := context.Background()
			ctx = log.IntoContext(ctx, log.Log)
			status, err := manager.GetStatus(ctx, tt.platform)
			
			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus.Name, status.Name)
				assert.Equal(t, tt.expectedStatus.Status, status.Status)
				assert.Equal(t, tt.expectedStatus.Message, status.Message)
				assert.Equal(t, tt.expectedStatus.Ready, status.Ready)
			}
		})
	}
}

func TestTempoManager_Validate(t *testing.T) {
	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 3,
							Storage: observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid configuration with S3",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 1,
							Storage: observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
							S3: &observabilityv1beta1.S3Config{
								BucketName: "tempo-traces",
								Region:     "us-east-1",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tempo disabled - skip validation",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled: false,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Replicas: 1,
							Storage: observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "tempo version is required",
		},
		{
			name: "invalid replicas",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 0,
							Storage: observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "tempo replicas must be at least 1",
		},
		{
			name: "missing storage size",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 1,
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "tempo storage size is required",
		},
		{
			name: "S3 missing bucket name",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 1,
							Storage: observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
							S3: &observabilityv1beta1.S3Config{
								Region: "us-east-1",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "tempo S3 bucket name is required when S3 is configured",
		},
		{
			name: "S3 missing region",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Version:  "v2.3.0",
							Replicas: 1,
							Storage: observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
							S3: &observabilityv1beta1.S3Config{
								BucketName: "tempo-traces",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "tempo S3 region is required when S3 is configured",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &TempoManager{}
			err := manager.Validate(tt.platform)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTempoManager_GetServiceURL(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}
	
	manager := &TempoManager{}
	url := manager.GetServiceURL(platform)
	
	expected := "test-platform-tempo.test-namespace.svc.cluster.local:3200"
	assert.Equal(t, expected, url)
}

func TestTempoManager_ConfigureReceivers(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Tempo: &observabilityv1beta1.TempoSpec{
					Enabled:  true,
					Version:  "v2.3.0",
					Replicas: 1,
					Storage: observabilityv1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			},
		},
	}
	
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo-config",
			Namespace: "test-namespace",
		},
	}
	
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}
	
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, cm, sts).
		Build()
	
	manager := &TempoManager{
		Client: c,
		Scheme: scheme,
	}
	
	// Execute
	ctx := context.Background()
	ctx = log.IntoContext(ctx, log.Log)
	err := manager.ConfigureReceivers(ctx, platform)
	
	// Assert
	assert.NoError(t, err)
	
	// Verify StatefulSet was updated with new annotation
	updatedSts := &appsv1.StatefulSet{}
	err = c.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, updatedSts)
	assert.NoError(t, err)
	assert.Contains(t, updatedSts.Spec.Template.Annotations, "tempo.observability.io/config-hash")
}

func TestTempoManager_UpdateSampling(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Tempo: &observabilityv1beta1.TempoSpec{
					Enabled:  true,
					Version:  "v2.3.0",
					Replicas: 1,
					Storage: observabilityv1beta1.StorageSpec{
						Size: "10Gi",
					},
				},
			},
		},
	}
	
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo-config",
			Namespace: "test-namespace",
		},
	}
	
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform-tempo",
			Namespace: "test-namespace",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}
	
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, cm, sts).
		Build()
	
	manager := &TempoManager{
		Client: c,
		Scheme: scheme,
	}
	
	// Execute
	ctx := context.Background()
	ctx = log.IntoContext(ctx, log.Log)
	err := manager.UpdateSampling(ctx, platform)
	
	// Assert
	assert.NoError(t, err)
	
	// Verify StatefulSet was updated with new annotation
	updatedSts := &appsv1.StatefulSet{}
	err = c.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, updatedSts)
	assert.NoError(t, err)
	assert.Contains(t, updatedSts.Spec.Template.Annotations, "tempo.observability.io/sampling-hash")
}

func TestTempoManager_generateTempoConfig(t *testing.T) {
	tests := []struct {
		name        string
		platform    *observabilityv1beta1.ObservabilityPlatform
		tempoSpec   *observabilityv1beta1.TempoSpec
		configCheck func(t *testing.T, config string)
	}{
		{
			name: "local storage configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-platform",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Global: observabilityv1beta1.GlobalConfig{
						ExternalLabels: map[string]string{
							"cluster": "test",
							"env":     "dev",
						},
					},
				},
			},
			tempoSpec: &observabilityv1beta1.TempoSpec{
				Retention: "168h",
			},
			configCheck: func(t *testing.T, config string) {
				assert.Contains(t, config, "backend: local")
				assert.Contains(t, config, "path: /var/tempo/traces")
				assert.Contains(t, config, "cluster: test")
				assert.Contains(t, config, "env: dev")
				assert.Contains(t, config, "max_search_duration: 168h")
			},
		},
		{
			name: "S3 storage configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-platform",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Global: observabilityv1beta1.GlobalConfig{
						ExternalLabels: map[string]string{
							"cluster": "prod",
						},
					},
				},
			},
			tempoSpec: &observabilityv1beta1.TempoSpec{
				Retention: "336h",
				S3: &observabilityv1beta1.S3Config{
					BucketName:      "tempo-traces",
					Region:          "us-east-1",
					Endpoint:        "s3.amazonaws.com",
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
			},
			configCheck: func(t *testing.T, config string) {
				assert.Contains(t, config, "backend: s3")
				assert.Contains(t, config, "bucket: tempo-traces")
				assert.Contains(t, config, "region: us-east-1")
				assert.Contains(t, config, "endpoint: s3.amazonaws.com")
				assert.Contains(t, config, "access_key: AKIAIOSFODNN7EXAMPLE")
				assert.Contains(t, config, "secret_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
				assert.Contains(t, config, "max_search_duration: 336h")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &TempoManager{}
			config := manager.generateTempoConfig(tt.platform, tt.tempoSpec)
			
			// Check common configuration
			assert.Contains(t, config, "http_listen_port: 3200")
			assert.Contains(t, config, "grpc_listen_port: 9095")
			assert.Contains(t, config, "otlp:")
			assert.Contains(t, config, "jaeger:")
			assert.Contains(t, config, "zipkin:")
			
			// Run specific checks
			tt.configCheck(t, config)
		})
	}
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrInt32(i int32) *int32 {
	return &i
}
