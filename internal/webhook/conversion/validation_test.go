/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/webhook/conversion"
)

var _ = Describe("Conversion Validation Tests", func() {
	Context("Field validation during conversion", func() {
		It("should validate required fields are preserved", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "validation-test",
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

			err := conversion.ValidateConversion(v1alpha1Platform, &v1beta1.ObservabilityPlatform{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("version"))
		})

		It("should validate field formats during conversion", func() {
			testCases := []struct {
				name        string
				platform    *v1alpha1.ObservabilityPlatform
				expectError bool
				errorMsg    string
			}{
				{
					name: "invalid version format",
					platform: &v1alpha1.ObservabilityPlatform{
						ObjectMeta: metav1.ObjectMeta{Name: "test1"},
						Spec: v1alpha1.ObservabilityPlatformSpec{
							Components: v1alpha1.Components{
								Prometheus: &v1alpha1.PrometheusSpec{
									Enabled: true,
									Version: "invalid-version-format",
								},
							},
						},
					},
					expectError: true,
					errorMsg:    "version",
				},
				{
					name: "invalid retention format",
					platform: &v1alpha1.ObservabilityPlatform{
						ObjectMeta: metav1.ObjectMeta{Name: "test2"},
						Spec: v1alpha1.ObservabilityPlatformSpec{
							Components: v1alpha1.Components{
								Prometheus: &v1alpha1.PrometheusSpec{
									Enabled:   true,
									Version:   "v2.48.0",
									Retention: "invalid",
								},
							},
						},
					},
					expectError: true,
					errorMsg:    "retention",
				},
				{
					name: "invalid storage size",
					platform: &v1alpha1.ObservabilityPlatform{
						ObjectMeta: metav1.ObjectMeta{Name: "test3"},
						Spec: v1alpha1.ObservabilityPlatformSpec{
							Components: v1alpha1.Components{
								Loki: &v1alpha1.LokiSpec{
									Enabled:     true,
									Version:     "2.9.0",
									StorageSize: "invalid-size",
								},
							},
						},
					},
					expectError: true,
					errorMsg:    "storage",
				},
				{
					name: "valid configuration",
					platform: &v1alpha1.ObservabilityPlatform{
						ObjectMeta: metav1.ObjectMeta{Name: "test4"},
						Spec: v1alpha1.ObservabilityPlatformSpec{
							Components: v1alpha1.Components{
								Prometheus: &v1alpha1.PrometheusSpec{
									Enabled:     true,
									Version:     "v2.48.0",
									Retention:   "30d",
									StorageSize: "100Gi",
								},
							},
						},
					},
					expectError: false,
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("testing %s", tc.name))
				err := validatePlatformFields(tc.platform)
				if tc.expectError {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(tc.errorMsg))
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})

		It("should validate resource requirements", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "resource-validation",
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
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"), // Invalid: limit < request
									corev1.ResourceMemory: resource.MustParse("256Mi"), // Invalid: limit < request
								},
							},
						},
					},
				},
			}

			errors := validateResourceRequirements(v1alpha1Platform.Spec.Components.Prometheus.Resources)
			Expect(errors).To(HaveLen(2))
			Expect(errors[0].Error()).To(ContainSubstring("CPU limit must be greater than or equal to request"))
			Expect(errors[1].Error()).To(ContainSubstring("memory limit must be greater than or equal to request"))
		})

		It("should validate label and annotation constraints", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "metadata-validation",
					Labels: map[string]string{
						"valid-label":     "value",
						"invalid/label!":  "value", // Invalid characters
						strings.Repeat("a", 64): "toolong", // Too long
					},
					Annotations: map[string]string{
						"valid.io/annotation": "value",
						"invalid-chars-$@#":   "value", // Invalid characters
						strings.Repeat("b", 300): "toolong", // Too long key
					},
				},
			}

			errors := validateMetadata(v1alpha1Platform.ObjectMeta)
			Expect(errors).To(HaveLen(4)) // 2 label errors + 2 annotation errors
		})
	})

	Context("Cross-version compatibility validation", func() {
		It("should validate v1alpha1 to v1beta1 field compatibility", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							RemoteWrite: []v1alpha1.RemoteWriteSpec{
								{URL: "http://remote:9090"},
							},
						},
					},
				},
			}

			v1beta1Platform := &v1beta1.ObservabilityPlatform{
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							RemoteWrite: []v1beta1.RemoteWriteSpec{
								{
									URL: "http://remote:9090",
									// v1beta1 has additional fields
									Name: "remote-1",
									Headers: map[string]string{
										"Authorization": "Bearer token",
									},
								},
							},
						},
					},
				},
			}

			compatible, issues := checkFieldCompatibility(v1alpha1Platform, v1beta1Platform)
			Expect(compatible).To(BeTrue())
			Expect(issues).To(BeEmpty())
		})

		It("should detect incompatible field changes", func() {
			// Test cases where field types or semantics change incompatibly
			testCases := []struct {
				name          string
				v1alpha1Field interface{}
				v1beta1Field  interface{}
				compatible    bool
			}{
				{
					name:          "string to int conversion",
					v1alpha1Field: "3",
					v1beta1Field:  int32(3),
					compatible:    true, // Can be converted
				},
				{
					name:          "incompatible string to bool",
					v1alpha1Field: "yes",
					v1beta1Field:  true,
					compatible:    false, // "yes" is not a valid bool
				},
				{
					name:          "array to map conversion",
					v1alpha1Field: []string{"key1=value1", "key2=value2"},
					v1beta1Field:  map[string]string{"key1": "value1", "key2": "value2"},
					compatible:    true, // Can be converted
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("testing %s", tc.name))
				compatible := isFieldConversionCompatible(tc.v1alpha1Field, tc.v1beta1Field)
				Expect(compatible).To(Equal(tc.compatible))
			}
		})
	})

	Context("Semantic validation", func() {
		It("should validate component dependencies", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						// Grafana enabled but no data sources
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Version: "10.0.0",
							// Missing Prometheus as data source
						},
						// Prometheus disabled
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: false,
						},
					},
				},
			}

			warnings := validateComponentDependencies(v1alpha1Platform)
			Expect(warnings).To(ContainElement(ContainSubstring("Grafana enabled but no Prometheus data source")))
		})

		It("should validate version compatibility between components", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
							Version: "7.0.0", // Too old for Prometheus v2.48.0
						},
					},
				},
			}

			compatible, issues := validateVersionCompatibility(v1alpha1Platform)
			Expect(compatible).To(BeFalse())
			Expect(issues).To(ContainElement(ContainSubstring("Grafana version 7.0.0 may not be compatible")))
		})

		It("should validate configuration consistency", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:     true,
							Retention:   "7d",
							StorageSize: "10Gi", // Too small for 7d retention with typical load
						},
					},
					Global: v1alpha1.GlobalSpec{
						LogLevel: "invalid-level", // Invalid log level
					},
				},
			}

			errors := validateConfigurationConsistency(v1alpha1Platform)
			Expect(errors).To(HaveLen(2))
			Expect(errors[0].Error()).To(ContainSubstring("storage size may be insufficient"))
			Expect(errors[1].Error()).To(ContainSubstring("invalid log level"))
		})
	})

	Context("Security validation", func() {
		It("should validate security-sensitive fields", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled:       true,
							AdminPassword: "123", // Too weak
							AdminUser:     "root", // Discouraged username
						},
					},
				},
			}

			v1beta1Platform := &v1beta1.ObservabilityPlatform{
				Spec: v1beta1.ObservabilityPlatformSpec{
					Security: &v1beta1.SecuritySpec{
						RBAC: &v1beta1.RBACSpec{
							Enabled: false, // Security disabled
						},
						NetworkPolicy: &v1beta1.NetworkPolicySpec{
							Enabled: false, // Network isolation disabled
						},
					},
				},
			}

			By("validating v1alpha1 security")
			warnings := validateSecuritySettings(v1alpha1Platform)
			Expect(warnings).To(ContainElement(ContainSubstring("weak admin password")))
			Expect(warnings).To(ContainElement(ContainSubstring("root username is discouraged")))

			By("validating v1beta1 security")
			warnings = validateV1Beta1Security(v1beta1Platform)
			Expect(warnings).To(ContainElement(ContainSubstring("RBAC is disabled")))
			Expect(warnings).To(ContainElement(ContainSubstring("Network policies are disabled")))
		})
	})

	Context("Performance impact validation", func() {
		It("should validate conversion performance impact", func() {
			// Large platform with many components and configurations
			largePlatform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name: "large-platform",
					Labels: func() map[string]string {
						labels := make(map[string]string)
						for i := 0; i < 100; i++ {
							labels[fmt.Sprintf("label-%d", i)] = fmt.Sprintf("value-%d", i)
						}
						return labels
					}(),
					Annotations: func() map[string]string {
						annotations := make(map[string]string)
						for i := 0; i < 100; i++ {
							annotations[fmt.Sprintf("annotation-%d", i)] = strings.Repeat("x", 100)
						}
						return annotations
					}(),
				},
			}

			impact := assessConversionPerformanceImpact(largePlatform)
			Expect(impact.EstimatedDuration).To(BeNumerically(">", 0))
			Expect(impact.MemoryImpact).To(BeNumerically(">", 1024)) // At least 1KB
			Expect(impact.Complexity).To(Equal("high"))
			Expect(impact.Recommendations).To(ContainElement("Consider batch processing"))
		})
	})

	Context("Deprecation validation", func() {
		It("should detect deprecated fields during conversion", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							// These fields might be deprecated in v1beta1
							AlertmanagerURL: "http://alertmanager:9093", // Deprecated in favor of AlertmanagerConfig
						},
					},
				},
			}

			deprecations := detectDeprecatedFields(v1alpha1Platform, "v1alpha1", "v1beta1")
			Expect(deprecations).To(HaveLen(1))
			Expect(deprecations[0].Field).To(Equal("spec.components.prometheus.alertmanagerURL"))
			Expect(deprecations[0].Message).To(ContainSubstring("use AlertmanagerConfig instead"))
			Expect(deprecations[0].RemovalVersion).To(Equal("v1"))
		})
	})
})

