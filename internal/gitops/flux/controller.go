// Package flux provides Flux integration for GitOps
package flux

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/gitops"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// FluxNamespace is the default Flux namespace
	FluxNamespace = "flux-system"
	
	// Source API Group
	SourceGroup   = "source.toolkit.fluxcd.io"
	SourceVersion = "v1"
	
	// Kustomize API Group
	KustomizeGroup   = "kustomize.toolkit.fluxcd.io"
	KustomizeVersion = "v1"
	
	// Helm API Group
	HelmGroup   = "helm.toolkit.fluxcd.io"
	HelmVersion = "v2beta2"
)

var (
	// gitRepositoryGVR defines the GitRepository GVR
	gitRepositoryGVR = schema.GroupVersionResource{
		Group:    SourceGroup,
		Version:  SourceVersion,
		Resource: "gitrepositories",
	}
	
	// kustomizationGVR defines the Kustomization GVR
	kustomizationGVR = schema.GroupVersionResource{
		Group:    KustomizeGroup,
		Version:  KustomizeVersion,
		Resource: "kustomizations",
	}
	
	// helmReleaseGVR defines the HelmRelease GVR
	helmReleaseGVR = schema.GroupVersionResource{
		Group:    HelmGroup,
		Version:  HelmVersion,
		Resource: "helmreleases",
	}
)

// Controller implements GitOps operations using Flux
type Controller struct {
	client       client.Client
	scheme       *runtime.Scheme
	synchronizer gitops.GitSynchronizer
	log          logr.Logger
}

// NewController creates a new Flux controller
func NewController(
	client client.Client,
	scheme *runtime.Scheme,
	synchronizer gitops.GitSynchronizer,
	log logr.Logger,
) (*Controller, error) {
	return &Controller{
		client:       client,
		scheme:       scheme,
		synchronizer: synchronizer,
		log:          log.WithName("flux-controller"),
	}, nil
}

// Reconcile reconciles Flux resources for the platform
func (c *Controller) Reconcile(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := c.log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	
	// Ensure Flux is installed
	if err := c.ensureFluxInstalled(ctx); err != nil {
		return fmt.Errorf("ensuring Flux is installed: %w", err)
	}
	
	// Create or update GitRepository
	if err := c.reconcileGitRepository(ctx, platform); err != nil {
		return fmt.Errorf("reconciling GitRepository: %w", err)
	}
	
	// Create or update Kustomization or HelmRelease
	if err := c.reconcileDeployment(ctx, platform); err != nil {
		return fmt.Errorf("reconciling deployment: %w", err)
	}
	
	// Configure image automation if needed
	if err := c.reconcileImageAutomation(ctx, platform); err != nil {
		return fmt.Errorf("reconciling image automation: %w", err)
	}
	
	// Configure notifications
	if err := c.reconcileNotifications(ctx, platform); err != nil {
		return fmt.Errorf("reconciling notifications: %w", err)
	}
	
	log.Info("Flux resources reconciled successfully")
	return nil
}

