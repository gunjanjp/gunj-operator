package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

const (
	// SyncAnnotationPrefix for sync-related annotations
	SyncAnnotationPrefix = "sync.gunj.io/"
	// LastSyncAnnotation records the last sync time
	LastSyncAnnotation = SyncAnnotationPrefix + "last-sync"
	// SourceClusterAnnotation records the source cluster
	SourceClusterAnnotation = SyncAnnotationPrefix + "source-cluster"
	// SourceResourceAnnotation records the source resource
	SourceResourceAnnotation = SyncAnnotationPrefix + "source-resource"
)

// Synchronizer implements cross-cluster resource synchronization
type Synchronizer struct {
	// registry for cluster connections
	registry multicluster.ClusterRegistry
	// hubClient is the client for the hub cluster
	hubClient client.Client
	// log is the logger
	log logr.Logger
	// syncs stores sync configurations
	syncs map[string]*multicluster.ResourceSync
	// watchers stores active watchers
	watchers map[string]context.CancelFunc
	// eventChannels stores event channels for syncs
	eventChannels map[string]chan *multicluster.ClusterEvent
	// mu for thread safety
	mu sync.RWMutex
}

// NewSynchronizer creates a new resource synchronizer
func NewSynchronizer(registry multicluster.ClusterRegistry, hubClient client.Client) multicluster.ResourceSynchronizer {
	return &Synchronizer{
		registry:      registry,
		hubClient:     hubClient,
		log:           log.Log.WithName("resource-synchronizer"),
		syncs:         make(map[string]*multicluster.ResourceSync),
		watchers:      make(map[string]context.CancelFunc),
		eventChannels: make(map[string]chan *multicluster.ClusterEvent),
	}
}

// CreateSync creates a new resource sync configuration
func (s *Synchronizer) CreateSync(ctx context.Context, sync *multicluster.ResourceSync) error {
	s.log.Info("Creating resource sync", "name", sync.Name)

	// Validate sync configuration
	if err := s.validateSync(sync); err != nil {
		return fmt.Errorf("invalid sync configuration: %w", err)
	}

	// Check if sync already exists
	s.mu.Lock()
	if _, exists := s.syncs[sync.Name]; exists {
		s.mu.Unlock()
		return fmt.Errorf("sync %s already exists", sync.Name)
	}

	// Initialize sync status
	sync.Status = multicluster.SyncStatus{
		Phase:      "Pending",
		Message:    "Sync created",
		Conditions: []metav1.Condition{},
	}
	sync.LastSyncTime = time.Now()

	// Store sync configuration
	s.syncs[sync.Name] = sync
	s.mu.Unlock()

	// Start watching source resources
	if err := s.startWatcher(ctx, sync); err != nil {
		s.mu.Lock()
		delete(s.syncs, sync.Name)
		s.mu.Unlock()
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Perform initial sync
	go func() {
		if err := s.SyncNow(context.Background(), sync.Name); err != nil {
			s.log.Error(err, "Initial sync failed", "sync", sync.Name)
		}
	}()

	s.log.Info("Successfully created resource sync", "name", sync.Name)
	return nil
}

// DeleteSync removes a sync configuration
func (s *Synchronizer) DeleteSync(ctx context.Context, name string) error {
	s.log.Info("Deleting resource sync", "name", name)

	s.mu.Lock()
	defer s.mu.Unlock()

	sync, exists := s.syncs[name]
	if !exists {
		return fmt.Errorf("sync %s not found", name)
	}

	// Stop watcher
	if cancel, exists := s.watchers[name]; exists {
		cancel()
		delete(s.watchers, name)
	}

	// Close event channel
	if ch, exists := s.eventChannels[name]; exists {
		close(ch)
		delete(s.eventChannels, name)
	}

	// Clean up synced resources
	if err := s.cleanupSyncedResources(ctx, sync); err != nil {
		s.log.Error(err, "Failed to cleanup synced resources", "sync", name)
	}

	delete(s.syncs, name)
	s.log.Info("Successfully deleted resource sync", "name", name)
	return nil
}

// ListSyncs returns all sync configurations
func (s *Synchronizer) ListSyncs(ctx context.Context) ([]*multicluster.ResourceSync, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*multicluster.ResourceSync, 0, len(s.syncs))
	for _, sync := range s.syncs {
		result = append(result, sync.DeepCopy())
	}

	return result, nil
}

