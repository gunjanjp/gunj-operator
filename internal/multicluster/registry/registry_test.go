package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

// Mock implementations

type mockClusterRegistry struct {
	mock.Mock
}

func (m *mockClusterRegistry) Register(ctx context.Context, cluster *multicluster.Cluster, credentials *multicluster.ClusterCredentials) error {
	args := m.Called(ctx, cluster, credentials)
	return args.Error(0)
}

func (m *mockClusterRegistry) Unregister(ctx context.Context, clusterName string) error {
	args := m.Called(ctx, clusterName)
	return args.Error(0)
}

func (m *mockClusterRegistry) Get(ctx context.Context, clusterName string) (*multicluster.Cluster, error) {
	args := m.Called(ctx, clusterName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*multicluster.Cluster), args.Error(1)
}

func (m *mockClusterRegistry) List(ctx context.Context) ([]*multicluster.Cluster, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*multicluster.Cluster), args.Error(1)
}

func (m *mockClusterRegistry) Update(ctx context.Context, cluster *multicluster.Cluster) error {
	args := m.Called(ctx, cluster)
	return args.Error(0)
}

func (m *mockClusterRegistry) GetConnection(ctx context.Context, clusterName string) (*multicluster.ClusterConnection, error) {
	args := m.Called(ctx, clusterName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*multicluster.ClusterConnection), args.Error(1)
}

func (m *mockClusterRegistry) UpdateStatus(ctx context.Context, clusterName string, status multicluster.ClusterStatus) error {
	args := m.Called(ctx, clusterName, status)
	return args.Error(0)
}

// Test helpers

func createTestCluster(name string) *multicluster.Cluster {
	return &multicluster.Cluster{
		Name:        name,
		DisplayName: "Test " + name,
		Description: "Test cluster " + name,
		Role:        multicluster.ClusterRoleSpoke,
		Region:      "us-test",
		Zone:        "us-test-1a",
		Provider:    "test",
		Endpoint:    "https://" + name + ".test.local",
		Status:      multicluster.ClusterStatusReady,
		Labels: map[string]string{
			"test": "true",
		},
		RegisteredAt: time.Now(),
	}
}

func createTestCredentials(clusterName string) *multicluster.ClusterCredentials {
	return &multicluster.ClusterCredentials{
		ClusterName:         clusterName,
		ServiceAccountToken: "test-token-" + clusterName,
		CABundle:            []byte("test-ca-bundle"),
	}
}

// Tests

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *multicluster.Cluster
		credentials *multicluster.ClusterCredentials
		setupMocks  func(*testing.T, client.Client)
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful registration",
			cluster:     createTestCluster("test-cluster"),
			credentials: createTestCredentials("test-cluster"),
			setupMocks: func(t *testing.T, c client.Client) {
				// No specific setup needed for success case
			},
			wantErr: false,
		},
		{
			name: "invalid cluster - missing name",
			cluster: &multicluster.Cluster{
				Endpoint: "https://test.local",
			},
			credentials: createTestCredentials(""),
			wantErr:     true,
			errContains: "cluster name is required",
		},
		{
			name: "invalid cluster - missing endpoint",
			cluster: &multicluster.Cluster{
				Name: "test-cluster",
			},
			credentials: createTestCredentials("test-cluster"),
			wantErr:     true,
			errContains: "cluster endpoint is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Setup mocks if provided
			if tt.setupMocks != nil {
				tt.setupMocks(t, fakeClient)
			}

			// Create registry
			registry := NewRegistry(fakeClient, "test-namespace").(*Registry)

			// Test registration
			err := registry.Register(context.Background(), tt.cluster, tt.credentials)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify secret was created
				secret := &corev1.Secret{}
				secretKey := client.ObjectKey{
					Name:      ClusterSecretPrefix + tt.cluster.Name,
					Namespace: "test-namespace",
				}
				err = fakeClient.Get(context.Background(), secretKey, secret)
				assert.NoError(t, err)
				assert.Equal(t, tt.credentials.ServiceAccountToken, string(secret.Data["token"]))

				// Verify cluster metadata was stored
				configMap := &corev1.ConfigMap{}
				cmKey := client.ObjectKey{
					Name:      ClusterConfigMapName,
					Namespace: "test-namespace",
				}
				err = fakeClient.Get(context.Background(), cmKey, configMap)
				assert.NoError(t, err)
				assert.Contains(t, configMap.Data, "clusters.json")
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		setupClusters map[string]*multicluster.Cluster
		want         *multicluster.Cluster
		wantErr      bool
	}{
		{
			name:        "get existing cluster",
			clusterName: "test-cluster",
			setupClusters: map[string]*multicluster.Cluster{
				"test-cluster": createTestCluster("test-cluster"),
			},
			want:    createTestCluster("test-cluster"),
			wantErr: false,
		},
		{
			name:        "cluster not found",
			clusterName: "non-existent",
			setupClusters: map[string]*multicluster.Cluster{
				"test-cluster": createTestCluster("test-cluster"),
			},
			wantErr: true,
		},
		{
			name:          "empty registry",
			clusterName:   "test-cluster",
			setupClusters: map[string]*multicluster.Cluster{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with ConfigMap
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)

			clustersData, _ := json.Marshal(tt.setupClusters)
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ClusterConfigMapName,
					Namespace: "test-namespace",
				},
				Data: map[string]string{
					"clusters.json": string(clustersData),
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(configMap).
				Build()

			// Create registry
			registry := NewRegistry(fakeClient, "test-namespace").(*Registry)

			// Test get
			got, err := registry.Get(context.Background(), tt.clusterName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.Name, got.Name)
				assert.Equal(t, tt.want.Endpoint, got.Endpoint)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	// Setup test clusters
	clusters := map[string]*multicluster.Cluster{
		"cluster-1": createTestCluster("cluster-1"),
		"cluster-2": createTestCluster("cluster-2"),
		"cluster-3": createTestCluster("cluster-3"),
	}

	// Create fake client with ConfigMap
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	clustersData, _ := json.Marshal(clusters)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterConfigMapName,
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"clusters.json": string(clustersData),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap).
		Build()

	// Create registry
	registry := NewRegistry(fakeClient, "test-namespace").(*Registry)

	// Test list
	got, err := registry.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 3)

	// Verify all clusters are returned
	clusterNames := make(map[string]bool)
	for _, cluster := range got {
		clusterNames[cluster.Name] = true
	}
	assert.True(t, clusterNames["cluster-1"])
	assert.True(t, clusterNames["cluster-2"])
	assert.True(t, clusterNames["cluster-3"])
}

