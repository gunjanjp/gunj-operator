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
	"strings"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// FluxManager manages Flux Kustomizations
type FluxManager struct {
	client client.Client
	log    logr.Logger
}

// NewFluxManager creates a new Flux manager
func NewFluxManager(client client.Client, log logr.Logger) *FluxManager {
	return &FluxManager{
		client: client,
		log:    log.WithName("flux-manager"),
	}
}

// FluxKustomizationConfig represents Flux Kustomization configuration
type FluxKustomizationConfig struct {
	Name            string
	SourceRef       string
	Path            string
	TargetNamespace string
	ServiceAccount  string
	Interval        string
	Timeout         string
	Prune           bool
	Values          map[string]string
	HealthChecks    []observabilityv1beta1.FluxHealthCheck
}

// CreateOrUpdateKustomization creates or updates a Flux Kustomization
func (m *FluxManager) CreateOrUpdateKustomization(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, config *FluxKustomizationConfig) error {
	log := m.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name), "kustomization", config.Name)

	// Create Flux Kustomization object
	kustomization := m.buildKustomization(deployment, config)

	// Check if kustomization exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "Kustomization",
	})

	err := m.client.Get(ctx, types.NamespacedName{
		Name:      config.Name,
		Namespace: deployment.Namespace,
	}, existing)

	if err == nil {
		// Update existing kustomization
		kustomization.SetResourceVersion(existing.GetResourceVersion())
		if err := m.client.Update(ctx, kustomization); err != nil {
			return fmt.Errorf("failed to update Flux Kustomization: %w", err)
		}
		log.Info("Flux Kustomization updated successfully")
	} else if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get Flux Kustomization: %w", err)
	} else {
		// Create new kustomization
		if err := m.client.Create(ctx, kustomization); err != nil {
			return fmt.Errorf("failed to create Flux Kustomization: %w", err)
		}
		log.Info("Flux Kustomization created successfully")
	}

	// Create GitRepository source if it doesn't exist
	if err := m.ensureGitRepositorySource(ctx, deployment); err != nil {
		return fmt.Errorf("failed to ensure GitRepository source: %w", err)
	}

	return nil
}

// buildKustomization builds a Flux Kustomization object
func (m *FluxManager) buildKustomization(deployment *observabilityv1beta1.GitOpsDeployment, config *FluxKustomizationConfig) *unstructured.Unstructured {
	kustomization := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
			"kind":       "Kustomization",
			"metadata": map[string]interface{}{
				"name":      config.Name,
				"namespace": deployment.Namespace,
				"labels": map[string]interface{}{
					"observability.io/gitops-deployment": deployment.Name,
					"observability.io/managed-by":        "gunj-operator",
				},
			},
			"spec": m.buildKustomizationSpec(config),
		},
	}

	return kustomization
}

// buildKustomizationSpec builds the Flux Kustomization spec
func (m *FluxManager) buildKustomizationSpec(config *FluxKustomizationConfig) map[string]interface{} {
	spec := map[string]interface{}{
		"interval": config.Interval,
		"path":     config.Path,
		"prune":    config.Prune,
		"sourceRef": map[string]interface{}{
			"kind": "GitRepository",
			"name": config.SourceRef,
		},
		"targetNamespace": config.TargetNamespace,
	}

	if config.ServiceAccount != "" {
		spec["serviceAccountName"] = config.ServiceAccount
	}

	if config.Timeout != "" {
		spec["timeout"] = config.Timeout
	}

	// Add post-build substitutions for values
	if len(config.Values) > 0 {
		spec["postBuild"] = map[string]interface{}{
			"substitute": config.Values,
		}
	}

	// Add health checks
	if len(config.HealthChecks) > 0 {
		healthChecks := make([]interface{}, 0, len(config.HealthChecks))
		for _, hc := range config.HealthChecks {
			check := map[string]interface{}{
				"kind": hc.Type,
				"name": hc.Name,
			}
			if hc.Namespace != "" {
				check["namespace"] = hc.Namespace
			}
			healthChecks = append(healthChecks, check)
		}
		spec["healthChecks"] = healthChecks
	}

	return spec
}