// Sync synchronizes the platform using Flux
func (c *Controller) Sync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := c.log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	
	// Reconcile GitRepository to trigger source update
	gitRepo, err := c.getGitRepository(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting GitRepository: %w", err)
	}
	
	// Annotate to trigger reconciliation
	if gitRepo.GetAnnotations() == nil {
		gitRepo.SetAnnotations(make(map[string]string))
	}
	gitRepo.GetAnnotations()["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
	
	if err := c.client.Update(ctx, gitRepo); err != nil {
		return fmt.Errorf("updating GitRepository: %w", err)
	}
	
	// Wait for GitRepository to be ready
	if err := c.waitForGitRepositoryReady(ctx, gitRepo, 5*time.Minute); err != nil {
		return fmt.Errorf("waiting for GitRepository: %w", err)
	}
	
	// Reconcile Kustomization/HelmRelease to trigger sync
	deployment, err := c.getDeployment(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting deployment: %w", err)
	}
	
	// Annotate to trigger reconciliation
	if deployment.GetAnnotations() == nil {
		deployment.SetAnnotations(make(map[string]string))
	}
	deployment.GetAnnotations()["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
	
	if err := c.client.Update(ctx, deployment); err != nil {
		return fmt.Errorf("updating deployment: %w", err)
	}
	
	// Wait for deployment to be ready
	if err := c.waitForDeploymentReady(ctx, deployment, 30*time.Minute); err != nil {
		return fmt.Errorf("waiting for deployment: %w", err)
	}
	
	log.Info("Flux sync completed successfully")
	return nil
}

// GetStatus returns the current GitOps status from Flux
func (c *Controller) GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*gitops.GitOpsStatus, error) {
	status := &gitops.GitOpsStatus{
		Provider: gitops.ProviderFlux,
	}
	
	// Get GitRepository status
	gitRepo, err := c.getGitRepository(ctx, platform)
	if err != nil {
		if apierrors.IsNotFound(err) {
			status.SyncStatus = gitops.SyncStatusUnknown
			return status, nil
		}
		return nil, fmt.Errorf("getting GitRepository: %w", err)
	}
	
	// Get deployment status
	deployment, err := c.getDeployment(ctx, platform)
	if err != nil {
		if apierrors.IsNotFound(err) {
			status.SyncStatus = gitops.SyncStatusUnknown
			return status, nil
		}
		return nil, fmt.Errorf("getting deployment: %w", err)
	}
	
	// Extract sync status
	status.SyncStatus = c.extractSyncStatus(deployment)
	
	// Get last sync time
	lastApplied, found, err := unstructured.NestedString(deployment.Object, "status", "lastAppliedRevision")
	if err == nil && found {
		status.LastSyncRevision = lastApplied
	}
	
	// Get conditions
	conditions, found, err := unstructured.NestedSlice(deployment.Object, "status", "conditions")
	if err == nil && found {
		status.Conditions = c.mapFluxConditions(conditions)
		
		// Extract last sync time from Ready condition
		for _, cond := range status.Conditions {
			if cond.Type == "Ready" && cond.Status == metav1.ConditionTrue {
				status.LastSyncTime = &cond.LastTransitionTime
				break
			}
		}
	}
	
	return status, nil
}

// Rollback performs a rollback using Flux
func (c *Controller) Rollback(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string) error {
	log := c.log.WithValues("platform", platform.Name, "namespace", platform.Namespace, "revision", revision)
	
	// Get the GitRepository
	gitRepo, err := c.getGitRepository(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting GitRepository: %w", err)
	}
	
	// Update GitRepository to specific revision
	ref := map[string]interface{}{
		"commit": revision,
	}
	if err := unstructured.SetNestedMap(gitRepo.Object, ref, "spec", "ref"); err != nil {
		return fmt.Errorf("setting ref: %w", err)
	}
	
	// Update the GitRepository
	if err := c.client.Update(ctx, gitRepo); err != nil {
		return fmt.Errorf("updating GitRepository: %w", err)
	}
	
	// Trigger sync
	if err := c.Sync(ctx, platform); err != nil {
		return fmt.Errorf("triggering sync: %w", err)
	}
	
	log.Info("Flux rollback initiated")
	return nil
}

// Promote promotes the platform to a new environment
func (c *Controller) Promote(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv string) error {
	// Flux promotion typically involves updating the Git repository
	// This would be handled by the promotion manager
	return fmt.Errorf("promotion should be handled by promotion manager")
}

// ensureFluxInstalled checks if Flux is installed
func (c *Controller) ensureFluxInstalled(ctx context.Context) error {
	// Check if Flux namespace exists
	namespace := &corev1.Namespace{}
	err := c.client.Get(ctx, types.NamespacedName{Name: FluxNamespace}, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("Flux namespace not found, please install Flux")
		}
		return fmt.Errorf("checking Flux namespace: %w", err)
	}
	
	// Check if Flux CRDs are installed by trying to list GitRepositories
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   SourceGroup,
		Version: SourceVersion,
		Kind:    "GitRepositoryList",
	})
	
	if err := c.client.List(ctx, list, client.InNamespace(FluxNamespace), client.Limit(1)); err != nil {
		return fmt.Errorf("Flux CRDs not found, please install Flux: %w", err)
	}
	
	return nil
}

// reconcileGitRepository creates or updates the GitRepository
func (c *Controller) reconcileGitRepository(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	gitRepo := c.buildGitRepository(platform)
	
	// Set owner reference
	if err := controllerutil.SetControllerReference(platform, gitRepo, c.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}
	
	// Create or update the GitRepository
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(gitRepo.GroupVersionKind())
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      gitRepo.GetName(),
		Namespace: gitRepo.GetNamespace(),
	}, found)
	
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new GitRepository
			if err := c.client.Create(ctx, gitRepo); err != nil {
				return fmt.Errorf("creating GitRepository: %w", err)
			}
			c.log.Info("Created Flux GitRepository", "name", gitRepo.GetName())
			return nil
		}
		return fmt.Errorf("getting GitRepository: %w", err)
	}
	
	// Update existing GitRepository
	found.Object["spec"] = gitRepo.Object["spec"]
	if err := c.client.Update(ctx, found); err != nil {
		return fmt.Errorf("updating GitRepository: %w", err)
	}
	
	return nil
}

