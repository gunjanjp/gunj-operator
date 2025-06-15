/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1beta1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Status Phase constants
const (
	// PhasePending indicates the platform is pending creation
	PhasePending = "Pending"
	// PhaseInstalling indicates the platform is being installed
	PhaseInstalling = "Installing"
	// PhaseReady indicates the platform is ready
	PhaseReady = "Ready"
	// PhaseFailed indicates the platform installation failed
	PhaseFailed = "Failed"
	// PhaseUpgrading indicates the platform is being upgraded
	PhaseUpgrading = "Upgrading"
	// PhaseDegraded indicates the platform is degraded
	PhaseDegraded = "Degraded"
)

// Health Status constants
const (
	// HealthStatusHealthy indicates all components are healthy
	HealthStatusHealthy = "Healthy"
	// HealthStatusDegraded indicates some components are unhealthy
	HealthStatusDegraded = "Degraded"
	// HealthStatusUnhealthy indicates critical components are unhealthy
	HealthStatusUnhealthy = "Unhealthy"
	// HealthStatusUnknown indicates health status is unknown
	HealthStatusUnknown = "Unknown"
)

// Component Phase constants
const (
	// ComponentPhasePending indicates the component is pending
	ComponentPhasePending = "Pending"
	// ComponentPhaseDeploying indicates the component is being deployed
	ComponentPhaseDeploying = "Deploying"
	// ComponentPhaseReady indicates the component is ready
	ComponentPhaseReady = "Ready"
	// ComponentPhaseFailed indicates the component failed
	ComponentPhaseFailed = "Failed"
	// ComponentPhaseUpgrading indicates the component is upgrading
	ComponentPhaseUpgrading = "Upgrading"
)

// Condition Types
const (
	// ConditionTypeReady indicates the resource is ready
	ConditionTypeReady = "Ready"
	// ConditionTypeProgressing indicates the resource is progressing
	ConditionTypeProgressing = "Progressing"
	// ConditionTypeDegraded indicates the resource is degraded
	ConditionTypeDegraded = "Degraded"
	// ConditionTypeAvailable indicates the resource is available
	ConditionTypeAvailable = "Available"
	// ConditionTypeReconciling indicates the resource is being reconciled
	ConditionTypeReconciling = "Reconciling"
)

// Condition Reasons
const (
	// ReasonInstalling indicates installation is in progress
	ReasonInstalling = "Installing"
	// ReasonReady indicates the resource is ready
	ReasonReady = "Ready"
	// ReasonFailed indicates an operation failed
	ReasonFailed = "Failed"
	// ReasonProgressing indicates an operation is progressing
	ReasonProgressing = "Progressing"
	// ReasonDegraded indicates the resource is degraded
	ReasonDegraded = "Degraded"
	// ReasonReconciling indicates reconciliation is in progress
	ReasonReconciling = "Reconciling"
	// ReasonPaused indicates reconciliation is paused
	ReasonPaused = "Paused"
	// ReasonUpgrading indicates an upgrade is in progress
	ReasonUpgrading = "Upgrading"
	// ReasonValidationFailed indicates validation failed
	ReasonValidationFailed = "ValidationFailed"
)

// GetCondition returns the condition with the provided type
func (p *ObservabilityPlatform) GetCondition(conditionType string) *metav1.Condition {
	for i := range p.Status.Conditions {
		if p.Status.Conditions[i].Type == conditionType {
			return &p.Status.Conditions[i]
		}
	}
	return nil
}

// SetCondition updates or adds a condition
func (p *ObservabilityPlatform) SetCondition(condition metav1.Condition) {
	// Set the last transition time
	condition.LastTransitionTime = metav1.Now()

	// Check if condition already exists
	existingCondition := p.GetCondition(condition.Type)
	if existingCondition != nil {
		// Update only if status changed
		if existingCondition.Status != condition.Status {
			existingCondition.Status = condition.Status
			existingCondition.Reason = condition.Reason
			existingCondition.Message = condition.Message
			existingCondition.LastTransitionTime = condition.LastTransitionTime
		} else {
			// Update reason and message even if status didn't change
			existingCondition.Reason = condition.Reason
			existingCondition.Message = condition.Message
		}
	} else {
		// Add new condition
		p.Status.Conditions = append(p.Status.Conditions, condition)
	}
}

// IsReady returns true if the platform is ready
func (p *ObservabilityPlatform) IsReady() bool {
	return p.Status.Phase == PhaseReady
}

// IsFailed returns true if the platform is in a failed state
func (p *ObservabilityPlatform) IsFailed() bool {
	return p.Status.Phase == PhaseFailed
}

// IsProgressing returns true if the platform is installing or upgrading
func (p *ObservabilityPlatform) IsProgressing() bool {
	return p.Status.Phase == PhaseInstalling || p.Status.Phase == PhaseUpgrading
}

// IsPaused returns true if reconciliation is paused
func (p *ObservabilityPlatform) IsPaused() bool {
	return p.Spec.Paused
}

// NeedsUpdate returns true if the observed generation is behind the object generation
func (p *ObservabilityPlatform) NeedsUpdate() bool {
	return p.Generation != p.Status.ObservedGeneration
}

// UpdatePhase updates the platform phase and message
func (p *ObservabilityPlatform) UpdatePhase(phase string, message string) {
	p.Status.Phase = phase
	p.Status.Message = message
	p.Status.LastReconcileTime = &metav1.Time{Time: time.Now()}
}

// UpdateComponentStatus updates the status of a specific component
func (p *ObservabilityPlatform) UpdateComponentStatus(name string, status ComponentStatus) {
	if p.Status.ComponentStatus == nil {
		p.Status.ComponentStatus = make(map[string]ComponentStatus)
	}
	status.LastUpdate = &metav1.Time{Time: time.Now()}
	p.Status.ComponentStatus[name] = status
}

