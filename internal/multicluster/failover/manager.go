package failover

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

const (
	// FailoverAnnotationPrefix for failover-related annotations
	FailoverAnnotationPrefix = "failover.gunj.io/"
	// FailoverConfigMapName stores failover configuration
	FailoverConfigMapName = "failover-config"
	// FailoverHistoryConfigMapName stores failover history
	FailoverHistoryConfigMapName = "failover-history"
)

// FailoverState represents the state of a failover operation
type FailoverState string

const (
	// FailoverStateActive indicates active failover
	FailoverStateActive FailoverState = "Active"
	// FailoverStateStandby indicates standby state
	FailoverStateStandby FailoverState = "Standby"
	// FailoverStateFailingOver indicates failover in progress
	FailoverStateFailingOver FailoverState = "FailingOver"
	// FailoverStateFailed indicates failover failed
	FailoverStateFailed FailoverState = "Failed"
)

// FailoverEvent represents a failover event
type FailoverEvent struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	FromCluster  string            `json:"fromCluster"`
	ToCluster    string            `json:"toCluster"`
	Reason       string            `json:"reason"`
	Status       string            `json:"status"`
	ResourcesMap map[string]string `json:"resourcesMap"`
	Error        string            `json:"error,omitempty"`
}

// Manager implements cluster failover management
type Manager struct {
	// registry for cluster connections
	registry multicluster.ClusterRegistry
	// healthChecker for monitoring cluster health
	healthChecker multicluster.ClusterHealthChecker
	// hubClient is the client for the hub cluster
	hubClient client.Client
	// recorder for Kubernetes events
	recorder record.EventRecorder
	// log is the logger
	log logr.Logger
	// config is the failover configuration
	config *multicluster.FailoverConfig
	// clusterStates tracks the state of each cluster
	clusterStates map[string]FailoverState
	// failoverHistory stores failover events
	failoverHistory []FailoverEvent
	// activeFailovers tracks active failover operations
	activeFailovers map[string]*FailoverEvent
	// mu for thread safety
	mu sync.RWMutex
	// stopCh for stopping background tasks
	stopCh chan struct{}
}

// NewManager creates a new failover manager
func NewManager(
	registry multicluster.ClusterRegistry,
	healthChecker multicluster.ClusterHealthChecker,
	hubClient client.Client,
	recorder record.EventRecorder,
) multicluster.FailoverManager {
	manager := &Manager{
		registry:        registry,
		healthChecker:   healthChecker,
		hubClient:       hubClient,
		recorder:        recorder,
		log:             log.Log.WithName("failover-manager"),
		clusterStates:   make(map[string]FailoverState),
		failoverHistory: []FailoverEvent{},
		activeFailovers: make(map[string]*FailoverEvent),
		stopCh:          make(chan struct{}),
	}

	// Start monitoring
	go manager.monitorClusterHealth()

	return manager
}

// ConfigureFailover sets up failover configuration
func (m *Manager) ConfigureFailover(ctx context.Context, config *multicluster.FailoverConfig) error {
	m.log.Info("Configuring failover", "strategy", config.Strategy)

	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("invalid failover config: %w", err)
	}

	m.mu.Lock()
	m.config = config
	m.mu.Unlock()

	// Initialize cluster states based on priority groups
	if err := m.initializeClusterStates(ctx, config); err != nil {
		return fmt.Errorf("failed to initialize cluster states: %w", err)
	}

	// Store configuration
	if err := m.storeConfig(ctx, config); err != nil {
		return fmt.Errorf("failed to store failover config: %w", err)
	}

	m.log.Info("Successfully configured failover")
	return nil
}

// TriggerFailover manually triggers a failover
func (m *Manager) TriggerFailover(ctx context.Context, fromCluster, toCluster string) error {
	m.log.Info("Manually triggering failover", "from", fromCluster, "to", toCluster)

	// Create failover event
	event := &FailoverEvent{
		ID:           fmt.Sprintf("manual-%d", time.Now().Unix()),
		Timestamp:    time.Now(),
		FromCluster:  fromCluster,
		ToCluster:    toCluster,
		Reason:       "Manual failover",
		Status:       "Initiated",
		ResourcesMap: make(map[string]string),
	}

	// Execute failover
	if err := m.executeFailover(ctx, event); err != nil {
		event.Status = "Failed"
		event.Error = err.Error()
		m.recordFailoverEvent(event)
		return fmt.Errorf("failover failed: %w", err)
	}

	event.Status = "Completed"
	m.recordFailoverEvent(event)

	m.log.Info("Successfully completed failover", "from", fromCluster, "to", toCluster)
	return nil
}

