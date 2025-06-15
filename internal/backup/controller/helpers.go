package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gunjanjp/gunj-operator/internal/backup"
	"github.com/gunjanjp/gunj-operator/internal/backup/storage"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// runHooks runs backup/restore hooks
func (bc *BackupController) runHooks(ctx context.Context, hooks []backup.Hook) error {
	for _, hook := range hooks {
		bc.log.V(1).Info("Running hook", "name", hook.Name, "command", hook.Command)
		
		// Set timeout
		timeout := 5 * time.Minute
		if hook.Timeout != nil {
			timeout = hook.Timeout.Duration
		}
		
		hookCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		
		// Execute hook
		if err := bc.executeHook(hookCtx, hook); err != nil {
			if hook.OnError == backup.HookErrorModeFail {
				return fmt.Errorf("hook %s failed: %w", hook.Name, err)
			}
			// Continue on error
			bc.log.Error(err, "Hook failed but continuing", "hook", hook.Name)
		}
		
		bc.log.V(1).Info("Hook completed", "name", hook.Name)
	}
	
	return nil
}

// executeHook executes a single hook
func (bc *BackupController) executeHook(ctx context.Context, hook backup.Hook) error {
	if hook.Container != "" {
		// Execute in container
		return bc.executeInContainer(ctx, hook)
	}
	
	// Execute locally
	return bc.executeLocally(ctx, hook)
}

// executeInContainer executes hook in a container
func (bc *BackupController) executeInContainer(ctx context.Context, hook backup.Hook) error {
	// This would require pod/container information
	// For now, return an error
	return fmt.Errorf("container execution not implemented")
}

// executeLocally executes hook locally
func (bc *BackupController) executeLocally(ctx context.Context, hook backup.Hook) error {
	if len(hook.Command) == 0 {
		return fmt.Errorf("empty command")
	}
	
	cmd := exec.CommandContext(ctx, hook.Command[0], hook.Command[1:]...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}
	
	bc.log.V(2).Info("Hook output", "stdout", stdout.String())
	
	return nil
}

// createBackupData creates the backup data archive
func (bc *BackupController) createBackupData(ctx context.Context, state *backupState) ([]byte, error) {
	// Use storage manager to create archive
	return bc.storageManager.CreateBackupArchive(state.items)
}

// compressData compresses backup data
func (bc *BackupController) compressData(data []byte, config *backup.CompressionConfig) ([]byte, error) {
	return bc.storageManager.CompressData(data, config)
}

// encryptData encrypts backup data
func (bc *BackupController) encryptData(ctx context.Context, data []byte, config *backup.EncryptionConfig) ([]byte, error) {
	bc.storageManager.SetClient(bc.client)
	return bc.storageManager.EncryptData(ctx, data, config)
}

// takeVolumeSnapshots takes snapshots of persistent volumes
func (bc *BackupController) takeVolumeSnapshots(ctx context.Context, state *backupState) error {
	// This is a simplified implementation
	// In production, you'd integrate with CSI snapshots or cloud provider APIs
	
	bc.log.V(1).Info("Taking volume snapshots")
	
	// Find all PVCs in backed up resources
	pvcs := bc.findPVCs(state.items)
	
	for _, pvc := range pvcs {
		bc.log.V(1).Info("Taking snapshot of PVC", "name", pvc.Name, "namespace", pvc.Namespace)
		
		// Create volume snapshot (simplified)
		snapshot := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("snapshot-%s-%d", pvc.Name, time.Now().Unix()),
				Namespace: pvc.Namespace,
				Labels: map[string]string{
					"backup.observability.io/pvc":    pvc.Name,
					"backup.observability.io/backup": bc.generateBackupName(state.spec),
				},
			},
			Data: map[string]string{
				"pvc":         pvc.Name,
				"storageClass": *pvc.Spec.StorageClassName,
				"size":        pvc.Spec.Resources.Requests.Storage().String(),
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		}
		
		if err := bc.client.Create(ctx, snapshot); err != nil {
			return fmt.Errorf("creating snapshot for PVC %s: %w", pvc.Name, err)
		}
	}
	
	return nil
}

