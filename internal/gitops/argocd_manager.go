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
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// ArgoCDManager manages ArgoCD applications
type ArgoCDManager struct {
	client client.Client
	log    logr.Logger
}

// NewArgoCDManager creates a new ArgoCD manager
func NewArgoCDManager(client client.Client, log logr.Logger) *ArgoCDManager {
	return &ArgoCDManager{
		client: client,
		log:    log.WithName("argocd-manager"),
	}
}

// ArgoCDAppConfig represents ArgoCD application configuration
type ArgoCDAppConfig struct {
	Name                 string
	Project              string
	RepoURL              string
	Path                 string
	TargetRevision       string
	DestinationServer    string
	DestinationNamespace string
	SyncOptions          []string
	AutoSync             bool
	Values               map[string]string
	IgnoreDifferences    []observabilityv1beta1.ResourceIgnoreDifferences
	RetryPolicy          *observabilityv1beta1.ArgoCDRetryPolicy
}

// CreateOrUpdateApplication creates or updates an ArgoCD application
func (m *ArgoCDManager) CreateOrUpdateApplication(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, config *ArgoCDAppConfig) error {
	log := m.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name), "app", config.Name)

	// Create ArgoCD Application object
	app := m.buildApplication(deployment, config)

	// Check if application exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := m.client.Get(ctx, types.NamespacedName{
		Name:      config.Name,
		Namespace: getArgoCDNamespace(),
	}, existing)

	if err == nil {
		// Update existing application
		app.SetResourceVersion(existing.GetResourceVersion())
		if err := m.client.Update(ctx, app); err != nil {
			return fmt.Errorf("failed to update ArgoCD application: %w", err)
		}
		log.Info("ArgoCD application updated successfully")
	} else if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	} else {
		// Create new application
		if err := m.client.Create(ctx, app); err != nil {
			return fmt.Errorf("failed to create ArgoCD application: %w", err)
		}
		log.Info("ArgoCD application created successfully")
	}

	return nil
}

// buildApplication builds an ArgoCD Application object
func (m *ArgoCDManager) buildApplication(deployment *observabilityv1beta1.GitOpsDeployment, config *ArgoCDAppConfig) *unstructured.Unstructured {
	app := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name":      config.Name,
				"namespace": getArgoCDNamespace(),
				"labels": map[string]interface{}{
					"observability.io/gitops-deployment": deployment.Name,
					"observability.io/managed-by":        "gunj-operator",
				},
				"annotations": map[string]interface{}{
					"observability.io/deployment-namespace": deployment.Namespace,
				},
				"finalizers": []interface{}{
					"resources-finalizer.argocd.argoproj.io",
				},
			},
			"spec": m.buildApplicationSpec(config),
		},
	}

	return app
}

// buildApplicationSpec builds the ArgoCD Application spec
func (m *ArgoCDManager) buildApplicationSpec(config *ArgoCDAppConfig) map[string]interface{} {
	spec := map[string]interface{}{
		"project": config.Project,
		"source": map[string]interface{}{
			"repoURL":        config.RepoURL,
			"path":           config.Path,
			"targetRevision": config.TargetRevision,
		},
		"destination": map[string]interface{}{
			"server":    config.DestinationServer,
			"namespace": config.DestinationNamespace,
		},
	}

	// Add Helm values if provided
	if len(config.Values) > 0 {
		helmValues := ""
		for k, v := range config.Values {
			helmValues += fmt.Sprintf("%s: %s\n", k, v)
		}
		spec["source"].(map[string]interface{})["helm"] = map[string]interface{}{
			"values": helmValues,
		}
	}

	// Add sync policy
	syncPolicy := map[string]interface{}{}
	
	if config.AutoSync {
		syncPolicy["automated"] = map[string]interface{}{
			"prune":    true,
			"selfHeal": true,
		}
	}

	if len(config.SyncOptions) > 0 {
		syncPolicy["syncOptions"] = config.SyncOptions
	}

	if config.RetryPolicy != nil {
		retry := map[string]interface{}{}
		if config.RetryPolicy.Limit != nil {
			retry["limit"] = *config.RetryPolicy.Limit
		}
		if config.RetryPolicy.Backoff != nil {
			backoff := map[string]interface{}{}
			if config.RetryPolicy.Backoff.Duration != "" {
				backoff["duration"] = config.RetryPolicy.Backoff.Duration
			}
			if config.RetryPolicy.Backoff.Factor != nil {
				backoff["factor"] = *config.RetryPolicy.Backoff.Factor
			}
			if config.RetryPolicy.Backoff.MaxDuration != "" {
				backoff["maxDuration"] = config.RetryPolicy.Backoff.MaxDuration
			}
			retry["backoff"] = backoff
		}
		syncPolicy["retry"] = retry
	}

	if len(syncPolicy) > 0 {
		spec["syncPolicy"] = syncPolicy
	}

	// Add ignore differences
	if len(config.IgnoreDifferences) > 0 {
		ignoreDiffs := make([]interface{}, 0, len(config.IgnoreDifferences))
		for _, diff := range config.IgnoreDifferences {
			ignoreDiff := map[string]interface{}{
				"kind": diff.Kind,
			}
			if diff.Group != "" {
				ignoreDiff["group"] = diff.Group
			}
			if diff.Name != "" {
				ignoreDiff["name"] = diff.Name
			}
			if len(diff.JSONPointers) > 0 {
				ignoreDiff["jsonPointers"] = diff.JSONPointers
			}
			if len(diff.JQPathExpressions) > 0 {
				ignoreDiff["jqPathExpressions"] = diff.JQPathExpressions
			}
			ignoreDiffs = append(ignoreDiffs, ignoreDiff)
		}
		spec["ignoreDifferences"] = ignoreDiffs
	}

	return spec
}