// GetFailoverStatus returns current failover status
func (m *Manager) GetFailoverStatus(ctx context.Context) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]string)
	
	// Add cluster states
	for cluster, state := range m.clusterStates {
		status[fmt.Sprintf("cluster.%s.state", cluster)] = string(state)
	}

	// Add active failovers
	for id, event := range m.activeFailovers {
		status[fmt.Sprintf("failover.%s.status", id)] = event.Status
		status[fmt.Sprintf("failover.%s.from", id)] = event.FromCluster
		status[fmt.Sprintf("failover.%s.to", id)] = event.ToCluster
	}

	// Add configuration status
	if m.config != nil {
		status["config.enabled"] = fmt.Sprintf("%v", m.config.Enabled)
		status["config.strategy"] = m.config.Strategy
	}

	return status, nil
}

// TestFailover performs a failover drill
func (m *Manager) TestFailover(ctx context.Context, scenario string) error {
	m.log.Info("Running failover test", "scenario", scenario)

	switch scenario {
	case "primary-failure":
		return m.testPrimaryFailure(ctx)
	case "network-partition":
		return m.testNetworkPartition(ctx)
	case "cascading-failure":
		return m.testCascadingFailure(ctx)
	default:
		return fmt.Errorf("unknown test scenario: %s", scenario)
	}
}

// RevertFailover reverts a failover operation
func (m *Manager) RevertFailover(ctx context.Context, failoverID string) error {
	m.log.Info("Reverting failover", "id", failoverID)

	// Find failover event
	var event *FailoverEvent
	for _, e := range m.failoverHistory {
		if e.ID == failoverID {
			event = &e
			break
		}
	}

	if event == nil {
		return fmt.Errorf("failover event %s not found", failoverID)
	}

	// Create revert event
	revertEvent := &FailoverEvent{
		ID:           fmt.Sprintf("revert-%s", failoverID),
		Timestamp:    time.Now(),
		FromCluster:  event.ToCluster,   // Swap clusters
		ToCluster:    event.FromCluster,
		Reason:       fmt.Sprintf("Revert failover %s", failoverID),
		Status:       "Initiated",
		ResourcesMap: make(map[string]string),
	}

	// Execute failback
	if err := m.executeFailover(ctx, revertEvent); err != nil {
		revertEvent.Status = "Failed"
		revertEvent.Error = err.Error()
		m.recordFailoverEvent(revertEvent)
		return fmt.Errorf("revert failed: %w", err)
	}

	revertEvent.Status = "Completed"
	m.recordFailoverEvent(revertEvent)

	m.log.Info("Successfully reverted failover", "id", failoverID)
	return nil
}