// reconcileDeployment creates or updates Kustomization or HelmRelease
func (c *Controller) reconcileDeployment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// Determine if using Kustomization or HelmRelease based on path
	if isHelmChart(platform.Spec.GitOps.Repository.Path) {
		return c.reconcileHelmRelease(ctx, platform)
	}
	return c.reconcileKustomization(ctx, platform)
}

// reconcileKustomization creates or updates the Kustomization
func (c *Controller) reconcileKustomization(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	kustomization := c.buildKustomization(platform)
	
	// Set owner reference
	if err := controllerutil.SetControllerReference(platform, kustomization, c.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}
	
	// Create or update the Kustomization
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(kustomization.GroupVersionKind())
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      kustomization.GetName(),
		Namespace: kustomization.GetNamespace(),
	}, found)
	
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new Kustomization
			if err := c.client.Create(ctx, kustomization); err != nil {
				return fmt.Errorf("creating Kustomization: %w", err)
			}
			c.log.Info("Created Flux Kustomization", "name", kustomization.GetName())
			return nil
		}
		return fmt.Errorf("getting Kustomization: %w", err)
	}
	
	// Update existing Kustomization
	found.Object["spec"] = kustomization.Object["spec"]
	if err := c.client.Update(ctx, found); err != nil {
		return fmt.Errorf("updating Kustomization: %w", err)
	}
	
	return nil
}

// reconcileHelmRelease creates or updates the HelmRelease
func (c *Controller) reconcileHelmRelease(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	helmRelease := c.buildHelmRelease(platform)
	
	// Set owner reference
	if err := controllerutil.SetControllerReference(platform, helmRelease, c.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}
	
	// Create or update the HelmRelease
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(helmRelease.GroupVersionKind())
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      helmRelease.GetName(),
		Namespace: helmRelease.GetNamespace(),
	}, found)
	
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new HelmRelease
			if err := c.client.Create(ctx, helmRelease); err != nil {
				return fmt.Errorf("creating HelmRelease: %w", err)
			}
			c.log.Info("Created Flux HelmRelease", "name", helmRelease.GetName())
			return nil
		}
		return fmt.Errorf("getting HelmRelease: %w", err)
	}
	
	// Update existing HelmRelease
	found.Object["spec"] = helmRelease.Object["spec"]
	if err := c.client.Update(ctx, found); err != nil {
		return fmt.Errorf("updating HelmRelease: %w", err)
	}
	
	return nil
}

// reconcileImageAutomation sets up image automation if configured
func (c *Controller) reconcileImageAutomation(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// TODO: Implement image automation for automatic updates
	// This would include:
	// - ImageRepository for scanning container registries
	// - ImagePolicy for selecting image tags
	// - ImageUpdateAutomation for updating Git repository
	return nil
}

// reconcileNotifications sets up notifications if configured
func (c *Controller) reconcileNotifications(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// TODO: Implement notification configuration
	// This would include:
	// - Provider for notification targets (Slack, Teams, etc.)
	// - Alert for defining alert rules
	return nil
}

// buildGitRepository builds a Flux GitRepository
func (c *Controller) buildGitRepository(platform *observabilityv1.ObservabilityPlatform) *unstructured.Unstructured {
	gitRepo := &unstructured.Unstructured{}
	gitRepo.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   SourceGroup,
		Version: SourceVersion,
		Kind:    "GitRepository",
	})
	gitRepo.SetName(fmt.Sprintf("gunj-%s", platform.Name))
	gitRepo.SetNamespace(platform.Namespace)
	
	// Build spec
	spec := map[string]interface{}{
		"url": platform.Spec.GitOps.Repository.URL,
		"ref": map[string]interface{}{
			"branch": platform.Spec.GitOps.Repository.Branch,
		},
	}
	
	// Set interval
	interval := platform.Spec.GitOps.Repository.Interval.Duration
	if interval == 0 {
		interval = 1 * time.Minute
	}
	spec["interval"] = interval.String()
	
	// Add secret reference if provided
	if platform.Spec.GitOps.Repository.SecretRef != nil {
		spec["secretRef"] = map[string]interface{}{
			"name": platform.Spec.GitOps.Repository.SecretRef.Name,
		}
	}
	
	_ = unstructured.SetNestedMap(gitRepo.Object, spec, "spec")
	
	return gitRepo
}

