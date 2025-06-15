package health

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

const (
	// DefaultHealthCheckInterval is the default interval for health checks
	DefaultHealthCheckInterval = 30 * time.Second
	// DefaultHealthCheckTimeout is the default timeout for health checks
	DefaultHealthCheckTimeout = 10 * time.Second
	// DefaultLatencyThreshold is the threshold for considering high latency
	DefaultLatencyThreshold = 500 * time.Millisecond
)

// HealthChecker implements cluster health monitoring
type HealthChecker struct {
	// registry for cluster connections
	registry multicluster.ClusterRegistry
	// log is the logger
	log logr.Logger
	// interval is the health check interval
	interval time.Duration
	// timeout for health checks
	timeout time.Duration
	// watchers stores health watchers
	watchers map[string]chan *multicluster.ClusterHealth
	// lastHealth stores the last health status
	lastHealth map[string]*multicluster.ClusterHealth
	// mu for thread safety
	mu sync.RWMutex
	// stopCh for stopping background checks
	stopCh chan struct{}
}

// NewHealthChecker creates a new cluster health checker
func NewHealthChecker(registry multicluster.ClusterRegistry) multicluster.ClusterHealthChecker {
	checker := &HealthChecker{
		registry:   registry,
		log:        log.Log.WithName("cluster-health-checker"),
		interval:   DefaultHealthCheckInterval,
		timeout:    DefaultHealthCheckTimeout,
		watchers:   make(map[string]chan *multicluster.ClusterHealth),
		lastHealth: make(map[string]*multicluster.ClusterHealth),
		stopCh:     make(chan struct{}),
	}

	// Start background health checking
	go checker.runHealthChecks()

	return checker
}

// CheckHealth performs a health check on a cluster
func (h *HealthChecker) CheckHealth(ctx context.Context, clusterName string) (*multicluster.ClusterHealth, error) {
	h.log.V(2).Info("Checking cluster health", "cluster", clusterName)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	// Get cluster connection
	conn, err := h.registry.GetConnection(ctx, clusterName)
	if err != nil {
		return h.createUnhealthyStatus(clusterName, "ConnectionFailed", err), nil
	}

	health := &multicluster.ClusterHealth{
		ClusterName: clusterName,
		Status:      multicluster.ClusterStatusReady,
		Components:  []multicluster.ComponentHealth{},
		LastCheck:   time.Now(),
		NextCheck:   time.Now().Add(h.interval),
	}

	// Check API server connectivity
	connectivity := h.checkConnectivity(ctx, conn)
	health.Connectivity = connectivity

	if !connectivity.APIServer {
		health.Status = multicluster.ClusterStatusOffline
		h.updateHealthCache(clusterName, health)
		return health, nil
	}

	// Check cluster components
	components := h.checkComponents(ctx, conn)
	health.Components = components

	// Check nodes
	nodeHealth := h.checkNodes(ctx, conn)
	health.Nodes = nodeHealth

	// Determine overall status
	health.Status = h.determineOverallStatus(components, nodeHealth, connectivity)

	// Update cache and notify watchers
	h.updateHealthCache(clusterName, health)
	h.notifyWatchers(clusterName, health)

	h.log.V(1).Info("Cluster health check completed", "cluster", clusterName, "status", health.Status)
	return health, nil
}

// CheckAllClusters checks health of all registered clusters
func (h *HealthChecker) CheckAllClusters(ctx context.Context) (map[string]*multicluster.ClusterHealth, error) {
	h.log.Info("Checking health of all clusters")

	clusters, err := h.registry.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	results := make(map[string]*multicluster.ClusterHealth)
	var wg sync.WaitGroup

	for _, cluster := range clusters {
		wg.Add(1)
		go func(c *multicluster.Cluster) {
			defer wg.Done()
			
			health, err := h.CheckHealth(ctx, c.Name)
			if err != nil {
				h.log.Error(err, "Failed to check cluster health", "cluster", c.Name)
				health = h.createUnhealthyStatus(c.Name, "HealthCheckFailed", err)
			}
			
			h.mu.Lock()
			results[c.Name] = health
			h.mu.Unlock()
		}(cluster)
	}

	wg.Wait()
	return results, nil
}

