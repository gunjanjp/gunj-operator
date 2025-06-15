package cost

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics for cost tracking
var (
	platformCostEstimate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_platform_cost_estimate_dollars",
			Help: "Estimated cost per platform in dollars",
		},
		[]string{"platform", "namespace", "component"},
	)

	resourceUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_resource_utilization_percent",
			Help: "Resource utilization percentage",
		},
		[]string{"platform", "namespace", "resource_type"},
	)

	potentialSavings = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gunj_operator_potential_savings_dollars",
			Help: "Potential cost savings in dollars",
		},
		[]string{"platform", "namespace", "optimization_type"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		platformCostEstimate,
		resourceUtilization,
		potentialSavings,
	)
}

// Manager handles cost optimization for observability platforms
type Manager struct {
	client    client.Client
	scheme    *runtime.Scheme
	log       logr.Logger
	pricing   *PricingModel
	analyzer  *ResourceAnalyzer
	optimizer *CostOptimizer
}

// NewManager creates a new cost optimization manager
func NewManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *Manager {
	return &Manager{
		client:    client,
		scheme:    scheme,
		log:       log.WithName("cost-manager"),
		pricing:   NewPricingModel(),
		analyzer:  NewResourceAnalyzer(client, log),
		optimizer: NewCostOptimizer(log),
	}
}

// AnalyzePlatformCosts analyzes costs for an observability platform
func (m *Manager) AnalyzePlatformCosts(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*CostAnalysis, error) {
	m.log.V(1).Info("Analyzing platform costs", "platform", platform.Name)

	analysis := &CostAnalysis{
		Platform:   platform.Name,
		Namespace:  platform.Namespace,
		Timestamp:  time.Now(),
		Components: make(map[string]*ComponentCost),
	}

	// Analyze Prometheus costs
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		promCost, err := m.analyzePrometheusCosts(ctx, platform)
		if err != nil {
			return nil, fmt.Errorf("analyzing Prometheus costs: %w", err)
		}
		analysis.Components["prometheus"] = promCost
	}

	// Analyze Grafana costs
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		grafanaCost, err := m.analyzeGrafanaCosts(ctx, platform)
		if err != nil {
			return nil, fmt.Errorf("analyzing Grafana costs: %w", err)
		}
		analysis.Components["grafana"] = grafanaCost
	}

	// Analyze Loki costs
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		lokiCost, err := m.analyzeLokiCosts(ctx, platform)
		if err != nil {
			return nil, fmt.Errorf("analyzing Loki costs: %w", err)
		}
		analysis.Components["loki"] = lokiCost
	}

	// Analyze Tempo costs
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		tempoCost, err := m.analyzeTempoCosts(ctx, platform)
		if err != nil {
			return nil, fmt.Errorf("analyzing Tempo costs: %w", err)
		}
		analysis.Components["tempo"] = tempoCost
	}

	// Calculate total costs
	analysis.calculateTotals()

	// Update metrics
	m.updateCostMetrics(analysis)

	return analysis, nil
}

// GetRecommendations generates cost optimization recommendations
func (m *Manager) GetRecommendations(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*OptimizationRecommendations, error) {
	m.log.V(1).Info("Generating optimization recommendations", "platform", platform.Name)

	// Get current resource usage
	usage, err := m.analyzer.GetResourceUsage(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("getting resource usage: %w", err)
	}

	// Generate recommendations
	recommendations := m.optimizer.GenerateRecommendations(platform, usage)

	// Calculate potential savings
	recommendations.calculateSavings(m.pricing)

	// Update metrics
	m.updateSavingsMetrics(platform, recommendations)

	return recommendations, nil
}

