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

package flux

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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

// FluxManager manages Flux v2 integration
type FluxManager struct {
	Client client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// NewFluxManager creates a new Flux manager
func NewFluxManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *FluxManager {
	return &FluxManager{
		Client: client,
		Scheme: scheme,
		Log:    log.WithName("flux-manager"),
	}
}

// CreateGitRepository creates a Flux GitRepository source
func (m *FluxManager) CreateGitRepository(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Creating Flux GitRepository")

	gitRepo := m.buildGitRepository(platform, gitOps)

	// Check if GitRepository already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "GitRepository",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      gitRepo.GetName(),
		Namespace: gitRepo.GetNamespace(),
	}, existing)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing GitRepository: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new GitRepository
		if err := m.Client.Create(ctx, gitRepo); err != nil {
			return fmt.Errorf("failed to create Flux GitRepository: %w", err)
		}
		log.Info("Flux GitRepository created successfully")
	} else {
		// Update existing GitRepository
		gitRepo.SetResourceVersion(existing.GetResourceVersion())
		if err := m.Client.Update(ctx, gitRepo); err != nil {
			return fmt.Errorf("failed to update Flux GitRepository: %w", err)
		}
		log.Info("Flux GitRepository updated successfully")
	}

	return nil
}

// buildGitRepository builds a Flux GitRepository resource
func (m *FluxManager) buildGitRepository(
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) *unstructured.Unstructured {
	gitRepo := &unstructured.Unstructured{}
	gitRepo.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "GitRepository",
	})

	gitRepo.SetName(fmt.Sprintf("%s-source", platform.Name))
	gitRepo.SetNamespace(platform.Namespace)

	spec := map[string]interface{}{
		"url": gitOps.Repository.URL,
		"ref": map[string]interface{}{
			"branch": gitOps.Repository.Branch,
		},
		"interval": "1m", // Default interval
	}

	// Add secret reference if credentials are provided
	if gitOps.Repository.CredentialsSecret != nil {
		spec["secretRef"] = map[string]interface{}{
			"name": gitOps.Repository.CredentialsSecret.Name,
		}
	}

	gitRepo.Object["spec"] = spec

	// Add labels and annotations
	labels := map[string]string{
		"app.kubernetes.io/name":       "gunj-operator",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  "gitops",
		"app.kubernetes.io/managed-by": "gunj-operator",
		"observability.io/platform":    platform.Name,
	}
	gitRepo.SetLabels(labels)

	return gitRepo
}

// CreateKustomization creates a Flux Kustomization
func (m *FluxManager) CreateKustomization(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Creating Flux Kustomization")

	kustomization := m.buildKustomization(platform, gitOps)

	// Check if Kustomization already exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      kustomization.GetName(),
		Namespace: kustomization.GetNamespace(),
	}, existing)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing Kustomization: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new Kustomization
		if err := m.Client.Create(ctx, kustomization); err != nil {
			return fmt.Errorf("failed to create Flux Kustomization: %w", err)
		}
		log.Info("Flux Kustomization created successfully")
	} else {
		// Update existing Kustomization
		kustomization.SetResourceVersion(existing.GetResourceVersion())
		if err := m.Client.Update(ctx, kustomization); err != nil {
			return fmt.Errorf("failed to update Flux Kustomization: %w", err)
		}
		log.Info("Flux Kustomization updated successfully")
	}

	return nil
}

// buildKustomization builds a Flux Kustomization resource
func (m *FluxManager) buildKustomization(
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) *unstructured.Unstructured {
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	})

	kustomization.SetName(fmt.Sprintf("%s-kustomization", platform.Name))
	kustomization.SetNamespace(platform.Namespace)

	spec := map[string]interface{}{
		"sourceRef": map[string]interface{}{
			"kind": "GitRepository",
			"name": fmt.Sprintf("%s-source", platform.Name),
		},
		"path":     gitOps.Repository.Path,
		"interval": "5m",
		"prune":    true,
	}

	// Add target namespace
	spec["targetNamespace"] = platform.Namespace

	// Add sync policy configuration
	if gitOps.SyncPolicy.Automated != nil {
		if gitOps.SyncPolicy.Automated.Prune {
			spec["prune"] = true
		}
		if gitOps.SyncPolicy.Automated.SelfHeal {
			spec["force"] = true
		}
	}

	// Add retry configuration
	if gitOps.SyncPolicy.Retry != nil {
		spec["retryInterval"] = "30s" // Default retry interval
		if gitOps.SyncPolicy.Retry.Limit > 0 {
			spec["retries"] = gitOps.SyncPolicy.Retry.Limit
		}
	}

	// Add health checks
	spec["healthChecks"] = []map[string]interface{}{
		{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"name":       platform.Name + "-prometheus",
			"namespace":  platform.Namespace,
		},
		{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"name":       platform.Name + "-grafana",
			"namespace":  platform.Namespace,
		},
	}

	// Add wait for resources
	spec["wait"] = true
	spec["timeout"] = "10m"

	kustomization.Object["spec"] = spec

	// Add labels and annotations
	labels := map[string]string{
		"app.kubernetes.io/name":       "gunj-operator",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  "gitops",
		"app.kubernetes.io/managed-by": "gunj-operator",
		"observability.io/platform":    platform.Name,
	}
	kustomization.SetLabels(labels)

	annotations := map[string]string{
		"observability.io/gitops-type": string(gitops.GitOpsFlux),
	}
	kustomization.SetAnnotations(annotations)

	return kustomization
}

