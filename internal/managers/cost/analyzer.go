package cost

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceAnalyzer analyzes resource usage for cost optimization
type ResourceAnalyzer struct {
	client    client.Client
	log       logr.Logger
	promAPI   promv1.API
	lookback  time.Duration
}

// NewResourceAnalyzer creates a new resource analyzer
func NewResourceAnalyzer(client client.Client, log logr.Logger) *ResourceAnalyzer {
	return &ResourceAnalyzer{
		client:   client,
		log:      log.WithName("resource-analyzer"),
		lookback: 7 * 24 * time.Hour, // 7 days lookback by default
	}
}

// GetResourceUsage analyzes resource usage for all components
func (ra *ResourceAnalyzer) GetResourceUsage(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (map[string]*ResourceUsage, error) {
	ra.log.V(1).Info("Analyzing resource usage", "platform", platform.Name)

	usage := make(map[string]*ResourceUsage)

	// Analyze each component
	components := []string{"prometheus", "grafana", "loki", "tempo"}
	for _, component := range components {
		if ra.isComponentEnabled(platform, component) {
			componentUsage, err := ra.analyzeComponentUsage(ctx, platform, component)
			if err != nil {
				ra.log.Error(err, "Failed to analyze component usage", "component", component)
				continue
			}
			usage[component] = componentUsage
		}
	}

	return usage, nil
}

// analyzeComponentUsage analyzes usage for a specific component
func (ra *ResourceAnalyzer) analyzeComponentUsage(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, component string) (*ResourceUsage, error) {
	// Get pods for the component
	podList := &corev1.PodList{}
	selector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/name":       component,
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
	})

	if err := ra.client.List(ctx, podList, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found for component %s", component)
	}

	// Use metrics from the platform's Prometheus if available
	if ra.promAPI != nil {
		return ra.getMetricsBasedUsage(ctx, platform, component, podList)
	}

	// Fallback to pod metrics API
	return ra.getPodMetricsUsage(ctx, podList)
}

// getMetricsBasedUsage gets usage data from Prometheus
func (ra *ResourceAnalyzer) getMetricsBasedUsage(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, component string, podList *corev1.PodList) (*ResourceUsage, error) {
	usage := &ResourceUsage{
		Component: component,
		Timestamp: time.Now(),
	}

	// Query CPU usage percentiles
	cpuQuery := fmt.Sprintf(`
		quantile_over_time(0.5, 
			rate(container_cpu_usage_seconds_total{
				namespace="%s",
				pod=~"%s.*",
				container!="POD"
			}[5m])[%s:5m]
		)`, platform.Namespace, component, ra.lookback.String())

	cpuPercentiles, err := ra.queryPercentiles(ctx, cpuQuery)
	if err != nil {
		return nil, fmt.Errorf("querying CPU percentiles: %w", err)
	}

	// Query memory usage percentiles
	memQuery := fmt.Sprintf(`
		quantile_over_time(0.5,
			container_memory_working_set_bytes{
				namespace="%s",
				pod=~"%s.*",
				container!="POD"
			}[%s:5m]
		)`, platform.Namespace, component, ra.lookback.String())

	memPercentiles, err := ra.queryPercentiles(ctx, memQuery)
	if err != nil {
		return nil, fmt.Errorf("querying memory percentiles: %w", err)
	}

	usage.Percentiles = ResourceUsagePercentiles{
		CPU:    cpuPercentiles,
		Memory: memPercentiles,
	}

	// Set current usage based on P95
	usage.CPUUsage = *resource.NewMilliQuantity(int64(cpuPercentiles.P95*1000), resource.DecimalSI)
	usage.MemoryUsage = *resource.NewQuantity(int64(memPercentiles.P95), resource.BinarySI)

	return usage, nil
}

