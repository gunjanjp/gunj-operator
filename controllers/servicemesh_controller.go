// Package controllers provides service mesh controller integration
package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/pkg/servicemesh"
)

// ServiceMeshReconciler reconciles service mesh configuration for ObservabilityPlatform
type ServiceMeshReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	MeshManager   *servicemesh.ExtendedManager
	EventRecorder EventRecorder
}

// EventRecorder interface for recording events
type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
	Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{})
}

// Reconcile handles service mesh reconciliation
func (r *ServiceMeshReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("observabilityplatform", req.NamespacedName)

	// Fetch the ObservabilityPlatform instance
	platform := &observabilityv1.ObservabilityPlatform{}
	err := r.Get(ctx, req.NamespacedName, platform)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, could have been deleted
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if the platform is being deleted
	if !platform.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, platform)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(platform, "servicemesh.observability.io/finalizer") {
		controllerutil.AddFinalizer(platform, "servicemesh.observability.io/finalizer")
		if err := r.Update(ctx, platform); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if service mesh is enabled
	if platform.Spec.ServiceMesh == nil || !platform.Spec.ServiceMesh.Enabled {
		// Service mesh not enabled, check if we need to clean up
		if platform.Status.ServiceMesh != nil && platform.Status.ServiceMesh.Ready {
			log.Info("Service mesh disabled, cleaning up")
			if err := r.MeshManager.RemoveServiceMesh(ctx, platform); err != nil {
				log.Error(err, "Failed to remove service mesh configuration")
				r.EventRecorder.Eventf(platform, corev1.EventTypeWarning, "ServiceMeshRemovalFailed", "Failed to remove service mesh: %v", err)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
			r.EventRecorder.Event(platform, corev1.EventTypeNormal, "ServiceMeshRemoved", "Service mesh configuration removed successfully")
		}
		return ctrl.Result{}, nil
	}

	// Validate service mesh configuration
	if err := r.MeshManager.ValidateServiceMeshConfig(platform); err != nil {
		log.Error(err, "Invalid service mesh configuration")
		r.EventRecorder.Eventf(platform, corev1.EventTypeWarning, "InvalidConfiguration", "Invalid service mesh configuration: %v", err)
		
		// Update status with error
		platform.Status.ServiceMesh = &observabilityv1.ServiceMeshStatus{
			Ready:   false,
			Message: fmt.Sprintf("Configuration validation failed: %v", err),
		}
		if err := r.Status().Update(ctx, platform); err != nil {
			log.Error(err, "Failed to update status")
		}
		
		return ctrl.Result{}, err
	}

	// Configure service mesh
	log.Info("Configuring service mesh")
	if err := r.MeshManager.ConfigureServiceMesh(ctx, platform); err != nil {
		log.Error(err, "Failed to configure service mesh")
		r.EventRecorder.Eventf(platform, corev1.EventTypeWarning, "ServiceMeshConfigurationFailed", "Failed to configure service mesh: %v", err)
		
		// Update status with error
		platform.Status.ServiceMesh = &observabilityv1.ServiceMeshStatus{
			Ready:   false,
			Message: fmt.Sprintf("Configuration failed: %v", err),
		}
		if err := r.Status().Update(ctx, platform); err != nil {
			log.Error(err, "Failed to update status")
		}
		
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Perform health check
	healthResult, err := r.MeshManager.PerformHealthCheck(ctx, platform)
	if err != nil {
		log.Error(err, "Failed to perform health check")
		// Don't fail reconciliation for health check errors
	} else {
		if !healthResult.OverallHealth {
			log.Info("Service mesh health check failed", "recommendations", healthResult.Recommendations)
			r.EventRecorder.Event(platform, corev1.EventTypeWarning, "ServiceMeshUnhealthy", "Service mesh components are not healthy")
		}
	}

	// Update status
	if err := r.updateStatus(ctx, platform); err != nil {
		log.Error(err, "Failed to update platform status")
		return ctrl.Result{}, err
	}

	r.EventRecorder.Event(platform, corev1.EventTypeNormal, "ServiceMeshConfigured", "Service mesh configured successfully")

	// Requeue for periodic health checks
	requeueAfter := 60 * time.Second
	if platform.Spec.ServiceMesh.HealthCheck != nil && platform.Spec.ServiceMesh.HealthCheck.Interval != "" {
		if duration, err := time.ParseDuration(platform.Spec.ServiceMesh.HealthCheck.Interval); err == nil {
			requeueAfter = duration
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// handleDeletion handles the deletion of service mesh resources
func (r *ServiceMeshReconciler) handleDeletion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(platform, "servicemesh.observability.io/finalizer") {
		// Cleanup service mesh configuration
		if err := r.MeshManager.RemoveServiceMesh(ctx, platform); err != nil {
			r.Log.Error(err, "Failed to cleanup service mesh during deletion")
			// Don't block deletion
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(platform, "servicemesh.observability.io/finalizer")
		if err := r.Update(ctx, platform); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// updateStatus updates the platform status with service mesh information
func (r *ServiceMeshReconciler) updateStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// The status is already updated by the mesh manager
	// This is just to ensure it's persisted
	return r.Status().Update(ctx, platform)
}

// SetupWithManager sets up the controller with the Manager
func (r *ServiceMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create predicate to watch for service mesh related changes
	serviceMeshPredicate := predicate.Or(
		predicate.GenerationChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
		predicate.LabelChangedPredicate{},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1.ObservabilityPlatform{}).
		WithEventFilter(serviceMeshPredicate).
		Complete(r)
}

// ReconcileServiceMesh is a helper function to be called from the main controller
func ReconcileServiceMesh(ctx context.Context, client client.Client, scheme *runtime.Scheme, platform *observabilityv1.ObservabilityPlatform, log logr.Logger) error {
	// Create mesh manager if not exists
	meshManager := servicemesh.NewExtendedManager(client, scheme)

	// Check if service mesh is enabled
	if platform.Spec.ServiceMesh == nil || !platform.Spec.ServiceMesh.Enabled {
		// Check if we need to clean up
		if platform.Status.ServiceMesh != nil && platform.Status.ServiceMesh.Ready {
			log.Info("Service mesh disabled, cleaning up")
			if err := meshManager.RemoveServiceMesh(ctx, platform); err != nil {
				return fmt.Errorf("removing service mesh: %w", err)
			}
		}
		return nil
	}

	// Validate configuration
	if err := meshManager.ValidateServiceMeshConfig(platform); err != nil {
		return fmt.Errorf("validating service mesh config: %w", err)
	}

	// Configure service mesh
	if err := meshManager.ConfigureServiceMesh(ctx, platform); err != nil {
		return fmt.Errorf("configuring service mesh: %w", err)
	}

	// Perform health check
	if _, err := meshManager.PerformHealthCheck(ctx, platform); err != nil {
		log.Error(err, "Service mesh health check failed")
		// Don't fail reconciliation for health check errors
	}

	return nil
}

// ServiceMeshCondition represents the condition type for service mesh
const (
	ServiceMeshReady     = "ServiceMeshReady"
	ServiceMeshHealthy   = "ServiceMeshHealthy"
	ServiceMeshConfigured = "ServiceMeshConfigured"
)

// UpdateServiceMeshCondition updates the service mesh condition in platform status
func UpdateServiceMeshCondition(platform *observabilityv1.ObservabilityPlatform, conditionType string, status corev1.ConditionStatus, reason, message string) {
	condition := observabilityv1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: observabilityv1.Time{Time: time.Now()},
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	found := false
	for i, c := range platform.Status.Conditions {
		if c.Type == conditionType {
			if c.Status != status {
				platform.Status.Conditions[i] = condition
			} else {
				// Update message and reason even if status hasn't changed
				platform.Status.Conditions[i].Message = message
				platform.Status.Conditions[i].Reason = reason
			}
			found = true
			break
		}
	}

	if !found {
		platform.Status.Conditions = append(platform.Status.Conditions, condition)
	}
}