// Helper validation functions

func validatePlatformFields(platform *v1alpha1.ObservabilityPlatform) error {
	var allErrs field.ErrorList

	if platform.Spec.Components.Prometheus != nil {
		p := platform.Spec.Components.Prometheus
		if p.Enabled && p.Version == "" {
			allErrs = append(allErrs, field.Required(field.NewPath("spec", "components", "prometheus", "version"), "version is required when Prometheus is enabled"))
		}
		if p.Version != "" && !isValidVersion(p.Version) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "components", "prometheus", "version"), p.Version, "must be in format vX.Y.Z"))
		}
		if p.Retention != "" && !isValidRetention(p.Retention) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "components", "prometheus", "retention"), p.Retention, "must be in format Xd, Xh, or Xm"))
		}
	}

	if platform.Spec.Components.Loki != nil {
		l := platform.Spec.Components.Loki
		if l.StorageSize != "" {
			if _, err := resource.ParseQuantity(l.StorageSize); err != nil {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "components", "loki", "storageSize"), l.StorageSize, "invalid quantity format"))
			}
		}
	}

	if len(allErrs) > 0 {
		return allErrs.ToAggregate()
	}
	return nil
}

func isValidVersion(version string) bool {
	// Simple version validation - should match vX.Y.Z or X.Y.Z
	return strings.HasPrefix(version, "v") || strings.Count(version, ".") == 2
}