// WatchHealth watches for health changes
func (h *HealthChecker) WatchHealth(ctx context.Context, clusterName string) (<-chan *multicluster.ClusterHealth, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Create new watcher channel
	ch := make(chan *multicluster.ClusterHealth, 10)
	h.watchers[clusterName] = ch

	// Send current health status if available
	if health, exists := h.lastHealth[clusterName]; exists {
		select {
		case ch <- health:
		default:
		}
	}

	// Cleanup on context cancellation
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.watchers, clusterName)
		close(ch)
		h.mu.Unlock()
	}()

	return ch, nil
}

// SetHealthCheckInterval updates the health check interval
func (h *HealthChecker) SetHealthCheckInterval(interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.interval = interval
	h.log.Info("Updated health check interval", "interval", interval)
}

// runHealthChecks runs periodic health checks in the background
func (h *HealthChecker) runHealthChecks() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if _, err := h.CheckAllClusters(ctx); err != nil {
				h.log.Error(err, "Failed to check all clusters")
			}
		case <-h.stopCh:
			h.log.Info("Stopping health checker")
			return
		}
	}
}

// checkConnectivity checks network connectivity to the cluster
func (h *HealthChecker) checkConnectivity(ctx context.Context, conn *multicluster.ClusterConnection) multicluster.ConnectivityStatus {
	connectivity := multicluster.ConnectivityStatus{
		APIServer:          false,
		ServiceEndpoints:   make(map[string]bool),
		LastSuccessfulPing: conn.LastConnected,
	}

	// Measure API server latency
	start := time.Now()
	version, err := conn.Client.Discovery().ServerVersion()
	if err != nil {
		h.log.Error(err, "Failed to connect to API server", "cluster", conn.Cluster.Name)
		return connectivity
	}
	latency := time.Since(start)

	connectivity.APIServer = true
	connectivity.Latency = latency.Milliseconds()
	connectivity.LastSuccessfulPing = time.Now()

	h.log.V(2).Info("API server connectivity check", 
		"cluster", conn.Cluster.Name, 
		"version", version.GitVersion,
		"latency", latency)

	// Check key service endpoints
	services := []string{"kube-dns", "metrics-server"}
	for _, svc := range services {
		endpoints, err := conn.Client.CoreV1().Endpoints("kube-system").Get(ctx, svc, metav1.GetOptions{})
		if err != nil {
			connectivity.ServiceEndpoints[svc] = false
			continue
		}
		
		// Check if endpoints have addresses
		hasEndpoints := false
		for _, subset := range endpoints.Subsets {
			if len(subset.Addresses) > 0 {
				hasEndpoints = true
				break
			}
		}
		connectivity.ServiceEndpoints[svc] = hasEndpoints
	}

	// Check network mesh connectivity if enabled
	if conn.Cluster.Features != nil {
		for _, feature := range conn.Cluster.Features {
			if feature == "istio" || feature == "linkerd" {
				connectivity.NetworkMesh = h.checkNetworkMesh(ctx, conn, feature)
				break
			}
		}
	}

	return connectivity
}