// ensureGitRepositorySource ensures the GitRepository source exists
func (m *FluxManager) ensureGitRepositorySource(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	source := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "source.toolkit.fluxcd.io/v1beta2",
			"kind":       "GitRepository",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-source", deployment.Name),
				"namespace": deployment.Namespace,
				"labels": map[string]interface{}{
					"observability.io/gitops-deployment": deployment.Name,
					"observability.io/managed-by":        "gunj-operator",
				},
			},
			"spec": map[string]interface{}{
				"interval": deployment.Spec.Repository.PollInterval,
				"url":      deployment.Spec.Repository.URL,
				"ref": map[string]interface{}{
					"branch": deployment.Spec.Repository.Branch,
				},
			},
		},
	}

	// Add secret reference if configured
	if deployment.Spec.Repository.SecretRef != nil {
		source.Object["spec"].(map[string]interface{})["secretRef"] = map[string]interface{}{
			"name": deployment.Spec.Repository.SecretRef.Name,
		}
	}

	// Check if source exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(source.GetObjectKind().GroupVersionKind())

	err := m.client.Get(ctx, types.NamespacedName{
		Name:      source.GetName(),
		Namespace: source.GetNamespace(),
	}, existing)

	if err == nil {
		// Update existing source
		source.SetResourceVersion(existing.GetResourceVersion())
		return m.client.Update(ctx, source)
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}

	// Create new source
	return m.client.Create(ctx, source)
}

// GetSyncStatus gets the sync status of Flux Kustomizations
func (m *FluxManager) GetSyncStatus(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.SyncStatus, error) {
	// List all kustomizations for this deployment
	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "KustomizationList",
	})

	if err := m.client.List(ctx, kustomizationList,
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return nil, fmt.Errorf("failed to list Flux Kustomizations: %w", err)
	}

	// Aggregate sync status
	allReady := true
	syncMessage := ""
	var lastRevision string

	for _, kustomization := range kustomizationList.Items {
		status, found, err := unstructured.NestedMap(kustomization.Object, "status")
		if err != nil || !found {
			continue
		}

		// Check conditions
		conditions, found, err := unstructured.NestedSlice(status, "conditions")
		if err != nil || !found {
			allReady = false
			continue
		}

		for _, c := range conditions {
			condition, ok := c.(map[string]interface{})
			if !ok {
				continue
			}

			if getStringValue(condition, "type") == "Ready" {
				if getStringValue(condition, "status") != "True" {
					allReady = false
					syncMessage = getStringValue(condition, "message")
				}
			}
		}

		// Get last applied revision
		if lastAppliedRevision, found, err := unstructured.NestedString(status, "lastAppliedRevision"); err == nil && found {
			lastRevision = lastAppliedRevision
		}
	}

	status := "Synced"
	if !allReady {
		status = "OutOfSync"
	}

	return &observabilityv1beta1.SyncStatus{
		Status:   status,
		Revision: lastRevision,
		Message:  syncMessage,
	}, nil
}

// GetHealthStatus gets the health status of Flux Kustomizations
func (m *FluxManager) GetHealthStatus(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.HealthStatus, error) {
	// List all kustomizations for this deployment
	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "KustomizationList",
	})

	if err := m.client.List(ctx, kustomizationList,
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return nil, fmt.Errorf("failed to list Flux Kustomizations: %w", err)
	}

	// Aggregate health status
	overallHealth := "Healthy"
	healthMessage := ""
	unhealthyCount := 0

	for _, kustomization := range kustomizationList.Items {
		status, found, err := unstructured.NestedMap(kustomization.Object, "status")
		if err != nil || !found {
			continue
		}

		// Check conditions
		conditions, found, err := unstructured.NestedSlice(status, "conditions")
		if err != nil || !found {
			continue
		}

		isHealthy := true
		for _, c := range conditions {
			condition, ok := c.(map[string]interface{})
			if !ok {
				continue
			}

			condType := getStringValue(condition, "type")
			condStatus := getStringValue(condition, "status")
			condReason := getStringValue(condition, "reason")

			if condType == "Ready" && condStatus != "True" {
				isHealthy = false
				if condReason == "ProgressingWithRetry" || condReason == "Progressing" {
					if overallHealth != "Degraded" {
						overallHealth = "Progressing"
					}
				} else {
					overallHealth = "Degraded"
					healthMessage = getStringValue(condition, "message")
				}
			}
		}

		if !isHealthy {
			unhealthyCount++
		}
	}

	// Build message
	if healthMessage == "" && unhealthyCount > 0 {
		healthMessage = fmt.Sprintf("%d kustomizations are not ready", unhealthyCount)
	}

	return &observabilityv1beta1.HealthStatus{
		Status:  overallHealth,
		Message: healthMessage,
	}, nil
}

