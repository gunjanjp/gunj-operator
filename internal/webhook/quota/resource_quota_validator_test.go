/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package quota

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

func TestResourceQuotaValidator_ValidateResourceQuota(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = observabilityv1beta1.AddToScheme(scheme)

	tests := []struct {
		name             string
		platform         *observabilityv1beta1.ObservabilityPlatform
		existingObjects  []runtime.Object
		clusterLimits    *ClusterResourceLimits
		wantErrors       int
		errorContains    []string
	}{
		{
			name: "within quota limits",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
							},
							Storage: &observabilityv1beta1.StorageSpec{
								Size: "5Gi",
							},
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
				&corev1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-quota",
						Namespace: "test-namespace",
					},
					Status: corev1.ResourceQuotaStatus{
						Hard: corev1.ResourceList{
							corev1.ResourceRequestsCPU:    resource.MustParse("1"),
							corev1.ResourceRequestsMemory: resource.MustParse("1Gi"),
							corev1.ResourceRequestsStorage: resource.MustParse("10Gi"),
							corev1.ResourcePods:           resource.MustParse("10"),
							corev1.ResourcePersistentVolumeClaims: resource.MustParse("5"),
						},
						Used: corev1.ResourceList{
							corev1.ResourceRequestsCPU:    resource.MustParse("200m"),
							corev1.ResourceRequestsMemory: resource.MustParse("512Mi"),
							corev1.ResourceRequestsStorage: resource.MustParse("2Gi"),
							corev1.ResourcePods:           resource.MustParse("2"),
							corev1.ResourcePersistentVolumeClaims: resource.MustParse("1"),
						},
					},
				},
			},
			wantErrors: 0,
		},
		{
			name: "exceeds CPU quota",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 3,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "500m",
									Memory: "256Mi",
								},
							},
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
				&corev1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-quota",
						Namespace: "test-namespace",
					},
					Status: corev1.ResourceQuotaStatus{
						Hard: corev1.ResourceList{
							corev1.ResourceRequestsCPU: resource.MustParse("1"),
						},
						Used: corev1.ResourceList{
							corev1.ResourceRequestsCPU: resource.MustParse("200m"),
						},
					},
				},
			},
			wantErrors:    1,
			errorContains: []string{"exceeds available CPU quota"},
		},
		{
			name: "exceeds memory quota",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 2,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "1Gi",
								},
							},
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
				&corev1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-quota",
						Namespace: "test-namespace",
					},
					Status: corev1.ResourceQuotaStatus{
						Hard: corev1.ResourceList{
							corev1.ResourceRequestsMemory: resource.MustParse("2Gi"),
						},
						Used: corev1.ResourceList{
							corev1.ResourceRequestsMemory: resource.MustParse("500Mi"),
						},
					},
				},
			},
			wantErrors:    1,
			errorContains: []string{"exceeds available memory quota"},
		},
		{
			name: "exceeds PVC quota",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 3,
							Storage: &observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled: true,
							Persistence: &observabilityv1beta1.PersistenceSpec{
								Enabled: true,
							},
						},
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled: true,
							Storage: &observabilityv1beta1.LokiStorageSpec{
								Size: "10Gi",
							},
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
				&corev1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-quota",
						Namespace: "test-namespace",
					},
					Status: corev1.ResourceQuotaStatus{
						Hard: corev1.ResourceList{
							corev1.ResourcePersistentVolumeClaims: resource.MustParse("3"),
						},
						Used: corev1.ResourceList{
							corev1.ResourcePersistentVolumeClaims: resource.MustParse("1"),
						},
					},
				},
			},
			wantErrors:    1,
			errorContains: []string{"exceeds available PVC quota"},
		},
		{
			name: "exceeds cluster CPU limit",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "900m",
									Memory: "256Mi",
								},
							},
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
			},
			clusterLimits: &ClusterResourceLimits{
				MaxCPUPerNamespace: resource.MustParse("1"),
			},
			wantErrors:    1,
			errorContains: []string{"exceeds cluster CPU limit"},
		},
		{
			name: "multi-tenant validation - exceeds tenant limit",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "tenant-a-prod",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tenant-a-prod",
						Labels: map[string]string{
							"tenant": "tenant-a",
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tenant-a-dev",
						Labels: map[string]string{
							"tenant": "tenant-a",
						},
					},
				},
				// Create 5 existing platforms for tenant-a
				createPlatform("platform-1", "tenant-a-prod"),
				createPlatform("platform-2", "tenant-a-prod"),
				createPlatform("platform-3", "tenant-a-dev"),
				createPlatform("platform-4", "tenant-a-dev"),
				createPlatform("platform-5", "tenant-a-dev"),
			},
			wantErrors:    1,
			errorContains: []string{"tenant tenant-a has reached the maximum number of platforms"},
		},
		{
			name: "multi-tenant validation - requires isolation",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "isolated-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "isolated-namespace",
						Labels: map[string]string{
							"tenant":            "tenant-b",
							"require-isolation": "true",
						},
					},
				},
			},
			wantErrors:    1,
			errorContains: []string{"must have nodeSelector with tenant=tenant-b"},
		},
		{
			name: "no quota in namespace - should pass",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "no-quota-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 3,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "1",
									Memory: "4Gi",
								},
							},
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "no-quota-namespace",
					},
				},
			},
			wantErrors: 0,
		},
		{
			name: "all components enabled with resources",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "full-platform",
					Namespace: "test-namespace",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 2,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
							},
							Storage: &observabilityv1beta1.StorageSpec{
								Size: "5Gi",
							},
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "50m",
									Memory: "128Mi",
								},
							},
						},
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
							},
							Storage: &observabilityv1beta1.LokiStorageSpec{
								Size: "10Gi",
							},
						},
						Tempo: &observabilityv1beta1.TempoSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "50m",
									Memory: "128Mi",
								},
							},
							Storage: &observabilityv1beta1.StorageSpec{
								Size: "5Gi",
							},
						},
					},
					Alerting: &observabilityv1beta1.AlertingSpec{
						Alertmanager: &observabilityv1beta1.AlertmanagerSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "10m",
									Memory: "32Mi",
								},
							},
							Storage: &observabilityv1beta1.StorageSpec{
								Size: "1Gi",
							},
						},
					},
				},
			},
			existingObjects: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				},
				&corev1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-quota",
						Namespace: "test-namespace",
					},
					Status: corev1.ResourceQuotaStatus{
						Hard: corev1.ResourceList{
							corev1.ResourceRequestsCPU:    resource.MustParse("2"),
							corev1.ResourceRequestsMemory: resource.MustParse("4Gi"),
							corev1.ResourceRequestsStorage: resource.MustParse("50Gi"),
							corev1.ResourcePods:           resource.MustParse("20"),
							corev1.ResourcePersistentVolumeClaims: resource.MustParse("10"),
						},
					},
				},
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with existing objects
			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.existingObjects...).
				WithStatusSubresource(&corev1.ResourceQuota{}).
				Build()

			validator := &ResourceQuotaValidator{
				Client:                cl,
				ClusterResourceLimits: tt.clusterLimits,
			}

			errs := validator.ValidateResourceQuota(context.Background(), tt.platform)

			if len(errs) != tt.wantErrors {
				t.Errorf("ValidateResourceQuota() returned %d errors, want %d", len(errs), tt.wantErrors)
				for _, err := range errs {
					t.Logf("Error: %v", err)
				}
			}

			// Check error messages contain expected strings
			for _, expectedError := range tt.errorContains {
				found := false
				for _, err := range errs {
					if contains(err.Error(), expectedError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s' not found", expectedError)
				}
			}
		})
	}
}

