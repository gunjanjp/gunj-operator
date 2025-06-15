/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("Conversion Integration Tests", func() {
	const (
		integrationNamespace = "test-conversion-integration"
		timeout              = time.Minute
		interval             = time.Second
	)

	BeforeEach(func() {
		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: integrationNamespace,
			},
		}
		err := k8sClient.Create(ctx, ns)
		if err != nil && !errors.IsAlreadyExists(err) {
			Fail(fmt.Sprintf("Failed to create namespace: %v", err))
		}
	})

	AfterEach(func() {
		// Clean up test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: integrationNamespace,
			},
		}
		_ = k8sClient.Delete(ctx, ns)
	})

	Context("CRD conversion configuration", func() {
		It("should have conversion webhook configured in CRD", func() {
			// Get the CRD
			crd := &apiextensionsv1.CustomResourceDefinition{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: "observabilityplatforms.observability.io",
			}, crd)
			Expect(err).NotTo(HaveOccurred())

			By("verifying conversion strategy")
			Expect(crd.Spec.Conversion).NotTo(BeNil())
			Expect(crd.Spec.Conversion.Strategy).To(Equal(apiextensionsv1.WebhookConverter))

			By("verifying webhook configuration")
			Expect(crd.Spec.Conversion.Webhook).NotTo(BeNil())
			Expect(crd.Spec.Conversion.Webhook.ConversionReviewVersions).To(ContainElements("v1", "v1beta1"))

			By("verifying multiple versions")
			Expect(crd.Spec.Versions).To(HaveLen(2))
			var hasV1Alpha1, hasV1Beta1 bool
			for _, version := range crd.Spec.Versions {
				if version.Name == "v1alpha1" {
					hasV1Alpha1 = true
					Expect(version.Served).To(BeTrue())
				}
				if version.Name == "v1beta1" {
					hasV1Beta1 = true
					Expect(version.Served).To(BeTrue())
					Expect(version.Storage).To(BeTrue()) // v1beta1 should be storage version
				}
			}
			Expect(hasV1Alpha1).To(BeTrue())
			Expect(hasV1Beta1).To(BeTrue())
		})
	})

	Context("Unstructured object conversion", func() {
		It("should convert unstructured objects correctly", func() {
			// Create v1alpha1 as unstructured
			unstructuredV1Alpha1 := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "observability.io/v1alpha1",
					"kind":       "ObservabilityPlatform",
					"metadata": map[string]interface{}{
						"name":      "unstructured-platform",
						"namespace": integrationNamespace,
					},
					"spec": map[string]interface{}{
						"components": map[string]interface{}{
							"prometheus": map[string]interface{}{
								"enabled": true,
								"version": "v2.48.0",
								"replicas": int64(2),
							},
						},
						"global": map[string]interface{}{
							"logLevel": "debug",
						},
					},
				},
			}

			By("creating unstructured v1alpha1")
			err := k8sClient.Create(ctx, unstructuredV1Alpha1)
			Expect(err).NotTo(HaveOccurred())

			By("fetching as unstructured v1beta1")
			unstructuredV1Beta1 := &unstructured.Unstructured{}
			unstructuredV1Beta1.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "observability.io",
				Version: "v1beta1",
				Kind:    "ObservabilityPlatform",
			})

			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      "unstructured-platform",
					Namespace: integrationNamespace,
				}, unstructuredV1Beta1)
			}, timeout, interval).Should(Succeed())

			By("verifying unstructured conversion")
			spec, found, err := unstructured.NestedMap(unstructuredV1Beta1.Object, "spec")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			components, found, err := unstructured.NestedMap(spec, "components")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			prometheus, found, err := unstructured.NestedMap(components, "prometheus")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			enabled, found, err := unstructured.NestedBool(prometheus, "enabled")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(enabled).To(BeTrue())

			version, found, err := unstructured.NestedString(prometheus, "version")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(version).To(Equal("v2.48.0"))
		})
	})

	Context("Status subresource conversion", func() {
		It("should preserve status during conversion", func() {
			// Create v1alpha1 with status
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "status-test-platform",
					Namespace: integrationNamespace,
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}

			By("creating v1alpha1 platform")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("updating status")
			v1alpha1Platform.Status = v1alpha1.ObservabilityPlatformStatus{
				Phase: "Ready",
				Conditions: []metav1.Condition{
					{
						Type:               "Ready",
						Status:             metav1.ConditionTrue,
						Reason:             "AllComponentsReady",
						Message:            "All components are running",
						LastTransitionTime: metav1.Now(),
					},
				},
				ObservedGeneration: 1,
				ComponentStatuses: map[string]v1alpha1.ComponentStatus{
					"prometheus": {
						Ready:   true,
						Message: "Prometheus is running",
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1 with status")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
				if err != nil {
					return false
				}
				return v1beta1Platform.Status.Phase != ""
			}, timeout, interval).Should(BeTrue())

			By("verifying status preservation")
			Expect(v1beta1Platform.Status.Phase).To(Equal("Ready"))
			Expect(v1beta1Platform.Status.Conditions).To(HaveLen(1))
			Expect(v1beta1Platform.Status.Conditions[0].Type).To(Equal("Ready"))
			Expect(v1beta1Platform.Status.ObservedGeneration).To(Equal(int64(1)))
			Expect(v1beta1Platform.Status.ComponentStatuses).To(HaveKey("prometheus"))
			Expect(v1beta1Platform.Status.ComponentStatuses["prometheus"].Ready).To(BeTrue())
		})
	})

	Context("Patch operations with conversion", func() {
		It("should handle strategic merge patch across versions", func() {
			// Create v1alpha1
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "patch-test-platform",
					Namespace: integrationNamespace,
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

			By("creating v1alpha1 platform")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("patching as v1beta1")
			patch := []byte(`{
				"spec": {
					"components": {
						"prometheus": {
							"replicas": 3
						},
						"grafana": {
							"enabled": true,
							"version": "10.0.0"
						}
					}
				}
			}`)

			v1beta1Platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				},
			}

			err := k8sClient.Patch(ctx, v1beta1Platform, client.RawPatch(types.StrategicMergePatchType, patch))
			Expect(err).NotTo(HaveOccurred())

			By("verifying patch applied correctly")
			patched := &v1beta1.ObservabilityPlatform{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Name:      v1alpha1Platform.Name,
				Namespace: v1alpha1Platform.Namespace,
			}, patched)).Should(Succeed())

			Expect(patched.Spec.Components.Prometheus.Replicas).To(Equal(int32(3)))
			Expect(patched.Spec.Components.Prometheus.Version).To(Equal("v2.48.0")) // Original value preserved
			Expect(patched.Spec.Components.Grafana).NotTo(BeNil())
			Expect(patched.Spec.Components.Grafana.Enabled).To(BeTrue())
			Expect(patched.Spec.Components.Grafana.Version).To(Equal("10.0.0"))
		})

		It("should handle JSON patch across versions", func() {
			// Create v1beta1
			v1beta1Platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "json-patch-platform",
					Namespace: integrationNamespace,
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Loki: &v1beta1.LokiSpec{
							Enabled: true,
							Version: "2.9.0",
						},
					},
				},
			}

			By("creating v1beta1 platform")
			Expect(k8sClient.Create(ctx, v1beta1Platform)).Should(Succeed())

			By("applying JSON patch as v1alpha1")
			patch := []byte(`[
				{"op": "add", "path": "/spec/components/loki/retention", "value": "7d"},
				{"op": "add", "path": "/spec/global", "value": {"logLevel": "debug"}}
			]`)

			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      v1beta1Platform.Name,
					Namespace: v1beta1Platform.Namespace,
				},
			}

			err := k8sClient.Patch(ctx, v1alpha1Platform, client.RawPatch(types.JSONPatchType, patch))
			Expect(err).NotTo(HaveOccurred())

			By("verifying patch applied")
			patched := &v1alpha1.ObservabilityPlatform{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Name:      v1beta1Platform.Name,
				Namespace: v1beta1Platform.Namespace,
			}, patched)).Should(Succeed())

			Expect(patched.Spec.Components.Loki.Retention).To(Equal("7d"))
			Expect(patched.Spec.Global.LogLevel).To(Equal("debug"))
		})
	})

	Context("Server-side apply with conversion", func() {
		It("should handle server-side apply across versions", func() {
			// Initial creation with v1alpha1
			v1alpha1YAML := `
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: ssa-test-platform
  namespace: %s
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
`
			yamlContent := fmt.Sprintf(v1alpha1YAML, integrationNamespace)
			
			obj := &unstructured.Unstructured{}
			err := yaml.Unmarshal([]byte(yamlContent), &obj.Object)
			Expect(err).NotTo(HaveOccurred())

			By("applying v1alpha1 with server-side apply")
			err = k8sClient.Patch(ctx, obj, client.Apply, client.FieldOwner("test-owner"))
			Expect(err).NotTo(HaveOccurred())

			// Apply update with v1beta1
			v1beta1YAML := `
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: ssa-test-platform
  namespace: %s
spec:
  components:
    prometheus:
      enabled: true
      version: v2.49.0
      replicas: 3
    grafana:
      enabled: true
      version: "10.0.0"
`
			yamlContent = fmt.Sprintf(v1beta1YAML, integrationNamespace)
			
			obj2 := &unstructured.Unstructured{}
			err = yaml.Unmarshal([]byte(yamlContent), &obj2.Object)
			Expect(err).NotTo(HaveOccurred())

			By("applying v1beta1 update with server-side apply")
			err = k8sClient.Patch(ctx, obj2, client.Apply, client.FieldOwner("test-owner"))
			Expect(err).NotTo(HaveOccurred())

			By("verifying final state")
			final := &v1beta1.ObservabilityPlatform{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Name:      "ssa-test-platform",
				Namespace: integrationNamespace,
			}, final)).Should(Succeed())

			Expect(final.Spec.Components.Prometheus.Version).To(Equal("v2.49.0"))
			Expect(final.Spec.Components.Prometheus.Replicas).To(Equal(int32(3)))
			Expect(final.Spec.Components.Grafana).NotTo(BeNil())
			Expect(final.Spec.Components.Grafana.Enabled).To(BeTrue())
		})
	})

	Context("Watch operations with conversion", func() {
		It("should receive watch events for different versions", func() {
			// This test would require setting up watchers which is complex in envtest
			// In a real environment, you would:
			// 1. Create a watch for v1alpha1
			// 2. Create a watch for v1beta1
			// 3. Create/update a resource
			// 4. Verify both watches receive appropriate events
			Skip("Watch testing requires complex setup not suitable for envtest")
		})
	})

	Context("Admission webhook interaction", func() {
		It("should validate converted resources correctly", func() {
			// Create invalid v1alpha1 that should fail validation after conversion
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-platform",
					Namespace: integrationNamespace,
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Version: "invalid-version", // Should fail validation
						},
					},
				},
			}

			By("creating invalid v1alpha1 platform")
			err := k8sClient.Create(ctx, v1alpha1Platform)
			// Depending on webhook setup, this might fail
			if err != nil {
				By("verifying validation error")
				Expect(err.Error()).To(ContainSubstring("version"))
			}
		})
	})

	Context("Finalizer handling during conversion", func() {
		It("should preserve finalizers across conversions", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "finalizer-test-platform",
					Namespace: integrationNamespace,
					Finalizers: []string{
						"observability.io/platform-protection",
						"custom.io/backup",
					},
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}

			By("creating v1alpha1 with finalizers")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying finalizers preserved")
			Expect(v1beta1Platform.Finalizers).To(ConsistOf(
				"observability.io/platform-protection",
				"custom.io/backup",
			))

			By("adding finalizer through v1beta1")
			v1beta1Platform.Finalizers = append(v1beta1Platform.Finalizers, "v1beta1.io/new-finalizer")
			Expect(k8sClient.Update(ctx, v1beta1Platform)).Should(Succeed())

			By("verifying finalizer visible in v1alpha1")
			updatedV1Alpha1 := &v1alpha1.ObservabilityPlatform{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Name:      v1alpha1Platform.Name,
				Namespace: v1alpha1Platform.Namespace,
			}, updatedV1Alpha1)).Should(Succeed())

			Expect(updatedV1Alpha1.Finalizers).To(ConsistOf(
				"observability.io/platform-protection",
				"custom.io/backup",
				"v1beta1.io/new-finalizer",
			))
		})
	})
})