// checkComponents checks the health of cluster components
func (h *HealthChecker) checkComponents(ctx context.Context, conn *multicluster.ClusterConnection) []multicluster.ComponentHealth {
	components := []multicluster.ComponentHealth{}

	// Check system pods in kube-system namespace
	systemComponents := map[string]string{
		"kube-apiserver":          "api-server",
		"kube-controller-manager": "controller-manager",
		"kube-scheduler":          "scheduler",
		"etcd":                    "etcd",
		"kube-proxy":              "proxy",
		"coredns":                 "dns",
	}

	pods, err := conn.Client.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		h.log.Error(err, "Failed to list system pods")
		return components
	}

	componentStatus := make(map[string]*multicluster.ComponentHealth)
	
	for _, pod := range pods.Items {
		for prefix, compType := range systemComponents {
			if strings.HasPrefix(pod.Name, prefix) {
				if _, exists := componentStatus[compType]; !exists {
					componentStatus[compType] = &multicluster.ComponentHealth{
						Name:   compType,
						Type:   compType,
						Status: "Unknown",
					}
				}

				// Check pod status
				if pod.Status.Phase == corev1.PodRunning {
					allReady := true
					for _, condition := range pod.Status.Conditions {
						if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
							allReady = false
							break
						}
					}
					if allReady {
						componentStatus[compType].Status = "Healthy"
					} else {
						componentStatus[compType].Status = "Degraded"
						componentStatus[compType].Message = "Pod not ready"
					}
				} else {
					componentStatus[compType].Status = "Unhealthy"
					componentStatus[compType].Message = fmt.Sprintf("Pod phase: %s", pod.Status.Phase)
				}
				
				componentStatus[compType].LastTransition = pod.CreationTimestamp.Time
			}
		}
	}

	// Convert map to slice
	for _, comp := range componentStatus {
		components = append(components, *comp)
	}

	// Check additional components via API
	h.checkAdditionalComponents(ctx, conn, &components)

	return components
}

// checkAdditionalComponents checks additional cluster components
func (h *HealthChecker) checkAdditionalComponents(ctx context.Context, conn *multicluster.ClusterConnection, components *[]multicluster.ComponentHealth) {
	// Check metrics server
	_, err := conn.Client.AppsV1().Deployments("kube-system").Get(ctx, "metrics-server", metav1.GetOptions{})
	metricsHealth := multicluster.ComponentHealth{
		Name: "metrics-server",
		Type: "metrics",
	}
	if err != nil {
		metricsHealth.Status = "NotInstalled"
		metricsHealth.Message = "Metrics server not found"
	} else {
		metricsHealth.Status = "Healthy"
	}
	*components = append(*components, metricsHealth)

	// Check ingress controller
	namespaces := []string{"ingress-nginx", "nginx-ingress", "kube-system"}
	ingressHealth := multicluster.ComponentHealth{
		Name:   "ingress-controller",
		Type:   "ingress",
		Status: "NotInstalled",
	}
	
	for _, ns := range namespaces {
		deps, err := conn.Client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		for _, dep := range deps.Items {
			if strings.Contains(dep.Name, "ingress") || strings.Contains(dep.Name, "nginx") {
				if dep.Status.ReadyReplicas > 0 {
					ingressHealth.Status = "Healthy"
				} else {
					ingressHealth.Status = "Unhealthy"
					ingressHealth.Message = "No ready replicas"
				}
				break
			}
		}
		if ingressHealth.Status != "NotInstalled" {
			break
		}
	}
	*components = append(*components, ingressHealth)
}

// checkNodes checks the health of cluster nodes
func (h *HealthChecker) checkNodes(ctx context.Context, conn *multicluster.ClusterConnection) multicluster.NodeHealthSummary {
	summary := multicluster.NodeHealthSummary{
		UnhealthyNodes: []string{},
	}

	nodes, err := conn.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		h.log.Error(err, "Failed to list nodes")
		return summary
	}

	summary.Total = int32(len(nodes.Items))

	for _, node := range nodes.Items {
		nodeReady := false
		var readyCondition *corev1.NodeCondition

		// Find Ready condition
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				readyCondition = &condition
				if condition.Status == corev1.ConditionTrue {
					nodeReady = true
				}
				break
			}
		}

		if nodeReady {
			summary.Ready++
		} else if readyCondition != nil && readyCondition.Status == corev1.ConditionFalse {
			summary.NotReady++
			summary.UnhealthyNodes = append(summary.UnhealthyNodes, node.Name)
		} else {
			summary.Unknown++
			summary.UnhealthyNodes = append(summary.UnhealthyNodes, node.Name)
		}

		// Check for other critical conditions
		for _, condition := range node.Status.Conditions {
			if condition.Type != corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				switch condition.Type {
				case corev1.NodeMemoryPressure, corev1.NodeDiskPressure, corev1.NodePIDPressure:
					if !contains(summary.UnhealthyNodes, node.Name) {
						summary.UnhealthyNodes = append(summary.UnhealthyNodes, node.Name)
					}
				}
			}
		}
	}

	return summary
}

