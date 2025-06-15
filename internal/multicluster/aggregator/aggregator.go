package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

const (
	// AggregatorNamespace for aggregator resources
	AggregatorNamespace = "gunj-aggregator"
	// GlobalPrometheusService for federated queries
	GlobalPrometheusService = "global-prometheus"
	// GlobalGrafanaService for unified dashboards
	GlobalGrafanaService = "global-grafana"
)

// MetricSample represents a metric sample from a cluster
type MetricSample struct {
	Cluster   string            `json:"cluster"`
	Metric    string            `json:"metric"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
}

// LogEntry represents a log entry from a cluster
type LogEntry struct {
	Cluster   string            `json:"cluster"`
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels"`
	Source    string            `json:"source"`
}

// Dashboard represents a Grafana dashboard
type Dashboard struct {
	UID       string                 `json:"uid"`
	Title     string                 `json:"title"`
	Panels    []Panel                `json:"panels"`
	Variables []DashboardVariable    `json:"variables"`
	Time      DashboardTime          `json:"time"`
	Refresh   string                 `json:"refresh"`
}

// Panel represents a dashboard panel
type Panel struct {
	ID         int                    `json:"id"`
	Title      string                 `json:"title"`
	Type       string                 `json:"type"`
	Datasource string                 `json:"datasource"`
	Targets    []Target               `json:"targets"`
	GridPos    GridPosition           `json:"gridPos"`
}

// Target represents a query target
type Target struct {
	Expr         string `json:"expr"`
	RefID        string `json:"refId"`
	LegendFormat string `json:"legendFormat"`
}

// GridPosition represents panel position
type GridPosition struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

// DashboardVariable represents a dashboard variable
type DashboardVariable struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Label   string   `json:"label"`
	Options []string `json:"options"`
	Current string   `json:"current"`
}

// DashboardTime represents dashboard time range
type DashboardTime struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// AggregatedMetrics represents aggregated metrics across clusters
type AggregatedMetrics struct {
	Timestamp time.Time                       `json:"timestamp"`
	Clusters  map[string]*ClusterMetrics      `json:"clusters"`
	Global    map[string]float64              `json:"global"`
}

// ClusterMetrics represents metrics from a single cluster
type ClusterMetrics struct {
	ClusterName string             `json:"clusterName"`
	Metrics     map[string]float64 `json:"metrics"`
	Status      string             `json:"status"`
	LastUpdate  time.Time          `json:"lastUpdate"`
}

// Aggregator handles multi-cluster dashboard aggregation
type Aggregator struct {
	// registry for cluster connections
	registry multicluster.ClusterRegistry
	// hubClient for the hub cluster
	hubClient client.Client
	// log is the logger
	log logr.Logger
	// prometheusClients for each cluster
	prometheusClients map[string]promv1.API
	// dashboards stores aggregated dashboards
	dashboards map[string]*Dashboard
	// metrics stores recent metrics
	metrics *MetricsBuffer
	// logs stores recent logs
	logs *LogBuffer
	// mu for thread safety
	mu sync.RWMutex
	// stopCh for stopping aggregation
	stopCh chan struct{}
}

// MetricsBuffer stores recent metrics with circular buffer
type MetricsBuffer struct {
	samples []MetricSample
	size    int
	head    int
	mu      sync.RWMutex
}

// LogBuffer stores recent logs with circular buffer
type LogBuffer struct {
	entries []LogEntry
	size    int
	head    int
	mu      sync.RWMutex
}

// NewAggregator creates a new dashboard aggregator
func NewAggregator(registry multicluster.ClusterRegistry, hubClient client.Client) *Aggregator {
	aggregator := &Aggregator{
		registry:          registry,
		hubClient:         hubClient,
		log:               log.Log.WithName("dashboard-aggregator"),
		prometheusClients: make(map[string]promv1.API),
		dashboards:        make(map[string]*Dashboard),
		metrics:           NewMetricsBuffer(10000),
		logs:              NewLogBuffer(10000),
		stopCh:            make(chan struct{}),
	}

	// Start aggregation loops
	go aggregator.aggregateMetrics()
	go aggregator.aggregateLogs()
	go aggregator.aggregateDashboards()

	return aggregator
}

