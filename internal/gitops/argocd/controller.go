// Package argocd provides ArgoCD integration for GitOps
package argocd

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
	// ArgoCDNamespace is the default ArgoCD namespace
	ArgoCDNamespace = "argocd"
	
	// ApplicationGVK defines the ArgoCD Application GVK
	ApplicationGroup   = "argoproj.io"
	ApplicationVersion = "v1alpha1"
	ApplicationKind    = "Application"
	
	// AppProjectKind defines the ArgoCD AppProject kind
	AppProjectKind = "AppProject"
)

var (
	// applicationGVR defines the ArgoCD Application GVR
	applicationGVR = schema.GroupVersionResource{
		Group:    ApplicationGroup,
		Version:  ApplicationVersion,
		Resource: "applications",
	}
	
	// appProjectGVR defines the ArgoCD AppProject GVR
	appProjectGVR = schema.GroupVersionResource{
		Group:    ApplicationGroup,
		Version:  ApplicationVersion,
		Resource: "appprojects",
	}
)

// Controller implements GitOps operations using ArgoCD
type Controller struct {
	client       client.Client
	scheme       *runtime.Scheme
	synchronizer gitops.GitSynchronizer
	log          logr.Logger
}

// NewController creates a new ArgoCD controller
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
		log:          log.WithName("argocd-controller"),
	}, nil
}

// Reconcile reconciles ArgoCD resources for the platform
func (c *Controller) Reconcile(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := c.log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	
	// Ensure ArgoCD is installed
	if err := c.ensureArgoCDInstalled(ctx); err != nil {
		return fmt.Errorf("ensuring ArgoCD is installed: %w", err)
	}
	
	// Create or update AppProject
	if err := c.reconcileAppProject(ctx, platform); err != nil {
		return fmt.Errorf("reconciling AppProject: %w", err)
	}
	
	// Create or update Application
	if err := c.reconcileApplication(ctx, platform); err != nil {
		return fmt.Errorf("reconciling Application: %w", err)
	}
	
	// Configure RBAC if needed
	if err := c.reconcileRBAC(ctx, platform); err != nil {
		return fmt.Errorf("reconciling RBAC: %w", err)
	}
	
	log.Info("ArgoCD resources reconciled successfully")
	return nil
}

// Sync synchronizes the platform using ArgoCD
func (c *Controller) Sync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := c.log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	
	// Get the Application
	app, err := c.getApplication(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting Application: %w", err)
	}
	
	// Trigger sync
	if err := c.triggerSync(ctx, app); err != nil {
		return fmt.Errorf("triggering sync: %w", err)
	}
	
	// Wait for sync to complete
	if err := c.waitForSync(ctx, app, 30*time.Minute); err != nil {
		return fmt.Errorf("waiting for sync: %w", err)
	}
	
	log.Info("ArgoCD sync completed successfully")
	return nil
}

// GetStatus returns the current GitOps status from ArgoCD
func (c *Controller) GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*gitops.GitOpsStatus, error) {
	// Get the Application
	app, err := c.getApplication(ctx, platform)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &gitops.GitOpsStatus{
				Provider:   gitops.ProviderArgoCD,
				SyncStatus: gitops.SyncStatusUnknown,
			}, nil
		}
		return nil, fmt.Errorf("getting Application: %w", err)
	}
	
	// Extract status from Application
	status := &gitops.GitOpsStatus{
		Provider: gitops.ProviderArgoCD,
	}
	
	// Get sync status
	syncStatus, found, err := unstructured.NestedString(app.Object, "status", "sync", "status")
	if err != nil || !found {
		status.SyncStatus = gitops.SyncStatusUnknown
	} else {
		status.SyncStatus = c.mapArgoCDSyncStatus(syncStatus)
	}
	
	// Get last sync time
	lastSyncTime, found, err := unstructured.NestedString(app.Object, "status", "operationState", "finishedAt")
	if err == nil && found {
		if t, err := time.Parse(time.RFC3339, lastSyncTime); err == nil {
			status.LastSyncTime = &metav1.Time{Time: t}
		}
	}
	
	// Get revision
	revision, found, err := unstructured.NestedString(app.Object, "status", "sync", "revision")
	if err == nil && found {
		status.LastSyncRevision = revision
	}
	
	// Get conditions
	conditions, found, err := unstructured.NestedSlice(app.Object, "status", "conditions")
	if err == nil && found {
		status.Conditions = c.mapArgoCDConditions(conditions)
	}
	
	return status, nil
}

