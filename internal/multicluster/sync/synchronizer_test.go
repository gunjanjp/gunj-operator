package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

// Mock registry
type mockRegistry struct {
	mock.Mock
}

func (m *mockRegistry) Register(ctx context.Context, cluster *multicluster.Cluster, credentials *multicluster.ClusterCredentials) error {
	args := m.Called(ctx, cluster, credentials)
	return args.Error(0)
}

func (m *mockRegistry) Unregister(ctx context.Context, clusterName string) error {
	args := m.Called(ctx, clusterName)
	return args.Error(0)
}

func (m *mockRegistry) Get(ctx context.Context, clusterName string) (*multicluster.Cluster, error) {
	args := m.Called(ctx, clusterName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*multicluster.Cluster), args.Error(1)
}

func (m *mockRegistry) List(ctx context.Context) ([]*multicluster.Cluster, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*multicluster.Cluster), args.Error(1)
}

func (m *mockRegistry) Update(ctx context.Context, cluster *multicluster.Cluster) error {
	args := m.Called(ctx, cluster)
	return args.Error(0)
}

func (m *mockRegistry) GetConnection(ctx context.Context, clusterName string) (*multicluster.ClusterConnection, error) {
	args := m.Called(ctx, clusterName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*multicluster.ClusterConnection), args.Error(1)
}

func (m *mockRegistry) UpdateStatus(ctx context.Context, clusterName string, status multicluster.ClusterStatus) error {
	args := m.Called(ctx, clusterName, status)
	return args.Error(0)
}

// Test helpers

func createTestSync(name string) *multicluster.ResourceSync {
	return &multicluster.ResourceSync{
		Name:           name,
		SourceCluster:  "source-cluster",
		TargetClusters: []string{"target-1", "target-2"},
		ResourceType:   "configmaps",
		Namespace:      "test-namespace",
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"sync": "enabled",
			},
		},
		ConflictResolution: "update",
		Bidirectional:      false,
		Status: multicluster.SyncStatus{
			Phase: "Pending",
		},
	}
}

func createTestConfigMap(name string) *unstructured.Unstructured {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-namespace",
			Labels: map[string]string{
				"sync": "enabled",
			},
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	// Convert to unstructured
	unstructuredCM := &unstructured.Unstructured{}
	unstructuredCM.SetAPIVersion("v1")
	unstructuredCM.SetKind("ConfigMap")
	unstructuredCM.SetName(name)
	unstructuredCM.SetNamespace("test-namespace")
	unstructuredCM.SetLabels(cm.Labels)
	unstructuredCM.Object["data"] = cm.Data

	return unstructuredCM
}

// Tests

func TestSynchronizer_CreateSync(t *testing.T) {
	tests := []struct {
		name        string
		sync        *multicluster.ResourceSync
		setupMocks  func(*mockRegistry)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful sync creation",
			sync: createTestSync("test-sync"),
			setupMocks: func(m *mockRegistry) {
				// Mock successful cluster connection
				m.On("GetConnection", mock.Anything, "source-cluster").Return(&multicluster.ClusterConnection{
					Cluster: &multicluster.Cluster{
						Name: "source-cluster",
					},
					Config: &rest.Config{
						Host: "https://source.test",
					},
					Connected: true,
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "invalid sync - no name",
			sync: &multicluster.ResourceSync{
				SourceCluster:  "source",
				TargetClusters: []string{"target"},
				ResourceType:   "configmaps",
				Namespace:      "default",
			},
			wantErr:     true,
			errContains: "sync name is required",
		},
		{
			name: "invalid sync - no source cluster",
			sync: &multicluster.ResourceSync{
				Name:           "test-sync",
				TargetClusters: []string{"target"},
				ResourceType:   "configmaps",
				Namespace:      "default",
			},
			wantErr:     true,
			errContains: "source cluster is required",
		},
		{
			name: "invalid sync - no target clusters",
			sync: &multicluster.ResourceSync{
				Name:          "test-sync",
				SourceCluster: "source",
				ResourceType:  "configmaps",
				Namespace:     "default",
			},
			wantErr:     true,
			errContains: "at least one target cluster is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock registry
			mockReg := new(mockRegistry)
			if tt.setupMocks != nil {
				tt.setupMocks(mockReg)
			}

			// Create fake client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create synchronizer
			sync := NewSynchronizer(mockReg, fakeClient).(*Synchronizer)

			// Test create sync
			err := sync.CreateSync(context.Background(), tt.sync)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify sync was stored
				syncs, err := sync.ListSyncs(context.Background())
				assert.NoError(t, err)
				assert.Len(t, syncs, 1)
				assert.Equal(t, tt.sync.Name, syncs[0].Name)
			}

			mockReg.AssertExpectations(t)
		})
	}
}

