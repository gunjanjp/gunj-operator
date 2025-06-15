package federation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/multicluster"
)

const (
	// FederationFinalizerName is the finalizer added to federated resources
	FederationFinalizerName = "federation.gunj.io/finalizer"
	// FederationAnnotationPrefix for federation-related annotations
	FederationAnnotationPrefix = "federation.gunj.io/"
)

// FederationController manages multi-cluster federation
type FederationController struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Log      logr.Logger

	// Registry for cluster management
	Registry multicluster.ClusterRegistry
	// Synchronizer for resource sync
	Synchronizer multicluster.ResourceSynchronizer
	// HealthChecker for cluster health
	HealthChecker multicluster.ClusterHealthChecker
	// NetworkManager for cross-cluster networking
	NetworkManager multicluster.NetworkManager
	// FailoverManager for failover scenarios
	FailoverManager multicluster.FailoverManager

	// Configuration
	Config *multicluster.FederationConfig

	// State management
	mu              sync.RWMutex
	clusterWatchers map[string]context.CancelFunc
	syncWorkers     map[string]context.CancelFunc
}

// NewFederationController creates a new federation controller
func NewFederationController(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	registry multicluster.ClusterRegistry,
	config *multicluster.FederationConfig,
) *FederationController {
	return &FederationController{
		Client:          client,
		Scheme:          scheme,
		Recorder:        recorder,
		Log:             log.Log.WithName("federation-controller"),
		Registry:        registry,
		Config:          config,
		clusterWatchers: make(map[string]context.CancelFunc),
		syncWorkers:     make(map[string]context.CancelFunc),
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *FederationController) SetupWithManager(mgr ctrl.Manager) error {
	// Create controller
	c, err := controller.New("federation-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch ObservabilityPlatform resources
	err = c.Watch(
		&source.Kind{Type: &observabilityv1.ObservabilityPlatform{}},
		&handler.EnqueueRequestForObject{},
		predicate.Or(
			predicate.GenerationChangedPredicate{},
			predicate.AnnotationChangedPredicate{},
		),
	)
	if err != nil {
		return err
	}

	// Watch ConfigMaps for cluster registry changes
	err = c.Watch(
		&source.Kind{Type: &corev1.ConfigMap{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj client.Object) []reconcile.Request {
				if obj.GetName() == "cluster-registry" {
					return []reconcile.Request{{
						NamespacedName: types.NamespacedName{
							Name:      "federation-sync",
							Namespace: obj.GetNamespace(),
						},
					}}
				}
				return nil
			}),
		},
	)
	if err != nil {
		return err
	}

	// Start background tasks
	go r.startClusterHealthMonitoring(mgr.GetEventRecorderFor("health-monitor"))
	go r.startResourceSynchronization(mgr.GetEventRecorderFor("resource-sync"))

	return nil
}

// Reconcile handles federation reconciliation
func (r *FederationController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("federation", req.NamespacedName)

	// Special handling for federation sync requests
	if req.Name == "federation-sync" {
		return r.reconcileFederation(ctx)
	}

	// Get ObservabilityPlatform
	platform := &observabilityv1.ObservabilityPlatform{}
	if err := r.Get(ctx, req.NamespacedName, platform); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if platform should be federated
	if !r.shouldFederate(platform) {
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if !platform.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, platform)
	}

	// Ensure finalizer
	if !controllerutil.ContainsFinalizer(platform, FederationFinalizerName) {
		controllerutil.AddFinalizer(platform, FederationFinalizerName)
		if err := r.Update(ctx, platform); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile federated platform
	if err := r.reconcilePlatform(ctx, platform); err != nil {
		log.Error(err, "Failed to reconcile federated platform")
		r.Recorder.Event(platform, corev1.EventTypeWarning, "FederationFailed", err.Error())
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	r.Recorder.Event(platform, corev1.EventTypeNormal, "FederationSynced", "Platform synchronized across clusters")
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// reconcileFederation handles federation-wide reconciliation
func (r *FederationController) reconcileFederation(ctx context.Context) (ctrl.Result, error) {
	log := r.Log.WithName("federation-sync")
	log.Info("Reconciling federation configuration")

	// List all clusters
	clusters, err := r.Registry.List(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list clusters: %w", err)
	}

	// Update cluster watchers
	r.updateClusterWatchers(ctx, clusters)

	// Check hub cluster
	hubFound := false
	for _, cluster := range clusters {
		if cluster.Role == multicluster.ClusterRoleHub {
			hubFound = true
			break
		}
	}

	if !hubFound {
		log.Error(nil, "No hub cluster found in federation")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Update network mesh if configured
	if r.Config.NetworkMesh != nil && r.Config.NetworkMesh.Enabled && r.NetworkManager != nil {
		if err := r.NetworkManager.SetupNetworkMesh(ctx, r.Config.NetworkMesh); err != nil {
			log.Error(err, "Failed to setup network mesh")
		}
	}

	return ctrl.Result{RequeueAfter: r.Config.SyncInterval}, nil
}

// reconcilePlatform reconciles a federated platform
func (r *FederationController) reconcilePlatform(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := r.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)

	// Get target clusters from annotations
	targetClusters := r.getTargetClusters(platform)
	if len(targetClusters) == 0 {
		log.Info("No target clusters specified, skipping federation")
		return nil
	}

	// Sync to each target cluster
	var errors []error
	for _, clusterName := range targetClusters {
		if err := r.syncPlatformToCluster(ctx, platform, clusterName); err != nil {
			log.Error(err, "Failed to sync platform to cluster", "cluster", clusterName)
			errors = append(errors, fmt.Errorf("cluster %s: %w", clusterName, err))
		}
	}

	// Update status
	if err := r.updateFederationStatus(ctx, platform, targetClusters, errors); err != nil {
		log.Error(err, "Failed to update federation status")
	}

	if len(errors) > 0 {
		return fmt.Errorf("federation errors: %v", errors)
	}

	return nil
}

// syncPlatformToCluster syncs a platform to a specific cluster
func (r *FederationController) syncPlatformToCluster(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, clusterName string) error {
	// Get cluster connection
	conn, err := r.Registry.GetConnection(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster connection: %w", err)
	}

	// Create platform copy for remote cluster
	remotePlatform := platform.DeepCopy()
	remotePlatform.ResourceVersion = ""
	remotePlatform.UID = ""
	
	// Add federation annotations
	if remotePlatform.Annotations == nil {
		remotePlatform.Annotations = make(map[string]string)
	}
	remotePlatform.Annotations[FederationAnnotationPrefix+"source-cluster"] = r.Config.HubCluster
	remotePlatform.Annotations[FederationAnnotationPrefix+"source-namespace"] = platform.Namespace
	remotePlatform.Annotations[FederationAnnotationPrefix+"source-name"] = platform.Name
	remotePlatform.Annotations[FederationAnnotationPrefix+"sync-time"] = time.Now().UTC().Format(time.RFC3339)

	// Apply cluster-specific transformations
	if err := r.applyClusterTransformations(remotePlatform, clusterName); err != nil {
		return fmt.Errorf("failed to apply transformations: %w", err)
	}

	// Check if platform exists in remote cluster
	existing := &observabilityv1.ObservabilityPlatform{}
	err = conn.RuntimeClient.Get(ctx, client.ObjectKeyFromObject(remotePlatform), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create platform in remote cluster
			if err := conn.RuntimeClient.Create(ctx, remotePlatform); err != nil {
				return fmt.Errorf("failed to create platform: %w", err)
			}
			r.Log.Info("Created platform in remote cluster", "cluster", clusterName, "platform", platform.Name)
			return nil
		}
		return fmt.Errorf("failed to get existing platform: %w", err)
	}

	// Update existing platform
	existing.Spec = remotePlatform.Spec
	existing.Labels = remotePlatform.Labels
	existing.Annotations = remotePlatform.Annotations

	if err := conn.RuntimeClient.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update platform: %w", err)
	}

	r.Log.Info("Updated platform in remote cluster", "cluster", clusterName, "platform", platform.Name)
	return nil
}

// handleDeletion handles platform deletion from federated clusters
func (r *FederationController) handleDeletion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := r.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)

	// Get target clusters
	targetClusters := r.getTargetClusters(platform)
	
	// Delete from each target cluster
	for _, clusterName := range targetClusters {
		if err := r.deletePlatformFromCluster(ctx, platform, clusterName); err != nil {
			log.Error(err, "Failed to delete platform from cluster", "cluster", clusterName)
			// Continue with other clusters
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(platform, FederationFinalizerName)
	if err := r.Update(ctx, platform); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// deletePlatformFromCluster deletes a platform from a specific cluster
func (r *FederationController) deletePlatformFromCluster(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, clusterName string) error {
	conn, err := r.Registry.GetConnection(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster connection: %w", err)
	}

	remotePlatform := &observabilityv1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      platform.Name,
			Namespace: platform.Namespace,
		},
	}

	if err := conn.RuntimeClient.Delete(ctx, remotePlatform); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete platform: %w", err)
	}

	r.Log.Info("Deleted platform from remote cluster", "cluster", clusterName, "platform", platform.Name)
	return nil
}

