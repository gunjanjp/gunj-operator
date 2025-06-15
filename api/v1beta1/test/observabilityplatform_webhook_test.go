/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("ObservabilityPlatform Webhook", func() {
	Context("When creating ObservabilityPlatform", func() {
		It("should accept valid configurations", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1,
							Resources: &v1beta1.ResourceRequirements{
								Requests: &v1beta1.ResourceList{
									CPU:    "100m",
									Memory: "256Mi",
								},
								Limits: &v1beta1.ResourceList{
									CPU:    "500m",
									Memory: "1Gi",
								},
							},
						},
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(BeNil())
			Expect(warnings).To(BeEmpty())
		})

		It("should reject when no components are enabled", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: false,
						},
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			statusErr, ok := err.(*errors.StatusError)
			Expect(ok).To(BeTrue())
			Expect(statusErr.ErrStatus.Details.Causes).To(HaveLen(1))
			Expect(statusErr.ErrStatus.Details.Causes[0].Message).To(ContainSubstring("at least one component must be enabled"))
		})

		It("should reject invalid version format", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "invalid-version",
						},
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			statusErr, ok := err.(*errors.StatusError)
			Expect(ok).To(BeTrue())
			Expect(statusErr.ErrStatus.Details.Causes[0].Field).To(Equal("spec.components.prometheus.version"))
		})

		It("should reject invalid resource quantities", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
							Resources: &v1beta1.ResourceRequirements{
								Requests: &v1beta1.ResourceList{
									CPU:    "invalid",
									Memory: "256Mi",
								},
							},
						},
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("should reject when requests exceed limits", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
							Resources: &v1beta1.ResourceRequirements{
								Requests: &v1beta1.ResourceList{
									CPU:    "2",
									Memory: "2Gi",
								},
								Limits: &v1beta1.ResourceList{
									CPU:    "1",
									Memory: "1Gi",
								},
							},
						},
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("should validate S3 configuration when enabled", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Loki: &v1beta1.LokiSpec{
							Enabled: true,
							Version: "2.9.0",
							Storage: &v1beta1.LokiStorageSpec{
								StorageSpec: v1beta1.StorageSpec{
									Size: "10Gi",
								},
								S3: &v1beta1.S3StorageSpec{
									Enabled: true,
									// Missing required fields
								},
							},
						},
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			statusErr, ok := err.(*errors.StatusError)
			Expect(ok).To(BeTrue())
			// Should have errors for missing bucket name and region
			Expect(len(statusErr.ErrStatus.Details.Causes)).To(BeNumerically(">=", 2))
		})

		It("should validate high availability settings", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.48.0",
							Replicas: 1, // Should be at least 2 for HA
						},
					},
					HighAvailability: &v1beta1.HighAvailabilitySettings{
						Enabled: true,
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			statusErr, ok := err.(*errors.StatusError)
			Expect(ok).To(BeTrue())
			Expect(statusErr.ErrStatus.Details.Causes[0].Message).To(ContainSubstring("must have at least 2 replicas when HA is enabled"))
		})

		It("should validate backup settings", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
					},
					Backup: &v1beta1.BackupSettings{
						Enabled:   true,
						Schedule:  "invalid-cron",
						Retention: 0, // Should be at least 1
					},
				},
			}

			warnings, err := platform.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			statusErr, ok := err.(*errors.StatusError)
			Expect(ok).To(BeTrue())
			// Should have errors for invalid cron and retention
			Expect(len(statusErr.ErrStatus.Details.Causes)).To(BeNumerically(">=", 2))
		})
	})

	Context("When updating ObservabilityPlatform", func() {
		It("should allow valid updates", func() {
			oldPlatform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled:  true,
							Version:  "v2.47.0",
							Replicas: 1,
						},
					},
				},
			}

			newPlatform := oldPlatform.DeepCopy()
			newPlatform.Spec.Components.Prometheus.Version = "v2.48.0"

			warnings, err := newPlatform.ValidateUpdate(oldPlatform)
			Expect(err).To(BeNil())
			Expect(warnings).To(BeEmpty())
		})

		It("should warn on version downgrade", func() {
			oldPlatform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
					},
				},
			}

			newPlatform := oldPlatform.DeepCopy()
			newPlatform.Spec.Components.Prometheus.Version = "v2.47.0"

			warnings, err := newPlatform.ValidateUpdate(oldPlatform)
			Expect(err).To(BeNil())
			Expect(warnings).To(HaveLen(1))
			Expect(warnings[0]).To(ContainSubstring("version downgrade detected"))
		})
	})

	Context("When deleting ObservabilityPlatform", func() {
		It("should allow deletion", func() {
			platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
			}

			warnings, err := platform.ValidateDelete()
			Expect(err).To(BeNil())
			Expect(warnings).To(BeEmpty())
		})
	})
})

// Unit tests for helper functions
func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"valid with v prefix", "v2.48.0", true},
		{"valid without v prefix", "2.48.0", true},
		{"valid with pre-release", "v2.48.0-rc.1", true},
		{"invalid format", "invalid", false},
		{"missing patch version", "v2.48", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We would test the actual isValidVersion function here
			// For now, this is a placeholder
		})
	}
}

func TestIsValidRetention(t *testing.T) {
	tests := []struct {
		name      string
		retention string
		expected  bool
	}{
		{"valid days", "30d", true},
		{"valid hours", "12h", true},
		{"valid weeks", "4w", true},
		{"valid months", "6m", true},
		{"valid years", "1y", true},
		{"invalid format", "30days", false},
		{"missing unit", "30", false},
		{"invalid unit", "30x", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We would test the actual isValidRetention function here
			// For now, this is a placeholder
		})
	}
}
