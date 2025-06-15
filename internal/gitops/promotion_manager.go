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
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// PromotionManager manages environment promotions for GitOps deployments
type PromotionManager struct {
	client client.Client
	log    logr.Logger
}

// NewPromotionManager creates a new promotion manager
func NewPromotionManager(client client.Client, log logr.Logger) *PromotionManager {
	return &PromotionManager{
		client: client,
		log:    log.WithName("promotion-manager"),
	}
}

// PromotionRequest represents a promotion request
type PromotionRequest struct {
	FromEnvironment string
	ToEnvironment   string
	Revision        string
	RequestedBy     string
	ApprovedBy      string
	Status          string
	CreatedAt       time.Time
	CompletedAt     *time.Time
	Reason          string
}

// ShouldPromote checks if an environment should be promoted
func (m *PromotionManager) ShouldPromote(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment) bool {
	log := m.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name), "environment", env.Name)

	// Check if promotion policy exists
	if env.PromotionPolicy == nil || !env.PromotionPolicy.AutoPromotion {
		return false
	}

	// Check if source environment is specified
	if env.PromotionPolicy.FromEnvironment == "" {
		log.Info("No source environment specified for promotion")
		return false
	}

	// Get source environment status
	sourceEnvStatus := m.getEnvironmentStatus(deployment, env.PromotionPolicy.FromEnvironment)
	if sourceEnvStatus == nil {
		log.Info("Source environment not found", "sourceEnv", env.PromotionPolicy.FromEnvironment)
		return false
	}

	// Get target environment status
	targetEnvStatus := m.getEnvironmentStatus(deployment, env.Name)
	if targetEnvStatus == nil {
		log.Info("Target environment status not found")
		return false
	}

	// Check if source environment is healthy and synced
	if sourceEnvStatus.Phase != "Synced" {
		log.Info("Source environment is not synced", "phase", sourceEnvStatus.Phase)
		return false
	}

	// Check if revisions are different
	if sourceEnvStatus.Revision == targetEnvStatus.Revision {
		log.V(1).Info("Source and target have same revision", "revision", sourceEnvStatus.Revision)
		return false
	}

	// Check promotion conditions
	if !m.checkPromotionConditions(ctx, deployment, env, sourceEnvStatus) {
		log.Info("Promotion conditions not met")
		return false
	}

	// Check if enough time has passed since last sync
	if sourceEnvStatus.LastSyncTime != nil {
		timeSinceSync := time.Since(sourceEnvStatus.LastSyncTime.Time)
		if timeSinceSync < 10*time.Minute {
			log.Info("Not enough time since last sync", "timeSinceSync", timeSinceSync)
			return false
		}
	}

	log.Info("Environment is ready for promotion", 
		"sourceRevision", sourceEnvStatus.Revision,
		"targetRevision", targetEnvStatus.Revision)

	return true
}

// Promote promotes an environment to a new revision
func (m *PromotionManager) Promote(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment) error {
	log := m.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name), "environment", env.Name)

	// Get source environment status
	sourceEnvStatus := m.getEnvironmentStatus(deployment, env.PromotionPolicy.FromEnvironment)
	if sourceEnvStatus == nil {
		return fmt.Errorf("source environment %s not found", env.PromotionPolicy.FromEnvironment)
	}

	log.Info("Starting promotion", 
		"fromEnvironment", env.PromotionPolicy.FromEnvironment,
		"toEnvironment", env.Name,
		"revision", sourceEnvStatus.Revision)

	// Create promotion request
	request := &PromotionRequest{
		FromEnvironment: env.PromotionPolicy.FromEnvironment,
		ToEnvironment:   env.Name,
		Revision:        sourceEnvStatus.Revision,
		RequestedBy:     "auto-promotion",
		Status:          "InProgress",
		CreatedAt:       time.Now(),
	}

	// Store promotion request
	if err := m.storePromotionRequest(ctx, deployment, request); err != nil {
		log.Error(err, "Failed to store promotion request")
	}

	// Check if manual approval is required
	if env.PromotionPolicy.ApprovalRequired {
		return m.createApprovalRequest(ctx, deployment, env, request)
	}

	// Perform the promotion
	if err := m.performPromotion(ctx, deployment, env, sourceEnvStatus.Revision); err != nil {
		request.Status = "Failed"
		request.Reason = err.Error()
		request.CompletedAt = &metav1.Time{Time: time.Now()}
		m.updatePromotionRequest(ctx, deployment, request)
		return err
	}

	// Update promotion request
	request.Status = "Completed"
	request.CompletedAt = &metav1.Time{Time: time.Now()}
	m.updatePromotionRequest(ctx, deployment, request)

	// Record promotion event
	m.recordPromotionEvent(ctx, deployment, env.Name, sourceEnvStatus.Revision, "Promotion completed successfully")

	return nil
}

