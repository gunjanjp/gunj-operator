package registry

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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

const (
	// ClusterRegistryNamespace is where cluster credentials are stored
	ClusterRegistryNamespace = "gunj-multicluster"
	// ClusterSecretPrefix for cluster credential secrets
	ClusterSecretPrefix = "cluster-"
	// ClusterConfigMapName stores cluster metadata
	ClusterConfigMapName = "cluster-registry"
)

// Registry implements the ClusterRegistry interface
type Registry struct {
	// hubClient is the client for the hub cluster
	hubClient client.Client
	// log is the logger
	log logr.Logger
	// connections stores active cluster connections
	connections map[string]*multicluster.ClusterConnection
	// mutex for thread safety
	mu sync.RWMutex
	// namespace for storing cluster data
	namespace string
}

// NewRegistry creates a new cluster registry
func NewRegistry(hubClient client.Client, namespace string) multicluster.ClusterRegistry {
	if namespace == "" {
		namespace = ClusterRegistryNamespace
	}
	return &Registry{
		hubClient:   hubClient,
		log:         log.Log.WithName("cluster-registry"),
		connections: make(map[string]*multicluster.ClusterConnection),
		namespace:   namespace,
	}
}

// Register adds a new cluster to the registry
func (r *Registry) Register(ctx context.Context, cluster *multicluster.Cluster, credentials *multicluster.ClusterCredentials) error {
	r.log.Info("Registering cluster", "cluster", cluster.Name)

	// Validate cluster
	if err := r.validateCluster(cluster); err != nil {
		return fmt.Errorf("invalid cluster: %w", err)
	}

	// Store credentials
	if err := r.storeCredentials(ctx, credentials); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Test connection
	conn, err := r.createConnection(ctx, cluster, credentials)
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Update cluster status
	cluster.Status = multicluster.ClusterStatusReady
	cluster.LastSeen = time.Now()
	cluster.RegisteredAt = time.Now()

	// Gather cluster info
	if err := r.gatherClusterInfo(ctx, cluster, conn); err != nil {
		r.log.Error(err, "Failed to gather cluster info", "cluster", cluster.Name)
	}

	// Store cluster metadata
	if err := r.storeClusterMetadata(ctx, cluster); err != nil {
		return fmt.Errorf("failed to store cluster metadata: %w", err)
	}

	// Cache connection
	r.mu.Lock()
	r.connections[cluster.Name] = conn
	r.mu.Unlock()

	r.log.Info("Successfully registered cluster", "cluster", cluster.Name, "role", cluster.Role)
	return nil
}

// Unregister removes a cluster from the registry
func (r *Registry) Unregister(ctx context.Context, clusterName string) error {
	r.log.Info("Unregistering cluster", "cluster", clusterName)

	// Remove connection
	r.mu.Lock()
	delete(r.connections, clusterName)
	r.mu.Unlock()

	// Delete credentials
	secretName := ClusterSecretPrefix + clusterName
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: r.namespace,
		},
	}
	if err := r.hubClient.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	// Remove from metadata
	if err := r.removeClusterMetadata(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to remove cluster metadata: %w", err)
	}

	r.log.Info("Successfully unregistered cluster", "cluster", clusterName)
	return nil
}

// Get retrieves a cluster by name
func (r *Registry) Get(ctx context.Context, clusterName string) (*multicluster.Cluster, error) {
	clusters, err := r.loadClusterMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster metadata: %w", err)
	}

	cluster, exists := clusters[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}

	return cluster, nil
}

// List returns all registered clusters
func (r *Registry) List(ctx context.Context) ([]*multicluster.Cluster, error) {
	clusters, err := r.loadClusterMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster metadata: %w", err)
	}

	result := make([]*multicluster.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		result = append(result, cluster)
	}

	return result, nil
}

// Update updates cluster information
func (r *Registry) Update(ctx context.Context, cluster *multicluster.Cluster) error {
	if err := r.validateCluster(cluster); err != nil {
		return fmt.Errorf("invalid cluster: %w", err)
	}

	// Check if cluster exists
	existing, err := r.Get(ctx, cluster.Name)
	if err != nil {
		return fmt.Errorf("cluster not found: %w", err)
	}

	// Preserve certain fields
	cluster.RegisteredAt = existing.RegisteredAt
	if cluster.LastSeen.IsZero() {
		cluster.LastSeen = existing.LastSeen
	}

	// Update metadata
	if err := r.storeClusterMetadata(ctx, cluster); err != nil {
		return fmt.Errorf("failed to update cluster metadata: %w", err)
	}

	r.log.Info("Updated cluster", "cluster", cluster.Name)
	return nil
}