// GetAggregatedMetrics returns aggregated metrics across all clusters
func (a *Aggregator) GetAggregatedMetrics(ctx context.Context) (*AggregatedMetrics, error) {
	a.log.V(2).Info("Getting aggregated metrics")

	clusters, err := a.registry.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	aggregated := &AggregatedMetrics{
		Timestamp: time.Now(),
		Clusters:  make(map[string]*ClusterMetrics),
		Global:    make(map[string]float64),
	}

	var wg sync.WaitGroup
	for _, cluster := range clusters {
		wg.Add(1)
		go func(c *multicluster.Cluster) {
			defer wg.Done()
			
			metrics, err := a.getClusterMetrics(ctx, c.Name)
			if err != nil {
				a.log.Error(err, "Failed to get cluster metrics", "cluster", c.Name)
				metrics = &ClusterMetrics{
					ClusterName: c.Name,
					Status:      "error",
					LastUpdate:  time.Now(),
				}
			}
			
			a.mu.Lock()
			aggregated.Clusters[c.Name] = metrics
			a.mu.Unlock()
		}(cluster)
	}

	wg.Wait()

	// Calculate global aggregates
	a.calculateGlobalMetrics(aggregated)

	return aggregated, nil
}

// GetGlobalDashboard returns a global dashboard with multi-cluster views
func (a *Aggregator) GetGlobalDashboard(ctx context.Context, dashboardUID string) (*Dashboard, error) {
	a.mu.RLock()
	dashboard, exists := a.dashboards[dashboardUID]
	a.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("dashboard %s not found", dashboardUID)
	}

	// Clone dashboard
	globalDashboard := a.cloneDashboard(dashboard)

	// Add cluster selector variable
	globalDashboard.Variables = append(globalDashboard.Variables, DashboardVariable{
		Name:    "cluster",
		Type:    "query",
		Label:   "Cluster",
		Options: a.getClusterNames(ctx),
		Current: "all",
	})

	// Transform panels for multi-cluster
	for i := range globalDashboard.Panels {
		a.transformPanelForMultiCluster(&globalDashboard.Panels[i])
	}

	return globalDashboard, nil
}

// SearchLogs searches logs across all clusters
func (a *Aggregator) SearchLogs(ctx context.Context, query string, timeRange time.Duration) ([]LogEntry, error) {
	a.log.V(2).Info("Searching logs", "query", query, "timeRange", timeRange)

	// Get recent logs from buffer
	logs := a.logs.GetRecent(timeRange)

	// Filter logs based on query
	filtered := []LogEntry{}
	for _, log := range logs {
		if a.matchesLogQuery(log, query) {
			filtered = append(filtered, log)
		}
	}

	return filtered, nil
}

// CreateFederatedQuery creates a federated Prometheus query
func (a *Aggregator) CreateFederatedQuery(query string, clusters []string) string {
	if len(clusters) == 0 {
		// Query all clusters
		return fmt.Sprintf(`group by (cluster) (%s)`, query)
	}

	// Query specific clusters
	clusterRegex := ""
	for i, cluster := range clusters {
		if i > 0 {
			clusterRegex += "|"
		}
		clusterRegex += cluster
	}

	return fmt.Sprintf(`%s{cluster=~"%s"}`, query, clusterRegex)
}

// aggregateMetrics continuously aggregates metrics from all clusters
func (a *Aggregator) aggregateMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			clusters, err := a.registry.List(ctx)
			if err != nil {
				a.log.Error(err, "Failed to list clusters")
				continue
			}

			for _, cluster := range clusters {
				go a.collectClusterMetrics(ctx, cluster)
			}
		case <-a.stopCh:
			return
		}
	}
}

// collectClusterMetrics collects metrics from a single cluster
func (a *Aggregator) collectClusterMetrics(ctx context.Context, cluster *multicluster.Cluster) {
	// Get Prometheus client for cluster
	promClient, err := a.getPrometheusClient(ctx, cluster.Name)
	if err != nil {
		a.log.Error(err, "Failed to get Prometheus client", "cluster", cluster.Name)
		return
	}

	// Query key metrics
	queries := []struct {
		name  string
		query string
	}{
		{"cpu_usage", `sum(rate(container_cpu_usage_seconds_total[5m]))`},
		{"memory_usage", `sum(container_memory_usage_bytes)`},
		{"pod_count", `count(kube_pod_info)`},
		{"node_count", `count(kube_node_info)`},
		{"error_rate", `sum(rate(http_requests_total{status=~"5.."}[5m]))`},
	}

	for _, q := range queries {
		result, _, err := promClient.Query(ctx, q.query, time.Now())
		if err != nil {
			a.log.Error(err, "Failed to query metric", "cluster", cluster.Name, "metric", q.name)
			continue
		}

		// Process result
		if vector, ok := result.(model.Vector); ok {
			for _, sample := range vector {
				a.metrics.Add(MetricSample{
					Cluster:   cluster.Name,
					Metric:    q.name,
					Value:     float64(sample.Value),
					Timestamp: time.Now(),
					Labels:    a.modelLabelsToMap(sample.Metric),
				})
			}
		}
	}
}

