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

package argocd

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
)

// ArgoCDManager manages ArgoCD integration
type ArgoCDManager struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewArgoCDManager creates a new ArgoCD manager
func NewArgoCDManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ArgoCDManager {
	return &ArgoCDManager{
		Client: client,
		Scheme: scheme,
		Log:    log.WithName("argocd-manager"),
	}
}

// CreateApplication creates an ArgoCD Application for the ObservabilityPlatform
func (m *ArgoCDManager) CreateApplication(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Creating ArgoCD Application")

	app := m.buildApplication(platform, gitOps)

	// Check if application already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      app.GetName(),
		Namespace: app.GetNamespace(),
	}, existing)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing application: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new application
		if err := m.Client.Create(ctx, app); err != nil {
			return fmt.Errorf("failed to create ArgoCD application: %w", err)
		}
		log.Info("ArgoCD Application created successfully")
	} else {
		// Update existing application
		app.SetResourceVersion(existing.GetResourceVersion())
		if err := m.Client.Update(ctx, app); err != nil {
			return fmt.Errorf("failed to update ArgoCD application: %w", err)
		}
		log.Info("ArgoCD Application updated successfully")
	}

	return nil
}

// buildApplication builds an ArgoCD Application resource
func (m *ArgoCDManager) buildApplication(
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) *unstructured.Unstructured {
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	app.SetName(fmt.Sprintf("%s-app", platform.Name))
	app.SetNamespace("argocd") // Default ArgoCD namespace

	// Build application spec
	spec := map[string]interface{}{
		"project": "default",
		"source": map[string]interface{}{
			"repoURL":        gitOps.Repository.URL,
			"targetRevision": gitOps.Repository.Branch,
			"path":           gitOps.Repository.Path,
		},
		"destination": map[string]interface{}{
			"server":    "https://kubernetes.default.svc",
			"namespace": platform.Namespace,
		},
	}

	// Add sync policy if configured
	if gitOps.SyncPolicy.Automated != nil {
		syncPolicy := map[string]interface{}{}
		
		if gitOps.SyncPolicy.Automated != nil {
			automated := map[string]interface{}{}
			if gitOps.SyncPolicy.Automated.Prune {
				automated["prune"] = true
			}
			if gitOps.SyncPolicy.Automated.SelfHeal {
				automated["selfHeal"] = true
			}
			if gitOps.SyncPolicy.Automated.AllowEmpty {
				automated["allowEmpty"] = true
			}
			syncPolicy["automated"] = automated
		}

		if gitOps.SyncPolicy.Retry != nil {
			retry := map[string]interface{}{
				"limit": gitOps.SyncPolicy.Retry.Limit,
			}
			if gitOps.SyncPolicy.Retry.Backoff != nil {
				backoff := map[string]interface{}{}
				if gitOps.SyncPolicy.Retry.Backoff.Duration != "" {
					backoff["duration"] = gitOps.SyncPolicy.Retry.Backoff.Duration
				}
				if gitOps.SyncPolicy.Retry.Backoff.Factor != 0 {
					backoff["factor"] = gitOps.SyncPolicy.Retry.Backoff.Factor
				}
				if gitOps.SyncPolicy.Retry.Backoff.MaxDuration != "" {
					backoff["maxDuration"] = gitOps.SyncPolicy.Retry.Backoff.MaxDuration
				}
				retry["backoff"] = backoff
			}
			syncPolicy["retry"] = retry
		}

		if len(gitOps.SyncPolicy.SyncOptions) > 0 {
			syncPolicy["syncOptions"] = gitOps.SyncPolicy.SyncOptions
		}

		spec["syncPolicy"] = syncPolicy
	}

	// Add revision history limit if rollback is configured
	if gitOps.Rollback != nil && gitOps.Rollback.RevisionHistoryLimit > 0 {
		spec["revisionHistoryLimit"] = gitOps.Rollback.RevisionHistoryLimit
	}

	app.Object["spec"] = spec

	// Add labels and annotations
	labels := map[string]string{
		"app.kubernetes.io/name":       "gunj-operator",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  "gitops",
		"app.kubernetes.io/managed-by": "gunj-operator",
		"observability.io/platform":    platform.Name,
	}
	app.SetLabels(labels)

	annotations := map[string]string{
		"observability.io/gitops-type": string(gitops.GitOpsArgoCD),
	}
	app.SetAnnotations(annotations)

	return app
}

