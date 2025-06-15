/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var _ = Describe("ObservabilityPlatform Webhook", func() {
	var (
		webhook  *ObservabilityPlatformWebhook
		platform *observabilityv1beta1.ObservabilityPlatform
		ctx      context.Context
	)

	BeforeEach(func() {
		webhook = &ObservabilityPlatformWebhook{}
		ctx = context.Background()
		platform = &observabilityv1beta1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
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
	})

	Describe("Default", func() {
		Context("when creating a platform with minimal configuration", func() {
			It("should set default values for Prometheus", func() {
				err := webhook.Default(ctx, platform)
				Expect(err).NotTo(HaveOccurred())

				Expect(platform.Spec.Components.Prometheus.Replicas).To(Equal(int32(1)))
				Expect(platform.Spec.Components.Prometheus.Retention).To(Equal("30d"))
				Expect(platform.Spec.Components.Prometheus.Storage.Size.String()).To(Equal("10Gi"))
				Expect(platform.Spec.Components.Prometheus.Resources.Requests).NotTo(BeNil())
				Expect(platform.Spec.Components.Prometheus.Resources.Limits).NotTo(BeNil())
			})
		})

		Context("when creating a platform with Grafana", func() {
			BeforeEach(func() {
				platform.Spec.Components.Grafana = &observabilityv1beta1.GrafanaSpec{
					Enabled: true,
					Version: "10.2.0",
				}
			})

			It("should set default values and generate admin password", func() {
				err := webhook.Default(ctx, platform)
				Expect(err).NotTo(HaveOccurred())

				Expect(platform.Spec.Components.Grafana.Replicas).To(Equal(int32(1)))
				Expect(platform.Spec.Components.Grafana.AdminPassword).NotTo(BeEmpty())
				Expect(platform.Spec.Components.Grafana.Resources.Requests).NotTo(BeNil())
				Expect(platform.Spec.Components.Grafana.Resources.Limits).NotTo(BeNil())
			})
		})

		Context("when HA is enabled", func() {
			BeforeEach(func() {
				platform.Spec.HighAvailability = &observabilityv1beta1.HighAvailabilityConfig{
					Enabled: true,
				}
			})

			It("should adjust replicas for HA mode", func() {
				err := webhook.Default(ctx, platform)
				Expect(err).NotTo(HaveOccurred())

				Expect(platform.Spec.HighAvailability.MinReplicas).To(Equal(int32(3)))
				Expect(platform.Spec.Components.Prometheus.Replicas).To(Equal(int32(3)))
			})
		})
	})

	Describe("ValidateCreate", func() {
		Context("when creating a valid platform", func() {
			It("should pass validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when no components are enabled", func() {
			BeforeEach(func() {
				platform.Spec.Components.Prometheus.Enabled = false
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("at least one component must be enabled"))
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when version format is invalid", func() {
			BeforeEach(func() {
				platform.Spec.Components.Prometheus.Version = "invalid-version"
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("version must follow semantic versioning"))
			})
		})

		Context("when resources are too low", func() {
			BeforeEach(func() {
				platform.Spec.Components.Prometheus.Resources = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("5m"),
						"memory": resource.MustParse("64Mi"),
					},
				}
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("CPU request must be at least 10m"))
				Expect(err.Error()).To(ContainSubstring("memory request must be at least 128Mi"))
			})
		})

		Context("when storage size is too small", func() {
			BeforeEach(func() {
				platform.Spec.Components.Prometheus.Storage = observabilityv1beta1.StorageSpec{
					Size: resource.MustParse("500Mi"),
				}
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("storage size must be at least 1Gi"))
			})
		})

		Context("when ingress is enabled without host", func() {
			BeforeEach(func() {
				platform.Spec.Components.Grafana = &observabilityv1beta1.GrafanaSpec{
					Enabled: true,
					Version: "10.2.0",
					Ingress: &observabilityv1beta1.IngressConfig{
						Enabled: true,
					},
				}
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("host is required when ingress is enabled"))
			})
		})

		Context("when backup is enabled with invalid schedule", func() {
			BeforeEach(func() {
				platform.Spec.Backup = &observabilityv1beta1.BackupConfig{
					Enabled:  true,
					Schedule: "invalid cron",
					Destination: observabilityv1beta1.BackupDestination{
						Type: "s3",
						S3: &observabilityv1beta1.S3Config{
							BucketName: "test-bucket",
							Region:     "us-east-1",
						},
					},
				}
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid cron schedule"))
			})
		})

		Context("when HA is enabled with insufficient replicas", func() {
			BeforeEach(func() {
				platform.Spec.HighAvailability = &observabilityv1beta1.HighAvailabilityConfig{
					Enabled:     true,
					MinReplicas: 2,
				}
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateCreate(ctx, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("minimum replicas for HA must be at least 3"))
			})
		})
	})

	Describe("ValidateUpdate", func() {
		var oldPlatform *observabilityv1beta1.ObservabilityPlatform

		BeforeEach(func() {
			oldPlatform = platform.DeepCopy()
			// Set some initial values
			oldPlatform.Spec.Components.Prometheus.Storage.StorageClassName = "fast-ssd"
		})

		Context("when updating with valid changes", func() {
			BeforeEach(func() {
				platform.Spec.Components.Prometheus.Replicas = 3
			})

			It("should pass validation", func() {
				warnings, err := webhook.ValidateUpdate(ctx, oldPlatform, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when changing immutable fields", func() {
			BeforeEach(func() {
				platform.Spec.Components.Prometheus.Storage.StorageClassName = "standard"
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateUpdate(ctx, oldPlatform, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("storage class cannot be changed once set"))
			})
		})

		Context("when downgrading version", func() {
			BeforeEach(func() {
				oldPlatform.Spec.Components.Prometheus.Version = "v2.48.0"
				platform.Spec.Components.Prometheus.Version = "v2.47.0"
			})

			It("should fail validation", func() {
				warnings, err := webhook.ValidateUpdate(ctx, oldPlatform, platform)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("downgrade from v2.48.0 to v2.47.0 is not supported"))
			})
		})

		Context("when doing major version upgrade", func() {
			BeforeEach(func() {
				oldPlatform.Spec.Components.Prometheus.Version = "v2.48.0"
				platform.Spec.Components.Prometheus.Version = "v3.0.0"
			})

			It("should warn about major version change", func() {
				warnings, err := webhook.ValidateUpdate(ctx, oldPlatform, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
				Expect(string(warnings[0])).To(ContainSubstring("Major version upgrade detected"))
			})
		})

		Context("when scaling down in HA mode", func() {
			BeforeEach(func() {
				oldPlatform.Spec.HighAvailability = &observabilityv1beta1.HighAvailabilityConfig{
					Enabled:     true,
					MinReplicas: 3,
				}
				oldPlatform.Spec.Components.Prometheus.Replicas = 3
				platform.Spec.HighAvailability = oldPlatform.Spec.HighAvailability
				platform.Spec.Components.Prometheus.Replicas = 2
			})

			It("should warn about scaling down", func() {
				warnings, err := webhook.ValidateUpdate(ctx, oldPlatform, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
				Expect(string(warnings[0])).To(ContainSubstring("Scaling down Prometheus below 3 replicas in HA mode"))
			})
		})
	})

	Describe("ValidateDelete", func() {
		Context("when deleting a platform with backups enabled", func() {
			BeforeEach(func() {
				platform.Spec.Backup = &observabilityv1beta1.BackupConfig{
					Enabled: true,
				}
			})

			It("should warn about data loss", func() {
				warnings, err := webhook.ValidateDelete(ctx, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
				Expect(string(warnings[0])).To(ContainSubstring("Platform test-platform has backups enabled"))
			})
		})

		Context("when deleting a platform with persistent storage", func() {
			BeforeEach(func() {
				platform.Spec.Components.Prometheus.Storage.Size = resource.MustParse("100Gi")
			})

			It("should warn about permanent data deletion", func() {
				warnings, err := webhook.ValidateDelete(ctx, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
				Expect(string(warnings[0])).To(ContainSubstring("Platform test-platform has persistent storage configured"))
			})
		})

		Context("when deleting a production platform", func() {
			BeforeEach(func() {
				platform.Labels = map[string]string{
					"environment": "production",
				}
			})

			It("should strongly warn about production deletion", func() {
				warnings, err := webhook.ValidateDelete(ctx, platform)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).NotTo(BeEmpty())
				Expect(string(warnings[0])).To(ContainSubstring("WARNING: You are deleting a PRODUCTION platform"))
			})
		})
	})
})

func TestObservabilityPlatformWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ObservabilityPlatform Webhook Suite")
}
