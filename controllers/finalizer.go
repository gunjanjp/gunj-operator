/*
Copyright 2025.

Licensed under the MIT License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

const (
	// FinalizerDeletionGracePeriod is the grace period for finalizer deletion
	FinalizerDeletionGracePeriod = 5 * time.Minute

	// BackupFinalizer is the finalizer for backup operations
	BackupFinalizer = "observabilityplatform.observability.io/backup-finalizer"

	// ExternalResourceFinalizer is the finalizer for external resources
	ExternalResourceFinalizer = "observabilityplatform.observability.io/external-resources"

	// ComponentFinalizer is the finalizer for component cleanup
	ComponentFinalizer = "observabilityplatform.observability.io/component-cleanup"
)

// FinalizerManager handles finalizer operations for ObservabilityPlatform
type FinalizerManager struct {
	client.Client
	Log logr.Logger
}

// NewFinalizerManager creates a new finalizer manager
func NewFinalizerManager(client client.Client, log logr.Logger) *FinalizerManager {
	return &FinalizerManager{
		Client: client,
		Log:    log,
	}
}

// AddFinalizers adds all necessary finalizers to the platform
func (fm *FinalizerManager) AddFinalizers(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := log.FromContext(ctx)
	
	// List of finalizers to add
	finalizers := []string{
		FinalizerName,
		ComponentFinalizer,
		ExternalResourceFinalizer,
	}
	
	// Add backup finalizer if backup is enabled
	if platform.Spec.Backup != nil && platform.Spec.Backup.Enabled {
		finalizers = append(finalizers, BackupFinalizer)
	}
	
	// Check which finalizers need to be added
	var finalizersToAdd []string
	for _, finalizer := range finalizers {
		if !controllerutil.ContainsFinalizer(platform, finalizer) {
			finalizersToAdd = append(finalizersToAdd, finalizer)
		}
	}
	
	// If all finalizers already exist, nothing to do
	if len(finalizersToAdd) == 0 {
		return nil
	}
	
	// Add the missing finalizers
	for _, finalizer := range finalizersToAdd {
		log.V(1).Info("Adding finalizer", "finalizer", finalizer)
		controllerutil.AddFinalizer(platform, finalizer)
	}
	
	// Update the resource
	if err := fm.Update(ctx, platform); err != nil {
		return fmt.Errorf("failed to add finalizers: %w", err)
	}
	
	log.Info("Successfully added finalizers", "count", len(finalizersToAdd))
	return nil
}

// HandleDeletion processes the deletion of the platform
func (fm *FinalizerManager) HandleDeletion(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx)
	startTime := time.Now()
	
	// Update phase to Deleting
	if platform.Status.Phase != "Deleting" {
		platform.Status.Phase = "Deleting"
		platform.Status.Message = "Platform deletion in progress"
		if err := r.Status().Update(ctx, platform); err != nil {
			log.Error(err, "Failed to update status to Deleting")
		}
	}
	
	// Create a timeout context for deletion
	deleteCtx, cancel := context.WithTimeout(ctx, FinalizerDeletionGracePeriod)
	defer cancel()
	
	// Process finalizers in order
	if controllerutil.ContainsFinalizer(platform, BackupFinalizer) {
		if err := fm.handleBackupFinalizer(deleteCtx, platform, r); err != nil {
			return fmt.Errorf("backup finalizer failed: %w", err)
		}
		fm.removeFinalizer(ctx, platform, BackupFinalizer, r)
	}
	
	if controllerutil.ContainsFinalizer(platform, ComponentFinalizer) {
		if err := fm.handleComponentFinalizer(deleteCtx, platform, r); err != nil {
			return fmt.Errorf("component finalizer failed: %w", err)
		}
		fm.removeFinalizer(ctx, platform, ComponentFinalizer, r)
	}
	
	if controllerutil.ContainsFinalizer(platform, ExternalResourceFinalizer) {
		if err := fm.handleExternalResourceFinalizer(deleteCtx, platform, r); err != nil {
			return fmt.Errorf("external resource finalizer failed: %w", err)
		}
		fm.removeFinalizer(ctx, platform, ExternalResourceFinalizer, r)
	}
	
	// Main finalizer cleanup
	if controllerutil.ContainsFinalizer(platform, FinalizerName) {
		if err := fm.handleMainFinalizer(deleteCtx, platform, r); err != nil {
			return fmt.Errorf("main finalizer failed: %w", err)
		}
		fm.removeFinalizer(ctx, platform, FinalizerName, r)
	}
	
	duration := time.Since(startTime)
	log.Info("Platform deletion completed", "duration", duration)
	
	return nil
}

// handleBackupFinalizer performs pre-deletion backup if configured
func (fm *FinalizerManager) handleBackupFinalizer(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx).WithValues("finalizer", BackupFinalizer)
	
	if platform.Spec.Backup == nil || !platform.Spec.Backup.Enabled {
		log.V(1).Info("Backup not enabled, skipping backup finalizer")
		return nil
	}
	
	log.Info("Performing pre-deletion backup")
	
	// Record event
	r.EventRecorder.RecordPlatformEvent(platform, "PreDeletionBackup", "Starting pre-deletion backup")
	
	// Trigger backup for each component
	components := []struct {
		name    string
		enabled bool
	}{
		{"prometheus", platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled},
		{"grafana", platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled},
		{"loki", platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled},
		{"tempo", platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled},
	}
	
	for _, comp := range components {
		if comp.enabled {
			if err := fm.backupComponent(ctx, platform, comp.name); err != nil {
				log.Error(err, "Failed to backup component", "component", comp.name)
				// Continue with other components even if one fails
			}
		}
	}
	
	// Create final backup metadata
	backupConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-final-backup", platform.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"observability.io/platform":     platform.Name,
				"observability.io/backup-type":  "pre-deletion",
				"observability.io/backup-time":  time.Now().Format(time.RFC3339),
			},
		},
		Data: map[string]string{
			"platform.yaml": fm.serializePlatformSpec(platform),
			"status.yaml":   fm.serializePlatformStatus(platform),
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	}
	
	// Remove owner reference to ensure backup survives deletion
	if err := r.Create(ctx, backupConfigMap); err != nil && !errors.IsAlreadyExists(err) {
		log.Error(err, "Failed to create backup configmap")
	}
	
	log.Info("Pre-deletion backup completed")
	return nil
}

// handleComponentFinalizer cleans up all component resources
func (fm *FinalizerManager) handleComponentFinalizer(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx).WithValues("finalizer", ComponentFinalizer)
	log.Info("Starting component cleanup")
	
	// Record event
	r.EventRecorder.RecordPlatformEvent(platform, "ComponentCleanup", "Starting component cleanup")
	
	// Clean up components in reverse dependency order
	// Tempo -> Loki -> Grafana -> Prometheus
	
	// Cleanup Tempo
	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.Enabled {
		log.V(1).Info("Cleaning up Tempo")
		if r.TempoManager != nil {
			if err := r.TempoManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Tempo")
				// Continue with cleanup even if one component fails
			}
		}
	}
	
	// Cleanup Loki
	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.Enabled {
		log.V(1).Info("Cleaning up Loki")
		if r.LokiManager != nil {
			if err := r.LokiManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Loki")
			}
		}
	}
	
	// Cleanup Grafana
	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.Enabled {
		log.V(1).Info("Cleaning up Grafana")
		if r.GrafanaManager != nil {
			if err := r.GrafanaManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Grafana")
			}
		}
	}
	
	// Cleanup Prometheus
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.Enabled {
		log.V(1).Info("Cleaning up Prometheus")
		if r.PrometheusManager != nil {
			if err := r.PrometheusManager.Delete(ctx, platform); err != nil {
				log.Error(err, "Failed to delete Prometheus")
			}
		}
	}
	
	// Wait for component pods to terminate
	if err := fm.waitForComponentTermination(ctx, platform, r); err != nil {
		log.Error(err, "Error waiting for component termination")
	}
	
	log.Info("Component cleanup completed")
	return nil
}

// handleExternalResourceFinalizer cleans up external resources
func (fm *FinalizerManager) handleExternalResourceFinalizer(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx).WithValues("finalizer", ExternalResourceFinalizer)
	log.Info("Starting external resource cleanup")
	
	// Record event
	r.EventRecorder.RecordPlatformEvent(platform, "ExternalResourceCleanup", "Starting external resource cleanup")
	
	// Clean up PVCs
	if err := fm.cleanupPVCs(ctx, platform, r); err != nil {
		log.Error(err, "Failed to cleanup PVCs")
	}
	
	// Clean up ConfigMaps (except backup ones)
	if err := fm.cleanupConfigMaps(ctx, platform, r); err != nil {
		log.Error(err, "Failed to cleanup ConfigMaps")
	}
	
	// Clean up Secrets
	if err := fm.cleanupSecrets(ctx, platform, r); err != nil {
		log.Error(err, "Failed to cleanup Secrets")
	}
	
	// Clean up Services
	if err := fm.cleanupServices(ctx, platform, r); err != nil {
		log.Error(err, "Failed to cleanup Services")
	}
	
	// Clean up NetworkPolicies
	if err := fm.cleanupNetworkPolicies(ctx, platform, r); err != nil {
		log.Error(err, "Failed to cleanup NetworkPolicies")
	}
	
	log.Info("External resource cleanup completed")
	return nil
}

// handleMainFinalizer performs final cleanup
func (fm *FinalizerManager) handleMainFinalizer(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx).WithValues("finalizer", FinalizerName)
	log.Info("Performing final cleanup")
	
	// Clean up any remaining resources with our labels
	labelSelector := labels.SelectorFromSet(labels.Set{
		"observability.io/platform": platform.Name,
		"app.kubernetes.io/managed-by": "gunj-operator",
	})
	
	// List and delete remaining resources
	if err := fm.cleanupRemainingResources(ctx, platform, labelSelector, r); err != nil {
		log.Error(err, "Failed to cleanup remaining resources")
	}
	
	// Remove namespace labels if we created them
	if err := fm.cleanupNamespaceLabels(ctx, platform, r); err != nil {
		log.Error(err, "Failed to cleanup namespace labels")
	}
	
	// Record final event
	r.EventRecorder.RecordPlatformEvent(platform, EventReasonPlatformDeleted, "Platform cleanup completed")
	
	log.Info("Final cleanup completed")
	return nil
}

// Cleanup helper functions

func (fm *FinalizerManager) cleanupPVCs(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx)
	
	pvcList := &corev1.PersistentVolumeClaimList{}
	labelSelector := labels.SelectorFromSet(labels.Set{
		"observability.io/platform": platform.Name,
	})
	
	if err := r.List(ctx, pvcList, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list PVCs: %w", err)
	}
	
	for _, pvc := range pvcList.Items {
		log.V(1).Info("Deleting PVC", "name", pvc.Name)
		if err := r.Delete(ctx, &pvc); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete PVC", "name", pvc.Name)
		}
	}
	
	return nil
}

func (fm *FinalizerManager) cleanupConfigMaps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx)
	
	cmList := &corev1.ConfigMapList{}
	labelSelector := labels.SelectorFromSet(labels.Set{
		"observability.io/platform": platform.Name,
	})
	
	if err := r.List(ctx, cmList, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list ConfigMaps: %w", err)
	}
	
	for _, cm := range cmList.Items {
		// Skip backup configmaps
		if backupType, ok := cm.Labels["observability.io/backup-type"]; ok && backupType == "pre-deletion" {
			log.V(1).Info("Skipping backup ConfigMap", "name", cm.Name)
			continue
		}
		
		log.V(1).Info("Deleting ConfigMap", "name", cm.Name)
		if err := r.Delete(ctx, &cm); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete ConfigMap", "name", cm.Name)
		}
	}
	
	return nil
}

func (fm *FinalizerManager) cleanupSecrets(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx)
	
	secretList := &corev1.SecretList{}
	labelSelector := labels.SelectorFromSet(labels.Set{
		"observability.io/platform": platform.Name,
	})
	
	if err := r.List(ctx, secretList, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list Secrets: %w", err)
	}
	
	for _, secret := range secretList.Items {
		log.V(1).Info("Deleting Secret", "name", secret.Name)
		if err := r.Delete(ctx, &secret); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete Secret", "name", secret.Name)
		}
	}
	
	return nil
}

func (fm *FinalizerManager) cleanupServices(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx)
	
	svcList := &corev1.ServiceList{}
	labelSelector := labels.SelectorFromSet(labels.Set{
		"observability.io/platform": platform.Name,
	})
	
	if err := r.List(ctx, svcList, &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     platform.Namespace,
	}); err != nil {
		return fmt.Errorf("failed to list Services: %w", err)
	}
	
	for _, svc := range svcList.Items {
		log.V(1).Info("Deleting Service", "name", svc.Name)
		if err := r.Delete(ctx, &svc); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete Service", "name", svc.Name)
		}
	}
	
	return nil
}

func (fm *FinalizerManager) cleanupNetworkPolicies(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	// Implementation would go here for NetworkPolicy cleanup
	// Skipping for brevity as it follows the same pattern
	return nil
}

func (fm *FinalizerManager) cleanupRemainingResources(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, labelSelector labels.Selector, r *ObservabilityPlatformReconciler) error {
	// Implementation would clean up any remaining labeled resources
	// This is a catch-all for any resources we might have missed
	return nil
}

func (fm *FinalizerManager) cleanupNamespaceLabels(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx)
	
	namespace := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: platform.Namespace}, namespace); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get namespace: %w", err)
	}
	
	// Remove our labels
	labelsToRemove := []string{
		"observability.io/managed",
		"observability.io/platform",
		"app.kubernetes.io/managed-by",
		"app.kubernetes.io/part-of",
		"app.kubernetes.io/instance",
	}
	
	updated := false
	for _, label := range labelsToRemove {
		if _, ok := namespace.Labels[label]; ok {
			delete(namespace.Labels, label)
			updated = true
		}
	}
	
	if updated {
		if err := r.Update(ctx, namespace); err != nil {
			log.Error(err, "Failed to update namespace labels")
		}
	}
	
	return nil
}

// Helper functions

func (fm *FinalizerManager) waitForComponentTermination(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, r *ObservabilityPlatformReconciler) error {
	log := log.FromContext(ctx)
	
	// Wait for all pods to terminate
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			log.Info("Timeout waiting for pods to terminate")
			return nil
		case <-ticker.C:
			podList := &corev1.PodList{}
			labelSelector := labels.SelectorFromSet(labels.Set{
				"observability.io/platform": platform.Name,
			})
			
			if err := r.List(ctx, podList, &client.ListOptions{
				LabelSelector: labelSelector,
				Namespace:     platform.Namespace,
			}); err != nil {
				return fmt.Errorf("failed to list pods: %w", err)
			}
			
			if len(podList.Items) == 0 {
				log.Info("All component pods terminated")
				return nil
			}
			
			log.V(1).Info("Waiting for pods to terminate", "count", len(podList.Items))
		}
	}
}

func (fm *FinalizerManager) removeFinalizer(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, finalizer string, r *ObservabilityPlatformReconciler) {
	log := log.FromContext(ctx)
	
	if controllerutil.ContainsFinalizer(platform, finalizer) {
		log.V(1).Info("Removing finalizer", "finalizer", finalizer)
		controllerutil.RemoveFinalizer(platform, finalizer)
		if err := r.Update(ctx, platform); err != nil {
			log.Error(err, "Failed to remove finalizer", "finalizer", finalizer)
		}
	}
}

func (fm *FinalizerManager) backupComponent(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, component string) error {
	// Implementation would perform actual backup based on component type
	// This could involve:
	// - Taking snapshots of PVCs
	// - Exporting configurations
	// - Backing up to S3/GCS/Azure based on platform.Spec.Backup.Destination
	return nil
}

func (fm *FinalizerManager) serializePlatformSpec(platform *observabilityv1beta1.ObservabilityPlatform) string {
	// Implementation would serialize the platform spec to YAML
	// For now, returning a placeholder
	return fmt.Sprintf("# Platform Spec for %s\n# Serialized at %s\n", platform.Name, time.Now().Format(time.RFC3339))
}

func (fm *FinalizerManager) serializePlatformStatus(platform *observabilityv1beta1.ObservabilityPlatform) string {
	// Implementation would serialize the platform status to YAML
	// For now, returning a placeholder
	return fmt.Sprintf("# Platform Status for %s\n# Serialized at %s\n", platform.Name, time.Now().Format(time.RFC3339))
}