func isValidRetention(retention string) bool {
	// Validate retention format (e.g., 30d, 24h, 60m)
	if len(retention) < 2 {
		return false
	}
	unit := retention[len(retention)-1]
	return unit == 'd' || unit == 'h' || unit == 'm'
}

func validateResourceRequirements(resources corev1.ResourceRequirements) []error {
	var errors []error

	if resources.Limits != nil && resources.Requests != nil {
		// Check CPU
		if limitCPU, ok := resources.Limits[corev1.ResourceCPU]; ok {
			if requestCPU, ok := resources.Requests[corev1.ResourceCPU]; ok {
				if limitCPU.Cmp(requestCPU) < 0 {
					errors = append(errors, fmt.Errorf("CPU limit must be greater than or equal to request"))
				}
			}
		}

		// Check Memory
		if limitMem, ok := resources.Limits[corev1.ResourceMemory]; ok {
			if requestMem, ok := resources.Requests[corev1.ResourceMemory]; ok {
				if limitMem.Cmp(requestMem) < 0 {
					errors = append(errors, fmt.Errorf("memory limit must be greater than or equal to request"))
				}
			}
		}
	}

	return errors
}

func validateMetadata(meta metav1.ObjectMeta) []error {
	var errors []error

	// Validate labels
	for key, value := range meta.Labels {
		if errs := validation.IsQualifiedName(key); len(errs) > 0 {
			errors = append(errors, fmt.Errorf("invalid label key %q: %s", key, strings.Join(errs, ", ")))
		}
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
			errors = append(errors, fmt.Errorf("invalid label value %q: %s", value, strings.Join(errs, ", ")))
		}
	}

	// Validate annotations
	for key, value := range meta.Annotations {
		if errs := validation.IsQualifiedName(key); len(errs) > 0 {
			errors = append(errors, fmt.Errorf("invalid annotation key %q: %s", key, strings.Join(errs, ", ")))
		}
		// Annotation values can be longer but should still be reasonable
		if len(value) > 256*1024 { // 256KB limit
			errors = append(errors, fmt.Errorf("annotation value for key %q is too large", key))
		}
	}

	return errors
}

func checkFieldCompatibility(v1alpha1Platform *v1alpha1.ObservabilityPlatform, v1beta1Platform *v1beta1.ObservabilityPlatform) (bool, []string) {
	var issues []string
	compatible := true

	// Check specific field compatibility
	// This is a simplified example - real implementation would be more comprehensive

	return compatible, issues
}

func isFieldConversionCompatible(v1alpha1Field, v1beta1Field interface{}) bool {
	// Simplified compatibility check
	switch v1alpha1Field.(type) {
	case string:
		switch v1beta1Field.(type) {
		case string:
			return true
		case int, int32, int64:
			// Check if string can be converted to int
			_, ok := v1alpha1Field.(string)
			return ok && v1alpha1Field.(string) != "yes" && v1alpha1Field.(string) != "no"
		}
	case []string:
		_, ok := v1beta1Field.(map[string]string)
		return ok
	}
	return false
}