// SyncNow triggers an immediate sync
func (s *Synchronizer) SyncNow(ctx context.Context, syncName string) error {
	s.log.Info("Triggering immediate sync", "sync", syncName)

	s.mu.RLock()
	sync, exists := s.syncs[syncName]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("sync %s not found", syncName)
	}

	// Update sync status
	s.updateSyncStatus(syncName, multicluster.SyncStatus{
		Phase:   "Syncing",
		Message: "Sync in progress",
	})

	// Get source cluster connection
	sourceConn, err := s.registry.GetConnection(ctx, sync.SourceCluster)
	if err != nil {
		return s.handleSyncError(syncName, fmt.Errorf("failed to connect to source cluster: %w", err))
	}

	// Get resources from source cluster
	resources, err := s.getResources(ctx, sourceConn, sync)
	if err != nil {
		return s.handleSyncError(syncName, fmt.Errorf("failed to get resources: %w", err))
	}

	// Sync to each target cluster
	var syncedCount int32
	var errors []error

	for _, targetCluster := range sync.TargetClusters {
		count, err := s.syncToCluster(ctx, sync, resources, targetCluster)
		if err != nil {
			errors = append(errors, fmt.Errorf("cluster %s: %w", targetCluster, err))
		} else {
			syncedCount += count
		}
	}

	// Update sync status
	status := multicluster.SyncStatus{
		Phase:           "Synced",
		Message:         fmt.Sprintf("Synced %d resources", syncedCount),
		ResourcesSynced: syncedCount,
	}

	if len(errors) > 0 {
		status.Phase = "PartialSync"
		status.LastError = fmt.Sprintf("%d errors occurred", len(errors))
	}

	s.updateSyncStatus(syncName, status)
	s.updateLastSyncTime(syncName, time.Now())

	if len(errors) > 0 {
		return fmt.Errorf("sync completed with errors: %v", errors)
	}

	s.log.Info("Successfully completed sync", "sync", syncName, "resources", syncedCount)
	return nil
}

// GetSyncStatus returns the status of a sync
func (s *Synchronizer) GetSyncStatus(ctx context.Context, syncName string) (*multicluster.SyncStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sync, exists := s.syncs[syncName]
	if !exists {
		return nil, fmt.Errorf("sync %s not found", syncName)
	}

	return &sync.Status, nil
}

// WatchSync watches for changes in synced resources
func (s *Synchronizer) WatchSync(ctx context.Context, syncName string) (<-chan *multicluster.ClusterEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.syncs[syncName]; !exists {
		return nil, fmt.Errorf("sync %s not found", syncName)
	}

	// Create or return existing event channel
	if ch, exists := s.eventChannels[syncName]; exists {
		return ch, nil
	}

	ch := make(chan *multicluster.ClusterEvent, 100)
	s.eventChannels[syncName] = ch

	return ch, nil
}

// validateSync validates sync configuration
func (s *Synchronizer) validateSync(sync *multicluster.ResourceSync) error {
	if sync.Name == "" {
		return fmt.Errorf("sync name is required")
	}
	if sync.SourceCluster == "" {
		return fmt.Errorf("source cluster is required")
	}
	if len(sync.TargetClusters) == 0 {
		return fmt.Errorf("at least one target cluster is required")
	}
	if sync.ResourceType == "" {
		return fmt.Errorf("resource type is required")
	}
	if sync.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if sync.ConflictResolution == "" {
		sync.ConflictResolution = "update" // default to update
	}
	return nil
}