// buildKustomization builds a Flux Kustomization
func (c *Controller) buildKustomization(platform *observabilityv1.ObservabilityPlatform) *unstructured.Unstructured {
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   KustomizeGroup,
		Version: KustomizeVersion,
		Kind:    "Kustomization",
	})
	kustomization.SetName(fmt.Sprintf("gunj-%s", platform.Name))
	kustomization.SetNamespace(platform.Namespace)
	
	// Build spec
	spec := map[string]interface{}{
		"sourceRef": map[string]interface{}{
			"kind": "GitRepository",
			"name": fmt.Sprintf("gunj-%s", platform.Name),
		},
		"path":           platform.Spec.GitOps.Repository.Path,
		"targetNamespace": platform.Namespace,
		"prune":           platform.Spec.GitOps.SyncPolicy.Prune,
	}
	
	// Set interval
	interval := platform.Spec.GitOps.Repository.Interval.Duration
	if interval == 0 {
		interval = 10 * time.Minute
	}
	spec["interval"] = interval.String()
	
	// Add health checks
	spec["healthChecks"] = []map[string]interface{}{
		{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"name":       "prometheus",
			"namespace":  platform.Namespace,
		},
		{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"name":       "grafana",
			"namespace":  platform.Namespace,
		},
	}
	
	// Add retry configuration
	if platform.Spec.GitOps.SyncPolicy.Retry != nil {
		spec["retryInterval"] = "1m"
	}
	
	_ = unstructured.SetNestedMap(kustomization.Object, spec, "spec")
	
	return kustomization
}

// buildHelmRelease builds a Flux HelmRelease
func (c *Controller) buildHelmRelease(platform *observabilityv1.ObservabilityPlatform) *unstructured.Unstructured {
	helmRelease := &unstructured.Unstructured{}
	helmRelease.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   HelmGroup,
		Version: HelmVersion,
		Kind:    "HelmRelease",
	})
	helmRelease.SetName(fmt.Sprintf("gunj-%s", platform.Name))
	helmRelease.SetNamespace(platform.Namespace)
	
	// Build spec
	spec := map[string]interface{}{
		"chart": map[string]interface{}{
			"spec": map[string]interface{}{
				"sourceRef": map[string]interface{}{
					"kind": "GitRepository",
					"name": fmt.Sprintf("gunj-%s", platform.Name),
				},
				"chart": platform.Spec.GitOps.Repository.Path,
			},
		},
		"targetNamespace": platform.Namespace,
		"install": map[string]interface{}{
			"remediation": map[string]interface{}{
				"retries": 3,
			},
		},
		"upgrade": map[string]interface{}{
			"remediation": map[string]interface{}{
				"retries": 3,
			},
		},
		"rollback": map[string]interface{}{
			"enable": true,
		},
		"uninstall": map[string]interface{}{
			"keepHistory": false,
		},
	}
	
	// Set interval
	interval := platform.Spec.GitOps.Repository.Interval.Duration
	if interval == 0 {
		interval = 10 * time.Minute
	}
	spec["interval"] = interval.String()
	
	// Add values
	spec["values"] = map[string]interface{}{
		"platform": map[string]interface{}{
			"name":      platform.Name,
			"namespace": platform.Namespace,
		},
	}
	
	_ = unstructured.SetNestedMap(helmRelease.Object, spec, "spec")
	
	return helmRelease
}

// Helper methods

func (c *Controller) getGitRepository(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*unstructured.Unstructured, error) {
	gitRepo := &unstructured.Unstructured{}
	gitRepo.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   SourceGroup,
		Version: SourceVersion,
		Kind:    "GitRepository",
	})
	
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("gunj-%s", platform.Name),
		Namespace: platform.Namespace,
	}, gitRepo)
	
	return gitRepo, err
}

func (c *Controller) getDeployment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*unstructured.Unstructured, error) {
	if isHelmChart(platform.Spec.GitOps.Repository.Path) {
		return c.getHelmRelease(ctx, platform)
	}
	return c.getKustomization(ctx, platform)
}

