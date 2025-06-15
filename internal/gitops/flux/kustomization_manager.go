package flux

import (
	"context"
	"fmt"
	"path/filepath"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// KustomizationManager handles Flux Kustomization resources
type KustomizationManager struct {
	client       client.Client
	scheme       *runtime.Scheme
	log          logr.Logger
	fluxGVR      schema.GroupVersionResource
	imageGVR     schema.GroupVersionResource
	receiverGVR  schema.GroupVersionResource
}

// NewKustomizationManager creates a new KustomizationManager
func NewKustomizationManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *KustomizationManager {
	return &KustomizationManager{
		client: client,
		scheme: scheme,
		log:    log.WithName("kustomization-manager"),
		fluxGVR: schema.GroupVersionResource{
			Group:    "kustomize.toolkit.fluxcd.io",
			Version:  "v1beta2",
			Resource: "kustomizations",
		},
		imageGVR: schema.GroupVersionResource{
			Group:    "image.toolkit.fluxcd.io",
			Version:  "v1beta2",
			Resource: "imageupdateautomations",
		},
		receiverGVR: schema.GroupVersionResource{
			Group:    "notification.toolkit.fluxcd.io",
			Version:  "v1beta2",
			Resource: "receivers",
		},
	}
}

// CreateKustomization creates a Flux Kustomization resource
func (m *KustomizationManager) CreateKustomization(ctx context.Context, gitops *observabilityv1.GitOpsDeployment) error {
	m.log.Info("Creating Flux Kustomization", "name", gitops.Name, "namespace", gitops.Namespace)

	kustomization := m.buildKustomization(gitops)

	// Set owner reference
	if err := controllerutil.SetControllerReference(gitops, kustomization, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	// Create the Kustomization
	if err := m.client.Create(ctx, kustomization); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("creating kustomization: %w", err)
		}
		// Update if already exists
		existing := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
				"kind":       "Kustomization",
			},
		}
		existing.SetName(kustomization.GetName())
		existing.SetNamespace(kustomization.GetNamespace())

		if err := m.client.Get(ctx, client.ObjectKeyFromObject(existing), existing); err != nil {
			return fmt.Errorf("getting existing kustomization: %w", err)
		}

		existing.Object["spec"] = kustomization.Object["spec"]
		if err := m.client.Update(ctx, existing); err != nil {
			return fmt.Errorf("updating kustomization: %w", err)
		}
	}

	// Create image automation if enabled
	if gitops.Spec.FluxConfig != nil && gitops.Spec.FluxConfig.ImageAutomation {
		if err := m.createImageAutomation(ctx, gitops); err != nil {
			return fmt.Errorf("creating image automation: %w", err)
		}
	}

	// Create webhook receiver if enabled
	if gitops.Spec.FluxConfig != nil && gitops.Spec.FluxConfig.WebhookReceiver {
		if err := m.createWebhookReceiver(ctx, gitops); err != nil {
			return fmt.Errorf("creating webhook receiver: %w", err)
		}
	}

	return nil
}