// getEnvironmentStatus gets the status of a specific environment
func (m *PromotionManager) getEnvironmentStatus(deployment *observabilityv1beta1.GitOpsDeployment, envName string) *observabilityv1beta1.EnvironmentStatus {
	for _, envStatus := range deployment.Status.Environments {
		if envStatus.Name == envName {
			return &envStatus
		}
	}
	return nil
}

// checkPromotionConditions checks if promotion conditions are met
func (m *PromotionManager) checkPromotionConditions(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment, sourceEnvStatus *observabilityv1beta1.EnvironmentStatus) bool {
	if env.PromotionPolicy == nil || len(env.PromotionPolicy.Conditions) == 0 {
		return true
	}

	for _, condition := range env.PromotionPolicy.Conditions {
		if !m.checkCondition(ctx, deployment, sourceEnvStatus, condition) {
			m.log.Info("Promotion condition not met", 
				"conditionType", condition.Type,
				"requiredStatus", condition.Status)
			return false
		}
	}

	return true
}

// checkCondition checks a single promotion condition
func (m *PromotionManager) checkCondition(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, envStatus *observabilityv1beta1.EnvironmentStatus, condition observabilityv1beta1.PromotionCondition) bool {
	switch condition.Type {
	case "HealthCheck":
		// Check if all resources in the environment are healthy
		healthyCount := 0
		for _, resource := range envStatus.Resources {
			if resource.Health == "Healthy" {
				healthyCount++
			}
		}
		return healthyCount == len(envStatus.Resources)

	case "TestsPassed":
		// Check if tests have passed (would need to integrate with CI/CD)
		return m.checkTestResults(ctx, deployment, envStatus)

	case "MetricsThreshold":
		// Check if metrics meet thresholds
		return m.checkMetricsThresholds(ctx, deployment, envStatus)

	case "TimeSinceDeployment":
		// Check if enough time has passed since deployment
		if envStatus.LastSyncTime != nil {
			timeSince := time.Since(envStatus.LastSyncTime.Time)
			// Parse duration from condition reason (e.g., "24h")
			if duration, err := time.ParseDuration(condition.Reason); err == nil {
				return timeSince >= duration
			}
		}
		return false

	default:
		// Unknown condition type, assume it's met
		return true
	}
}

// performPromotion performs the actual promotion
func (m *PromotionManager) performPromotion(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment, targetRevision string) error {
	switch deployment.Spec.GitOpsEngine {
	case "argocd":
		return m.promoteArgoCD(ctx, deployment, env, targetRevision)
	case "flux":
		return m.promoteFlux(ctx, deployment, env, targetRevision)
	default:
		return fmt.Errorf("unknown GitOps engine: %s", deployment.Spec.GitOpsEngine)
	}
}

// promoteArgoCD promotes an ArgoCD application
func (m *PromotionManager) promoteArgoCD(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment, targetRevision string) error {
	// Update the ArgoCD application for this environment
	if deployment.Spec.ArgoCD == nil {
		return fmt.Errorf("ArgoCD configuration not found")
	}

	appName := fmt.Sprintf("%s-%s", deployment.Spec.ArgoCD.ApplicationName, env.Name)
	
	// Use the ArgoCD manager to update the application
	argoCDManager := NewArgoCDManager(m.client, m.log)
	return argoCDManager.UpdateApplicationSpec(ctx, appName, func(spec map[string]interface{}) error {
		source, found := spec["source"].(map[string]interface{})
		if !found {
			return fmt.Errorf("source not found in application spec")
		}

		// Update target revision
		source["targetRevision"] = targetRevision

		// Update path if environment has specific path
		if env.Path != "" {
			source["path"] = env.Path
		}

		return nil
	})
}