// shouldFederate checks if a platform should be federated
func (r *FederationController) shouldFederate(platform *observabilityv1.ObservabilityPlatform) bool {
	// Check for federation annotation
	if platform.Annotations == nil {
		return false
	}

	federate, exists := platform.Annotations[FederationAnnotationPrefix+"enabled"]
	return exists && federate == "true"
}

// getTargetClusters gets the list of target clusters for federation
func (r *FederationController) getTargetClusters(platform *observabilityv1.ObservabilityPlatform) []string {
	if platform.Annotations == nil {
		return nil
	}

	// Check for explicit cluster list
	if clusters, exists := platform.Annotations[FederationAnnotationPrefix+"target-clusters"]; exists {
		return splitCommaSeparated(clusters)
	}

	// Check for cluster selector
	if selector, exists := platform.Annotations[FederationAnnotationPrefix+"cluster-selector"]; exists {
		return r.getClustersbySelector(selector)
	}

	return nil
}

// getClustersbySelector returns clusters matching a label selector
func (r *FederationController) getClustersbySelector(selector string) []string {
	// Parse selector and match against cluster labels
	// This is simplified - in production, use proper label selector parsing
	clusters, err := r.Registry.List(context.Background())
	if err != nil {
		return nil
	}

	var matched []string
	for _, cluster := range clusters {
		// Skip hub cluster
		if cluster.Role == multicluster.ClusterRoleHub {
			continue
		}
		// Simple matching - in production, use proper selector
		matched = append(matched, cluster.Name)
	}

	return matched
}

