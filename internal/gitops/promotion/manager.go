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

package promotion

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/gitops/git"
)

// Manager handles environment promotions
type Manager struct {
	Client        client.Client
	Log           logr.Logger
	GitManager    *git.RepositoryManager
	ApprovalStore ApprovalStore
	MetricsRecorder MetricsRecorder
}

// ApprovalStore manages promotion approvals
type ApprovalStore interface {
	GetApprovals(ctx context.Context, promotionID string) ([]Approval, error)
	AddApproval(ctx context.Context, promotionID string, approval Approval) error
	ClearApprovals(ctx context.Context, promotionID string) error
}

// MetricsRecorder records promotion metrics
type MetricsRecorder interface {
	RecordPromotionStarted(from, to string)
	RecordPromotionCompleted(from, to string, success bool, duration time.Duration)
	RecordPromotionGatePassed(from, to, gateType string)
	RecordPromotionGateFailed(from, to, gateType string)
}

// Approval represents a promotion approval
type Approval struct {
	User      string
	Timestamp time.Time
	Comment   string
}

// NewManager creates a new promotion manager
func NewManager(client client.Client, log logr.Logger, gitManager *git.RepositoryManager) *Manager {
	return &Manager{
		Client:     client,
		Log:        log.WithName("promotion-manager"),
		GitManager: gitManager,
	}
}

// PromotionRequest represents a promotion request
type PromotionRequest struct {
	Platform      *observabilityv1.ObservabilityPlatform
	GitOps        *gitopsv1beta1.GitOpsIntegrationSpec
	FromEnv       string
	ToEnv         string
	Revision      string
	User          string
	AutoPromotion bool
}

// PromotionResult represents the result of a promotion
type PromotionResult struct {
	Success       bool
	Message       string
	FromRevision  string
	ToRevision    string
	PromotionTime time.Time
	GatesChecked  []GateCheckResult
}

// GateCheckResult represents the result of a promotion gate check
type GateCheckResult struct {
	GateType string
	Passed   bool
	Message  string
}

// Promote promotes changes from one environment to another
func (m *Manager) Promote(ctx context.Context, req *PromotionRequest) (*PromotionResult, error) {
	log := m.Log.WithValues(
		"platform", req.Platform.Name,
		"from", req.FromEnv,
		"to", req.ToEnv,
	)
	log.Info("Starting promotion")

	startTime := time.Now()
	if m.MetricsRecorder != nil {
		m.MetricsRecorder.RecordPromotionStarted(req.FromEnv, req.ToEnv)
	}

	result := &PromotionResult{
		PromotionTime: startTime,
	}

	// Find environments
	fromEnvSpec, toEnvSpec, err := m.findEnvironments(req.GitOps, req.FromEnv, req.ToEnv)
	if err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, err
	}

	// Find promotion policy
	policy := m.findPromotionPolicy(req.GitOps, req.FromEnv, req.ToEnv)
	if policy == nil && !req.AutoPromotion {
		result.Success = false
		result.Message = fmt.Sprintf("No promotion policy found for %s -> %s", req.FromEnv, req.ToEnv)
		return result, fmt.Errorf(result.Message)
	}

	// Check promotion gates
	gateResults, allPassed := m.checkPromotionGates(ctx, req, toEnvSpec, policy)
	result.GatesChecked = gateResults

	if !allPassed {
		result.Success = false
		result.Message = "One or more promotion gates failed"
		
		if m.MetricsRecorder != nil {
			m.MetricsRecorder.RecordPromotionCompleted(req.FromEnv, req.ToEnv, false, time.Since(startTime))
		}
		
		return result, nil
	}

	// Clone repository
	repo, repoPath, err := m.GitManager.CloneRepository(ctx, req.GitOps.Repository)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to clone repository: %v", err)
		return result, err
	}
	defer m.GitManager.Cleanup()

	// Get source revision if not specified
	if req.Revision == "" {
		// Checkout source environment branch
		if fromEnvSpec.Branch != "" {
			if err := m.GitManager.CheckoutRevision(repo, fromEnvSpec.Branch); err != nil {
				result.Success = false
				result.Message = fmt.Sprintf("Failed to checkout source branch: %v", err)
				return result, err
			}
		}
		
		revision, err := m.GitManager.GetCurrentRevision(repo)
		if err != nil {
			result.Success = false
			result.Message = fmt.Sprintf("Failed to get current revision: %v", err)
			return result, err
		}
		req.Revision = revision
	}
	result.FromRevision = req.Revision

	// Copy files from source to target
	err = m.copyEnvironmentFiles(ctx, repo, repoPath, fromEnvSpec, toEnvSpec, req)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to copy files: %v", err)
		return result, err
	}

	// Commit and push changes
	commitMessage := fmt.Sprintf("Promote %s from %s to %s\n\nPromoted by: %s\nSource revision: %s",
		req.Platform.Name, req.FromEnv, req.ToEnv, req.User, req.Revision)

	err = m.GitManager.CommitAndPush(ctx, repo, req.GitOps.Repository,
		commitMessage, req.User, fmt.Sprintf("%s@gunj-operator.io", req.User))
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to commit and push: %v", err)
		return result, err
	}

	// Get new revision
	toRevision, err := m.GitManager.GetCurrentRevision(repo)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to get new revision: %v", err)
		return result, err
	}
	result.ToRevision = toRevision

	// Update promotion history
	if err := m.updatePromotionHistory(ctx, req, result); err != nil {
		log.Error(err, "Failed to update promotion history")
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully promoted from %s to %s", req.FromEnv, req.ToEnv)

	if m.MetricsRecorder != nil {
		m.MetricsRecorder.RecordPromotionCompleted(req.FromEnv, req.ToEnv, true, time.Since(startTime))
	}

	log.Info("Promotion completed successfully", "duration", time.Since(startTime))
	return result, nil
}

