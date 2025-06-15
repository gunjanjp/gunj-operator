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
	"sort"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// RollbackManager manages rollback operations for GitOps deployments
type RollbackManager struct {
	client client.Client
	log    logr.Logger
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(client client.Client, log logr.Logger) *RollbackManager {
	return &RollbackManager{
		client: client,
		log:    log.WithName("rollback-manager"),
	}
}

// RevisionHistory represents a deployment revision
type RevisionHistory struct {
	Revision    string
	DeployTime  time.Time
	Status      string
	Health      string
	Environment string
	Metrics     map[string]float64
}

// Rollback performs a rollback to a previous revision
func (m *RollbackManager) Rollback(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	log := m.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	// Get revision history
	history, err := m.getRevisionHistory(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to get revision history: %w", err)
	}

	if len(history) < 2 {
		return fmt.Errorf("insufficient revision history for rollback")
	}

	// Find the last known good revision
	targetRevision := m.findLastGoodRevision(history)
	if targetRevision == nil {
		return fmt.Errorf("no suitable revision found for rollback")
	}

	log.Info("Rolling back to previous revision", "targetRevision", targetRevision.Revision)

	// Perform rollback based on GitOps engine
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		return m.rollbackArgoCD(ctx, deployment, targetRevision.Revision)
	case "flux":
		return m.rollbackFlux(ctx, deployment, targetRevision.Revision)
	default:
		return fmt.Errorf("unknown GitOps engine: %s", deployment.Spec.GitOpsEngine)
	}
}

// getRevisionHistory gets the revision history for a deployment
func (m *RollbackManager) getRevisionHistory(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) ([]RevisionHistory, error) {
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		return m.getArgoCDRevisionHistory(ctx, deployment)
	case "flux":
		return m.getFluxRevisionHistory(ctx, deployment)
	default:
		return nil, fmt.Errorf("unknown GitOps engine: %s", deployment.Spec.GitOpsEngine)
	}
}

// getArgoCDRevisionHistory gets revision history from ArgoCD
func (m *RollbackManager) getArgoCDRevisionHistory(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) ([]RevisionHistory, error) {
	var history []RevisionHistory

	// List ArgoCD applications
	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	if err := m.client.List(ctx, appList,
		client.InNamespace("argocd"),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return nil, err
	}

	for _, app := range appList.Items {
		// Get revision history from application status
		status, found, err := unstructured.NestedMap(app.Object, "status")
		if err != nil || !found {
			continue
		}

		// Get history from status
		historyRaw, found, err := unstructured.NestedSlice(status, "history")
		if err != nil || !found {
			continue
		}

		for _, h := range historyRaw {
			histItem, ok := h.(map[string]interface{})
			if !ok {
				continue
			}

			deployedAt, _ := time.Parse(time.RFC3339, getStringValue(histItem, "deployedAt"))
			
			rev := RevisionHistory{
				Revision:    getStringValue(histItem, "revision"),
				DeployTime:  deployedAt,
				Environment: app.GetLabels()["observability.io/environment"],
			}

			// Get sync status
			if sync, found := histItem["source"].(map[string]interface{}); found {
				rev.Status = getStringValue(sync, "status")
			}

			history = append(history, rev)
		}
	}

	// Sort by deploy time descending
	sort.Slice(history, func(i, j int) bool {
		return history[i].DeployTime.After(history[j].DeployTime)
	})

	return history, nil
}

// getFluxRevisionHistory gets revision history from Flux
func (m *RollbackManager) getFluxRevisionHistory(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) ([]RevisionHistory, error) {
	var history []RevisionHistory

	// For Flux, we need to look at the GitRepository source history
	source := &unstructured.Unstructured{}
	source.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "GitRepository",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-source", deployment.Name),
		Namespace: deployment.Namespace,
	}, source); err != nil {
		return nil, err
	}

	// Get status
	status, found, err := unstructured.NestedMap(source.Object, "status")
	if err != nil || !found {
		return history, nil
	}

	// Current revision
	if artifact, found := status["artifact"].(map[string]interface{}); found {
		rev := RevisionHistory{
			Revision:   getStringValue(artifact, "revision"),
			DeployTime: time.Now(), // Flux doesn't store deploy time
			Status:     "Synced",
		}
		history = append(history, rev)
	}

	// For more history, we would need to query the Git repository directly
	// or maintain our own history in ConfigMaps/Secrets

	return history, nil
}