// applyClusterTransformations applies cluster-specific transformations
func (r *FederationController) applyClusterTransformations(platform *observabilityv1.ObservabilityPlatform, clusterName string) error {
	// Get cluster info
	cluster, err := r.Registry.Get(context.Background(), clusterName)
	if err != nil {
		return err
	}

	// Apply region/zone labels
	if platform.Labels == nil {
		platform.Labels = make(map[string]string)
	}
	platform.Labels["federation.gunj.io/cluster"] = clusterName
	platform.Labels["federation.gunj.io/region"] = cluster.Region
	platform.Labels["federation.gunj.io/zone"] = cluster.Zone

	// Adjust resources based on cluster capacity
	if cluster.Metrics != nil {
		// This is a simplified example - implement proper resource adjustment
		if cluster.Metrics.CPUCapacity < 16 {
			// Reduce resource requests for smaller clusters
			r.adjustResourceRequests(platform, 0.5)
		}
	}

	return nil
}

// adjustResourceRequests adjusts resource requests by a factor
func (r *FederationController) adjustResourceRequests(platform *observabilityv1.ObservabilityPlatform, factor float64) {
	// Adjust Prometheus resources
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Resources.Requests != nil {
		cpu := platform.Spec.Components.Prometheus.Resources.Requests.Cpu()
		if cpu != nil {
			scaled := int64(float64(cpu.MilliValue()) * factor)
			platform.Spec.Components.Prometheus.Resources.Requests[corev1.ResourceCPU] = *resource.NewMilliQuantity(scaled, resource.DecimalSI)
		}
	}
	// Similar adjustments for other components...
}