// CreateMultiEnvironmentKustomizations creates Kustomizations for multiple environments
func (m *FluxManager) CreateMultiEnvironmentKustomizations(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) error {
	if len(gitOps.Environments) == 0 {
		return nil
	}

	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Creating Flux Kustomizations for multiple environments")

	// Create a GitRepository for each environment
	for _, env := range gitOps.Environments {
		// Clone GitOps spec and modify for environment
		envGitOps := *gitOps
		if env.Branch != "" {
			envGitOps.Repository.Branch = env.Branch
		}
		if env.Path != "" {
			envGitOps.Repository.Path = env.Path
		}

		// Create GitRepository for environment
		gitRepo := m.buildEnvironmentGitRepository(platform, &envGitOps, env)
		if err := m.createOrUpdateResource(ctx, gitRepo); err != nil {
			return fmt.Errorf("failed to create GitRepository for environment %s: %w", env.Name, err)
		}

		// Create Kustomization for environment
		kustomization := m.buildEnvironmentKustomization(platform, &envGitOps, env)
		if err := m.createOrUpdateResource(ctx, kustomization); err != nil {
			return fmt.Errorf("failed to create Kustomization for environment %s: %w", env.Name, err)
		}
	}

	return nil
}

// buildEnvironmentGitRepository builds a GitRepository for a specific environment
func (m *FluxManager) buildEnvironmentGitRepository(
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
	env gitopsv1beta1.EnvironmentSpec,
) *unstructured.Unstructured {
	gitRepo := m.buildGitRepository(platform, gitOps)
	gitRepo.SetName(fmt.Sprintf("%s-%s-source", platform.Name, env.Name))
	gitRepo.SetNamespace(fmt.Sprintf("%s-%s", platform.Namespace, env.Name))

	// Update spec for environment
	spec := gitRepo.Object["spec"].(map[string]interface{})
	if env.Branch != "" {
		ref := spec["ref"].(map[string]interface{})
		ref["branch"] = env.Branch
	}
	if env.TargetRevision != "" {
		ref := spec["ref"].(map[string]interface{})
		ref["tag"] = env.TargetRevision
	}

	return gitRepo
}

// buildEnvironmentKustomization builds a Kustomization for a specific environment
func (m *FluxManager) buildEnvironmentKustomization(
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
	env gitopsv1beta1.EnvironmentSpec,
) *unstructured.Unstructured {
	kustomization := m.buildKustomization(platform, gitOps)
	kustomization.SetName(fmt.Sprintf("%s-%s-kustomization", platform.Name, env.Name))
	kustomization.SetNamespace(fmt.Sprintf("%s-%s", platform.Namespace, env.Name))

	// Update spec for environment
	spec := kustomization.Object["spec"].(map[string]interface{})
	sourceRef := spec["sourceRef"].(map[string]interface{})
	sourceRef["name"] = fmt.Sprintf("%s-%s-source", platform.Name, env.Name)

	if env.Path != "" {
		spec["path"] = env.Path
	}

	// Handle environment-specific auto-sync
	if !env.AutoSync {
		// Remove automated sync for this environment
		delete(spec, "prune")
		delete(spec, "force")
	}

	// Add environment label
	labels := kustomization.GetLabels()
	labels["observability.io/environment"] = env.Name
	kustomization.SetLabels(labels)

	return kustomization
}