// UpdateHealth updates the platform health status
func (p *ObservabilityPlatform) UpdateHealth() {
	if p.Status.ComponentStatus == nil {
		p.Status.Health = HealthStatus{
			Status:    HealthStatusUnknown,
			LastCheck: &metav1.Time{Time: time.Now()},
		}
		return
	}

	healthyCount := int32(0)
	totalCount := int32(0)
	
	for _, status := range p.Status.ComponentStatus {
		totalCount++
		if status.Phase == ComponentPhaseReady {
			healthyCount++
		}
	}

	p.Status.Health.HealthyCount = healthyCount
	p.Status.Health.TotalCount = totalCount
	p.Status.Health.LastCheck = &metav1.Time{Time: time.Now()}

	// Determine overall health status
	switch {
	case healthyCount == totalCount && totalCount > 0:
		p.Status.Health.Status = HealthStatusHealthy
	case healthyCount == 0:
		p.Status.Health.Status = HealthStatusUnhealthy
	case healthyCount < totalCount:
		p.Status.Health.Status = HealthStatusDegraded
	default:
		p.Status.Health.Status = HealthStatusUnknown
	}
}

// SetReadyCondition sets the Ready condition
func (p *ObservabilityPlatform) SetReadyCondition(status metav1.ConditionStatus, reason, message string) {
	p.SetCondition(metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: p.Generation,
	})
}

// SetProgressingCondition sets the Progressing condition
func (p *ObservabilityPlatform) SetProgressingCondition(status metav1.ConditionStatus, reason, message string) {
	p.SetCondition(metav1.Condition{
		Type:               ConditionTypeProgressing,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: p.Generation,
	})
}

// SetDegradedCondition sets the Degraded condition
func (p *ObservabilityPlatform) SetDegradedCondition(status metav1.ConditionStatus, reason, message string) {
	p.SetCondition(metav1.Condition{
		Type:               ConditionTypeDegraded,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: p.Generation,
	})
}

// SetAvailableCondition sets the Available condition
func (p *ObservabilityPlatform) SetAvailableCondition(status metav1.ConditionStatus, reason, message string) {
	p.SetCondition(metav1.Condition{
		Type:               ConditionTypeAvailable,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: p.Generation,
	})
}

// GetEnabledComponents returns a list of enabled component names
func (p *ObservabilityPlatform) GetEnabledComponents() []string {
	components := []string{}
	
	if p.Spec.Components.Prometheus != nil && p.Spec.Components.Prometheus.Enabled {
		components = append(components, "prometheus")
	}
	if p.Spec.Components.Grafana != nil && p.Spec.Components.Grafana.Enabled {
		components = append(components, "grafana")
	}
	if p.Spec.Components.Loki != nil && p.Spec.Components.Loki.Enabled {
		components = append(components, "loki")
	}
	if p.Spec.Components.Tempo != nil && p.Spec.Components.Tempo.Enabled {
		components = append(components, "tempo")
	}
	if p.Spec.Components.OpenTelemetryCollector != nil && p.Spec.Components.OpenTelemetryCollector.Enabled {
		components = append(components, "opentelemetry-collector")
	}
	
	return components
}

// GetComponentSpec returns the spec for a specific component
func (p *ObservabilityPlatform) GetComponentSpec(name string) interface{} {
	switch name {
	case "prometheus":
		return p.Spec.Components.Prometheus
	case "grafana":
		return p.Spec.Components.Grafana
	case "loki":
		return p.Spec.Components.Loki
	case "tempo":
		return p.Spec.Components.Tempo
	case "opentelemetry-collector":
		return p.Spec.Components.OpenTelemetryCollector
	default:
		return nil
	}
}

// UpdateEndpoint updates the endpoint for a component
func (p *ObservabilityPlatform) UpdateEndpoint(component, endpoint string) {
	if p.Status.Endpoints == nil {
		p.Status.Endpoints = make(map[string]string)
	}
	p.Status.Endpoints[component] = endpoint
}

// GetEndpoint returns the endpoint for a component
func (p *ObservabilityPlatform) GetEndpoint(component string) string {
	if p.Status.Endpoints == nil {
		return ""
	}
	return p.Status.Endpoints[component]
}

// InitializeStatus initializes the status fields
func (p *ObservabilityPlatform) InitializeStatus() {
	if p.Status.Phase == "" {
		p.Status.Phase = PhasePending
	}
	if p.Status.ComponentStatus == nil {
		p.Status.ComponentStatus = make(map[string]ComponentStatus)
	}
	if p.Status.Endpoints == nil {
		p.Status.Endpoints = make(map[string]string)
	}
	if p.Status.Conditions == nil {
		p.Status.Conditions = []metav1.Condition{}
	}
	
	// Initialize health status
	p.UpdateHealth()
	
	// Set initial conditions
	p.SetReadyCondition(metav1.ConditionFalse, ReasonInstalling, "Platform is being installed")
	p.SetProgressingCondition(metav1.ConditionTrue, ReasonInstalling, "Installation in progress")
}

// GetStatusSummary returns a human-readable summary of the current status
func (p *ObservabilityPlatform) GetStatusSummary() string {
	summary := fmt.Sprintf("Phase: %s", p.Status.Phase)
	
	if p.Status.Health.Status != "" {
		summary += fmt.Sprintf(", Health: %s (%d/%d components healthy)",
			p.Status.Health.Status,
			p.Status.Health.HealthyCount,
			p.Status.Health.TotalCount)
	}
	
	if p.Status.Message != "" {
		summary += fmt.Sprintf(", Message: %s", p.Status.Message)
	}
	
	return summary
}
