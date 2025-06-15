package cost

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Cost Analyzer", func() {
	var (
		analyzer *ResourceAnalyzer
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		analyzer = NewResourceAnalyzer(nil, log.Log)
	})

	Describe("Storage Usage Analysis", func() {
		It("should calculate utilization percentage", func() {
			storageUsage := &StorageUsage{
				Component:    "prometheus",
				PVCName:      "prometheus-data",
				StorageClass: "standard",
				Capacity:     100 * 1024 * 1024 * 1024, // 100Gi
				Used:         75 * 1024 * 1024 * 1024,  // 75Gi
			}

			utilization := storageUsage.GetUtilizationPercentage()
			Expect(utilization).To(BeNumerically("==", 75.0))
		})

		It("should calculate days until full", func() {
			storageUsage := &StorageUsage{
				Component:        "loki",
				Capacity:         100 * 1024 * 1024 * 1024, // 100Gi
				Used:             50 * 1024 * 1024 * 1024,  // 50Gi
				GrowthRatePerDay: 5 * 1024 * 1024 * 1024,  // 5Gi per day
			}

			daysUntilFull := storageUsage.GetDaysUntilFull()
			Expect(daysUntilFull).To(BeNumerically("==", 10.0))
		})

		It("should handle zero growth rate", func() {
			storageUsage := &StorageUsage{
				Component:        "grafana",
				Capacity:         10 * 1024 * 1024 * 1024, // 10Gi
				Used:             5 * 1024 * 1024 * 1024,  // 5Gi
				GrowthRatePerDay: 0,
			}

			daysUntilFull := storageUsage.GetDaysUntilFull()
			Expect(daysUntilFull).To(BeNumerically("==", -1))
		})

		It("should handle negative growth rate", func() {
			storageUsage := &StorageUsage{
				Component:        "tempo",
				Capacity:         20 * 1024 * 1024 * 1024,  // 20Gi
				Used:             15 * 1024 * 1024 * 1024,  // 15Gi
				GrowthRatePerDay: -1 * 1024 * 1024 * 1024, // -1Gi per day (shrinking)
			}

			daysUntilFull := storageUsage.GetDaysUntilFull()
			Expect(daysUntilFull).To(BeNumerically("==", -1))
		})
	})
})

var _ = Describe("Cost Optimizer", func() {
	var (
		optimizer *CostOptimizer
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		optimizer = NewCostOptimizer(log.Log)
	})

	Describe("Resource Sizing", func() {
		It("should round to standard sizes correctly", func() {
			testCases := []struct {
				input    float64
				expected float64
			}{
				{0.1, 0.25},
				{0.3, 0.5},
				{0.7, 1.0},
				{1.5, 2.0},
				{3.0, 4.0},
				{5.0, 8.0},
				{10.0, 16.0},
				{20.0, 32.0},
				{50.0, 64.0},
				{100.0, 104.0}, // Multiple of 8
			}

			for _, tc := range testCases {
				result := optimizer.RoundToStandardSize(tc.input)
				Expect(result).To(Equal(tc.expected), "Input: %f", tc.input)
			}
		})
	})

	Describe("Spot Instance Eligibility", func() {
		It("should identify spot eligible components", func() {
			eligibleComponents := []string{"grafana", "loki-querier", "tempo-querier"}
			for _, component := range eligibleComponents {
				Expect(IsSpotEligible(component)).To(BeTrue(), "Component: %s", component)
			}
		})

		It("should identify spot ineligible components", func() {
			ineligibleComponents := []string{"prometheus", "loki-ingester", "tempo-ingester"}
			for _, component := range ineligibleComponents {
				Expect(IsSpotEligible(component)).To(BeFalse(), "Component: %s", component)
			}
		})
	})

	Describe("Cost Calculations", func() {
		It("should calculate resource costs correctly", func() {
			resources := &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),      // 2 vCPU
					corev1.ResourceMemory: resource.MustParse("8Gi"),    // 8GB
				},
			}

			// Default pricing model:
			// CPU: $0.0464/hour = $33.408/month per vCPU
			// Memory: $0.0116/hour = $8.352/month per GB
			expectedCPUCost := 2 * 0.0464 * 24 * 30      // ~$66.816
			expectedMemoryCost := 8 * 0.0116 * 24 * 30   // ~$66.816
			expectedTotalCost := expectedCPUCost + expectedMemoryCost

			cost := optimizer.CalculateResourceCost(resources)
			Expect(cost).To(BeNumerically("~", expectedTotalCost, 0.01))
		})

		It("should calculate storage costs with different tiers", func() {
			testCases := []struct {
				size         string
				storageClass string
				expectedCost float64
			}{
				{"100Gi", "standard", 10.74},      // 100GB * $0.10/GB
				{"100Gi", "premium", 13.42},       // 100GB * $0.125/GB
				{"100Gi", "cold", 2.47},           // 100GB * $0.023/GB
			}

			for _, tc := range testCases {
				cost := optimizer.CalculateStorageCost(tc.size, tc.storageClass)
				Expect(cost).To(BeNumerically("~", tc.expectedCost, 0.01),
					"Size: %s, Class: %s", tc.size, tc.storageClass)
			}
		})
	})

	Describe("Optimization Recommendations", func() {
		It("should recommend right-sizing when over-provisioned", func() {
			usage := &ResourceUsage{
				Component: "prometheus",
				Percentiles: ResourceUsagePercentiles{
					CPU: UsagePercentiles{
						P95: 0.5, // Using 0.5 vCPU
					},
					Memory: UsagePercentiles{
						P95: 2.0 * 1024 * 1024 * 1024, // Using 2GB
					},
				},
			}

			current := &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),    // Allocated 2 vCPU
					corev1.ResourceMemory: resource.MustParse("8Gi"),  // Allocated 8GB
				},
			}

			recommendation := optimizer.GenerateResourceRecommendation("prometheus", usage, current)
			Expect(recommendation).NotTo(BeNil())
			Expect(recommendation.EstimatedSavings).To(BeNumerically(">", 0))
			
			// Should recommend less than current
			recommendedCPU := recommendation.Recommended.Requests[corev1.ResourceCPU]
			currentCPU := current.Requests[corev1.ResourceCPU]
			Expect(recommendedCPU.Cmp(currentCPU)).To(Equal(-1)) // recommended < current
		})

		It("should not recommend changes for well-sized resources", func() {
			usage := &ResourceUsage{
				Component: "grafana",
				Percentiles: ResourceUsagePercentiles{
					CPU: UsagePercentiles{
						P95: 0.45, // Using 0.45 vCPU
					},
					Memory: UsagePercentiles{
						P95: 0.9 * 1024 * 1024 * 1024, // Using 0.9GB
					},
				},
			}

			current := &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"), // Allocated 0.5 vCPU
					corev1.ResourceMemory: resource.MustParse("1Gi"),  // Allocated 1GB
				},
			}

			recommendation := optimizer.GenerateResourceRecommendation("grafana", usage, current)
			Expect(recommendation).To(BeNil()) // No significant optimization needed
		})
	})
})