// createOrUpdateResource creates or updates a resource
func (m *FluxManager) createOrUpdateResource(ctx context.Context, resource *unstructured.Unstructured) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(resource.GroupVersionKind())

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      resource.GetName(),
		Namespace: resource.GetNamespace(),
	}, existing)

	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		return m.Client.Create(ctx, resource)
	}

	resource.SetResourceVersion(existing.GetResourceVersion())
	return m.Client.Update(ctx, resource)
}

// GetSyncStatus gets the sync status of Flux resources
func (m *FluxManager) GetSyncStatus(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) (*SyncStatus, error) {
	// Get the Kustomization
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-kustomization", platform.Name),
		Namespace: platform.Namespace,
	}, kustomization)

	if err != nil {
		if errors.IsNotFound(err) {
			return &SyncStatus{
				Ready:   false,
				Message: "Flux Kustomization not found",
			}, nil
		}
		return nil, fmt.Errorf("failed to get Flux Kustomization: %w", err)
	}

	// Extract status
	status, exists := kustomization.Object["status"].(map[string]interface{})
	if !exists {
		return &SyncStatus{
			Ready:   false,
			Message: "Status not available",
		}, nil
	}

	// Extract conditions
	conditions, _ := status["conditions"].([]interface{})
	ready := false
	message := ""
	
	for _, cond := range conditions {
		condition := cond.(map[string]interface{})
		if condition["type"] == "Ready" {
			ready = condition["status"] == "True"
			message = getStringValue(condition, "message", "")
			break
		}
	}

	// Extract last applied revision
	lastAppliedRevision := ""
	if lastApplied, ok := status["lastAppliedRevision"].(string); ok {
		lastAppliedRevision = lastApplied
	}

	// Extract last attempted revision
	lastAttemptedRevision := ""
	if lastAttempted, ok := status["lastAttemptedRevision"].(string); ok {
		lastAttemptedRevision = lastAttempted
	}

	result := &SyncStatus{
		Ready:                 ready,
		Message:               message,
		LastAppliedRevision:   lastAppliedRevision,
		LastAttemptedRevision: lastAttemptedRevision,
	}

	// Extract last handled reconcile at
	if lastHandled, ok := status["lastHandledReconcileAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastHandled); err == nil {
			result.LastHandledReconcileAt = &metav1.Time{Time: t}
		}
	}

	return result, nil
}

// SyncStatus represents the sync status of Flux resources
type SyncStatus struct {
	Ready                  bool
	Message                string
	LastAppliedRevision    string
	LastAttemptedRevision  string
	LastHandledReconcileAt *metav1.Time
}

// TriggerReconciliation triggers a reconciliation of Flux resources
func (m *FluxManager) TriggerReconciliation(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Triggering Flux reconciliation")

	// Update Kustomization annotation to trigger reconciliation
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	})

	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-kustomization", platform.Name),
		Namespace: platform.Namespace,
	}, kustomization)

	if err != nil {
		return fmt.Errorf("failed to get Flux Kustomization: %w", err)
	}

	// Add or update reconcile annotation
	annotations := kustomization.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
	kustomization.SetAnnotations(annotations)

	if err := m.Client.Update(ctx, kustomization); err != nil {
		return fmt.Errorf("failed to trigger reconciliation: %w", err)
	}

	log.Info("Flux reconciliation triggered successfully")
	return nil
}

// DeleteFluxResources deletes all Flux resources for a platform
func (m *FluxManager) DeleteFluxResources(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Deleting Flux resources")

	// Delete Kustomization
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	})
	kustomization.SetName(fmt.Sprintf("%s-kustomization", platform.Name))
	kustomization.SetNamespace(platform.Namespace)

	if err := m.Client.Delete(ctx, kustomization); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete Flux Kustomization: %w", err)
		}
	}

	// Delete GitRepository
	gitRepo := &unstructured.Unstructured{}
	gitRepo.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "GitRepository",
	})
	gitRepo.SetName(fmt.Sprintf("%s-source", platform.Name))
	gitRepo.SetNamespace(platform.Namespace)

	if err := m.Client.Delete(ctx, gitRepo); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete Flux GitRepository: %w", err)
		}
	}

	log.Info("Flux resources deleted successfully")
	return nil
}