// GetConnection returns an active connection to a cluster
func (r *Registry) GetConnection(ctx context.Context, clusterName string) (*multicluster.ClusterConnection, error) {
	// Check cache first
	r.mu.RLock()
	conn, exists := r.connections[clusterName]
	r.mu.RUnlock()

	if exists && conn.Connected {
		// Test connection is still valid
		if err := r.testConnection(ctx, conn); err == nil {
			return conn, nil
		}
		r.log.Info("Cached connection is stale, reconnecting", "cluster", clusterName)
	}

	// Get cluster info
	cluster, err := r.Get(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	// Load credentials
	credentials, err := r.loadCredentials(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Create new connection
	conn, err = r.createConnection(ctx, cluster, credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// Update cache
	r.mu.Lock()
	r.connections[clusterName] = conn
	r.mu.Unlock()

	return conn, nil
}

// UpdateStatus updates the cluster status
func (r *Registry) UpdateStatus(ctx context.Context, clusterName string, status multicluster.ClusterStatus) error {
	cluster, err := r.Get(ctx, clusterName)
	if err != nil {
		return err
	}

	cluster.Status = status
	if status == multicluster.ClusterStatusReady {
		cluster.LastSeen = time.Now()
	}

	return r.Update(ctx, cluster)
}

// validateCluster validates cluster configuration
func (r *Registry) validateCluster(cluster *multicluster.Cluster) error {
	if cluster.Name == "" {
		return fmt.Errorf("cluster name is required")
	}
	if cluster.Endpoint == "" {
		return fmt.Errorf("cluster endpoint is required")
	}
	if cluster.Role == "" {
		cluster.Role = multicluster.ClusterRoleSpoke
	}
	return nil
}

// storeCredentials stores cluster credentials in a secret
func (r *Registry) storeCredentials(ctx context.Context, credentials *multicluster.ClusterCredentials) error {
	secretName := ClusterSecretPrefix + credentials.ClusterName

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"gunj.io/cluster-name": credentials.ClusterName,
				"gunj.io/secret-type":  "cluster-credentials",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	// Store credentials based on type
	if len(credentials.KubeConfig) > 0 {
		secret.Data["kubeconfig"] = credentials.KubeConfig
	}
	if credentials.ServiceAccountToken != "" {
		secret.Data["token"] = []byte(credentials.ServiceAccountToken)
	}
	if len(credentials.ClientCertificate) > 0 {
		secret.Data["client-cert"] = credentials.ClientCertificate
	}
	if len(credentials.ClientKey) > 0 {
		secret.Data["client-key"] = credentials.ClientKey
	}
	if len(credentials.CABundle) > 0 {
		secret.Data["ca-bundle"] = credentials.CABundle
	}

	// Create or update secret
	existing := &corev1.Secret{}
	err := r.hubClient.Get(ctx, client.ObjectKeyFromObject(secret), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.hubClient.Create(ctx, secret)
		}
		return err
	}

	existing.Data = secret.Data
	existing.Labels = secret.Labels
	return r.hubClient.Update(ctx, existing)
}

// loadCredentials loads cluster credentials from secret
func (r *Registry) loadCredentials(ctx context.Context, clusterName string) (*multicluster.ClusterCredentials, error) {
	secretName := ClusterSecretPrefix + clusterName

	secret := &corev1.Secret{}
	key := client.ObjectKey{Name: secretName, Namespace: r.namespace}
	if err := r.hubClient.Get(ctx, key, secret); err != nil {
		return nil, err
	}

	credentials := &multicluster.ClusterCredentials{
		ClusterName: clusterName,
	}

	// Load credentials from secret data
	if data, ok := secret.Data["kubeconfig"]; ok {
		credentials.KubeConfig = data
	}
	if data, ok := secret.Data["token"]; ok {
		credentials.ServiceAccountToken = string(data)
	}
	if data, ok := secret.Data["client-cert"]; ok {
		credentials.ClientCertificate = data
	}
	if data, ok := secret.Data["client-key"]; ok {
		credentials.ClientKey = data
	}
	if data, ok := secret.Data["ca-bundle"]; ok {
		credentials.CABundle = data
	}

	return credentials, nil
}

// storeClusterMetadata stores cluster information in a ConfigMap
func (r *Registry) storeClusterMetadata(ctx context.Context, cluster *multicluster.Cluster) error {
	clusters, err := r.loadClusterMetadata(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if clusters == nil {
		clusters = make(map[string]*multicluster.Cluster)
	}

	clusters[cluster.Name] = cluster

	// Marshal clusters to JSON
	data, err := json.Marshal(clusters)
	if err != nil {
		return fmt.Errorf("failed to marshal clusters: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterConfigMapName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"gunj.io/component": "cluster-registry",
			},
		},
		Data: map[string]string{
			"clusters.json": string(data),
		},
	}

	// Create or update ConfigMap
	existing := &corev1.ConfigMap{}
	err = r.hubClient.Get(ctx, client.ObjectKeyFromObject(configMap), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.hubClient.Create(ctx, configMap)
		}
		return err
	}

	existing.Data = configMap.Data
	return r.hubClient.Update(ctx, existing)
}

// loadClusterMetadata loads cluster information from ConfigMap
func (r *Registry) loadClusterMetadata(ctx context.Context) (map[string]*multicluster.Cluster, error) {
	configMap := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: ClusterConfigMapName, Namespace: r.namespace}
	if err := r.hubClient.Get(ctx, key, configMap); err != nil {
		return nil, err
	}

	data, ok := configMap.Data["clusters.json"]
	if !ok {
		return make(map[string]*multicluster.Cluster), nil
	}

	var clusters map[string]*multicluster.Cluster
	if err := json.Unmarshal([]byte(data), &clusters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clusters: %w", err)
	}

	return clusters, nil
}