// GetSyncStatus gets the sync status of ArgoCD applications
func (m *ArgoCDManager) GetSyncStatus(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.SyncStatus, error) {
	// List all applications for this deployment
	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	if err := m.client.List(ctx, appList, client.InNamespace(getArgoCDNamespace()),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return nil, fmt.Errorf("failed to list ArgoCD applications: %w", err)
	}

	// Aggregate sync status
	allSynced := true
	syncMessage := ""
	var lastRevision string

	for _, app := range appList.Items {
		status, found, err := unstructured.NestedMap(app.Object, "status")
		if err != nil || !found {
			continue
		}

		// Get sync status
		sync, found, err := unstructured.NestedMap(status, "sync")
		if err != nil || !found {
			allSynced = false
			continue
		}

		syncStatus, found, err := unstructured.NestedString(sync, "status")
		if err != nil || !found || syncStatus != "Synced" {
			allSynced = false
			if message, found, _ := unstructured.NestedString(sync, "message"); found {
				syncMessage = message
			}
		}

		// Get revision
		if revision, found, err := unstructured.NestedString(sync, "revision"); err == nil && found {
			lastRevision = revision
		}
	}

	status := "Synced"
	if !allSynced {
		status = "OutOfSync"
	}

	return &observabilityv1beta1.SyncStatus{
		Status:   status,
		Revision: lastRevision,
		Message:  syncMessage,
	}, nil
}

// GetHealthStatus gets the health status of ArgoCD applications
func (m *ArgoCDManager) GetHealthStatus(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.HealthStatus, error) {
	// List all applications for this deployment
	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	if err := m.client.List(ctx, appList, client.InNamespace(getArgoCDNamespace()),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return nil, fmt.Errorf("failed to list ArgoCD applications: %w", err)
	}

	// Aggregate health status
	overallHealth := "Healthy"
	healthMessage := ""
	degradedCount := 0
	progressingCount := 0

	for _, app := range appList.Items {
		status, found, err := unstructured.NestedMap(app.Object, "status")
		if err != nil || !found {
			continue
		}

		// Get health status
		health, found, err := unstructured.NestedMap(status, "health")
		if err != nil || !found {
			continue
		}

		healthStatus, found, err := unstructured.NestedString(health, "status")
		if err != nil || !found {
			continue
		}

		switch healthStatus {
		case "Degraded":
			degradedCount++
			overallHealth = "Degraded"
			if message, found, _ := unstructured.NestedString(health, "message"); found {
				healthMessage = message
			}
		case "Progressing":
			progressingCount++
			if overallHealth != "Degraded" {
				overallHealth = "Progressing"
			}
		case "Suspended", "Missing":
			overallHealth = healthStatus
			if message, found, _ := unstructured.NestedString(health, "message"); found {
				healthMessage = message
			}
		}
	}

	// Build message
	if healthMessage == "" {
		if degradedCount > 0 {
			healthMessage = fmt.Sprintf("%d applications are degraded", degradedCount)
		} else if progressingCount > 0 {
			healthMessage = fmt.Sprintf("%d applications are progressing", progressingCount)
		}
	}

	return &observabilityv1beta1.HealthStatus{
		Status:  overallHealth,
		Message: healthMessage,
	}, nil
}

