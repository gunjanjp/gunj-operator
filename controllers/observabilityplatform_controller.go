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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
)

const (
	finalizerName = "observability.io/finalizer"
	
	// Platform phases
	PhasePending    = "Pending"
	PhaseInstalling = "Installing"
	PhaseReady      = "Ready"
	PhaseFailed     = "Failed"
	PhaseUpgrading  = "Upgrading"
	
	// Condition types
	ConditionReady       = "Ready"
	ConditionProgressing = "Progressing"
	ConditionDegraded    = "Degraded"
	ConditionAvailable   = "Available"
	
	// Condition reasons
	ReasonValidationFailed = "ValidationFailed"
	ReasonInstalling       = "Installing"
	ReasonReady           = "Ready"
	ReasonFailed          = "Failed"
	ReasonUpgrading       = "Upgrading"
)

// ObservabilityPlatformReconciler reconciles a ObservabilityPlatform object
type ObservabilityPlatformReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	
	// Component managers
	prometheusManager ComponentManager
	grafanaManager    ComponentManager
	lokiManager       ComponentManager
	tempoManager      ComponentManager
}

// ComponentManager interface for managing individual components
type ComponentManager interface {
	Deploy(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error
	GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*observabilityv1.ComponentStatus, error)
	Upgrade(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, fromVersion string) error
	Delete(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error
}

// RetryConfig defines retry behavior for transient errors
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

var defaultRetryConfig = RetryConfig{
	MaxRetries:     5,
	InitialBackoff: 10 * time.Second,
	MaxBackoff:     5 * time.Minute,
	BackoffFactor:  2.0,
}

//+kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services;configmaps;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ObservabilityPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconciliation")

	// Fetch the ObservabilityPlatform instance
	platform := &observabilityv1.ObservabilityPlatform{}
	if err := r.Get(ctx, req.NamespacedName, platform); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted
			log.Info("ObservabilityPlatform resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ObservabilityPlatform")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !platform.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, platform)
	}

	// Add finalizer if needed
	if !controllerutil.ContainsFinalizer(platform, finalizerName) {
		log.Info("Adding finalizer")
		controllerutil.AddFinalizer(platform, finalizerName)
		if err := r.Update(ctx, platform); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if paused
	if platform.Spec.Paused {
		log.Info("Platform is paused, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Initialize status if needed
	if platform.Status.Phase == "" {
		platform.Status.Phase = PhasePending
		platform.Status.ObservedGeneration = platform.Generation
		if err := r.Status().Update(ctx, platform); err != nil {
			log.Error(err, "Failed to update initial status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if spec has changed
	if platform.Status.ObservedGeneration != platform.Generation {
		log.Info("Spec has changed, processing update", 
			"observedGeneration", platform.Status.ObservedGeneration,
			"generation", platform.Generation)
		return r.handleSpecChange(ctx, platform)
	}

	// Handle state transitions
	log.Info("Processing state", "phase", platform.Status.Phase)
	switch platform.Status.Phase {
	case PhasePending:
		return r.handlePendingState(ctx, platform)
	case PhaseInstalling:
		return r.handleInstallingState(ctx, platform)
	case PhaseReady:
		return r.handleReadyState(ctx, platform)
	case PhaseFailed:
		return r.handleFailedState(ctx, platform)
	case PhaseUpgrading:
		return r.handleUpgradingState(ctx, platform)
	default:
		err := fmt.Errorf("unknown phase: %s", platform.Status.Phase)
		log.Error(err, "Invalid platform phase")
		return ctrl.Result{}, err
	}
}

// handlePendingState processes platforms in Pending state
func (r *ObservabilityPlatformReconciler) handlePendingState(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Pending state")

	// Perform validation
	if err := r.validatePlatform(ctx, platform); err != nil {
		log.Error(err, "Platform validation failed")
		return r.transitionToFailed(ctx, platform, ReasonValidationFailed, err.Error())
	}

	// Check prerequisites
	if err := r.checkPrerequisites(ctx, platform); err != nil {
		log.Info("Prerequisites not met, requeueing", "error", err)
		r.updateCondition(platform, ConditionProgressing, metav1.ConditionFalse, 
			ReasonValidationFailed, fmt.Sprintf("Prerequisites check failed: %v", err))
		if err := r.Status().Update(ctx, platform); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Transition to Installing
	log.Info("Validation passed, transitioning to Installing")
	return r.transitionToInstalling(ctx, platform)
}

// handleInstallingState processes platforms in Installing state
func (r *ObservabilityPlatformReconciler) handleInstallingState(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Installing state")

	// Initialize component status map if needed
	if platform.Status.ComponentStatus == nil {
		platform.Status.ComponentStatus = make(map[string]observabilityv1.ComponentStatus)
	}

	// Deploy components in order
	allReady := true
	
	// Deploy Prometheus
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		ready, err := r.deployComponent(ctx, platform, "prometheus", r.prometheusManager)
		if err != nil {
			return r.transitionToFailed(ctx, platform, ReasonFailed, fmt.Sprintf("Failed to deploy Prometheus: %v", err))
		}
		if !ready {
			allReady = false
		}
	}

	// Deploy Loki
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		ready, err := r.deployComponent(ctx, platform, "loki", r.lokiManager)
		if err != nil {
			return r.transitionToFailed(ctx, platform, ReasonFailed, fmt.Sprintf("Failed to deploy Loki: %v", err))
		}
		if !ready {
			allReady = false
		}
	}

	// Deploy Tempo
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		ready, err := r.deployComponent(ctx, platform, "tempo", r.tempoManager)
		if err != nil {
			return r.transitionToFailed(ctx, platform, ReasonFailed, fmt.Sprintf("Failed to deploy Tempo: %v", err))
		}
		if !ready {
			allReady = false
		}
	}

	// Deploy Grafana (depends on other components)
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		ready, err := r.deployComponent(ctx, platform, "grafana", r.grafanaManager)
		if err != nil {
			return r.transitionToFailed(ctx, platform, ReasonFailed, fmt.Sprintf("Failed to deploy Grafana: %v", err))
		}
		if !ready {
			allReady = false
		}
	}

	// Update status
	if err := r.Status().Update(ctx, platform); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Check if all components are ready
	if allReady {
		log.Info("All components ready, transitioning to Ready state")
		return r.transitionToReady(ctx, platform)
	}

	// Still installing, requeue
	log.Info("Components still being deployed, requeueing")
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// handleReadyState processes platforms in Ready state
func (r *ObservabilityPlatformReconciler) handleReadyState(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Ready state")

	// Check component health
	allHealthy := true
	for componentName, manager := range r.getComponentManagers() {
		if !r.isComponentEnabled(platform, componentName) {
			continue
		}

		status, err := manager.GetStatus(ctx, platform)
		if err != nil {
			log.Error(err, "Failed to get component status", "component", componentName)
			allHealthy = false
			continue
		}

		platform.Status.ComponentStatus[componentName] = *status
		if status.Phase != PhaseReady {
			allHealthy = false
		}
	}

	// Update status
	platform.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, platform); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Check if platform is still healthy
	if !allHealthy {
		log.Info("Platform degraded, transitioning to Failed state")
		return r.transitionToFailed(ctx, platform, ReasonFailed, "One or more components are unhealthy")
	}

	// Regular health check interval
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// handleFailedState processes platforms in Failed state
func (r *ObservabilityPlatformReconciler) handleFailedState(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Failed state")

	// Check if we should attempt recovery
	// For now, we'll wait for manual intervention or spec change
	// In the future, we could implement automatic recovery strategies

	// Regular check interval
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// handleUpgradingState processes platforms in Upgrading state
func (r *ObservabilityPlatformReconciler) handleUpgradingState(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Upgrading state")

	// TODO: Implement upgrade logic
	// For now, transition back to Ready
	return r.transitionToReady(ctx, platform)
}

// handleDeletion handles platform deletion
func (r *ObservabilityPlatformReconciler) handleDeletion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling deletion")

	// Delete components in reverse order
	managers := []struct {
		name    string
		manager ComponentManager
	}{
		{"grafana", r.grafanaManager},
		{"tempo", r.tempoManager},
		{"loki", r.lokiManager},
		{"prometheus", r.prometheusManager},
	}

	for _, m := range managers {
		if r.isComponentEnabled(platform, m.name) && m.manager != nil {
			if err := m.manager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete component", "component", m.name)
				// Continue with other components
			}
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(platform, finalizerName)
	if err := r.Update(ctx, platform); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Deletion completed")
	return ctrl.Result{}, nil
}

// handleSpecChange handles changes to the platform spec
func (r *ObservabilityPlatformReconciler) handleSpecChange(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling spec change")

	// Update observed generation
	platform.Status.ObservedGeneration = platform.Generation

	// Determine if this is an upgrade or just a configuration change
	// For now, we'll transition to Upgrading state
	// TODO: Implement logic to differentiate between minor updates and upgrades

	return r.transitionToUpgrading(ctx, platform)
}

// State transition helpers

func (r *ObservabilityPlatformReconciler) transitionToInstalling(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	platform.Status.Phase = PhaseInstalling
	platform.Status.Message = "Installing observability components"
	platform.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	
	r.updateCondition(platform, ConditionProgressing, metav1.ConditionTrue, ReasonInstalling, "Platform installation in progress")
	r.updateCondition(platform, ConditionReady, metav1.ConditionFalse, ReasonInstalling, "Platform is being installed")
	
	if err := r.Status().Update(ctx, platform); err != nil {
		return ctrl.Result{}, err
	}
	
	return ctrl.Result{Requeue: true}, nil
}

func (r *ObservabilityPlatformReconciler) transitionToReady(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	platform.Status.Phase = PhaseReady
	platform.Status.Message = "All components are running"
	platform.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	
	r.updateCondition(platform, ConditionReady, metav1.ConditionTrue, ReasonReady, "Platform is ready")
	r.updateCondition(platform, ConditionProgressing, metav1.ConditionFalse, ReasonReady, "Installation completed")
	r.updateCondition(platform, ConditionAvailable, metav1.ConditionTrue, ReasonReady, "Platform is available")
	
	if err := r.Status().Update(ctx, platform); err != nil {
		return ctrl.Result{}, err
	}
	
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *ObservabilityPlatformReconciler) transitionToFailed(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, reason, message string) (ctrl.Result, error) {
	platform.Status.Phase = PhaseFailed
	platform.Status.Message = message
	platform.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	
	r.updateCondition(platform, ConditionReady, metav1.ConditionFalse, reason, message)
	r.updateCondition(platform, ConditionProgressing, metav1.ConditionFalse, reason, "Platform operation failed")
	r.updateCondition(platform, ConditionDegraded, metav1.ConditionTrue, reason, message)
	
	if err := r.Status().Update(ctx, platform); err != nil {
		return ctrl.Result{}, err
	}
	
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *ObservabilityPlatformReconciler) transitionToUpgrading(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	platform.Status.Phase = PhaseUpgrading
	platform.Status.Message = "Upgrading platform components"
	platform.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	
	r.updateCondition(platform, ConditionProgressing, metav1.ConditionTrue, ReasonUpgrading, "Platform upgrade in progress")
	r.updateCondition(platform, ConditionReady, metav1.ConditionTrue, ReasonUpgrading, "Platform is available during upgrade")
	
	if err := r.Status().Update(ctx, platform); err != nil {
		return ctrl.Result{}, err
	}
	
	return ctrl.Result{Requeue: true}, nil
}

// Helper methods

func (r *ObservabilityPlatformReconciler) updateCondition(platform *observabilityv1.ObservabilityPlatform, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: platform.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (r *ObservabilityPlatformReconciler) validatePlatform(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Implement validation logic
	// Check for required fields, valid configurations, etc.
	
	// At least one component should be enabled
	if platform.Spec.Components.Prometheus == nil &&
		platform.Spec.Components.Grafana == nil &&
		platform.Spec.Components.Loki == nil &&
		platform.Spec.Components.Tempo == nil {
		return fmt.Errorf("at least one component must be configured")
	}
	
	return nil
}

func (r *ObservabilityPlatformReconciler) checkPrerequisites(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Check namespace exists
	// Check required secrets
	// Check resource quotas
	// etc.
	
	return nil
}

func (r *ObservabilityPlatformReconciler) deployComponent(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, componentName string, manager ComponentManager) (bool, error) {
	if manager == nil {
		return false, fmt.Errorf("component manager not initialized for %s", componentName)
	}

	// Deploy the component
	if err := manager.Deploy(ctx, platform); err != nil {
		return false, err
	}

	// Get component status
	status, err := manager.GetStatus(ctx, platform)
	if err != nil {
		return false, err
	}

	// Update component status
	platform.Status.ComponentStatus[componentName] = *status

	return status.Phase == PhaseReady, nil
}

func (r *ObservabilityPlatformReconciler) isComponentEnabled(platform *observabilityv1.ObservabilityPlatform, componentName string) bool {
	switch componentName {
	case "prometheus":
		return platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled
	case "grafana":
		return platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled
	case "loki":
		return platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled
	case "tempo":
		return platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled
	default:
		return false
	}
}

func (r *ObservabilityPlatformReconciler) getComponentManagers() map[string]ComponentManager {
	return map[string]ComponentManager{
		"prometheus": r.prometheusManager,
		"grafana":    r.grafanaManager,
		"loki":       r.lokiManager,
		"tempo":      r.tempoManager,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObservabilityPlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1.ObservabilityPlatform{}).
		Complete(r)
}