// checkNetworkMesh checks network mesh connectivity
func (h *HealthChecker) checkNetworkMesh(ctx context.Context, conn *multicluster.ClusterConnection, meshType string) bool {
	switch meshType {
	case "istio":
		// Check Istio control plane
		_, err := conn.Client.AppsV1().Deployments("istio-system").Get(ctx, "istiod", metav1.GetOptions{})
		return err == nil
	case "linkerd":
		// Check Linkerd control plane
		_, err := conn.Client.AppsV1().Deployments("linkerd").Get(ctx, "linkerd-controller", metav1.GetOptions{})
		return err == nil
	default:
		return false
	}
}

// determineOverallStatus determines the overall cluster status
func (h *HealthChecker) determineOverallStatus(components []multicluster.ComponentHealth, nodes multicluster.NodeHealthSummary, connectivity multicluster.ConnectivityStatus) multicluster.ClusterStatus {
	// If API server is not reachable, cluster is offline
	if !connectivity.APIServer {
		return multicluster.ClusterStatusOffline
	}

	// Check for critical component failures
	criticalComponents := []string{"api-server", "etcd", "controller-manager", "scheduler"}
	for _, comp := range components {
		for _, critical := range criticalComponents {
			if comp.Type == critical && comp.Status == "Unhealthy" {
				return multicluster.ClusterStatusNotReady
			}
		}
	}

	// Check node health
	if nodes.Total > 0 {
		readyPercentage := float64(nodes.Ready) / float64(nodes.Total)
		if readyPercentage < 0.5 {
			return multicluster.ClusterStatusNotReady
		}
	}

	// Check for high latency
	if connectivity.Latency > DefaultLatencyThreshold.Milliseconds() {
		return multicluster.ClusterStatusNotReady
	}

	// If we get here, cluster is ready
	return multicluster.ClusterStatusReady
}

// createUnhealthyStatus creates an unhealthy status for a cluster
func (h *HealthChecker) createUnhealthyStatus(clusterName string, reason string, err error) *multicluster.ClusterHealth {
	return &multicluster.ClusterHealth{
		ClusterName: clusterName,
		Status:      multicluster.ClusterStatusUnknown,
		Components: []multicluster.ComponentHealth{
			{
				Name:    "health-check",
				Type:    "system",
				Status:  "Failed",
				Message: fmt.Sprintf("%s: %v", reason, err),
			},
		},
		Connectivity: multicluster.ConnectivityStatus{
			APIServer: false,
		},
		LastCheck: time.Now(),
		NextCheck: time.Now().Add(h.interval),
	}
}

// updateHealthCache updates the cached health status
func (h *HealthChecker) updateHealthCache(clusterName string, health *multicluster.ClusterHealth) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.lastHealth[clusterName] = health
}

// notifyWatchers notifies all watchers of a health change
func (h *HealthChecker) notifyWatchers(clusterName string, health *multicluster.ClusterHealth) {
	h.mu.RLock()
	ch, exists := h.watchers[clusterName]
	h.mu.RUnlock()

	if exists {
		select {
		case ch <- health:
		default:
			h.log.V(2).Info("Health watcher channel full, dropping update", "cluster", clusterName)
		}
	}
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Stop stops the health checker
func (h *HealthChecker) Stop() {
	close(h.stopCh)
}