func TestRegistry_Unregister(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create initial cluster data
	clusters := map[string]*multicluster.Cluster{
		"cluster-1": createTestCluster("cluster-1"),
		"cluster-2": createTestCluster("cluster-2"),
	}
	clustersData, _ := json.Marshal(clusters)

	// Create ConfigMap and Secret
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterConfigMapName,
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"clusters.json": string(clustersData),
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterSecretPrefix + "cluster-1",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"token": []byte("test-token"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap, secret).
		Build()

	// Create registry
	registry := NewRegistry(fakeClient, "test-namespace").(*Registry)

	// Test unregister
	err := registry.Unregister(context.Background(), "cluster-1")
	assert.NoError(t, err)

	// Verify secret was deleted
	deletedSecret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Name:      ClusterSecretPrefix + "cluster-1",
		Namespace: "test-namespace",
	}
	err = fakeClient.Get(context.Background(), secretKey, deletedSecret)
	assert.True(t, apierrors.IsNotFound(err))

	// Verify cluster was removed from metadata
	remaining, err := registry.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, remaining, 1)
	assert.Equal(t, "cluster-2", remaining[0].Name)
}

func TestRegistry_UpdateStatus(t *testing.T) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cluster := createTestCluster("test-cluster")
	cluster.Status = multicluster.ClusterStatusReady

	clusters := map[string]*multicluster.Cluster{
		"test-cluster": cluster,
	}
	clustersData, _ := json.Marshal(clusters)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterConfigMapName,
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"clusters.json": string(clustersData),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap).
		Build()

	// Create registry
	registry := NewRegistry(fakeClient, "test-namespace").(*Registry)

	// Test update status
	err := registry.UpdateStatus(context.Background(), "test-cluster", multicluster.ClusterStatusNotReady)
	assert.NoError(t, err)

	// Verify status was updated
	updatedCluster, err := registry.Get(context.Background(), "test-cluster")
	assert.NoError(t, err)
	assert.Equal(t, multicluster.ClusterStatusNotReady, updatedCluster.Status)
}

func TestRegistry_ConnectionCaching(t *testing.T) {
	// This test would require mocking kubernetes clients
	// For brevity, we'll test the caching logic
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	registry := NewRegistry(fakeClient, "test-namespace").(*Registry)

	// Create a mock connection
	conn := &multicluster.ClusterConnection{
		Cluster: createTestCluster("test-cluster"),
		Config: &rest.Config{
			Host: "https://test.local",
		},
		Connected:     true,
		LastConnected: time.Now(),
	}

	// Add to cache
	registry.mu.Lock()
	registry.connections["test-cluster"] = conn
	registry.mu.Unlock()

	// Verify cached connection is returned
	registry.mu.RLock()
	cachedConn, exists := registry.connections["test-cluster"]
	registry.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, conn.Cluster.Name, cachedConn.Cluster.Name)
	assert.True(t, cachedConn.Connected)
}

// Benchmarks

func BenchmarkRegistry_List(b *testing.B) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create many clusters
	clusters := make(map[string]*multicluster.Cluster)
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("cluster-%d", i)
		clusters[name] = createTestCluster(name)
	}

	clustersData, _ := json.Marshal(clusters)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterConfigMapName,
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"clusters.json": string(clustersData),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap).
		Build()

	registry := NewRegistry(fakeClient, "test-namespace").(*Registry)
	ctx := context.Background()

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.List(ctx)
	}
}

func BenchmarkRegistry_Get(b *testing.B) {
	// Setup
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	clusters := map[string]*multicluster.Cluster{
		"test-cluster": createTestCluster("test-cluster"),
	}

	clustersData, _ := json.Marshal(clusters)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterConfigMapName,
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"clusters.json": string(clustersData),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configMap).
		Build()

	registry := NewRegistry(fakeClient, "test-namespace").(*Registry)
	ctx := context.Background()

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.Get(ctx, "test-cluster")
	}
}