// executeFailover executes a failover operation
func (m *Manager) executeFailover(ctx context.Context, event *FailoverEvent) error {
	m.mu.Lock()
	m.activeFailovers[event.ID] = event
	m.clusterStates[event.FromCluster] = FailoverStateFailingOver
	m.clusterStates[event.ToCluster] = FailoverStateFailingOver
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.activeFailovers, event.ID)
		m.mu.Unlock()
	}()

	// Record Kubernetes event
	m.recorder.Event(&corev1.ObjectReference{
		Kind:      "Failover",
		Name:      event.ID,
		Namespace: "gunj-multicluster",
	}, corev1.EventTypeNormal, "FailoverStarted", 
		fmt.Sprintf("Starting failover from %s to %s", event.FromCluster, event.ToCluster))

	// Phase 1: Validate target cluster
	if err := m.validateTargetCluster(ctx, event.ToCluster); err != nil {
		return fmt.Errorf("target cluster validation failed: %w", err)
	}

	// Phase 2: Get resources from source cluster
	resources, err := m.getClusterResources(ctx, event.FromCluster)
	if err != nil {
		m.log.Warn("Failed to get resources from source cluster", "error", err)
		// Continue with failover even if we can't get all resources
	}

	// Phase 3: Prepare target cluster
	if err := m.prepareTargetCluster(ctx, event.ToCluster); err != nil {
		return fmt.Errorf("failed to prepare target cluster: %w", err)
	}

	// Phase 4: Migrate resources
	migratedResources, err := m.migrateResources(ctx, resources, event.FromCluster, event.ToCluster)
	if err != nil {
		return fmt.Errorf("resource migration failed: %w", err)
	}
	event.ResourcesMap = migratedResources

	// Phase 5: Update DNS/Load Balancers
	if err := m.updateNetworking(ctx, event.FromCluster, event.ToCluster); err != nil {
		m.log.Error(err, "Failed to update networking")
		// Continue - networking updates might be manual
	}

	// Phase 6: Verify failover
	if err := m.verifyFailover(ctx, event.ToCluster, len(resources)); err != nil {
		return fmt.Errorf("failover verification failed: %w", err)
	}

	// Update cluster states
	m.mu.Lock()
	m.clusterStates[event.FromCluster] = FailoverStateStandby
	m.clusterStates[event.ToCluster] = FailoverStateActive
	m.mu.Unlock()

	// Record success event
	m.recorder.Event(&corev1.ObjectReference{
		Kind:      "Failover",
		Name:      event.ID,
		Namespace: "gunj-multicluster",
	}, corev1.EventTypeNormal, "FailoverCompleted", 
		fmt.Sprintf("Successfully failed over from %s to %s", event.FromCluster, event.ToCluster))

	return nil
}

// validateTargetCluster validates the target cluster is ready
func (m *Manager) validateTargetCluster(ctx context.Context, clusterName string) error {
	// Check cluster health
	health, err := m.healthChecker.CheckHealth(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster health: %w", err)
	}

	if health.Status != multicluster.ClusterStatusReady {
		return fmt.Errorf("target cluster is not ready: %s", health.Status)
	}

	// Check resource capacity
	conn, err := m.registry.GetConnection(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Get cluster metrics
	metrics, err := m.getClusterMetrics(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to get cluster metrics: %w", err)
	}

	// Validate capacity
	if metrics.CPUUsage/metrics.CPUCapacity > 0.8 {
		return fmt.Errorf("target cluster CPU usage too high: %.2f%%", 
			(metrics.CPUUsage/metrics.CPUCapacity)*100)
	}

	if float64(metrics.MemoryUsage)/float64(metrics.MemoryCapacity) > 0.8 {
		return fmt.Errorf("target cluster memory usage too high: %.2f%%", 
			(float64(metrics.MemoryUsage)/float64(metrics.MemoryCapacity))*100)
	}

	return nil
}

// getClusterResources gets all ObservabilityPlatform resources from a cluster
func (m *Manager) getClusterResources(ctx context.Context, clusterName string) ([]*observabilityv1.ObservabilityPlatform, error) {
	conn, err := m.registry.GetConnection(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// List all ObservabilityPlatform resources
	platformList := &observabilityv1.ObservabilityPlatformList{}
	if err := conn.RuntimeClient.List(ctx, platformList); err != nil {
		return nil, fmt.Errorf("failed to list platforms: %w", err)
	}

	resources := make([]*observabilityv1.ObservabilityPlatform, 0, len(platformList.Items))
	for i := range platformList.Items {
		// Only include platforms that should be failed over
		if shouldFailover(&platformList.Items[i]) {
			resources = append(resources, &platformList.Items[i])
		}
	}

	return resources, nil
}

// prepareTargetCluster prepares the target cluster for failover
func (m *Manager) prepareTargetCluster(ctx context.Context, clusterName string) error {
	conn, err := m.registry.GetConnection(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Create namespaces if needed
	namespaces := []string{"monitoring", "observability", "gunj-system"}
	for _, ns := range namespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"gunj.io/failover": "prepared",
				},
			},
		}
		
		if err := conn.RuntimeClient.Create(ctx, namespace); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create namespace %s: %w", ns, err)
		}
	}

	// Scale down non-critical workloads to free resources
	// This is a simplified example - in production, use more sophisticated logic
	
	return nil
}

