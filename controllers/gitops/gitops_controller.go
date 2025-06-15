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

package gitops

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/gitops/argocd"
	"github.com/gunjanjp/gunj-operator/internal/gitops/drift"
	"github.com/gunjanjp/gunj-operator/internal/gitops/flux"
	"github.com/gunjanjp/gunj-operator/internal/gitops/git"
	"github.com/gunjanjp/gunj-operator/internal/gitops/promotion"
	"github.com/gunjanjp/gunj-operator/internal/gitops/webhook"
)

// GitOpsController reconciles GitOps integrations for ObservabilityPlatforms
type GitOpsController struct {
	client.Client
	Log              logr.Logger
	Scheme           *runtime.Scheme
	ArgoCDManager    *argocd.ArgoCDManager
	FluxManager      *flux.FluxManager
	GitManager       *git.RepositoryManager
	DriftDetector    *drift.DetectionManager
	PromotionManager *promotion.Manager
	WebhookHandler   *webhook.Handler
}

// NewGitOpsController creates a new GitOps controller
func NewGitOpsController(
	client client.Client,
	scheme *runtime.Scheme,
	log logr.Logger,
) (*GitOpsController, error) {
	// Create Git manager
	gitManager, err := git.NewRepositoryManager(client, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create git manager: %w", err)
	}

	controller := &GitOpsController{
		Client:           client,
		Log:              log.WithName("gitops-controller"),
		Scheme:           scheme,
		ArgoCDManager:    argocd.NewArgoCDManager(client, scheme, log),
		FluxManager:      flux.NewFluxManager(client, scheme, log),
		GitManager:       gitManager,
		DriftDetector:    drift.NewDetectionManager(client, scheme, log),
		PromotionManager: promotion.NewManager(client, log, gitManager),
	}

	// Create webhook handler with event processor
	controller.WebhookHandler = webhook.NewHandler(client, log, controller)

	return controller, nil
}

// SetupWithManager sets up the controller with the Manager
func (c *GitOpsController) SetupWithManager(mgr ctrl.Manager) error {
	// Create controller
	gitopsController, err := ctrl.NewControllerManagedBy(mgr).
		For(&observabilityv1.ObservabilityPlatform{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3,
		}).
		WithEventFilter(c.gitOpsEnabledPredicate()).
		Build(c)
	
	if err != nil {
		return err
	}

	// Watch for GitOps-related events
	if err := c.watchGitOpsResources(gitopsController); err != nil {
		return err
	}

	// Start drift detection worker
	go c.startDriftDetectionWorker(mgr.GetClient())

	// Register webhook routes
	if c.WebhookHandler != nil {
		// This assumes you have a way to register HTTP routes
		// In a real implementation, you would integrate with your HTTP server
		c.Log.Info("GitOps webhook handler registered")
	}

	return nil
}

