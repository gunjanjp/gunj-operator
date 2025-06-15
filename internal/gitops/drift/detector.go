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

package drift

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pmezard/go-difflib/difflib"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
)

// DetectionManager handles drift detection and remediation
type DetectionManager struct {
	Client         client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	Notifier       Notifier
	MetricsRecorder MetricsRecorder
}

// Notifier sends drift notifications
type Notifier interface {
	NotifyDrift(ctx context.Context, drift *DriftReport) error
}

// MetricsRecorder records drift metrics
type MetricsRecorder interface {
	RecordDriftDetected(platform, resource string, driftType string)
	RecordDriftRemediated(platform, resource string, success bool)
	RecordDriftCheckDuration(platform string, duration time.Duration)
}

// NewDetectionManager creates a new drift detection manager
func NewDetectionManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *DetectionManager {
	return &DetectionManager{
		Client: client,
		Scheme: scheme,
		Log:    log.WithName("drift-detection"),
	}
}

// DriftReport represents a drift detection report
type DriftReport struct {
	Platform       string
	Namespace      string
	CheckTime      time.Time
	DriftDetected  bool
	DriftedResources []gitopsv1beta1.DriftedResource
	Remediation    *RemediationReport
}

// RemediationReport represents remediation results
type RemediationReport struct {
	Attempted     bool
	Success       bool
	FailedResources []string
	Error         error
}

// CheckDrift checks for drift in the platform resources
func (m *DetectionManager) CheckDrift(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
	desiredState map[string]*unstructured.Unstructured,
) (*DriftReport, error) {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.Info("Starting drift detection")

	startTime := time.Now()
	defer func() {
		if m.MetricsRecorder != nil {
			m.MetricsRecorder.RecordDriftCheckDuration(platform.Name, time.Since(startTime))
		}
	}()

	report := &DriftReport{
		Platform:  platform.Name,
		Namespace: platform.Namespace,
		CheckTime: time.Now(),
	}

	// Get current state
	currentState, err := m.getCurrentState(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state: %w", err)
	}

	// Compare states
	drifts := m.compareStates(desiredState, currentState)
	
	if len(drifts) > 0 {
		report.DriftDetected = true
		report.DriftedResources = drifts
		
		// Record metrics
		if m.MetricsRecorder != nil {
			for _, drift := range drifts {
				m.MetricsRecorder.RecordDriftDetected(platform.Name, 
					fmt.Sprintf("%s/%s", drift.Kind, drift.Name), drift.DriftType)
			}
		}

		// Send notifications if configured
		if gitOps.DriftDetection != nil && gitOps.DriftDetection.NotificationPolicy != nil && m.Notifier != nil {
			if err := m.Notifier.NotifyDrift(ctx, report); err != nil {
				log.Error(err, "Failed to send drift notification")
			}
		}

		// Auto-remediate if configured
		if gitOps.DriftDetection != nil && gitOps.DriftDetection.AutoRemediate {
			log.Info("Auto-remediating drift", "resources", len(drifts))
			remediation := m.remediate(ctx, platform, desiredState, drifts)
			report.Remediation = remediation
		}
	}

	log.Info("Drift detection completed", "driftDetected", report.DriftDetected, "driftedResources", len(report.DriftedResources))
	return report, nil
}