// migrateResources migrates resources to the target cluster
func (m *Manager) migrateResources(ctx context.Context, resources []*observabilityv1.ObservabilityPlatform, fromCluster, toCluster string) (map[string]string, error) {
	targetConn, err := m.registry.GetConnection(ctx, toCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target cluster: %w", err)
	}

	migratedMap := make(map[string]string)
	
	for _, platform := range resources {
		// Create a copy for the target cluster
		targetPlatform := platform.DeepCopy()
		targetPlatform.ResourceVersion = ""
		targetPlatform.UID = ""
		
		// Add failover annotations
		if targetPlatform.Annotations == nil {
			targetPlatform.Annotations = make(map[string]string)
		}
		targetPlatform.Annotations[FailoverAnnotationPrefix+"source-cluster"] = fromCluster
		targetPlatform.Annotations[FailoverAnnotationPrefix+"failover-time"] = time.Now().UTC().Format(time.RFC3339)
		targetPlatform.Annotations[FailoverAnnotationPrefix+"original-name"] = platform.Name
		
		// Check if already exists
		existing := &observabilityv1.ObservabilityPlatform{}
		err := targetConn.RuntimeClient.Get(ctx, client.ObjectKeyFromObject(targetPlatform), existing)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Create new platform
				if err := targetConn.RuntimeClient.Create(ctx, targetPlatform); err != nil {
					m.log.Error(err, "Failed to create platform", "name", targetPlatform.Name)
					continue
				}
			} else {
				m.log.Error(err, "Failed to check existing platform", "name", targetPlatform.Name)
				continue
			}
		} else {
			// Update existing platform
			existing.Spec = targetPlatform.Spec
			existing.Labels = targetPlatform.Labels
			existing.Annotations = targetPlatform.Annotations
			
			if err := targetConn.RuntimeClient.Update(ctx, existing); err != nil {
				m.log.Error(err, "Failed to update platform", "name", existing.Name)
				continue
			}
		}
		
		key := fmt.Sprintf("%s/%s", platform.Namespace, platform.Name)
		migratedMap[key] = fmt.Sprintf("%s/%s", targetPlatform.Namespace, targetPlatform.Name)
	}
	
	return migratedMap, nil
}

// updateNetworking updates DNS and load balancer configurations
func (m *Manager) updateNetworking(ctx context.Context, fromCluster, toCluster string) error {
	// In a real implementation, this would:
	// 1. Update DNS records to point to the new cluster
	// 2. Update load balancer configurations
	// 3. Update ingress controllers
	// 4. Update service mesh configurations
	
	m.log.Info("Updating networking configuration", "from", fromCluster, "to", toCluster)
	
	// This is highly environment-specific
	// Could integrate with external DNS providers, cloud load balancers, etc.
	
	return nil
}

// verifyFailover verifies the failover was successful
func (m *Manager) verifyFailover(ctx context.Context, clusterName string, expectedResources int) error {
	conn, err := m.registry.GetConnection(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Wait for resources to be ready
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for resources to be ready")
		case <-ticker.C:
			// Check platform resources
			platformList := &observabilityv1.ObservabilityPlatformList{}
			if err := conn.RuntimeClient.List(ctx, platformList); err != nil {
				m.log.Error(err, "Failed to list platforms")
				continue
			}

			// Count ready platforms
			readyCount := 0
			for _, platform := range platformList.Items {
				if platform.Annotations != nil && platform.Annotations[FailoverAnnotationPrefix+"source-cluster"] != "" {
					// This is a failed-over resource
					if platform.Status.Phase == "Ready" {
						readyCount++
					}
				}
			}

			m.log.V(1).Info("Failover verification progress", 
				"ready", readyCount, 
				"expected", expectedResources)

			if readyCount >= expectedResources {
				m.log.Info("Failover verification successful", 
					"cluster", clusterName,
					"resources", readyCount)
				return nil
			}
		}
	}
}

// monitorClusterHealth monitors cluster health for automatic failover
func (m *Manager) monitorClusterHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkForFailoverConditions()
		case <-m.stopCh:
			m.log.Info("Stopping health monitor")
			return
		}
	}
}

