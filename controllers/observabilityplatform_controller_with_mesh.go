// ObservabilityPlatformReconciler reconciles a ObservabilityPlatform object
// Updated to include service mesh integration
package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/managers"
)

// ObservabilityPlatformReconciler reconciles a ObservabilityPlatform object
type ObservabilityPlatformReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	
	// Component managers
	PrometheusManager  managers.ComponentManager
	GrafanaManager     managers.ComponentManager
	LokiManager        managers.ComponentManager
	TempoManager       managers.ComponentManager
	
	// Service mesh reconciler
	ServiceMeshReconciler *ServiceMeshReconciler
}

// +kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=observability.io,resources=observabilityplatforms/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps;secrets;persistentvolumeclaims;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses;networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheuses;alertmanagers;servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices;destinationrules;gateways;serviceentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.istio.io,resources=peerauthentications;authorizationpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy.linkerd.io,resources=servers;serverauthorizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=linkerd.io,resources=serviceprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=split.smi-spec.io,resources=trafficsplits,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *ObservabilityPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the ObservabilityPlatform instance
	platform := &observabilityv1.ObservabilityPlatform{}
	err := r.Get(ctx, req.NamespacedName, platform)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			log.Info("ObservabilityPlatform resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ObservabilityPlatform")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !platform.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, platform)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(platform, "observability.io/finalizer") {
		controllerutil.AddFinalizer(platform, "observability.io/finalizer")
		if err := r.Update(ctx, platform); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update status phase
	oldPhase := platform.Status.Phase
	if platform.Status.Phase == "" {
		platform.Status.Phase = "Pending"
	}

	// Initialize component statuses
	if platform.Status.ComponentStatuses == nil {
		platform.Status.ComponentStatuses = make(map[string]observabilityv1.ComponentStatus)
	}

	// Service Mesh Integration - Early Phase
	if r.ServiceMeshReconciler != nil && platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.Enabled {
		log.Info("Reconciling service mesh integration")
		if err := r.ServiceMeshReconciler.Reconcile(ctx, platform); err != nil {
			log.Error(err, "Failed to reconcile service mesh")
			platform.Status.Phase = "Failed"
			platform.Status.Message = fmt.Sprintf("Service mesh reconciliation failed: %v", err)
			_ = r.Status().Update(ctx, platform)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
	}

	// Reconcile components
	platform.Status.Phase = "Installing"
	_ = r.Status().Update(ctx, platform)

	// Reconcile Prometheus
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		log.Info("Reconciling Prometheus")
		if err := r.PrometheusManager.Reconcile(ctx, platform); err != nil {
			log.Error(err, "Failed to reconcile Prometheus")
			platform.Status.ComponentStatuses["prometheus"] = observabilityv1.ComponentStatus{
				Ready:   false,
				Message: err.Error(),
			}
			platform.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, platform)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		platform.Status.ComponentStatuses["prometheus"] = observabilityv1.ComponentStatus{
			Ready:   true,
			Version: platform.Spec.Components.Prometheus.Version,
			Message: "Prometheus is running",
		}
	}

	// Reconcile Grafana
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		log.Info("Reconciling Grafana")
		if err := r.GrafanaManager.Reconcile(ctx, platform); err != nil {
			log.Error(err, "Failed to reconcile Grafana")
			platform.Status.ComponentStatuses["grafana"] = observabilityv1.ComponentStatus{
				Ready:   false,
				Message: err.Error(),
			}
			platform.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, platform)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		platform.Status.ComponentStatuses["grafana"] = observabilityv1.ComponentStatus{
			Ready:   true,
			Version: platform.Spec.Components.Grafana.Version,
			Message: "Grafana is running",
		}
	}

	// Reconcile Loki
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		log.Info("Reconciling Loki")
		if err := r.LokiManager.Reconcile(ctx, platform); err != nil {
			log.Error(err, "Failed to reconcile Loki")
			platform.Status.ComponentStatuses["loki"] = observabilityv1.ComponentStatus{
				Ready:   false,
				Message: err.Error(),
			}
			platform.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, platform)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		platform.Status.ComponentStatuses["loki"] = observabilityv1.ComponentStatus{
			Ready:   true,
			Version: platform.Spec.Components.Loki.Version,
			Message: "Loki is running",
		}
	}

	// Reconcile Tempo
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		log.Info("Reconciling Tempo")
		if err := r.TempoManager.Reconcile(ctx, platform); err != nil {
			log.Error(err, "Failed to reconcile Tempo")
			platform.Status.ComponentStatuses["tempo"] = observabilityv1.ComponentStatus{
				Ready:   false,
				Message: err.Error(),
			}
			platform.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, platform)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		platform.Status.ComponentStatuses["tempo"] = observabilityv1.ComponentStatus{
			Ready:   true,
			Version: platform.Spec.Components.Tempo.Version,
			Message: "Tempo is running",
		}
	}

	// Service Mesh Integration - Post Component Phase
	// Update traffic policies after components are deployed
	if r.ServiceMeshReconciler != nil && platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.Enabled {
		if err := r.ServiceMeshReconciler.UpdateTrafficPolicy(ctx, platform); err != nil {
			log.Error(err, "Failed to update traffic policies")
			// Non-critical error, don't fail the reconciliation
		}
	}

	// Check overall status
	allReady := true
	for _, status := range platform.Status.ComponentStatuses {
		if !status.Ready {
			allReady = false
			break
		}
	}

	if allReady {
		platform.Status.Phase = "Ready"
		platform.Status.Message = "All components are running"
	} else {
		platform.Status.Phase = "Progressing"
		platform.Status.Message = "Some components are not ready"
	}

	// Update status
	platform.Status.ObservedGeneration = platform.Generation
	now := metav1.Now()
	platform.Status.LastReconcileTime = &now

	if oldPhase != platform.Status.Phase {
		log.Info("Phase changed", "from", oldPhase, "to", platform.Status.Phase)
	}

	if err := r.Status().Update(ctx, platform); err != nil {
		log.Error(err, "Failed to update ObservabilityPlatform status")
		return ctrl.Result{}, err
	}

	// Requeue after 5 minutes for continuous reconciliation
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// handleDeletion handles the deletion of the ObservabilityPlatform
func (r *ObservabilityPlatformReconciler) handleDeletion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(platform, "observability.io/finalizer") {
		// Run finalization logic
		log.Info("Running finalization logic for ObservabilityPlatform", "name", platform.Name)

		// Clean up service mesh configurations
		if r.ServiceMeshReconciler != nil && platform.Spec.ServiceMesh != nil && platform.Spec.ServiceMesh.Enabled {
			if err := r.ServiceMeshReconciler.Cleanup(ctx, platform); err != nil {
				log.Error(err, "Failed to cleanup service mesh configurations")
				// Continue with deletion even if cleanup fails
			}
		}

		// Clean up components in reverse order
		if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
			if err := r.TempoManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Tempo")
			}
		}

		if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
			if err := r.LokiManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Loki")
			}
		}

		if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
			if err := r.GrafanaManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Grafana")
			}
		}

		if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
			if err := r.PrometheusManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Prometheus")
			}
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(platform, "observability.io/finalizer")
		if err := r.Update(ctx, platform); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObservabilityPlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize service mesh reconciler
	r.ServiceMeshReconciler = NewServiceMeshReconciler(mgr.GetClient())

	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1.ObservabilityPlatform{}).
		Complete(r)
}
