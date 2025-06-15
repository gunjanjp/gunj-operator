/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package status

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/gunjanjp/gunj-operator/api/v1alpha1"
	v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// Updater provides methods to update the status subresource
type Updater struct {
	client client.Client
}

// NewUpdater creates a new status updater
func NewUpdater(client client.Client) *Updater {
	return &Updater{
		client: client,
	}
}

// UpdateObservabilityPlatformV1Beta1 updates the status of a v1beta1 ObservabilityPlatform
func (u *Updater) UpdateObservabilityPlatformV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform, updateFunc func(*v1beta1.ObservabilityPlatform)) error {
	log := log.FromContext(ctx)
	
	// Use exponential backoff for retries
	backoff := retry.DefaultBackoff
	backoff.Duration = 1 * time.Second
	backoff.Factor = 2
	backoff.Steps = 5
	
	return retry.RetryOnConflict(backoff, func() error {
		// Get the latest version
		latest := &v1beta1.ObservabilityPlatform{}
		if err := u.client.Get(ctx, types.NamespacedName{
			Name:      platform.Name,
			Namespace: platform.Namespace,
		}, latest); err != nil {
			return err
		}
		
		// Apply the update function
		updateFunc(latest)
		
		// Update observed generation
		latest.Status.ObservedGeneration = latest.Generation
		
		// Update the status
		if err := u.client.Status().Update(ctx, latest); err != nil {
			log.Error(err, "Failed to update ObservabilityPlatform status",
				"name", platform.Name,
				"namespace", platform.Namespace)
			return err
		}
		
		// Copy the updated status back to the original object
		platform.Status = latest.Status
		
		return nil
	})
}

// UpdateObservabilityPlatformV1Alpha1 updates the status of a v1alpha1 ObservabilityPlatform
func (u *Updater) UpdateObservabilityPlatformV1Alpha1(ctx context.Context, platform *v1alpha1.ObservabilityPlatform, updateFunc func(*v1alpha1.ObservabilityPlatform)) error {
	log := log.FromContext(ctx)
	
	// Use exponential backoff for retries
	backoff := retry.DefaultBackoff
	backoff.Duration = 1 * time.Second
	backoff.Factor = 2
	backoff.Steps = 5
	
	return retry.RetryOnConflict(backoff, func() error {
		// Get the latest version
		latest := &v1alpha1.ObservabilityPlatform{}
		if err := u.client.Get(ctx, types.NamespacedName{
			Name:      platform.Name,
			Namespace: platform.Namespace,
		}, latest); err != nil {
			return err
		}
		
		// Apply the update function
		updateFunc(latest)
		
		// Update observed generation
		latest.Status.ObservedGeneration = latest.Generation
		
		// Update the status
		if err := u.client.Status().Update(ctx, latest); err != nil {
			log.Error(err, "Failed to update ObservabilityPlatform status",
				"name", platform.Name,
				"namespace", platform.Namespace)
			return err
		}
		
		// Copy the updated status back to the original object
		platform.Status = latest.Status
		
		return nil
	})
}

// PatchStatusV1Beta1 patches the status of a v1beta1 ObservabilityPlatform
func (u *Updater) PatchStatusV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform, patch client.Patch) error {
	log := log.FromContext(ctx)
	
	if err := u.client.Status().Patch(ctx, platform, patch); err != nil {
		log.Error(err, "Failed to patch ObservabilityPlatform status",
			"name", platform.Name,
			"namespace", platform.Namespace)
		return err
	}
	
	return nil
}

// PatchStatusV1Alpha1 patches the status of a v1alpha1 ObservabilityPlatform
func (u *Updater) PatchStatusV1Alpha1(ctx context.Context, platform *v1alpha1.ObservabilityPlatform, patch client.Patch) error {
	log := log.FromContext(ctx)
	
	if err := u.client.Status().Patch(ctx, platform, patch); err != nil {
		log.Error(err, "Failed to patch ObservabilityPlatform status",
			"name", platform.Name,
			"namespace", platform.Namespace)
		return err
	}
	
	return nil
}

// WaitForReadyV1Beta1 waits for a v1beta1 ObservabilityPlatform to become ready
func (u *Updater) WaitForReadyV1Beta1(ctx context.Context, name, namespace string, timeout time.Duration) error {
	log := log.FromContext(ctx)
	
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		platform := &v1beta1.ObservabilityPlatform{}
		if err := u.client.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, platform); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Platform not found, waiting...", "name", name, "namespace", namespace)
				return false, nil
			}
			return false, err
		}
		
		if platform.IsReady() {
			log.Info("Platform is ready", "name", name, "namespace", namespace)
			return true, nil
		}
		
		log.V(1).Info("Platform not ready yet", 
			"name", name, 
			"namespace", namespace,
			"phase", platform.Status.Phase)
		return false, nil
	})
}