// findLastGoodRevision finds the last known good revision
func (m *RollbackManager) findLastGoodRevision(history []RevisionHistory) *RevisionHistory {
	// Skip the current revision (first in the list)
	for i := 1; i < len(history); i++ {
		rev := &history[i]
		
		// Check if this revision was healthy
		if rev.Status == "Synced" && (rev.Health == "Healthy" || rev.Health == "") {
			// Additional checks could be added here
			// e.g., checking metrics, error rates, etc.
			return rev
		}
	}

	// If no healthy revision found, return the previous one
	if len(history) > 1 {
		return &history[1]
	}

	return nil
}

// rollbackArgoCD performs rollback for ArgoCD
func (m *RollbackManager) rollbackArgoCD(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, targetRevision string) error {
	// List ArgoCD applications
	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	if err := m.client.List(ctx, appList,
		client.InNamespace("argocd"),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return err
	}

	// Rollback each application
	for _, app := range appList.Items {
		if err := m.rollbackArgoCDApplication(ctx, &app, targetRevision); err != nil {
			m.log.Error(err, "Failed to rollback ArgoCD application", "app", app.GetName())
			return err
		}
	}

	// Record rollback event
	m.recordRollbackEvent(ctx, deployment, targetRevision, "ArgoCD rollback initiated")

	return nil
}

// rollbackArgoCDApplication rolls back a single ArgoCD application
func (m *RollbackManager) rollbackArgoCDApplication(ctx context.Context, app *unstructured.Unstructured, targetRevision string) error {
	// Update the target revision in the application spec
	spec, found, err := unstructured.NestedMap(app.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("application spec not found")
	}

	source, found, err := unstructured.NestedMap(spec, "source")
	if err != nil || !found {
		return fmt.Errorf("application source not found")
	}

	// Set the target revision
	source["targetRevision"] = targetRevision

	if err := unstructured.SetNestedField(app.Object, source, "spec", "source"); err != nil {
		return err
	}

	// Trigger sync
	operation := map[string]interface{}{
		"sync": map[string]interface{}{
			"revision": targetRevision,
			"prune":    true,
			"dryRun":   false,
		},
	}

	if err := unstructured.SetNestedField(app.Object, operation, "operation"); err != nil {
		return err
	}

	// Update the application
	if err := m.client.Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update ArgoCD application: %w", err)
	}

	return nil
}

// rollbackFlux performs rollback for Flux
func (m *RollbackManager) rollbackFlux(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, targetRevision string) error {
	// For Flux, we need to update the GitRepository source to point to the target revision
	source := &unstructured.Unstructured{}
	source.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "GitRepository",
	})

	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-source", deployment.Name),
		Namespace: deployment.Namespace,
	}, source); err != nil {
		return err
	}

	// Update the ref to use a specific commit
	spec, found, err := unstructured.NestedMap(source.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("source spec not found")
	}

	ref := map[string]interface{}{
		"commit": targetRevision,
	}
	spec["ref"] = ref

	if err := unstructured.SetNestedField(source.Object, spec, "spec"); err != nil {
		return err
	}

	// Update the source
	if err := m.client.Update(ctx, source); err != nil {
		return fmt.Errorf("failed to update GitRepository source: %w", err)
	}

	// Trigger reconciliation for all Kustomizations
	if err := m.triggerFluxReconciliation(ctx, deployment); err != nil {
		return err
	}

	// Record rollback event
	m.recordRollbackEvent(ctx, deployment, targetRevision, "Flux rollback initiated")

	return nil
}