func TestResourceQuotaValidator_calculatePlatformResources(t *testing.T) {
	validator := &ResourceQuotaValidator{}

	tests := []struct {
		name         string
		platform     *observabilityv1beta1.ObservabilityPlatform
		wantCPU      string
		wantMemory   string
		wantStorage  string
		wantPods     int
		wantPVCs     int
	}{
		{
			name: "single component",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 2,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
							},
							Storage: &observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
						},
					},
				},
			},
			wantCPU:     "200m",
			wantMemory:  "512Mi",
			wantStorage: "10Gi",
			wantPods:    2,
			wantPVCs:    2,
		},
		{
			name: "multiple components",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: &observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled:  true,
							Replicas: 2,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
							},
							Storage: &observabilityv1beta1.StorageSpec{
								Size: "10Gi",
							},
						},
						Grafana: &observabilityv1beta1.GrafanaSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "50m",
									Memory: "128Mi",
								},
							},
							Persistence: &observabilityv1beta1.PersistenceSpec{
								Enabled: true,
								Size:    "1Gi",
							},
						},
						Loki: &observabilityv1beta1.LokiSpec{
							Enabled:  true,
							Replicas: 1,
							Resources: &observabilityv1beta1.ResourceRequirements{
								Requests: &observabilityv1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
							},
							Storage: &observabilityv1beta1.LokiStorageSpec{
								Size: "20Gi",
								S3: &observabilityv1beta1.S3StorageSpec{
									Enabled: true, // Using S3, so no PVC
								},
							},
						},
					},
				},
			},
			wantCPU:     "350m",
			wantMemory:  "896Mi",
			wantStorage: "31Gi",
			wantPods:    4,
			wantPVCs:    3, // Prometheus: 2, Grafana: 1, Loki: 0 (using S3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := validator.calculatePlatformResources(tt.platform)
			if err != nil {
				t.Fatalf("calculatePlatformResources() error = %v", err)
			}

			if resources.CPU.String() != tt.wantCPU {
				t.Errorf("CPU = %v, want %v", resources.CPU.String(), tt.wantCPU)
			}
			if resources.Memory.String() != tt.wantMemory {
				t.Errorf("Memory = %v, want %v", resources.Memory.String(), tt.wantMemory)
			}
			if resources.Storage.String() != tt.wantStorage {
				t.Errorf("Storage = %v, want %v", resources.Storage.String(), tt.wantStorage)
			}
			if resources.Pods != tt.wantPods {
				t.Errorf("Pods = %v, want %v", resources.Pods, tt.wantPods)
			}
			if resources.PVCs != tt.wantPVCs {
				t.Errorf("PVCs = %v, want %v", resources.PVCs, tt.wantPVCs)
			}
		})
	}
}