// aggregateLogs continuously aggregates logs from all clusters
func (a *Aggregator) aggregateLogs() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			clusters, err := a.registry.List(ctx)
			if err != nil {
				a.log.Error(err, "Failed to list clusters")
				continue
			}

			for _, cluster := range clusters {
				go a.collectClusterLogs(ctx, cluster)
			}
		case <-a.stopCh:
			return
		}
	}
}

// collectClusterLogs collects logs from a single cluster
func (a *Aggregator) collectClusterLogs(ctx context.Context, cluster *multicluster.Cluster) {
	// In a real implementation, this would query Loki or similar
	// For now, we'll simulate log collection
	
	// This is a placeholder - actual implementation would:
	// 1. Connect to cluster's Loki instance
	// 2. Query recent logs
	// 3. Stream logs to the aggregator
}

// aggregateDashboards aggregates dashboards from all clusters
func (a *Aggregator) aggregateDashboards() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if err := a.syncDashboards(ctx); err != nil {
				a.log.Error(err, "Failed to sync dashboards")
			}
		case <-a.stopCh:
			return
		}
	}
}

// syncDashboards syncs dashboards from all clusters
func (a *Aggregator) syncDashboards(ctx context.Context) error {
	clusters, err := a.registry.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	dashboards := make(map[string]*Dashboard)
	
	for _, cluster := range clusters {
		clusterDashboards, err := a.getClusterDashboards(ctx, cluster.Name)
		if err != nil {
			a.log.Error(err, "Failed to get dashboards", "cluster", cluster.Name)
			continue
		}

		// Merge dashboards
		for uid, dashboard := range clusterDashboards {
			if existing, exists := dashboards[uid]; exists {
				// Merge panels from multiple clusters
				a.mergeDashboards(existing, dashboard)
			} else {
				dashboards[uid] = dashboard
			}
		}
	}

	a.mu.Lock()
	a.dashboards = dashboards
	a.mu.Unlock()

	return nil
}

// getClusterMetrics gets metrics from a specific cluster
func (a *Aggregator) getClusterMetrics(ctx context.Context, clusterName string) (*ClusterMetrics, error) {
	// Get recent metrics from buffer
	samples := a.metrics.GetByCluster(clusterName, 5*time.Minute)
	
	metrics := &ClusterMetrics{
		ClusterName: clusterName,
		Metrics:     make(map[string]float64),
		Status:      "healthy",
		LastUpdate:  time.Now(),
	}

	// Aggregate samples by metric name
	for _, sample := range samples {
		metrics.Metrics[sample.Metric] = sample.Value
	}

	return metrics, nil
}

// calculateGlobalMetrics calculates global aggregated metrics
func (a *Aggregator) calculateGlobalMetrics(aggregated *AggregatedMetrics) {
	metricSums := make(map[string]float64)
	metricCounts := make(map[string]int)

	// Sum metrics across clusters
	for _, clusterMetrics := range aggregated.Clusters {
		for metric, value := range clusterMetrics.Metrics {
			metricSums[metric] += value
			metricCounts[metric]++
		}
	}

	// Calculate aggregates
	for metric, sum := range metricSums {
		aggregated.Global[metric+"_sum"] = sum
		if count := metricCounts[metric]; count > 0 {
			aggregated.Global[metric+"_avg"] = sum / float64(count)
		}
	}

	// Add cluster count
	aggregated.Global["cluster_count"] = float64(len(aggregated.Clusters))
	
	// Calculate health percentage
	healthyCount := 0
	for _, cluster := range aggregated.Clusters {
		if cluster.Status == "healthy" {
			healthyCount++
		}
	}
	if len(aggregated.Clusters) > 0 {
		aggregated.Global["cluster_health_percentage"] = float64(healthyCount) / float64(len(aggregated.Clusters)) * 100
	}
}

// getPrometheusClient gets or creates a Prometheus client for a cluster
func (a *Aggregator) getPrometheusClient(ctx context.Context, clusterName string) (promv1.API, error) {
	a.mu.RLock()
	client, exists := a.prometheusClients[clusterName]
	a.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Create new client
	// In a real implementation, this would:
	// 1. Get cluster connection
	// 2. Find Prometheus endpoint
	// 3. Create authenticated client
	
	// For now, return a mock
	return nil, fmt.Errorf("prometheus client not implemented")
}