// getPodMetricsUsage gets usage from pod metrics API (fallback)
func (ra *ResourceAnalyzer) getPodMetricsUsage(ctx context.Context, podList *corev1.PodList) (*ResourceUsage, error) {
	// This would typically use the metrics.k8s.io API
	// For now, we'll estimate based on resource requests
	
	totalCPU := resource.NewQuantity(0, resource.DecimalSI)
	totalMemory := resource.NewQuantity(0, resource.BinarySI)
	
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests.Cpu() != nil {
				totalCPU.Add(*container.Resources.Requests.Cpu())
			}
			if container.Resources.Requests.Memory() != nil {
				totalMemory.Add(*container.Resources.Requests.Memory())
			}
		}
	}

	// Assume 70% utilization as a conservative estimate
	cpuUsage := totalCPU.DeepCopy()
	cpuUsage.Set(int64(float64(cpuUsage.Value()) * 0.7))
	
	memUsage := totalMemory.DeepCopy()
	memUsage.Set(int64(float64(memUsage.Value()) * 0.7))

	return &ResourceUsage{
		Component:   podList.Items[0].Labels["app.kubernetes.io/name"],
		Timestamp:   time.Now(),
		CPUUsage:    cpuUsage,
		MemoryUsage: memUsage,
		Percentiles: ResourceUsagePercentiles{
			CPU: UsagePercentiles{
				P50:  float64(cpuUsage.MilliValue()) / 1000,
				P90:  float64(cpuUsage.MilliValue()) / 1000 * 1.2,
				P95:  float64(cpuUsage.MilliValue()) / 1000 * 1.3,
				P99:  float64(cpuUsage.MilliValue()) / 1000 * 1.5,
				Max:  float64(totalCPU.MilliValue()) / 1000,
				Mean: float64(cpuUsage.MilliValue()) / 1000,
			},
			Memory: UsagePercentiles{
				P50:  float64(memUsage.Value()),
				P90:  float64(memUsage.Value()) * 1.2,
				P95:  float64(memUsage.Value()) * 1.3,
				P99:  float64(memUsage.Value()) * 1.5,
				Max:  float64(totalMemory.Value()),
				Mean: float64(memUsage.Value()),
			},
		},
	}, nil
}

// queryPercentiles queries Prometheus for usage percentiles
func (ra *ResourceAnalyzer) queryPercentiles(ctx context.Context, baseQuery string) (UsagePercentiles, error) {
	percentiles := []float64{0.5, 0.9, 0.95, 0.99}
	results := make([]float64, len(percentiles))

	for i, p := range percentiles {
		query := fmt.Sprintf(baseQuery, p)
		result, _, err := ra.promAPI.Query(ctx, query, time.Now())
		if err != nil {
			return UsagePercentiles{}, fmt.Errorf("querying percentile %.2f: %w", p, err)
		}

		if vector, ok := result.(model.Vector); ok && len(vector) > 0 {
			results[i] = float64(vector[0].Value)
		}
	}

	// Query max value
	maxQuery := fmt.Sprintf("max_over_time(%s[%s])", baseQuery, ra.lookback.String())
	maxResult, _, err := ra.promAPI.Query(ctx, maxQuery, time.Now())
	if err != nil {
		return UsagePercentiles{}, fmt.Errorf("querying max: %w", err)
	}

	maxVal := 0.0
	if vector, ok := maxResult.(model.Vector); ok && len(vector) > 0 {
		maxVal = float64(vector[0].Value)
	}

	// Query average
	avgQuery := fmt.Sprintf("avg_over_time(%s[%s])", baseQuery, ra.lookback.String())
	avgResult, _, err := ra.promAPI.Query(ctx, avgQuery, time.Now())
	if err != nil {
		return UsagePercentiles{}, fmt.Errorf("querying average: %w", err)
	}

	avgVal := 0.0
	if vector, ok := avgResult.(model.Vector); ok && len(vector) > 0 {
		avgVal = float64(vector[0].Value)
	}

	return UsagePercentiles{
		P50:  results[0],
		P90:  results[1],
		P95:  results[2],
		P99:  results[3],
		Max:  maxVal,
		Mean: avgVal,
	}, nil
}

