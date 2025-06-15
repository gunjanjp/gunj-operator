package cost

import (
	"fmt"
	"math"

	"github.com/go-logr/logr"
	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/utils/ptr"
)

// CostOptimizer generates cost optimization recommendations
type CostOptimizer struct {
	log logr.Logger
	// Optimization thresholds
	cpuUtilizationTarget    float64
	memoryUtilizationTarget float64
	minResourceMargin       float64
	spotEligibilityRules    map[string]bool
}

// NewCostOptimizer creates a new cost optimizer
func NewCostOptimizer(log logr.Logger) *CostOptimizer {
	return &CostOptimizer{
		log:                     log.WithName("cost-optimizer"),
		cpuUtilizationTarget:    70.0,  // Target 70% CPU utilization
		memoryUtilizationTarget: 80.0,  // Target 80% memory utilization
		minResourceMargin:       1.2,   // 20% margin above P95
		spotEligibilityRules: map[string]bool{
			"grafana":        true,
			"loki-querier":   true,
			"tempo-querier":  true,
			"prometheus":     false,
			"loki-ingester":  false,
			"tempo-ingester": false,
		},
	}
}

// GenerateRecommendations creates optimization recommendations based on usage
func (co *CostOptimizer) GenerateRecommendations(platform *observabilityv1.ObservabilityPlatform, usage map[string]*ResourceUsage) *OptimizationRecommendations {
	co.log.V(1).Info("Generating optimization recommendations", "platform", platform.Name)

	recommendations := &OptimizationRecommendations{
		Platform:                platform.Name,
		Namespace:               platform.Namespace,
		ResourceRecommendations: make(map[string]*ResourceRecommendation),
		SpotInstanceCandidates:  make(map[string]bool),
		ScalingPolicies:         make(map[string]*ScalingPolicy),
		StorageOptimizations:    make(map[string]*StorageOptimization),
		ComponentCosts:          make(map[string]float64),
	}

	// Generate resource recommendations for each component
	for component, componentUsage := range usage {
		// Resource right-sizing
		if resourceRec := co.generateResourceRecommendation(component, componentUsage, platform); resourceRec != nil {
			recommendations.ResourceRecommendations[component] = resourceRec
		}

		// Spot instance eligibility
		if co.isSpotEligible(component, componentUsage) {
			recommendations.SpotInstanceCandidates[component] = true
		}

		// Scaling policies
		if scalingPolicy := co.generateScalingPolicy(component, componentUsage); scalingPolicy != nil {
			recommendations.ScalingPolicies[component] = scalingPolicy
		}
	}

	// Storage optimizations
	co.generateStorageOptimizations(platform, recommendations)

	return recommendations
}

// generateResourceRecommendation creates right-sizing recommendations
func (co *CostOptimizer) generateResourceRecommendation(component string, usage *ResourceUsage, platform *observabilityv1.ObservabilityPlatform) *ResourceRecommendation {
	current := co.getCurrentResources(component, platform)
	if current == nil {
		return nil
	}

	// Calculate recommended resources based on P95 usage with margin
	recommendedCPU := co.calculateRecommendedResource(
		usage.Percentiles.CPU.P95,
		current.Requests.Cpu().AsApproximateFloat64(),
		co.cpuUtilizationTarget,
	)

	recommendedMemory := co.calculateRecommendedResource(
		usage.Percentiles.Memory.P95 / (1024 * 1024 * 1024), // Convert to GB
		current.Requests.Memory().AsApproximateFloat64() / (1024 * 1024 * 1024),
		co.memoryUtilizationTarget,
	)

	// Only recommend if there's significant difference (>10%)
	cpuDiff := math.Abs(recommendedCPU-current.Requests.Cpu().AsApproximateFloat64()) / current.Requests.Cpu().AsApproximateFloat64()
	memDiff := math.Abs(recommendedMemory-current.Requests.Memory().AsApproximateFloat64()/(1024*1024*1024)) / (current.Requests.Memory().AsApproximateFloat64() / (1024 * 1024 * 1024))

	if cpuDiff < 0.1 && memDiff < 0.1 {
		return nil // No significant optimization needed
	}

	recommended := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(recommendedCPU*1000), resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(int64(recommendedMemory*1024*1024*1024), resource.BinarySI),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(recommendedCPU*1500), resource.DecimalSI), // 1.5x headroom
			corev1.ResourceMemory: *resource.NewQuantity(int64(recommendedMemory*1.2*1024*1024*1024), resource.BinarySI), // 1.2x headroom
		},
	}

	// Calculate estimated savings
	currentCost := co.calculateResourceCost(current)
	recommendedCost := co.calculateResourceCost(&recommended)
	savings := currentCost - recommendedCost

	return &ResourceRecommendation{
		Component:         component,
		Current:           *current,
		Recommended:       recommended,
		EstimatedSavings:  savings,
		UtilizationBased:  true,
	}
}

