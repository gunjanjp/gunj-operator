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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// EventReason defines reasons for events
type EventReason string

// Event reasons
const (
	// Platform lifecycle events
	EventReasonPlatformCreated  EventReason = "PlatformCreated"
	EventReasonPlatformDeleted  EventReason = "PlatformDeleted"
	EventReasonPlatformUpdated  EventReason = "PlatformUpdated"
	EventReasonPlatformReady    EventReason = "PlatformReady"
	EventReasonPlatformFailed   EventReason = "PlatformFailed"
	EventReasonPlatformDegraded EventReason = "PlatformDegraded"

	// Component events
	EventReasonComponentDeploying    EventReason = "ComponentDeploying"
	EventReasonComponentReady        EventReason = "ComponentReady"
	EventReasonComponentFailed       EventReason = "ComponentFailed"
	EventReasonComponentUpgrading    EventReason = "ComponentUpgrading"
	EventReasonComponentDeleted      EventReason = "ComponentDeleted"
	EventReasonComponentScaling      EventReason = "ComponentScaling"
	EventReasonComponentConfigUpdate EventReason = "ComponentConfigUpdate"

	// Resource events
	EventReasonResourceCreated   EventReason = "ResourceCreated"
	EventReasonResourceUpdated   EventReason = "ResourceUpdated"
	EventReasonResourceDeleted   EventReason = "ResourceDeleted"
	EventReasonResourceFailed    EventReason = "ResourceFailed"
	EventReasonInsufficientQuota EventReason = "InsufficientQuota"
	EventReasonStorageError      EventReason = "StorageError"

	// Operation events
	EventReasonBackupStarted   EventReason = "BackupStarted"
	EventReasonBackupCompleted EventReason = "BackupCompleted"
	EventReasonBackupFailed    EventReason = "BackupFailed"
	EventReasonRestoreStarted  EventReason = "RestoreStarted"
	EventReasonRestoreComplete EventReason = "RestoreComplete"
	EventReasonRestoreFailed   EventReason = "RestoreFailed"

	// Configuration events
	EventReasonConfigValidationFailed EventReason = "ConfigValidationFailed"
	EventReasonConfigApplied          EventReason = "ConfigApplied"
	EventReasonSecretMissing          EventReason = "SecretMissing"
	EventReasonConfigMapMissing       EventReason = "ConfigMapMissing"

	// Network events
	EventReasonNetworkPolicyApplied EventReason = "NetworkPolicyApplied"
	EventReasonServiceCreated       EventReason = "ServiceCreated"
	EventReasonIngressCreated       EventReason = "IngressCreated"
	EventReasonDNSError             EventReason = "DNSError"
)

// EventType defines the type of event
type EventType string

const (
	EventTypeNormal  EventType = "Normal"
	EventTypeWarning EventType = "Warning"
)

// EnhancedEventRecorder wraps the standard event recorder with additional functionality
type EnhancedEventRecorder struct {
	recorder        record.EventRecorder
	eventHistory    []EventRecord
	maxHistorySize  int
	componentPrefix string
}

// EventRecord represents a recorded event with metadata
type EventRecord struct {
	Timestamp   time.Time
	Type        EventType
	Reason      EventReason
	Object      runtime.Object
	Message     string
	Component   string
	ActionTaken string
	Details     map[string]string
}

// NewEnhancedEventRecorder creates a new enhanced event recorder
func NewEnhancedEventRecorder(recorder record.EventRecorder, componentPrefix string) *EnhancedEventRecorder {
	return &EnhancedEventRecorder{
		recorder:        recorder,
		eventHistory:    make([]EventRecord, 0),
		maxHistorySize:  1000,
		componentPrefix: componentPrefix,
	}
}

// RecordEvent records an event with enhanced metadata
func (er *EnhancedEventRecorder) RecordEvent(object runtime.Object, eventType EventType, reason EventReason, message string, details map[string]string) {
	// Record to standard Kubernetes events
	if eventType == EventTypeWarning {
		er.recorder.Event(object, corev1.EventTypeWarning, string(reason), message)
	} else {
		er.recorder.Event(object, corev1.EventTypeNormal, string(reason), message)
	}

	// Store in history with metadata
	record := EventRecord{
		Timestamp: time.Now(),
		Type:      eventType,
		Reason:    reason,
		Object:    object,
		Message:   message,
		Component: er.componentPrefix,
		Details:   details,
	}

	er.addToHistory(record)
}