func TestSynchronizer_SyncNow(t *testing.T) {
	// Create test resources
	cm1 := createTestConfigMap("config-1")
	cm2 := createTestConfigMap("config-2")

	// Create mock registry
	mockReg := new(mockRegistry)

	// Mock source cluster connection
	sourceConn := &multicluster.ClusterConnection{
		Cluster: &multicluster.Cluster{
			Name: "source-cluster",
		},
		Config: &rest.Config{
			Host: "https://source.test",
		},
		RuntimeClient: fake.NewClientBuilder().
			WithRuntimeObjects(cm1, cm2).
			Build(),
		Connected: true,
	}

	// Mock target cluster connections
	targetConn1 := &multicluster.ClusterConnection{
		Cluster: &multicluster.Cluster{
			Name: "target-1",
		},
		Config: &rest.Config{
			Host: "https://target1.test",
		},
		RuntimeClient: fake.NewClientBuilder().Build(),
		Connected:     true,
	}

	targetConn2 := &multicluster.ClusterConnection{
		Cluster: &multicluster.Cluster{
			Name: "target-2",
		},
		Config: &rest.Config{
			Host: "https://target2.test",
		},
		RuntimeClient: fake.NewClientBuilder().Build(),
		Connected:     true,
	}

	mockReg.On("GetConnection", mock.Anything, "source-cluster").Return(sourceConn, nil)
	mockReg.On("GetConnection", mock.Anything, "target-1").Return(targetConn1, nil)
	mockReg.On("GetConnection", mock.Anything, "target-2").Return(targetConn2, nil)

	// Create synchronizer
	fakeClient := fake.NewClientBuilder().Build()
	sync := NewSynchronizer(mockReg, fakeClient).(*Synchronizer)

	// Create and store sync
	testSync := createTestSync("test-sync")
	sync.syncs[testSync.Name] = testSync

	// Test sync now
	err := sync.SyncNow(context.Background(), testSync.Name)
	assert.NoError(t, err)

	// Verify sync status was updated
	status, err := sync.GetSyncStatus(context.Background(), testSync.Name)
	assert.NoError(t, err)
	assert.Equal(t, "Synced", status.Phase)
	assert.Equal(t, int32(2), status.ResourcesSynced) // 2 configmaps

	mockReg.AssertExpectations(t)
}

func TestSynchronizer_DeleteSync(t *testing.T) {
	// Create mock registry
	mockReg := new(mockRegistry)

	// Create synchronizer
	fakeClient := fake.NewClientBuilder().Build()
	sync := NewSynchronizer(mockReg, fakeClient).(*Synchronizer)

	// Create and store sync
	testSync := createTestSync("test-sync")
	sync.syncs[testSync.Name] = testSync

	// Add mock watcher and event channel
	ctx, cancel := context.WithCancel(context.Background())
	sync.watchers[testSync.Name] = cancel
	sync.eventChannels[testSync.Name] = make(chan *multicluster.ClusterEvent, 1)

	// Test delete sync
	err := sync.DeleteSync(context.Background(), testSync.Name)
	assert.NoError(t, err)

	// Verify sync was removed
	syncs, err := sync.ListSyncs(context.Background())
	assert.NoError(t, err)
	assert.Len(t, syncs, 0)

	// Verify watcher was stopped
	_, exists := sync.watchers[testSync.Name]
	assert.False(t, exists)

	// Verify event channel was closed
	_, exists = sync.eventChannels[testSync.Name]
	assert.False(t, exists)
}