func TestResourceQuotaValidator_GetQuotaSummary(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	namespace := "test-namespace"
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-quota",
			Namespace: namespace,
		},
		Status: corev1.ResourceQuotaStatus{
			Hard: corev1.ResourceList{
				corev1.ResourceRequestsCPU:     resource.MustParse("2"),
				corev1.ResourceRequestsMemory:  resource.MustParse("4Gi"),
				corev1.ResourceRequestsStorage: resource.MustParse("100Gi"),
				corev1.ResourcePods:            resource.MustParse("20"),
				corev1.ResourcePersistentVolumeClaims: resource.MustParse("10"),
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-container",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(quota, pod, pvc).
		Build()

	validator := &ResourceQuotaValidator{
		Client: cl,
	}

	summary, err := validator.GetQuotaSummary(context.Background(), namespace)
	if err != nil {
		t.Fatalf("GetQuotaSummary() error = %v", err)
	}

	// Check that summary contains expected information
	expectedStrings := []string{
		"Resource Quota Summary for namespace 'test-namespace'",
		"Quota: test-quota",
		"CPU: 500m / 2 (25.0% used)",
		"Memory: 1Gi / 4Gi (25.0% used)",
		"Storage: 10Gi / 100Gi (10.0% used)",
		"Pods: 1 / 20 (5.0% used)",
		"PVCs: 1 / 10 (10.0% used)",
	}

	for _, expected := range expectedStrings {
		if !contains(summary, expected) {
			t.Errorf("Expected summary to contain '%s'", expected)
		}
	}
}

// Helper functions

func createPlatform(name, namespace string) *observabilityv1beta1.ObservabilityPlatform {
	return &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: observabilityv1beta1.ObservabilityPlatformSpec{
			Components: &observabilityv1beta1.Components{
				Prometheus: &observabilityv1beta1.PrometheusSpec{
					Enabled: true,
				},
			},
		},
	}
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr) && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
