package cost

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestCostManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cost Manager Suite")
}

var _ = Describe("Cost Manager", func() {
	var (
		ctx        context.Context
		k8sClient  client.Client
		scheme     *runtime.Scheme
		manager    *Manager
		platform   *observabilityv1.ObservabilityPlatform
		namespace  string
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(observabilityv1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		namespace = "test-namespace"
		platform = &observabilityv1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: namespace,
			},
			Spec: observabilityv1.ObservabilityPlatformSpec{
				Components: observabilityv1.Components{
					Prometheus: &observabilityv1.PrometheusSpec{
						Enabled: true,
						Version: "v2.48.0",
						Replicas: 2,
						Resources: &corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1"),
								corev1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
						Storage: &observabilityv1.StorageSpec{
							Size:             "100Gi",
							StorageClassName: "standard",
						},
					},
					Grafana: &observabilityv1.GrafanaSpec{
						Enabled: true,
						Version: "10.2.0",
						Replicas: 2,
						Resources: &corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				},
				CostOptimization: &observabilityv1.CostOptimizationSpec{
					Enabled:              true,
					ResourceOptimization: true,
					AutoScaling:          true,
					SpotInstances: &observabilityv1.SpotInstancesConfig{
						Enabled: true,
						AllowedComponents: []string{"grafana"},
						MaxSpotPercentage: 70,
					},
					Budget: &observabilityv1.BudgetConfig{
						MonthlyLimit:    1000.0,
						AlertThresholds: []int{80, 90, 100},
					},
				},
			},
		}

		// Create fake client with initial objects
		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(platform).
			Build()

		// Create manager
		manager = NewManager(k8sClient, scheme, log.Log)
	})

	Describe("Cost Analysis", func() {
		It("should analyze platform costs", func() {
			// Create some pods to analyze
			prometheusPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-0",
					Namespace: namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":       "prometheus",
						"app.kubernetes.io/instance":   platform.Name,
						"app.kubernetes.io/managed-by": "gunj-operator",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "prometheus",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, prometheusPod)).To(Succeed())

			prometheusPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-data-0",
					Namespace: namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name":       "prometheus",
						"app.kubernetes.io/instance":   platform.Name,
						"app.kubernetes.io/managed-by": "gunj-operator",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
					StorageClassName: stringPtr("standard"),
				},
			}
			Expect(k8sClient.Create(ctx, prometheusPVC)).To(Succeed())

			// Analyze costs
			analysis, err := manager.AnalyzePlatformCosts(ctx, platform)
			Expect(err).NotTo(HaveOccurred())
			Expect(analysis).NotTo(BeNil())
			Expect(analysis.Platform).To(Equal(platform.Name))
			Expect(analysis.Namespace).To(Equal(namespace))
			Expect(analysis.TotalCost).To(BeNumerically(">", 0))
			Expect(analysis.Currency).To(Equal("USD"))
			Expect(analysis.Components).To(HaveKey("prometheus"))
		})
	})

	Describe("Recommendations", func() {
		It("should generate optimization recommendations", func() {
			// Create resource usage data
			usage := map[string]*ResourceUsage{
				"prometheus": {
					Component:   "prometheus",
					Timestamp:   time.Now(),
					CPUUsage:    *resource.NewMilliQuantity(500, resource.DecimalSI), // 50% of requested
					MemoryUsage: *resource.NewQuantity(2*1024*1024*1024, resource.BinarySI), // 2Gi of 4Gi
					Percentiles: ResourceUsagePercentiles{
						CPU: UsagePercentiles{
							P50:  0.4,
							P90:  0.5,
							P95:  0.6,
							P99:  0.7,
							Max:  0.8,
							Mean: 0.5,
						},
						Memory: UsagePercentiles{
							P50:  1.5 * 1024 * 1024 * 1024,
							P90:  2.0 * 1024 * 1024 * 1024,
							P95:  2.5 * 1024 * 1024 * 1024,
							P99:  3.0 * 1024 * 1024 * 1024,
							Max:  3.5 * 1024 * 1024 * 1024,
							Mean: 2.0 * 1024 * 1024 * 1024,
						},
					},
				},
			}

			optimizer := NewCostOptimizer(log.Log)
			recommendations := optimizer.GenerateRecommendations(platform, usage)
			
			Expect(recommendations).NotTo(BeNil())
			Expect(recommendations.Platform).To(Equal(platform.Name))
			Expect(recommendations.ResourceRecommendations).To(HaveKey("prometheus"))
			
			// Should recommend lower resources since usage is 50%
			promRec := recommendations.ResourceRecommendations["prometheus"].(*ResourceRecommendation)
			Expect(promRec.EstimatedSavings).To(BeNumerically(">", 0))
		})

		It("should identify spot instance candidates", func() {
			usage := map[string]*ResourceUsage{
				"grafana": {
					Component: "grafana",
					Timestamp: time.Now(),
					Percentiles: ResourceUsagePercentiles{
						CPU: UsagePercentiles{
							P50:  0.3,
							P99:  0.4,
							Mean: 0.35,
						},
					},
				},
			}

			optimizer := NewCostOptimizer(log.Log)
			recommendations := optimizer.GenerateRecommendations(platform, usage)
			
			Expect(recommendations.SpotInstanceCandidates).To(HaveKey("grafana"))
			Expect(recommendations.SpotInstanceCandidates["grafana"]).To(BeTrue())
		})
	})

	Describe("Budget Alerts", func() {
		It("should create budget monitoring rules", func() {
			budgetConfig := &BudgetConfig{
				MonthlyLimit:   1000.0,
				AlertThreshold: 80.0,
			}

			err := manager.SetBudgetAlert(ctx, platform, budgetConfig)
			Expect(err).NotTo(HaveOccurred())

			// Check that PrometheusRule was created
			rules := &observabilityv1.PrometheusRuleList{}
			err = k8sClient.List(ctx, rules, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(rules.Items).To(HaveLen(1))
			
			rule := rules.Items[0]
			Expect(rule.Name).To(Equal("test-platform-budget-alerts"))
			Expect(rule.Spec.Groups).To(HaveLen(1))
			Expect(rule.Spec.Groups[0].Rules).To(HaveLen(2)) // Warning and critical alerts
		})
	})

	Describe("Apply Optimizations", func() {
		It("should apply resource optimizations", func() {
			// Create recommendations
			recommendations := &OptimizationRecommendations{
				ResourceRecommendations: map[string]interface{}{
					"prometheus": &ResourceRecommendation{
						Component: "prometheus",
						Current: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1"),
								corev1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
						Recommended: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("750m"),
								corev1.ResourceMemory: resource.MustParse("3Gi"),
							},
						},
						EstimatedSavings: 50.0,
					},
				},
			}

			opts := &OptimizationOptions{
				ApplyResourceOptimization: true,
			}

			err := manager.ApplyOptimizations(ctx, platform, opts)
			Expect(err).NotTo(HaveOccurred())

			// Verify platform was updated
			updated := &observabilityv1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(platform), updated)
			Expect(err).NotTo(HaveOccurred())
			
			Expect(updated.Spec.Components.Prometheus.Resources.Requests[corev1.ResourceCPU]).To(
				Equal(resource.MustParse("750m")))
			Expect(updated.Status.CostOptimization.Applied).To(BeTrue())
		})

		It("should configure spot instances", func() {
			recommendations := &OptimizationRecommendations{
				SpotInstanceCandidates: map[string]bool{
					"grafana": true,
				},
			}

			opts := &OptimizationOptions{
				EnableSpotInstances: true,
			}

			err := manager.ApplyOptimizations(ctx, platform, opts)
			Expect(err).NotTo(HaveOccurred())

			// Verify Grafana has spot instance configuration
			updated := &observabilityv1.ObservabilityPlatform{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(platform), updated)
			Expect(err).NotTo(HaveOccurred())
			
			Expect(updated.Spec.Components.Grafana.NodeSelector).To(HaveKeyWithValue(
				"node.kubernetes.io/instance-type", "spot"))
			Expect(updated.Spec.Components.Grafana.Tolerations).To(HaveLen(1))
		})
	})
})

func stringPtr(s string) *string {
	return &s
}