// updateFederationStatus updates the federation status annotations
func (r *FederationController) updateFederationStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetClusters []string, errors []error) error {
	if platform.Annotations == nil {
		platform.Annotations = make(map[string]string)
	}

	platform.Annotations[FederationAnnotationPrefix+"synced-clusters"] = joinCommaSeparated(targetClusters)
	platform.Annotations[FederationAnnotationPrefix+"sync-time"] = time.Now().UTC().Format(time.RFC3339)
	
	if len(errors) > 0 {
		platform.Annotations[FederationAnnotationPrefix+"sync-status"] = "partial"
		platform.Annotations[FederationAnnotationPrefix+"sync-errors"] = fmt.Sprintf("%d errors", len(errors))
	} else {
		platform.Annotations[FederationAnnotationPrefix+"sync-status"] = "success"
		delete(platform.Annotations, FederationAnnotationPrefix+"sync-errors")
	}

	return r.Update(ctx, platform)
}

// startClusterHealthMonitoring starts background health monitoring
func (r *FederationController) startClusterHealthMonitoring(recorder record.EventRecorder) {
	ticker := time.NewTicker(r.Config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		clusters, err := r.Registry.List(ctx)
		if err != nil {
			r.Log.Error(err, "Failed to list clusters for health check")
			continue
		}

		for _, cluster := range clusters {
			if r.HealthChecker != nil {
				health, err := r.HealthChecker.CheckHealth(ctx, cluster.Name)
				if err != nil {
					r.Log.Error(err, "Failed to check cluster health", "cluster", cluster.Name)
					continue
				}

				// Update cluster status based on health
				if health.Status != cluster.Status {
					if err := r.Registry.UpdateStatus(ctx, cluster.Name, health.Status); err != nil {
						r.Log.Error(err, "Failed to update cluster status", "cluster", cluster.Name)
					}
				}

				// Record events for status changes
				if health.Status != multicluster.ClusterStatusReady {
					recorder.Eventf(&corev1.ObjectReference{
						Kind:      "Cluster",
						Name:      cluster.Name,
						Namespace: "gunj-multicluster",
					}, corev1.EventTypeWarning, "ClusterUnhealthy", "Cluster %s is %s", cluster.Name, health.Status)
				}
			}
		}
	}
}

// startResourceSynchronization starts background resource synchronization
func (r *FederationController) startResourceSynchronization(recorder record.EventRecorder) {
	ticker := time.NewTicker(r.Config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		
		if r.Synchronizer != nil {
			syncs, err := r.Synchronizer.ListSyncs(ctx)
			if err != nil {
				r.Log.Error(err, "Failed to list resource syncs")
				continue
			}

			for _, sync := range syncs {
				if err := r.Synchronizer.SyncNow(ctx, sync.Name); err != nil {
					r.Log.Error(err, "Failed to sync resources", "sync", sync.Name)
					recorder.Eventf(&corev1.ObjectReference{
						Kind:      "ResourceSync",
						Name:      sync.Name,
						Namespace: "gunj-multicluster",
					}, corev1.EventTypeWarning, "SyncFailed", "Resource sync %s failed: %v", sync.Name, err)
				}
			}
		}
	}
}

// updateClusterWatchers updates watchers for cluster changes
func (r *FederationController) updateClusterWatchers(ctx context.Context, clusters []*multicluster.Cluster) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create set of current clusters
	currentClusters := make(map[string]bool)
	for _, cluster := range clusters {
		currentClusters[cluster.Name] = true
	}

	// Stop watchers for removed clusters
	for clusterName, cancel := range r.clusterWatchers {
		if !currentClusters[clusterName] {
			cancel()
			delete(r.clusterWatchers, clusterName)
			r.Log.Info("Stopped watching cluster", "cluster", clusterName)
		}
	}

	// Start watchers for new clusters
	for _, cluster := range clusters {
		if _, exists := r.clusterWatchers[cluster.Name]; !exists {
			ctx, cancel := context.WithCancel(context.Background())
			r.clusterWatchers[cluster.Name] = cancel
			go r.watchCluster(ctx, cluster)
			r.Log.Info("Started watching cluster", "cluster", cluster.Name)
		}
	}
}

// watchCluster watches for changes in a specific cluster
func (r *FederationController) watchCluster(ctx context.Context, cluster *multicluster.Cluster) {
	// This would implement watching for ObservabilityPlatform resources in the remote cluster
	// and triggering reconciliation when changes are detected
}

// Helper functions

func splitCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func joinCommaSeparated(parts []string) string {
	return strings.Join(parts, ",")
}
