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
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pmezard/go-difflib/difflib"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// DriftDetector detects and remediates configuration drift
type DriftDetector struct {
	client client.Client
	log    logr.Logger
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(client client.Client, log logr.Logger) *DriftDetector {
	return &DriftDetector{
		client: client,
		log:    log.WithName("drift-detector"),
	}
}

// DetectDrift detects configuration drift in deployed resources
func (d *DriftDetector) DetectDrift(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*observabilityv1beta1.DriftStatus, error) {
	log := d.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	driftStatus := &observabilityv1beta1.DriftStatus{
		Detected:         false,
		LastCheck:        &metav1.Time{Time: time.Now()},
		DriftedResources: []observabilityv1beta1.DriftedResource{},
	}

	// Get managed resources based on GitOps engine
	managedResources, err := d.getManagedResources(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed resources: %w", err)
	}

	// Check each managed resource for drift
	for _, resource := range managedResources {
		drift, err := d.checkResourceDrift(ctx, deployment, resource)
		if err != nil {
			log.Error(err, "Failed to check drift for resource", 
				"resource", fmt.Sprintf("%s/%s", resource.Kind, resource.Name))
			continue
		}

		if drift != nil {
			driftStatus.Detected = true
			driftStatus.DriftedResources = append(driftStatus.DriftedResources, *drift)
		}
	}

	log.Info("Drift detection completed", 
		"driftDetected", driftStatus.Detected,
		"driftedResources", len(driftStatus.DriftedResources))

	return driftStatus, nil
}

// getManagedResources gets the list of resources managed by the GitOps deployment
func (d *DriftDetector) getManagedResources(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) ([]observabilityv1beta1.ResourceStatus, error) {
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		return d.getArgoCDManagedResources(ctx, deployment)
	case "flux":
		return d.getFluxManagedResources(ctx, deployment)
	default:
		return nil, fmt.Errorf("unknown GitOps engine: %s", deployment.Spec.GitOpsEngine)
	}
}

// getArgoCDManagedResources gets resources managed by ArgoCD
func (d *DriftDetector) getArgoCDManagedResources(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) ([]observabilityv1beta1.ResourceStatus, error) {
	var allResources []observabilityv1beta1.ResourceStatus

	// List ArgoCD applications for this deployment
	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	if err := d.client.List(ctx, appList, 
		client.InNamespace("argocd"),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return nil, err
	}

	// Get resources from each application
	for _, app := range appList.Items {
		resources, err := d.getArgoCDApplicationResources(ctx, &app)
		if err != nil {
			d.log.Error(err, "Failed to get resources from ArgoCD application", "app", app.GetName())
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// getArgoCDApplicationResources gets resources from an ArgoCD application
func (d *DriftDetector) getArgoCDApplicationResources(ctx context.Context, app *unstructured.Unstructured) ([]observabilityv1beta1.ResourceStatus, error) {
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

		resources = append(resources, rs)
	}

	return resources, nil
}

// getFluxManagedResources gets resources managed by Flux
func (d *DriftDetector) getFluxManagedResources(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) ([]observabilityv1beta1.ResourceStatus, error) {
	var allResources []observabilityv1beta1.ResourceStatus

	// List Flux Kustomizations for this deployment
	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "KustomizationList",
	})

	if err := d.client.List(ctx, kustomizationList,
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return nil, err
	}

	// Get resources from each Kustomization
	for _, kustomization := range kustomizationList.Items {
		resources, err := d.getFluxKustomizationResources(ctx, &kustomization)
		if err != nil {
			d.log.Error(err, "Failed to get resources from Flux Kustomization", "kustomization", kustomization.GetName())
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// getFluxKustomizationResources gets resources from a Flux Kustomization
func (d *DriftDetector) getFluxKustomizationResources(ctx context.Context, kustomization *unstructured.Unstructured) ([]observabilityv1beta1.ResourceStatus, error) {
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

		// Parse the ID field which contains "namespace_name_group_version_kind"
		id := getStringValue(entry, "id")
		parts := strings.Split(id, "_")
		if len(parts) >= 5 {
			rs := observabilityv1beta1.ResourceStatus{
				Namespace: parts[0],
				Name:      parts[1],
				Group:     parts[2],
				Version:   parts[3],
				Kind:      parts[4],
				Status:    "Synced", // Flux doesn't provide detailed status
			}
			resources = append(resources, rs)
		}
	}

	return resources, nil
}

// checkResourceDrift checks if a specific resource has drifted
func (d *DriftDetector) checkResourceDrift(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, resource observabilityv1beta1.ResourceStatus) (*observabilityv1beta1.DriftedResource, error) {
	// Get the actual resource from cluster
	actual, err := d.getResource(ctx, resource)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Resource doesn't exist - it was removed
			return &observabilityv1beta1.DriftedResource{
				ResourceStatus: resource,
				DriftType:      "Removed",
				Details:        "Resource not found in cluster",
			}, nil
		}
		return nil, err
	}

	// Get the desired state from Git
	desired, err := d.getDesiredState(ctx, deployment, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get desired state: %w", err)
	}

	// Compare actual vs desired
	drift := d.compareResources(actual, desired, deployment.Spec.DriftDetection)
	if drift != nil {
		drift.ResourceStatus = resource
		return drift, nil
	}

	return nil, nil
}