// GenerateResourceRecommendation creates right-sizing recommendations (exported for testing)
func (co *CostOptimizer) GenerateResourceRecommendation(component string, usage *ResourceUsage, current *corev1.ResourceRequirements) *ResourceRecommendation {
	// For testing, we accept current resources directly
	if current == nil {
		return nil
	}

	// Calculate recommended resources based on P95 usage with margin
	recommendedCPU := co.calculateRecommendedResource(
		usage.Percentiles.CPU.P95,
		current.Requests.Cpu().AsApproximateFloat64(),
		co.cpuUtilizationTarget,
	)

	recommendedMemory := co.calculateRecommendedResource(
		usage.Percentiles.Memory.P95 / (1024 * 1024 * 1024), // Convert to GB
		current.Requests.Memory().AsApproximateFloat64() / (1024 * 1024 * 1024),
		co.memoryUtilizationTarget,
	)

	// Only recommend if there's significant difference (>10%)
	cpuDiff := math.Abs(recommendedCPU-current.Requests.Cpu().AsApproximateFloat64()) / current.Requests.Cpu().AsApproximateFloat64()
	memDiff := math.Abs(recommendedMemory-current.Requests.Memory().AsApproximateFloat64()/(1024*1024*1024)) / (current.Requests.Memory().AsApproximateFloat64() / (1024 * 1024 * 1024))

	if cpuDiff < 0.1 && memDiff < 0.1 {
		return nil // No significant optimization needed
	}

	recommended := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(recommendedCPU*1000), resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(int64(recommendedMemory*1024*1024*1024), resource.BinarySI),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(recommendedCPU*1500), resource.DecimalSI), // 1.5x headroom
			corev1.ResourceMemory: *resource.NewQuantity(int64(recommendedMemory*1.2*1024*1024*1024), resource.BinarySI), // 1.2x headroom
		},
	}

	// Calculate estimated savings
	currentCost := co.calculateResourceCost(current)
	recommendedCost := co.calculateResourceCost(&recommended)
	savings := currentCost - recommendedCost

	return &ResourceRecommendation{
		Component:         component,
		Current:           *current,
		Recommended:       recommended,
		EstimatedSavings:  savings,
		UtilizationBased:  true,
	}
}

// calculateRecommendedResource calculates the recommended resource size
func (co *CostOptimizer) calculateRecommendedResource(actualUsage, currentAllocation, targetUtilization float64) float64 {
	// Calculate what the allocation should be to achieve target utilization
	recommendedAllocation := actualUsage * co.minResourceMargin / (targetUtilization / 100)
	
	// Round to nearest standard size (0.25, 0.5, 1, 2, 4, 8, etc.)
	return co.roundToStandardSize(recommendedAllocation)
}

// roundToStandardSize rounds to standard cloud resource sizes
func (co *CostOptimizer) roundToStandardSize(value float64) float64 {
	standardSizes := []float64{0.25, 0.5, 1, 2, 4, 8, 16, 32, 64}
	
	for _, size := range standardSizes {
		if value <= size {
			return size
		}
	}
	
	// For very large values, round to nearest multiple of 8
	return math.Ceil(value/8) * 8
}

// RoundToStandardSize rounds to standard cloud resource sizes (exported for testing)
func (co *CostOptimizer) RoundToStandardSize(value float64) float64 {
	return co.roundToStandardSize(value)
}

// isSpotEligible determines if a component can use spot instances
func (co *CostOptimizer) isSpotEligible(component string, usage *ResourceUsage) bool {
	// Check base eligibility rules
	if eligible, exists := co.spotEligibilityRules[component]; exists && !eligible {
		return false
	}

	// Additional checks based on usage patterns
	// Components with high variance in usage are less suitable for spot
	cpuVariance := (usage.Percentiles.CPU.P99 - usage.Percentiles.CPU.P50) / usage.Percentiles.CPU.Mean
	if cpuVariance > 0.5 { // More than 50% variance
		return false
	}

	return true
}

