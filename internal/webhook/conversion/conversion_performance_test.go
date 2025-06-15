/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("Conversion Performance Tests", func() {
	const (
		perfTestNamespace = "test-conversion-perf"
		timeout           = time.Minute * 5
		interval          = time.Second
	)

	BeforeEach(func() {
		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: perfTestNamespace,
			},
		}
		err := k8sClient.Create(ctx, ns)
		if err != nil {
			// Namespace might already exist
			_ = k8sClient.Get(ctx, client.ObjectKey{Name: perfTestNamespace}, ns)
		}
	})

	AfterEach(func() {
		// Clean up test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: perfTestNamespace,
			},
		}
		_ = k8sClient.Delete(ctx, ns)
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: perfTestNamespace}, &corev1.Namespace{})
			return err != nil
		}, timeout, interval).Should(BeTrue())
	})

	Context("Bulk conversion performance", func() {
		It("should handle bulk conversion of 100 resources efficiently", func() {
			const resourceCount = 100
			startTime := time.Now()

			By("creating many v1alpha1 resources")
			var wg sync.WaitGroup
			errChan := make(chan error, resourceCount)

			for i := 0; i < resourceCount; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()

					platform := &v1alpha1.ObservabilityPlatform{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("perf-platform-%d", index),
							Namespace: perfTestNamespace,
							Labels: map[string]string{
								"test":  "performance",
								"index": fmt.Sprintf("%d", index),
							},
						},
						Spec: v1alpha1.ObservabilityPlatformSpec{
							Components: v1alpha1.Components{
								Prometheus: &v1alpha1.PrometheusSpec{
									Enabled:     true,
									Version:     "v2.48.0",
									Replicas:    int32(index%3 + 1),
									StorageSize: fmt.Sprintf("%dGi", 10+index%10),
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", 100+index%10*10)),
											corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", 512+index%10*100)),
										},
									},
								},
							},
						},
					}

					if err := k8sClient.Create(context.Background(), platform); err != nil {
						errChan <- err
					}
				}(i)
			}

			wg.Wait()
			close(errChan)

			// Check for any errors
			var errors []error
			for err := range errChan {
				errors = append(errors, err)
			}
			Expect(errors).To(BeEmpty())

			creationDuration := time.Since(startTime)
			By(fmt.Sprintf("created %d resources in %v", resourceCount, creationDuration))

			// Measure conversion performance
			conversionStart := time.Now()

			By("fetching all resources as v1beta1 (triggering conversion)")
			var conversionWg sync.WaitGroup
			conversionErrChan := make(chan error, resourceCount)

			for i := 0; i < resourceCount; i++ {
				conversionWg.Add(1)
				go func(index int) {
					defer conversionWg.Done()

					v1beta1Platform := &v1beta1.ObservabilityPlatform{}
					err := k8sClient.Get(context.Background(), client.ObjectKey{
						Name:      fmt.Sprintf("perf-platform-%d", index),
						Namespace: perfTestNamespace,
					}, v1beta1Platform)

					if err != nil {
						conversionErrChan <- err
						return
					}

					// Verify conversion
					if v1beta1Platform.Spec.Components.Prometheus.Version != "v2.48.0" {
						conversionErrChan <- fmt.Errorf("conversion failed for platform %d", index)
					}
				}(i)
			}

			conversionWg.Wait()
			close(conversionErrChan)

			// Check conversion errors
			var conversionErrors []error
			for err := range conversionErrChan {
				conversionErrors = append(conversionErrors, err)
			}
			Expect(conversionErrors).To(BeEmpty())

			conversionDuration := time.Since(conversionStart)
			By(fmt.Sprintf("converted %d resources in %v", resourceCount, conversionDuration))

			// Performance assertions
			avgConversionTime := conversionDuration.Milliseconds() / int64(resourceCount)
			By(fmt.Sprintf("average conversion time per resource: %dms", avgConversionTime))
			
			// Conversion should be fast - less than 50ms per resource on average
			Expect(avgConversionTime).To(BeNumerically("<", 50))
		})

		It("should handle concurrent conversions efficiently", func() {
			const concurrentRequests = 50

			// Create a single large resource
			largePlatform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrent-test-platform",
					Namespace: perfTestNamespace,
					Labels: map[string]string{
						"test": "concurrent",
					},
					Annotations: map[string]string{
						"description": "Platform for concurrent conversion testing",
					},
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:         true,
							Version:         "v2.48.0",
							Replicas:        3,
							Retention:       "30d",
							StorageSize:     "100Gi",
							StorageClass:    "fast-ssd",
							AlertmanagerURL: "http://alertmanager:9093",
							RemoteWrite: []v1alpha1.RemoteWriteSpec{
								{URL: "http://remote1:9090/api/v1/write"},
								{URL: "http://remote2:9090/api/v1/write"},
								{URL: "http://remote3:9090/api/v1/write"},
							},
							ExternalLabels: map[string]string{
								"cluster":     "production",
								"region":      "us-east-1",
								"environment": "prod",
								"team":        "platform",
							},
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled:       true,
							Version:       "10.0.0",
							AdminPassword: "secure-password",
							AdminUser:     "admin",
							DataSources: []v1alpha1.DataSourceSpec{
								{Name: "Prometheus", Type: "prometheus", URL: "http://prometheus:9090"},
								{Name: "Loki", Type: "loki", URL: "http://loki:3100"},
								{Name: "Tempo", Type: "tempo", URL: "http://tempo:3200"},
							},
						},
						Loki: &v1alpha1.LokiSpec{
							Enabled:      true,
							Version:      "2.9.0",
							StorageSize:  "200Gi",
							StorageClass: "fast-ssd",
							Retention:    "7d",
						},
						Tempo: &v1alpha1.TempoSpec{
							Enabled:      true,
							Version:      "2.3.0",
							StorageSize:  "50Gi",
							StorageClass: "standard",
						},
					},
					Global: v1alpha1.GlobalSpec{
						LogLevel: "info",
						ExternalLabels: map[string]string{
							"datacenter": "us-east-1a",
							"cluster":    "primary",
						},
					},
				},
			}

			By("creating the test platform")
			Expect(k8sClient.Create(ctx, largePlatform)).Should(Succeed())

			By(fmt.Sprintf("performing %d concurrent conversions", concurrentRequests))
			startTime := time.Now()

			var wg sync.WaitGroup
			errChan := make(chan error, concurrentRequests)
			successCount := int32(0)

			for i := 0; i < concurrentRequests; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()

					// Alternate between fetching as v1alpha1 and v1beta1
					if index%2 == 0 {
						v1beta1Platform := &v1beta1.ObservabilityPlatform{}
						err := k8sClient.Get(context.Background(), client.ObjectKey{
							Name:      largePlatform.Name,
							Namespace: largePlatform.Namespace,
						}, v1beta1Platform)

						if err != nil {
							errChan <- err
						} else if v1beta1Platform.Spec.Components.Prometheus.Version == "v2.48.0" {
							atomic.AddInt32(&successCount, 1)
						}
					} else {
						v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
						err := k8sClient.Get(context.Background(), client.ObjectKey{
							Name:      largePlatform.Name,
							Namespace: largePlatform.Namespace,
						}, v1alpha1Platform)

						if err != nil {
							errChan <- err
						} else if v1alpha1Platform.Spec.Components.Prometheus.Version == "v2.48.0" {
							atomic.AddInt32(&successCount, 1)
						}
					}
				}(i)
			}

			wg.Wait()
			close(errChan)

			duration := time.Since(startTime)
			By(fmt.Sprintf("completed %d concurrent conversions in %v", concurrentRequests, duration))

			// Check for errors
			var errors []error
			for err := range errChan {
				errors = append(errors, err)
			}
			Expect(errors).To(BeEmpty())
			Expect(atomic.LoadInt32(&successCount)).To(Equal(int32(concurrentRequests)))

			// Performance check - should handle 50 concurrent conversions in under 5 seconds
			Expect(duration).To(BeNumerically("<", 5*time.Second))
		})

		It("should efficiently convert resources with large data payloads", func() {
			// Create a platform with many labels, annotations, and configuration
			largePlatform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "large-payload-platform",
					Namespace: perfTestNamespace,
					Labels:    make(map[string]string),
					Annotations: make(map[string]string),
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:        true,
							Version:        "v2.48.0",
							ExternalLabels: make(map[string]string),
							RemoteWrite:    make([]v1alpha1.RemoteWriteSpec, 0),
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled:     true,
							DataSources: make([]v1alpha1.DataSourceSpec, 0),
						},
					},
				},
			}

			// Add many labels and annotations
			for i := 0; i < 50; i++ {
				largePlatform.Labels[fmt.Sprintf("label-%d", i)] = fmt.Sprintf("value-%d", i)
				largePlatform.Annotations[fmt.Sprintf("annotation-%d", i)] = fmt.Sprintf("long-value-%d-%s", i, string(make([]byte, 100)))
			}

			// Add many external labels
			for i := 0; i < 30; i++ {
				largePlatform.Spec.Components.Prometheus.ExternalLabels[fmt.Sprintf("external-label-%d", i)] = fmt.Sprintf("value-%d", i)
			}

			// Add many remote write endpoints
			for i := 0; i < 20; i++ {
				largePlatform.Spec.Components.Prometheus.RemoteWrite = append(
					largePlatform.Spec.Components.Prometheus.RemoteWrite,
					v1alpha1.RemoteWriteSpec{
						URL: fmt.Sprintf("http://remote-write-%d:9090/api/v1/write", i),
					},
				)
			}

			// Add many data sources
			for i := 0; i < 20; i++ {
				largePlatform.Spec.Components.Grafana.DataSources = append(
					largePlatform.Spec.Components.Grafana.DataSources,
					v1alpha1.DataSourceSpec{
						Name: fmt.Sprintf("datasource-%d", i),
						Type: "prometheus",
						URL:  fmt.Sprintf("http://prometheus-%d:9090", i),
					},
				)
			}

			By("creating large payload platform")
			startTime := time.Now()
			Expect(k8sClient.Create(ctx, largePlatform)).Should(Succeed())
			creationDuration := time.Since(startTime)

			By("converting large payload platform")
			conversionStart := time.Now()
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      largePlatform.Name,
					Namespace: largePlatform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())
			conversionDuration := time.Since(conversionStart)

			By("verifying large payload conversion")
			Expect(v1beta1Platform.Labels).To(HaveLen(50))
			Expect(v1beta1Platform.Annotations).To(HaveLen(50))
			Expect(v1beta1Platform.Spec.Components.Prometheus.ExternalLabels).To(HaveLen(30))
			Expect(v1beta1Platform.Spec.Components.Prometheus.RemoteWrite).To(HaveLen(20))
			Expect(v1beta1Platform.Spec.Components.Grafana.DataSources).To(HaveLen(20))

			By(fmt.Sprintf("large payload creation took %v, conversion took %v", creationDuration, conversionDuration))
			// Large payload conversion should still be fast - under 100ms
			Expect(conversionDuration).To(BeNumerically("<", 100*time.Millisecond))
		})
	})

	Context("Memory efficiency", func() {
		It("should not leak memory during repeated conversions", func() {
			Skip("Memory profiling requires special setup")
			// This test would require memory profiling tools
			// In a real implementation, you would:
			// 1. Get initial memory stats
			// 2. Perform many conversions
			// 3. Force garbage collection
			// 4. Check final memory stats
			// 5. Ensure memory usage is stable
		})
	})

	Context("Stress testing", func() {
		It("should handle rapid create/update/delete cycles with conversions", func() {
			const cycles = 20
			
			for i := 0; i < cycles; i++ {
				platform := &v1alpha1.ObservabilityPlatform{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("stress-platform-%d", i),
						Namespace: perfTestNamespace,
					},
					Spec: v1alpha1.ObservabilityPlatformSpec{
						Components: v1alpha1.Components{
							Prometheus: &v1alpha1.PrometheusSpec{
								Enabled: true,
								Version: fmt.Sprintf("v2.%d.0", 40+i),
							},
						},
					},
				}

				By(fmt.Sprintf("cycle %d: creating", i))
				Expect(k8sClient.Create(ctx, platform)).Should(Succeed())

				By(fmt.Sprintf("cycle %d: fetching as v1beta1", i))
				v1beta1Platform := &v1beta1.ObservabilityPlatform{}
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{
						Name:      platform.Name,
						Namespace: platform.Namespace,
					}, v1beta1Platform)
				}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

				By(fmt.Sprintf("cycle %d: updating", i))
				v1beta1Platform.Spec.Components.Prometheus.Replicas = int32(i + 1)
				Expect(k8sClient.Update(ctx, v1beta1Platform)).Should(Succeed())

				By(fmt.Sprintf("cycle %d: deleting", i))
				Expect(k8sClient.Delete(ctx, v1beta1Platform)).Should(Succeed())

				By(fmt.Sprintf("cycle %d: verifying deletion", i))
				Eventually(func() bool {
					err := k8sClient.Get(ctx, client.ObjectKey{
						Name:      platform.Name,
						Namespace: platform.Namespace,
					}, &v1alpha1.ObservabilityPlatform{})
					return err != nil
				}, 10*time.Second, 100*time.Millisecond).Should(BeTrue())
			}
		})
	})
})