// startWatcher starts watching resources in the source cluster
func (s *Synchronizer) startWatcher(ctx context.Context, sync *multicluster.ResourceSync) error {
	// Get source cluster connection
	sourceConn, err := s.registry.GetConnection(ctx, sync.SourceCluster)
	if err != nil {
		return fmt.Errorf("failed to connect to source cluster: %w", err)
	}

	// Parse resource type
	gvr, err := s.parseResourceType(sync.ResourceType)
	if err != nil {
		return fmt.Errorf("failed to parse resource type: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(sourceConn.Config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create watch context
	watchCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.watchers[sync.Name] = cancel
	s.mu.Unlock()

	// Start watching in background
	go s.watchResources(watchCtx, sync, dynamicClient, gvr)

	return nil
}

// watchResources watches for resource changes
func (s *Synchronizer) watchResources(ctx context.Context, sync *multicluster.ResourceSync, client dynamic.Interface, gvr schema.GroupVersionResource) {
	s.log.Info("Starting resource watcher", "sync", sync.Name, "resource", gvr.String())

	// Create informer
	informer := cache.NewSharedInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if sync.Selector != nil {
					options.LabelSelector = labels.Set(sync.Selector.MatchLabels).String()
				}
				return client.Resource(gvr).Namespace(sync.Namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if sync.Selector != nil {
					options.LabelSelector = labels.Set(sync.Selector.MatchLabels).String()
				}
				return client.Resource(gvr).Namespace(sync.Namespace).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		time.Minute,
	)

	// Add event handlers
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.handleResourceEvent(sync, "add", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			s.handleResourceEvent(sync, "update", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			s.handleResourceEvent(sync, "delete", obj)
		},
	})

	// Run informer
	informer.Run(ctx.Done())
	s.log.Info("Resource watcher stopped", "sync", sync.Name)
}

// handleResourceEvent handles a resource change event
func (s *Synchronizer) handleResourceEvent(sync *multicluster.ResourceSync, eventType string, obj interface{}) {
	// Convert to unstructured
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		s.log.Error(nil, "Failed to convert object to unstructured", "type", reflect.TypeOf(obj))
		return
	}

	// Create cluster event
	event := &multicluster.ClusterEvent{
		Type:      eventType,
		Cluster:   sync.SourceCluster,
		Object:    u,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Resource %s in sync %s", eventType, sync.Name),
		Severity:  "info",
	}

	// Send to event channel if exists
	s.mu.RLock()
	ch, exists := s.eventChannels[sync.Name]
	s.mu.RUnlock()

	if exists {
		select {
		case ch <- event:
		default:
			s.log.Warn("Event channel full, dropping event", "sync", sync.Name)
		}
	}

	// Trigger sync
	go func() {
		if err := s.SyncNow(context.Background(), sync.Name); err != nil {
			s.log.Error(err, "Failed to sync after event", "sync", sync.Name, "event", eventType)
		}
	}()
}