// triggerFluxReconciliation triggers reconciliation for all Flux Kustomizations
func (m *RollbackManager) triggerFluxReconciliation(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "KustomizationList",
	})

	if err := m.client.List(ctx, kustomizationList,
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return err
	}

	for _, kustomization := range kustomizationList.Items {
		// Update annotation to trigger reconciliation
		annotations := kustomization.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
		kustomization.SetAnnotations(annotations)

		if err := m.client.Update(ctx, &kustomization); err != nil {
			m.log.Error(err, "Failed to trigger reconciliation", "kustomization", kustomization.GetName())
		}
	}

	return nil
}

// ValidateRollback validates if a rollback can be performed
func (m *RollbackManager) ValidateRollback(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// Check if rollback is enabled
	if deployment.Spec.Rollback == nil || !deployment.Spec.Rollback.Enabled {
		return fmt.Errorf("rollback is not enabled for this deployment")
	}

	// Check if there's an ongoing sync
	if deployment.Status.Phase == "Syncing" {
		return fmt.Errorf("cannot rollback while sync is in progress")
	}

	// Get revision history
	history, err := m.getRevisionHistory(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to get revision history: %w", err)
	}

	if len(history) < 2 {
		return fmt.Errorf("insufficient revision history for rollback")
	}

	return nil
}

// GetRollbackStatus gets the status of a rollback operation
func (m *RollbackManager) GetRollbackStatus(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (string, error) {
	// Check for rollback annotations or conditions
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == "RollbackInProgress" && condition.Status == metav1.ConditionTrue {
			return "InProgress", nil
		}
		if condition.Type == "RollbackCompleted" && condition.ObservedGeneration == deployment.Generation {
			return "Completed", nil
		}
	}

	return "None", nil
}

// recordRollbackEvent records a rollback event
func (m *RollbackManager) recordRollbackEvent(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, targetRevision, message string) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-rollback-%d", deployment.Name, time.Now().Unix()),
			Namespace: deployment.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "GitOpsDeployment",
			APIVersion: "observability.io/v1beta1",
			Name:       deployment.Name,
			Namespace:  deployment.Namespace,
			UID:        deployment.UID,
		},
		Type:    corev1.EventTypeNormal,
		Reason:  "Rollback",
		Message: fmt.Sprintf("%s (target revision: %s)", message, targetRevision),
		Source: corev1.EventSource{
			Component: "rollback-manager",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	if err := m.client.Create(ctx, event); err != nil {
		m.log.Error(err, "Failed to create rollback event")
	}
}

// StoreRevisionMetrics stores metrics for a revision
func (m *RollbackManager) StoreRevisionMetrics(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, revision string, metrics map[string]float64) error {
	// Store metrics in a ConfigMap for historical tracking
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-revision-metrics", deployment.Name),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/type":              "revision-metrics",
			},
		},
		Data: make(map[string]string),
	}

	// Get existing ConfigMap
	existing := &corev1.ConfigMap{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      cm.Name,
		Namespace: cm.Namespace,
	}, existing)

	if err == nil {
		cm = existing
	}

	// Add metrics for this revision
	metricsData := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"metrics":   metrics,
	}

	metricsJSON, err := json.Marshal(metricsData)
	if err != nil {
		return err
	}

	cm.Data[revision] = string(metricsJSON)

	// Keep only the last N revisions
	if len(cm.Data) > int(deployment.Spec.Rollback.RevisionHistoryLimit) {
		// Remove oldest entries
		var revisions []string
		for rev := range cm.Data {
			revisions = append(revisions, rev)
		}
		sort.Strings(revisions)
		
		for i := 0; i < len(revisions)-int(deployment.Spec.Rollback.RevisionHistoryLimit); i++ {
			delete(cm.Data, revisions[i])
		}
	}

	// Create or update ConfigMap
	if existing.Name == "" {
		return m.client.Create(ctx, cm)
	}
	return m.client.Update(ctx, cm)
}
