/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/gitops"
)

// GitOpsDeploymentReconciler reconciles a GitOpsDeployment object
type GitOpsDeploymentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// GitOps managers
	ArgoCDManager    *gitops.ArgoCDManager
	FluxManager      *gitops.FluxManager
	GitSyncManager   *gitops.GitSyncManager
	DriftDetector    *gitops.DriftDetector
	RollbackManager  *gitops.RollbackManager
	PromotionManager *gitops.PromotionManager
}

const (
	gitopsDeploymentFinalizer = "gitopsdeployment.observability.io/finalizer"
	
	// Conditions
	ConditionReady             = "Ready"
	ConditionSynced            = "Synced"
	ConditionHealthy           = "Healthy"
	ConditionDriftDetected     = "DriftDetected"
	ConditionPromotionReady    = "PromotionReady"
	
	// Reasons
	ReasonInitializing         = "Initializing"
	ReasonSyncInProgress       = "SyncInProgress"
	ReasonSyncFailed           = "SyncFailed"
	ReasonSyncSucceeded        = "SyncSucceeded"
	ReasonHealthCheckFailed    = "HealthCheckFailed"
	ReasonDriftDetected        = "DriftDetected"
	ReasonDriftRemediated      = "DriftRemediated"
	ReasonRollbackInProgress   = "RollbackInProgress"
	ReasonRollbackCompleted    = "RollbackCompleted"
	ReasonPromotionInProgress  = "PromotionInProgress"
	ReasonPromotionCompleted   = "PromotionCompleted"
)