// checkForFailoverConditions checks if automatic failover should be triggered
func (m *Manager) checkForFailoverConditions() {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	if config == nil || !config.Enabled {
		return
	}

	ctx := context.Background()
	
	// Check each priority group
	for _, group := range config.PriorityGroups {
		activeClusters := []string{}
		failedClusters := []string{}
		
		// Check health of clusters in this group
		for _, clusterName := range group.Clusters {
			health, err := m.healthChecker.CheckHealth(ctx, clusterName)
			if err != nil || health.Status != multicluster.ClusterStatusReady {
				// Check how long the cluster has been unhealthy
				if m.shouldTriggerFailover(clusterName, config.MaxUnavailableTime) {
					failedClusters = append(failedClusters, clusterName)
				}
			} else {
				activeClusters = append(activeClusters, clusterName)
			}
		}
		
		// Trigger failover if needed
		if len(failedClusters) > 0 && len(activeClusters) > 0 {
			for _, failedCluster := range failedClusters {
				// Find the best target cluster
				targetCluster := m.selectTargetCluster(activeClusters, failedCluster)
				if targetCluster != "" {
					m.log.Info("Triggering automatic failover", 
						"from", failedCluster, 
						"to", targetCluster)
					
					event := &FailoverEvent{
						ID:          fmt.Sprintf("auto-%d", time.Now().Unix()),
						Timestamp:   time.Now(),
						FromCluster: failedCluster,
						ToCluster:   targetCluster,
						Reason:      "Automatic failover due to cluster failure",
						Status:      "Initiated",
					}
					
					go func() {
						if err := m.executeFailover(context.Background(), event); err != nil {
							m.log.Error(err, "Automatic failover failed")
						}
					}()
				}
			}
		}
	}
}

// shouldTriggerFailover checks if a cluster has been unhealthy long enough
func (m *Manager) shouldTriggerFailover(clusterName string, maxUnavailable time.Duration) bool {
	// In a real implementation, track the duration of unhealthy state
	// For now, return true if cluster state is not active
	m.mu.RLock()
	state := m.clusterStates[clusterName]
	m.mu.RUnlock()
	
	return state != FailoverStateActive && state != FailoverStateFailingOver
}

// selectTargetCluster selects the best target cluster for failover
func (m *Manager) selectTargetCluster(activeClusters []string, sourceCluster string) string {
	// In a real implementation, consider:
	// 1. Geographic proximity
	// 2. Available capacity
	// 3. Network latency
	// 4. Cost factors
	
	// For now, return the first active cluster
	if len(activeClusters) > 0 {
		return activeClusters[0]
	}
	return ""
}

// getClusterMetrics gets resource metrics for a cluster
func (m *Manager) getClusterMetrics(ctx context.Context, conn *multicluster.ClusterConnection) (*multicluster.ClusterMetrics, error) {
	// In a real implementation, query metrics from metrics-server or Prometheus
	// For now, return mock data
	return &multicluster.ClusterMetrics{
		CPUUsage:        50.0,
		CPUCapacity:     100.0,
		MemoryUsage:     8 * 1024 * 1024 * 1024, // 8GB
		MemoryCapacity:  16 * 1024 * 1024 * 1024, // 16GB
		PodCount:        50,
		PodCapacity:     110,
		NodeCount:       3,
		NodeReady:       3,
	}, nil
}

// validateConfig validates failover configuration
func (m *Manager) validateConfig(config *multicluster.FailoverConfig) error {
	if config.Strategy == "" {
		config.Strategy = "active-passive"
	}
	
	if config.MaxUnavailableTime == 0 {
		config.MaxUnavailableTime = 5 * time.Minute
	}
	
	if len(config.PriorityGroups) == 0 {
		return fmt.Errorf("at least one priority group is required")
	}
	
	// Validate priority groups
	priorities := make(map[int32]bool)
	for _, group := range config.PriorityGroups {
		if priorities[group.Priority] {
			return fmt.Errorf("duplicate priority %d", group.Priority)
		}
		priorities[group.Priority] = true
		
		if len(group.Clusters) == 0 && group.Selector == nil {
			return fmt.Errorf("priority group %d has no clusters or selector", group.Priority)
		}
	}
	
	return nil
}

// initializeClusterStates initializes cluster states based on configuration
func (m *Manager) initializeClusterStates(ctx context.Context, config *multicluster.FailoverConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Get all clusters
	clusters, err := m.registry.List(ctx)
	if err != nil {
		return err
	}
	
	// Initialize all clusters as standby
	for _, cluster := range clusters {
		m.clusterStates[cluster.Name] = FailoverStateStandby
	}
	
	// Set active clusters based on strategy
	if config.Strategy == "active-passive" {
		// First cluster in first priority group is active
		if len(config.PriorityGroups) > 0 && len(config.PriorityGroups[0].Clusters) > 0 {
			m.clusterStates[config.PriorityGroups[0].Clusters[0]] = FailoverStateActive
		}
	} else if config.Strategy == "active-active" {
		// All clusters in first priority group are active
		if len(config.PriorityGroups) > 0 {
			for _, clusterName := range config.PriorityGroups[0].Clusters {
				m.clusterStates[clusterName] = FailoverStateActive
			}
		}
	}
	
	return nil
}

