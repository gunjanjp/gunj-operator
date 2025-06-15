/*
Copyright 2025.

Licensed under the MIT License.
*/

package loki

import (
	"context"
	"testing"

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

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestNewLokiManager(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	
	manager := NewLokiManager(fakeClient, scheme)
	assert.NotNil(t, manager)
}

func TestLokiManager_Reconcile(t *testing.T) {
	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErr  bool
		validate func(t *testing.T, client client.Client)
	}{
		{
			name: "reconcile with loki disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled: false,
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, client client.Client) {
				// Should not create any resources
				sts := &appsv1.StatefulSet{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform",
					Namespace: "default",
				}, sts)
				assert.True(t, err != nil)
			},
		},
		{
			name: "reconcile with basic loki configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Version:  "2.9.0",
							Replicas: 1,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							Storage: observabilityv1beta1.StorageSpec{
								Size: resource.MustParse("10Gi"),
							},
						},
					},
					Global: observabilityv1beta1.GlobalConfig{
						LogLevel: "info",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, client client.Client) {
				// Check ConfigMap
				cm := &corev1.ConfigMap{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform-config",
					Namespace: "default",
				}, cm)
				require.NoError(t, err)
				assert.Contains(t, cm.Data, "loki.yaml")
				assert.Contains(t, cm.Data["loki.yaml"], "auth_enabled: false")
				
				// Check Service
				svc := &corev1.Service{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform",
					Namespace: "default",
				}, svc)
				require.NoError(t, err)
				assert.Equal(t, int32(3100), svc.Spec.Ports[0].Port)
				
				// Check Headless Service
				headlessSvc := &corev1.Service{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform-headless",
					Namespace: "default",
				}, headlessSvc)
				require.NoError(t, err)
				assert.Equal(t, "None", headlessSvc.Spec.ClusterIP)
				
				// Check StatefulSet
				sts := &appsv1.StatefulSet{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform",
					Namespace: "default",
				}, sts)
				require.NoError(t, err)
				assert.Equal(t, int32(1), *sts.Spec.Replicas)
				assert.Equal(t, "grafana/loki:2.9.0", sts.Spec.Template.Spec.Containers[0].Image)
			},
		},
		{
			name: "reconcile with S3 storage",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Version:  "2.9.0",
							Replicas: 3,
							Storage: observabilityv1beta1.StorageSpec{
								Size: resource.MustParse("10Gi"),
							},
							S3: &observabilityv1beta1.S3Config{
								Enabled:         true,
								BucketName:      "loki-logs",
								Region:          "us-east-1",
								AccessKeyID:     "test-key",
								SecretAccessKey: "test-secret",
							},
							Retention: "720h",
						},
					},
					Global: observabilityv1beta1.GlobalConfig{
						LogLevel: "debug",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, client client.Client) {
				// Check S3 Secret
				secret := &corev1.Secret{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform-s3",
					Namespace: "default",
				}, secret)
				require.NoError(t, err)
				assert.Equal(t, "test-key", string(secret.Data["access_key_id"]))
				assert.Equal(t, "test-secret", string(secret.Data["secret_access_key"]))
				
				// Check ConfigMap contains S3 config
				cm := &corev1.ConfigMap{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform-config",
					Namespace: "default",
				}, cm)
				require.NoError(t, err)
				assert.Contains(t, cm.Data["loki.yaml"], "s3:")
				assert.Contains(t, cm.Data["loki.yaml"], "bucketnames: loki-logs")
				assert.Contains(t, cm.Data["loki.yaml"], "retention_period: 720h")
				
				// Check StatefulSet has S3 env vars
				sts := &appsv1.StatefulSet{}
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform",
					Namespace: "default",
				}, sts)
				require.NoError(t, err)
				assert.Equal(t, int32(3), *sts.Spec.Replicas)
				
				// Check for AWS env vars
				container := sts.Spec.Template.Spec.Containers[0]
				hasAWSKey := false
				hasAWSSecret := false
				hasAWSRegion := false
				for _, env := range container.Env {
					if env.Name == "AWS_ACCESS_KEY_ID" {
						hasAWSKey = true
					}
					if env.Name == "AWS_SECRET_ACCESS_KEY" {
						hasAWSSecret = true
					}
					if env.Name == "AWS_REGION" {
						hasAWSRegion = true
						assert.Equal(t, "us-east-1", env.Value)
					}
				}
				assert.True(t, hasAWSKey)
				assert.True(t, hasAWSSecret)
				assert.True(t, hasAWSRegion)
			},
		},
		{
			name: "reconcile with compactor enabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:          true,
							Version:          "2.9.0",
							Replicas:         1,
							CompactorEnabled: true,
							Storage: observabilityv1beta1.StorageSpec{
								Size: resource.MustParse("10Gi"),
							},
						},
					},
					Global: observabilityv1beta1.GlobalConfig{
						LogLevel: "info",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, client client.Client) {
				// Check Compactor Deployment
				deployment := &appsv1.Deployment{}
				err := client.Get(context.Background(), types.NamespacedName{
					Name:      "loki-test-platform-compactor",
					Namespace: "default",
				}, deployment)
				require.NoError(t, err)
				assert.Equal(t, int32(1), *deployment.Spec.Replicas)
				assert.Equal(t, "grafana/loki:2.9.0", deployment.Spec.Template.Spec.Containers[0].Image)
				assert.Contains(t, deployment.Spec.Template.Spec.Containers[0].Args, "-target=compactor")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = observabilityv1beta1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.platform).
				Build()

			manager := &LokiManager{
				Client: fakeClient,
				Scheme: scheme,
			}

			err := manager.Reconcile(context.Background(), tt.platform)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, fakeClient)
			}
		})
	}
}