// RecordPlatformEvent records platform-level events
func (er *EnhancedEventRecorder) RecordPlatformEvent(object runtime.Object, reason EventReason, message string) {
	eventType := EventTypeNormal
	if er.isErrorReason(reason) {
		eventType = EventTypeWarning
	}

	details := map[string]string{
		"component": "platform",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	er.RecordEvent(object, eventType, reason, message, details)
}

// RecordComponentEvent records component-specific events
func (er *EnhancedEventRecorder) RecordComponentEvent(object runtime.Object, component string, reason EventReason, message string) {
	eventType := EventTypeNormal
	if er.isErrorReason(reason) {
		eventType = EventTypeWarning
	}

	details := map[string]string{
		"component": component,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	er.RecordEvent(object, eventType, reason, fmt.Sprintf("[%s] %s", component, message), details)
}

// RecordOperationEvent records operation events with duration
func (er *EnhancedEventRecorder) RecordOperationEvent(object runtime.Object, operation string, reason EventReason, message string, duration time.Duration) {
	eventType := EventTypeNormal
	if er.isErrorReason(reason) {
		eventType = EventTypeWarning
	}

	details := map[string]string{
		"operation": operation,
		"duration":  FormatDuration(duration),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	fullMessage := fmt.Sprintf("%s (took %s)", message, FormatDuration(duration))
	er.RecordEvent(object, eventType, reason, fullMessage, details)
}

// RecordResourceEvent records resource-related events
func (er *EnhancedEventRecorder) RecordResourceEvent(object runtime.Object, resourceType, resourceName string, reason EventReason, message string) {
	eventType := EventTypeNormal
	if er.isErrorReason(reason) {
		eventType = EventTypeWarning
	}

	details := map[string]string{
		"resourceType": resourceType,
		"resourceName": resourceName,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	fullMessage := fmt.Sprintf("[%s/%s] %s", resourceType, resourceName, message)
	er.RecordEvent(object, eventType, reason, fullMessage, details)
}

// RecordErrorEvent records error events with additional context
func (er *EnhancedEventRecorder) RecordErrorEvent(object runtime.Object, reason EventReason, err error, context string) {
	details := map[string]string{
		"error":     err.Error(),
		"context":   context,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	message := fmt.Sprintf("%s: %v", context, err)
	er.RecordEvent(object, EventTypeWarning, reason, message, details)
}

// RecordProgressEvent records progress updates for long-running operations
func (er *EnhancedEventRecorder) RecordProgressEvent(object runtime.Object, operation string, current, total int, message string) {
	percentage := 0
	if total > 0 {
		percentage = (current * 100) / total
	}

	details := map[string]string{
		"operation":  operation,
		"current":    fmt.Sprintf("%d", current),
		"total":      fmt.Sprintf("%d", total),
		"percentage": fmt.Sprintf("%d%%", percentage),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	fullMessage := fmt.Sprintf("[%s] %s (%d/%d - %d%%)", operation, message, current, total, percentage)
	er.RecordEvent(object, EventTypeNormal, EventReasonPlatformUpdated, fullMessage, details)
}

// GetEventHistory returns recent events
func (er *EnhancedEventRecorder) GetEventHistory() []EventRecord {
	return er.eventHistory
}

// GetEventsByComponent returns events for a specific component
func (er *EnhancedEventRecorder) GetEventsByComponent(component string) []EventRecord {
	var events []EventRecord
	for _, event := range er.eventHistory {
		if event.Details["component"] == component {
			events = append(events, event)
		}
	}
	return events
}

// GetEventsByReason returns events with a specific reason
func (er *EnhancedEventRecorder) GetEventsByReason(reason EventReason) []EventRecord {
	var events []EventRecord
	for _, event := range er.eventHistory {
		if event.Reason == reason {
			events = append(events, event)
		}
	}
	return events
}

// GetRecentErrors returns recent error events
func (er *EnhancedEventRecorder) GetRecentErrors(limit int) []EventRecord {
	var errors []EventRecord
	count := 0
	
	// Iterate from the end for most recent events
	for i := len(er.eventHistory) - 1; i >= 0 && count < limit; i-- {
		if er.eventHistory[i].Type == EventTypeWarning {
			errors = append(errors, er.eventHistory[i])
			count++
		}
	}
	
	return errors
}

// addToHistory adds an event to the history with size management
func (er *EnhancedEventRecorder) addToHistory(record EventRecord) {
	er.eventHistory = append(er.eventHistory, record)
	
	// Trim history if it exceeds max size
	if len(er.eventHistory) > er.maxHistorySize {
		// Remove oldest 10% of events
		trimSize := er.maxHistorySize / 10
		er.eventHistory = er.eventHistory[trimSize:]
	}
}

// isErrorReason determines if an event reason indicates an error
func (er *EnhancedEventRecorder) isErrorReason(reason EventReason) bool {
	errorReasons := []EventReason{
		EventReasonPlatformFailed,
		EventReasonComponentFailed,
		EventReasonResourceFailed,
		EventReasonBackupFailed,
		EventReasonRestoreFailed,
		EventReasonConfigValidationFailed,
		EventReasonSecretMissing,
		EventReasonConfigMapMissing,
		EventReasonInsufficientQuota,
		EventReasonStorageError,
		EventReasonDNSError,
	}

	for _, errReason := range errorReasons {
		if reason == errReason {
			return true
		}
	}
	return false
}

// GenerateEventSummary creates a summary of recent events
func (er *EnhancedEventRecorder) GenerateEventSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"totalEvents":   len(er.eventHistory),
		"recentErrors":  len(er.GetRecentErrors(10)),
		"eventsByType":  make(map[EventType]int),
		"eventsByReason": make(map[EventReason]int),
		"components":    make(map[string]int),
	}

	// Count events by type and reason
	for _, event := range er.eventHistory {
		summary["eventsByType"].(map[EventType]int)[event.Type]++
		summary["eventsByReason"].(map[EventReason]int)[event.Reason]++
		if component, ok := event.Details["component"]; ok {
			summary["components"].(map[string]int)[component]++
		}
	}

	return summary
}