// promoteFlux promotes a Flux Kustomization
func (m *PromotionManager) promoteFlux(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment, targetRevision string) error {
	// Update the Flux Kustomization for this environment
	if deployment.Spec.Flux == nil {
		return fmt.Errorf("Flux configuration not found")
	}

	kustomizationName := fmt.Sprintf("%s-%s", deployment.Spec.Flux.KustomizationName, env.Name)

	// Use the Flux manager to update the kustomization
	fluxManager := NewFluxManager(m.client, m.log)
	return fluxManager.UpdateKustomizationSpec(ctx, kustomizationName, deployment.Namespace, func(spec map[string]interface{}) error {
		// Update source reference to use specific revision
		sourceRef, found := spec["sourceRef"].(map[string]interface{})
		if !found {
			return fmt.Errorf("sourceRef not found in kustomization spec")
		}

		// Create a new GitRepository source with the specific revision
		promotedSourceName := fmt.Sprintf("%s-promoted-%s", deployment.Name, targetRevision[:7])
		if err := m.createPromotedGitSource(ctx, deployment, promotedSourceName, targetRevision); err != nil {
			return err
		}

		sourceRef["name"] = promotedSourceName
		return nil
	})
}

// createPromotedGitSource creates a GitRepository source for a specific revision
func (m *PromotionManager) createPromotedGitSource(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, sourceName, revision string) error {
	source := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/promotion":         "true",
			},
		},
		Data: map[string]string{
			"apiVersion": "source.toolkit.fluxcd.io/v1beta2",
			"kind":       "GitRepository",
			"metadata": fmt.Sprintf(`{"name": "%s", "namespace": "%s"}`, sourceName, deployment.Namespace),
			"spec": fmt.Sprintf(`{
				"interval": "%s",
				"url": "%s",
				"ref": {"commit": "%s"}
			}`, deployment.Spec.Repository.PollInterval, deployment.Spec.Repository.URL, revision),
		},
	}

	// This is a simplified version - in reality, we'd create the actual GitRepository resource
	return m.client.Create(ctx, source)
}

// createApprovalRequest creates a manual approval request
func (m *PromotionManager) createApprovalRequest(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, env observabilityv1beta1.Environment, request *PromotionRequest) error {
	// Create a ConfigMap to store the approval request
	approval := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-promotion-approval-%d", deployment.Name, time.Now().Unix()),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/approval-type":     "promotion",
				"observability.io/environment":       env.Name,
			},
		},
		Data: map[string]string{
			"fromEnvironment": request.FromEnvironment,
			"toEnvironment":   request.ToEnvironment,
			"revision":        request.Revision,
			"requestedBy":     request.RequestedBy,
			"requestedAt":     request.CreatedAt.Format(time.RFC3339),
			"status":          "PendingApproval",
		},
	}

	if err := m.client.Create(ctx, approval); err != nil {
		return fmt.Errorf("failed to create approval request: %w", err)
	}

	// Update promotion request status
	request.Status = "PendingApproval"
	m.updatePromotionRequest(ctx, deployment, request)

	// Record event
	m.recordPromotionEvent(ctx, deployment, env.Name, request.Revision, "Promotion pending approval")

	return nil
}

// ApprovePromotion approves a pending promotion
func (m *PromotionManager) ApprovePromotion(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, approvalName, approvedBy string) error {
	// Get the approval request
	approval := &corev1.ConfigMap{}
	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      approvalName,
		Namespace: deployment.Namespace,
	}, approval); err != nil {
		return fmt.Errorf("failed to get approval request: %w", err)
	}

	// Check if already processed
	if approval.Data["status"] != "PendingApproval" {
		return fmt.Errorf("approval request is not pending")
	}

	// Update approval status
	approval.Data["status"] = "Approved"
	approval.Data["approvedBy"] = approvedBy
	approval.Data["approvedAt"] = time.Now().Format(time.RFC3339)

	if err := m.client.Update(ctx, approval); err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	// Find the environment
	envName := approval.Data["toEnvironment"]
	var targetEnv *observabilityv1beta1.Environment
	for _, env := range deployment.Spec.Environments {
		if env.Name == envName {
			targetEnv = &env
			break
		}
	}

	if targetEnv == nil {
		return fmt.Errorf("environment %s not found", envName)
	}

	// Perform the promotion
	return m.performPromotion(ctx, deployment, *targetEnv, approval.Data["revision"])
}