// getResource gets a resource from the cluster
func (d *DriftDetector) getResource(ctx context.Context, resource observabilityv1beta1.ResourceStatus) (*unstructured.Unstructured, error) {
	gvk := schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	err := d.client.Get(ctx, types.NamespacedName{
		Name:      resource.Name,
		Namespace: resource.Namespace,
	}, obj)

	return obj, err
}

// getDesiredState gets the desired state from Git/GitOps engine
func (d *DriftDetector) getDesiredState(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, resource observabilityv1beta1.ResourceStatus) (*unstructured.Unstructured, error) {
	// In a real implementation, this would:
	// 1. Fetch the manifest from Git repository
	// 2. Apply any transformations (Helm values, Kustomize patches, etc.)
	// 3. Return the desired state
	
	// For now, we'll get it from the GitOps engine
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		return d.getArgoCDDesiredState(ctx, deployment, resource)
	case "flux":
		return d.getFluxDesiredState(ctx, deployment, resource)
	default:
		return nil, fmt.Errorf("unknown GitOps engine: %s", deployment.Spec.GitOpsEngine)
	}
}

// getArgoCDDesiredState gets desired state from ArgoCD
func (d *DriftDetector) getArgoCDDesiredState(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, resource observabilityv1beta1.ResourceStatus) (*unstructured.Unstructured, error) {
	// This would query ArgoCD API to get the desired manifest
	// For now, return a placeholder
	desired := &unstructured.Unstructured{}
	desired.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	})
	desired.SetName(resource.Name)
	desired.SetNamespace(resource.Namespace)

	return desired, nil
}

// getFluxDesiredState gets desired state from Flux
func (d *DriftDetector) getFluxDesiredState(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, resource observabilityv1beta1.ResourceStatus) (*unstructured.Unstructured, error) {
	// This would query Flux source controller to get the desired manifest
	// For now, return a placeholder
	desired := &unstructured.Unstructured{}
	desired.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	})
	desired.SetName(resource.Name)
	desired.SetNamespace(resource.Namespace)

	return desired, nil
}

// compareResources compares actual vs desired state
func (d *DriftDetector) compareResources(actual, desired *unstructured.Unstructured, config *observabilityv1beta1.DriftDetectionConfig) *observabilityv1beta1.DriftedResource {
	// Remove fields that should be ignored
	actualCopy := actual.DeepCopy()
	desiredCopy := desired.DeepCopy()

	// Clean metadata fields that are set by the system
	d.cleanMetadata(actualCopy)
	d.cleanMetadata(desiredCopy)

	// Remove ignored fields if configured
	if config != nil && len(config.IgnoreFields) > 0 {
		for _, field := range config.IgnoreFields {
			d.removeField(actualCopy.Object, field)
			d.removeField(desiredCopy.Object, field)
		}
	}

	// Compare the objects
	if !reflect.DeepEqual(actualCopy.Object, desiredCopy.Object) {
		// Generate diff details
		actualYAML, _ := yaml.Marshal(actualCopy.Object)
		desiredYAML, _ := yaml.Marshal(desiredCopy.Object)

		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(desiredYAML)),
			B:        difflib.SplitLines(string(actualYAML)),
			FromFile: "Desired",
			ToFile:   "Actual",
			Context:  3,
		}
		diffText, _ := difflib.GetUnifiedDiffString(diff)

		return &observabilityv1beta1.DriftedResource{
			DriftType: "Modified",
			Details:   diffText,
		}
	}

	return nil
}

// cleanMetadata removes system-managed metadata fields
func (d *DriftDetector) cleanMetadata(obj *unstructured.Unstructured) {
	metadata := obj.Object["metadata"].(map[string]interface{})
	
	// Remove system-managed fields
	delete(metadata, "uid")
	delete(metadata, "resourceVersion")
	delete(metadata, "generation")
	delete(metadata, "creationTimestamp")
	delete(metadata, "managedFields")
	delete(metadata, "selfLink")
	
	// Remove empty annotations and labels
	if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
		// Remove kubectl annotations
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		if len(annotations) == 0 {
			delete(metadata, "annotations")
		}
	}
}

// removeField removes a field from an object using a dot-separated path
func (d *DriftDetector) removeField(obj map[string]interface{}, fieldPath string) {
	parts := strings.Split(fieldPath, ".")
	current := obj

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - delete the field
			delete(current, part)
			return
		}

		// Navigate to the next level
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			// Path doesn't exist
			return
		}
	}
}