// generateScalingPolicy creates HPA configuration
func (co *CostOptimizer) generateScalingPolicy(component string, usage *ResourceUsage) *ScalingPolicy {
	// Don't create scaling policies for single-instance components
	if component == "grafana" {
		return nil
	}

	// Calculate min/max replicas based on usage patterns
	minReplicas := int32(1)
	maxReplicas := int32(10)

	// Adjust based on component type
	switch component {
	case "prometheus":
		// Prometheus typically scales vertically, not horizontally
		// unless using sharding
		return nil
	case "loki-ingester":
		minReplicas = 3 // For data redundancy
		maxReplicas = 20
	case "tempo-ingester":
		minReplicas = 3
		maxReplicas = 20
	}

	// Create scaling metrics
	metrics := []autoscalingv2.MetricSpec{
		{
			Type: autoscalingv2.ResourceMetricType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: ptr.To(int32(co.cpuUtilizationTarget)),
				},
			},
		},
		{
			Type: autoscalingv2.ResourceMetricType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: ptr.To(int32(co.memoryUtilizationTarget)),
				},
			},
		},
	}

	// Add custom metrics for specific components
	if component == "loki-ingester" || component == "tempo-ingester" {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.PodsMetricType,
			Pods: &autoscalingv2.PodsMetricSource{
				Metric: autoscalingv2.MetricIdentifier{
					Name: "ingestion_rate",
				},
				Target: autoscalingv2.MetricTarget{
					Type:         autoscalingv2.AverageValueMetricType,
					AverageValue: resource.NewQuantity(100000, resource.DecimalSI), // 100k/sec per pod
				},
			},
		})
	}

	// Define scaling behavior to prevent flapping
	behavior := &autoscalingv2.HorizontalPodAutoscalerBehavior{
		ScaleDown: &autoscalingv2.HPAScalingRules{
			StabilizationWindowSeconds: ptr.To(int32(300)), // 5 minutes
			Policies: []autoscalingv2.HPAScalingPolicy{
				{
					Type:          autoscalingv2.PercentScalingPolicy,
					Value:         10, // Scale down by 10% at a time
					PeriodSeconds: 60,
				},
			},
		},
		ScaleUp: &autoscalingv2.HPAScalingRules{
			StabilizationWindowSeconds: ptr.To(int32(60)), // 1 minute
			Policies: []autoscalingv2.HPAScalingPolicy{
				{
					Type:          autoscalingv2.PercentScalingPolicy,
					Value:         50, // Scale up by 50% at a time
					PeriodSeconds: 60,
				},
				{
					Type:          autoscalingv2.PodsScalingPolicy,
					Value:         2, // But at least 2 pods
					PeriodSeconds: 60,
				},
			},
			SelectPolicy: ptr.To(autoscalingv2.MaxChangePolicySelect),
		},
	}

	return &ScalingPolicy{
		Component:   component,
		MinReplicas: minReplicas,
		MaxReplicas: maxReplicas,
		Metrics:     metrics,
		Behavior:    behavior,
	}
}

// generateStorageOptimizations creates storage optimization recommendations
func (co *CostOptimizer) generateStorageOptimizations(platform *observabilityv1.ObservabilityPlatform, recommendations *OptimizationRecommendations) {
	// Prometheus storage optimization
	if prom := platform.Spec.Components.Prometheus; prom != nil && prom.Enabled {
		if prom.Storage != nil {
			optimization := co.optimizePrometheusStorage(prom)
			if optimization != nil {
				recommendations.StorageOptimizations["prometheus"] = optimization
			}
		}
	}

	// Loki storage optimization
	if loki := platform.Spec.Components.Loki; loki != nil && loki.Enabled {
		if loki.Storage != nil {
			optimization := co.optimizeLokiStorage(loki)
			if optimization != nil {
				recommendations.StorageOptimizations["loki"] = optimization
			}
		}
	}
}

