/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/api/v1beta1/conversion/preservation"
)

var _ = Describe("Data Preservation Tests", func() {
	Context("Annotation preservation", func() {
		It("should preserve custom annotations according to rules", func() {
			rules := preservation.NewAnnotationPreservationRules()
			rules.AddPattern("custom.io/*")
			rules.AddPattern("backup.io/*")
			rules.AddExactMatch("special-annotation")

			annotations := map[string]string{
				"custom.io/config":          "preserved",
				"custom.io/settings":        "preserved",
				"backup.io/last-backup":     "preserved",
				"special-annotation":        "preserved",
				"kubernetes.io/internal":    "not-preserved",
				"random-annotation":         "not-preserved",
			}

			preserved := rules.Filter(annotations)
			Expect(preserved).To(HaveLen(4))
			Expect(preserved).To(HaveKey("custom.io/config"))
			Expect(preserved).To(HaveKey("custom.io/settings"))
			Expect(preserved).To(HaveKey("backup.io/last-backup"))
			Expect(preserved).To(HaveKey("special-annotation"))
			Expect(preserved).NotTo(HaveKey("kubernetes.io/internal"))
			Expect(preserved).NotTo(HaveKey("random-annotation"))
		})

		It("should handle system annotations specially", func() {
			handler := preservation.NewSystemAnnotationHandler()
			
			annotations := map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"v1alpha1"}`,
				"kubernetes.io/change-cause":                       "kubectl apply",
				"deployment.kubernetes.io/revision":                "3",
				"custom.io/data":                                   "user-data",
			}

			processed := handler.Process(annotations, "v1alpha1", "v1beta1")
			
			// Should update API version in kubectl annotation
			Expect(processed).To(HaveKey("kubectl.kubernetes.io/last-applied-configuration"))
			Expect(processed["kubectl.kubernetes.io/last-applied-configuration"]).To(ContainSubstring("v1beta1"))
			
			// Should preserve other annotations
			Expect(processed).To(HaveKey("kubernetes.io/change-cause"))
			Expect(processed).To(HaveKey("deployment.kubernetes.io/revision"))
			Expect(processed).To(HaveKey("custom.io/data"))
		})
	})

	Context("Label preservation", func() {
		It("should preserve labels based on policies", func() {
			policy := preservation.NewLabelPreservationPolicy()
			policy.SetMode(preservation.PreserveModeWhitelist)
			policy.AddWhitelistLabel("app")
			policy.AddWhitelistLabel("version")
			policy.AddWhitelistPattern("team/*")

			labels := map[string]string{
				"app":                "observability",
				"version":            "1.0.0",
				"team/name":          "platform",
				"team/contact":       "platform@company.com",
				"environment":        "production",
				"kubernetes.io/name": "should-not-preserve",
			}

			preserved := policy.Apply(labels)
			Expect(preserved).To(HaveLen(4))
			Expect(preserved).To(HaveKey("app"))
			Expect(preserved).To(HaveKey("version"))
			Expect(preserved).To(HaveKey("team/name"))
			Expect(preserved).To(HaveKey("team/contact"))
			Expect(preserved).NotTo(HaveKey("environment"))
			Expect(preserved).NotTo(HaveKey("kubernetes.io/name"))
		})

		It("should handle label value transformations", func() {
			transformer := preservation.NewLabelTransformer()
			
			// Add transformation rules
			transformer.AddRule("version", func(value string) string {
				// Convert v1alpha1 version format to v1beta1 format
				if value == "1.0.0-alpha" {
					return "1.0.0-beta"
				}
				return value
			})

			labels := map[string]string{
				"app":     "observability",
				"version": "1.0.0-alpha",
			}

			transformed := transformer.Transform(labels)
			Expect(transformed["version"]).To(Equal("1.0.0-beta"))
			Expect(transformed["app"]).To(Equal("observability"))
		})
	})

	Context("Status field preservation", func() {
		It("should preserve status fields during conversion", func() {
			v1alpha1Status := v1alpha1.ObservabilityPlatformStatus{
				Phase: "Ready",
				Conditions: []metav1.Condition{
					{
						Type:               "Ready",
						Status:             metav1.ConditionTrue,
						Reason:             "AllComponentsReady",
						Message:            "All components are running",
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               "Progressing",
						Status:             metav1.ConditionFalse,
						Reason:             "Stable",
						Message:            "No changes in progress",
						LastTransitionTime: metav1.Now(),
					},
				},
				ObservedGeneration: 5,
				ComponentStatuses: map[string]v1alpha1.ComponentStatus{
					"prometheus": {
						Ready:   true,
						Message: "Prometheus is healthy",
					},
					"grafana": {
						Ready:   true,
						Message: "Grafana is running",
					},
				},
			}

			preserver := preservation.NewStatusPreserver()
			v1beta1Status := preserver.PreserveStatus(v1alpha1Status)

			Expect(v1beta1Status.Phase).To(Equal("Ready"))
			Expect(v1beta1Status.Conditions).To(HaveLen(2))
			Expect(v1beta1Status.ObservedGeneration).To(Equal(int64(5)))
			Expect(v1beta1Status.ComponentStatuses).To(HaveLen(2))
			
			// Verify condition details preserved
			readyCondition := findCondition(v1beta1Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(readyCondition.Reason).To(Equal("AllComponentsReady"))
		})

		It("should merge status updates during conversion", func() {
			oldStatus := v1beta1.ObservabilityPlatformStatus{
				Phase: "Installing",
				Conditions: []metav1.Condition{
					{
						Type:               "Ready",
						Status:             metav1.ConditionFalse,
						Reason:             "Installing",
						Message:            "Installation in progress",
						LastTransitionTime: metav1.Time{Time: metav1.Now().Add(-5 * time.Minute)},
					},
				},
			}

			newStatus := v1beta1.ObservabilityPlatformStatus{
				Phase: "Ready",
				Conditions: []metav1.Condition{
					{
						Type:               "Ready",
						Status:             metav1.ConditionTrue,
						Reason:             "InstallationComplete",
						Message:            "All components installed",
						LastTransitionTime: metav1.Now(),
					},
				},
			}

			merger := preservation.NewStatusMerger()
			merged := merger.Merge(oldStatus, newStatus)

			Expect(merged.Phase).To(Equal("Ready"))
			// Should preserve transition times when status changes
			readyCondition := findCondition(merged.Conditions, "Ready")
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			// LastTransitionTime should be from newStatus since status changed
		})
	})

	Context("Unknown fields detection", func() {
		It("should detect and preserve unknown fields", func() {
			// Simulate a resource with extra fields
			jsonData := `{
				"apiVersion": "observability.io/v1alpha1",
				"kind": "ObservabilityPlatform",
				"metadata": {
					"name": "test-platform"
				},
				"spec": {
					"components": {
						"prometheus": {
							"enabled": true,
							"version": "v2.48.0",
							"customField": "this-is-unknown",
							"nestedUnknown": {
								"key": "value"
							}
						}
					},
					"futureFeature": {
						"enabled": true,
						"config": "some-config"
					}
				}
			}`

			detector := preservation.NewUnknownFieldDetector()
			unknownFields := detector.Detect([]byte(jsonData), v1alpha1.ObservabilityPlatform{})

			Expect(unknownFields).To(HaveKey("spec.components.prometheus.customField"))
			Expect(unknownFields).To(HaveKey("spec.components.prometheus.nestedUnknown"))
			Expect(unknownFields).To(HaveKey("spec.futureFeature"))

			// Should be able to preserve these fields
			preserved := detector.PreserveUnknownFields([]byte(jsonData), unknownFields)
			Expect(preserved).To(HaveKey("spec.components.prometheus.customField"))
			Expect(preserved["spec.components.prometheus.customField"]).To(Equal("this-is-unknown"))
		})

		It("should handle array fields with unknown elements", func() {
			jsonData := `{
				"spec": {
					"components": {
						"prometheus": {
							"remoteWrite": [
								{
									"url": "http://remote1:9090",
									"unknownField": "value1"
								},
								{
									"url": "http://remote2:9090",
									"customConfig": {
										"key": "value"
									}
								}
							]
						}
					}
				}
			}`

			detector := preservation.NewUnknownFieldDetector()
			unknownFields := detector.Detect([]byte(jsonData), v1alpha1.ObservabilityPlatform{})

			Expect(unknownFields).To(HaveKey("spec.components.prometheus.remoteWrite[0].unknownField"))
			Expect(unknownFields).To(HaveKey("spec.components.prometheus.remoteWrite[1].customConfig"))
		})
	})

	Context("Complex preservation scenarios", func() {
		It("should handle preservation during round-trip conversion", func() {
			original := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "round-trip-test",
					Namespace: "default",
					Labels: map[string]string{
						"app":         "observability",
						"version":     "1.0",
						"custom/tag":  "preserved",
					},
					Annotations: map[string]string{
						"description":        "Test platform",
						"custom.io/setting": "value",
						"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"v1alpha1"}`,
					},
					Finalizers: []string{
						"observability.io/cleanup",
						"custom.io/protection",
					},
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:     true,
							Version:     "v2.48.0",
							StorageSize: "100Gi",
							ExternalLabels: map[string]string{
								"cluster": "production",
								"region":  "us-east-1",
							},
						},
					},
				},
			}

			// First conversion: v1alpha1 -> v1beta1
			preservationEngine := preservation.NewPreservationEngine()
			preservationEngine.Configure(preservation.Config{
				PreserveAllCustomAnnotations: true,
				PreserveAllLabels:            true,
				PreserveFinalizers:           true,
				PreserveUnknownFields:        true,
			})

			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			preservedData := preservationEngine.CaptureForConversion(original)
			
			// Simulate conversion
			convertToV1Beta1(original, v1beta1Platform)
			
			// Apply preserved data
			preservationEngine.ApplyPreservedData(v1beta1Platform, preservedData)

			// Verify preservation
			Expect(v1beta1Platform.Labels).To(Equal(original.Labels))
			Expect(v1beta1Platform.Annotations).To(HaveLen(3))
			Expect(v1beta1Platform.Finalizers).To(Equal(original.Finalizers))
			Expect(v1beta1Platform.Spec.Components.Prometheus.ExternalLabels).To(Equal(
				original.Spec.Components.Prometheus.ExternalLabels,
			))

			// Second conversion: v1beta1 -> v1alpha1
			v1alpha1Converted := &v1alpha1.ObservabilityPlatform{}
			preservedData2 := preservationEngine.CaptureForConversion(v1beta1Platform)
			
			// Simulate backward conversion
			convertToV1Alpha1(v1beta1Platform, v1alpha1Converted)
			
			// Apply preserved data
			preservationEngine.ApplyPreservedData(v1alpha1Converted, preservedData2)

			// Verify round-trip preservation
			Expect(v1alpha1Converted.Labels).To(Equal(original.Labels))
			Expect(v1alpha1Converted.Annotations["custom.io/setting"]).To(Equal("value"))
			Expect(v1alpha1Converted.Finalizers).To(Equal(original.Finalizers))
		})

		It("should handle resource size limits during preservation", func() {
			// Create a resource near the size limit
			largePlatform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "large-platform",
					Annotations: make(map[string]string),
				},
			}

			// Add many annotations (simulate near etcd limit)
			totalSize := 0
			for i := 0; totalSize < 1*1024*1024; i++ { // 1MB of annotations
				key := fmt.Sprintf("annotation-%d", i)
				value := string(make([]byte, 1024)) // 1KB each
				largePlatform.Annotations[key] = value
				totalSize += len(key) + len(value)
			}

			limiter := preservation.NewSizeLimiter(1.5 * 1024 * 1024) // 1.5MB limit
			
			// Should truncate to fit within limits
			limited, truncated := limiter.LimitSize(largePlatform)
			Expect(truncated).To(BeTrue())
			
			// Verify size is under limit
			data, err := json.Marshal(limited)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(data)).To(BeNumerically("<", 1.5*1024*1024))
		})
	})

	Context("Preservation policies", func() {
		It("should apply configurable preservation policies", func() {
			policy := preservation.NewPolicy()
			policy.AddRule(preservation.Rule{
				Type:    preservation.RuleTypeAnnotation,
				Pattern: "backup.io/*",
				Action:  preservation.ActionPreserve,
			})
			policy.AddRule(preservation.Rule{
				Type:    preservation.RuleTypeLabel,
				Pattern: "version",
				Action:  preservation.ActionTransform,
				Transformer: func(value string) string {
					// Add prefix to version
					return "v" + value
				},
			})
			policy.AddRule(preservation.Rule{
				Type:    preservation.RuleTypeFinalizer,
				Pattern: "*",
				Action:  preservation.ActionPreserve,
			})

			platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"version": "1.0.0",
						"app":     "observability",
					},
					Annotations: map[string]string{
						"backup.io/schedule": "daily",
						"other.io/config":    "ignored",
					},
					Finalizers: []string{
						"protection.io/cleanup",
					},
				},
			}

			result := policy.Apply(platform)
			
			Expect(result.Labels["version"]).To(Equal("v1.0.0")) // Transformed
			Expect(result.Labels).To(HaveKey("app")) // Preserved by default
			Expect(result.Annotations).To(HaveKey("backup.io/schedule"))
			Expect(result.Annotations).NotTo(HaveKey("other.io/config"))
			Expect(result.Finalizers).To(Equal(platform.Finalizers))
		})

		It("should handle conflicting preservation rules", func() {
			policy := preservation.NewPolicy()
			
			// Add conflicting rules
			policy.AddRule(preservation.Rule{
				Type:     preservation.RuleTypeAnnotation,
				Pattern:  "conflict/*",
				Action:   preservation.ActionPreserve,
				Priority: 1,
			})
			policy.AddRule(preservation.Rule{
				Type:     preservation.RuleTypeAnnotation,
				Pattern:  "conflict/specific",
				Action:   preservation.ActionDrop,
				Priority: 2, // Higher priority wins
			})

			annotations := map[string]string{
				"conflict/general":  "preserved",
				"conflict/specific": "dropped",
			}

			result := policy.ApplyToAnnotations(annotations)
			Expect(result).To(HaveKey("conflict/general"))
			Expect(result).NotTo(HaveKey("conflict/specific"))
		})
	})
})

// Helper functions
func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func convertToV1Beta1(src *v1alpha1.ObservabilityPlatform, dst *v1beta1.ObservabilityPlatform) {
	// Simulate basic conversion
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Components.Prometheus = &v1beta1.PrometheusSpec{
		Enabled:        src.Spec.Components.Prometheus.Enabled,
		Version:        src.Spec.Components.Prometheus.Version,
		StorageSize:    src.Spec.Components.Prometheus.StorageSize,
		ExternalLabels: src.Spec.Components.Prometheus.ExternalLabels,
	}
}

func convertToV1Alpha1(src *v1beta1.ObservabilityPlatform, dst *v1alpha1.ObservabilityPlatform) {
	// Simulate backward conversion
	dst.ObjectMeta = src.ObjectMeta
	if src.Spec.Components.Prometheus != nil {
		dst.Spec.Components.Prometheus = &v1alpha1.PrometheusSpec{
			Enabled:        src.Spec.Components.Prometheus.Enabled,
			Version:        src.Spec.Components.Prometheus.Version,
			StorageSize:    src.Spec.Components.Prometheus.StorageSize,
			ExternalLabels: src.Spec.Components.Prometheus.ExternalLabels,
		}
	}
}