// removeClusterMetadata removes a cluster from metadata
func (r *Registry) removeClusterMetadata(ctx context.Context, clusterName string) error {
	clusters, err := r.loadClusterMetadata(ctx)
	if err != nil {
		return err
	}

	delete(clusters, clusterName)
	
	// Create a dummy cluster to trigger the update
	dummyCluster := &multicluster.Cluster{Name: ""}
	if err := r.storeClusterMetadata(ctx, dummyCluster); err != nil {
		return err
	}

	// Now store the updated clusters
	for _, cluster := range clusters {
		if err := r.storeClusterMetadata(ctx, cluster); err != nil {
			return err
		}
		break // Only need to store once to update the entire ConfigMap
	}

	return nil
}

// createConnection creates a connection to a cluster
func (r *Registry) createConnection(ctx context.Context, cluster *multicluster.Cluster, credentials *multicluster.ClusterCredentials) (*multicluster.ClusterConnection, error) {
	var config *rest.Config
	var err error

	// Create rest config based on credential type
	if len(credentials.KubeConfig) > 0 {
		config, err = clientcmd.RESTConfigFromKubeConfig(credentials.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create config from kubeconfig: %w", err)
		}
	} else if credentials.ServiceAccountToken != "" {
		config = &rest.Config{
			Host:        cluster.Endpoint,
			BearerToken: credentials.ServiceAccountToken,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: credentials.CABundle,
			},
		}
	} else if len(credentials.ClientCertificate) > 0 {
		config = &rest.Config{
			Host: cluster.Endpoint,
			TLSClientConfig: rest.TLSClientConfig{
				CertData: credentials.ClientCertificate,
				KeyData:  credentials.ClientKey,
				CAData:   credentials.CABundle,
			},
		}
	} else {
		return nil, fmt.Errorf("no valid credentials provided")
	}

	// Create clients
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	runtimeClient, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime client: %w", err)
	}

	conn := &multicluster.ClusterConnection{
		Cluster:       cluster,
		Config:        config,
		Client:        kubeClient,
		RuntimeClient: runtimeClient,
		Connected:     true,
		LastConnected: time.Now(),
	}

	// Test connection
	if err := r.testConnection(ctx, conn); err != nil {
		conn.Connected = false
		conn.LastError = err
		return nil, err
	}

	return conn, nil
}

// testConnection tests if a cluster connection is valid
func (r *Registry) testConnection(ctx context.Context, conn *multicluster.ClusterConnection) error {
	// Test API server connectivity
	_, err := conn.Client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to API server: %w", err)
	}

	// Update connection status
	conn.Connected = true
	conn.LastConnected = time.Now()
	conn.LastError = nil

	return nil
}

// gatherClusterInfo gathers information about the cluster
func (r *Registry) gatherClusterInfo(ctx context.Context, cluster *multicluster.Cluster, conn *multicluster.ClusterConnection) error {
	// Get nodes for metrics
	nodes, err := conn.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	metrics := &multicluster.ClusterMetrics{
		NodeCount: int32(len(nodes.Items)),
		LastUpdated: time.Now(),
	}

	// Count ready nodes and gather capacity
	var cpuCapacity, memoryCapacity int64
	var readyNodes int32

	for _, node := range nodes.Items {
		// Check if node is ready
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				readyNodes++
				break
			}
		}

		// Aggregate capacity
		if cpu := node.Status.Capacity.Cpu(); cpu != nil {
			cpuCapacity += cpu.MilliValue()
		}
		if memory := node.Status.Capacity.Memory(); memory != nil {
			memoryCapacity += memory.Value()
		}
	}

	metrics.NodeReady = readyNodes
	metrics.CPUCapacity = float64(cpuCapacity) / 1000.0 // Convert to cores
	metrics.MemoryCapacity = memoryCapacity

	// Get pod metrics
	pods, err := conn.Client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		metrics.PodCount = int32(len(pods.Items))
	}

	// Detect features
	cluster.Features = r.detectFeatures(ctx, conn)

	cluster.Metrics = metrics
	return nil
}

// detectFeatures detects available features in the cluster
func (r *Registry) detectFeatures(ctx context.Context, conn *multicluster.ClusterConnection) []string {
	features := []string{}

	// Check for common resources/features
	apiGroups, err := conn.Client.Discovery().ServerGroups()
	if err == nil {
		for _, group := range apiGroups.Groups {
			switch group.Name {
			case "monitoring.coreos.com":
				features = append(features, "prometheus-operator")
			case "networking.istio.io":
				features = append(features, "istio")
			case "linkerd.io":
				features = append(features, "linkerd")
			case "cert-manager.io":
				features = append(features, "cert-manager")
			case "policy":
				features = append(features, "pod-security-policy")
			}
		}
	}

	return features
}