// CreateApplicationSet creates an ArgoCD ApplicationSet for multi-environment deployments
func (m *ArgoCDManager) CreateApplicationSet(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) error {
	if len(gitOps.Environments) == 0 {
		return nil // No environments configured
	}

	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Creating ArgoCD ApplicationSet for multi-environment")

	appSet := m.buildApplicationSet(platform, gitOps)

	// Check if ApplicationSet already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationSet",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      appSet.GetName(),
		Namespace: appSet.GetNamespace(),
	}, existing)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing ApplicationSet: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new ApplicationSet
		if err := m.Client.Create(ctx, appSet); err != nil {
			return fmt.Errorf("failed to create ArgoCD ApplicationSet: %w", err)
		}
		log.Info("ArgoCD ApplicationSet created successfully")
	} else {
		// Update existing ApplicationSet
		appSet.SetResourceVersion(existing.GetResourceVersion())
		if err := m.Client.Update(ctx, appSet); err != nil {
			return fmt.Errorf("failed to update ArgoCD ApplicationSet: %w", err)
		}
		log.Info("ArgoCD ApplicationSet updated successfully")
	}

	return nil
}

// buildApplicationSet builds an ArgoCD ApplicationSet resource
func (m *ArgoCDManager) buildApplicationSet(
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) *unstructured.Unstructured {
	appSet := &unstructured.Unstructured{}
	appSet.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationSet",
	})

	appSet.SetName(fmt.Sprintf("%s-appset", platform.Name))
	appSet.SetNamespace("argocd")

	// Build generators for environments
	generators := []interface{}{}
	for _, env := range gitOps.Environments {
		generator := map[string]interface{}{
			"list": map[string]interface{}{
				"elements": []interface{}{
					map[string]interface{}{
						"env":      env.Name,
						"branch":   m.getEnvBranch(env, gitOps.Repository.Branch),
						"path":     m.getEnvPath(env, gitOps.Repository.Path),
						"revision": m.getEnvRevision(env),
						"autoSync": env.AutoSync,
					},
				},
			},
		}
		generators = append(generators, generator)
	}

	// Build template
	template := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "{{env}}-" + platform.Name,
		},
		"spec": map[string]interface{}{
			"project": "default",
			"source": map[string]interface{}{
				"repoURL":        gitOps.Repository.URL,
				"targetRevision": "{{branch}}",
				"path":           "{{path}}",
			},
			"destination": map[string]interface{}{
				"server":    "https://kubernetes.default.svc",
				"namespace": platform.Namespace + "-{{env}}",
			},
		},
	}

	// Add sync policy to template
	if gitOps.SyncPolicy.Automated != nil {
		syncPolicy := map[string]interface{}{}
		
		// Use autoSync from environment if available
		automated := map[string]interface{}{
			"prune":   gitOps.SyncPolicy.Automated.Prune,
			"selfHeal": gitOps.SyncPolicy.Automated.SelfHeal,
		}
		
		// Conditional auto-sync based on environment
		if gitOps.SyncPolicy.Automated.AllowEmpty {
			automated["allowEmpty"] = true
		}
		
		syncPolicy["automated"] = automated
		templateSpec := template["spec"].(map[string]interface{})
		templateSpec["syncPolicy"] = syncPolicy
	}

	spec := map[string]interface{}{
		"generators": generators,
		"template":   template,
	}

	appSet.Object["spec"] = spec

	// Add labels and annotations
	labels := map[string]string{
		"app.kubernetes.io/name":       "gunj-operator",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  "gitops",
		"app.kubernetes.io/managed-by": "gunj-operator",
		"observability.io/platform":    platform.Name,
	}
	appSet.SetLabels(labels)

	return appSet
}

// getEnvBranch returns the branch for an environment
func (m *ArgoCDManager) getEnvBranch(env gitopsv1beta1.EnvironmentSpec, defaultBranch string) string {
	if env.Branch != "" {
		return env.Branch
	}
	return defaultBranch
}

// getEnvPath returns the path for an environment
func (m *ArgoCDManager) getEnvPath(env gitopsv1beta1.EnvironmentSpec, defaultPath string) string {
	if env.Path != "" {
		return env.Path
	}
	return defaultPath
}