func TestSynchronizer_ApplyTransformations(t *testing.T) {
	tests := []struct {
		name       string
		resource   *unstructured.Unstructured
		transforms []multicluster.TransformRule
		wantData   map[string]interface{}
	}{
		{
			name:     "add transformation",
			resource: createTestConfigMap("test"),
			transforms: []multicluster.TransformRule{
				{
					Type:  "add",
					Path:  "metadata.labels.transformed",
					Value: "true",
				},
			},
			wantData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"sync":        "enabled",
						"transformed": "true",
					},
				},
			},
		},
		{
			name:     "remove transformation",
			resource: createTestConfigMap("test"),
			transforms: []multicluster.TransformRule{
				{
					Type: "remove",
					Path: "metadata.labels.sync",
				},
			},
			wantData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{},
				},
			},
		},
		{
			name:     "replace transformation",
			resource: createTestConfigMap("test"),
			transforms: []multicluster.TransformRule{
				{
					Type:  "replace",
					Path:  "data.key",
					Value: "new-value",
				},
			},
			wantData: map[string]interface{}{
				"data": map[string]interface{}{
					"key": "new-value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create synchronizer
			sync := &Synchronizer{}

			// Apply transformations
			err := sync.applyTransformations(tt.resource, tt.transforms)
			assert.NoError(t, err)

			// Verify transformations
			for path, expectedValue := range tt.wantData {
				actualValue, found, err := unstructured.NestedMap(tt.resource.Object, path)
				assert.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, expectedValue, actualValue)
			}
		})
	}
}

func TestSynchronizer_ParseResourceType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		want         schema.GroupVersionResource
		wantErr      bool
	}{
		{
			name:         "core resource",
			resourceType: "configmaps",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "configmaps",
			},
			wantErr: false,
		},
		{
			name:         "full GVR",
			resourceType: "apps/v1/deployments",
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			wantErr: false,
		},
		{
			name:         "invalid format",
			resourceType: "invalid/format",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sync := &Synchronizer{}
			got, err := sync.parseResourceType(tt.resourceType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSynchronizer_ResolveConflict(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		existing *unstructured.Unstructured
		desired  *unstructured.Unstructured
		want     bool
	}{
		{
			name:     "update strategy",
			strategy: "update",
			existing: createTestConfigMap("test"),
			desired:  createTestConfigMap("test"),
			want:     true,
		},
		{
			name:     "skip strategy",
			strategy: "skip",
			existing: createTestConfigMap("test"),
			desired:  createTestConfigMap("test"),
			want:     false,
		},
		{
			name:     "newer strategy - desired is newer",
			strategy: "newer",
			existing: func() *unstructured.Unstructured {
				cm := createTestConfigMap("test")
				cm.SetAnnotations(map[string]string{
					LastSyncAnnotation: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				})
				return cm
			}(),
			desired: func() *unstructured.Unstructured {
				cm := createTestConfigMap("test")
				cm.SetAnnotations(map[string]string{
					LastSyncAnnotation: time.Now().Format(time.RFC3339),
				})
				return cm
			}(),
			want: true,
		},
		{
			name:     "newer strategy - existing is newer",
			strategy: "newer",
			existing: func() *unstructured.Unstructured {
				cm := createTestConfigMap("test")
				cm.SetAnnotations(map[string]string{
					LastSyncAnnotation: time.Now().Format(time.RFC3339),
				})
				return cm
			}(),
			desired: func() *unstructured.Unstructured {
				cm := createTestConfigMap("test")
				cm.SetAnnotations(map[string]string{
					LastSyncAnnotation: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				})
				return cm
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sync := &Synchronizer{}
			got := sync.resolveConflict(tt.strategy, tt.existing, tt.desired)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Benchmarks

func BenchmarkSynchronizer_ApplyTransformations(b *testing.B) {
	sync := &Synchronizer{}
	resource := createTestConfigMap("bench-test")
	transforms := []multicluster.TransformRule{
		{Type: "add", Path: "metadata.labels.bench1", Value: "value1"},
		{Type: "add", Path: "metadata.labels.bench2", Value: "value2"},
		{Type: "replace", Path: "data.key", Value: "new-value"},
		{Type: "remove", Path: "metadata.labels.sync"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a fresh copy for each iteration
		res := resource.DeepCopy()
		_ = sync.applyTransformations(res, transforms)
	}
}