// Rollback performs a rollback using ArgoCD
func (c *Controller) Rollback(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string) error {
	log := c.log.WithValues("platform", platform.Name, "namespace", platform.Namespace, "revision", revision)
	
	// Get the Application
	app, err := c.getApplication(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting Application: %w", err)
	}
	
	// Update Application to target revision
	if err := unstructured.SetNestedField(app.Object, revision, "spec", "source", "targetRevision"); err != nil {
		return fmt.Errorf("setting target revision: %w", err)
	}
	
	// Update the Application
	if err := c.client.Update(ctx, app); err != nil {
		return fmt.Errorf("updating Application: %w", err)
	}
	
	// Trigger sync
	if err := c.triggerSync(ctx, app); err != nil {
		return fmt.Errorf("triggering sync: %w", err)
	}
	
	log.Info("ArgoCD rollback initiated")
	return nil
}

// Promote promotes the platform to a new environment
func (c *Controller) Promote(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv string) error {
	// ArgoCD promotion typically involves updating the target branch or path
	// This would be handled by the promotion manager
	return fmt.Errorf("promotion should be handled by promotion manager")
}

// ensureArgoCDInstalled checks if ArgoCD is installed
func (c *Controller) ensureArgoCDInstalled(ctx context.Context) error {
	// Check if ArgoCD namespace exists
	namespace := &corev1.Namespace{}
	err := c.client.Get(ctx, types.NamespacedName{Name: ArgoCDNamespace}, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("ArgoCD namespace not found, please install ArgoCD")
		}
		return fmt.Errorf("checking ArgoCD namespace: %w", err)
	}
	
	// Check if ArgoCD CRDs are installed
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   ApplicationGroup,
		Version: ApplicationVersion,
		Kind:    ApplicationKind,
	})
	
	// Try to list Applications to verify CRDs are installed
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   ApplicationGroup,
		Version: ApplicationVersion,
		Kind:    ApplicationKind + "List",
	})
	
	if err := c.client.List(ctx, list, client.InNamespace(ArgoCDNamespace), client.Limit(1)); err != nil {
		return fmt.Errorf("ArgoCD CRDs not found, please install ArgoCD: %w", err)
	}
	
	return nil
}

// reconcileAppProject creates or updates the ArgoCD AppProject
func (c *Controller) reconcileAppProject(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	project := c.buildAppProject(platform)
	
	// Set owner reference
	if err := controllerutil.SetControllerReference(platform, project, c.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}
	
	// Create or update the AppProject
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(project.GroupVersionKind())
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      project.GetName(),
		Namespace: project.GetNamespace(),
	}, found)
	
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new AppProject
			if err := c.client.Create(ctx, project); err != nil {
				return fmt.Errorf("creating AppProject: %w", err)
			}
			c.log.Info("Created ArgoCD AppProject", "name", project.GetName())
			return nil
		}
		return fmt.Errorf("getting AppProject: %w", err)
	}
	
	// Update existing AppProject
	found.Object["spec"] = project.Object["spec"]
	if err := c.client.Update(ctx, found); err != nil {
		return fmt.Errorf("updating AppProject: %w", err)
	}
	
	return nil
}

// reconcileApplication creates or updates the ArgoCD Application
func (c *Controller) reconcileApplication(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	app := c.buildApplication(platform)
	
	// Set owner reference
	if err := controllerutil.SetControllerReference(platform, app, c.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}
	
	// Create or update the Application
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(app.GroupVersionKind())
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}, found)
	
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create new Application
			if err := c.client.Create(ctx, app); err != nil {
				return fmt.Errorf("creating Application: %w", err)
			}
			c.log.Info("Created ArgoCD Application", "name", app.GetName())
			return nil
		}
		return fmt.Errorf("getting Application: %w", err)
	}
	
	// Update existing Application
	// Preserve status
	app.Object["status"] = found.Object["status"]
	if err := c.client.Update(ctx, app); err != nil {
		return fmt.Errorf("updating Application: %w", err)
	}
	
	return nil
}