// RemediateDrift remediates detected drift
func (d *DriftDetector) RemediateDrift(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, driftStatus *observabilityv1beta1.DriftStatus) error {
	log := d.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	if !driftStatus.Detected || len(driftStatus.DriftedResources) == 0 {
		log.Info("No drift to remediate")
		return nil
	}

	log.Info("Starting drift remediation", "driftedResources", len(driftStatus.DriftedResources))

	// Remediate each drifted resource
	for _, drift := range driftStatus.DriftedResources {
		if err := d.remediateResource(ctx, deployment, drift); err != nil {
			log.Error(err, "Failed to remediate drift for resource",
				"resource", fmt.Sprintf("%s/%s", drift.Kind, drift.Name))
			return err
		}
	}

	// Trigger a sync in the GitOps engine
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		if err := d.triggerArgoCDSync(ctx, deployment); err != nil {
			return fmt.Errorf("failed to trigger ArgoCD sync: %w", err)
		}
	case "flux":
		if err := d.triggerFluxReconciliation(ctx, deployment); err != nil {
			return fmt.Errorf("failed to trigger Flux reconciliation: %w", err)
		}
	}

	log.Info("Drift remediation completed successfully")
	return nil
}

// remediateResource remediates a single drifted resource
func (d *DriftDetector) remediateResource(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, drift observabilityv1beta1.DriftedResource) error {
	switch drift.DriftType {
	case "Removed":
		// Resource was removed - recreate it
		return d.recreateResource(ctx, deployment, drift.ResourceStatus)
	case "Modified":
		// Resource was modified - restore to desired state
		return d.restoreResource(ctx, deployment, drift.ResourceStatus)
	case "Added":
		// Resource was added outside of GitOps - remove it
		return d.removeResource(ctx, drift.ResourceStatus)
	default:
		return fmt.Errorf("unknown drift type: %s", drift.DriftType)
	}
}

// recreateResource recreates a removed resource
func (d *DriftDetector) recreateResource(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, resource observabilityv1beta1.ResourceStatus) error {
	// Get desired state
	desired, err := d.getDesiredState(ctx, deployment, resource)
	if err != nil {
		return fmt.Errorf("failed to get desired state: %w", err)
	}

	// Create the resource
	if err := d.client.Create(ctx, desired); err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	d.log.Info("Recreated removed resource", "resource", fmt.Sprintf("%s/%s", resource.Kind, resource.Name))
	return nil
}

// restoreResource restores a modified resource to desired state
func (d *DriftDetector) restoreResource(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, resource observabilityv1beta1.ResourceStatus) error {
	// Get desired state
	desired, err := d.getDesiredState(ctx, deployment, resource)
	if err != nil {
		return fmt.Errorf("failed to get desired state: %w", err)
	}

	// Get current resource
	current, err := d.getResource(ctx, resource)
	if err != nil {
		return fmt.Errorf("failed to get current resource: %w", err)
	}

	// Preserve system-managed fields
	desired.SetResourceVersion(current.GetResourceVersion())
	desired.SetUID(current.GetUID())

	// Update the resource
	if err := d.client.Update(ctx, desired); err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	d.log.Info("Restored modified resource", "resource", fmt.Sprintf("%s/%s", resource.Kind, resource.Name))
	return nil
}

// removeResource removes an unwanted resource
func (d *DriftDetector) removeResource(ctx context.Context, resource observabilityv1beta1.ResourceStatus) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	})
	obj.SetName(resource.Name)
	obj.SetNamespace(resource.Namespace)

	if err := d.client.Delete(ctx, obj); err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	d.log.Info("Removed unwanted resource", "resource", fmt.Sprintf("%s/%s", resource.Kind, resource.Name))
	return nil
}

// triggerArgoCDSync triggers an ArgoCD sync
func (d *DriftDetector) triggerArgoCDSync(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// List ArgoCD applications for this deployment
	appList := &unstructured.UnstructuredList{}
	appList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "ApplicationList",
	})

	if err := d.client.List(ctx, appList,
		client.InNamespace("argocd"),
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return err
	}

	// Trigger sync for each application
	for _, app := range appList.Items {
		// Update sync annotation to trigger sync
		annotations := app.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["argocd.argoproj.io/refresh"] = "normal"
		app.SetAnnotations(annotations)

		if err := d.client.Update(ctx, &app); err != nil {
			d.log.Error(err, "Failed to trigger sync for ArgoCD application", "app", app.GetName())
			continue
		}
	}

	return nil
}

// triggerFluxReconciliation triggers a Flux reconciliation
func (d *DriftDetector) triggerFluxReconciliation(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// List Flux Kustomizations for this deployment
	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1beta2",
		Kind:    "KustomizationList",
	})

	if err := d.client.List(ctx, kustomizationList,
		client.MatchingLabels{"observability.io/gitops-deployment": deployment.Name}); err != nil {
		return err
	}

	// Trigger reconciliation for each Kustomization
	for _, kustomization := range kustomizationList.Items {
		// Update annotation to trigger reconciliation
		annotations := kustomization.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
		kustomization.SetAnnotations(annotations)

		if err := d.client.Update(ctx, &kustomization); err != nil {
			d.log.Error(err, "Failed to trigger reconciliation for Flux Kustomization", "kustomization", kustomization.GetName())
			continue
		}
	}

	return nil
}