// getEnvRevision returns the target revision for an environment
func (m *ArgoCDManager) getEnvRevision(env gitopsv1beta1.EnvironmentSpec) string {
	if env.TargetRevision != "" {
		return env.TargetRevision
	}
	return "HEAD"
}

// SyncApplication triggers a sync for the ArgoCD application
func (m *ArgoCDManager) SyncApplication(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Syncing ArgoCD Application")

	// Get the application
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-app", platform.Name),
		Namespace: "argocd",
	}, app)

	if err != nil {
		return fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Update sync operation
	operation := map[string]interface{}{
		"sync": map[string]interface{}{
			"revision": "HEAD",
			"prune":    true,
			"syncOptions": []string{
				"CreateNamespace=true",
			},
		},
		"initiatedBy": map[string]interface{}{
			"username": "gunj-operator",
		},
		"info": []map[string]string{
			{
				"name":  "Reason",
				"value": "Triggered by Gunj Operator",
			},
		},
	}

	spec := app.Object["spec"].(map[string]interface{})
	spec["operation"] = operation
	app.Object["spec"] = spec

	// Update the application to trigger sync
	if err := m.Client.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to trigger sync: %w", err)
	}

	log.Info("ArgoCD Application sync triggered successfully")
	return nil
}

// GetSyncStatus gets the sync status of the ArgoCD application
func (m *ArgoCDManager) GetSyncStatus(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) (*SyncStatus, error) {
	// Get the application
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-app", platform.Name),
		Namespace: "argocd",
	}, app)

	if err != nil {
		if errors.IsNotFound(err) {
			return &SyncStatus{
				Phase:   "NotFound",
				Message: "ArgoCD application not found",
			}, nil
		}
		return nil, fmt.Errorf("failed to get ArgoCD application: %w", err)
	}

	// Extract status
	status, exists := app.Object["status"].(map[string]interface{})
	if !exists {
		return &SyncStatus{
			Phase:   "Unknown",
			Message: "Status not available",
		}, nil
	}

	// Extract sync status
	syncStatus, exists := status["sync"].(map[string]interface{})
	if !exists {
		return &SyncStatus{
			Phase:   "Unknown",
			Message: "Sync status not available",
		}, nil
	}

	// Extract health status
	healthStatus, _ := status["health"].(map[string]interface{})

	result := &SyncStatus{
		Phase:    getStringValue(syncStatus, "status", "Unknown"),
		Revision: getStringValue(syncStatus, "revision", ""),
		Message:  getStringValue(status, "message", ""),
	}

	// Set health status if available
	if healthStatus != nil {
		result.Health = getStringValue(healthStatus, "status", "Unknown")
	}

	// Extract completed at time
	if comparedTo, ok := syncStatus["comparedTo"].(map[string]interface{}); ok {
		if finishedAt, ok := status["operationState"].(map[string]interface{})["finishedAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339, finishedAt); err == nil {
				result.CompletedAt = &metav1.Time{Time: t}
			}
		}
		result.Revision = getStringValue(comparedTo, "source.targetRevision", result.Revision)
	}

	return result, nil
}

// getStringValue safely extracts a string value from a map
func getStringValue(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

// SyncStatus represents the sync status of an ArgoCD application
type SyncStatus struct {
	Phase       string
	Health      string
	Revision    string
	Message     string
	CompletedAt *metav1.Time
}

// DeleteApplication deletes the ArgoCD application
func (m *ArgoCDManager) DeleteApplication(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Deleting ArgoCD Application")

	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})
	app.SetName(fmt.Sprintf("%s-app", platform.Name))
	app.SetNamespace("argocd")

	if err := m.Client.Delete(ctx, app); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete ArgoCD application: %w", err)
		}
	}

	// Also delete ApplicationSet if exists
	appSet := &unstructured.Unstructured{}
	appSet.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationSet",
	})
	appSet.SetName(fmt.Sprintf("%s-appset", platform.Name))
	appSet.SetNamespace("argocd")

	if err := m.Client.Delete(ctx, appSet); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete ArgoCD ApplicationSet: %w", err)
		}
	}

	log.Info("ArgoCD resources deleted successfully")
	return nil
}