// RejectPromotion rejects a pending promotion
func (m *PromotionManager) RejectPromotion(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, approvalName, rejectedBy, reason string) error {
	// Get the approval request
	approval := &corev1.ConfigMap{}
	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      approvalName,
		Namespace: deployment.Namespace,
	}, approval); err != nil {
		return fmt.Errorf("failed to get approval request: %w", err)
	}

	// Check if already processed
	if approval.Data["status"] != "PendingApproval" {
		return fmt.Errorf("approval request is not pending")
	}

	// Update approval status
	approval.Data["status"] = "Rejected"
	approval.Data["rejectedBy"] = rejectedBy
	approval.Data["rejectedAt"] = time.Now().Format(time.RFC3339)
	approval.Data["reason"] = reason

	if err := m.client.Update(ctx, approval); err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	// Record event
	m.recordPromotionEvent(ctx, deployment, approval.Data["toEnvironment"], approval.Data["revision"], 
		fmt.Sprintf("Promotion rejected: %s", reason))

	return nil
}

// GetPromotionHistory gets the promotion history for a deployment
func (m *PromotionManager) GetPromotionHistory(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) ([]PromotionRequest, error) {
	// Get promotion history from ConfigMap
	cm := &corev1.ConfigMap{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-promotion-history", deployment.Name),
		Namespace: deployment.Namespace,
	}, cm)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		return []PromotionRequest{}, nil
	}

	var history []PromotionRequest
	for _, data := range cm.Data {
		var request PromotionRequest
		if err := json.Unmarshal([]byte(data), &request); err != nil {
			m.log.Error(err, "Failed to unmarshal promotion request")
			continue
		}
		history = append(history, request)
	}

	return history, nil
}

// storePromotionRequest stores a promotion request in history
func (m *PromotionManager) storePromotionRequest(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, request *PromotionRequest) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-promotion-history", deployment.Name),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/type":              "promotion-history",
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

	// Add new promotion request
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s-%d", request.ToEnvironment, request.CreatedAt.Unix())
	cm.Data[key] = string(requestJSON)

	// Create or update ConfigMap
	if existing.Name == "" {
		return m.client.Create(ctx, cm)
	}
	return m.client.Update(ctx, cm)
}

// updatePromotionRequest updates a promotion request in history
func (m *PromotionManager) updatePromotionRequest(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, request *PromotionRequest) error {
	return m.storePromotionRequest(ctx, deployment, request)
}

// recordPromotionEvent records a promotion event
func (m *PromotionManager) recordPromotionEvent(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, environment, revision, message string) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-promotion-%d", deployment.Name, time.Now().Unix()),
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
		Reason:  "Promotion",
		Message: fmt.Sprintf("Environment %s: %s (revision: %s)", environment, message, revision),
		Source: corev1.EventSource{
			Component: "promotion-manager",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	if err := m.client.Create(ctx, event); err != nil {
		m.log.Error(err, "Failed to create promotion event")
	}
}

// checkTestResults checks if tests have passed for an environment
func (m *PromotionManager) checkTestResults(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, envStatus *observabilityv1beta1.EnvironmentStatus) bool {
	// This would integrate with CI/CD systems to check test results
	// For now, we'll check for a test results annotation
	testResults := &corev1.ConfigMap{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s-test-results", deployment.Name, envStatus.Name),
		Namespace: deployment.Namespace,
	}, testResults)

	if err != nil {
		return false
	}

	// Check if all tests passed
	return testResults.Data["status"] == "passed"
}

// checkMetricsThresholds checks if metrics meet promotion thresholds
func (m *PromotionManager) checkMetricsThresholds(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, envStatus *observabilityv1beta1.EnvironmentStatus) bool {
	// This would query Prometheus or other metrics systems
	// For now, we'll check for a metrics status ConfigMap
	metricsStatus := &corev1.ConfigMap{}
	err := m.client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s-metrics-status", deployment.Name, envStatus.Name),
		Namespace: deployment.Namespace,
	}, metricsStatus)

	if err != nil {
		return false
	}

	// Check if metrics are within thresholds
	errorRate := metricsStatus.Data["errorRate"]
	if errorRate != "" {
		// Parse and check error rate
		if strings.TrimSuffix(errorRate, "%") > "1" {
			return false
		}
	}

	return true
}