// DeleteKustomizations deletes all Flux Kustomizations for a deployment
func (m *FluxManager) DeleteKustomizations(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// List all kustomizations for this deployment
	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "KustomizationList",
	})

	if err := m.client.List(ctx, kustomizationList,
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return fmt.Errorf("failed to list Flux Kustomizations: %w", err)
	}

	// Delete each kustomization
	for _, kustomization := range kustomizationList.Items {
		log := m.log.WithValues("kustomization", kustomization.GetName())

		if err := m.client.Delete(ctx, &kustomization); err != nil {
			log.Error(err, "Failed to delete Flux Kustomization")
			continue
		}
		log.Info("Flux Kustomization deleted")
	}

	// Delete GitRepository source
	source := &unstructured.Unstructured{}
	source.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "GitRepository",
	})
	source.SetName(fmt.Sprintf("%s-source", deployment.Name))
	source.SetNamespace(deployment.Namespace)

	if err := m.client.Delete(ctx, source); err != nil && !client.IgnoreNotFound(err) != nil {
		m.log.Error(err, "Failed to delete GitRepository source")
	}

	return nil
}

// ReconcileKustomization triggers a reconciliation for a Flux Kustomization
func (m *FluxManager) ReconcileKustomization(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, kustomizationName string) error {
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "Kustomization",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      kustomizationName,
		Namespace: deployment.Namespace,
	}, kustomization); err != nil {
		return fmt.Errorf("failed to get Flux Kustomization: %w", err)
	}

	// Update annotation to trigger reconciliation
	annotations := kustomization.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
	kustomization.SetAnnotations(annotations)

	if err := m.client.Update(ctx, kustomization); err != nil {
		return fmt.Errorf("failed to update Flux Kustomization: %w", err)
	}

	return nil
}

// GetKustomizationResources gets the resources managed by a Flux Kustomization
func (m *FluxManager) GetKustomizationResources(ctx context.Context, kustomizationName, namespace string) ([]observabilityv1beta1.ResourceStatus, error) {
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "Kustomization",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      kustomizationName,
		Namespace: namespace,
	}, kustomization); err != nil {
		return nil, fmt.Errorf("failed to get Flux Kustomization: %w", err)
	}

	status, found, err := unstructured.NestedMap(kustomization.Object, "status")
	if err != nil || !found {
		return nil, fmt.Errorf("kustomization status not found")
	}

	inventory, found, err := unstructured.NestedMap(status, "inventory")
	if err != nil || !found {
		return []observabilityv1beta1.ResourceStatus{}, nil
	}

	entriesRaw, found, err := unstructured.NestedSlice(inventory, "entries")
	if err != nil || !found {
		return []observabilityv1beta1.ResourceStatus{}, nil
	}

	var resources []observabilityv1beta1.ResourceStatus
	for _, e := range entriesRaw {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse the ID field
		id := getStringValue(entry, "id")
		parts := strings.Split(id, "_")
		if len(parts) >= 5 {
			rs := observabilityv1beta1.ResourceStatus{
				Namespace: parts[0],
				Name:      parts[1],
				Group:     parts[2],
				Version:   parts[3],
				Kind:      parts[4],
				Status:    "Synced",
			}
			resources = append(resources, rs)
		}
	}

	return resources, nil
}

// UpdateKustomizationSpec updates the spec of a Flux Kustomization
func (m *FluxManager) UpdateKustomizationSpec(ctx context.Context, kustomizationName, namespace string, updateFunc func(spec map[string]interface{}) error) error {
	kustomization := &unstructured.Unstructured{}
	kustomization.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "Kustomization",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      kustomizationName,
		Namespace: namespace,
	}, kustomization); err != nil {
		return fmt.Errorf("failed to get Flux Kustomization: %w", err)
	}

	spec, found, err := unstructured.NestedMap(kustomization.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("kustomization spec not found")
	}

	if err := updateFunc(spec); err != nil {
		return err
	}

	if err := unstructured.SetNestedField(kustomization.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set spec: %w", err)
	}

	if err := m.client.Update(ctx, kustomization); err != nil {
		return fmt.Errorf("failed to update Flux Kustomization: %w", err)
	}

	return nil
}

// SuspendKustomization suspends a Flux Kustomization
func (m *FluxManager) SuspendKustomization(ctx context.Context, kustomizationName, namespace string, suspend bool) error {
	return m.UpdateKustomizationSpec(ctx, kustomizationName, namespace, func(spec map[string]interface{}) error {
		spec["suspend"] = suspend
		return nil
	})
}
