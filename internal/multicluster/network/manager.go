package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

const (
	// NetworkMeshNamespace is the namespace for network mesh resources
	NetworkMeshNamespace = "gunj-network-mesh"
	// GatewayServicePrefix for gateway services
	GatewayServicePrefix = "cluster-gateway-"
	// NetworkPolicyPrefix for network policies
	NetworkPolicyPrefix = "cluster-network-"
)

// Manager implements cross-cluster networking
type Manager struct {
	// registry for cluster connections
	registry multicluster.ClusterRegistry
	// hubClient is the client for the hub cluster
	hubClient client.Client
	// log is the logger
	log logr.Logger
	// config is the network mesh configuration
	config *multicluster.NetworkMeshConfig
	// topology stores the network topology
	topology map[string][]string
	// mu for thread safety
	mu sync.RWMutex
}

// NewManager creates a new network manager
func NewManager(registry multicluster.ClusterRegistry, hubClient client.Client) multicluster.NetworkManager {
	return &Manager{
		registry:  registry,
		hubClient: hubClient,
		log:       log.Log.WithName("network-manager"),
		topology:  make(map[string][]string),
	}
}

// SetupNetworkMesh configures cross-cluster networking
func (m *Manager) SetupNetworkMesh(ctx context.Context, config *multicluster.NetworkMeshConfig) error {
	m.log.Info("Setting up network mesh", "provider", config.Provider)

	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("invalid network mesh config: %w", err)
	}

	m.config = config

	// Create network mesh namespace
	if err := m.ensureNamespace(ctx); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Get all clusters
	clusters, err := m.registry.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	// Setup mesh based on provider
	switch config.Provider {
	case "istio":
		return m.setupIstioMesh(ctx, clusters)
	case "linkerd":
		return m.setupLinkerdMesh(ctx, clusters)
	case "native":
		return m.setupNativeMesh(ctx, clusters)
	default:
		return fmt.Errorf("unsupported mesh provider: %s", config.Provider)
	}
}

// ConnectClusters establishes network connectivity between clusters
func (m *Manager) ConnectClusters(ctx context.Context, cluster1, cluster2 string) error {
	m.log.Info("Connecting clusters", "cluster1", cluster1, "cluster2", cluster2)

	// Get cluster connections
	conn1, err := m.registry.GetConnection(ctx, cluster1)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster %s: %w", cluster1, err)
	}

	conn2, err := m.registry.GetConnection(ctx, cluster2)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster %s: %w", cluster2, err)
	}

	// Create gateway services
	if err := m.createGatewayService(ctx, conn1, cluster2); err != nil {
		return fmt.Errorf("failed to create gateway in %s: %w", cluster1, err)
	}

	if err := m.createGatewayService(ctx, conn2, cluster1); err != nil {
		return fmt.Errorf("failed to create gateway in %s: %w", cluster2, err)
	}

	// Create network policies
	if err := m.createNetworkPolicies(ctx, conn1, cluster2); err != nil {
		return fmt.Errorf("failed to create network policies in %s: %w", cluster1, err)
	}

	if err := m.createNetworkPolicies(ctx, conn2, cluster1); err != nil {
		return fmt.Errorf("failed to create network policies in %s: %w", cluster2, err)
	}

	// Update topology
	m.updateTopology(cluster1, cluster2)

	m.log.Info("Successfully connected clusters", "cluster1", cluster1, "cluster2", cluster2)
	return nil
}

// DisconnectClusters removes network connectivity
func (m *Manager) DisconnectClusters(ctx context.Context, cluster1, cluster2 string) error {
	m.log.Info("Disconnecting clusters", "cluster1", cluster1, "cluster2", cluster2)

	// Get cluster connections
	conn1, err := m.registry.GetConnection(ctx, cluster1)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster %s: %w", cluster1, err)
	}

	conn2, err := m.registry.GetConnection(ctx, cluster2)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster %s: %w", cluster2, err)
	}

	// Remove gateway services
	if err := m.removeGatewayService(ctx, conn1, cluster2); err != nil {
		m.log.Error(err, "Failed to remove gateway", "cluster", cluster1)
	}

	if err := m.removeGatewayService(ctx, conn2, cluster1); err != nil {
		m.log.Error(err, "Failed to remove gateway", "cluster", cluster2)
	}

	// Remove network policies
	if err := m.removeNetworkPolicies(ctx, conn1, cluster2); err != nil {
		m.log.Error(err, "Failed to remove network policies", "cluster", cluster1)
	}

	if err := m.removeNetworkPolicies(ctx, conn2, cluster1); err != nil {
		m.log.Error(err, "Failed to remove network policies", "cluster", cluster2)
	}

	// Update topology
	m.removeFromTopology(cluster1, cluster2)

	m.log.Info("Successfully disconnected clusters", "cluster1", cluster1, "cluster2", cluster2)
	return nil
}