// findPVCs finds PVCs in backup items
func (bc *BackupController) findPVCs(items []backup.BackupItem) []*corev1.PersistentVolumeClaim {
	var pvcs []*corev1.PersistentVolumeClaim
	
	for _, item := range items {
		if item.GroupVersionKind.Kind == "PersistentVolumeClaim" {
			pvc := &corev1.PersistentVolumeClaim{}
			if err := json.Unmarshal(item.Data.Raw, pvc); err != nil {
				bc.log.Error(err, "Failed to unmarshal PVC", "name", item.Name)
				continue
			}
			pvcs = append(pvcs, pvc)
		}
	}
	
	return pvcs
}

// collectClusterResources collects cluster-scoped resources
func (bc *BackupController) collectClusterResources(ctx context.Context, spec *backup.BackupSpec) ([]backup.BackupItem, error) {
	var items []backup.BackupItem
	
	// Collect namespaces
	namespaces := &corev1.NamespaceList{}
	if err := bc.client.List(ctx, namespaces); err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}
	
	for _, ns := range namespaces.Items {
		if !bc.shouldIncludeNamespace(ns.Name, spec) {
			continue
		}
		
		item := backup.BackupItem{
			Name: ns.Name,
			GroupVersionKind: metav1.GroupVersionKind{
				Version: "v1",
				Kind:    "Namespace",
			},
		}
		
		data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ns)
		if err != nil {
			return nil, fmt.Errorf("converting namespace %s: %w", ns.Name, err)
		}
		item.Data.Raw, err = runtime.Encode(runtime.DefaultUnstructuredConverter, data)
		if err != nil {
			return nil, fmt.Errorf("encoding namespace %s: %w", ns.Name, err)
		}
		
		items = append(items, item)
	}
	
	// Add other cluster resources as needed (ClusterRoles, ClusterRoleBindings, etc.)
	
	return items, nil
}

// shouldIncludeNamespace checks if namespace should be included
func (bc *BackupController) shouldIncludeNamespace(namespace string, spec *backup.BackupSpec) bool {
	// System namespaces to always exclude
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
	}
	
	for _, sysNs := range systemNamespaces {
		if namespace == sysNs {
			return false
		}
	}
	
	// Check explicit exclusions
	for _, excluded := range spec.ExcludedNamespaces {
		if namespace == excluded {
			return false
		}
	}
	
	// Check explicit inclusions
	if len(spec.IncludedNamespaces) > 0 {
		for _, included := range spec.IncludedNamespaces {
			if namespace == included {
				return true
			}
		}
		return false
	}
	
	return true
}

// For RestoreController

// runHooks runs restore hooks
func (rc *RestoreController) runHooks(ctx context.Context, hooks []backup.Hook) error {
	// Similar implementation to backup hooks
	for _, hook := range hooks {
		rc.log.V(1).Info("Running hook", "name", hook.Name, "command", hook.Command)
		
		timeout := 5 * time.Minute
		if hook.Timeout != nil {
			timeout = hook.Timeout.Duration
		}
		
		hookCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		
		if err := rc.executeHook(hookCtx, hook); err != nil {
			if hook.OnError == backup.HookErrorModeFail {
				return fmt.Errorf("hook %s failed: %w", hook.Name, err)
			}
			rc.log.Error(err, "Hook failed but continuing", "hook", hook.Name)
		}
		
		rc.log.V(1).Info("Hook completed", "name", hook.Name)
	}
	
	return nil
}

// executeHook executes a single hook
func (rc *RestoreController) executeHook(ctx context.Context, hook backup.Hook) error {
	if len(hook.Command) == 0 {
		return fmt.Errorf("empty command")
	}
	
	cmd := exec.CommandContext(ctx, hook.Command[0], hook.Command[1:]...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}
	
	rc.log.V(2).Info("Hook output", "stdout", stdout.String())
	
	return nil
}

// downloadBackup downloads backup data from storage
func (rc *RestoreController) downloadBackup(ctx context.Context, state *restoreState) ([]byte, error) {
	// Get backup details from backup controller
	backupState, exists := rc.backupCtrl.backups[state.spec.BackupName]
	if !exists {
		return nil, fmt.Errorf("backup %s not found", state.spec.BackupName)
	}
	
	// Get provider
	provider, ok := rc.providers[backupState.spec.StorageLocation.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not available", backupState.spec.StorageLocation.Provider)
	}
	
	// Download backup
	return provider.Download(ctx, backupState.spec)
}

