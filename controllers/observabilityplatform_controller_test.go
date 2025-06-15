/*
Copyright 2025.

Licensed under the MIT License.
*/

package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/managers"
)

var _ = Describe("ObservabilityPlatform Controller", func() {
	const (
		PlatformName      = "test-platform"
		PlatformNamespace = "test-namespace"
		timeout           = time.Second * 30
		interval          = time.Millisecond * 250
	)

	Context("When reconciling a resource", func() {
		ctx := context.Background()

		BeforeEach(func() {
			// Create namespace
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: PlatformNamespace,
				},
			}
			err := k8sClient.Create(ctx, namespace)
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			// Clean up
			platform := &observabilityv1beta1.ObservabilityPlatform{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      PlatformName,
				Namespace: PlatformNamespace,
			}, platform)
			if err == nil {
				Expect(k8sClient.Delete(ctx, platform)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Creating a new ObservabilityPlatform")
			platform := &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())

			By("Checking if the platform was created")
			Eventually(func() error {
				found := &observabilityv1beta1.ObservabilityPlatform{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				}, found)
			}, timeout, interval).Should(Succeed())

			By("Checking if finalizer was added")
			Eventually(func() bool {
				found := &observabilityv1beta1.ObservabilityPlatform{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				}, found)
				if err != nil {
					return false
				}
				for _, finalizer := range found.GetFinalizers() {
					if finalizer == FinalizerName {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			By("Checking if status was updated")
			Eventually(func() observabilityv1beta1.Phase {
				found := &observabilityv1beta1.ObservabilityPlatform{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				}, found)
				if err != nil {
					return ""
				}
				return found.Status.Phase
			}, timeout, interval).Should(Equal(observabilityv1beta1.PhaseReady))
		})

		It("should handle paused reconciliation", func() {
			By("Creating a paused ObservabilityPlatform")
			platform := &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Paused: true,
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())

			By("Checking if status shows paused")
			Eventually(func() observabilityv1beta1.Phase {
				found := &observabilityv1beta1.ObservabilityPlatform{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				}, found)
				if err != nil {
					return ""
				}
				return found.Status.Phase
			}, timeout, interval).Should(Equal(observabilityv1beta1.PhasePaused))
		})

		It("should handle deletion with finalizer", func() {
			By("Creating an ObservabilityPlatform")
			platform := &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					Components: observabilityv1beta1.Components{
						Prometheus: &observabilityv1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, platform)).To(Succeed())

			By("Waiting for finalizer to be added")
			Eventually(func() bool {
				found := &observabilityv1beta1.ObservabilityPlatform{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				}, found)
				if err != nil {
					return false
				}
				return len(found.GetFinalizers()) > 0
			}, timeout, interval).Should(BeTrue())

			By("Deleting the ObservabilityPlatform")
			found := &observabilityv1beta1.ObservabilityPlatform{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      PlatformName,
				Namespace: PlatformNamespace,
			}, found)).To(Succeed())
			Expect(k8sClient.Delete(ctx, found)).To(Succeed())

			By("Checking if the platform is eventually deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      PlatformName,
					Namespace: PlatformNamespace,
				}, &observabilityv1beta1.ObservabilityPlatform{})
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When testing controller client operations", func() {
		It("should demonstrate client usage patterns", func() {
			ctx := context.Background()

			By("Creating a resource using the client")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"key": "value",
				},
			}
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())

			By("Getting a resource using the client")
			retrieved := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), retrieved)).To(Succeed())
			Expect(retrieved.Data["key"]).To(Equal("value"))

			By("Updating a resource using the client")
			retrieved.Data["key"] = "updated-value"
			Expect(k8sClient.Update(ctx, retrieved)).To(Succeed())

			By("Listing resources using the client")
			list := &corev1.ConfigMapList{}
			Expect(k8sClient.List(ctx, list, client.InNamespace("default"))).To(Succeed())
			Expect(len(list.Items)).To(BeNumerically(">", 0))

			By("Patching a resource using the client")
			patch := client.MergeFrom(retrieved.DeepCopy())
			retrieved.Data["new-key"] = "new-value"
			Expect(k8sClient.Patch(ctx, retrieved, patch)).To(Succeed())

			By("Deleting a resource using the client")
			Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
		})
	})
})

// Test reconciler with mock managers
var _ = Describe("ObservabilityPlatformReconciler Unit Tests", func() {
	var (
		reconciler *ObservabilityPlatformReconciler
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &ObservabilityPlatformReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			PrometheusManager: &managers.MockPrometheusManager{
				ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
					return nil
				},
			},
			GrafanaManager: &managers.MockGrafanaManager{
				ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
					return nil
				},
			},
			LokiManager: &managers.MockLokiManager{
				ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
					return nil
				},
			},
			TempoManager: &managers.MockTempoManager{
				ReconcileFn: func(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
					return nil
				},
			},
		}
	})

	It("should reconcile successfully with all components", func() {
		// Create platform
		platform := &observabilityv1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-all-components",
				Namespace: "default",
			},
			Spec: observabilityv1beta1.ObservabilityPlatformSpec{
				Components: observabilityv1beta1.Components{
					Prometheus: &observabilityv1beta1.PrometheusSpec{Enabled: true},
					Grafana:    &observabilityv1beta1.GrafanaSpec{Enabled: true},
					Loki:       &observabilityv1beta1.LokiSpec{Enabled: true},
					Tempo:      &observabilityv1beta1.TempoSpec{Enabled: true},
				},
			},
		}
		Expect(k8sClient.Create(ctx, platform)).To(Succeed())

		// Reconcile
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      platform.Name,
				Namespace: platform.Namespace,
			},
		}
		result, err := reconciler.Reconcile(ctx, req)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeNumerically(">", 0))
	})
})