// GetStorageUsage analyzes storage usage and trends
func (ra *ResourceAnalyzer) GetStorageUsage(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (map[string]*StorageUsage, error) {
	storageUsage := make(map[string]*StorageUsage)

	// Get PVCs for the platform
	pvcList := &corev1.PersistentVolumeClaimList{}
	selector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
	})

	if err := ra.client.List(ctx, pvcList, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return nil, fmt.Errorf("listing PVCs: %w", err)
	}

	for _, pvc := range pvcList.Items {
		component := pvc.Labels["app.kubernetes.io/name"]
		if component == "" {
			continue
		}

		usage := &StorageUsage{
			Component:    component,
			PVCName:      pvc.Name,
			StorageClass: *pvc.Spec.StorageClassName,
			Capacity:     pvc.Status.Capacity.Storage().Value(),
		}

		// Query actual usage if Prometheus is available
		if ra.promAPI != nil {
			actualUsage, growthRate, err := ra.queryStorageMetrics(ctx, pvc.Name, platform.Namespace)
			if err == nil {
				usage.Used = actualUsage
				usage.GrowthRatePerDay = growthRate
			}
		}

		storageUsage[component] = usage
	}

	return storageUsage, nil
}

// queryStorageMetrics queries storage usage from Prometheus
func (ra *ResourceAnalyzer) queryStorageMetrics(ctx context.Context, pvcName, namespace string) (int64, float64, error) {
	// Query current usage
	usageQuery := fmt.Sprintf(`kubelet_volume_stats_used_bytes{persistentvolumeclaim="%s", namespace="%s"}`, pvcName, namespace)
	result, _, err := ra.promAPI.Query(ctx, usageQuery, time.Now())
	if err != nil {
		return 0, 0, err
	}

	currentUsage := int64(0)
	if vector, ok := result.(model.Vector); ok && len(vector) > 0 {
		currentUsage = int64(vector[0].Value)
	}

	// Calculate growth rate over the lookback period
	growthQuery := fmt.Sprintf(`
		deriv(kubelet_volume_stats_used_bytes{persistentvolumeclaim="%s", namespace="%s"}[%s])
	`, pvcName, namespace, ra.lookback.String())
	
	growthResult, _, err := ra.promAPI.Query(ctx, growthQuery, time.Now())
	if err != nil {
		return currentUsage, 0, err
	}

	growthRate := 0.0
	if vector, ok := growthResult.(model.Vector); ok && len(vector) > 0 {
		// Convert to daily growth rate
		growthRate = float64(vector[0].Value) * 86400 // seconds to days
	}

	return currentUsage, growthRate, nil
}

// isComponentEnabled checks if a component is enabled
func (ra *ResourceAnalyzer) isComponentEnabled(platform *observabilityv1.ObservabilityPlatform, component string) bool {
	switch component {
	case "prometheus":
		return platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled
	case "grafana":
		return platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled
	case "loki":
		return platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled
	case "tempo":
		return platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled
	default:
		return false
	}
}

// StorageUsage represents storage utilization
type StorageUsage struct {
	Component        string  `json:"component"`
	PVCName          string  `json:"pvcName"`
	StorageClass     string  `json:"storageClass"`
	Capacity         int64   `json:"capacity"`
	Used             int64   `json:"used"`
	GrowthRatePerDay float64 `json:"growthRatePerDay"`
}

// GetUtilizationPercentage returns the utilization percentage
func (su *StorageUsage) GetUtilizationPercentage() float64 {
	if su.Capacity == 0 {
		return 0
	}
	return float64(su.Used) / float64(su.Capacity) * 100
}

// GetDaysUntilFull estimates days until storage is full
func (su *StorageUsage) GetDaysUntilFull() float64 {
	if su.GrowthRatePerDay <= 0 {
		return -1 // Not growing or shrinking
	}
	
	remainingCapacity := su.Capacity - su.Used
	if remainingCapacity <= 0 {
		return 0 // Already full
	}
	
	return float64(remainingCapacity) / su.GrowthRatePerDay
}
