package autoscaling

import (
	"context"
	"fmt"
	"time"

	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PrometheusMetricsProvider provides metrics from Prometheus
type PrometheusMetricsProvider struct {
	client       client.Client
	promClient   promv1.API
	namespace    string
	platformName string
}

// NewPrometheusMetricsProvider creates a new Prometheus metrics provider
func NewPrometheusMetricsProvider(client client.Client, promEndpoint string, namespace string, platformName string) (*PrometheusMetricsProvider, error) {
	config := api.Config{
		Address: promEndpoint,
	}
	
	promClient, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}
	
	return &PrometheusMetricsProvider{
		client:       client,
		promClient:   promv1.NewAPI(promClient),
		namespace:    namespace,
		platformName: platformName,
	}, nil
}

// GetCPUUtilization gets the CPU utilization for a component
func (m *PrometheusMetricsProvider) GetCPUUtilization(ctx context.Context, component v1beta1.ComponentType) (float64, error) {
	log := log.FromContext(ctx)
	
	// Build Prometheus query
	query := fmt.Sprintf(`
		avg(
			rate(container_cpu_usage_seconds_total{
				namespace="%s",
				pod=~"%s-%s-.*",
				container="%s"
			}[5m])
		) / 
		avg(
			kube_pod_container_resource_requests{
				namespace="%s",
				pod=~"%s-%s-.*",
				container="%s",
				resource="cpu"
			}
		) * 100
	`, m.namespace, m.platformName, component, component,
	   m.namespace, m.platformName, component, component)
	
	result, warnings, err := m.promClient.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to query CPU utilization: %w", err)
	}
	
	if len(warnings) > 0 {
		log.Info("Prometheus query warnings", "warnings", warnings)
	}
	
	return m.extractScalarValue(result)
}

// GetMemoryUtilization gets the memory utilization for a component
func (m *PrometheusMetricsProvider) GetMemoryUtilization(ctx context.Context, component v1beta1.ComponentType) (float64, error) {
	log := log.FromContext(ctx)
	
	// Build Prometheus query
	query := fmt.Sprintf(`
		avg(
			container_memory_working_set_bytes{
				namespace="%s",
				pod=~"%s-%s-.*",
				container="%s"
			}
		) / 
		avg(
			kube_pod_container_resource_requests{
				namespace="%s",
				pod=~"%s-%s-.*",
				container="%s",
				resource="memory"
			}
		) * 100
	`, m.namespace, m.platformName, component, component,
	   m.namespace, m.platformName, component, component)
	
	result, warnings, err := m.promClient.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to query memory utilization: %w", err)
	}
	
	if len(warnings) > 0 {
		log.Info("Prometheus query warnings", "warnings", warnings)
	}
	
	return m.extractScalarValue(result)
}

// GetCustomMetric gets a custom metric value
func (m *PrometheusMetricsProvider) GetCustomMetric(ctx context.Context, component v1beta1.ComponentType, metric CustomMetric) (float64, error) {
	log := log.FromContext(ctx)
	
	// Replace placeholders in the query
	query := m.replaceQueryPlaceholders(metric.Query, component)
	
	result, warnings, err := m.promClient.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to query custom metric %s: %w", metric.Name, err)
	}
	
	if len(warnings) > 0 {
		log.Info("Prometheus query warnings", "metric", metric.Name, "warnings", warnings)
	}
	
	return m.extractScalarValue(result)
}