// reconcileRBAC creates or updates RBAC for ArgoCD
func (c *Controller) reconcileRBAC(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	// TODO: Implement RBAC configuration for ArgoCD
	// This would include creating ServiceAccounts, Roles, and RoleBindings
	// for fine-grained access control
	return nil
}

// buildAppProject builds an ArgoCD AppProject for the platform
func (c *Controller) buildAppProject(platform *observabilityv1.ObservabilityPlatform) *unstructured.Unstructured {
	project := &unstructured.Unstructured{}
	project.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   ApplicationGroup,
		Version: ApplicationVersion,
		Kind:    AppProjectKind,
	})
	project.SetName(fmt.Sprintf("gunj-%s", platform.Name))
	project.SetNamespace(ArgoCDNamespace)
	
	// Set spec
	spec := map[string]interface{}{
		"description": fmt.Sprintf("ArgoCD project for observability platform %s", platform.Name),
		"sourceRepos": []string{
			platform.Spec.GitOps.Repository.URL,
		},
		"destinations": []map[string]interface{}{
			{
				"namespace": platform.Namespace,
				"server":    "https://kubernetes.default.svc",
			},
		},
		"clusterResourceWhitelist": []map[string]interface{}{
			{
				"group": "*",
				"kind":  "*",
			},
		},
		"namespaceResourceWhitelist": []map[string]interface{}{
			{
				"group": "*",
				"kind":  "*",
			},
		},
		"roles": c.buildProjectRoles(platform),
	}
	
	_ = unstructured.SetNestedMap(project.Object, spec, "spec")
	
	return project
}

// buildApplication builds an ArgoCD Application for the platform
func (c *Controller) buildApplication(platform *observabilityv1.ObservabilityPlatform) *unstructured.Unstructured {
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   ApplicationGroup,
		Version: ApplicationVersion,
		Kind:    ApplicationKind,
	})
	app.SetName(fmt.Sprintf("gunj-%s", platform.Name))
	app.SetNamespace(ArgoCDNamespace)
	
	// Build source
	source := map[string]interface{}{
		"repoURL":        platform.Spec.GitOps.Repository.URL,
		"targetRevision": platform.Spec.GitOps.Repository.Branch,
		"path":           platform.Spec.GitOps.Repository.Path,
	}
	
	// Add Helm values if using Helm
	if platform.Spec.GitOps.Repository.Path != "" {
		// Assume Helm if path contains "chart"
		if contains(platform.Spec.GitOps.Repository.Path, "chart") {
			source["helm"] = map[string]interface{}{
				"valueFiles": []string{"values.yaml"},
			}
		}
	}
	
	// Build destination
	destination := map[string]interface{}{
		"server":    "https://kubernetes.default.svc",
		"namespace": platform.Namespace,
	}
	
	// Build sync policy
	syncPolicy := map[string]interface{}{}
	if platform.Spec.GitOps.SyncPolicy.Automated {
		syncPolicy["automated"] = map[string]interface{}{
			"prune":    platform.Spec.GitOps.SyncPolicy.Prune,
			"selfHeal": platform.Spec.GitOps.SyncPolicy.SelfHeal,
		}
	}
	
	if platform.Spec.GitOps.SyncPolicy.Retry != nil {
		syncPolicy["retry"] = map[string]interface{}{
			"limit": platform.Spec.GitOps.SyncPolicy.Retry.Limit,
		}
		
		if platform.Spec.GitOps.SyncPolicy.Retry.Backoff != nil {
			syncPolicy["retry"].(map[string]interface{})["backoff"] = map[string]interface{}{
				"duration":    platform.Spec.GitOps.SyncPolicy.Retry.Backoff.Duration.Duration.String(),
				"factor":      platform.Spec.GitOps.SyncPolicy.Retry.Backoff.Factor,
				"maxDuration": platform.Spec.GitOps.SyncPolicy.Retry.Backoff.MaxDuration.Duration.String(),
			}
		}
	}
	
	// Set spec
	spec := map[string]interface{}{
		"project":     fmt.Sprintf("gunj-%s", platform.Name),
		"source":      source,
		"destination": destination,
		"syncPolicy":  syncPolicy,
	}
	
	// Add sync options if provided
	if len(platform.Spec.GitOps.SyncPolicy.SyncOptions) > 0 {
		spec["syncOptions"] = platform.Spec.GitOps.SyncPolicy.SyncOptions
	}
	
	_ = unstructured.SetNestedMap(app.Object, spec, "spec")
	
	// Add finalizers
	app.SetFinalizers([]string{"resources-finalizer.argocd.argoproj.io"})
	
	return app
}