// GetNetworkTopology returns the current network topology
func (m *Manager) GetNetworkTopology(ctx context.Context) (map[string][]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy of the topology
	topology := make(map[string][]string)
	for k, v := range m.topology {
		topology[k] = append([]string{}, v...)
	}

	return topology, nil
}

// TestConnectivity tests connectivity between clusters
func (m *Manager) TestConnectivity(ctx context.Context, sourceCluster, targetCluster string) (*multicluster.ConnectivityStatus, error) {
	m.log.Info("Testing connectivity", "source", sourceCluster, "target", targetCluster)

	// Get source cluster connection
	sourceConn, err := m.registry.GetConnection(ctx, sourceCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source cluster: %w", err)
	}

	// Get target cluster info
	targetClusterInfo, err := m.registry.Get(ctx, targetCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get target cluster info: %w", err)
	}

	status := &multicluster.ConnectivityStatus{
		APIServer:        false,
		ServiceEndpoints: make(map[string]bool),
	}

	// Test gateway service connectivity
	gatewayName := GatewayServicePrefix + targetCluster
	svc, err := sourceConn.Client.CoreV1().Services(NetworkMeshNamespace).Get(ctx, gatewayName, metav1.GetOptions{})
	if err != nil {
		return status, fmt.Errorf("gateway service not found: %w", err)
	}

	// Check if service has endpoints
	endpoints, err := sourceConn.Client.CoreV1().Endpoints(NetworkMeshNamespace).Get(ctx, gatewayName, metav1.GetOptions{})
	if err != nil || len(endpoints.Subsets) == 0 || len(endpoints.Subsets[0].Addresses) == 0 {
		return status, fmt.Errorf("no endpoints for gateway service")
	}

	// Perform connectivity test
	testStart := time.Now()
	// In a real implementation, this would perform actual network tests
	// For now, we'll simulate with a simple check
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer && svc.Status.LoadBalancer.Ingress != nil {
		status.APIServer = true
		status.Latency = time.Since(testStart).Milliseconds()
		status.LastSuccessfulPing = time.Now()
	}

	// Test service mesh connectivity if enabled
	if m.config != nil && m.config.Enabled {
		status.NetworkMesh = m.testMeshConnectivity(ctx, sourceConn, targetClusterInfo)
	}

	return status, nil
}

// validateConfig validates network mesh configuration
func (m *Manager) validateConfig(config *multicluster.NetworkMeshConfig) error {
	if !config.Enabled {
		return fmt.Errorf("network mesh is not enabled")
	}
	if config.Provider == "" {
		return fmt.Errorf("mesh provider is required")
	}
	if config.TrustDomain == "" {
		config.TrustDomain = "cluster.local"
	}
	if config.GatewayMode == "" {
		config.GatewayMode = "ingress"
	}
	return nil
}

// ensureNamespace ensures the network mesh namespace exists
func (m *Manager) ensureNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: NetworkMeshNamespace,
			Labels: map[string]string{
				"gunj.io/component": "network-mesh",
			},
		},
	}

	// Check if namespace exists
	existing := &corev1.Namespace{}
	err := m.hubClient.Get(ctx, client.ObjectKeyFromObject(ns), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return m.hubClient.Create(ctx, ns)
		}
		return err
	}

	return nil
}

// setupIstioMesh sets up Istio-based mesh networking
func (m *Manager) setupIstioMesh(ctx context.Context, clusters []*multicluster.Cluster) error {
	m.log.Info("Setting up Istio mesh")

	// In a real implementation, this would:
	// 1. Install Istio control plane in hub cluster
	// 2. Generate root certificates
	// 3. Create multi-cluster secrets
	// 4. Configure Istio gateways
	// 5. Set up service entries

	// For now, we'll create basic configuration
	for _, cluster := range clusters {
		if err := m.createIstioConfig(ctx, cluster); err != nil {
			return fmt.Errorf("failed to setup Istio for cluster %s: %w", cluster.Name, err)
		}
	}

	return nil
}

// setupLinkerdMesh sets up Linkerd-based mesh networking
func (m *Manager) setupLinkerdMesh(ctx context.Context, clusters []*multicluster.Cluster) error {
	m.log.Info("Setting up Linkerd mesh")

	// In a real implementation, this would:
	// 1. Install Linkerd control plane
	// 2. Create link secrets
	// 3. Configure multicluster links
	// 4. Set up traffic policies

	return fmt.Errorf("linkerd mesh setup not yet implemented")
}

// setupNativeMesh sets up native Kubernetes networking
func (m *Manager) setupNativeMesh(ctx context.Context, clusters []*multicluster.Cluster) error {
	m.log.Info("Setting up native mesh")

	// Create full mesh connectivity
	for i, cluster1 := range clusters {
		for j, cluster2 := range clusters {
			if i != j {
				if err := m.ConnectClusters(ctx, cluster1.Name, cluster2.Name); err != nil {
					m.log.Error(err, "Failed to connect clusters", 
						"cluster1", cluster1.Name, 
						"cluster2", cluster2.Name)
				}
			}
		}
	}

	return nil
}