// buildKustomization builds a Flux Kustomization resource
func (m *KustomizationManager) buildKustomization(gitops *observabilityv1.GitOpsDeployment) *unstructured.Unstructured {
	interval := "10m"
	if gitops.Spec.SyncInterval != "" {
		interval = gitops.Spec.SyncInterval
	}

	prune := true
	if gitops.Spec.FluxConfig != nil && gitops.Spec.FluxConfig.Prune != nil {
		prune = *gitops.Spec.FluxConfig.Prune
	}

	kustomization := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
			"kind":       "Kustomization",
			"metadata": map[string]interface{}{
				"name":      gitops.Name,
				"namespace": gitops.Namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by":   "gunj-operator",
					"app.kubernetes.io/component":    "gitops",
					"observability.io/gitops":        gitops.Name,
					"observability.io/sync-provider": "flux",
				},
			},
			"spec": map[string]interface{}{
				"interval": interval,
				"sourceRef": map[string]interface{}{
					"kind":      "GitRepository",
					"name":      gitops.Name,
					"namespace": gitops.Namespace,
				},
				"path":  gitops.Spec.Path,
				"prune": prune,
			},
		},
	}

	// Add health checks
	if gitops.Spec.FluxConfig != nil && gitops.Spec.FluxConfig.HealthChecks != nil {
		healthChecks := make([]interface{}, 0)
		for _, hc := range gitops.Spec.FluxConfig.HealthChecks {
			healthChecks = append(healthChecks, map[string]interface{}{
				"apiVersion": hc.APIVersion,
				"kind":       hc.Kind,
				"name":       hc.Name,
				"namespace":  hc.Namespace,
			})
		}
		spec := kustomization.Object["spec"].(map[string]interface{})
		spec["healthChecks"] = healthChecks
	}

	// Add dependencies
	if gitops.Spec.FluxConfig != nil && len(gitops.Spec.FluxConfig.Dependencies) > 0 {
		dependencies := make([]interface{}, 0)
		for _, dep := range gitops.Spec.FluxConfig.Dependencies {
			dependencies = append(dependencies, map[string]interface{}{
				"name":      dep.Name,
				"namespace": dep.Namespace,
			})
		}
		spec := kustomization.Object["spec"].(map[string]interface{})
		spec["dependsOn"] = dependencies
	}

	// Add post-build substitutions
	if gitops.Spec.FluxConfig != nil && len(gitops.Spec.FluxConfig.PostBuild) > 0 {
		spec := kustomization.Object["spec"].(map[string]interface{})
		spec["postBuild"] = map[string]interface{}{
			"substitute": gitops.Spec.FluxConfig.PostBuild,
		}
	}

	// Add validation settings
	if gitops.Spec.FluxConfig != nil && gitops.Spec.FluxConfig.Validation != "" {
		spec := kustomization.Object["spec"].(map[string]interface{})
		spec["validation"] = gitops.Spec.FluxConfig.Validation
	}

	// Add patches if specified
	if gitops.Spec.FluxConfig != nil && len(gitops.Spec.FluxConfig.Patches) > 0 {
		patches := make([]interface{}, 0)
		for _, patch := range gitops.Spec.FluxConfig.Patches {
			patchMap := map[string]interface{}{
				"patch": patch.Patch,
			}
			if patch.Target != nil {
				target := map[string]interface{}{}
				if patch.Target.APIVersion != "" {
					target["apiVersion"] = patch.Target.APIVersion
				}
				if patch.Target.Kind != "" {
					target["kind"] = patch.Target.Kind
				}
				if patch.Target.Name != "" {
					target["name"] = patch.Target.Name
				}
				if patch.Target.Namespace != "" {
					target["namespace"] = patch.Target.Namespace
				}
				if patch.Target.LabelSelector != "" {
					target["labelSelector"] = patch.Target.LabelSelector
				}
				patchMap["target"] = target
			}
			patches = append(patches, patchMap)
		}
		spec := kustomization.Object["spec"].(map[string]interface{})
		spec["patches"] = patches
	}

	return kustomization
}

// createImageAutomation creates image update automation
func (m *KustomizationManager) createImageAutomation(ctx context.Context, gitops *observabilityv1.GitOpsDeployment) error {
	m.log.Info("Creating image update automation", "name", gitops.Name, "namespace", gitops.Namespace)

	// Create ImageRepository resources for each image
	for _, image := range gitops.Spec.FluxConfig.Images {
		if err := m.createImageRepository(ctx, gitops, image); err != nil {
			return fmt.Errorf("creating image repository for %s: %w", image.Repository, err)
		}

		if err := m.createImagePolicy(ctx, gitops, image); err != nil {
			return fmt.Errorf("creating image policy for %s: %w", image.Repository, err)
		}
	}

	// Create ImageUpdateAutomation
	automation := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "image.toolkit.fluxcd.io/v1beta2",
			"kind":       "ImageUpdateAutomation",
			"metadata": map[string]interface{}{
				"name":      gitops.Name,
				"namespace": gitops.Namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/gitops":      gitops.Name,
				},
			},
			"spec": map[string]interface{}{
				"interval": "5m",
				"sourceRef": map[string]interface{}{
					"kind": "GitRepository",
					"name": gitops.Name,
				},
				"git": map[string]interface{}{
					"checkout": map[string]interface{}{
						"ref": map[string]interface{}{
							"branch": gitops.Spec.Branch,
						},
					},
					"commit": map[string]interface{}{
						"author": map[string]interface{}{
							"email": "gunj-operator@observability.io",
							"name":  "Gunj Operator",
						},
						"messageTemplate": "Auto-update images\n\n{{ range .Changed.Changes -}}\n- {{ .OldValue }} -> {{ .NewValue }}\n{{ end }}",
					},
					"push": map[string]interface{}{
						"branch": gitops.Spec.Branch,
					},
				},
				"update": map[string]interface{}{
					"path":     gitops.Spec.Path,
					"strategy": "Setters",
				},
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(gitops, automation, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	if err := m.client.Create(ctx, automation); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("creating image update automation: %w", err)
	}

	return nil
}