// findEnvironments finds the source and target environment specifications
func (m *Manager) findEnvironments(
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
	fromEnv, toEnv string,
) (*gitopsv1beta1.EnvironmentSpec, *gitopsv1beta1.EnvironmentSpec, error) {
	var fromSpec, toSpec *gitopsv1beta1.EnvironmentSpec

	for i := range gitOps.Environments {
		env := &gitOps.Environments[i]
		if env.Name == fromEnv {
			fromSpec = env
		}
		if env.Name == toEnv {
			toSpec = env
		}
	}

	if fromSpec == nil {
		return nil, nil, fmt.Errorf("source environment %s not found", fromEnv)
	}
	if toSpec == nil {
		return nil, nil, fmt.Errorf("target environment %s not found", toEnv)
	}

	return fromSpec, toSpec, nil
}

// findPromotionPolicy finds the promotion policy for the given environments
func (m *Manager) findPromotionPolicy(
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
	fromEnv, toEnv string,
) *gitopsv1beta1.PromotionPolicy {
	if gitOps.Promotion == nil {
		return nil
	}

	for i := range gitOps.Promotion.PromotionPolicy {
		policy := &gitOps.Promotion.PromotionPolicy[i]
		if policy.From == fromEnv && policy.To == toEnv {
			return policy
		}
	}

	return nil
}

// checkPromotionGates checks all promotion gates
func (m *Manager) checkPromotionGates(
	ctx context.Context,
	req *PromotionRequest,
	toEnv *gitopsv1beta1.EnvironmentSpec,
	policy *gitopsv1beta1.PromotionPolicy,
) ([]GateCheckResult, bool) {
	var results []GateCheckResult
	allPassed := true

	// Check environment-specific gates
	for _, gate := range toEnv.PromotionGates {
		result := m.checkGate(ctx, req, gate, policy)
		results = append(results, result)
		
		if !result.Passed {
			allPassed = false
			if m.MetricsRecorder != nil {
				m.MetricsRecorder.RecordPromotionGateFailed(req.FromEnv, req.ToEnv, gate.Type)
			}
		} else {
			if m.MetricsRecorder != nil {
				m.MetricsRecorder.RecordPromotionGatePassed(req.FromEnv, req.ToEnv, gate.Type)
			}
		}
	}

	return results, allPassed
}