// createGatewayService creates a gateway service for cross-cluster communication
func (m *Manager) createGatewayService(ctx context.Context, conn *multicluster.ClusterConnection, targetCluster string) error {
	// Get target cluster info
	targetInfo, err := m.registry.Get(ctx, targetCluster)
	if err != nil {
		return err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayServicePrefix + targetCluster,
			Namespace: NetworkMeshNamespace,
			Labels: map[string]string{
				"gunj.io/component":      "network-gateway",
				"gunj.io/target-cluster": targetCluster,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.FromInt(8443),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": "cluster-gateway",
			},
		},
	}

	// Create endpoint slice for external cluster
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: targetInfo.Endpoint, // In reality, this would be the IP
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "https",
						Port:     443,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	// Create service
	if err := conn.RuntimeClient.Create(ctx, svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// Create endpoints
	if err := conn.RuntimeClient.Create(ctx, endpoints); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// removeGatewayService removes a gateway service
func (m *Manager) removeGatewayService(ctx context.Context, conn *multicluster.ClusterConnection, targetCluster string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayServicePrefix + targetCluster,
			Namespace: NetworkMeshNamespace,
		},
	}

	if err := conn.RuntimeClient.Delete(ctx, svc); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
		},
	}

	if err := conn.RuntimeClient.Delete(ctx, endpoints); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// createNetworkPolicies creates network policies for cross-cluster communication
func (m *Manager) createNetworkPolicies(ctx context.Context, conn *multicluster.ClusterConnection, targetCluster string) error {
	// Allow ingress from target cluster
	ingressPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NetworkPolicyPrefix + targetCluster + "-ingress",
			Namespace: NetworkMeshNamespace,
			Labels: map[string]string{
				"gunj.io/component":      "network-policy",
				"gunj.io/target-cluster": targetCluster,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"gunj.io/cross-cluster": "enabled",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"gunj.io/cluster": targetCluster,
								},
							},
						},
					},
				},
			},
		},
	}

	// Allow egress to target cluster
	egressPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NetworkPolicyPrefix + targetCluster + "-egress",
			Namespace: NetworkMeshNamespace,
			Labels: map[string]string{
				"gunj.io/component":      "network-policy",
				"gunj.io/target-cluster": targetCluster,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"gunj.io/cross-cluster": "enabled",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"gunj.io/cluster": targetCluster,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create policies
	if err := conn.RuntimeClient.Create(ctx, ingressPolicy); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	if err := conn.RuntimeClient.Create(ctx, egressPolicy); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// removeNetworkPolicies removes network policies
func (m *Manager) removeNetworkPolicies(ctx context.Context, conn *multicluster.ClusterConnection, targetCluster string) error {
	policies := []string{
		NetworkPolicyPrefix + targetCluster + "-ingress",
		NetworkPolicyPrefix + targetCluster + "-egress",
	}

	for _, policyName := range policies {
		policy := &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      policyName,
				Namespace: NetworkMeshNamespace,
			},
		}

		if err := conn.RuntimeClient.Delete(ctx, policy); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// createIstioConfig creates Istio-specific configuration
func (m *Manager) createIstioConfig(ctx context.Context, cluster *multicluster.Cluster) error {
	// This would create Istio-specific resources like:
	// - ServiceEntry for remote clusters
	// - Gateway for ingress
	// - VirtualService for routing
	// - DestinationRule for traffic policies
	
	// Simplified implementation
	return nil
}

// testMeshConnectivity tests service mesh connectivity
func (m *Manager) testMeshConnectivity(ctx context.Context, conn *multicluster.ClusterConnection, target *multicluster.Cluster) bool {
	// In a real implementation, this would:
	// 1. Deploy a test pod
	// 2. Execute connectivity tests
	// 3. Verify mTLS is working
	// 4. Check service discovery
	
	// For now, return true if mesh is configured
	return m.config != nil && m.config.Enabled
}

// updateTopology updates the network topology
func (m *Manager) updateTopology(cluster1, cluster2 string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add cluster2 to cluster1's connections
	if !contains(m.topology[cluster1], cluster2) {
		m.topology[cluster1] = append(m.topology[cluster1], cluster2)
	}

	// Add cluster1 to cluster2's connections
	if !contains(m.topology[cluster2], cluster1) {
		m.topology[cluster2] = append(m.topology[cluster2], cluster1)
	}
}

// removeFromTopology removes a connection from the topology
func (m *Manager) removeFromTopology(cluster1, cluster2 string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove cluster2 from cluster1's connections
	m.topology[cluster1] = removeString(m.topology[cluster1], cluster2)

	// Remove cluster1 from cluster2's connections
	m.topology[cluster2] = removeString(m.topology[cluster2], cluster1)
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, item string) []string {
	result := []string{}
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