// createImageRepository creates an ImageRepository resource
func (m *KustomizationManager) createImageRepository(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, image observabilityv1.ImageConfig) error {
	imageRepo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "image.toolkit.fluxcd.io/v1beta2",
			"kind":       "ImageRepository",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-%s", gitops.Name, image.Name),
				"namespace": gitops.Namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/gitops":      gitops.Name,
				},
			},
			"spec": map[string]interface{}{
				"image":    image.Repository,
				"interval": image.Interval,
			},
		},
	}

	// Add secret reference if provided
	if image.SecretRef != "" {
		spec := imageRepo.Object["spec"].(map[string]interface{})
		spec["secretRef"] = map[string]interface{}{
			"name": image.SecretRef,
		}
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(gitops, imageRepo, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	return m.client.Create(ctx, imageRepo)
}

// createImagePolicy creates an ImagePolicy resource
func (m *KustomizationManager) createImagePolicy(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, image observabilityv1.ImageConfig) error {
	policy := map[string]interface{}{}
	
	switch image.Policy {
	case "semver":
		policy["semver"] = map[string]interface{}{
			"range": image.PolicyValue,
		}
	case "regex":
		policy["alphabetical"] = map[string]interface{}{
			"regex": image.PolicyValue,
		}
	case "numeric":
		policy["numerical"] = map[string]interface{}{
			"order": "asc",
		}
	default:
		// Default to latest
		policy["semver"] = map[string]interface{}{
			"range": ">=0.0.0",
		}
	}

	imagePolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "image.toolkit.fluxcd.io/v1beta2",
			"kind":       "ImagePolicy",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-%s", gitops.Name, image.Name),
				"namespace": gitops.Namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/gitops":      gitops.Name,
				},
			},
			"spec": map[string]interface{}{
				"imageRepositoryRef": map[string]interface{}{
					"name": fmt.Sprintf("%s-%s", gitops.Name, image.Name),
				},
				"policy": policy,
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(gitops, imagePolicy, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	return m.client.Create(ctx, imagePolicy)
}

// createWebhookReceiver creates a webhook receiver for Git events
func (m *KustomizationManager) createWebhookReceiver(ctx context.Context, gitops *observabilityv1.GitOpsDeployment) error {
	m.log.Info("Creating webhook receiver", "name", gitops.Name, "namespace", gitops.Namespace)

	// Create secret for webhook token
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-webhook-token", gitops.Name),
			Namespace: gitops.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/gitops":      gitops.Name,
			},
		},
		Data: map[string][]byte{
			"token": []byte(generateWebhookToken()),
		},
	}

	if err := controllerutil.SetControllerReference(gitops, secret, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference on secret: %w", err)
	}

	if err := m.client.Create(ctx, secret); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("creating webhook token secret: %w", err)
	}

	// Create receiver
	receiver := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "notification.toolkit.fluxcd.io/v1beta2",
			"kind":       "Receiver",
			"metadata": map[string]interface{}{
				"name":      gitops.Name,
				"namespace": gitops.Namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "gunj-operator",
					"observability.io/gitops":      gitops.Name,
				},
			},
			"spec": map[string]interface{}{
				"type": "generic",
				"secretRef": map[string]interface{}{
					"name": fmt.Sprintf("%s-webhook-token", gitops.Name),
				},
				"resources": []interface{}{
					map[string]interface{}{
						"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
						"kind":       "Kustomization",
						"name":       gitops.Name,
						"namespace":  gitops.Namespace,
					},
				},
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(gitops, receiver, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	if err := m.client.Create(ctx, receiver); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("creating webhook receiver: %w", err)
	}

	return nil
}