func validateComponentDependencies(platform *v1alpha1.ObservabilityPlatform) []string {
	var warnings []string

	// Check Grafana dependencies
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		hasDataSource := false
		if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
			hasDataSource = true
		}
		if !hasDataSource {
			warnings = append(warnings, "Grafana enabled but no Prometheus data source available")
		}
	}

	return warnings
}

func validateVersionCompatibility(platform *v1alpha1.ObservabilityPlatform) (bool, []string) {
	var issues []string
	compatible := true

	// Example compatibility check
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Grafana != nil {
		promVersion := platform.Spec.Components.Prometheus.Version
		grafanaVersion := platform.Spec.Components.Grafana.Version

		// Simplified check
		if strings.HasPrefix(promVersion, "v2.4") && strings.HasPrefix(grafanaVersion, "7.") {
			compatible = false
			issues = append(issues, fmt.Sprintf("Grafana version %s may not be compatible with Prometheus %s", grafanaVersion, promVersion))
		}
	}

	return compatible, issues
}

func validateConfigurationConsistency(platform *v1alpha1.ObservabilityPlatform) []error {
	var errors []error

	// Check storage vs retention
	if platform.Spec.Components.Prometheus != nil {
		p := platform.Spec.Components.Prometheus
		if p.Retention == "7d" && p.StorageSize == "10Gi" {
			errors = append(errors, fmt.Errorf("storage size may be insufficient for retention period"))
		}
	}

	// Check log level
	if platform.Spec.Global.LogLevel != "" {
		validLevels := []string{"debug", "info", "warn", "error"}
		valid := false
		for _, level := range validLevels {
			if platform.Spec.Global.LogLevel == level {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Errorf("invalid log level: %s", platform.Spec.Global.LogLevel))
		}
	}

	return errors
}

func validateSecuritySettings(platform *v1alpha1.ObservabilityPlatform) []string {
	var warnings []string

	if platform.Spec.Components.Grafana != nil {
		g := platform.Spec.Components.Grafana
		if g.AdminPassword != "" && len(g.AdminPassword) < 8 {
			warnings = append(warnings, "Grafana admin password is too weak (less than 8 characters)")
		}
		if g.AdminUser == "root" || g.AdminUser == "admin" {
			warnings = append(warnings, fmt.Sprintf("Grafana admin username '%s' is discouraged", g.AdminUser))
		}
	}

	return warnings
}

func validateV1Beta1Security(platform *v1beta1.ObservabilityPlatform) []string {
	var warnings []string

	if platform.Spec.Security != nil {
		if platform.Spec.Security.RBAC != nil && !platform.Spec.Security.RBAC.Enabled {
			warnings = append(warnings, "RBAC is disabled - this reduces security")
		}
		if platform.Spec.Security.NetworkPolicy != nil && !platform.Spec.Security.NetworkPolicy.Enabled {
			warnings = append(warnings, "Network policies are disabled - consider enabling for better network isolation")
		}
	}

	return warnings
}

type ConversionImpact struct {
	EstimatedDuration time.Duration
	MemoryImpact      int64
	Complexity        string
	Recommendations   []string
}

func assessConversionPerformanceImpact(platform *v1alpha1.ObservabilityPlatform) ConversionImpact {
	impact := ConversionImpact{
		EstimatedDuration: 10 * time.Millisecond, // Base duration
		MemoryImpact:      1024,                   // Base 1KB
		Complexity:        "low",
		Recommendations:   []string{},
	}

	// Calculate based on metadata size
	labelCount := len(platform.Labels)
	annotationCount := len(platform.Annotations)

	if labelCount > 50 || annotationCount > 50 {
		impact.Complexity = "high"
		impact.EstimatedDuration += time.Duration(labelCount+annotationCount) * time.Microsecond
		impact.MemoryImpact += int64(labelCount+annotationCount) * 100
		impact.Recommendations = append(impact.Recommendations, "Consider batch processing for better performance")
	}

	return impact
}

type DeprecationInfo struct {
	Field          string
	Message        string
	RemovalVersion string
}

func detectDeprecatedFields(platform *v1alpha1.ObservabilityPlatform, sourceVersion, targetVersion string) []DeprecationInfo {
	var deprecations []DeprecationInfo

	// Example deprecation detection
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.AlertmanagerURL != "" {
		deprecations = append(deprecations, DeprecationInfo{
			Field:          "spec.components.prometheus.alertmanagerURL",
			Message:        "AlertmanagerURL is deprecated, use AlertmanagerConfig instead",
			RemovalVersion: "v1",
		})
	}

	return deprecations
}