// ApplyOptimizations applies cost optimization recommendations
func (m *Manager) ApplyOptimizations(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, opts *OptimizationOptions) error {
	m.log.Info("Applying cost optimizations", "platform", platform.Name, "options", opts)

	// Get recommendations
	recommendations, err := m.GetRecommendations(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting recommendations: %w", err)
	}

	// Apply resource right-sizing
	if opts.ApplyResourceOptimization {
		if err := m.applyResourceOptimizations(ctx, platform, recommendations); err != nil {
			return fmt.Errorf("applying resource optimizations: %w", err)
		}
	}

	// Apply spot instance configurations
	if opts.EnableSpotInstances {
		if err := m.configureSpotInstances(ctx, platform, recommendations); err != nil {
			return fmt.Errorf("configuring spot instances: %w", err)
		}
	}

	// Apply scaling policies
	if opts.ApplyScalingPolicies {
		if err := m.applyScalingPolicies(ctx, platform, recommendations); err != nil {
			return fmt.Errorf("applying scaling policies: %w", err)
		}
	}

	// Update platform status
	platform.Status.CostOptimization = &observabilityv1.CostOptimizationStatus{
		LastOptimization: metav1.Now(),
		Applied:          true,
		EstimatedSavings: recommendations.TotalSavings,
	}

	return nil
}

// SetBudgetAlert configures budget alerts for a platform
func (m *Manager) SetBudgetAlert(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, budget *BudgetConfig) error {
	m.log.Info("Setting budget alert", "platform", platform.Name, "budget", budget.MonthlyLimit)

	// Store budget configuration
	if platform.Annotations == nil {
		platform.Annotations = make(map[string]string)
	}
	platform.Annotations["cost.observability.io/monthly-budget"] = fmt.Sprintf("%.2f", budget.MonthlyLimit)
	platform.Annotations["cost.observability.io/alert-threshold"] = fmt.Sprintf("%.0f", budget.AlertThreshold)

	// Create or update budget monitoring
	if err := m.createBudgetMonitor(ctx, platform, budget); err != nil {
		return fmt.Errorf("creating budget monitor: %w", err)
	}

	return nil
}

// analyzePrometheusCosts analyzes costs for Prometheus component
func (m *Manager) analyzePrometheusCosts(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*ComponentCost, error) {
	cost := &ComponentCost{
		Component: "prometheus",
		Resources: ResourceCost{},
	}

	// Get Prometheus pods
	podList := &corev1.PodList{}
	selector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/name":       "prometheus",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
	})

	if err := m.client.List(ctx, podList, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return nil, fmt.Errorf("listing Prometheus pods: %w", err)
	}

	// Calculate compute costs
	for _, pod := range podList.Items {
		podCost := m.calculatePodCost(&pod)
		cost.Resources.CPU += podCost.CPU
		cost.Resources.Memory += podCost.Memory
	}

	// Calculate storage costs
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := m.client.List(ctx, pvcList, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return nil, fmt.Errorf("listing Prometheus PVCs: %w", err)
	}

	for _, pvc := range pvcList.Items {
		storageCost := m.calculateStorageCost(&pvc)
		cost.Resources.Storage += storageCost
	}

	// Calculate network costs (egress for remote write)
	if platform.Spec.Components.Prometheus.RemoteWrite != nil {
		cost.Resources.Network = m.estimateNetworkCost("prometheus", platform.Spec.Components.Prometheus)
	}

	cost.calculateTotal()
	return cost, nil
}

// analyzeGrafanaCosts analyzes costs for Grafana component
func (m *Manager) analyzeGrafanaCosts(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*ComponentCost, error) {
	cost := &ComponentCost{
		Component: "grafana",
		Resources: ResourceCost{},
	}

	// Similar implementation to Prometheus
	// Get pods, calculate compute costs
	// Get PVCs, calculate storage costs
	// Estimate network costs

	cost.calculateTotal()
	return cost, nil
}

// analyzeLokiCosts analyzes costs for Loki component
func (m *Manager) analyzeLokiCosts(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*ComponentCost, error) {
	cost := &ComponentCost{
		Component: "loki",
		Resources: ResourceCost{},
	}

	// Implementation similar to Prometheus but with
	// special consideration for S3 storage costs

	cost.calculateTotal()
	return cost, nil
}

// analyzeTempoCosts analyzes costs for Tempo component
func (m *Manager) analyzeTempoCosts(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*ComponentCost, error) {
	cost := &ComponentCost{
		Component: "tempo",
		Resources: ResourceCost{},
	}

	// Implementation similar to Loki

	cost.calculateTotal()
	return cost, nil
}