// getResources gets resources from the source cluster
func (s *Synchronizer) getResources(ctx context.Context, conn *multicluster.ClusterConnection, sync *multicluster.ResourceSync) ([]*unstructured.Unstructured, error) {
	// Parse resource type
	gvr, err := s.parseResourceType(sync.ResourceType)
	if err != nil {
		return nil, err
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(conn.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// List resources
	listOptions := metav1.ListOptions{}
	if sync.Selector != nil {
		listOptions.LabelSelector = labels.Set(sync.Selector.MatchLabels).String()
	}

	list, err := dynamicClient.Resource(gvr).Namespace(sync.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Convert to array
	resources := make([]*unstructured.Unstructured, 0, len(list.Items))
	for i := range list.Items {
		resources = append(resources, &list.Items[i])
	}

	return resources, nil
}

// syncToCluster syncs resources to a target cluster
func (s *Synchronizer) syncToCluster(ctx context.Context, sync *multicluster.ResourceSync, resources []*unstructured.Unstructured, targetCluster string) (int32, error) {
	// Get target cluster connection
	targetConn, err := s.registry.GetConnection(ctx, targetCluster)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to target cluster: %w", err)
	}

	// Parse resource type
	gvr, err := s.parseResourceType(sync.ResourceType)
	if err != nil {
		return 0, err
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(targetConn.Config)
	if err != nil {
		return 0, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	var syncedCount int32
	for _, resource := range resources {
		// Apply transformations
		transformed := resource.DeepCopy()
		if err := s.applyTransformations(transformed, sync.Transform); err != nil {
			s.log.Error(err, "Failed to apply transformations", "resource", resource.GetName())
			continue
		}

		// Add sync annotations
		annotations := transformed.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[LastSyncAnnotation] = time.Now().UTC().Format(time.RFC3339)
		annotations[SourceClusterAnnotation] = sync.SourceCluster
		annotations[SourceResourceAnnotation] = fmt.Sprintf("%s/%s", resource.GetNamespace(), resource.GetName())
		transformed.SetAnnotations(annotations)

		// Remove cluster-specific metadata
		transformed.SetResourceVersion("")
		transformed.SetUID("")
		transformed.SetCreationTimestamp(metav1.Time{})
		transformed.SetDeletionTimestamp(nil)
		transformed.SetGeneration(0)

		// Check if resource exists in target
		existing, err := dynamicClient.Resource(gvr).Namespace(sync.Namespace).Get(ctx, transformed.GetName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Create resource
				if _, err := dynamicClient.Resource(gvr).Namespace(sync.Namespace).Create(ctx, transformed, metav1.CreateOptions{}); err != nil {
					s.log.Error(err, "Failed to create resource", "resource", transformed.GetName(), "cluster", targetCluster)
					continue
				}
				syncedCount++
			} else {
				s.log.Error(err, "Failed to get existing resource", "resource", transformed.GetName(), "cluster", targetCluster)
				continue
			}
		} else {
			// Handle conflict resolution
			if shouldUpdate := s.resolveConflict(sync.ConflictResolution, existing, transformed); shouldUpdate {
				// Preserve target-specific metadata
				transformed.SetResourceVersion(existing.GetResourceVersion())
				
				if _, err := dynamicClient.Resource(gvr).Namespace(sync.Namespace).Update(ctx, transformed, metav1.UpdateOptions{}); err != nil {
					s.log.Error(err, "Failed to update resource", "resource", transformed.GetName(), "cluster", targetCluster)
					continue
				}
				syncedCount++
			}
		}
	}

	// Handle bidirectional sync if enabled
	if sync.Bidirectional {
		// This would sync resources from target back to source
		// Implementation omitted for brevity
	}

	return syncedCount, nil
}

// applyTransformations applies transformation rules to a resource
func (s *Synchronizer) applyTransformations(resource *unstructured.Unstructured, transforms []multicluster.TransformRule) error {
	for _, transform := range transforms {
		switch transform.Type {
		case "add":
			if err := unstructured.SetNestedField(resource.Object, transform.Value, strings.Split(transform.Path, ".")...); err != nil {
				return fmt.Errorf("failed to add field %s: %w", transform.Path, err)
			}
		case "remove":
			unstructured.RemoveNestedField(resource.Object, strings.Split(transform.Path, ".")...)
		case "replace":
			if err := unstructured.SetNestedField(resource.Object, transform.Value, strings.Split(transform.Path, ".")...); err != nil {
				return fmt.Errorf("failed to replace field %s: %w", transform.Path, err)
			}
		case "regex":
			// Get current value
			value, found, err := unstructured.NestedString(resource.Object, strings.Split(transform.Path, ".")...)
			if err != nil || !found {
				continue
			}
			// Apply regex replacement
			// Implementation simplified - would use regexp package
			newValue := strings.Replace(value, transform.Pattern, transform.Replacement, -1)
			if err := unstructured.SetNestedField(resource.Object, newValue, strings.Split(transform.Path, ".")...); err != nil {
				return fmt.Errorf("failed to apply regex to field %s: %w", transform.Path, err)
			}
		}
	}
	return nil
}

// resolveConflict determines whether to update a resource based on conflict resolution strategy
func (s *Synchronizer) resolveConflict(strategy string, existing, desired *unstructured.Unstructured) bool {
	switch strategy {
	case "update":
		return true
	case "skip":
		return false
	case "newer":
		// Compare last sync times
		existingTime := s.getLastSyncTime(existing)
		desiredTime := s.getLastSyncTime(desired)
		return desiredTime.After(existingTime)
	default:
		return true
	}
}

// getLastSyncTime gets the last sync time from annotations
func (s *Synchronizer) getLastSyncTime(resource *unstructured.Unstructured) time.Time {
	annotations := resource.GetAnnotations()
	if annotations == nil {
		return time.Time{}
	}

	if timeStr, exists := annotations[LastSyncAnnotation]; exists {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// cleanupSyncedResources removes resources that were synced by this sync configuration
func (s *Synchronizer) cleanupSyncedResources(ctx context.Context, sync *multicluster.ResourceSync) error {
	// Parse resource type
	gvr, err := s.parseResourceType(sync.ResourceType)
	if err != nil {
		return err
	}

	var errors []error
	for _, targetCluster := range sync.TargetClusters {
		// Get target cluster connection
		targetConn, err := s.registry.GetConnection(ctx, targetCluster)
		if err != nil {
			errors = append(errors, fmt.Errorf("cluster %s: %w", targetCluster, err))
			continue
		}

		// Create dynamic client
		dynamicClient, err := dynamic.NewForConfig(targetConn.Config)
		if err != nil {
			errors = append(errors, fmt.Errorf("cluster %s: failed to create client: %w", targetCluster, err))
			continue
		}

		// List resources with sync annotations
		list, err := dynamicClient.Resource(gvr).Namespace(sync.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			errors = append(errors, fmt.Errorf("cluster %s: failed to list resources: %w", targetCluster, err))
			continue
		}

		// Delete resources created by this sync
		for i := range list.Items {
			resource := &list.Items[i]
			annotations := resource.GetAnnotations()
			if annotations != nil && annotations[SourceClusterAnnotation] == sync.SourceCluster {
				if err := dynamicClient.Resource(gvr).Namespace(sync.Namespace).Delete(ctx, resource.GetName(), metav1.DeleteOptions{}); err != nil {
					s.log.Error(err, "Failed to delete synced resource", "resource", resource.GetName(), "cluster", targetCluster)
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}
	return nil
}

// parseResourceType parses a resource type string into GVR
func (s *Synchronizer) parseResourceType(resourceType string) (schema.GroupVersionResource, error) {
	// Simple parsing - in production, use more robust parsing
	parts := strings.Split(resourceType, "/")
	if len(parts) == 1 {
		// Core resource (e.g., "configmaps")
		return schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: parts[0],
		}, nil
	} else if len(parts) == 3 {
		// Full GVR (e.g., "apps/v1/deployments")
		return schema.GroupVersionResource{
			Group:    parts[0],
			Version:  parts[1],
			Resource: parts[2],
		}, nil
	}
	
	return schema.GroupVersionResource{}, fmt.Errorf("invalid resource type format: %s", resourceType)
}

// updateSyncStatus updates the status of a sync
func (s *Synchronizer) updateSyncStatus(syncName string, status multicluster.SyncStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sync, exists := s.syncs[syncName]; exists {
		sync.Status = status
	}
}

// updateLastSyncTime updates the last sync time
func (s *Synchronizer) updateLastSyncTime(syncName string, time time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sync, exists := s.syncs[syncName]; exists {
		sync.LastSyncTime = time
	}
}

// handleSyncError handles sync errors and updates status
func (s *Synchronizer) handleSyncError(syncName string, err error) error {
	s.updateSyncStatus(syncName, multicluster.SyncStatus{
		Phase:     "Failed",
		Message:   "Sync failed",
		LastError: err.Error(),
	})
	return err
}

// DeepCopy creates a deep copy of ResourceSync
func (rs *multicluster.ResourceSync) DeepCopy() *multicluster.ResourceSync {
	// Simple deep copy - in production, use proper deep copy generation
	data, _ := json.Marshal(rs)
	var copy multicluster.ResourceSync
	json.Unmarshal(data, &copy)
	return &copy
}