// buildProjectRoles builds roles for the AppProject
func (c *Controller) buildProjectRoles(platform *observabilityv1.ObservabilityPlatform) []map[string]interface{} {
	// Create admin role
	adminRole := map[string]interface{}{
		"name": "admin",
		"policies": []string{
			fmt.Sprintf("p, proj:gunj-%s:admin, applications, *, gunj-%s/*, allow", platform.Name, platform.Name),
		},
		"groups": []string{
			"gunj-admins",
		},
	}
	
	// Create readonly role
	readonlyRole := map[string]interface{}{
		"name": "readonly",
		"policies": []string{
			fmt.Sprintf("p, proj:gunj-%s:readonly, applications, get, gunj-%s/*, allow", platform.Name, platform.Name),
		},
		"groups": []string{
			"gunj-developers",
		},
	}
	
	return []map[string]interface{}{adminRole, readonlyRole}
}

// getApplication gets the ArgoCD Application for the platform
func (c *Controller) getApplication(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*unstructured.Unstructured, error) {
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   ApplicationGroup,
		Version: ApplicationVersion,
		Kind:    ApplicationKind,
	})
	
	err := c.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("gunj-%s", platform.Name),
		Namespace: ArgoCDNamespace,
	}, app)
	
	return app, err
}

// triggerSync triggers a sync operation on the Application
func (c *Controller) triggerSync(ctx context.Context, app *unstructured.Unstructured) error {
	// Set operation to trigger sync
	operation := map[string]interface{}{
		"sync": map[string]interface{}{
			"revision": "HEAD",
			"prune":    true,
		},
	}
	
	if err := unstructured.SetNestedMap(app.Object, operation, "operation"); err != nil {
		return fmt.Errorf("setting operation: %w", err)
	}
	
	// Update the Application to trigger sync
	if err := c.client.Update(ctx, app); err != nil {
		return fmt.Errorf("updating Application to trigger sync: %w", err)
	}
	
	return nil
}

// waitForSync waits for sync operation to complete
func (c *Controller) waitForSync(ctx context.Context, app *unstructured.Unstructured, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		// Get latest Application state
		current := &unstructured.Unstructured{}
		current.SetGroupVersionKind(app.GroupVersionKind())
		if err := c.client.Get(ctx, types.NamespacedName{
			Name:      app.GetName(),
			Namespace: app.GetNamespace(),
		}, current); err != nil {
			return fmt.Errorf("getting Application: %w", err)
		}
		
		// Check operation state
		phase, found, err := unstructured.NestedString(current.Object, "status", "operationState", "phase")
		if err != nil {
			return fmt.Errorf("getting operation phase: %w", err)
		}
		
		if found {
			switch phase {
			case "Succeeded":
				return nil
			case "Failed", "Error":
				message, _, _ := unstructured.NestedString(current.Object, "status", "operationState", "message")
				return fmt.Errorf("sync failed: %s", message)
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
	
	return fmt.Errorf("sync timeout after %v", timeout)
}

// mapArgoCDSyncStatus maps ArgoCD sync status to GitOps sync status
func (c *Controller) mapArgoCDSyncStatus(status string) gitops.SyncStatus {
	switch status {
	case "Synced":
		return gitops.SyncStatusSynced
	case "OutOfSync":
		return gitops.SyncStatusOutOfSync
	case "Syncing":
		return gitops.SyncStatusSyncing
	default:
		return gitops.SyncStatusUnknown
	}
}

// mapArgoCDConditions maps ArgoCD conditions to Kubernetes conditions
func (c *Controller) mapArgoCDConditions(conditions []interface{}) []metav1.Condition {
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