// calculatePodCost calculates the cost of a single pod
func (m *Manager) calculatePodCost(pod *corev1.Pod) ResourceCost {
	cost := ResourceCost{}

	for _, container := range pod.Spec.Containers {
		// CPU costs
		if container.Resources.Requests.Cpu() != nil {
			cpuCores := container.Resources.Requests.Cpu().AsApproximateFloat64()
			cost.CPU += cpuCores * m.pricing.CPUHourlyRate * 24 * 30 // Monthly cost
		}

		// Memory costs
		if container.Resources.Requests.Memory() != nil {
			memoryGB := container.Resources.Requests.Memory().AsApproximateFloat64() / (1024 * 1024 * 1024)
			cost.Memory += memoryGB * m.pricing.MemoryHourlyRate * 24 * 30 // Monthly cost
		}
	}

	return cost
}

// calculateStorageCost calculates the cost of storage
func (m *Manager) calculateStorageCost(pvc *corev1.PersistentVolumeClaim) float64 {
	if pvc.Spec.Resources.Requests.Storage() == nil {
		return 0
	}

	storageGB := pvc.Spec.Resources.Requests.Storage().AsApproximateFloat64() / (1024 * 1024 * 1024)
	
	// Determine storage class and apply appropriate pricing
	storageClass := pvc.Spec.StorageClassName
	rate := m.pricing.StorageStandardRate // Default rate
	
	if storageClass != nil {
		switch *storageClass {
		case "fast-ssd", "premium":
			rate = m.pricing.StoragePremiumRate
		case "cold", "archive":
			rate = m.pricing.StorageArchiveRate
		}
	}

	return storageGB * rate
}

// estimateNetworkCost estimates network egress costs
func (m *Manager) estimateNetworkCost(component string, spec interface{}) float64 {
	// Estimate based on component type and configuration
	// This is a simplified estimation
	switch component {
	case "prometheus":
		// Estimate based on number of metrics and scrape interval
		return 50.0 // $50/month estimate
	case "loki":
		// Estimate based on log volume
		return 100.0 // $100/month estimate
	default:
		return 25.0 // Default estimate
	}
}

// applyResourceOptimizations applies resource right-sizing
func (m *Manager) applyResourceOptimizations(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, recommendations *OptimizationRecommendations) error {
	// Apply CPU and memory recommendations
	for component, recs := range recommendations.ResourceRecommendations {
		switch component {
		case "prometheus":
			if platform.Spec.Components.Prometheus != nil {
				platform.Spec.Components.Prometheus.Resources = recs.Recommended
			}
		case "grafana":
			if platform.Spec.Components.Grafana != nil {
				platform.Spec.Components.Grafana.Resources = recs.Recommended
			}
		case "loki":
			if platform.Spec.Components.Loki != nil {
				platform.Spec.Components.Loki.Resources = recs.Recommended
			}
		case "tempo":
			if platform.Spec.Components.Tempo != nil {
				platform.Spec.Components.Tempo.Resources = recs.Recommended
			}
		}
	}

	return m.client.Update(ctx, platform)
}

