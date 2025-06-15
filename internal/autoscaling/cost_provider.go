package autoscaling

import (
	"context"
	"fmt"
	"math"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CloudProvider represents a cloud provider
type CloudProvider string

const (
	AWS   CloudProvider = "aws"
	GCP   CloudProvider = "gcp"
	Azure CloudProvider = "azure"
	OnPrem CloudProvider = "onprem"
)

// CostProviderImpl implements cost calculations for different cloud providers
type CostProviderImpl struct {
	provider CloudProvider
	region   string
	pricing  map[string]float64 // Resource type to price per unit
}

// NewCostProvider creates a new cost provider
func NewCostProvider(provider string, region string) *CostProviderImpl {
	cp := &CostProviderImpl{
		provider: CloudProvider(provider),
		region:   region,
	}
	
	// Initialize pricing based on provider
	cp.initializePricing()
	
	return cp
}

// initializePricing sets up pricing data for the provider
func (c *CostProviderImpl) initializePricing() {
	switch c.provider {
	case AWS:
		c.pricing = map[string]float64{
			"cpu_per_core_hour":    0.0464,  // m5.large pricing
			"memory_per_gb_hour":   0.0058,  // Based on m5 series
			"storage_per_gb_month": 0.10,    // EBS gp3
			"network_per_gb":       0.09,    // Data transfer
		}
	case GCP:
		c.pricing = map[string]float64{
			"cpu_per_core_hour":    0.0475,
			"memory_per_gb_hour":   0.0050,
			"storage_per_gb_month": 0.17,
			"network_per_gb":       0.12,
		}
	case Azure:
		c.pricing = map[string]float64{
			"cpu_per_core_hour":    0.0496,
			"memory_per_gb_hour":   0.0055,
			"storage_per_gb_month": 0.15,
			"network_per_gb":       0.087,
		}
	default:
		// Default on-premise estimations
		c.pricing = map[string]float64{
			"cpu_per_core_hour":    0.03,
			"memory_per_gb_hour":   0.004,
			"storage_per_gb_month": 0.05,
			"network_per_gb":       0.01,
		}
	}
}

// GetCurrentCost calculates the current cost for a component
func (c *CostProviderImpl) GetCurrentCost(ctx context.Context, component v1beta1.ComponentType, replicas int32, resources corev1.ResourceRequirements) (float64, error) {
	log := log.FromContext(ctx)
	
	// Extract resource values
	cpuRequest := resources.Requests[corev1.ResourceCPU]
	memoryRequest := resources.Requests[corev1.ResourceMemory]
	
	// Convert to standard units
	cpuCores := cpuRequest.AsApproximateFloat64()
	memoryGB := memoryRequest.AsApproximateFloat64() / (1024 * 1024 * 1024)
	
	// Calculate hourly cost
	cpuCostPerHour := cpuCores * c.pricing["cpu_per_core_hour"] * float64(replicas)
	memoryCostPerHour := memoryGB * c.pricing["memory_per_gb_hour"] * float64(replicas)
	
	// Add storage cost based on component type
	storageCostPerHour := c.getStorageCost(component, replicas)
	
	// Total hourly cost
	totalHourlyCost := cpuCostPerHour + memoryCostPerHour + storageCostPerHour
	
	// Convert to monthly
	monthlyCost := totalHourlyCost * 24 * 30
	
	log.V(2).Info("Calculated current cost",
		"component", component,
		"replicas", replicas,
		"cpuCores", cpuCores,
		"memoryGB", memoryGB,
		"monthlyCost", monthlyCost,
	)
	
	return monthlyCost, nil
}

// GetProjectedCost calculates the projected cost for a scaling decision
func (c *CostProviderImpl) GetProjectedCost(ctx context.Context, component v1beta1.ComponentType, replicas int32, resources corev1.ResourceRequirements) (float64, error) {
	// Use the same calculation as current cost
	return c.GetCurrentCost(ctx, component, replicas, resources)
}

// GetOptimalConfiguration finds the optimal configuration within a budget
func (c *CostProviderImpl) GetOptimalConfiguration(ctx context.Context, component v1beta1.ComponentType, currentReplicas int32, budget float64) (*ScalingDecision, error) {
	log := log.FromContext(ctx)
	
	// Define resource configurations to try
	configurations := c.getResourceConfigurations(component)
	
	var bestConfig *ScalingDecision
	bestPerformancePerDollar := 0.0
	
	for replicas := int32(1); replicas <= 10; replicas++ {
		for _, config := range configurations {
			cost, err := c.GetCurrentCost(ctx, component, replicas, config.resources)
			if err != nil {
				continue
			}
			
			// Skip if over budget
			if cost > budget {
				continue
			}
			
			// Calculate performance score
			performance := c.calculatePerformanceScore(replicas, config.resources)
			performancePerDollar := performance / cost
			
			if performancePerDollar > bestPerformancePerDollar {
				bestPerformancePerDollar = performancePerDollar
				bestConfig = &ScalingDecision{
					Type:             CostAwareScaling,
					Component:        component,
					CurrentReplicas:  currentReplicas,
					TargetReplicas:   replicas,
					CurrentResources: corev1.ResourceRequirements{}, // Would get from current state
					TargetResources:  config.resources,
					Reason:           fmt.Sprintf("Optimal configuration within budget $%.2f", budget),
					CostImpact: &CostImpact{
						ProjectedCost:  cost,
						Currency:       "USD",
					},
				}
			}
		}
	}
	
	if bestConfig == nil {
		return nil, fmt.Errorf("no configuration found within budget $%.2f", budget)
	}
	
	// Calculate savings
	currentCost, _ := c.GetCurrentCost(ctx, component, currentReplicas, bestConfig.CurrentResources)
	bestConfig.CostImpact.CurrentCost = currentCost
	bestConfig.CostImpact.MonthlySavings = currentCost - bestConfig.CostImpact.ProjectedCost
	
	log.Info("Found optimal configuration",
		"component", component,
		"targetReplicas", bestConfig.TargetReplicas,
		"monthlyCost", bestConfig.CostImpact.ProjectedCost,
		"savings", bestConfig.CostImpact.MonthlySavings,
	)
	
	return bestConfig, nil
}

// getStorageCost estimates storage cost based on component type
func (c *CostProviderImpl) getStorageCost(component v1beta1.ComponentType, replicas int32) float64 {
	var storageGB float64
	
	switch component {
	case v1beta1.ComponentPrometheus:
		storageGB = 100 * float64(replicas) // 100GB per replica
	case v1beta1.ComponentLoki:
		storageGB = 200 * float64(replicas) // 200GB per replica
	case v1beta1.ComponentTempo:
		storageGB = 50 * float64(replicas) // 50GB per replica
	case v1beta1.ComponentGrafana:
		storageGB = 10 * float64(replicas) // 10GB per replica
	default:
		storageGB = 20 * float64(replicas) // Default 20GB
	}
	
	// Convert monthly storage cost to hourly
	monthlyStorageCost := storageGB * c.pricing["storage_per_gb_month"]
	return monthlyStorageCost / (24 * 30)
}

// calculatePerformanceScore estimates performance based on resources
func (c *CostProviderImpl) calculatePerformanceScore(replicas int32, resources corev1.ResourceRequirements) float64 {
	cpuRequest := resources.Requests[corev1.ResourceCPU]
	memoryRequest := resources.Requests[corev1.ResourceMemory]
	
	cpuCores := cpuRequest.AsApproximateFloat64()
	memoryGB := memoryRequest.AsApproximateFloat64() / (1024 * 1024 * 1024)
	
	// Simple performance score: weighted sum of resources
	// In production, this would be based on actual performance metrics
	cpuWeight := 0.6
	memoryWeight := 0.4
	
	performancePerReplica := (cpuCores * cpuWeight) + (memoryGB * memoryWeight)
	
	// Apply diminishing returns for high replica counts
	replicaEfficiency := 1.0 - (0.05 * float64(replicas-1)) // 5% efficiency loss per additional replica
	if replicaEfficiency < 0.5 {
		replicaEfficiency = 0.5
	}
	
	return performancePerReplica * float64(replicas) * replicaEfficiency
}

// ResourceConfiguration represents a resource configuration option
type ResourceConfiguration struct {
	name      string
	resources corev1.ResourceRequirements
}

// getResourceConfigurations returns possible resource configurations for a component
func (c *CostProviderImpl) getResourceConfigurations(component v1beta1.ComponentType) []ResourceConfiguration {
	switch component {
	case v1beta1.ComponentPrometheus:
		return []ResourceConfiguration{
			{
				name: "small",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
			{
				name: "medium",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
			{
				name: "large",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("8Gi"),
					},
				},
			},
		}
	case v1beta1.ComponentGrafana:
		return []ResourceConfiguration{
			{
				name: "small",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			},
			{
				name: "medium",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
			},
		}
	default:
		// Default configurations
		return []ResourceConfiguration{
			{
				name: "default",
				resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
			},
		}
	}
}

// AnalyzeCostTrends analyzes cost trends over time
func (c *CostProviderImpl) AnalyzeCostTrends(ctx context.Context, component v1beta1.ComponentType, history []ScalingEvent) (*CostAnalysis, error) {
	if len(history) == 0 {
		return nil, fmt.Errorf("no historical data available")
	}
	
	totalCost := 0.0
	minCost := math.MaxFloat64
	maxCost := 0.0
	
	costOverTime := make([]CostDataPoint, 0, len(history))
	
	for _, event := range history {
		if event.CostImpact != nil {
			cost := event.CostImpact.ProjectedCost
			totalCost += cost
			
			if cost < minCost {
				minCost = cost
			}
			if cost > maxCost {
				maxCost = cost
			}
			
			costOverTime = append(costOverTime, CostDataPoint{
				Timestamp: event.Timestamp,
				Cost:      cost,
				Replicas:  event.ToReplicas,
			})
		}
	}
	
	avgCost := totalCost / float64(len(history))
	
	// Calculate trend
	trend := "stable"
	if len(costOverTime) >= 2 {
		firstCost := costOverTime[0].Cost
		lastCost := costOverTime[len(costOverTime)-1].Cost
		change := (lastCost - firstCost) / firstCost * 100
		
		if change > 10 {
			trend = "increasing"
		} else if change < -10 {
			trend = "decreasing"
		}
	}
	
	return &CostAnalysis{
		Component:    component,
		AverageCost:  avgCost,
		MinCost:      minCost,
		MaxCost:      maxCost,
		Trend:        trend,
		CostOverTime: costOverTime,
		Recommendations: c.generateCostRecommendations(component, avgCost, trend),
	}, nil
}

// generateCostRecommendations generates cost optimization recommendations
func (c *CostProviderImpl) generateCostRecommendations(component v1beta1.ComponentType, avgCost float64, trend string) []string {
	recommendations := []string{}
	
	if trend == "increasing" {
		recommendations = append(recommendations, 
			"Consider implementing more aggressive auto-scaling policies",
			"Review resource requests and limits for over-provisioning",
			"Enable predictive scaling to anticipate load patterns",
		)
	}
	
	if avgCost > 1000 { // $1000/month threshold
		recommendations = append(recommendations,
			"Consider using spot instances for non-critical workloads",
			"Implement time-based scaling for known quiet periods",
			"Review storage retention policies to reduce costs",
		)
	}
	
	// Component-specific recommendations
	switch component {
	case v1beta1.ComponentPrometheus:
		recommendations = append(recommendations,
			"Consider downsampling old metrics to reduce storage",
			"Use recording rules to pre-compute expensive queries",
		)
	case v1beta1.ComponentLoki:
		recommendations = append(recommendations,
			"Implement log retention policies",
			"Use log sampling for high-volume, low-value logs",
		)
	}
	
	return recommendations
}

// CostAnalysis represents a cost analysis result
type CostAnalysis struct {
	Component       v1beta1.ComponentType
	AverageCost     float64
	MinCost         float64
	MaxCost         float64
	Trend           string
	CostOverTime    []CostDataPoint
	Recommendations []string
}

// CostDataPoint represents a cost measurement at a point in time
type CostDataPoint struct {
	Timestamp time.Time
	Cost      float64
	Replicas  int32
}