// getCurrentState gets the current state of all managed resources
func (m *DetectionManager) getCurrentState(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
) (map[string]*unstructured.Unstructured, error) {
	currentState := make(map[string]*unstructured.Unstructured)

	// Get all resources with the platform label
	labelSelector := client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}

	// List of resource types to check
	resourceTypes := []schema.GroupVersionKind{
		{Group: "apps", Version: "v1", Kind: "Deployment"},
		{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		{Group: "apps", Version: "v1", Kind: "DaemonSet"},
		{Group: "", Version: "v1", Kind: "Service"},
		{Group: "", Version: "v1", Kind: "ConfigMap"},
		{Group: "", Version: "v1", Kind: "Secret"},
		{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
		{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
		{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"},
		{Group: "monitoring.coreos.com", Version: "v1", Kind: "PrometheusRule"},
	}

	for _, gvk := range resourceTypes {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)

		if err := m.Client.List(ctx, list, labelSelector, client.InNamespace(platform.Namespace)); err != nil {
			// Skip if CRD doesn't exist
			if errors.IsNotFound(err) || strings.Contains(err.Error(), "no matches for kind") {
				continue
			}
			return nil, fmt.Errorf("failed to list %s: %w", gvk.Kind, err)
		}

		for _, item := range list.Items {
			key := m.getResourceKey(&item)
			currentState[key] = item.DeepCopy()
		}
	}

	return currentState, nil
}

// compareStates compares desired and current states to detect drift
func (m *DetectionManager) compareStates(
	desiredState map[string]*unstructured.Unstructured,
	currentState map[string]*unstructured.Unstructured,
) []gitopsv1beta1.DriftedResource {
	var drifts []gitopsv1beta1.DriftedResource

	// Check for modified or deleted resources
	for key, desired := range desiredState {
		current, exists := currentState[key]
		
		if !exists {
			// Resource was deleted
			drifts = append(drifts, gitopsv1beta1.DriftedResource{
				APIVersion: desired.GetAPIVersion(),
				Kind:       desired.GetKind(),
				Name:       desired.GetName(),
				Namespace:  desired.GetNamespace(),
				DriftType:  "Deleted",
				Details:    "Resource exists in desired state but not in cluster",
			})
			continue
		}

		// Check for modifications
		if drift := m.detectModifications(desired, current); drift != nil {
			drifts = append(drifts, *drift)
		}
	}

	// Check for added resources (exist in cluster but not in desired state)
	for key, current := range currentState {
		if _, exists := desiredState[key]; !exists {
			drifts = append(drifts, gitopsv1beta1.DriftedResource{
				APIVersion: current.GetAPIVersion(),
				Kind:       current.GetKind(),
				Name:       current.GetName(),
				Namespace:  current.GetNamespace(),
				DriftType:  "Added",
				Details:    "Resource exists in cluster but not in desired state",
			})
		}
	}

	return drifts
}

// detectModifications checks if a resource has been modified
func (m *DetectionManager) detectModifications(
	desired *unstructured.Unstructured,
	current *unstructured.Unstructured,
) *gitopsv1beta1.DriftedResource {
	// Normalize resources for comparison
	desiredNorm := m.normalizeForComparison(desired)
	currentNorm := m.normalizeForComparison(current)

	// Compare specs
	desiredSpec, _, _ := unstructured.NestedMap(desiredNorm.Object, "spec")
	currentSpec, _, _ := unstructured.NestedMap(currentNorm.Object, "spec")

	if !reflect.DeepEqual(desiredSpec, currentSpec) {
		// Generate diff
		desiredYAML, _ := yaml.Marshal(desiredSpec)
		currentYAML, _ := yaml.Marshal(currentSpec)
		
		diff := difflib.UnifiedDiff{
			A:        strings.Split(string(desiredYAML), "\n"),
			B:        strings.Split(string(currentYAML), "\n"),
			FromFile: "desired",
			ToFile:   "current",
			Context:  3,
		}
		
		diffStr, _ := difflib.GetUnifiedDiffString(diff)

		return &gitopsv1beta1.DriftedResource{
			APIVersion: current.GetAPIVersion(),
			Kind:       current.GetKind(),
			Name:       current.GetName(),
			Namespace:  current.GetNamespace(),
			DriftType:  "Modified",
			Details:    fmt.Sprintf("Spec differs from desired state:\n%s", diffStr),
		}
	}

	// Compare important metadata
	if m.hasMetadataDrift(desired, current) {
		return &gitopsv1beta1.DriftedResource{
			APIVersion: current.GetAPIVersion(),
			Kind:       current.GetKind(),
			Name:       current.GetName(),
			Namespace:  current.GetNamespace(),
			DriftType:  "Modified",
			Details:    "Metadata (labels/annotations) differs from desired state",
		}
	}

	return nil
}

// normalizeForComparison normalizes a resource for drift comparison
func (m *DetectionManager) normalizeForComparison(obj *unstructured.Unstructured) *unstructured.Unstructured {
	normalized := obj.DeepCopy()

	// Remove fields that shouldn't be compared
	unstructured.RemoveNestedField(normalized.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(normalized.Object, "metadata", "uid")
	unstructured.RemoveNestedField(normalized.Object, "metadata", "generation")
	unstructured.RemoveNestedField(normalized.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(normalized.Object, "metadata", "deletionTimestamp")
	unstructured.RemoveNestedField(normalized.Object, "metadata", "selfLink")
	unstructured.RemoveNestedField(normalized.Object, "metadata", "managedFields")
	unstructured.RemoveNestedField(normalized.Object, "status")

	// Normalize annotations
	annotations := normalized.GetAnnotations()
	if annotations != nil {
		// Remove kubectl annotations
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
		delete(annotations, "deployment.kubernetes.io/revision")
		normalized.SetAnnotations(annotations)
	}

	return normalized
}

// hasMetadataDrift checks if important metadata has drifted
func (m *DetectionManager) hasMetadataDrift(desired, current *unstructured.Unstructured) bool {
	// Compare labels
	desiredLabels := desired.GetLabels()
	currentLabels := current.GetLabels()
	
	// Remove dynamic labels
	delete(currentLabels, "pod-template-hash")
	
	if !reflect.DeepEqual(desiredLabels, currentLabels) {
		return true
	}

	// Compare important annotations
	desiredAnnotations := desired.GetAnnotations()
	currentAnnotations := current.GetAnnotations()
	
	// Only compare non-system annotations
	importantAnnotations := []string{
		"prometheus.io/scrape",
		"prometheus.io/port",
		"prometheus.io/path",
	}
	
	for _, key := range importantAnnotations {
		if desiredAnnotations[key] != currentAnnotations[key] {
			return true
		}
	}

	return false
}

// remediate attempts to fix drifted resources
func (m *DetectionManager) remediate(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	desiredState map[string]*unstructured.Unstructured,
	drifts []gitopsv1beta1.DriftedResource,
) *RemediationReport {
	report := &RemediationReport{
		Attempted: true,
		Success:   true,
	}

	for _, drift := range drifts {
		key := fmt.Sprintf("%s/%s/%s/%s", drift.APIVersion, drift.Kind, drift.Namespace, drift.Name)
		
		switch drift.DriftType {
		case "Deleted":
			// Recreate deleted resource
			if desired, ok := desiredState[key]; ok {
				if err := m.Client.Create(ctx, desired); err != nil {
					report.Success = false
					report.FailedResources = append(report.FailedResources, key)
					m.Log.Error(err, "Failed to recreate resource", "resource", key)
					
					if m.MetricsRecorder != nil {
						m.MetricsRecorder.RecordDriftRemediated(platform.Name, key, false)
					}
				} else {
					if m.MetricsRecorder != nil {
						m.MetricsRecorder.RecordDriftRemediated(platform.Name, key, true)
					}
				}
			}

		case "Modified":
			// Update modified resource
			if desired, ok := desiredState[key]; ok {
				// Get current resource to preserve resourceVersion
				current := &unstructured.Unstructured{}
				current.SetGroupVersionKind(desired.GroupVersionKind())
				
				err := m.Client.Get(ctx, types.NamespacedName{
					Name:      desired.GetName(),
					Namespace: desired.GetNamespace(),
				}, current)
				
				if err == nil {
					// Preserve resourceVersion for update
					desired.SetResourceVersion(current.GetResourceVersion())
					
					if err := m.Client.Update(ctx, desired); err != nil {
						report.Success = false
						report.FailedResources = append(report.FailedResources, key)
						m.Log.Error(err, "Failed to update resource", "resource", key)
						
						if m.MetricsRecorder != nil {
							m.MetricsRecorder.RecordDriftRemediated(platform.Name, key, false)
						}
					} else {
						if m.MetricsRecorder != nil {
							m.MetricsRecorder.RecordDriftRemediated(platform.Name, key, true)
						}
					}
				}
			}

		case "Added":
			// Delete added resource
			obj := &unstructured.Unstructured{}
			obj.SetAPIVersion(drift.APIVersion)
			obj.SetKind(drift.Kind)
			obj.SetName(drift.Name)
			obj.SetNamespace(drift.Namespace)
			
			if err := m.Client.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
				report.Success = false
				report.FailedResources = append(report.FailedResources, key)
				m.Log.Error(err, "Failed to delete resource", "resource", key)
				
				if m.MetricsRecorder != nil {
					m.MetricsRecorder.RecordDriftRemediated(platform.Name, key, false)
				}
			} else {
				if m.MetricsRecorder != nil {
					m.MetricsRecorder.RecordDriftRemediated(platform.Name, key, true)
				}
			}
		}
	}

	return report
}

// getResourceKey generates a unique key for a resource
func (m *DetectionManager) getResourceKey(obj *unstructured.Unstructured) string {
	return fmt.Sprintf("%s/%s/%s/%s",
		obj.GetAPIVersion(),
		obj.GetKind(),
		obj.GetNamespace(),
		obj.GetName(),
	)
}

// CalculateChecksum calculates a checksum for a resource
func (m *DetectionManager) CalculateChecksum(obj *unstructured.Unstructured) string {
	normalized := m.normalizeForComparison(obj)
	
	// Sort keys for consistent hashing
	data, _ := yaml.Marshal(normalized.Object)
	
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// StoreDriftHistory stores drift detection history
func (m *DetectionManager) StoreDriftHistory(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	report *DriftReport,
) error {
	// Create or update ConfigMap with drift history
	historyMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-drift-history", platform.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"observability.io/platform": platform.Name,
				"observability.io/component": "drift-detection",
			},
		},
	}

	// Get existing history
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      historyMap.Name,
		Namespace: historyMap.Namespace,
	}, historyMap)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get drift history: %w", err)
	}

	if historyMap.Data == nil {
		historyMap.Data = make(map[string]string)
	}

	// Add new entry
	timestamp := report.CheckTime.Format(time.RFC3339)
	reportData, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	historyMap.Data[timestamp] = string(reportData)

	// Keep only last 100 entries
	if len(historyMap.Data) > 100 {
		// Sort keys and remove oldest
		keys := make([]string, 0, len(historyMap.Data))
		for k := range historyMap.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		for i := 0; i < len(keys)-100; i++ {
			delete(historyMap.Data, keys[i])
		}
	}

	// Create or update
	if errors.IsNotFound(err) {
		return m.Client.Create(ctx, historyMap)
	}
	return m.Client.Update(ctx, historyMap)
}