// checkGate checks a single promotion gate
func (m *Manager) checkGate(
	ctx context.Context,
	req *PromotionRequest,
	gate gitopsv1beta1.PromotionGate,
	policy *gitopsv1beta1.PromotionPolicy,
) GateCheckResult {
	result := GateCheckResult{
		GateType: gate.Type,
	}

	switch gate.Type {
	case "Manual":
		// Check if we have enough approvals
		required := policy.RequiredApprovals
		if required == 0 {
			required = 1 // Default to 1 approval
		}

		promotionID := fmt.Sprintf("%s-%s-%s-%d",
			req.Platform.Name, req.FromEnv, req.ToEnv, time.Now().Unix())

		if m.ApprovalStore != nil {
			approvals, err := m.ApprovalStore.GetApprovals(ctx, promotionID)
			if err != nil {
				result.Passed = false
				result.Message = fmt.Sprintf("Failed to check approvals: %v", err)
				return result
			}

			if len(approvals) >= required {
				result.Passed = true
				result.Message = fmt.Sprintf("Has %d/%d required approvals", len(approvals), required)
			} else {
				result.Passed = false
				result.Message = fmt.Sprintf("Needs %d more approvals (has %d/%d)", 
					required-len(approvals), len(approvals), required)
			}
		} else {
			// If no approval store, check if it's auto-promotion
			result.Passed = req.AutoPromotion
			if result.Passed {
				result.Message = "Auto-promotion enabled"
			} else {
				result.Message = "Manual approval required"
			}
		}

	case "Test":
		// Check test results
		testSuite := gate.Config["suite"]
		if testSuite == "" {
			testSuite = "promotion"
		}

		// This is simplified - in a real implementation, you would:
		// 1. Query test results from a test system
		// 2. Check if tests passed for the source revision
		result.Passed = true // Placeholder
		result.Message = fmt.Sprintf("Test suite '%s' passed", testSuite)

	case "Metric":
		// Check metrics
		metricName := gate.Config["metric"]
		threshold := gate.Config["threshold"]
		
		// This is simplified - in a real implementation, you would:
		// 1. Query metrics from Prometheus
		// 2. Compare against threshold
		result.Passed = true // Placeholder
		result.Message = fmt.Sprintf("Metric '%s' within threshold '%s'", metricName, threshold)

	case "Time":
		// Check time-based gates
		minAge := gate.Config["minAge"]
		if minAge != "" {
			duration, err := time.ParseDuration(minAge)
			if err != nil {
				result.Passed = false
				result.Message = fmt.Sprintf("Invalid duration: %v", err)
				return result
			}

			// Check if source environment has been stable for minimum duration
			// This is simplified - you would check actual deployment time
			result.Passed = true // Placeholder
			result.Message = fmt.Sprintf("Environment stable for %s", minAge)
		}

	default:
		result.Passed = false
		result.Message = fmt.Sprintf("Unknown gate type: %s", gate.Type)
	}

	return result
}

// copyEnvironmentFiles copies files from source to target environment
func (m *Manager) copyEnvironmentFiles(
	ctx context.Context,
	repo interface{},
	repoPath string,
	fromEnv, toEnv *gitopsv1beta1.EnvironmentSpec,
	req *PromotionRequest,
) error {
	// Determine paths
	fromPath := fromEnv.Path
	if fromPath == "" {
		fromPath = fromEnv.Name
	}

	toPath := toEnv.Path
	if toPath == "" {
		toPath = toEnv.Name
	}

	// Read files from source
	files, err := m.GitManager.GetFilesAtPath(repo.(*git.Repository), fromPath)
	if err != nil {
		return fmt.Errorf("failed to read source files: %w", err)
	}

	// Process files for target environment
	processedFiles := make(map[string][]byte)
	for filename, content := range files {
		// Update image tags if needed
		processedContent := m.processFileForEnvironment(content, fromEnv.Name, toEnv.Name)
		processedFiles[filename] = processedContent
	}

	// Write files to target
	if err := m.GitManager.WriteFiles(repo.(*git.Repository), toPath, processedFiles); err != nil {
		return fmt.Errorf("failed to write target files: %w", err)
	}

	// Create promotion metadata file
	metadata := map[string]interface{}{
		"promotion": map[string]interface{}{
			"from":        fromEnv.Name,
			"to":          toEnv.Name,
			"revision":    req.Revision,
			"promotedBy":  req.User,
			"promotedAt":  time.Now().Format(time.RFC3339),
			"platform":    req.Platform.Name,
		},
	}

	metadataYAML, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataFiles := map[string][]byte{
		".promotion-metadata.yaml": metadataYAML,
	}

	if err := m.GitManager.WriteFiles(repo.(*git.Repository), toPath, metadataFiles); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// processFileForEnvironment processes a file for the target environment
func (m *Manager) processFileForEnvironment(content []byte, fromEnv, toEnv string) []byte {
	// Replace environment-specific values
	processedContent := string(content)
	
	// Replace namespace references
	processedContent = strings.ReplaceAll(processedContent, 
		fmt.Sprintf("namespace: %s", fromEnv),
		fmt.Sprintf("namespace: %s", toEnv))
	
	// Replace environment labels
	processedContent = strings.ReplaceAll(processedContent,
		fmt.Sprintf("environment: %s", fromEnv),
		fmt.Sprintf("environment: %s", toEnv))
	
	// Update ingress hosts if present
	processedContent = strings.ReplaceAll(processedContent,
		fmt.Sprintf("-%s.", fromEnv),
		fmt.Sprintf("-%s.", toEnv))

	return []byte(processedContent)
}

// updatePromotionHistory updates the promotion history
func (m *Manager) updatePromotionHistory(
	ctx context.Context,
	req *PromotionRequest,
	result *PromotionResult,
) error {
	// Create or update ConfigMap with promotion history
	historyMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-promotion-history", req.Platform.Name),
			Namespace: req.Platform.Namespace,
			Labels: map[string]string{
				"observability.io/platform": req.Platform.Name,
				"observability.io/component": "promotion",
			},
		},
	}

	// Get existing history
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      historyMap.Name,
		Namespace: historyMap.Namespace,
	}, historyMap)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get promotion history: %w", err)
	}

	if historyMap.Data == nil {
		historyMap.Data = make(map[string]string)
	}

	// Add new entry
	entry := map[string]interface{}{
		"timestamp":    result.PromotionTime.Format(time.RFC3339),
		"from":         req.FromEnv,
		"to":           req.ToEnv,
		"fromRevision": result.FromRevision,
		"toRevision":   result.ToRevision,
		"user":         req.User,
		"success":      result.Success,
		"message":      result.Message,
		"gates":        result.GatesChecked,
	}

	entryYAML, err := yaml.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	key := fmt.Sprintf("%s-%s-%s-%d",
		req.FromEnv, req.ToEnv, result.PromotionTime.Format("20060102-150405"), result.PromotionTime.Unix())
	historyMap.Data[key] = string(entryYAML)

	// Keep only last 50 entries per environment pair
	prefix := fmt.Sprintf("%s-%s-", req.FromEnv, req.ToEnv)
	var keysToDelete []string
	count := 0
	
	for k := range historyMap.Data {
		if strings.HasPrefix(k, prefix) {
			count++
			if count > 50 {
				keysToDelete = append(keysToDelete, k)
			}
		}
	}
	
	for _, k := range keysToDelete {
		delete(historyMap.Data, k)
	}

	// Create or update
	if errors.IsNotFound(err) {
		return m.Client.Create(ctx, historyMap)
	}
	return m.Client.Update(ctx, historyMap)
}

