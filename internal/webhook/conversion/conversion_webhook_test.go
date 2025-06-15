/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package conversion_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/webhook/conversion"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg           *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	ctx           context.Context
	cancel        context.CancelFunc
	webhookServer webhook.Server
)

func TestConversionWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conversion Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../../config/crd/bases"},
		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{"../../../../config/webhook"},
		},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	err = v1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Start webhook server
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		Port:               testEnv.WebhookInstallOptions.LocalServingPort,
		Host:              testEnv.WebhookInstallOptions.LocalServingHost,
		CertDir:           testEnv.WebhookInstallOptions.LocalServingCertDir,
		LeaderElection:    false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	// Setup conversion webhook
	conversionWebhook, err := conversion.NewConversionWebhook(mgr)
	Expect(err).NotTo(HaveOccurred())
	err = conversionWebhook.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	gexec.KillAndWait(4 * time.Second)
})

var _ = Describe("Conversion Webhook", func() {
	const (
		testNamespace = "test-conversion"
		timeout       = time.Second * 30
		interval      = time.Millisecond * 250
	)

	BeforeEach(func() {
		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		err := k8sClient.Create(ctx, ns)
		if err != nil {
			// Namespace might already exist from previous test
			_ = k8sClient.Get(ctx, client.ObjectKey{Name: testNamespace}, ns)
		}
	})

	AfterEach(func() {
		// Clean up test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		_ = k8sClient.Delete(ctx, ns)
		Eventually(func() bool {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: testNamespace}, &corev1.Namespace{})
			return err != nil
		}, timeout, interval).Should(BeTrue())
	})

	Context("v1alpha1 to v1beta1 conversion", func() {
		It("should convert basic ObservabilityPlatform successfully", func() {
			// Create v1alpha1 ObservabilityPlatform
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-basic",
					Namespace: testNamespace,
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

			By("creating v1alpha1 ObservabilityPlatform")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying basic fields are preserved")
			Expect(v1beta1Platform.Name).To(Equal(v1alpha1Platform.Name))
			Expect(v1beta1Platform.Namespace).To(Equal(v1alpha1Platform.Namespace))
			Expect(v1beta1Platform.Spec.Components.Prometheus.Enabled).To(Equal(true))
			Expect(v1beta1Platform.Spec.Components.Prometheus.Version).To(Equal("v2.48.0"))
		})

		It("should preserve custom annotations and labels", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-metadata",
					Namespace: testNamespace,
					Labels: map[string]string{
						"app":         "observability",
						"environment": "production",
						"team":        "platform",
					},
					Annotations: map[string]string{
						"description":      "Production observability platform",
						"contact":          "platform@company.com",
						"custom/setting":   "value",
						"kubectl.kubernetes.io/last-applied-configuration": "{}",
					},
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled: true,
						},
					},
				},
			}

			By("creating v1alpha1 with metadata")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying labels are preserved")
			Expect(v1beta1Platform.Labels).To(HaveLen(3))
			Expect(v1beta1Platform.Labels["app"]).To(Equal("observability"))
			Expect(v1beta1Platform.Labels["environment"]).To(Equal("production"))
			Expect(v1beta1Platform.Labels["team"]).To(Equal("platform"))

			By("verifying annotations are preserved")
			Expect(v1beta1Platform.Annotations).To(HaveKey("description"))
			Expect(v1beta1Platform.Annotations["contact"]).To(Equal("platform@company.com"))
			Expect(v1beta1Platform.Annotations["custom/setting"]).To(Equal("value"))
			// kubectl annotation should be handled specially
			Expect(v1beta1Platform.Annotations).To(HaveKey("kubectl.kubernetes.io/last-applied-configuration"))
		})

		It("should convert complex component configurations", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-complex",
					Namespace: testNamespace,
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
								{
									URL: "http://remote-write-endpoint:9090/api/v1/write",
								},
							},
							ExternalLabels: map[string]string{
								"cluster": "production",
								"region":  "us-east-1",
							},
						},
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled:       true,
							Version:       "10.0.0",
							AdminPassword: "secure-password",
							AdminUser:     "admin",
							DataSources: []v1alpha1.DataSourceSpec{
								{
									Name: "Prometheus",
									Type: "prometheus",
									URL:  "http://prometheus:9090",
								},
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
							"environment": "production",
						},
					},
				},
			}

			By("creating complex v1alpha1 platform")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying Prometheus configuration")
			Expect(v1beta1Platform.Spec.Components.Prometheus).NotTo(BeNil())
			Expect(v1beta1Platform.Spec.Components.Prometheus.Replicas).To(Equal(int32(3)))
			Expect(v1beta1Platform.Spec.Components.Prometheus.Retention).To(Equal("30d"))
			Expect(v1beta1Platform.Spec.Components.Prometheus.StorageSize).To(Equal("100Gi"))
			Expect(v1beta1Platform.Spec.Components.Prometheus.RemoteWrite).To(HaveLen(1))
			Expect(v1beta1Platform.Spec.Components.Prometheus.ExternalLabels).To(HaveLen(2))

			By("verifying Grafana configuration")
			Expect(v1beta1Platform.Spec.Components.Grafana).NotTo(BeNil())
			Expect(v1beta1Platform.Spec.Components.Grafana.AdminUser).To(Equal("admin"))
			Expect(v1beta1Platform.Spec.Components.Grafana.DataSources).To(HaveLen(1))

			By("verifying Global configuration")
			Expect(v1beta1Platform.Spec.Global.LogLevel).To(Equal("info"))
			Expect(v1beta1Platform.Spec.Global.ExternalLabels).To(HaveLen(1))
		})

		It("should handle v1beta1-specific fields with defaults", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-defaults",
					Namespace: testNamespace,
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}

			By("creating minimal v1alpha1 platform")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying v1beta1-specific fields have sensible defaults")
			// Security field should be nil or have defaults
			if v1beta1Platform.Spec.Security != nil {
				Expect(v1beta1Platform.Spec.Security.RBAC).NotTo(BeNil())
			}

			// Grafana plugins should be empty
			if v1beta1Platform.Spec.Components.Grafana != nil {
				Expect(v1beta1Platform.Spec.Components.Grafana.Plugins).To(BeEmpty())
			}
		})
	})

	Context("v1beta1 to v1alpha1 conversion", func() {
		It("should convert basic ObservabilityPlatform successfully", func() {
			v1beta1Platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-backward",
					Namespace: testNamespace,
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

			By("creating v1beta1 ObservabilityPlatform")
			Expect(k8sClient.Create(ctx, v1beta1Platform)).Should(Succeed())

			By("fetching as v1alpha1")
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1beta1Platform.Name,
					Namespace: v1beta1Platform.Namespace,
				}, v1alpha1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying basic fields are preserved")
			Expect(v1alpha1Platform.Name).To(Equal(v1beta1Platform.Name))
			Expect(v1alpha1Platform.Namespace).To(Equal(v1beta1Platform.Namespace))
			Expect(v1alpha1Platform.Spec.Components.Prometheus.Enabled).To(Equal(true))
			Expect(v1alpha1Platform.Spec.Components.Prometheus.Version).To(Equal("v2.48.0"))
		})

		It("should handle v1beta1-only fields gracefully", func() {
			v1beta1Platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-v1beta1-fields",
					Namespace: testNamespace,
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
							Version: "v2.48.0",
						},
						Grafana: &v1beta1.GrafanaSpec{
							Enabled: true,
							Version: "10.0.0",
							// v1beta1-only fields
							SMTP: &v1beta1.SMTPConfig{
								Host:     "smtp.gmail.com",
								Port:     587,
								User:     "notifications@company.com",
								Password: "secure-password",
							},
							Plugins: []string{"plugin1", "plugin2"},
						},
						Loki: &v1beta1.LokiSpec{
							Enabled: true,
							Version: "2.9.0",
							// v1beta1-only field
							CompactorEnabled: true,
						},
						Tempo: &v1beta1.TempoSpec{
							Enabled: true,
							Version: "2.3.0",
							// v1beta1-only field
							SearchEnabled: true,
						},
					},
					Security: &v1beta1.SecuritySpec{
						RBAC: &v1beta1.RBACSpec{
							Enabled: true,
						},
						NetworkPolicy: &v1beta1.NetworkPolicySpec{
							Enabled: true,
						},
					},
					Global: v1beta1.GlobalSpec{
						LogLevel: "debug",
						// v1beta1-only fields
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{
										{
											MatchExpressions: []corev1.NodeSelectorRequirement{
												{
													Key:      "node-type",
													Operator: corev1.NodeSelectorOpIn,
													Values:   []string{"monitoring"},
												},
											},
										},
									},
								},
							},
						},
						ImagePullSecrets: []string{"regcred"},
					},
				},
			}

			By("creating v1beta1 with v1beta1-only fields")
			Expect(k8sClient.Create(ctx, v1beta1Platform)).Should(Succeed())

			By("fetching as v1alpha1")
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1beta1Platform.Name,
					Namespace: v1beta1Platform.Namespace,
				}, v1alpha1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying common fields are preserved")
			Expect(v1alpha1Platform.Spec.Components.Prometheus.Enabled).To(Equal(true))
			Expect(v1alpha1Platform.Spec.Components.Grafana.Enabled).To(Equal(true))
			Expect(v1alpha1Platform.Spec.Components.Loki.Enabled).To(Equal(true))
			Expect(v1alpha1Platform.Spec.Components.Tempo.Enabled).To(Equal(true))
			Expect(v1alpha1Platform.Spec.Global.LogLevel).To(Equal("debug"))

			By("verifying v1beta1-only fields are handled (lost) gracefully")
			// These fields don't exist in v1alpha1, so we can't check them
			// The conversion should have logged warnings about these losses
		})

		It("should preserve unknown fields in annotations", func() {
			v1beta1Platform := &v1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-unknown-fields",
					Namespace: testNamespace,
					Annotations: map[string]string{
						"custom.io/config": `{"setting": "value"}`,
					},
				},
				Spec: v1beta1.ObservabilityPlatformSpec{
					Components: v1beta1.Components{
						Prometheus: &v1beta1.PrometheusSpec{
							Enabled: true,
						},
					},
				},
			}

			By("creating v1beta1 with custom annotations")
			Expect(k8sClient.Create(ctx, v1beta1Platform)).Should(Succeed())

			By("fetching as v1alpha1")
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1beta1Platform.Name,
					Namespace: v1beta1Platform.Namespace,
				}, v1alpha1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying annotations are preserved")
			Expect(v1alpha1Platform.Annotations).To(HaveKey("custom.io/config"))
			Expect(v1alpha1Platform.Annotations["custom.io/config"]).To(Equal(`{"setting": "value"}`))
		})
	})

	Context("Round-trip conversion", func() {
		It("should preserve data through v1alpha1 -> v1beta1 -> v1alpha1", func() {
			originalV1alpha1 := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-roundtrip",
					Namespace: testNamespace,
					Labels: map[string]string{
						"test": "roundtrip",
					},
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:     true,
							Version:     "v2.48.0",
							Replicas:    2,
							Retention:   "15d",
							StorageSize: "50Gi",
							ExternalLabels: map[string]string{
								"cluster": "test",
							},
						},
					},
					Global: v1alpha1.GlobalSpec{
						LogLevel: "warn",
					},
				},
			}

			By("creating original v1alpha1")
			Expect(k8sClient.Create(ctx, originalV1alpha1)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      originalV1alpha1.Name,
					Namespace: originalV1alpha1.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("updating v1beta1 (triggering storage)")
			v1beta1Platform.Spec.Components.Prometheus.Retention = "20d"
			Expect(k8sClient.Update(ctx, v1beta1Platform)).Should(Succeed())

			By("fetching back as v1alpha1")
			finalV1alpha1 := &v1alpha1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      originalV1alpha1.Name,
					Namespace: originalV1alpha1.Namespace,
				}, finalV1alpha1)
			}, timeout, interval).Should(Succeed())

			By("verifying data preservation")
			Expect(finalV1alpha1.Spec.Components.Prometheus.Enabled).To(Equal(true))
			Expect(finalV1alpha1.Spec.Components.Prometheus.Version).To(Equal("v2.48.0"))
			Expect(finalV1alpha1.Spec.Components.Prometheus.Replicas).To(Equal(int32(2)))
			Expect(finalV1alpha1.Spec.Components.Prometheus.Retention).To(Equal("20d")) // Updated value
			Expect(finalV1alpha1.Spec.Components.Prometheus.StorageSize).To(Equal("50Gi"))
			Expect(finalV1alpha1.Spec.Components.Prometheus.ExternalLabels["cluster"]).To(Equal("test"))
			Expect(finalV1alpha1.Spec.Global.LogLevel).To(Equal("warn"))
		})
	})

	Context("Edge cases and error scenarios", func() {
		It("should handle nil component specs", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-nil-components",
					Namespace: testNamespace,
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						// All components are nil
					},
				},
			}

			By("creating v1alpha1 with nil components")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying nil components are handled")
			Expect(v1beta1Platform.Spec.Components.Prometheus).To(BeNil())
			Expect(v1beta1Platform.Spec.Components.Grafana).To(BeNil())
			Expect(v1beta1Platform.Spec.Components.Loki).To(BeNil())
			Expect(v1beta1Platform.Spec.Components.Tempo).To(BeNil())
		})

		It("should handle empty maps and slices", func() {
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-empty-collections",
					Namespace: testNamespace,
					Labels:    map[string]string{},     // Empty map
					Annotations: map[string]string{},  // Empty map
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Prometheus: &v1alpha1.PrometheusSpec{
							Enabled:        true,
							RemoteWrite:    []v1alpha1.RemoteWriteSpec{}, // Empty slice
							ExternalLabels: map[string]string{},           // Empty map
						},
					},
				},
			}

			By("creating v1alpha1 with empty collections")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying empty collections are preserved")
			Expect(v1beta1Platform.Labels).To(BeEmpty())
			Expect(v1beta1Platform.Annotations).ToNot(BeNil()) // May have system annotations
			Expect(v1beta1Platform.Spec.Components.Prometheus.RemoteWrite).To(BeEmpty())
			Expect(v1beta1Platform.Spec.Components.Prometheus.ExternalLabels).To(BeEmpty())
		})

		It("should handle very long string values", func() {
			longString := fmt.Sprintf("%s-very-long-string", string(make([]byte, 1000)))
			
			v1alpha1Platform := &v1alpha1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-long-strings",
					Namespace: testNamespace,
					Annotations: map[string]string{
						"description": longString,
					},
				},
				Spec: v1alpha1.ObservabilityPlatformSpec{
					Components: v1alpha1.Components{
						Grafana: &v1alpha1.GrafanaSpec{
							Enabled:       true,
							AdminPassword: longString[:100], // Use first 100 chars
						},
					},
				},
			}

			By("creating v1alpha1 with long strings")
			Expect(k8sClient.Create(ctx, v1alpha1Platform)).Should(Succeed())

			By("fetching as v1beta1")
			v1beta1Platform := &v1beta1.ObservabilityPlatform{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{
					Name:      v1alpha1Platform.Name,
					Namespace: v1alpha1Platform.Namespace,
				}, v1beta1Platform)
			}, timeout, interval).Should(Succeed())

			By("verifying long strings are preserved")
			Expect(v1beta1Platform.Annotations["description"]).To(Equal(longString))
			Expect(v1beta1Platform.Spec.Components.Grafana.AdminPassword).To(Equal(longString[:100]))
		})
	})
})