// CreateImageUpdateAutomation creates Flux image update automation
func (m *FluxManager) CreateImageUpdateAutomation(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Creating Flux image update automation")

	// Create ImageRepository for each component
	components := []string{"prometheus", "grafana", "loki", "tempo"}
	for _, component := range components {
		imageRepo := m.buildImageRepository(platform, component)
		if err := m.createOrUpdateResource(ctx, imageRepo); err != nil {
			return fmt.Errorf("failed to create ImageRepository for %s: %w", component, err)
		}

		// Create ImagePolicy
		imagePolicy := m.buildImagePolicy(platform, component)
		if err := m.createOrUpdateResource(ctx, imagePolicy); err != nil {
			return fmt.Errorf("failed to create ImagePolicy for %s: %w", component, err)
		}
	}

	// Create ImageUpdateAutomation
	automation := m.buildImageUpdateAutomation(platform, gitOps)
	if err := m.createOrUpdateResource(ctx, automation); err != nil {
		return fmt.Errorf("failed to create ImageUpdateAutomation: %w", err)
	}

	return nil
}

// buildImageRepository builds an ImageRepository resource
func (m *FluxManager) buildImageRepository(platform *observabilityv1.ObservabilityPlatform, component string) *unstructured.Unstructured {
	imageRepo := &unstructured.Unstructured{}
	imageRepo.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "image.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "ImageRepository",
	})

	imageRepo.SetName(fmt.Sprintf("%s-%s", platform.Name, component))
	imageRepo.SetNamespace(platform.Namespace)

	// Map component to image
	imageMap := map[string]string{
		"prometheus": "prom/prometheus",
		"grafana":    "grafana/grafana",
		"loki":       "grafana/loki",
		"tempo":      "grafana/tempo",
	}

	spec := map[string]interface{}{
		"image":    imageMap[component],
		"interval": "5m",
	}

	imageRepo.Object["spec"] = spec

	// Add labels
	labels := map[string]string{
		"app.kubernetes.io/name":       "gunj-operator",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/managed-by": "gunj-operator",
	}
	imageRepo.SetLabels(labels)

	return imageRepo
}

// buildImagePolicy builds an ImagePolicy resource
func (m *FluxManager) buildImagePolicy(platform *observabilityv1.ObservabilityPlatform, component string) *unstructured.Unstructured {
	imagePolicy := &unstructured.Unstructured{}
	imagePolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "image.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "ImagePolicy",
	})

	imagePolicy.SetName(fmt.Sprintf("%s-%s", platform.Name, component))
	imagePolicy.SetNamespace(platform.Namespace)

	spec := map[string]interface{}{
		"imageRepositoryRef": map[string]interface{}{
			"name": fmt.Sprintf("%s-%s", platform.Name, component),
		},
		"policy": map[string]interface{}{
			"semver": map[string]interface{}{
				"range": ">=1.0.0",
			},
		},
	}

	imagePolicy.Object["spec"] = spec

	// Add labels
	labels := map[string]string{
		"app.kubernetes.io/name":       "gunj-operator",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/managed-by": "gunj-operator",
	}
	imagePolicy.SetLabels(labels)

	return imagePolicy
}

// buildImageUpdateAutomation builds an ImageUpdateAutomation resource
func (m *FluxManager) buildImageUpdateAutomation(
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
) *unstructured.Unstructured {
	automation := &unstructured.Unstructured{}
	automation.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "image.toolkit.fluxcd.io",
		Version: "v1beta1",
		Kind:    "ImageUpdateAutomation",
	})

	automation.SetName(fmt.Sprintf("%s-automation", platform.Name))
	automation.SetNamespace(platform.Namespace)

	spec := map[string]interface{}{
		"sourceRef": map[string]interface{}{
			"kind": "GitRepository",
			"name": fmt.Sprintf("%s-source", platform.Name),
		},
		"git": map[string]interface{}{
			"checkout": map[string]interface{}{
				"ref": map[string]interface{}{
					"branch": gitOps.Repository.Branch,
				},
			},
			"commit": map[string]interface{}{
				"author": map[string]interface{}{
					"email": "gunj-operator@example.com",
					"name":  "Gunj Operator",
				},
				"messageTemplate": "Auto-update images for {{.AutomationObject.Name}}",
			},
			"push": map[string]interface{}{
				"branch": gitOps.Repository.Branch,
			},
		},
		"interval": "5m",
		"update": map[string]interface{}{
			"strategy": "Setters",
		},
	}

	automation.Object["spec"] = spec

	// Add labels
	labels := map[string]string{
		"app.kubernetes.io/name":       "gunj-operator",
		"app.kubernetes.io/instance":   platform.Name,
		"app.kubernetes.io/component":  "gitops",
		"app.kubernetes.io/managed-by": "gunj-operator",
	}
	automation.SetLabels(labels)

	return automation
}
