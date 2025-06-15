package cost

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

// CostAnalysis represents the cost analysis for a platform
type CostAnalysis struct {
	Platform   string                     `json:"platform"`
	Namespace  string                     `json:"namespace"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]*ComponentCost  `json:"components"`
	TotalCost  float64                    `json:"totalCost"`
	Currency   string                     `json:"currency"`
}

// ComponentCost represents costs for a single component
type ComponentCost struct {
	Component string       `json:"component"`
	Resources ResourceCost `json:"resources"`
	Total     float64      `json:"total"`
}

// ResourceCost breakdown by resource type
type ResourceCost struct {
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Storage float64 `json:"storage"`
	Network float64 `json:"network"`
}

// OptimizationRecommendations contains cost optimization suggestions
type OptimizationRecommendations struct {
	Platform                string                              `json:"platform"`
	Namespace               string                              `json:"namespace"`
	Timestamp               time.Time                           `json:"timestamp"`
	ResourceRecommendations map[string]*ResourceRecommendation  `json:"resourceRecommendations"`
	SpotInstanceCandidates  map[string]bool                     `json:"spotInstanceCandidates"`
	ScalingPolicies         map[string]*ScalingPolicy           `json:"scalingPolicies"`
	StorageOptimizations    map[string]*StorageOptimization     `json:"storageOptimizations"`
	ComponentCosts          map[string]float64                  `json:"componentCosts"`
	TotalSavings            float64                             `json:"totalSavings"`
}

// ResourceRecommendation for right-sizing
type ResourceRecommendation struct {
	Component         string                      `json:"component"`
	Current           corev1.ResourceRequirements `json:"current"`
	Recommended       corev1.ResourceRequirements `json:"recommended"`
	EstimatedSavings  float64                     `json:"estimatedSavings"`
	UtilizationBased  bool                        `json:"utilizationBased"`
}

// ScalingPolicy for autoscaling configuration
type ScalingPolicy struct {
	Component   string                          `json:"component"`
	MinReplicas int32                           `json:"minReplicas"`
	MaxReplicas int32                           `json:"maxReplicas"`
	Metrics     []autoscalingv2.MetricSpec      `json:"metrics"`
	Behavior    *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

// StorageOptimization recommendations
type StorageOptimization struct {
	Component        string  `json:"component"`
	CurrentSize      string  `json:"currentSize"`
	RecommendedSize  string  `json:"recommendedSize"`
	CurrentClass     string  `json:"currentClass"`
	RecommendedClass string  `json:"recommendedClass"`
	EstimatedSavings float64 `json:"estimatedSavings"`
}

// OptimizationOptions for applying optimizations
type OptimizationOptions struct {
	ApplyResourceOptimization bool     `json:"applyResourceOptimization"`
	EnableSpotInstances       bool     `json:"enableSpotInstances"`
	ApplyScalingPolicies      bool     `json:"applyScalingPolicies"`
	OptimizeStorage           bool     `json:"optimizeStorage"`
	Categories                []string `json:"categories"`
	DryRun                    bool     `json:"dryRun"`
}

// BudgetConfig for cost alerts
type BudgetConfig struct {
	MonthlyLimit    float64           `json:"monthlyLimit"`
	AlertThreshold  float64           `json:"alertThreshold"` // Percentage (e.g., 80 for 80%)
	AlertRecipients []string          `json:"alertRecipients"`
	Actions         []BudgetAction    `json:"actions"`
}

// BudgetAction to take when budget threshold is reached
type BudgetAction struct {
	Threshold float64 `json:"threshold"`
	Action    string  `json:"action"` // "alert", "scale-down", "disable-non-essential"
}

// PricingModel contains cloud provider pricing information
type PricingModel struct {
	Provider             string  `json:"provider"`
	Region               string  `json:"region"`
	CPUHourlyRate        float64 `json:"cpuHourlyRate"`        // Per vCPU hour
	MemoryHourlyRate     float64 `json:"memoryHourlyRate"`     // Per GB hour
	StorageStandardRate  float64 `json:"storageStandardRate"`  // Per GB month
	StoragePremiumRate   float64 `json:"storagePremiumRate"`   // Per GB month
	StorageArchiveRate   float64 `json:"storageArchiveRate"`   // Per GB month
	NetworkEgressRate    float64 `json:"networkEgressRate"`    // Per GB
	SpotDiscountPercent  float64 `json:"spotDiscountPercent"`  // Percentage discount for spot
}

// ResourceUsage tracks actual resource usage
type ResourceUsage struct {
	Component   string                  `json:"component"`
	Timestamp   time.Time               `json:"timestamp"`
	CPUUsage    resource.Quantity       `json:"cpuUsage"`
	MemoryUsage resource.Quantity       `json:"memoryUsage"`
	Percentiles ResourceUsagePercentiles `json:"percentiles"`
}

// ResourceUsagePercentiles for statistical analysis
type ResourceUsagePercentiles struct {
	CPU    UsagePercentiles `json:"cpu"`
	Memory UsagePercentiles `json:"memory"`
}

// UsagePercentiles represents usage distribution
type UsagePercentiles struct {
	P50  float64 `json:"p50"`
	P90  float64 `json:"p90"`
	P95  float64 `json:"p95"`
	P99  float64 `json:"p99"`
	Max  float64 `json:"max"`
	Mean float64 `json:"mean"`
}

// CostAllocation for chargeback/showback
type CostAllocation struct {
	Platform    string                   `json:"platform"`
	Namespace   string                   `json:"namespace"`
	Period      CostPeriod               `json:"period"`
	Allocations map[string]*Allocation   `json:"allocations"`
	TotalCost   float64                  `json:"totalCost"`
}

// CostPeriod represents the time period for cost allocation
type CostPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Allocation represents cost allocation to a team/department
type Allocation struct {
	Team       string            `json:"team"`
	Department string            `json:"department"`
	CostCenter string            `json:"costCenter"`
	Components map[string]float64 `json:"components"`
	Total      float64           `json:"total"`
	Tags       map[string]string `json:"tags"`
}

// Helper methods

// calculateTotals calculates total costs for the analysis
func (ca *CostAnalysis) calculateTotals() {
	total := 0.0
	for _, cost := range ca.Components {
		cost.calculateTotal()
		total += cost.Total
	}
	ca.TotalCost = total
	if ca.Currency == "" {
		ca.Currency = "USD"
	}
}

// CalculateTotals calculates total costs for the analysis (exported for testing)
func (ca *CostAnalysis) CalculateTotals() {
	ca.calculateTotals()
}

// calculateTotal calculates total cost for a component
func (cc *ComponentCost) calculateTotal() {
	cc.Total = cc.Resources.CPU + cc.Resources.Memory + cc.Resources.Storage + cc.Resources.Network
}

// calculateSavings calculates total potential savings
func (or *OptimizationRecommendations) calculateSavings(pricing *PricingModel) {
	totalSavings := 0.0

	// Resource optimization savings
	for _, rec := range or.ResourceRecommendations {
		totalSavings += rec.EstimatedSavings
	}

	// Spot instance savings
	for component, eligible := range or.SpotInstanceCandidates {
		if eligible {
			componentCost := or.ComponentCosts[component]
			spotSavings := componentCost * (pricing.SpotDiscountPercent / 100)
			totalSavings += spotSavings
		}
	}

	// Storage optimization savings
	for _, opt := range or.StorageOptimizations {
		totalSavings += opt.EstimatedSavings
	}

	or.TotalSavings = totalSavings
}

// CalculateSavings calculates total potential savings (exported for testing)
func (or *OptimizationRecommendations) CalculateSavings(pricing *PricingModel) {
	or.calculateSavings(pricing)
}

// NewPricingModel creates a default pricing model
func NewPricingModel() *PricingModel {
	// Default AWS pricing (simplified)
	return &PricingModel{
		Provider:             "aws",
		Region:               "us-east-1",
		CPUHourlyRate:        0.0464,  // Per vCPU hour (m5.large equivalent)
		MemoryHourlyRate:     0.0116,  // Per GB hour
		StorageStandardRate:  0.10,    // Per GB month (EBS gp3)
		StoragePremiumRate:   0.125,   // Per GB month (EBS io2)
		StorageArchiveRate:   0.023,   // Per GB month (S3 Glacier)
		NetworkEgressRate:    0.09,    // Per GB
		SpotDiscountPercent:  70,      // 70% discount for spot instances
	}
}

// IsSpotEligible determines if a component can use spot instances
func IsSpotEligible(component string) bool {
	// Components that can tolerate interruptions
	spotEligible := map[string]bool{
		"grafana":      true,  // Stateless, can be restarted
		"loki-querier": true,  // Read path can use spot
		"tempo-querier": true, // Read path can use spot
		"prometheus":   false, // Needs persistent storage
		"loki-ingester": false, // Needs persistence
		"tempo-ingester": false, // Needs persistence
	}

	return spotEligible[component]
}