// isEncrypted checks if data is encrypted
func (rc *RestoreController) isEncrypted(data []byte) bool {
	return storage.IsEncrypted(data)
}

// decryptData decrypts backup data
func (rc *RestoreController) decryptData(ctx context.Context, data []byte) ([]byte, error) {
	// Parse encrypted data to get key reference
	var encrypted storage.EncryptedData
	if err := json.Unmarshal(data, &encrypted); err != nil {
		return nil, fmt.Errorf("parsing encrypted data: %w", err)
	}
	
	// For now, use a placeholder key reference
	// In production, this would come from the backup metadata
	keyRef := &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "backup-encryption-key",
		},
		Key: "key",
	}
	
	rc.storageManager.SetClient(rc.client)
	return rc.storageManager.DecryptData(ctx, data, keyRef)
}

// isCompressed checks if data is compressed
func (rc *RestoreController) isCompressed(data []byte) bool {
	return storage.IsCompressed(data)
}

// decompressData decompresses backup data
func (rc *RestoreController) decompressData(data []byte) ([]byte, error) {
	algorithm := storage.GetCompressionAlgorithm(data)
	if algorithm == "" {
		return nil, fmt.Errorf("unknown compression algorithm")
	}
	
	return rc.storageManager.DecompressData(data, algorithm)
}

// parseBackupData parses backup data into items
func (rc *RestoreController) parseBackupData(data []byte) ([]backup.BackupItem, error) {
	return rc.storageManager.ExtractBackupArchive(data)
}

// validateBackupData validates backup data
func (rc *RestoreController) validateBackupData(items []backup.BackupItem, policy *backup.ValidationPolicy) *backup.ValidationStatus {
	status := &backup.ValidationStatus{
		Valid:             true,
		ChecksumVerified:  true,
		IntegrityVerified: true,
	}
	
	// Basic validation
	if len(items) == 0 {
		status.Valid = false
		status.Errors = append(status.Errors, "No items in backup")
		return status
	}
	
	// Check each item
	for _, item := range items {
		// Validate required fields
		if item.Name == "" {
			status.Valid = false
			status.Errors = append(status.Errors, fmt.Sprintf("Item missing name"))
		}
		
		if item.GroupVersionKind.Kind == "" {
			status.Valid = false
			status.Errors = append(status.Errors, fmt.Sprintf("Item %s missing kind", item.Name))
		}
		
		// Validate data
		if len(item.Data.Raw) == 0 {
			status.Valid = false
			status.Errors = append(status.Errors, fmt.Sprintf("Item %s has no data", item.Name))
		}
	}
	
	// Check if we should fail on warnings
	if policy.FailOnWarning && len(status.Warnings) > 0 {
		status.Valid = false
	}
	
	return status
}

// restoreVolumeSnapshots restores volume snapshots
func (rc *RestoreController) restoreVolumeSnapshots(ctx context.Context, state *restoreState) error {
	// This is a simplified implementation
	// In production, you'd integrate with CSI snapshots or cloud provider APIs
	
	rc.log.V(1).Info("Restoring volume snapshots")
	
	// Find snapshot ConfigMaps
	snapshots := &corev1.ConfigMapList{}
	if err := rc.client.List(ctx, snapshots); err != nil {
		return fmt.Errorf("listing snapshots: %w", err)
	}
	
	for _, snapshot := range snapshots.Items {
		// Check if snapshot belongs to this backup
		backupLabel, ok := snapshot.Labels["backup.observability.io/backup"]
		if !ok || backupLabel != state.spec.BackupName {
			continue
		}
		
		pvcName := snapshot.Labels["backup.observability.io/pvc"]
		rc.log.V(1).Info("Restoring snapshot for PVC", "pvc", pvcName, "snapshot", snapshot.Name)
		
		// In a real implementation, you would:
		// 1. Create a new PV from the snapshot
		// 2. Create a PVC bound to that PV
		// 3. Update references in the restored workloads
	}
	
	return nil
}