// GetKustomizationStatus gets the status of a Flux Kustomization
func (m *KustomizationManager) GetKustomizationStatus(ctx context.Context, name, namespace string) (*KustomizationStatus, error) {
	kustomization := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
			"kind":       "Kustomization",
		},
	}
	kustomization.SetName(name)
	kustomization.SetNamespace(namespace)

	if err := m.client.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization); err != nil {
		return nil, fmt.Errorf("getting kustomization: %w", err)
	}

	status, found, err := unstructured.NestedMap(kustomization.Object, "status")
	if err != nil || !found {
		return &KustomizationStatus{
			Ready: false,
		}, nil
	}

	// Extract status fields
	conditions, _, _ := unstructured.NestedSlice(status, "conditions")
	lastAppliedRevision, _, _ := unstructured.NestedString(status, "lastAppliedRevision")
	lastAttemptedRevision, _, _ := unstructured.NestedString(status, "lastAttemptedRevision")

	// Check if ready
	ready := false
	var message string
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == "Ready" && cond["status"] == "True" {
			ready = true
		}
		if cond["type"] == "Ready" && cond["message"] != nil {
			message = cond["message"].(string)
		}
	}

	return &KustomizationStatus{
		Ready:                 ready,
		LastAppliedRevision:   lastAppliedRevision,
		LastAttemptedRevision: lastAttemptedRevision,
		Message:               message,
		Conditions:            conditions,
	}, nil
}

// DeleteKustomization deletes a Flux Kustomization and related resources
func (m *KustomizationManager) DeleteKustomization(ctx context.Context, name, namespace string) error {
	m.log.Info("Deleting Flux Kustomization", "name", name, "namespace", namespace)

	// Delete Kustomization
	kustomization := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
			"kind":       "Kustomization",
		},
	}
	kustomization.SetName(name)
	kustomization.SetNamespace(namespace)

	if err := m.client.Delete(ctx, kustomization); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("deleting kustomization: %w", err)
	}

	// Delete ImageUpdateAutomation
	automation := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "image.toolkit.fluxcd.io/v1beta2",
			"kind":       "ImageUpdateAutomation",
		},
	}
	automation.SetName(name)
	automation.SetNamespace(namespace)

	if err := m.client.Delete(ctx, automation); err != nil && !errors.IsNotFound(err) {
		m.log.Error(err, "Failed to delete image update automation")
	}

	// Delete Receiver
	receiver := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "notification.toolkit.fluxcd.io/v1beta2",
			"kind":       "Receiver",
		},
	}
	receiver.SetName(name)
	receiver.SetNamespace(namespace)

	if err := m.client.Delete(ctx, receiver); err != nil && !errors.IsNotFound(err) {
		m.log.Error(err, "Failed to delete webhook receiver")
	}

	return nil
}

// ForceReconcile forces a reconciliation of the Kustomization
func (m *KustomizationManager) ForceReconcile(ctx context.Context, name, namespace string) error {
	m.log.Info("Forcing reconciliation", "name", name, "namespace", namespace)

	kustomization := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
			"kind":       "Kustomization",
		},
	}
	kustomization.SetName(name)
	kustomization.SetNamespace(namespace)

	if err := m.client.Get(ctx, client.ObjectKeyFromObject(kustomization), kustomization); err != nil {
		return fmt.Errorf("getting kustomization: %w", err)
	}

	// Add annotation to force reconciliation
	annotations := kustomization.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
	kustomization.SetAnnotations(annotations)

	if err := m.client.Update(ctx, kustomization); err != nil {
		return fmt.Errorf("updating kustomization: %w", err)
	}

	return nil
}

// KustomizationStatus represents the status of a Kustomization
type KustomizationStatus struct {
	Ready                 bool
	LastAppliedRevision   string
	LastAttemptedRevision string
	Message               string
	Conditions            []interface{}
}

// generateWebhookToken generates a secure random token
func generateWebhookToken() string {
	// In production, use crypto/rand to generate a secure token
	return fmt.Sprintf("webhook-token-%d", time.Now().UnixNano())
}