// optimizePrometheusStorage optimizes Prometheus storage configuration
func (co *CostOptimizer) optimizePrometheusStorage(prom *observabilityv1.PrometheusSpec) *StorageOptimization {
	currentSize := prom.Storage.Size
	currentClass := prom.Storage.StorageClassName

	// Calculate recommended size based on retention and metrics
	// Assume 2KB per sample, 10k active series
	samplesPerDay := int64(10000 * 60 * 60 * 24 / 15) // 10k series, 15s scrape interval
	bytesPerDay := samplesPerDay * 2048               // 2KB per sample
	
	retention := prom.Retention
	if retention == "" {
		retention = "15d"
	}
	
	// Parse retention (simplified)
	retentionDays := 15
	if retention == "30d" {
		retentionDays = 30
	} else if retention == "90d" {
		retentionDays = 90
	}
	
	recommendedSize := bytesPerDay * int64(retentionDays) * 2 // 2x for safety
	recommendedSizeGi := recommendedSize / (1024 * 1024 * 1024)
	
	// Round up to nearest 10Gi
	recommendedSizeGi = ((recommendedSizeGi + 9) / 10) * 10
	
	// Recommend storage class based on access patterns
	recommendedClass := "standard" // Default to standard storage
	if retentionDays > 30 {
		recommendedClass = "cold" // Use cheaper storage for long retention
	}

	// Calculate savings
	currentCost := co.calculateStorageCost(currentSize, currentClass)
	recommendedCost := co.calculateStorageCost(fmt.Sprintf("%dGi", recommendedSizeGi), recommendedClass)
	
	if math.Abs(currentCost-recommendedCost) < 10 { // Less than $10 difference
		return nil
	}

	return &StorageOptimization{
		Component:        "prometheus",
		CurrentSize:      currentSize,
		RecommendedSize:  fmt.Sprintf("%dGi", recommendedSizeGi),
		CurrentClass:     currentClass,
		RecommendedClass: recommendedClass,
		EstimatedSavings: currentCost - recommendedCost,
	}
}

// optimizeLokiStorage optimizes Loki storage configuration
func (co *CostOptimizer) optimizeLokiStorage(loki *observabilityv1.LokiSpec) *StorageOptimization {
	// Similar to Prometheus but consider S3 backend
	if loki.S3 != nil && loki.S3.Enabled {
		// When using S3, local storage can be minimal
		if loki.Storage.Size != "10Gi" {
			currentCost := co.calculateStorageCost(loki.Storage.Size, loki.Storage.StorageClassName)
			recommendedCost := co.calculateStorageCost("10Gi", "standard")
			
			return &StorageOptimization{
				Component:        "loki",
				CurrentSize:      loki.Storage.Size,
				RecommendedSize:  "10Gi",
				CurrentClass:     loki.Storage.StorageClassName,
				RecommendedClass: "standard",
				EstimatedSavings: currentCost - recommendedCost,
			}
		}
	}
	
	return nil
}

// getCurrentResources gets current resource allocation for a component
func (co *CostOptimizer) getCurrentResources(component string, platform *observabilityv1.ObservabilityPlatform) *corev1.ResourceRequirements {
	switch component {
	case "prometheus":
		if platform.Spec.Components.Prometheus != nil {
			return platform.Spec.Components.Prometheus.Resources
		}
	case "grafana":
		if platform.Spec.Components.Grafana != nil {
			return platform.Spec.Components.Grafana.Resources
		}
	case "loki":
		if platform.Spec.Components.Loki != nil {
			return platform.Spec.Components.Loki.Resources
		}
	case "tempo":
		if platform.Spec.Components.Tempo != nil {
			return platform.Spec.Components.Tempo.Resources
		}
	}
	return nil
}

// calculateResourceCost estimates monthly cost for resources
func (co *CostOptimizer) calculateResourceCost(resources *corev1.ResourceRequirements) float64 {
	if resources == nil || resources.Requests == nil {
		return 0
	}

	pricing := NewPricingModel() // Use default pricing
	
	cpuCores := resources.Requests.Cpu().AsApproximateFloat64()
	memoryGB := resources.Requests.Memory().AsApproximateFloat64() / (1024 * 1024 * 1024)
	
	// Monthly cost = hourly rate * 24 * 30
	cpuCost := cpuCores * pricing.CPUHourlyRate * 24 * 30
	memoryCost := memoryGB * pricing.MemoryHourlyRate * 24 * 30
	
	return cpuCost + memoryCost
}

// CalculateResourceCost estimates monthly cost for resources (exported for testing)
func (co *CostOptimizer) CalculateResourceCost(resources *corev1.ResourceRequirements) float64 {
	return co.calculateResourceCost(resources)
}

// calculateStorageCost estimates monthly storage cost
func (co *CostOptimizer) calculateStorageCost(size, storageClass string) float64 {
	// Parse size
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return 0
	}
	
	sizeGB := quantity.AsApproximateFloat64() / (1024 * 1024 * 1024)
	pricing := NewPricingModel()
	
	rate := pricing.StorageStandardRate
	switch storageClass {
	case "premium", "fast-ssd":
		rate = pricing.StoragePremiumRate
	case "cold", "archive":
		rate = pricing.StorageArchiveRate
	}
	
	return sizeGB * rate
}

// CalculateStorageCost estimates monthly storage cost (exported for testing)
func (co *CostOptimizer) CalculateStorageCost(size, storageClass string) float64 {
	return co.calculateStorageCost(size, storageClass)
}
