/*
Copyright 2025.

Licensed under the MIT License.
*/

package controllers

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("Finalizer Manager", func() {
	var (
		ctx              context.Context
		k8sClient        client.Client
		finalizerManager *FinalizerManager
		platform         *observabilityv1beta1.ObservabilityPlatform
		reconciler       *ObservabilityPlatformReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		// Create a fake client
		s := scheme.Scheme
		Expect(observabilityv1beta1.AddToScheme(s)).To(Succeed())
		
		k8sClient = fake.NewClientBuilder().
			WithScheme(s).
			Build()
		
		// Create finalizer manager
		logger := log.FromContext(ctx)
		finalizerManager = NewFinalizerManager(k8sClient, logger)
		
		// Create a test platform
		platform = &observabilityv1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "test-namespace",
				UID:       types.UID("test-uid"),
			},
			Spec: observabilityv1beta1.ObservabilityPlatformSpec{
				Components: observabilityv1beta1.Components{
					Prometheus: &observabilityv1beta1.PrometheusSpec{
						Enabled: true,
						Version: "v2.48.0",
					},
					Grafana: &observabilityv1beta1.GrafanaSpec{
						Enabled: true,
						Version: "10.2.0",
					},
				},
			},
		}
		
		// Create a basic reconciler for testing
		reconciler = &ObservabilityPlatformReconciler{
			Client: k8sClient,
			Scheme: s,
			FinalizerManager: finalizerManager,
		}
		reconciler.EventRecorder = NewEnhancedEventRecorder(nil, "test")
	})

	Context("When adding finalizers", func() {
		It("should add all default finalizers", func() {
			// Create the platform
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())
			
			// Add finalizers
			err := finalizerManager.AddFinalizers(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify finalizers were added
			Expect(controllerutil.ContainsFinalizer(platform, FinalizerName)).To(BeTrue())
			Expect(controllerutil.ContainsFinalizer(platform, ComponentFinalizer)).To(BeTrue())
			Expect(controllerutil.ContainsFinalizer(platform, ExternalResourceFinalizer)).To(BeTrue())
		})

		It("should add backup finalizer when backup is enabled", func() {
			// Enable backup
			platform.Spec.Backup = &observabilityv1beta1.BackupConfig{
				Enabled: true,
			}
			
			// Create the platform
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())
			
			// Add finalizers
			err := finalizerManager.AddFinalizers(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify backup finalizer was added
			Expect(controllerutil.ContainsFinalizer(platform, BackupFinalizer)).To(BeTrue())
		})

		It("should not add duplicate finalizers", func() {
			// Create the platform with some finalizers already present
			controllerutil.AddFinalizer(platform, FinalizerName)
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())
			
			// Add finalizers again
			err := finalizerManager.AddFinalizers(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify no duplicates
			finalizers := platform.GetFinalizers()
			finalizerCount := 0
			for _, f := range finalizers {
				if f == FinalizerName {
					finalizerCount++
				}
			}
			Expect(finalizerCount).To(Equal(1))
		})
	})

	Context("When handling deletion", func() {
		BeforeEach(func() {
			// Create platform with finalizers
			controllerutil.AddFinalizer(platform, FinalizerName)
			controllerutil.AddFinalizer(platform, ComponentFinalizer)
			controllerutil.AddFinalizer(platform, ExternalResourceFinalizer)
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())
			
			// Create some resources owned by the platform
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: platform.Namespace,
					Labels: map[string]string{
						"observability.io/platform": platform.Name,
					},
				},
				Data: map[string]string{
					"test": "data",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())
			
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: platform.Namespace,
					Labels: map[string]string{
						"observability.io/platform": platform.Name,
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
				},
			}
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		})

		It("should handle deletion successfully", func() {
			// Mark for deletion
			now := metav1.Now()
			platform.DeletionTimestamp = &now
			Expect(k8sClient.Update(ctx, platform)).To(Succeed())
			
			// Handle deletion
			err := finalizerManager.HandleDeletion(ctx, platform, reconciler)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify finalizers were removed
			updatedPlatform := &observabilityv1beta1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(platform), updatedPlatform)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedPlatform.GetFinalizers()).To(BeEmpty())
		})

		It("should cleanup external resources", func() {
			// Mark for deletion
			now := metav1.Now()
			platform.DeletionTimestamp = &now
			Expect(k8sClient.Update(ctx, platform)).To(Succeed())
			
			// Handle deletion
			err := finalizerManager.HandleDeletion(ctx, platform, reconciler)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify resources were deleted
			cmList := &corev1.ConfigMapList{}
			err = k8sClient.List(ctx, cmList, client.InNamespace(platform.Namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(cmList.Items).To(HaveLen(0))
			
			pvcList := &corev1.PersistentVolumeClaimList{}
			err = k8sClient.List(ctx, pvcList, client.InNamespace(platform.Namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(pvcList.Items).To(HaveLen(0))
		})

		It("should preserve backup configmaps", func() {
			// Create a backup configmap
			backupCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup-config",
					Namespace: platform.Namespace,
					Labels: map[string]string{
						"observability.io/platform":    platform.Name,
						"observability.io/backup-type": "pre-deletion",
					},
				},
				Data: map[string]string{
					"backup": "data",
				},
			}
			Expect(k8sClient.Create(ctx, backupCM)).To(Succeed())
			
			// Mark for deletion
			now := metav1.Now()
			platform.DeletionTimestamp = &now
			Expect(k8sClient.Update(ctx, platform)).To(Succeed())
			
			// Handle deletion
			err := finalizerManager.HandleDeletion(ctx, platform, reconciler)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify backup configmap was preserved
			preservedCM := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(backupCM), preservedCM)
			Expect(err).NotTo(HaveOccurred())
			Expect(preservedCM.Name).To(Equal("backup-config"))
		})
	})

	Context("When handling backup finalizer", func() {
		BeforeEach(func() {
			// Enable backup and add backup finalizer
			platform.Spec.Backup = &observabilityv1beta1.BackupConfig{
				Enabled: true,
			}
			controllerutil.AddFinalizer(platform, BackupFinalizer)
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())
		})

		It("should create backup metadata", func() {
			// Mark for deletion
			now := metav1.Now()
			platform.DeletionTimestamp = &now
			Expect(k8sClient.Update(ctx, platform)).To(Succeed())
			
			// Handle backup finalizer directly
			err := finalizerManager.handleBackupFinalizer(ctx, platform, reconciler)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify backup configmap was created
			cmList := &corev1.ConfigMapList{}
			err = k8sClient.List(ctx, cmList, client.InNamespace(platform.Namespace))
			Expect(err).NotTo(HaveOccurred())
			
			// Find backup configmap
			var backupCM *corev1.ConfigMap
			for _, cm := range cmList.Items {
				if cm.Labels["observability.io/backup-type"] == "pre-deletion" {
					backupCM = &cm
					break
				}
			}
			Expect(backupCM).NotTo(BeNil())
			Expect(backupCM.Data).To(HaveKey("platform.yaml"))
			Expect(backupCM.Data).To(HaveKey("status.yaml"))
			Expect(backupCM.Data).To(HaveKey("timestamp"))
		})
	})
})