// GetHistoricalMetrics gets historical metrics for a component
func (m *PrometheusMetricsProvider) GetHistoricalMetrics(ctx context.Context, component v1beta1.ComponentType, duration time.Duration) ([]MetricDataPoint, error) {
	log := log.FromContext(ctx)
	
	endTime := time.Now()
	startTime := endTime.Add(-duration)
	
	// Query for CPU usage over time
	query := fmt.Sprintf(`
		avg(
			rate(container_cpu_usage_seconds_total{
				namespace="%s",
				pod=~"%s-%s-.*",
				container="%s"
			}[5m])
		)
	`, m.namespace, m.platformName, component, component)
	
	result, warnings, err := m.promClient.QueryRange(ctx, query, promv1.Range{
		Start: startTime,
		End:   endTime,
		Step:  5 * time.Minute,
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to query historical metrics: %w", err)
	}
	
	if len(warnings) > 0 {
		log.Info("Prometheus query warnings", "warnings", warnings)
	}
	
	return m.convertToDataPoints(result, "cpu_usage")
}

// GetPodMetrics gets metrics for specific pods
func (m *PrometheusMetricsProvider) GetPodMetrics(ctx context.Context, component v1beta1.ComponentType) (map[string]PodMetrics, error) {
	// List pods for the component
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(map[string]string{
		"app.kubernetes.io/name":     string(component),
		"app.kubernetes.io/instance": m.platformName,
	})
	
	if err := m.client.List(ctx, podList, &client.ListOptions{
		Namespace:     m.namespace,
		LabelSelector: labelSelector,
	}); err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	
	podMetrics := make(map[string]PodMetrics)
	
	for _, pod := range podList.Items {
		// Get CPU usage
		cpuQuery := fmt.Sprintf(`
			rate(container_cpu_usage_seconds_total{
				namespace="%s",
				pod="%s",
				container="%s"
			}[5m])
		`, m.namespace, pod.Name, component)
		
		cpuResult, _, err := m.promClient.Query(ctx, cpuQuery, time.Now())
		if err != nil {
			continue
		}
		
		cpuValue, _ := m.extractScalarValue(cpuResult)
		
		// Get memory usage
		memQuery := fmt.Sprintf(`
			container_memory_working_set_bytes{
				namespace="%s",
				pod="%s",
				container="%s"
			}
		`, m.namespace, pod.Name, component)
		
		memResult, _, err := m.promClient.Query(ctx, memQuery, time.Now())
		if err != nil {
			continue
		}
		
		memValue, _ := m.extractScalarValue(memResult)
		
		podMetrics[pod.Name] = PodMetrics{
			PodName:     pod.Name,
			CPUUsage:    cpuValue,
			MemoryUsage: memValue,
			Timestamp:   time.Now(),
		}
	}
	
	return podMetrics, nil
}

// extractScalarValue extracts a scalar value from Prometheus result
func (m *PrometheusMetricsProvider) extractScalarValue(result model.Value) (float64, error) {
	switch v := result.(type) {
	case model.Vector:
		if len(v) > 0 {
			return float64(v[0].Value), nil
		}
		return 0, fmt.Errorf("empty result vector")
	case *model.Scalar:
		return float64(v.Value), nil
	default:
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// convertToDataPoints converts Prometheus result to data points
func (m *PrometheusMetricsProvider) convertToDataPoints(result model.Value, metricName string) ([]MetricDataPoint, error) {
	dataPoints := []MetricDataPoint{}
	
	switch v := result.(type) {
	case model.Matrix:
		for _, series := range v {
			for _, sample := range series.Values {
				dataPoints = append(dataPoints, MetricDataPoint{
					Timestamp: sample.Timestamp.Time(),
					Value:     float64(sample.Value),
					Labels: map[string]string{
						"metric": metricName,
					},
				})
			}
		}
	default:
		return nil, fmt.Errorf("unexpected result type for range query: %T", result)
	}
	
	return dataPoints, nil
}

// replaceQueryPlaceholders replaces placeholders in custom queries
func (m *PrometheusMetricsProvider) replaceQueryPlaceholders(query string, component v1beta1.ComponentType) string {
	replacements := map[string]string{
		"$namespace":  m.namespace,
		"$platform":   m.platformName,
		"$component":  string(component),
		"${namespace}": m.namespace,
		"${platform}":  m.platformName,
		"${component}": string(component),
	}
	
	result := query
	for placeholder, value := range replacements {
		result = fmt.Sprintf(result, value)
	}
	
	return result
}

// PodMetrics represents metrics for a single pod
type PodMetrics struct {
	PodName     string
	CPUUsage    float64 // in cores
	MemoryUsage float64 // in bytes
	Timestamp   time.Time
}

// MockMetricsProvider is a mock implementation for testing
type MockMetricsProvider struct {
	CPUUtilization    float64
	MemoryUtilization float64
	CustomMetricValue float64
	HistoricalData    []MetricDataPoint
}

// GetCPUUtilization returns mock CPU utilization
func (m *MockMetricsProvider) GetCPUUtilization(ctx context.Context, component v1beta1.ComponentType) (float64, error) {
	return m.CPUUtilization, nil
}

// GetMemoryUtilization returns mock memory utilization
func (m *MockMetricsProvider) GetMemoryUtilization(ctx context.Context, component v1beta1.ComponentType) (float64, error) {
	return m.MemoryUtilization, nil
}

// GetCustomMetric returns mock custom metric value
func (m *MockMetricsProvider) GetCustomMetric(ctx context.Context, component v1beta1.ComponentType, metric CustomMetric) (float64, error) {
	return m.CustomMetricValue, nil
}

// GetHistoricalMetrics returns mock historical data
func (m *MockMetricsProvider) GetHistoricalMetrics(ctx context.Context, component v1beta1.ComponentType, duration time.Duration) ([]MetricDataPoint, error) {
	return m.HistoricalData, nil
}
