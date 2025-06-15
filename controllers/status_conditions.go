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

package controllers

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition Types for ObservabilityPlatform
const (
	// ConditionReady indicates the platform is fully operational
	ConditionReady = "Ready"

	// ConditionProgressing indicates the platform is being reconciled
	ConditionProgressing = "Progressing"

	// ConditionDegraded indicates the platform is operational but degraded
	ConditionDegraded = "Degraded"

	// ConditionError indicates the platform has errors
	ConditionError = "Error"

	// Component-specific conditions
	ConditionPrometheusReady = "PrometheusReady"
	ConditionGrafanaReady    = "GrafanaReady"
	ConditionLokiReady       = "LokiReady"
	ConditionTempoReady      = "TempoReady"

	// Resource conditions
	ConditionResourcesAvailable = "ResourcesAvailable"
	ConditionStorageReady       = "StorageReady"
	ConditionNetworkReady       = "NetworkReady"
)

// Condition Reasons
const (
	// General reasons
	ReasonReconciling       = "Reconciling"
	ReasonReady             = "Ready"
	ReasonFailed            = "Failed"
	ReasonInProgress        = "InProgress"
	ReasonValidationFailed  = "ValidationFailed"
	ReasonDependencyMissing = "DependencyMissing"

	// Component reasons
	ReasonComponentDeploying   = "ComponentDeploying"
	ReasonComponentReady       = "ComponentReady"
	ReasonComponentFailed      = "ComponentFailed"
	ReasonComponentUpgrading   = "ComponentUpgrading"
	ReasonComponentScaling     = "ComponentScaling"
	ReasonComponentConfiguring = "ComponentConfiguring"

	// Resource reasons
	ReasonInsufficientResources = "InsufficientResources"
	ReasonStorageUnavailable    = "StorageUnavailable"
	ReasonNetworkPolicyFailed   = "NetworkPolicyFailed"
	ReasonQuotaExceeded         = "QuotaExceeded"
)

// Phase transitions and their meanings
const (
	PhasePending    = "Pending"
	PhaseInstalling = "Installing"
	PhaseUpgrading  = "Upgrading"
	PhaseReady      = "Ready"
	PhaseDegraded   = "Degraded"
	PhaseFailed     = "Failed"
	PhaseDeleting   = "Deleting"
	PhaseUnknown    = "Unknown"
)

// ConditionBuilder helps build conditions with consistent formatting
type ConditionBuilder struct {
	conditions []metav1.Condition
}

// NewConditionBuilder creates a new condition builder
func NewConditionBuilder() *ConditionBuilder {
	return &ConditionBuilder{
		conditions: []metav1.Condition{},
	}
}

// WithCondition adds a condition to the builder
func (cb *ConditionBuilder) WithCondition(condType string, status metav1.ConditionStatus, reason, message string) *ConditionBuilder {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: 0, // Will be set by the controller
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	cb.conditions = append(cb.conditions, condition)
	return cb
}

// WithTrueCondition adds a True condition
func (cb *ConditionBuilder) WithTrueCondition(condType, reason, message string) *ConditionBuilder {
	return cb.WithCondition(condType, metav1.ConditionTrue, reason, message)
}

// WithFalseCondition adds a False condition
func (cb *ConditionBuilder) WithFalseCondition(condType, reason, message string) *ConditionBuilder {
	return cb.WithCondition(condType, metav1.ConditionFalse, reason, message)
}

// WithUnknownCondition adds an Unknown condition
func (cb *ConditionBuilder) WithUnknownCondition(condType, reason, message string) *ConditionBuilder {
	return cb.WithCondition(condType, metav1.ConditionUnknown, reason, message)
}

// Build returns the built conditions
func (cb *ConditionBuilder) Build() []metav1.Condition {
	return cb.conditions
}

// ConditionUtils provides utility functions for working with conditions
type ConditionUtils struct{}

// NewConditionUtils creates a new condition utils instance
func NewConditionUtils() *ConditionUtils {
	return &ConditionUtils{}
}