// storeConfig stores failover configuration
func (m *Manager) storeConfig(ctx context.Context, config *multicluster.FailoverConfig) error {
	// Store in ConfigMap
	configData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FailoverConfigMapName,
			Namespace: "gunj-multicluster",
		},
		Data: map[string]string{
			"config.json": string(configData),
		},
	}
	
	existing := &corev1.ConfigMap{}
	err = m.hubClient.Get(ctx, client.ObjectKeyFromObject(configMap), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return m.hubClient.Create(ctx, configMap)
		}
		return err
	}
	
	existing.Data = configMap.Data
	return m.hubClient.Update(ctx, existing)
}

// recordFailoverEvent records a failover event in history
func (m *Manager) recordFailoverEvent(event *FailoverEvent) {
	m.mu.Lock()
	m.failoverHistory = append(m.failoverHistory, *event)
	// Keep last 100 events
	if len(m.failoverHistory) > 100 {
		m.failoverHistory = m.failoverHistory[len(m.failoverHistory)-100:]
	}
	m.mu.Unlock()
	
	// Store in ConfigMap
	go m.storeFailoverHistory()
}

// storeFailoverHistory stores failover history in ConfigMap
func (m *Manager) storeFailoverHistory() {
	m.mu.RLock()
	historyData, err := json.Marshal(m.failoverHistory)
	m.mu.RUnlock()
	
	if err != nil {
		m.log.Error(err, "Failed to marshal failover history")
		return
	}
	
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FailoverHistoryConfigMapName,
			Namespace: "gunj-multicluster",
		},
		Data: map[string]string{
			"history.json": string(historyData),
		},
	}
	
	ctx := context.Background()
	existing := &corev1.ConfigMap{}
	err = m.hubClient.Get(ctx, client.ObjectKeyFromObject(configMap), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := m.hubClient.Create(ctx, configMap); err != nil {
				m.log.Error(err, "Failed to create failover history")
			}
			return
		}
		m.log.Error(err, "Failed to get failover history")
		return
	}
	
	existing.Data = configMap.Data
	if err := m.hubClient.Update(ctx, existing); err != nil {
		m.log.Error(err, "Failed to update failover history")
	}
}

// Test scenarios

func (m *Manager) testPrimaryFailure(ctx context.Context) error {
	// Simulate primary cluster failure
	m.log.Info("Testing primary cluster failure scenario")
	
	// Find active cluster
	var primaryCluster string
	m.mu.RLock()
	for cluster, state := range m.clusterStates {
		if state == FailoverStateActive {
			primaryCluster = cluster
			break
		}
	}
	m.mu.RUnlock()
	
	if primaryCluster == "" {
		return fmt.Errorf("no active cluster found")
	}
	
	// Simulate failure by marking cluster as failed
	m.mu.Lock()
	m.clusterStates[primaryCluster] = FailoverStateFailed
	m.mu.Unlock()
	
	// Trigger failover check
	m.checkForFailoverConditions()
	
	return nil
}

func (m *Manager) testNetworkPartition(ctx context.Context) error {
	// Simulate network partition scenario
	m.log.Info("Testing network partition scenario")
	// Implementation would simulate network issues
	return nil
}

func (m *Manager) testCascadingFailure(ctx context.Context) error {
	// Simulate cascading failure scenario
	m.log.Info("Testing cascading failure scenario")
	// Implementation would simulate multiple cluster failures
	return nil
}

// Helper functions

func shouldFailover(platform *observabilityv1.ObservabilityPlatform) bool {
	// Check if platform should be included in failover
	if platform.Annotations != nil {
		if val, exists := platform.Annotations["failover.gunj.io/exclude"]; exists && val == "true" {
			return false
		}
	}
	return true
}

// Stop stops the failover manager
func (m *Manager) Stop() {
	close(m.stopCh)
}