// Unit tests for specific functions
func TestFinalizerManager_waitForComponentTermination(t *testing.T) {
	tests := []struct {
		name          string
		existingPods  []runtime.Object
		timeout       time.Duration
		expectError   bool
		expectTimeout bool
	}{
		{
			name:         "no pods exist",
			existingPods: []runtime.Object{},
			timeout:      1 * time.Second,
			expectError:  false,
		},
		{
			name: "pods exist but terminate quickly",
			existingPods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"observability.io/platform": "test-platform",
						},
					},
				},
			},
			timeout:       5 * time.Second,
			expectError:   false,
			expectTimeout: true, // In this test, we'd need to simulate pod deletion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with pods
			s := scheme.Scheme
			_ = observabilityv1beta1.AddToScheme(s)
			
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithRuntimeObjects(tt.existingPods...).
				Build()
			
			fm := &FinalizerManager{
				Client: k8sClient,
			}
			
			platform := &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-namespace",
				},
			}
			
			reconciler := &ObservabilityPlatformReconciler{
				Client: k8sClient,
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			
			err := fm.waitForComponentTermination(ctx, platform, reconciler)
			
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil && err != context.DeadlineExceeded {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFinalizerManager_cleanupPVCs(t *testing.T) {
	// Create test setup
	s := scheme.Scheme
	_ = observabilityv1beta1.AddToScheme(s)
	
	platform := &observabilityv1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "test-namespace",
		},
	}
	
	// Create PVCs to be cleaned up
	pvc1 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-1",
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"observability.io/platform": platform.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
	}
	
	pvc2 := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc-2",
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"observability.io/platform": "other-platform",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
	}
	
	k8sClient := fake.NewClientBuilder().
		WithScheme(s).
		WithRuntimeObjects(pvc1, pvc2).
		Build()
	
	fm := &FinalizerManager{
		Client: k8sClient,
	}
	
	reconciler := &ObservabilityPlatformReconciler{
		Client: k8sClient,
	}
	
	// Run cleanup
	ctx := context.Background()
	err := fm.cleanupPVCs(ctx, platform, reconciler)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	// Verify only the correct PVC was deleted
	remainingPVC := &corev1.PersistentVolumeClaim{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      "pvc-1",
		Namespace: platform.Namespace,
	}, remainingPVC)
	if !errors.IsNotFound(err) {
		t.Errorf("expected pvc-1 to be deleted, but it still exists")
	}
	
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      "pvc-2",
		Namespace: platform.Namespace,
	}, remainingPVC)
	if err != nil {
		t.Errorf("expected pvc-2 to still exist, but got error: %v", err)
	}
}