// SetCondition updates or adds a condition to the list
func (cu *ConditionUtils) SetCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	for i, condition := range conditions {
		if condition.Type == newCondition.Type {
			if condition.Status != newCondition.Status ||
				condition.Reason != newCondition.Reason ||
				condition.Message != newCondition.Message {
				conditions[i] = newCondition
			} else {
				// Keep the existing LastTransitionTime if nothing changed
				newCondition.LastTransitionTime = condition.LastTransitionTime
				conditions[i] = newCondition
			}
			return conditions
		}
	}
	// Condition not found, add it
	return append(conditions, newCondition)
}

// GetCondition returns the condition with the given type
func (cu *ConditionUtils) GetCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for _, condition := range conditions {
		if condition.Type == condType {
			return &condition
		}
	}
	return nil
}

// IsConditionTrue returns true if the condition is present and True
func (cu *ConditionUtils) IsConditionTrue(conditions []metav1.Condition, condType string) bool {
	condition := cu.GetCondition(conditions, condType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsConditionFalse returns true if the condition is present and False
func (cu *ConditionUtils) IsConditionFalse(conditions []metav1.Condition, condType string) bool {
	condition := cu.GetCondition(conditions, condType)
	return condition != nil && condition.Status == metav1.ConditionFalse
}

// RemoveCondition removes a condition from the list
func (cu *ConditionUtils) RemoveCondition(conditions []metav1.Condition, condType string) []metav1.Condition {
	var result []metav1.Condition
	for _, condition := range conditions {
		if condition.Type != condType {
			result = append(result, condition)
		}
	}
	return result
}

// AggregateComponentConditions creates an aggregated Ready condition based on component conditions
func (cu *ConditionUtils) AggregateComponentConditions(conditions []metav1.Condition) metav1.Condition {
	componentsReady := 0
	componentsTotal := 0
	var notReadyComponents []string

	componentConditions := []string{
		ConditionPrometheusReady,
		ConditionGrafanaReady,
		ConditionLokiReady,
		ConditionTempoReady,
	}

	for _, condType := range componentConditions {
		if cond := cu.GetCondition(conditions, condType); cond != nil {
			componentsTotal++
			if cond.Status == metav1.ConditionTrue {
				componentsReady++
			} else {
				notReadyComponents = append(notReadyComponents, condType)
			}
		}
	}

	if componentsTotal == 0 {
		return metav1.Condition{
			Type:               ConditionReady,
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
			Reason:             ReasonInProgress,
			Message:            "No components configured",
		}
	}

	if componentsReady == componentsTotal {
		return metav1.Condition{
			Type:               ConditionReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             ReasonReady,
			Message:            fmt.Sprintf("All %d components are ready", componentsTotal),
		}
	}

	if componentsReady > 0 {
		return metav1.Condition{
			Type:               ConditionReady,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             ReasonInProgress,
			Message:            fmt.Sprintf("%d of %d components ready. Waiting for: %v", componentsReady, componentsTotal, notReadyComponents),
		}
	}

	return metav1.Condition{
		Type:               ConditionReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonFailed,
		Message:            fmt.Sprintf("No components ready. Failed: %v", notReadyComponents),
	}
}

// CalculatePhase determines the phase based on conditions
func (cu *ConditionUtils) CalculatePhase(conditions []metav1.Condition) string {
	// Check for error condition first
	if cu.IsConditionTrue(conditions, ConditionError) {
		return PhaseFailed
	}

	// Check if ready
	if cu.IsConditionTrue(conditions, ConditionReady) {
		// Check if degraded
		if cu.IsConditionTrue(conditions, ConditionDegraded) {
			return PhaseDegraded
		}
		return PhaseReady
	}

	// Check if progressing
	if cu.IsConditionTrue(conditions, ConditionProgressing) {
		progCond := cu.GetCondition(conditions, ConditionProgressing)
		if progCond != nil && progCond.Reason == ReasonComponentUpgrading {
			return PhaseUpgrading
		}
		return PhaseInstalling
	}

	// Default to pending
	return PhasePending
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}