// Reconcile handles reconciliation for GitOps-enabled platforms
func (c *GitOpsController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := c.Log.WithValues("observabilityplatform", req.NamespacedName)
	log.Info("Reconciling GitOps integration")

	// Fetch the ObservabilityPlatform
	platform := &observabilityv1.ObservabilityPlatform{}
	err := c.Get(ctx, req.NamespacedName, platform)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if GitOps is enabled
	if platform.Spec.GitOps == nil {
		log.Info("GitOps not configured for platform")
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if platform.GetDeletionTimestamp() != nil {
		return c.handleDeletion(ctx, platform)
	}

	// Ensure finalizer
	if !containsString(platform.GetFinalizers(), "gitops.observability.io/finalizer") {
		platform.SetFinalizers(append(platform.GetFinalizers(), "gitops.observability.io/finalizer"))
		if err := c.Update(ctx, platform); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile based on GitOps type
	gitOps := convertToGitOpsSpec(platform.Spec.GitOps)
	
	switch gitOps.Type {
	case gitopsv1beta1.GitOpsArgoCD:
		return c.reconcileArgoCD(ctx, platform, gitOps)
	case gitopsv1beta1.GitOpsFlux:
		return c.reconcileFlux(ctx, platform, gitOps)
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported GitOps type: %s", gitOps.Type)
	}
}

// reconcileArgoCD reconciles ArgoCD integration
func (c *GitOpsController) reconcileArgoCD(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) (ctrl.Result, error) {
	log := c.Log.WithValues("platform", platform.Name, "gitops", "argocd")
	log.Info("Reconciling ArgoCD integration")

	// Create or update ArgoCD Application
	if err := c.ArgoCDManager.CreateApplication(ctx, platform, gitOps); err != nil {
		c.updateGitOpsStatus(ctx, platform, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	// Create ApplicationSet for multi-environment
	if len(gitOps.Environments) > 0 {
		if err := c.ArgoCDManager.CreateApplicationSet(ctx, platform, gitOps); err != nil {
			c.updateGitOpsStatus(ctx, platform, "Failed", err.Error())
			return ctrl.Result{}, err
		}
	}

	// Get sync status
	syncStatus, err := c.ArgoCDManager.GetSyncStatus(ctx, platform)
	if err != nil {
		log.Error(err, "Failed to get sync status")
	} else {
		// Update status based on sync
		phase := "Syncing"
		if syncStatus.Phase == "Synced" {
			phase = "Synced"
		} else if syncStatus.Phase == "OutOfSync" {
			phase = "OutOfSync"
		}
		
		c.updateGitOpsStatus(ctx, platform, phase, syncStatus.Message)
		c.updateGitOpsRevision(ctx, platform, syncStatus.Revision)
	}

	// Schedule next reconciliation
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// reconcileFlux reconciles Flux integration
func (c *GitOpsController) reconcileFlux(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) (ctrl.Result, error) {
	log := c.Log.WithValues("platform", platform.Name, "gitops", "flux")
	log.Info("Reconciling Flux integration")

	// Create GitRepository source
	if err := c.FluxManager.CreateGitRepository(ctx, platform, gitOps); err != nil {
		c.updateGitOpsStatus(ctx, platform, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	// Create Kustomization
	if err := c.FluxManager.CreateKustomization(ctx, platform, gitOps); err != nil {
		c.updateGitOpsStatus(ctx, platform, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	// Create multi-environment Kustomizations
	if len(gitOps.Environments) > 0 {
		if err := c.FluxManager.CreateMultiEnvironmentKustomizations(ctx, platform, gitOps); err != nil {
			c.updateGitOpsStatus(ctx, platform, "Failed", err.Error())
			return ctrl.Result{}, err
		}
	}

	// Create image update automation if configured
	if gitOps.Type == gitopsv1beta1.GitOpsFlux {
		if err := c.FluxManager.CreateImageUpdateAutomation(ctx, platform, gitOps); err != nil {
			log.Error(err, "Failed to create image update automation")
		}
	}

	// Get sync status
	syncStatus, err := c.FluxManager.GetSyncStatus(ctx, platform)
	if err != nil {
		log.Error(err, "Failed to get sync status")
	} else {
		// Update status based on sync
		phase := "Syncing"
		if syncStatus.Ready {
			phase = "Synced"
		}
		
		c.updateGitOpsStatus(ctx, platform, phase, syncStatus.Message)
		c.updateGitOpsRevision(ctx, platform, syncStatus.LastAppliedRevision)
	}

	// Schedule next reconciliation
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// handleDeletion handles cleanup when platform is deleted
func (c *GitOpsController) handleDeletion(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) (ctrl.Result, error) {
	log := c.Log.WithValues("platform", platform.Name)
	log.Info("Handling GitOps cleanup")

	if platform.Spec.GitOps == nil {
		return ctrl.Result{}, nil
	}

	gitOps := convertToGitOpsSpec(platform.Spec.GitOps)

	// Cleanup based on type
	switch gitOps.Type {
	case gitopsv1beta1.GitOpsArgoCD:
		if err := c.ArgoCDManager.DeleteApplication(ctx, platform); err != nil {
			log.Error(err, "Failed to delete ArgoCD resources")
			return ctrl.Result{}, err
		}
	case gitopsv1beta1.GitOpsFlux:
		if err := c.FluxManager.DeleteFluxResources(ctx, platform); err != nil {
			log.Error(err, "Failed to delete Flux resources")
			return ctrl.Result{}, err
		}
	}

	// Remove finalizer
	platform.SetFinalizers(removeString(platform.GetFinalizers(), "gitops.observability.io/finalizer"))
	if err := c.Update(ctx, platform); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// gitOpsEnabledPredicate returns a predicate that filters for GitOps-enabled platforms
func (c *GitOpsController) gitOpsEnabledPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if platform, ok := e.Object.(*observabilityv1.ObservabilityPlatform); ok {
				return platform.Spec.GitOps != nil
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if platform, ok := e.ObjectNew.(*observabilityv1.ObservabilityPlatform); ok {
				return platform.Spec.GitOps != nil
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			if platform, ok := e.Object.(*observabilityv1.ObservabilityPlatform); ok {
				return platform.Spec.GitOps != nil
			}
			return false
		},
	}
}

// watchGitOpsResources sets up watches for GitOps resources
func (c *GitOpsController) watchGitOpsResources(controller controller.Controller) error {
	// Watch ArgoCD Applications
	argoCDAppPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return c.isOurGitOpsResource(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return c.isOurGitOpsResource(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return c.isOurGitOpsResource(e.Object)
		},
	}

	// Watch Flux Kustomizations
	fluxKustomizationPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return c.isOurGitOpsResource(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return c.isOurGitOpsResource(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return c.isOurGitOpsResource(e.Object)
		},
	}

	// Note: In a real implementation, you would use the actual types for ArgoCD and Flux
	// These are placeholders as the actual CRDs would need to be imported
	
	return nil
}

// isOurGitOpsResource checks if a resource is managed by our operator
func (c *GitOpsController) isOurGitOpsResource(obj client.Object) bool {
	labels := obj.GetLabels()
	return labels["app.kubernetes.io/managed-by"] == "gunj-operator"
}

// startDriftDetectionWorker starts the drift detection background worker
func (c *GitOpsController) startDriftDetectionWorker(client client.Client) {
	ticker := time.NewTicker(5 * time.Minute) // Default check interval
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkAllPlatformsForDrift(context.Background())
		}
	}
}

// checkAllPlatformsForDrift checks all platforms for drift
func (c *GitOpsController) checkAllPlatformsForDrift(ctx context.Context) {
	// List all platforms with GitOps and drift detection enabled
	platformList := &observabilityv1.ObservabilityPlatformList{}
	if err := c.List(ctx, platformList); err != nil {
		c.Log.Error(err, "Failed to list platforms for drift detection")
		return
	}

	for _, platform := range platformList.Items {
		if platform.Spec.GitOps == nil {
			continue
		}

		gitOps := convertToGitOpsSpec(platform.Spec.GitOps)
		if gitOps.DriftDetection == nil || !gitOps.DriftDetection.Enabled {
			continue
		}

		// Check drift for this platform
		go c.checkPlatformDrift(ctx, &platform, gitOps)
	}
}

// checkPlatformDrift checks a single platform for drift
func (c *GitOpsController) checkPlatformDrift(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) {
	log := c.Log.WithValues("platform", platform.Name)
	log.Info("Checking for drift")

	// Get desired state from Git
	repo, _, err := c.GitManager.CloneRepository(ctx, gitOps.Repository)
	if err != nil {
		log.Error(err, "Failed to clone repository for drift detection")
		return
	}
	defer c.GitManager.Cleanup()

	// Get files from repository
	files, err := c.GitManager.GetFilesAtPath(repo, gitOps.Repository.Path)
	if err != nil {
		log.Error(err, "Failed to get files from repository")
		return
	}

	// Parse desired state
	desiredState, err := c.parseDesiredState(files)
	if err != nil {
		log.Error(err, "Failed to parse desired state")
		return
	}

	// Check drift
	report, err := c.DriftDetector.CheckDrift(ctx, platform, gitOps, desiredState)
	if err != nil {
		log.Error(err, "Failed to check drift")
		return
	}

	// Update drift status
	c.updateDriftStatus(ctx, platform, report)

	// Store drift history
	if err := c.DriftDetector.StoreDriftHistory(ctx, platform, report); err != nil {
		log.Error(err, "Failed to store drift history")
	}
}

// Webhook event processing methods

// ProcessPushEvent processes git push events
func (c *GitOpsController) ProcessPushEvent(ctx context.Context, event *webhook.PushEvent) error {
	log := c.Log.WithValues("event", "push", "repository", event.Repository)
	log.Info("Processing push event")

	// Find platforms using this repository
	platforms, err := c.findPlatformsByRepository(ctx, event.Repository)
	if err != nil {
		return err
	}

	// Trigger sync for each platform
	for _, platform := range platforms {
		gitOps := convertToGitOpsSpec(platform.Spec.GitOps)
		
		switch gitOps.Type {
		case gitopsv1beta1.GitOpsArgoCD:
			if err := c.ArgoCDManager.SyncApplication(ctx, &platform); err != nil {
				log.Error(err, "Failed to sync ArgoCD application", "platform", platform.Name)
			}
		case gitopsv1beta1.GitOpsFlux:
			if err := c.FluxManager.TriggerReconciliation(ctx, &platform); err != nil {
				log.Error(err, "Failed to trigger Flux reconciliation", "platform", platform.Name)
			}
		}
	}

	return nil
}

// ProcessPullRequestEvent processes pull request events
func (c *GitOpsController) ProcessPullRequestEvent(ctx context.Context, event *webhook.PullRequestEvent) error {
	log := c.Log.WithValues("event", "pull_request", "repository", event.Repository)
	log.Info("Processing pull request event")

	// This could trigger preview environments or PR validation
	// Implementation depends on specific requirements

	return nil
}

// ProcessTagEvent processes tag creation events
func (c *GitOpsController) ProcessTagEvent(ctx context.Context, event *webhook.TagEvent) error {
	log := c.Log.WithValues("event", "tag", "repository", event.Repository, "tag", event.Tag)
	log.Info("Processing tag event")

	// This could trigger automatic promotions based on tags
	// Implementation depends on specific requirements

	return nil
}

// Helper methods

// updateGitOpsStatus updates the GitOps status in the platform
func (c *GitOpsController) updateGitOpsStatus(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	phase string,
	message string,
) {
	// Update GitOps condition
	condition := metav1.Condition{
		Type:               "GitOpsSynced",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             phase,
		Message:            message,
	}

	if phase == "Failed" || phase == "OutOfSync" {
		condition.Status = metav1.ConditionFalse
	}

	// Update condition
	meta.SetStatusCondition(&platform.Status.Conditions, condition)
	
	if err := c.Status().Update(ctx, platform); err != nil {
		c.Log.Error(err, "Failed to update GitOps status")
	}
}

// updateGitOpsRevision updates the last synced revision
func (c *GitOpsController) updateGitOpsRevision(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	revision string,
) {
	if platform.Status.GitOps == nil {
		platform.Status.GitOps = &observabilityv1.GitOpsStatus{}
	}
	
	platform.Status.GitOps.LastSyncedRevision = revision
	platform.Status.GitOps.LastSyncTime = &metav1.Time{Time: time.Now()}
	
	if err := c.Status().Update(ctx, platform); err != nil {
		c.Log.Error(err, "Failed to update GitOps revision")
	}
}

// updateDriftStatus updates the drift detection status
func (c *GitOpsController) updateDriftStatus(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	report *drift.DriftReport,
) {
	if platform.Status.GitOps == nil {
		platform.Status.GitOps = &observabilityv1.GitOpsStatus{}
	}
	
	platform.Status.GitOps.DriftDetected = report.DriftDetected
	platform.Status.GitOps.LastDriftCheck = &metav1.Time{Time: report.CheckTime}
	
	if report.DriftDetected {
		platform.Status.GitOps.DriftedResources = len(report.DriftedResources)
	} else {
		platform.Status.GitOps.DriftedResources = 0
	}
	
	if err := c.Status().Update(ctx, platform); err != nil {
		c.Log.Error(err, "Failed to update drift status")
	}
}

// findPlatformsByRepository finds platforms using a specific repository
func (c *GitOpsController) findPlatformsByRepository(
	ctx context.Context,
	repository string,
) ([]observabilityv1.ObservabilityPlatform, error) {
	platformList := &observabilityv1.ObservabilityPlatformList{}
	if err := c.List(ctx, platformList); err != nil {
		return nil, err
	}

	var platforms []observabilityv1.ObservabilityPlatform
	for _, platform := range platformList.Items {
		if platform.Spec.GitOps != nil && platform.Spec.GitOps.Repository.URL == repository {
			platforms = append(platforms, platform)
		}
	}

	return platforms, nil
}

// parseDesiredState parses Kubernetes manifests from files
func (c *GitOpsController) parseDesiredState(files map[string][]byte) (map[string]*unstructured.Unstructured, error) {
	// This is a simplified implementation
	// In a real implementation, you would:
	// 1. Parse YAML/JSON files
	// 2. Handle Kustomize overlays
	// 3. Process Helm templates
	// 4. Apply any transformations
	
	desiredState := make(map[string]*unstructured.Unstructured)
	
	// Placeholder implementation
	return desiredState, nil
}

// convertToGitOpsSpec converts from API types to internal types
func convertToGitOpsSpec(apiGitOps *observabilityv1.GitOpsSpec) *gitopsv1beta1.GitOpsIntegrationSpec {
	// This is a placeholder - in a real implementation, you would properly convert between types
	return &gitopsv1beta1.GitOpsIntegrationSpec{
		Type: gitopsv1beta1.GitOpsIntegrationType(apiGitOps.Type),
		Repository: gitopsv1beta1.GitRepositorySpec{
			URL:    apiGitOps.Repository.URL,
			Branch: apiGitOps.Repository.Branch,
			Path:   apiGitOps.Repository.Path,
		},
		// Add other field mappings as needed
	}
}

// Utility functions

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// Import this package to use the SetStatusCondition function
var meta = struct {
	SetStatusCondition func(conditions *[]metav1.Condition, newCondition metav1.Condition)
}{
	SetStatusCondition: func(conditions *[]metav1.Condition, newCondition metav1.Condition) {
		// Simple implementation - in production use apimachinery/pkg/api/meta
		for i, c := range *conditions {
			if c.Type == newCondition.Type {
				(*conditions)[i] = newCondition
				return
			}
		}
		*conditions = append(*conditions, newCondition)
	},
}