func TestLokiManager_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		platform       *observabilityv1beta1.ObservabilityPlatform
		existingObjs   []client.Object
		expectedPhase  string
		expectedMsg    string
	}{
		{
			name: "loki disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled: false,
						},
					},
				},
			},
			expectedPhase: "Disabled",
			expectedMsg:   "Loki is disabled",
		},
		{
			name: "statefulset not found",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled: true,
							Version: "2.9.0",
						},
					},
				},
			},
			expectedPhase: "Failed",
			expectedMsg:   "Failed to get StatefulSet",
		},
		{
			name: "all replicas ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Version:  "2.9.0",
							Replicas: 3,
						},
					},
				},
			},
			existingObjs: []client.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "loki-test-platform",
						Namespace: "default",
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
			expectedMsg:   "All 3 replicas are ready",
		},
		{
			name: "partially ready",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Version:  "2.9.0",
							Replicas: 3,
						},
					},
				},
			},
			existingObjs: []client.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "loki-test-platform",
						Namespace: "default",
					},
					Spec: appsv1.StatefulSetSpec{
						Replicas: &[]int32{3}[0],
					},
					Status: appsv1.StatefulSetStatus{
						Replicas:      3,
						ReadyReplicas: 2,
					},
				},
			},
			expectedPhase: "Degraded",
			expectedMsg:   "2/3 replicas are ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = observabilityv1beta1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)

			objs := append([]client.Object{tt.platform}, tt.existingObjs...)
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			manager := &LokiManager{
				Client: fakeClient,
				Scheme: scheme,
			}

			status, err := manager.GetStatus(context.Background(), tt.platform)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPhase, status.Phase)
			assert.Contains(t, status.Message, tt.expectedMsg)
		})
	}
}

func TestLokiManager_Validate(t *testing.T) {
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
						Loki: &observabilityv1beta1.LokiSpec{
							Version:   "2.9.0",
							Retention: "168h",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid version format",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Version: "invalid",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "version must be in format",
		},
		{
			name: "invalid retention format",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Version:   "2.9.0",
							Retention: "30days",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid retention format",
		},
		{
			name: "S3 missing bucket name",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Version: "2.9.0",
							S3: &observabilityv1beta1.S3Config{
								Enabled: true,
								Region:  "us-east-1",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "S3 bucket name is required",
		},
		{
			name: "S3 missing region",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Loki: &observabilityv1beta1.LokiSpec{
							Version: "2.9.0",
							S3: &observabilityv1beta1.S3Config{
								Enabled:    true,
								BucketName: "loki-logs",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "S3 region is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &LokiManager{}
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

func TestLokiManager_GetServiceURL(t *testing.T) {
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "monitoring",
		},
	}

	manager := &LokiManager{}
	url := manager.GetServiceURL(platform)
	assert.Equal(t, "http://loki-test-platform.monitoring.svc.cluster.local:3100", url)
}

func TestLokiManager_ConfigureStorage(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Loki: &observabilityv1beta1.LokiSpec{
					Enabled: true,
					Version: "2.9.0",
					S3: &observabilityv1beta1.S3Config{
						Enabled:         true,
						BucketName:      "new-bucket",
						Region:          "eu-west-1",
						AccessKeyID:     "new-key",
						SecretAccessKey: "new-secret",
					},
				},
			},
			Global: observabilityv1beta1.GlobalConfig{
				LogLevel: "info",
			},
		},
	}

	existingSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki-test-platform",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &[]int32{1}[0],
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, existingSts).
		Build()

	manager := &LokiManager{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := manager.ConfigureStorage(context.Background(), platform)
	require.NoError(t, err)

	// Verify S3 secret was updated
	secret := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "loki-test-platform-s3",
		Namespace: "default",
	}, secret)
	require.NoError(t, err)
	assert.Equal(t, "new-key", string(secret.Data["access_key_id"]))

	// Verify StatefulSet was updated with restart annotation
	updatedSts := &appsv1.StatefulSet{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "loki-test-platform",
		Namespace: "default",
	}, updatedSts)
	require.NoError(t, err)
	assert.Contains(t, updatedSts.Spec.Template.Annotations, "observability.io/storage-update")
}

func TestLokiManager_UpdateRetention(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "default",
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: observabilityv1beta1.Components{
				Loki: &observabilityv1beta1.LokiSpec{
					Enabled:   true,
					Version:   "2.9.0",
					Retention: "720h", // 30 days
				},
			},
			Global: observabilityv1beta1.GlobalConfig{
				LogLevel: "info",
			},
		},
	}

	existingSts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki-test-platform",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &[]int32{1}[0],
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(platform, existingSts).
		Build()

	manager := &LokiManager{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := manager.UpdateRetention(context.Background(), platform)
	require.NoError(t, err)

	// Verify ConfigMap was updated with new retention
	cm := &corev1.ConfigMap{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "loki-test-platform-config",
		Namespace: "default",
	}, cm)
	require.NoError(t, err)
	assert.Contains(t, cm.Data["loki.yaml"], "retention_period: 720h")

	// Verify StatefulSet was updated with restart annotation
	updatedSts := &appsv1.StatefulSet{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "loki-test-platform",
		Namespace: "default",
	}, updatedSts)
	require.NoError(t, err)
	assert.Contains(t, updatedSts.Spec.Template.Annotations, "observability.io/retention-update")
}
