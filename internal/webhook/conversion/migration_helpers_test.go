/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/migration"
)

var _ = Describe("Migration Helpers Tests", func() {
	var (
		migrationManager *migration.MigrationManager
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		migrationManager, err = migration.NewMigrationManager(
			k8sClient,
			runtime.NewScheme(),
			migration.MigrationConfig{
				BatchSize:            10,
				MaxConcurrency:       5,
				ProgressReportInterval: 1 * time.Second,
			},
		)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("MigrationManager", func() {
		It("should create migration plans correctly", func() {
			// Create test platforms
			platforms := []runtime.Object{
				&v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-platform-1",
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Version: "v2.48.0",
							},
						},
					},
				},
				&v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-platform-2",
						Namespace: "monitoring",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Grafana: &v1alpha1.GrafanaSpec{
								Enabled: true,
								Version: "10.0.0",
							},
						},
					},
				},
			}

			plan, err := migrationManager.PlanMigration(ctx, platforms, "v1alpha1", "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			Expect(plan).NotTo(BeNil())

			By("verifying migration plan details")
			Expect(plan.SourceVersion).To(Equal("v1alpha1"))
			Expect(plan.TargetVersion).To(Equal("v1beta1"))
			Expect(plan.TotalResources).To(Equal(2))
			Expect(plan.EstimatedDuration).To(BeNumerically(">", 0))
			Expect(plan.Phases).To(HaveLen(4)) // Pre-migration, Migration, Validation, Cleanup
		})

		It("should validate resources before migration", func() {
			invalidPlatform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-platform",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Version: "", // Invalid: empty version
						},
					},
				},
			}

			validPlatform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-platform",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
					},
				},
			}

			By("validating invalid resource")
			err := migrationManager.ValidateResource(ctx, invalidPlatform, "v1alpha1", "v1beta1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("version"))

			By("validating valid resource")
			err = migrationManager.ValidateResource(ctx, validPlatform, "v1alpha1", "v1beta1")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle dry-run migrations", func() {
			platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dry-run-platform",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			}

			result, err := migrationManager.DryRunMigration(ctx, []runtime.Object{platform}, "v1alpha1", "v1beta1")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			By("verifying dry-run results")
			Expect(result.Success).To(BeTrue())
			Expect(result.ProcessedCount).To(Equal(1))
			Expect(result.Errors).To(BeEmpty())
			Expect(result.Warnings).To(BeEmpty())
			Expect(result.ChangeSummary).To(HaveKey("test-conversion/dry-run-platform"))
		})
	})

	Context("SchemaEvolutionTracker", func() {
		var tracker *migration.SchemaEvolutionTracker

		BeforeEach(func() {
			var err error
			tracker, err = migration.NewSchemaEvolutionTracker()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should track schema changes between versions", func() {
			// Register v1alpha1 schema
			v1alpha1Schema := migration.VersionSchema{
				Version: "v1alpha1",
				Fields: map[string]migration.FieldInfo{
					"spec.components.prometheus.enabled": {
						Type:     "boolean",
						Required: false,
					},
					"spec.components.prometheus.version": {
						Type:     "string",
						Required: true,
					},
				},
			}
			err := tracker.RegisterVersion(v1alpha1Schema)
			Expect(err).NotTo(HaveOccurred())

			// Register v1beta1 schema with changes
			v1beta1Schema := migration.VersionSchema{
				Version: "v1beta1",
				Fields: map[string]migration.FieldInfo{
					"spec.components.prometheus.enabled": {
						Type:     "boolean",
						Required: false,
					},
					"spec.components.prometheus.version": {
						Type:     "string",
						Required: true,
					},
					"spec.components.prometheus.replicas": {
						Type:     "integer",
						Required: false,
						Default:  int32(1),
					},
					"spec.security.rbac.enabled": {
						Type:     "boolean",
						Required: false,
					},
				},
			}
			err = tracker.RegisterVersion(v1beta1Schema)
			Expect(err).NotTo(HaveOccurred())

			By("getting field changes")
			changes := tracker.GetFieldChanges("v1alpha1", "v1beta1")
			Expect(changes.AddedFields).To(ConsistOf(
				"spec.components.prometheus.replicas",
				"spec.security.rbac.enabled",
			))
			Expect(changes.RemovedFields).To(BeEmpty())
			Expect(changes.ModifiedFields).To(BeEmpty())

			By("checking field compatibility")
			compatible, reason := tracker.IsFieldCompatible(
				"spec.components.prometheus.version",
				"v1alpha1",
				"v1beta1",
			)
			Expect(compatible).To(BeTrue())
			Expect(reason).To(BeEmpty())
		})

		It("should generate migration paths", func() {
			// Register multiple versions
			versions := []migration.VersionSchema{
				{Version: "v1alpha1", Fields: map[string]migration.FieldInfo{}},
				{Version: "v1alpha2", Fields: map[string]migration.FieldInfo{}},
				{Version: "v1beta1", Fields: map[string]migration.FieldInfo{}},
				{Version: "v1beta2", Fields: map[string]migration.FieldInfo{}},
				{Version: "v1", Fields: map[string]migration.FieldInfo{}},
			}

			for _, v := range versions {
				err := tracker.RegisterVersion(v)
				Expect(err).NotTo(HaveOccurred())
			}

			By("finding migration path from v1alpha1 to v1")
			path, err := tracker.GetMigrationPath("v1alpha1", "v1")
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal([]string{"v1alpha1", "v1alpha2", "v1beta1", "v1beta2", "v1"}))

			By("finding migration path from v1beta1 to v1alpha2")
			path, err = tracker.GetMigrationPath("v1beta1", "v1alpha2")
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal([]string{"v1beta1", "v1alpha2"}))
		})
	})

	Context("ConversionOptimizer", func() {
		var optimizer *migration.ConversionOptimizer

		BeforeEach(func() {
			optimizer = migration.NewConversionOptimizer(migration.OptimizerConfig{
				EnableCaching:     true,
				EnableBatching:    true,
				CacheTTL:          5 * time.Minute,
				MaxBatchSize:      50,
			})
		})

		It("should optimize conversion strategies", func() {
			// Create test resources with different characteristics
			smallResource := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "small-platform",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}

			largeResource := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "large-platform",
					Labels: func() map[string]string {
						labels := make(map[string]string)
						for i := 0; i < 50; i++ {
							labels[fmt.Sprintf("label-%d", i)] = fmt.Sprintf("value-%d", i)
						}
						return labels
					}(),
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:     true,
							RemoteWrite: make([]v1alpha1.RemoteWriteSpec, 20),
						},
					},
				},
			}

			By("selecting strategy for small resource")
			strategy := optimizer.SelectStrategy(ctx, smallResource, "v1alpha1", "v1beta1")
			Expect(strategy).To(Equal(migration.StrategyDirect))

			By("selecting strategy for large resource")
			strategy = optimizer.SelectStrategy(ctx, largeResource, "v1alpha1", "v1beta1")
			Expect(strategy).To(Equal(migration.StrategyOptimized))

			By("analyzing conversion complexity")
			complexity := optimizer.AnalyzeComplexity(smallResource, largeResource)
			Expect(complexity.Score).To(BeNumerically(">", 5))
			Expect(complexity.Factors).To(HaveKey("field_count"))
			Expect(complexity.Factors).To(HaveKey("annotation_size"))
		})

		It("should cache conversion results", func() {
			resource := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cached-platform",
					Namespace: "default",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}

			// First conversion
			key := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
			result1, cached := optimizer.GetCachedResult(key, "v1alpha1", "v1beta1")
			Expect(cached).To(BeFalse())
			Expect(result1).To(BeNil())

			// Simulate conversion and cache result
			converted := &v1beta1.ObservabilityPlatform{
				ObjectMeta: resource.ObjectMeta,
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}
			optimizer.CacheResult(key, "v1alpha1", "v1beta1", converted)

			// Second conversion should use cache
			result2, cached := optimizer.GetCachedResult(key, "v1alpha1", "v1beta1")
			Expect(cached).To(BeTrue())
			Expect(result2).NotTo(BeNil())
		})
	})

	Context("BatchConversionProcessor", func() {
		var processor *migration.BatchConversionProcessor

		BeforeEach(func() {
			processor = migration.NewBatchConversionProcessor(migration.ProcessorConfig{
				BatchSize:      10,
				Concurrency:    3,
				RetryAttempts:  3,
				RetryDelay:     100 * time.Millisecond,
			})
		})

		It("should process resources in batches", func() {
			// Create multiple resources
			var resources []runtime.Object
			for i := 0; i < 25; i++ {
				resources = append(resources, &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("batch-platform-%d", i),
						Namespace: "default",
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Version: "v2.48.0",
							},
						},
					},
				})
			}

			results, err := processor.ProcessBatch(ctx, resources, func(obj runtime.Object) (runtime.Object, error) {
				// Simulate conversion
				v1alpha1Platform := obj.(*v1alpha1.ObservabilityPlatform)
				return &v1beta1.ObservabilityPlatform{
					ObjectMeta: v1alpha1Platform.ObjectMeta,
					Spec: v1beta1.ObservabilityPlatformSpec{
						Components: v1beta1.Components{
							Prometheus: &v1beta1.PrometheusSpec{
								Enabled: v1alpha1Platform.Spec.Components.Prometheus.Enabled,
								Version: v1alpha1Platform.Spec.Components.Prometheus.Version,
							},
						},
					},
				}, nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(results.TotalProcessed).To(Equal(25))
			Expect(results.Successful).To(Equal(25))
			Expect(results.Failed).To(Equal(0))
			Expect(results.BatchesProcessed).To(Equal(3)) // 25 resources / 10 batch size = 3 batches
		})

		It("should handle errors with retry", func() {
			attemptCount := 0
			resources := []runtime.Object{
				&v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-platform",
						Namespace: "default",
					},
				},
			}

			results, err := processor.ProcessBatch(ctx, resources, func(obj runtime.Object) (runtime.Object, error) {
				attemptCount++
				if attemptCount < 3 {
					return nil, fmt.Errorf("temporary error")
				}
				// Success on third attempt
				v1alpha1Platform := obj.(*v1alpha1.ObservabilityPlatform)
				return &v1beta1.ObservabilityPlatform{
					ObjectMeta: v1alpha1Platform.ObjectMeta,
				}, nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(results.Successful).To(Equal(1))
			Expect(results.RetryCount).To(Equal(2))
			Expect(attemptCount).To(Equal(3))
		})

		It("should respect concurrency limits", func() {
			concurrentCount := int32(0)
			maxConcurrent := int32(0)
			resources := make([]runtime.Object, 20)
			for i := 0; i < 20; i++ {
				resources[i] = &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("concurrent-%d", i),
					},
				}
			}

			_, err := processor.ProcessBatch(ctx, resources, func(obj runtime.Object) (runtime.Object, error) {
				current := atomic.AddInt32(&concurrentCount, 1)
				defer atomic.AddInt32(&concurrentCount, -1)

				// Track maximum concurrent executions
				for {
					max := atomic.LoadInt32(&maxConcurrent)
					if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
						break
					}
				}

				// Simulate some work
				time.Sleep(10 * time.Millisecond)

				return &v1beta1.ObservabilityPlatform{
					ObjectMeta: obj.(*v1alpha1.ObservabilityPlatform).ObjectMeta,
				}, nil
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(atomic.LoadInt32(&maxConcurrent)).To(BeNumerically("<=", 3)) // Configured concurrency
		})
	})

	Context("Data preservation", func() {
		It("should preserve custom annotations during migration", func() {
			platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "preservation-test",
					Namespace: "default",
					Annotations: map[string]string{
						"custom.io/config":     `{"key": "value"}`,
						"backup.io/last-backup": "2025-06-12T10:00:00Z",
						"migration.io/source-version": "v1alpha1",
					},
				},
			}

			// Use the data preservation functions
			preservedData := migration.PreserveCustomData(platform)
			Expect(preservedData).To(HaveKey("annotations"))
			
			annotations, ok := preservedData["annotations"].(map[string]string)
			Expect(ok).To(BeTrue())
			Expect(annotations).To(HaveLen(3))
			Expect(annotations["custom.io/config"]).To(Equal(`{"key": "value"}`))
		})

		It("should handle preservation policies", func() {
			policy := migration.PreservationPolicy{
				PreserveAnnotations: []string{"custom.io/*", "backup.io/*"},
				PreserveLabels:      []string{"app", "version"},
				PreserveFinalizers:  true,
				CustomRules: []migration.PreservationRule{
					{
						Field: "spec.components.prometheus.externalLabels",
						Action: migration.PreserveAlways,
					},
				},
			}

			platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "observability",
						"version":     "1.0",
						"environment": "prod", // Should not be preserved
					},
					Annotations: map[string]string{
						"custom.io/setting": "value",    // Should be preserved
						"other.io/config":   "ignored",  // Should not be preserved
					},
				},
			}

			preserved := policy.Apply(platform)
			Expect(preserved.Labels).To(HaveLen(2))
			Expect(preserved.Labels).To(HaveKey("app"))
			Expect(preserved.Labels).To(HaveKey("version"))
			Expect(preserved.Labels).NotTo(HaveKey("environment"))

			Expect(preserved.Annotations).To(HaveLen(1))
			Expect(preserved.Annotations).To(HaveKey("custom.io/setting"))
			Expect(preserved.Annotations).NotTo(HaveKey("other.io/config"))
		})
	})

	Context("Edge cases", func() {
		It("should handle nil fields gracefully", func() {
			platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nil-test",
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						// All nil
					},
				},
			}

			err := migrationManager.ValidateResource(ctx, platform, "v1alpha1", "v1beta1")
			// Should not panic, but might return validation error
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("component"))
			}
		})

		It("should handle very large resources", func() {
			// Create a resource with maximum allowed size
			platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "large-resource",
					Annotations: func() map[string]string {
						annotations := make(map[string]string)
						// Create large annotation data (but stay under etcd limit)
						largeData := make([]byte, 1024) // 1KB
						for i := 0; i < 100; i++ {
							annotations[fmt.Sprintf("data-%d", i)] = string(largeData)
						}
						return annotations
					}(),
				},
			}

			// Should handle without issues
			result, err := migrationManager.DryRunMigration(
				ctx,
				[]runtime.Object{platform},
				"v1alpha1",
				"v1beta1",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})
	})
})