var _ = Describe("Pricing Model", func() {
	It("should create default AWS pricing model", func() {
		pricing := NewPricingModel()
		
		Expect(pricing.Provider).To(Equal("aws"))
		Expect(pricing.Region).To(Equal("us-east-1"))
		Expect(pricing.CPUHourlyRate).To(BeNumerically(">", 0))
		Expect(pricing.MemoryHourlyRate).To(BeNumerically(">", 0))
		Expect(pricing.StorageStandardRate).To(BeNumerically(">", 0))
		Expect(pricing.SpotDiscountPercent).To(Equal(float64(70)))
	})

	It("should calculate spot instance savings", func() {
		pricing := NewPricingModel()
		regularCost := 100.0
		spotCost := regularCost * (1 - pricing.SpotDiscountPercent/100)
		
		Expect(spotCost).To(Equal(30.0)) // 70% discount
	})
})

var _ = Describe("Cost Analysis Types", func() {
	Describe("CostAnalysis", func() {
		It("should calculate totals correctly", func() {
			analysis := &CostAnalysis{
				Platform:  "test-platform",
				Namespace: "test-namespace",
				Components: map[string]*ComponentCost{
					"prometheus": {
						Component: "prometheus",
						Resources: ResourceCost{
							CPU:     50.0,
							Memory:  40.0,
							Storage: 10.0,
							Network: 5.0,
						},
					},
					"grafana": {
						Component: "grafana",
						Resources: ResourceCost{
							CPU:     20.0,
							Memory:  15.0,
							Storage: 5.0,
							Network: 2.0,
						},
					},
				},
			}

			analysis.CalculateTotals()
			
			Expect(analysis.Components["prometheus"].Total).To(Equal(105.0))
			Expect(analysis.Components["grafana"].Total).To(Equal(42.0))
			Expect(analysis.TotalCost).To(Equal(147.0))
			Expect(analysis.Currency).To(Equal("USD"))
		})
	})

	Describe("OptimizationRecommendations", func() {
		It("should calculate total savings", func() {
			recommendations := &OptimizationRecommendations{
				ResourceRecommendations: map[string]interface{}{
					"prometheus": &ResourceRecommendation{
						EstimatedSavings: 30.0,
					},
					"grafana": &ResourceRecommendation{
						EstimatedSavings: 10.0,
					},
				},
				SpotInstanceCandidates: map[string]bool{
					"grafana": true,
				},
				ComponentCosts: map[string]float64{
					"grafana": 50.0,
				},
			}

			pricing := NewPricingModel()
			recommendations.CalculateSavings(pricing)
			
			// Resource savings: 30 + 10 = 40
			// Spot savings: 50 * 0.7 = 35
			// Total: 75
			Expect(recommendations.TotalSavings).To(Equal(75.0))
		})
	})
})
