/*
Copyright 2025.

Licensed under the MIT License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
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

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/managers"
	"github.com/gunjanjp/gunj-operator/internal/gitops"
	"github.com/gunjanjp/gunj-operator/internal/metrics"
)

const (
	// FinalizerName is the name of the finalizer used by this controller
	FinalizerName = "observabilityplatform.observability.io/finalizer"

	// RequeueAfterSuccess is the duration after which a successful reconciliation will be requeued
	RequeueAfterSuccess = 5 * time.Minute

	// RequeueAfterError is the duration after which a failed reconciliation will be requeued
	RequeueAfterError = 30 * time.Second
)

// ObservabilityPlatformReconciler reconciles a ObservabilityPlatform object
type ObservabilityPlatformReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	Recorder   record.EventRecorder
	Log        logr.Logger

	// Component managers
	PrometheusManager managers.PrometheusManager
	GrafanaManager    managers.GrafanaManager
	LokiManager       managers.LokiManager
	TempoManager      managers.TempoManager

	// GitOps manager
	GitOpsManager *gitops.Manager

	// Metrics collector
	Metrics *metrics.Collector

	// Status management
	StatusManager *StatusManager
	EventRecorder *EnhancedEventRecorder

	// Finalizer management
	FinalizerManager *FinalizerManager

	// Health management
	HealthCheckManager *HealthCheckManager
	HealthServer       *HealthServer

	// Cost optimization
	CostManager managers.CostManager

	// Configuration
	MaxConcurrentReconciles int
	RequeueDuration         time.Duration
}

// +kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services;serviceaccounts;configmaps;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses;networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheuses;alertmanagers;servicemonitors;podmonitors;prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ObservabilityPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("observabilityplatform", req.NamespacedName)
	startTime := time.Now()

	// Record reconciliation metrics
	defer func() {
		duration := time.Since(startTime)
		r.Metrics.RecordReconciliation("observabilityplatform", duration.Seconds())
		// Update metrics in status
		r.StatusManager.metricsCollector.reconciliationDuration = duration
		r.StatusManager.metricsCollector.lastReconcileTime = time.Now()
	}()

	log.V(1).Info("Starting reconciliation")

	// Fetch the ObservabilityPlatform instance
	platform := &observabilityv1beta1.ObservabilityPlatform{}
	if err := r.Get(ctx, req.NamespacedName, platform); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request
			log.Info("ObservabilityPlatform resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		log.Error(err, "Failed to get ObservabilityPlatform")
		r.Metrics.RecordReconciliationError("observabilityplatform")
		return ctrl.Result{RequeueAfter: RequeueAfterError}, err
	}

	// Check if the platform is being deleted
	if !platform.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, platform)
	}

	// Add finalizers using FinalizerManager
	if err := r.FinalizerManager.AddFinalizers(ctx, platform); err != nil {
		log.Error(err, "Failed to add finalizers")
		return ctrl.Result{RequeueAfter: RequeueAfterError}, err
	}

	// Check if we need to requeue after adding finalizers
	if !controllerutil.ContainsFinalizer(platform, FinalizerName) {
		// Finalizers were just added, requeue to ensure we have the updated object
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if reconciliation is paused
	if platform.Spec.Paused {
		log.Info("Reconciliation is paused")
		r.StatusManager.SetCondition(ctx, platform, ConditionProgressing, metav1.ConditionFalse, "Paused", "Reconciliation paused by user")
		r.StatusManager.CalculateAndSetPhase(ctx, platform)
		return ctrl.Result{RequeueAfter: time.Hour}, nil
	}

	// Main reconciliation logic
	return r.reconcilePlatform(ctx, platform)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObservabilityPlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Set defaults if not provided
	if r.MaxConcurrentReconciles == 0 {
		r.MaxConcurrentReconciles = 3
	}
	if r.RequeueDuration == 0 {
		r.RequeueDuration = RequeueAfterSuccess
	}

	// Initialize logger
	r.Log = ctrl.Log.WithName("controllers").WithName("ObservabilityPlatform")

	// Initialize event recorder
	r.Recorder = mgr.GetEventRecorderFor("observabilityplatform-controller")

	// Initialize enhanced event recorder
	r.EventRecorder = NewEnhancedEventRecorder(r.Recorder, "observabilityplatform-controller")

	// Initialize status manager
	r.StatusManager = NewStatusManager(r.Client, r.Log, r.EventRecorder)

	// Initialize finalizer manager
	r.FinalizerManager = NewFinalizerManager(r.Client, r.Log)

	// Initialize metrics collector
	if r.Metrics == nil {
		r.Metrics = metrics.NewCollector()
	}

	// Initialize component managers with Helm support if not already set
	if r.PrometheusManager == nil || r.GrafanaManager == nil || r.LokiManager == nil || r.TempoManager == nil {
		// Create manager factory with REST config for Helm support
		managerFactory := managers.NewDefaultManagerFactoryWithConfig(r.Client, r.Scheme, r.RestConfig)
		
		if r.PrometheusManager == nil {
			r.PrometheusManager = managerFactory.CreatePrometheusManager()
		}
		if r.GrafanaManager == nil {
			r.GrafanaManager = managerFactory.CreateGrafanaManager()
		}
		if r.LokiManager == nil {
			r.LokiManager = managerFactory.CreateLokiManager()
		}
		if r.TempoManager == nil {
			r.TempoManager = managerFactory.CreateTempoManager()
		}
	}

	// Initialize GitOps manager
	if r.GitOpsManager == nil {
		r.GitOpsManager = gitops.NewManager(r.Client, r.Scheme, ctrl.Log.WithName("gitops-manager"))
	}

	// Initialize Cost Manager
	if r.CostManager == nil {
		r.CostManager = managerFactory.CreateCostManager()
	}

	// Initialize health check manager
	if r.HealthCheckManager == nil {
		r.HealthCheckManager = NewHealthCheckManager(r.Client)
	}

	// Initialize and start health server
	if r.HealthServer == nil {
		r.HealthServer = NewHealthServer("8081", r.HealthCheckManager)
		go func() {
			if err := r.HealthServer.Start(context.Background()); err != nil {
				r.Log.Error(err, "Failed to start health server")
			}
		}()
		// Mark as ready after setup
		r.HealthServer.SetReady(true)
	}

	// Build the controller
	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1beta1.ObservabilityPlatform{}, builder.WithPredicates(
			predicate.GenerationChangedPredicate{},
		)).
		// Watch owned resources
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		// Set controller options
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.MaxConcurrentReconciles,
			LogConstructor: func(req *reconcile.Request) logr.Logger {
				return r.Log.WithValues("observabilityplatform", req.NamespacedName)
			},
		}).
		// Add custom watches for component-specific resources
		Watches(
			&source.Kind{Type: &corev1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.findPlatformsForNamespace),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

// reconcilePlatform performs the main reconciliation logic
func (r *ObservabilityPlatformReconciler) reconcilePlatform(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	startTime := time.Now()

	// Clear any previous errors
	r.StatusManager.ClearError(ctx, platform)

	// Set progressing condition
	r.StatusManager.SetCondition(ctx, platform, ConditionProgressing, metav1.ConditionTrue, ReasonReconciling, "Reconciling platform components")

	// Record reconciliation start
	r.EventRecorder.RecordPlatformEvent(platform, EventReasonPlatformUpdated, "Starting reconciliation")

	// Reconcile namespace
	if err := r.ensureNamespace(ctx, platform); err != nil {
		return r.handleError(ctx, platform, err, "Failed to ensure namespace")
	}

	// Reconcile common resources (RBAC, NetworkPolicies, etc.)
	if err := r.reconcileCommonResources(ctx, platform); err != nil {
		return r.handleError(ctx, platform, err, "Failed to reconcile common resources")
	}

	// Reconcile components with dependency management
	if err := r.ReconcileWithDependencies(ctx, platform); err != nil {
		return r.handleError(ctx, platform, err, "Failed to reconcile components")
	}

	// Reconcile GitOps if configured
	if platform.Spec.GitOps != nil {
		if err := r.reconcileGitOps(ctx, platform); err != nil {
			return r.handleError(ctx, platform, err, "Failed to reconcile GitOps")
		}
	}

	// Reconcile cost optimization if configured
	if platform.Spec.CostOptimization != nil && platform.Spec.CostOptimization.Enabled {
		if err := r.reconcileCostOptimization(ctx, platform); err != nil {
			// Don't fail reconciliation on cost optimization errors
			log.Error(err, "Failed to reconcile cost optimization")
			r.EventRecorder.RecordPlatformEvent(platform, "CostOptimizationError", err.Error())
		}
	}

	// Perform health checks on all components
	healthCheckStart := time.Now()
	healthStatus, err := r.HealthCheckManager.CheckComponentHealth(ctx, platform)
	if err != nil {
		log.Error(err, "Failed to check component health")
		RecordHealthCheckError(platform.Name, platform.Namespace, "all", "check_failed")
	} else {
		// Update health metrics
		RecordHealthCheckDuration(platform.Name, platform.Namespace, time.Since(healthCheckStart).Seconds())
		componentHealthMap := make(map[string]*ComponentHealth)
		for name, status := range healthStatus.Components {
			componentHealthMap[name] = &ComponentHealth{
				Name:              name,
				Healthy:           status.Healthy,
				LastChecked:       status.LastChecked.Time,
				Message:           status.Message,
				AvailableReplicas: status.AvailableReplicas,
				DesiredReplicas:   status.DesiredReplicas,
			}
		}
		UpdateHealthMetrics(platform.Name, platform.Namespace, componentHealthMap)
		
		// Update platform status with health information
		platform.Status.Health = *healthStatus
	}

	// Update health server timestamp
	r.HealthServer.UpdateLastHealthCheck()

	// All components reconciled successfully
	log.Info("All components reconciled successfully with health status", "healthy", healthStatus.Healthy)

	// Complete the operation
	duration := time.Since(startTime)
	r.StatusManager.CompleteOperation(ctx, platform, "reconciliation", true, "All components reconciled successfully", duration)

	// Update conditions
	r.StatusManager.SetCondition(ctx, platform, ConditionProgressing, metav1.ConditionFalse, ReasonReady, "Reconciliation complete")
	r.StatusManager.AggregateComponentStatuses(ctx, platform)
	r.StatusManager.CalculateAndSetPhase(ctx, platform)
	r.StatusManager.UpdateMetrics(ctx, platform)

	// Record successful reconciliation
	r.EventRecorder.RecordPlatformEvent(platform, EventReasonPlatformReady, "Platform is ready")
	r.Metrics.RecordPlatformStatus(platform.Name, platform.Namespace, string(platform.Status.Phase))

	// Requeue after success duration for continuous reconciliation
	return ctrl.Result{RequeueAfter: r.RequeueDuration}, nil
}

// handleDeletion handles the deletion of the ObservabilityPlatform
func (r *ObservabilityPlatformReconciler) handleDeletion(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling platform deletion")

	// Use FinalizerManager to handle deletion
	if err := r.FinalizerManager.HandleDeletion(ctx, platform, r); err != nil {
		log.Error(err, "Failed to handle deletion")
		r.Recorder.Event(platform, corev1.EventTypeWarning, "DeletionFailed", err.Error())
		return ctrl.Result{RequeueAfter: RequeueAfterError}, err
	}

	return ctrl.Result{}, nil
}

// cleanup performs cleanup operations (deprecated - use FinalizerManager)
func (r *ObservabilityPlatformReconciler) cleanup(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	// This method is deprecated in favor of FinalizerManager.HandleDeletion
	// Kept for backward compatibility
	return nil
}

// ensureNamespace ensures the namespace exists with proper labels
func (r *ObservabilityPlatformReconciler) ensureNamespace(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)

	namespace := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: platform.Namespace}, namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			// Namespace doesn't exist, create it
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: platform.Namespace,
					Labels: map[string]string{
						"observability.io/managed":      "true",
						"observability.io/platform":     platform.Name,
						"app.kubernetes.io/managed-by":  "gunj-operator",
						"app.kubernetes.io/part-of":     "observability-platform",
						"app.kubernetes.io/instance":    platform.Name,
					},
				},
			}
			if err := r.Create(ctx, namespace); err != nil {
				return fmt.Errorf("failed to create namespace: %w", err)
			}
			log.Info("Created namespace", "namespace", namespace.Name)
		} else {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}
	return nil
}

// reconcileCommonResources creates common resources like RBAC, NetworkPolicies
func (r *ObservabilityPlatformReconciler) reconcileCommonResources(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	// This will be implemented in the next micro-task
	return nil
}

// updateComponentStatus updates the status for a specific component
func (r *ObservabilityPlatformReconciler) updateComponentStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, component string, ready bool, message string) error {
	return r.StatusManager.SetComponentStatus(ctx, platform, component, ready, message)
}

// handleError handles errors during reconciliation
func (r *ObservabilityPlatformReconciler) handleError(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, err error, message string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Error(err, message)

	// Record error in status
	r.StatusManager.RecordError(ctx, platform, err, message)

	// Update phase
	r.StatusManager.CalculateAndSetPhase(ctx, platform)

	// Update metrics
	r.StatusManager.metricsCollector.errorCount++
	r.StatusManager.UpdateMetrics(ctx, platform)

	// Record error metric
	r.Metrics.RecordReconciliationError("observabilityplatform")

	return ctrl.Result{RequeueAfter: RequeueAfterError}, err
}

// findPlatformsForNamespace returns requests for all platforms that might be affected by a namespace change
func (r *ObservabilityPlatformReconciler) findPlatformsForNamespace(obj client.Object) []reconcile.Request {
	namespace := obj.(*corev1.Namespace)
	
	// Check if this namespace has our label
	if value, ok := namespace.Labels["observability.io/platform"]; ok {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      value,
					Namespace: namespace.Name,
				},
			},
		}
	}
	
	return nil
}

// reconcileGitOps handles GitOps configuration for the platform
func (r *ObservabilityPlatformReconciler) reconcileGitOps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("gitops", "reconcile")
	log.Info("Reconciling GitOps configuration")

	// Reconcile GitOps
	if err := r.GitOpsManager.ReconcileGitOps(ctx, platform); err != nil {
		return fmt.Errorf("failed to reconcile GitOps: %w", err)
	}

	// Check for drift if enabled
	if platform.Spec.GitOps.DriftDetection != nil && platform.Spec.GitOps.DriftDetection.Enabled {
		if err := r.GitOpsManager.DetectAndRemediateDrift(ctx, platform); err != nil {
			log.Error(err, "Failed to detect/remediate drift")
			// Don't fail reconciliation on drift detection errors
		}
	}

	// Check for automatic rollback conditions
	if platform.Spec.GitOps.Rollback != nil && platform.Spec.GitOps.Rollback.Enabled {
		if err := r.GitOpsManager.RollbackMgr.CheckHealthAndRollback(ctx, platform); err != nil {
			log.Error(err, "Failed to check rollback conditions")
			// Don't fail reconciliation on rollback check errors
		}
	}

	// Update status conditions based on GitOps status
	if platform.Status.GitOps != nil {
		synced := platform.Status.GitOps.SyncStatus == "Synced"
		r.StatusManager.SetCondition(ctx, platform, "GitOpsSynced",
			metav1.ConditionStatus(fmt.Sprintf("%t", synced)),
			platform.Status.GitOps.SyncStatus,
			platform.Status.GitOps.Message)
		
		if platform.Status.GitOps.DriftDetected {
			r.StatusManager.SetCondition(ctx, platform, "GitOpsDrift",
				metav1.ConditionTrue,
				"DriftDetected",
				fmt.Sprintf("%d drift items detected", platform.Status.GitOps.DriftItems))
		}
	}

	log.Info("GitOps reconciliation complete")
	return nil
}

// PromoteEnvironment handles environment promotion
func (r *ObservabilityPlatformReconciler) PromoteEnvironment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, targetEnv string) error {
	log := log.FromContext(ctx).WithValues("targetEnv", targetEnv)
	log.Info("Promoting environment")

	if platform.Spec.GitOps == nil {
		return fmt.Errorf("GitOps not configured for platform")
	}

	// Perform promotion
	if err := r.GitOpsManager.PromoteEnvironment(ctx, platform, targetEnv); err != nil {
		return fmt.Errorf("failed to promote environment: %w", err)
	}

	// Record event
	r.EventRecorder.RecordPlatformEvent(platform, "EnvironmentPromoted",
		fmt.Sprintf("Promoted configuration to %s", targetEnv))

	return nil
}

// RollbackPlatform handles platform rollback
func (r *ObservabilityPlatformReconciler) RollbackPlatform(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, reason string) error {
	log := log.FromContext(ctx).WithValues("reason", reason)
	log.Info("Rolling back platform")

	if platform.Spec.GitOps == nil {
		return fmt.Errorf("GitOps not configured for platform")
	}

	// Perform rollback
	if err := r.GitOpsManager.RollbackDeployment(ctx, platform, reason); err != nil {
		return fmt.Errorf("failed to rollback: %w", err)
	}

	// Record event
	r.EventRecorder.RecordPlatformEvent(platform, "PlatformRolledBack",
		fmt.Sprintf("Rolled back due to: %s", reason))

	return nil
}

// SyncGitOps manually triggers a GitOps sync
func (r *ObservabilityPlatformReconciler) SyncGitOps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	log.Info("Manually syncing GitOps")

	if platform.Spec.GitOps == nil {
		return fmt.Errorf("GitOps not configured for platform")
	}

	// Trigger sync
	if err := r.GitOpsManager.SyncManager.TriggerSync(ctx, platform); err != nil {
		return fmt.Errorf("failed to sync with Git: %w", err)
	}

	// Record event
	r.EventRecorder.RecordPlatformEvent(platform, "GitOpsSyncTriggered",
		"Manual GitOps sync triggered")

	return nil
}

// reconcileCostOptimization handles cost optimization for the platform
func (r *ObservabilityPlatformReconciler) reconcileCostOptimization(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx).WithValues("costOptimization", "reconcile")
	log.Info("Reconciling cost optimization")

	// Analyze current costs
	costAnalysis, err := r.CostManager.AnalyzePlatformCosts(ctx, platform)
	if err != nil {
		return fmt.Errorf("failed to analyze platform costs: %w", err)
	}

	// Update status with current cost information
	platform.Status.CostOptimization = &observabilityv1beta1.CostOptimizationStatus{
		LastOptimization: metav1.Now(),
		CurrentCost:      costAnalysis.TotalCost,
	}

	// Get optimization recommendations
	recommendations, err := r.CostManager.GetRecommendations(ctx, platform)
	if err != nil {
		return fmt.Errorf("failed to get cost recommendations: %w", err)
	}

	// Update status with recommendations
	platform.Status.CostOptimization.RecommendationsCount = len(recommendations.ResourceRecommendations) +
		len(recommendations.SpotInstanceCandidates) +
		len(recommendations.ScalingPolicies) +
		len(recommendations.StorageOptimizations)
	platform.Status.CostOptimization.EstimatedSavings = recommendations.TotalSavings
	platform.Status.CostOptimization.OptimizedCost = costAnalysis.TotalCost - recommendations.TotalSavings

	// Apply optimizations if configured
	if platform.Spec.CostOptimization.ResourceOptimization || 
	   (platform.Spec.CostOptimization.SpotInstances != nil && platform.Spec.CostOptimization.SpotInstances.Enabled) ||
	   platform.Spec.CostOptimization.AutoScaling {
		
		opts := &managers.OptimizationOptions{
			ApplyResourceOptimization: platform.Spec.CostOptimization.ResourceOptimization,
			EnableSpotInstances:       platform.Spec.CostOptimization.SpotInstances != nil && platform.Spec.CostOptimization.SpotInstances.Enabled,
			ApplyScalingPolicies:      platform.Spec.CostOptimization.AutoScaling,
		}

		if err := r.CostManager.ApplyOptimizations(ctx, platform, opts); err != nil {
			return fmt.Errorf("failed to apply cost optimizations: %w", err)
		}

		platform.Status.CostOptimization.Applied = true
	}

	// Set up budget alerts if configured
	if platform.Spec.CostOptimization.Budget != nil {
		budgetConfig := &managers.BudgetConfig{
			MonthlyLimit:   platform.Spec.CostOptimization.Budget.MonthlyLimit,
			AlertThreshold: float64(platform.Spec.CostOptimization.Budget.AlertThresholds[0]), // Use first threshold
		}

		if err := r.CostManager.SetBudgetAlert(ctx, platform, budgetConfig); err != nil {
			log.Error(err, "Failed to set budget alert")
			// Don't fail reconciliation
		}
	}

	// Update conditions
	r.StatusManager.SetCondition(ctx, platform, "CostOptimized",
		metav1.ConditionTrue, "Optimized", 
		fmt.Sprintf("Estimated savings: $%.2f/month", recommendations.TotalSavings))

	// Record event
	r.EventRecorder.RecordPlatformEvent(platform, "CostOptimizationComplete",
		fmt.Sprintf("Cost optimization complete. Current: $%.2f, Potential savings: $%.2f", 
			costAnalysis.TotalCost, recommendations.TotalSavings))

	log.Info("Cost optimization reconciliation complete", 
		"currentCost", costAnalysis.TotalCost,
		"potentialSavings", recommendations.TotalSavings)

	return nil
}