// getClusterDashboards gets dashboards from a specific cluster
func (a *Aggregator) getClusterDashboards(ctx context.Context, clusterName string) (map[string]*Dashboard, error) {
	// In a real implementation, this would:
	// 1. Connect to cluster's Grafana
	// 2. List dashboards
	// 3. Export dashboard JSON
	
	// For now, return empty map
	return make(map[string]*Dashboard), nil
}

// transformPanelForMultiCluster transforms a panel for multi-cluster view
func (a *Aggregator) transformPanelForMultiCluster(panel *Panel) {
	// Add cluster label to queries
	for i := range panel.Targets {
		// Add cluster variable to query
		if panel.Targets[i].Expr != "" {
			panel.Targets[i].Expr = fmt.Sprintf(`%s * on(cluster) group_left() up{cluster="$cluster"}`, panel.Targets[i].Expr)
		}
		
		// Update legend to include cluster
		if panel.Targets[i].LegendFormat != "" {
			panel.Targets[i].LegendFormat = "{{cluster}} - " + panel.Targets[i].LegendFormat
		}
	}
}

// cloneDashboard creates a deep copy of a dashboard
func (a *Aggregator) cloneDashboard(dashboard *Dashboard) *Dashboard {
	// Simple deep copy using JSON
	data, _ := json.Marshal(dashboard)
	var clone Dashboard
	json.Unmarshal(data, &clone)
	return &clone
}

// mergeDashboards merges two dashboards
func (a *Aggregator) mergeDashboards(target, source *Dashboard) {
	// Merge panels
	panelMap := make(map[int]bool)
	for _, panel := range target.Panels {
		panelMap[panel.ID] = true
	}
	
	for _, panel := range source.Panels {
		if !panelMap[panel.ID] {
			target.Panels = append(target.Panels, panel)
		}
	}
	
	// Merge variables
	varMap := make(map[string]bool)
	for _, v := range target.Variables {
		varMap[v.Name] = true
	}
	
	for _, v := range source.Variables {
		if !varMap[v.Name] {
			target.Variables = append(target.Variables, v)
		}
	}
}

// getClusterNames gets all cluster names
func (a *Aggregator) getClusterNames(ctx context.Context) []string {
	clusters, err := a.registry.List(ctx)
	if err != nil {
		return []string{}
	}

	names := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		names = append(names, cluster.Name)
	}
	
	return names
}

// matchesLogQuery checks if a log entry matches a query
func (a *Aggregator) matchesLogQuery(log LogEntry, query string) bool {
	// Simple string matching - in production, use proper query parser
	return contains(log.Message, query) || 
		contains(log.Cluster, query) ||
		contains(log.Level, query)
}

// modelLabelsToMap converts Prometheus model labels to map
func (a *Aggregator) modelLabelsToMap(labels model.Metric) map[string]string {
	result := make(map[string]string)
	for k, v := range labels {
		result[string(k)] = string(v)
	}
	return result
}

// NewMetricsBuffer creates a new metrics buffer
func NewMetricsBuffer(size int) *MetricsBuffer {
	return &MetricsBuffer{
		samples: make([]MetricSample, size),
		size:    size,
		head:    0,
	}
}

// Add adds a metric sample to the buffer
func (b *MetricsBuffer) Add(sample MetricSample) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.samples[b.head] = sample
	b.head = (b.head + 1) % b.size
}

// GetByCluster gets recent samples for a cluster
func (b *MetricsBuffer) GetByCluster(cluster string, duration time.Duration) []MetricSample {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	cutoff := time.Now().Add(-duration)
	result := []MetricSample{}
	
	for i := 0; i < b.size; i++ {
		idx := (b.head - 1 - i + b.size) % b.size
		sample := b.samples[idx]
		
		if sample.Timestamp.IsZero() || sample.Timestamp.Before(cutoff) {
			break
		}
		
		if sample.Cluster == cluster {
			result = append(result, sample)
		}
	}
	
	return result
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(size int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, size),
		size:    size,
		head:    0,
	}
}

// Add adds a log entry to the buffer
func (b *LogBuffer) Add(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.entries[b.head] = entry
	b.head = (b.head + 1) % b.size
}

// GetRecent gets recent log entries
func (b *LogBuffer) GetRecent(duration time.Duration) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	cutoff := time.Now().Add(-duration)
	result := []LogEntry{}
	
	for i := 0; i < b.size; i++ {
		idx := (b.head - 1 - i + b.size) % b.size
		entry := b.entries[idx]
		
		if entry.Timestamp.IsZero() || entry.Timestamp.Before(cutoff) {
			break
		}
		
		result = append(result, entry)
	}
	
	return result
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && s[0:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s[1:len(s)-1], substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Stop stops the aggregator
func (a *Aggregator) Stop() {
	close(a.stopCh)
}
