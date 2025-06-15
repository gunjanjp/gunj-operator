/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package migration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	observabilityv1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/migration"
)

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migration Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	
	// Register schemes
	Expect(observabilityv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(observabilityv1beta1.AddToScheme(scheme.Scheme)).To(Succeed())
})

var _ = Describe("Migration Manager", func() {
	var (
		ctx              context.Context
		k8sClient        client.Client
		migrationManager *migration.MigrationManager
		testScheme       *runtime.Scheme
	)
	
	BeforeEach(func() {
		ctx = context.Background()
		
		// Create scheme
		testScheme = runtime.NewScheme()
		Expect(observabilityv1alpha1.AddToScheme(testScheme)).To(Succeed())
		Expect(observabilityv1beta1.AddToScheme(testScheme)).To(Succeed())
		
		// Create fake client
		k8sClient = fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		
		// Create migration manager
		config := migration.MigrationConfig{
			MaxConcurrentMigrations: 5,
			BatchSize:               10,
			RetryAttempts:           3,
			RetryInterval:           time.Second,
			EnableOptimizations:     true,
			DryRun:                  false,
			ProgressReportInterval:  time.Second,
		}
		
		logger := logf.Log.WithName("test")
		migrationManager = migration.NewMigrationManager(k8sClient, testScheme, logger, config)
	})
	
	Describe("Single Resource Migration", func() {
		It("should migrate v1alpha1 to v1beta1 successfully", func() {
			// Create v1alpha1 resource
			v1alpha1Platform := &observabilityv1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1alpha1.ObservabilityPlatformSpec{
					Monitoring: &observabilityv1alpha1.MonitoringSpec{
						Enabled: true,
					},
					Components: observabilityv1alpha1.Components{
						Prometheus: &observabilityv1alpha1.PrometheusSpec{
							Enabled: true,
							Version: "v2.45.0",
							Config:  "global:\n  scrape_interval: 15s",
						},
					},
				},
			}
			
			// Create resource in cluster
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).To(Succeed())
			
			// Migrate to v1beta1
			err := migrationManager.MigrateResource(ctx, 
				types.NamespacedName{Name: "test-platform", Namespace: "default"}, 
				"v1beta1")
			Expect(err).NotTo(HaveOccurred())
			
			// Verify migration
			v1beta1Platform := &observabilityv1beta1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-platform", Namespace: "default"}, v1beta1Platform)
			Expect(err).NotTo(HaveOccurred())
			
			// Check field transformation
			Expect(v1beta1Platform.Spec.Observability).NotTo(BeNil())
			Expect(v1beta1Platform.Spec.Observability.Enabled).To(BeTrue())
			
			// Check component transformation
			Expect(v1beta1Platform.Spec.Components.Prometheus).NotTo(BeNil())
			Expect(v1beta1Platform.Spec.Components.Prometheus.Version).To(Equal("v2.45.0"))
		})
		
		It("should handle migration failures gracefully", func() {
			// Create resource with invalid data
			resource := types.NamespacedName{Name: "non-existent", Namespace: "default"}
			
			// Attempt migration
			err := migrationManager.MigrateResource(ctx, resource, "v1beta1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get resource"))
		})
		
		It("should skip already migrated resources", func() {
			// Create v1beta1 resource
			v1beta1Platform := &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "already-migrated",
					Namespace: "default",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}
			
			Expect(k8sClient.Create(ctx, v1beta1Platform)).To(Succeed())
			
			// Attempt migration
			err := migrationManager.MigrateResource(ctx,
				types.NamespacedName{Name: "already-migrated", Namespace: "default"},
				"v1beta1")
			// Should succeed without errors
			Expect(err).NotTo(HaveOccurred())
		})
	})
	
	Describe("Batch Migration", func() {
		It("should migrate multiple resources in batch", func() {
			// Create multiple v1alpha1 resources
			resources := []types.NamespacedName{}
			for i := 0; i < 5; i++ {
				platform := &observabilityv1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("batch-platform-%d", i),
						Namespace: "default",
					},
					Spec: observabilityv1alpha1.ObservabilityPlatformSpec{
						Components: observabilityv1alpha1.Components{
							Prometheus: &observabilityv1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, platform)).To(Succeed())
				
				resources = append(resources, types.NamespacedName{
					Name:      platform.Name,
					Namespace: platform.Namespace,
				})
			}
			
			// Perform batch migration
			task, err := migrationManager.MigrateBatch(ctx, resources, "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			Expect(task).NotTo(BeNil())
			
			// Wait for migration to complete
			Eventually(func() migration.MigrationStatus {
				status, _ := migrationManager.GetMigrationStatus(task.ID)
				if status != nil {
					return status.Status
				}
				return ""
			}, 30*time.Second, 1*time.Second).Should(Equal(migration.MigrationStatusCompleted))
			
			// Verify all resources were migrated
			for _, resource := range resources {
				platform := &observabilityv1beta1.ObservabilityPlatform{}
				err := k8sClient.Get(ctx, resource, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(platform.APIVersion).To(Equal("observability.io/v1beta1"))
			}
		})
		
		It("should handle partial batch failures", func() {
			// Create mix of valid and invalid resources
			resources := []types.NamespacedName{
				{Name: "valid-platform", Namespace: "default"},
				{Name: "non-existent", Namespace: "default"},
				{Name: "another-valid", Namespace: "default"},
			}
			
			// Create only valid resources
			for _, name := range []string{"valid-platform", "another-valid"} {
				platform := &observabilityv1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: "default",
					},
					Spec: observabilityv1alpha1.ObservabilityPlatformSpec{
						Components: observabilityv1alpha1.Components{
							Prometheus: &observabilityv1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, platform)).To(Succeed())
			}
			
			// Perform batch migration
			task, err := migrationManager.MigrateBatch(ctx, resources, "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for completion
			Eventually(func() bool {
				status, _ := migrationManager.GetMigrationStatus(task.ID)
				return status != nil && status.Status != migration.MigrationStatusInProgress
			}, 30*time.Second, 1*time.Second).Should(BeTrue())
			
			// Check final status
			finalStatus, _ := migrationManager.GetMigrationStatus(task.ID)
			Expect(finalStatus.Progress.MigratedResources).To(Equal(2))
			Expect(finalStatus.Progress.FailedResources).To(BeNumerically(">=", 1))
		})
	})
	
	Describe("Migration Status Tracking", func() {
		It("should track migration progress", func() {
			// Create resources
			resources := []types.NamespacedName{}
			for i := 0; i < 3; i++ {
				platform := &observabilityv1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("progress-platform-%d", i),
						Namespace: "default",
					},
					Spec: observabilityv1alpha1.ObservabilityPlatformSpec{
						Components: observabilityv1alpha1.Components{
							Prometheus: &observabilityv1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, platform)).To(Succeed())
				resources = append(resources, types.NamespacedName{
					Name:      platform.Name,
					Namespace: platform.Namespace,
				})
			}
			
			// Start migration
			task, err := migrationManager.MigrateBatch(ctx, resources, "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			
			// Track progress
			progressUpdates := []migration.MigrationProgress{}
			Eventually(func() bool {
				status, err := migrationManager.GetMigrationStatus(task.ID)
				if err == nil && status != nil {
					progressUpdates = append(progressUpdates, status.Progress)
					return status.Status != migration.MigrationStatusInProgress
				}
				return false
			}, 30*time.Second, 500*time.Millisecond).Should(BeTrue())
			
			// Verify progress was tracked
			Expect(len(progressUpdates)).To(BeNumerically(">", 0))
			
			// Final progress should show all resources processed
			finalStatus, _ := migrationManager.GetMigrationStatus(task.ID)
			totalProcessed := finalStatus.Progress.MigratedResources + 
				finalStatus.Progress.FailedResources + 
				finalStatus.Progress.SkippedResources
			Expect(totalProcessed).To(Equal(3))
		})
	})
	
	Describe("Migration Cancellation", func() {
		It("should cancel an in-progress migration", func() {
			// Create many resources to ensure migration takes time
			resources := []types.NamespacedName{}
			for i := 0; i < 20; i++ {
				platform := &observabilityv1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("cancel-platform-%d", i),
						Namespace: "default",
					},
					Spec: observabilityv1alpha1.ObservabilityPlatformSpec{
						Components: observabilityv1alpha1.Components{
							Prometheus: &observabilityv1alpha1.PrometheusSpec{
								Enabled: true,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, platform)).To(Succeed())
				resources = append(resources, types.NamespacedName{
					Name:      platform.Name,
					Namespace: platform.Namespace,
				})
			}
			
			// Start migration
			task, err := migrationManager.MigrateBatch(ctx, resources, "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			
			// Wait a bit then cancel
			time.Sleep(100 * time.Millisecond)
			err = migrationManager.CancelMigration(task.ID)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify migration was cancelled
			Eventually(func() migration.MigrationStatus {
				status, _ := migrationManager.GetMigrationStatus(task.ID)
				if status != nil {
					return status.Status
				}
				return ""
			}, 10*time.Second, 1*time.Second).Should(Equal(migration.MigrationStatusRolledBack))
		})
	})
})