// Helper functions for testing
func createTestPlatform(name string, withComplexity bool) *v1alpha1.ObservabilityPlatform {
	platform := &v1alpha1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.ObservabilityPlatformSpec{
			Components: v1alpha1.Components{
				Prometheus: &v1alpha1.PrometheusSpec{
					Enabled: true,
					Version: "v2.48.0",
				},
			},
		},
	}

	if withComplexity {
		// Add complexity
		platform.Spec.Components.Prometheus.RemoteWrite = []v1alpha1.RemoteWriteSpec{
			{URL: "http://remote1:9090"},
			{URL: "http://remote2:9090"},
		}
		platform.Spec.Components.Grafana = &v1alpha1.GrafanaSpec{
			Enabled: true,
			DataSources: []v1alpha1.DataSourceSpec{
				{Name: "prom", Type: "prometheus", URL: "http://prom:9090"},
			},
		}
		platform.Labels = map[string]string{
			"complexity": "high",
			"test":       "true",
		}
	}

	return platform
}

// BenchmarkConversion benchmarks conversion performance
func BenchmarkConversion(b *testing.B) {
	platform := createTestPlatform("bench-platform", true)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converted := &v1beta1.ObservabilityPlatform{}
		err := convertPlatform(platform, converted)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func convertPlatform(src *v1alpha1.ObservabilityPlatform, dst *v1beta1.ObservabilityPlatform) error {
	// Simulate conversion
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