func (c *Controller) getKustomization(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*unstructured.Unstructured, error) {
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   KustomizeGroup,
		Version: KustomizeVersion,
		Kind:    "Kustomization",
	})
	
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("gunj-%s", platform.Name),
		Namespace: platform.Namespace,
	}, kustomization)
	
	return kustomization, err
}

func (c *Controller) getHelmRelease(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*unstructured.Unstructured, error) {
	helmRelease := &unstructured.Unstructured{}
	helmRelease.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   HelmGroup,
		Version: HelmVersion,
		Kind:    "HelmRelease",
	})
	
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("gunj-%s", platform.Name),
		Namespace: platform.Namespace,
	}, helmRelease)
	
	return helmRelease, err
}

func (c *Controller) waitForGitRepositoryReady(ctx context.Context, gitRepo *unstructured.Unstructured, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		// Get latest state
		current := &unstructured.Unstructured{}
		current.SetGroupVersionKind(gitRepo.GroupVersionKind())
		if err := c.client.Get(ctx, types.NamespacedName{
			Name:      gitRepo.GetName(),
			Namespace: gitRepo.GetNamespace(),
		}, current); err != nil {
			return fmt.Errorf("getting GitRepository: %w", err)
		}
		
		// Check ready condition
		conditions, found, err := unstructured.NestedSlice(current.Object, "status", "conditions")
		if err == nil && found {
			for _, cond := range conditions {
				if condMap, ok := cond.(map[string]interface{}); ok {
					if condMap["type"] == "Ready" && condMap["status"] == "True" {
						return nil
					}
				}
			}
		}
		
		// Wait before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue checking
		}
	}
	
	return fmt.Errorf("GitRepository not ready after %v", timeout)
}

func (c *Controller) waitForDeploymentReady(ctx context.Context, deployment *unstructured.Unstructured, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		// Get latest state
		current := &unstructured.Unstructured{}
		current.SetGroupVersionKind(deployment.GroupVersionKind())
		if err := c.client.Get(ctx, types.NamespacedName{
			Name:      deployment.GetName(),
			Namespace: deployment.GetNamespace(),
		}, current); err != nil {
			return fmt.Errorf("getting deployment: %w", err)
		}
		
		// Check ready condition
		conditions, found, err := unstructured.NestedSlice(current.Object, "status", "conditions")
		if err == nil && found {
			for _, cond := range conditions {
				if condMap, ok := cond.(map[string]interface{}); ok {
					if condMap["type"] == "Ready" && condMap["status"] == "True" {
						return nil
					}
				}
			}
		}
		
		// Wait before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue checking
		}
	}
	
	return fmt.Errorf("deployment not ready after %v", timeout)
}

func (c *Controller) extractSyncStatus(deployment *unstructured.Unstructured) gitops.SyncStatus {
	// Check ready condition
	conditions, found, err := unstructured.NestedSlice(deployment.Object, "status", "conditions")
	if err != nil || !found {
		return gitops.SyncStatusUnknown
	}
	
	for _, cond := range conditions {
		if condMap, ok := cond.(map[string]interface{}); ok {
			if condMap["type"] == "Ready" {
				if condMap["status"] == "True" {
					return gitops.SyncStatusSynced
				} else if condMap["reason"] == "Progressing" {
					return gitops.SyncStatusSyncing
				} else {
					return gitops.SyncStatusOutOfSync
				}
			}
		}
	}
	
	return gitops.SyncStatusUnknown
}

func (c *Controller) mapFluxConditions(conditions []interface{}) []metav1.Condition {
	var result []metav1.Condition
	
	for _, cond := range conditions {
		if condMap, ok := cond.(map[string]interface{}); ok {
			condition := metav1.Condition{
				Type:               getStringValue(condMap, "type"),
				Status:             metav1.ConditionStatus(getStringValue(condMap, "status")),
				Reason:             getStringValue(condMap, "reason"),
				Message:            getStringValue(condMap, "message"),
				LastTransitionTime: metav1.Now(),
			}
			
			// Parse last transition time if available
			if lastTransition := getStringValue(condMap, "lastTransitionTime"); lastTransition != "" {
				if t, err := time.Parse(time.RFC3339, lastTransition); err == nil {
					condition.LastTransitionTime = metav1.Time{Time: t}
				}
			}
			
			result = append(result, condition)
		}
	}
	
	return result
}

// Helper functions

func isHelmChart(path string) bool {
	// Simple heuristic: if path contains "chart" or "helm", assume it's a Helm chart
	return contains(path, "chart") || contains(path, "helm")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