// configureSpotInstances configures components to use spot instances
func (m *Manager) configureSpotInstances(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, recommendations *OptimizationRecommendations) error {
	// Add node selector and tolerations for spot instances
	spotNodeSelector := map[string]string{
		"node.kubernetes.io/instance-type": "spot",
	}

	spotTolerations := []corev1.Toleration{
		{
			Key:      "spot",
			Operator: corev1.TolerationOpEqual,
			Value:    "true",
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}

	// Apply to non-critical components
	if recommendations.SpotInstanceCandidates["grafana"] {
		if platform.Spec.Components.Grafana != nil {
			platform.Spec.Components.Grafana.NodeSelector = spotNodeSelector
			platform.Spec.Components.Grafana.Tolerations = spotTolerations
		}
	}

	// Loki and Tempo can use spot for queriers but not ingesters
	// This would require more detailed component configuration

	return m.client.Update(ctx, platform)
}

// applyScalingPolicies configures autoscaling
func (m *Manager) applyScalingPolicies(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, recommendations *OptimizationRecommendations) error {
	// Apply HPA configurations
	for component, policy := range recommendations.ScalingPolicies {
		switch component {
		case "prometheus":
			if platform.Spec.Components.Prometheus != nil {
				platform.Spec.Components.Prometheus.Autoscaling = &observabilityv1.AutoscalingSpec{
					Enabled:     true,
					MinReplicas: policy.MinReplicas,
					MaxReplicas: policy.MaxReplicas,
					Metrics:     policy.Metrics,
				}
			}
		// Similar for other components
		}
	}

	return m.client.Update(ctx, platform)
}

// createBudgetMonitor creates monitoring for budget alerts
func (m *Manager) createBudgetMonitor(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, budget *BudgetConfig) error {
	// Create PrometheusRule for budget monitoring
	rule := &observabilityv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-budget-alerts", platform.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "budget-monitor",
				"app.kubernetes.io/instance":   platform.Name,
				"app.kubernetes.io/managed-by": "gunj-operator",
			},
		},
		Spec: observabilityv1.PrometheusRuleSpec{
			Groups: []observabilityv1.RuleGroup{
				{
					Name:     "budget_alerts",
					Interval: "5m",
					Rules: []observabilityv1.Rule{
						{
							Alert: "BudgetThresholdExceeded",
							Expr:  fmt.Sprintf(`gunj_operator_platform_cost_estimate_dollars{platform="%s"} > %f`, platform.Name, budget.MonthlyLimit*budget.AlertThreshold/100),
							For:   "10m",
							Labels: map[string]string{
								"severity": "warning",
								"platform": platform.Name,
							},
							Annotations: map[string]string{
								"summary":     fmt.Sprintf("Platform %s is approaching budget limit", platform.Name),
								"description": "Platform {{ $labels.platform }} cost estimate ({{ $value }}) exceeds {{ .threshold }}% of budget",
							},
						},
						{
							Alert: "BudgetLimitExceeded",
							Expr:  fmt.Sprintf(`gunj_operator_platform_cost_estimate_dollars{platform="%s"} > %f`, platform.Name, budget.MonthlyLimit),
							For:   "5m",
							Labels: map[string]string{
								"severity": "critical",
								"platform": platform.Name,
							},
							Annotations: map[string]string{
								"summary":     fmt.Sprintf("Platform %s has exceeded budget limit", platform.Name),
								"description": "Platform {{ $labels.platform }} cost estimate ({{ $value }}) exceeds budget limit of ${{ .limit }}",
							},
						},
					},
				},
			},
		},
	}

	// Create or update the rule
	return m.client.Create(ctx, rule)
}

// updateCostMetrics updates Prometheus metrics for cost tracking
func (m *Manager) updateCostMetrics(analysis *CostAnalysis) {
	for component, cost := range analysis.Components {
		platformCostEstimate.WithLabelValues(
			analysis.Platform,
			analysis.Namespace,
			component,
		).Set(cost.Total)
	}

	// Update total platform cost
	platformCostEstimate.WithLabelValues(
		analysis.Platform,
		analysis.Namespace,
		"total",
	).Set(analysis.TotalCost)
}

// updateSavingsMetrics updates metrics for potential savings
func (m *Manager) updateSavingsMetrics(platform *observabilityv1.ObservabilityPlatform, recommendations *OptimizationRecommendations) {
	// Resource optimization savings
	resourceSavings := 0.0
	for _, rec := range recommendations.ResourceRecommendations {
		resourceSavings += rec.EstimatedSavings
	}
	potentialSavings.WithLabelValues(
		platform.Name,
		platform.Namespace,
		"resource_optimization",
	).Set(resourceSavings)

	// Spot instance savings
	spotSavings := 0.0
	for component, eligible := range recommendations.SpotInstanceCandidates {
		if eligible {
			// Estimate 70% savings for spot instances
			spotSavings += recommendations.ComponentCosts[component] * 0.7
		}
	}
	potentialSavings.WithLabelValues(
		platform.Name,
		platform.Namespace,
		"spot_instances",
	).Set(spotSavings)

	// Total potential savings
	potentialSavings.WithLabelValues(
		platform.Name,
		platform.Namespace,
		"total",
	).Set(recommendations.TotalSavings)
}