// +kubebuilder:rbac:groups=observability.io,resources=gitopsdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=observability.io,resources=gitopsdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=observability.io,resources=gitopsdeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kustomize.toolkit.fluxcd.io,resources=kustomizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GitOpsDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("gitopsdeployment", req.NamespacedName)

	// Fetch the GitOpsDeployment instance
	deployment := &observabilityv1beta1.GitOpsDeployment{}
	if err := r.Get(ctx, req.NamespacedName, deployment); err != nil {
		if errors.IsNotFound(err) {
			log.Info("GitOpsDeployment resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get GitOpsDeployment")
		return ctrl.Result{}, err
	}

	// Check if the deployment is being deleted
	if !deployment.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, deployment)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(deployment, gitopsDeploymentFinalizer) {
		controllerutil.AddFinalizer(deployment, gitopsDeploymentFinalizer)
		if err := r.Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Initialize status if needed
	if deployment.Status.Phase == "" {
		deployment.Status.Phase = "Pending"
		deployment.Status.ObservedGeneration = deployment.Generation
		if err := r.Status().Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Main reconciliation logic
	return r.reconcileDeployment(ctx, deployment)
}

// reconcileDeployment handles the main reconciliation logic
func (r *GitOpsDeploymentReconciler) reconcileDeployment(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (ctrl.Result, error) {
	log := r.Log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	// Update phase to Initializing
	deployment.Status.Phase = "Initializing"
	deployment.Status.ObservedGeneration = deployment.Generation

	// Step 1: Setup Git repository synchronization
	if err := r.setupGitSync(ctx, deployment); err != nil {
		deployment.Status.Phase = "Failed"
		r.setCondition(deployment, ConditionReady, metav1.ConditionFalse, ReasonInitializing, err.Error())
		r.Recorder.Event(deployment, corev1.EventTypeWarning, "GitSyncFailed", err.Error())
		if err := r.Status().Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Step 2: Setup GitOps engine (ArgoCD or Flux)
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		if err := r.setupArgoCD(ctx, deployment); err != nil {
			deployment.Status.Phase = "Failed"
			r.setCondition(deployment, ConditionReady, metav1.ConditionFalse, ReasonInitializing, err.Error())
			r.Recorder.Event(deployment, corev1.EventTypeWarning, "ArgoCDSetupFailed", err.Error())
			if err := r.Status().Update(ctx, deployment); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	case "flux":
		if err := r.setupFlux(ctx, deployment); err != nil {
			deployment.Status.Phase = "Failed"
			r.setCondition(deployment, ConditionReady, metav1.ConditionFalse, ReasonInitializing, err.Error())
			r.Recorder.Event(deployment, corev1.EventTypeWarning, "FluxSetupFailed", err.Error())
			if err := r.Status().Update(ctx, deployment); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Step 3: Setup environments
	if err := r.setupEnvironments(ctx, deployment); err != nil {
		deployment.Status.Phase = "Failed"
		r.setCondition(deployment, ConditionReady, metav1.ConditionFalse, ReasonInitializing, err.Error())
		r.Recorder.Event(deployment, corev1.EventTypeWarning, "EnvironmentSetupFailed", err.Error())
		if err := r.Status().Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Step 4: Check sync status
	syncStatus, err := r.checkSyncStatus(ctx, deployment)
	if err != nil {
		log.Error(err, "Failed to check sync status")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	deployment.Status.SyncStatus = syncStatus

	// Step 5: Check health status
	healthStatus, err := r.checkHealthStatus(ctx, deployment)
	if err != nil {
		log.Error(err, "Failed to check health status")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	deployment.Status.HealthStatus = healthStatus

	// Step 6: Run drift detection if enabled
	if deployment.Spec.DriftDetection != nil && deployment.Spec.DriftDetection.Enabled {
		driftStatus, err := r.runDriftDetection(ctx, deployment)
		if err != nil {
			log.Error(err, "Failed to run drift detection")
		} else {
			deployment.Status.DriftStatus = driftStatus
			
			// Auto-remediate if configured and drift detected
			if driftStatus.Detected && deployment.Spec.DriftDetection.AutoRemediate {
				if err := r.remediateDrift(ctx, deployment, driftStatus); err != nil {
					log.Error(err, "Failed to remediate drift")
					r.Recorder.Event(deployment, corev1.EventTypeWarning, "DriftRemediationFailed", err.Error())
				} else {
					r.Recorder.Event(deployment, corev1.EventTypeNormal, ReasonDriftRemediated, "Drift remediated successfully")
				}
			}
		}
	}

	// Step 7: Handle rollback if needed
	if deployment.Spec.Rollback != nil && deployment.Spec.Rollback.Enabled {
		if shouldRollback(deployment) {
			if err := r.performRollback(ctx, deployment); err != nil {
				log.Error(err, "Failed to perform rollback")
				r.Recorder.Event(deployment, corev1.EventTypeWarning, "RollbackFailed", err.Error())
			} else {
				r.Recorder.Event(deployment, corev1.EventTypeNormal, ReasonRollbackCompleted, "Rollback completed successfully")
			}
		}
	}

	// Step 8: Handle promotion if configured
	if err := r.handlePromotion(ctx, deployment); err != nil {
		log.Error(err, "Failed to handle promotion")
	}

	// Update overall phase
	deployment.Status.Phase = r.calculatePhase(deployment)
	
	// Update conditions
	r.updateConditions(deployment)

	// Update status
	if err := r.Status().Update(ctx, deployment); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue based on status
	requeueAfter := r.calculateRequeueTime(deployment)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// setupGitSync sets up Git repository synchronization
func (r *GitOpsDeploymentReconciler) setupGitSync(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	log := r.Log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	// Get credentials if configured
	var creds *gitops.GitCredentials
	if deployment.Spec.Repository.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      deployment.Spec.Repository.SecretRef.Name,
			Namespace: deployment.Namespace,
		}, secret); err != nil {
			return fmt.Errorf("failed to get Git credentials secret: %w", err)
		}

		creds = &gitops.GitCredentials{
			Username: string(secret.Data["username"]),
			Password: string(secret.Data["password"]),
			SSHKey:   string(secret.Data["sshKey"]),
		}
	}

	// Setup Git sync
	syncConfig := &gitops.GitSyncConfig{
		Repository:   deployment.Spec.Repository.URL,
		Branch:       deployment.Spec.Repository.Branch,
		Path:         deployment.Spec.Repository.Path,
		PollInterval: deployment.Spec.Repository.PollInterval,
		Credentials:  creds,
	}

	if err := r.GitSyncManager.SetupSync(ctx, deployment, syncConfig); err != nil {
		return fmt.Errorf("failed to setup Git sync: %w", err)
	}

	log.Info("Git sync setup completed successfully")
	return nil
}

// setupArgoCD sets up ArgoCD Application
func (r *GitOpsDeploymentReconciler) setupArgoCD(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	log := r.Log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	if deployment.Spec.ArgoCD == nil {
		return fmt.Errorf("ArgoCD configuration not provided")
	}

	// Create ArgoCD applications for each environment
	for _, env := range deployment.Spec.Environments {
		appConfig := &gitops.ArgoCDAppConfig{
			Name:               fmt.Sprintf("%s-%s", deployment.Spec.ArgoCD.ApplicationName, env.Name),
			Project:            deployment.Spec.ArgoCD.Project,
			RepoURL:            deployment.Spec.Repository.URL,
			Path:               getEnvironmentPath(deployment, env),
			TargetRevision:     getEnvironmentBranch(deployment, env),
			DestinationServer:  "https://kubernetes.default.svc",
			DestinationNamespace: env.Namespace,
			SyncOptions:        deployment.Spec.ArgoCD.SyncOptions,
			AutoSync:           deployment.Spec.AutoSync,
			Values:             env.Values,
		}

		if err := r.ArgoCDManager.CreateOrUpdateApplication(ctx, deployment, appConfig); err != nil {
			return fmt.Errorf("failed to create ArgoCD application for environment %s: %w", env.Name, err)
		}

		log.Info("ArgoCD application created/updated", "environment", env.Name, "app", appConfig.Name)
	}

	return nil
}

// setupFlux sets up Flux Kustomization
func (r *GitOpsDeploymentReconciler) setupFlux(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	log := r.Log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	if deployment.Spec.Flux == nil {
		return fmt.Errorf("Flux configuration not provided")
	}

	// Create Flux Kustomizations for each environment
	for _, env := range deployment.Spec.Environments {
		kustomizationConfig := &gitops.FluxKustomizationConfig{
			Name:            fmt.Sprintf("%s-%s", deployment.Spec.Flux.KustomizationName, env.Name),
			SourceRef:       fmt.Sprintf("%s-source", deployment.Name),
			Path:            getEnvironmentPath(deployment, env),
			TargetNamespace: env.Namespace,
			ServiceAccount:  deployment.Spec.Flux.ServiceAccount,
			Interval:        deployment.Spec.Flux.Interval,
			Timeout:         deployment.Spec.Flux.Timeout,
			Prune:           deployment.Spec.Flux.Prune,
			Values:          env.Values,
		}

		if err := r.FluxManager.CreateOrUpdateKustomization(ctx, deployment, kustomizationConfig); err != nil {
			return fmt.Errorf("failed to create Flux Kustomization for environment %s: %w", env.Name, err)
		}

		log.Info("Flux Kustomization created/updated", "environment", env.Name, "kustomization", kustomizationConfig.Name)
	}

	return nil
}

// setupEnvironments sets up deployment environments
func (r *GitOpsDeploymentReconciler) setupEnvironments(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	log := r.Log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	// Initialize environment status
	deployment.Status.Environments = make([]observabilityv1beta1.EnvironmentStatus, 0, len(deployment.Spec.Environments))

	for _, env := range deployment.Spec.Environments {
		// Create namespace if it doesn't exist
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: env.Namespace,
				Labels: map[string]string{
					"observability.io/gitops-deployment": deployment.Name,
					"observability.io/environment":       env.Name,
				},
			},
		}

		if err := r.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create namespace %s: %w", env.Namespace, err)
		}

		// Run pre-sync hooks
		if err := r.runSyncHooks(ctx, deployment, env, env.PreSync, "pre-sync"); err != nil {
			log.Error(err, "Failed to run pre-sync hooks", "environment", env.Name)
		}

		// Add environment status
		envStatus := observabilityv1beta1.EnvironmentStatus{
			Name:  env.Name,
			Phase: "Pending",
		}
		deployment.Status.Environments = append(deployment.Status.Environments, envStatus)

		log.Info("Environment setup completed", "environment", env.Name, "namespace", env.Namespace)
	}

	return nil
}

// checkSyncStatus checks the synchronization status
func (r *GitOpsDeploymentReconciler) checkSyncStatus(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.SyncStatus, error) {
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		return r.ArgoCDManager.GetSyncStatus(ctx, deployment)
	case "flux":
		return r.FluxManager.GetSyncStatus(ctx, deployment)
	default:
		return nil, fmt.Errorf("unknown GitOps engine: %s", deployment.Spec.GitOpsEngine)
	}
}

// checkHealthStatus checks the health status
func (r *GitOpsDeploymentReconciler) checkHealthStatus(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.HealthStatus, error) {
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		return r.ArgoCDManager.GetHealthStatus(ctx, deployment)
	case "flux":
		return r.FluxManager.GetHealthStatus(ctx, deployment)
	default:
		return nil, fmt.Errorf("unknown GitOps engine: %s", deployment.Spec.GitOpsEngine)
	}
}

// runDriftDetection runs drift detection
func (r *GitOpsDeploymentReconciler) runDriftDetection(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.DriftStatus, error) {
	return r.DriftDetector.DetectDrift(ctx, deployment)
}

// remediateDrift remediates detected drift
func (r *GitOpsDeploymentReconciler) remediateDrift(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, driftStatus *observabilityv1beta1.DriftStatus) error {
	return r.DriftDetector.RemediateDrift(ctx, deployment, driftStatus)
}

// shouldRollback determines if rollback is needed
func shouldRollback(deployment *observabilityv1beta1.GitOpsDeployment) bool {
	// Check if health is degraded
	if deployment.Status.HealthStatus != nil && deployment.Status.HealthStatus.Status == "Degraded" {
		return true
	}

	// Check if sync has failed multiple times
	if deployment.Status.SyncStatus != nil && deployment.Status.SyncStatus.Status == "Failed" {
		// TODO: Check retry count
		return true
	}

	return false
}

// performRollback performs a rollback
func (r *GitOpsDeploymentReconciler) performRollback(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	return r.RollbackManager.Rollback(ctx, deployment)
}

// handlePromotion handles environment promotion
func (r *GitOpsDeploymentReconciler) handlePromotion(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	for _, env := range deployment.Spec.Environments {
		if env.PromotionPolicy != nil && env.PromotionPolicy.AutoPromotion {
			if r.PromotionManager.ShouldPromote(ctx, deployment, env) {
				if err := r.PromotionManager.Promote(ctx, deployment, env); err != nil {
					return fmt.Errorf("failed to promote to environment %s: %w", env.Name, err)
				}
				r.Recorder.Event(deployment, corev1.EventTypeNormal, ReasonPromotionCompleted, 
					fmt.Sprintf("Promoted to environment %s", env.Name))
			}
		}
	}
	return nil
}

// runSyncHooks runs sync hooks
func (r *GitOpsDeploymentReconciler) runSyncHooks(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment, hooks []observabilityv1beta1.SyncHook, phase string) error {
	for _, hook := range hooks {
		log := r.Log.WithValues("hook", hook.Name, "phase", phase, "environment", env.Name)
		
		switch hook.Type {
		case "Job":
			// TODO: Run Kubernetes Job
			log.Info("Running Job hook")
		case "Webhook":
			// TODO: Call webhook
			log.Info("Calling webhook")
		case "Script":
			// TODO: Execute script
			log.Info("Executing script")
		default:
			log.Error(nil, "Unknown hook type", "type", hook.Type)
		}
	}
	return nil
}

// calculatePhase calculates the overall phase
func (r *GitOpsDeploymentReconciler) calculatePhase(deployment *observabilityv1beta1.GitOpsDeployment) string {
	// Check sync status
	if deployment.Status.SyncStatus != nil {
		switch deployment.Status.SyncStatus.Status {
		case "OutOfSync":
			return "Syncing"
		case "Unknown":
			return "Unknown"
		}
	}

	// Check health status
	if deployment.Status.HealthStatus != nil {
		switch deployment.Status.HealthStatus.Status {
		case "Degraded":
			return "Failed"
		case "Progressing":
			return "Syncing"
		case "Suspended", "Missing":
			return "Failed"
		case "Unknown":
			return "Unknown"
		}
	}

	// If synced and healthy
	if deployment.Status.SyncStatus != nil && deployment.Status.SyncStatus.Status == "Synced" &&
		deployment.Status.HealthStatus != nil && deployment.Status.HealthStatus.Status == "Healthy" {
		return "Synced"
	}

	return "Unknown"
}

// updateConditions updates the deployment conditions
func (r *GitOpsDeploymentReconciler) updateConditions(deployment *observabilityv1beta1.GitOpsDeployment) {
	// Ready condition
	if deployment.Status.Phase == "Synced" {
		r.setCondition(deployment, ConditionReady, metav1.ConditionTrue, "Ready", "GitOps deployment is ready")
	} else {
		r.setCondition(deployment, ConditionReady, metav1.ConditionFalse, deployment.Status.Phase, "GitOps deployment is not ready")
	}

	// Synced condition
	if deployment.Status.SyncStatus != nil {
		if deployment.Status.SyncStatus.Status == "Synced" {
			r.setCondition(deployment, ConditionSynced, metav1.ConditionTrue, ReasonSyncSucceeded, "Application is synced")
		} else {
			r.setCondition(deployment, ConditionSynced, metav1.ConditionFalse, ReasonSyncFailed, deployment.Status.SyncStatus.Message)
		}
	}

	// Healthy condition
	if deployment.Status.HealthStatus != nil {
		if deployment.Status.HealthStatus.Status == "Healthy" {
			r.setCondition(deployment, ConditionHealthy, metav1.ConditionTrue, "Healthy", "Application is healthy")
		} else {
			r.setCondition(deployment, ConditionHealthy, metav1.ConditionFalse, ReasonHealthCheckFailed, deployment.Status.HealthStatus.Message)
		}
	}

	// DriftDetected condition
	if deployment.Status.DriftStatus != nil {
		if deployment.Status.DriftStatus.Detected {
			r.setCondition(deployment, ConditionDriftDetected, metav1.ConditionTrue, ReasonDriftDetected, 
				fmt.Sprintf("Drift detected in %d resources", len(deployment.Status.DriftStatus.DriftedResources)))
		} else {
			r.setCondition(deployment, ConditionDriftDetected, metav1.ConditionFalse, "NoDrift", "No drift detected")
		}
	}
}

// calculateRequeueTime calculates the requeue time based on status
func (r *GitOpsDeploymentReconciler) calculateRequeueTime(deployment *observabilityv1beta1.GitOpsDeployment) time.Duration {
	// If syncing, check more frequently
	if deployment.Status.Phase == "Syncing" {
		return 10 * time.Second
	}

	// If failed, retry less frequently
	if deployment.Status.Phase == "Failed" {
		return 60 * time.Second
	}

	// Default poll interval
	pollInterval, err := time.ParseDuration(deployment.Spec.Repository.PollInterval)
	if err != nil {
		return 60 * time.Second
	}

	return pollInterval
}

// handleDeletion handles deployment deletion
func (r *GitOpsDeploymentReconciler) handleDeletion(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (ctrl.Result, error) {
	log := r.Log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	// Clean up GitOps resources
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		if err := r.ArgoCDManager.DeleteApplications(ctx, deployment); err != nil {
			log.Error(err, "Failed to delete ArgoCD applications")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	case "flux":
		if err := r.FluxManager.DeleteKustomizations(ctx, deployment); err != nil {
			log.Error(err, "Failed to delete Flux Kustomizations")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Clean up Git sync resources
	if err := r.GitSyncManager.CleanupSync(ctx, deployment); err != nil {
		log.Error(err, "Failed to cleanup Git sync")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(deployment, gitopsDeploymentFinalizer)
	if err := r.Update(ctx, deployment); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("GitOpsDeployment deleted successfully")
	return ctrl.Result{}, nil
}

// setCondition sets a condition on the deployment
func (r *GitOpsDeploymentReconciler) setCondition(deployment *observabilityv1beta1.GitOpsDeployment, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&deployment.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: deployment.Generation,
		Reason:             reason,
		Message:            message,
	})
}

// getEnvironmentPath returns the path for an environment
func getEnvironmentPath(deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment) string {
	if env.Path != "" {
		return env.Path
	}
	return fmt.Sprintf("%s/%s", deployment.Spec.Repository.Path, env.Name)
}

// getEnvironmentBranch returns the branch for an environment
func getEnvironmentBranch(deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment) string {
	if env.Branch != "" {
		return env.Branch
	}
	return deployment.Spec.Repository.Branch
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitOpsDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize managers
	r.ArgoCDManager = gitops.NewArgoCDManager(mgr.GetClient(), r.Log)
	r.FluxManager = gitops.NewFluxManager(mgr.GetClient(), r.Log)
	r.GitSyncManager = gitops.NewGitSyncManager(mgr.GetClient(), r.Log)
	r.DriftDetector = gitops.NewDriftDetector(mgr.GetClient(), r.Log)
	r.RollbackManager = gitops.NewRollbackManager(mgr.GetClient(), r.Log)
	r.PromotionManager = gitops.NewPromotionManager(mgr.GetClient(), r.Log)

	return ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1beta1.GitOpsDeployment{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.findDeploymentsForSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

// findDeploymentsForSecret finds GitOpsDeployments that reference a secret
func (r *GitOpsDeploymentReconciler) findDeploymentsForSecret(secret client.Object) []reconcile.Request {
	deployments := &observabilityv1beta1.GitOpsDeploymentList{}
	if err := r.List(context.Background(), deployments); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, deployment := range deployments.Items {
		if deployment.Spec.Repository.SecretRef != nil &&
			deployment.Spec.Repository.SecretRef.Name == secret.GetName() &&
			deployment.Namespace == secret.GetNamespace() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      deployment.Name,
					Namespace: deployment.Namespace,
				},
			})
		}
	}

	return requests
}