// WaitForReadyV1Alpha1 waits for a v1alpha1 ObservabilityPlatform to become ready
func (u *Updater) WaitForReadyV1Alpha1(ctx context.Context, name, namespace string, timeout time.Duration) error {
	log := log.FromContext(ctx)
	
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		platform := &v1alpha1.ObservabilityPlatform{}
		if err := u.client.Get(ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, platform); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Platform not found, waiting...", "name", name, "namespace", namespace)
				return false, nil
			}
			return false, err
		}
		
		if platform.IsReady() {
			log.Info("Platform is ready", "name", name, "namespace", namespace)
			return true, nil
		}
		
		log.V(1).Info("Platform not ready yet", 
			"name", name, 
			"namespace", namespace,
			"phase", platform.Status.Phase)
		return false, nil
	})
}

// SetPhaseV1Beta1 is a convenience method to update only the phase
func (u *Updater) SetPhaseV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform, phase string, message string) error {
	return u.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
		p.UpdatePhase(phase, message)
	})
}

// SetPhaseV1Alpha1 is a convenience method to update only the phase
func (u *Updater) SetPhaseV1Alpha1(ctx context.Context, platform *v1alpha1.ObservabilityPlatform, phase string, message string) error {
	return u.UpdateObservabilityPlatformV1Alpha1(ctx, platform, func(p *v1alpha1.ObservabilityPlatform) {
		p.UpdatePhase(phase, message)
	})
}

// SetConditionV1Beta1 is a convenience method to set a condition
func (u *Updater) SetConditionV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform, conditionType string, status metav1.ConditionStatus, reason, message string) error {
	return u.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
		p.SetCondition(metav1.Condition{
			Type:               conditionType,
			Status:             status,
			Reason:             reason,
			Message:            message,
			ObservedGeneration: p.Generation,
		})
	})
}

// SetConditionV1Alpha1 is a convenience method to set a condition
func (u *Updater) SetConditionV1Alpha1(ctx context.Context, platform *v1alpha1.ObservabilityPlatform, conditionType string, status metav1.ConditionStatus, reason, message string) error {
	return u.UpdateObservabilityPlatformV1Alpha1(ctx, platform, func(p *v1alpha1.ObservabilityPlatform) {
		p.SetCondition(metav1.Condition{
			Type:               conditionType,
			Status:             status,
			Reason:             reason,
			Message:            message,
			ObservedGeneration: p.Generation,
		})
	})
}

// UpdateComponentStatusV1Beta1 updates the status of a specific component
func (u *Updater) UpdateComponentStatusV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform, componentName string, componentStatus v1beta1.ComponentStatus) error {
	return u.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
		p.UpdateComponentStatus(componentName, componentStatus)
		p.UpdateHealth()
	})
}

// UpdateComponentStatusV1Alpha1 updates the status of a specific component
func (u *Updater) UpdateComponentStatusV1Alpha1(ctx context.Context, platform *v1alpha1.ObservabilityPlatform, componentName string, componentStatus v1alpha1.ComponentStatus) error {
	return u.UpdateObservabilityPlatformV1Alpha1(ctx, platform, func(p *v1alpha1.ObservabilityPlatform) {
		p.UpdateComponentStatus(componentName, componentStatus)
	})
}

// MarkReconcilingV1Beta1 marks the platform as reconciling
func (u *Updater) MarkReconcilingV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
	return u.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
		p.SetCondition(metav1.Condition{
			Type:               v1beta1.ConditionTypeReconciling,
			Status:             metav1.ConditionTrue,
			Reason:             v1beta1.ReasonReconciling,
			Message:            "Reconciliation in progress",
			ObservedGeneration: p.Generation,
		})
	})
}

// MarkReconciledV1Beta1 marks the platform as reconciled
func (u *Updater) MarkReconciledV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
	return u.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
		p.SetCondition(metav1.Condition{
			Type:               v1beta1.ConditionTypeReconciling,
			Status:             metav1.ConditionFalse,
			Reason:             v1beta1.ReasonReady,
			Message:            "Reconciliation complete",
			ObservedGeneration: p.Generation,
		})
		p.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
	})
}

// MarkPausedV1Beta1 marks the platform as paused
func (u *Updater) MarkPausedV1Beta1(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
	return u.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
		p.SetCondition(metav1.Condition{
			Type:               v1beta1.ConditionTypeReconciling,
			Status:             metav1.ConditionFalse,
			Reason:             v1beta1.ReasonPaused,
			Message:            "Reconciliation is paused",
			ObservedGeneration: p.Generation,
		})
	})
}

// Helper function to create error with context
func statusUpdateError(operation string, err error) error {
	return fmt.Errorf("failed to %s: %w", operation, err)
}