// DeleteApplications deletes all ArgoCD applications for a deployment
func (m *ArgoCDManager) DeleteApplications(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// List all applications for this deployment
	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	if err := m.client.List(ctx, appList, client.InNamespace(getArgoCDNamespace()),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return fmt.Errorf("failed to list ArgoCD applications: %w", err)
	}

	// Delete each application
	for _, app := range appList.Items {
		log := m.log.WithValues("application", app.GetName())
		
		// Remove finalizer to allow deletion
		finalizers := app.GetFinalizers()
		var newFinalizers []string
		for _, f := range finalizers {
			if f != "resources-finalizer.argocd.argoproj.io" {
				newFinalizers = append(newFinalizers, f)
			}
		}
		app.SetFinalizers(newFinalizers)
		
		if err := m.client.Update(ctx, &app); err != nil {
			log.Error(err, "Failed to remove finalizer")
		}

		if err := m.client.Delete(ctx, &app); err != nil {
			log.Error(err, "Failed to delete ArgoCD application")
			continue
		}
		log.Info("ArgoCD application deleted")
	}

	return nil
}

// SyncApplication triggers a sync for an ArgoCD application
func (m *ArgoCDManager) SyncApplication(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, appName string) error {
	// This would typically use the ArgoCD API to trigger a sync
	// For now, we'll update the annotation to trigger ArgoCD's auto-sync
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      appName,
		Namespace: getArgoCDNamespace(),
	}, app); err != nil {
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Update sync annotation
	annotations := app.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["argocd.argoproj.io/sync-operation"] = fmt.Sprintf(`{"revision":"%s"}`, deployment.Status.LastSyncRevision)
	app.SetAnnotations(annotations)

	if err := m.client.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update ArgoCD application: %w", err)
	}

	return nil
}

// GetApplicationResources gets the resources managed by an ArgoCD application
func (m *ArgoCDManager) GetApplicationResources(ctx context.Context, appName string) ([]observabilityv1beta1.ResourceStatus, error) {
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      appName,
		Namespace: getArgoCDNamespace(),
	}, app); err != nil {
		return nil, fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	status, found, err := unstructured.NestedMap(app.Object, "status")
	if err != nil || !found {
		return nil, fmt.Errorf("application status not found")
	}

	resourcesRaw, found, err := unstructured.NestedSlice(status, "resources")
	if err != nil || !found {
		return []observabilityv1beta1.ResourceStatus{}, nil
	}

	var resources []observabilityv1beta1.ResourceStatus
	for _, r := range resourcesRaw {
		resource, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		rs := observabilityv1beta1.ResourceStatus{
			Group:     getStringValue(resource, "group"),
			Version:   getStringValue(resource, "version"),
			Kind:      getStringValue(resource, "kind"),
			Name:      getStringValue(resource, "name"),
			Namespace: getStringValue(resource, "namespace"),
			Status:    getStringValue(resource, "status"),
		}

		// Get health status if available
		if health, found := resource["health"].(map[string]interface{}); found {
			rs.Health = getStringValue(health, "status")
		}

		resources = append(resources, rs)
	}

	return resources, nil
}

// UpdateApplicationSpec updates the spec of an ArgoCD application
func (m *ArgoCDManager) UpdateApplicationSpec(ctx context.Context, appName string, updateFunc func(spec map[string]interface{}) error) error {
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      appName,
		Namespace: getArgoCDNamespace(),
	}, app); err != nil {
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	spec, found, err := unstructured.NestedMap(app.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("application spec not found")
	}

	if err := updateFunc(spec); err != nil {
		return err
	}

	if err := unstructured.SetNestedField(app.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set spec: %w", err)
	}

	if err := m.client.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update ArgoCD application: %w", err)
	}

	return nil
}

// getArgoCDNamespace returns the namespace where ArgoCD is installed
func getArgoCDNamespace() string {
	// This could be made configurable
	return "argocd"
}

// getStringValue safely gets a string value from a map
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