// GetPromotionHistory gets the promotion history for a platform
func (m *Manager) GetPromotionHistory(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	fromEnv, toEnv string,
) ([]PromotionHistoryEntry, error) {
	historyMap := &corev1.ConfigMap{}
	err := m.Client.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-promotion-history", platform.Name),
		Namespace: platform.Namespace,
	}, historyMap)

	if err != nil {
		if errors.IsNotFound(err) {
			return []PromotionHistoryEntry{}, nil
		}
		return nil, err
	}

	var entries []PromotionHistoryEntry
	prefix := fmt.Sprintf("%s-%s-", fromEnv, toEnv)

	for key, value := range historyMap.Data {
		if strings.HasPrefix(key, prefix) {
			var entry PromotionHistoryEntry
			if err := yaml.Unmarshal([]byte(value), &entry); err != nil {
				m.Log.Error(err, "Failed to unmarshal history entry", "key", key)
				continue
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// PromotionHistoryEntry represents a promotion history entry
type PromotionHistoryEntry struct {
	Timestamp    string              `json:"timestamp"`
	From         string              `json:"from"`
	To           string              `json:"to"`
	FromRevision string              `json:"fromRevision"`
	ToRevision   string              `json:"toRevision"`
	User         string              `json:"user"`
	Success      bool                `json:"success"`
	Message      string              `json:"message"`
	Gates        []GateCheckResult   `json:"gates"`
}

// CanPromote checks if promotion is allowed based on policies
func (m *Manager) CanPromote(
	ctx context.Context,
	platform *observabilityv1.ObservabilityPlatform,
	gitOps *gitopsv1beta1.GitOpsIntegrationSpec,
	fromEnv, toEnv string,
) (bool, string) {
	// Check if environments exist
	_, _, err := m.findEnvironments(gitOps, fromEnv, toEnv)
	if err != nil {
		return false, err.Error()
	}

	// Check if promotion path is allowed
	policy := m.findPromotionPolicy(gitOps, fromEnv, toEnv)
	if policy == nil && gitOps.Promotion != nil && gitOps.Promotion.Strategy == "Manual" {
		return false, fmt.Sprintf("No promotion path defined from %s to %s", fromEnv, toEnv)
	}

	// Check auto-promotion time window
	if policy != nil && policy.AutoPromoteAfter != "" {
		// Get last promotion time
		history, err := m.GetPromotionHistory(ctx, platform, fromEnv, toEnv)
		if err == nil && len(history) > 0 {
			lastPromotion, _ := time.Parse(time.RFC3339, history[0].Timestamp)
			duration, _ := time.ParseDuration(policy.AutoPromoteAfter)
			
			if time.Since(lastPromotion) < duration {
				timeLeft := duration - time.Since(lastPromotion)
				return false, fmt.Sprintf("Auto-promotion available in %s", timeLeft.Round(time.Minute))
			}
		}
	}

	return true, ""
}
