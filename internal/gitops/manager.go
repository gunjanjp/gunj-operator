// Package gitops provides GitOps integration for the Gunj Operator
package gitops

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/gitops/argocd"
	"github.com/gunjanjp/gunj-operator/internal/gitops/drift"
	"github.com/gunjanjp/gunj-operator/internal/gitops/flux"
	"github.com/gunjanjp/gunj-operator/internal/gitops/rollback"
	"github.com/gunjanjp/gunj-operator/internal/gitops/sync"
)

// Manager coordinates GitOps operations for ObservabilityPlatforms
type Manager struct {
	Client         client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	ArgoCDManager  *argocd.Manager
	FluxManager    *flux.Manager
	SyncManager    *sync.Manager
	RollbackMgr    *rollback.Manager
	DriftDetector  *drift.Detector
}

// NewManager creates a new GitOps manager
func NewManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *Manager {
	return &Manager{
		Client:        client,
		Scheme:        scheme,
		Log:           log.WithName("gitops-manager"),
		ArgoCDManager: argocd.NewManager(client, scheme, log),
		FluxManager:   flux.NewManager(client, scheme, log),
		SyncManager:   sync.NewManager(client, scheme, log),
		RollbackMgr:   rollback.NewManager(client, scheme, log),
		DriftDetector: drift.NewDetector(client, scheme, log),
	}
}

// ReconcileGitOps ensures GitOps integration for a platform
func (m *Manager) ReconcileGitOps(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name, "namespace", platform.Namespace)
	log.V(1).Info("Reconciling GitOps integration")

	// Check if GitOps is enabled
	if platform.Spec.GitOps == nil || !platform.Spec.GitOps.Enabled {
		log.V(1).Info("GitOps not enabled for platform")
		return m.cleanupGitOpsResources(ctx, platform)
	}

	// Validate GitOps configuration
	if err := m.validateGitOpsConfig(platform.Spec.GitOps); err != nil {
		return fmt.Errorf("invalid GitOps configuration: %w", err)
	}

	// Reconcile based on GitOps provider
	switch platform.Spec.GitOps.Provider {
	case observabilityv1.GitOpsProviderArgoCD:
		return m.reconcileArgoCD(ctx, platform)
	case observabilityv1.GitOpsProviderFlux:
		return m.reconcileFlux(ctx, platform)
	default:
		return fmt.Errorf("unsupported GitOps provider: %s", platform.Spec.GitOps.Provider)
	}
}

// reconcileArgoCD sets up ArgoCD integration
func (m *Manager) reconcileArgoCD(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name, "provider", "argocd")
	log.V(1).Info("Reconciling ArgoCD integration")

	// Create ArgoCD Application
	app, err := m.ArgoCDManager.CreateApplication(ctx, platform)
	if err != nil {
		return fmt.Errorf("creating ArgoCD application: %w", err)
	}

	// Set up sync policies
	if err := m.ArgoCDManager.ConfigureSyncPolicy(ctx, app, platform.Spec.GitOps); err != nil {
		return fmt.Errorf("configuring sync policy: %w", err)
	}

	// Set up rollback hooks
	if platform.Spec.GitOps.Rollback != nil && platform.Spec.GitOps.Rollback.Enabled {
		if err := m.ArgoCDManager.ConfigureRollback(ctx, app, platform.Spec.GitOps.Rollback); err != nil {
			return fmt.Errorf("configuring rollback: %w", err)
		}
	}

	// Update platform status
	platform.Status.GitOps = &observabilityv1.GitOpsStatus{
		Provider:     observabilityv1.GitOpsProviderArgoCD,
		SyncStatus:   "Synced",
		LastSyncTime: &metav1.Time{Time: time.Now()},
		AppName:      app.Name,
		AppNamespace: app.Namespace,
	}

	return nil
}

// reconcileFlux sets up Flux integration
func (m *Manager) reconcileFlux(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name, "provider", "flux")
	log.V(1).Info("Reconciling Flux integration")

	// Create Flux GitRepository
	repo, err := m.FluxManager.CreateGitRepository(ctx, platform)
	if err != nil {
		return fmt.Errorf("creating Git repository: %w", err)
	}

	// Create Flux Kustomization
	kustomization, err := m.FluxManager.CreateKustomization(ctx, platform, repo)
	if err != nil {
		return fmt.Errorf("creating Kustomization: %w", err)
	}

	// Configure health checks
	if err := m.FluxManager.ConfigureHealthChecks(ctx, kustomization, platform); err != nil {
		return fmt.Errorf("configuring health checks: %w", err)
	}

	// Set up automated rollback
	if platform.Spec.GitOps.Rollback != nil && platform.Spec.GitOps.Rollback.Enabled {
		if err := m.FluxManager.ConfigureRollback(ctx, kustomization, platform.Spec.GitOps.Rollback); err != nil {
			return fmt.Errorf("configuring rollback: %w", err)
		}
	}

	// Update platform status
	platform.Status.GitOps = &observabilityv1.GitOpsStatus{
		Provider:          observabilityv1.GitOpsProviderFlux,
		SyncStatus:        "Synced",
		LastSyncTime:      &metav1.Time{Time: time.Now()},
		KustomizationName: kustomization.Name,
		GitRepositoryName: repo.Name,
	}

	return nil
}

// HandleGitWebhook processes Git webhook events
func (m *Manager) HandleGitWebhook(ctx context.Context, event *WebhookEvent) error {
	log := m.Log.WithValues("event", event.Type, "repo", event.Repository)
	log.V(1).Info("Handling Git webhook event")

	// Find platforms using this repository
	platforms, err := m.findPlatformsByRepo(ctx, event.Repository)
	if err != nil {
		return fmt.Errorf("finding platforms by repo: %w", err)
	}

	for _, platform := range platforms {
		// Process event based on type
		switch event.Type {
		case WebhookEventPush:
			if err := m.handlePushEvent(ctx, platform, event); err != nil {
				log.Error(err, "Failed to handle push event", "platform", platform.Name)
			}
		case WebhookEventPullRequest:
			if err := m.handlePullRequestEvent(ctx, platform, event); err != nil {
				log.Error(err, "Failed to handle PR event", "platform", platform.Name)
			}
		case WebhookEventTag:
			if err := m.handleTagEvent(ctx, platform, event); err != nil {
				log.Error(err, "Failed to handle tag event", "platform", platform.Name)
			}
		}
	}

	return nil
}

// handlePushEvent handles git push events
func (m *Manager) handlePushEvent(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, event *WebhookEvent) error {
	log := m.Log.WithValues("platform", platform.Name, "commit", event.Commit)
	log.V(1).Info("Processing push event")

	// Check if this is the tracked branch
	if platform.Spec.GitOps.Repository.Branch != event.Branch {
		log.V(2).Info("Ignoring push to non-tracked branch", "branch", event.Branch)
		return nil
	}

	// Trigger sync
	if err := m.SyncManager.TriggerSync(ctx, platform); err != nil {
		return fmt.Errorf("triggering sync: %w", err)
	}

	// Check for drift if enabled
	if platform.Spec.GitOps.DriftDetection != nil && platform.Spec.GitOps.DriftDetection.Enabled {
		if err := m.DriftDetector.CheckDrift(ctx, platform); err != nil {
			log.Error(err, "Failed to check drift")
		}
	}

	return nil
}

// handlePullRequestEvent handles PR events
func (m *Manager) handlePullRequestEvent(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, event *WebhookEvent) error {
	log := m.Log.WithValues("platform", platform.Name, "pr", event.PullRequest.Number)
	log.V(1).Info("Processing pull request event")

	// Create preview environment if enabled
	if platform.Spec.GitOps.PreviewEnvironments != nil && platform.Spec.GitOps.PreviewEnvironments.Enabled {
		switch event.PullRequest.Action {
		case "opened", "reopened", "synchronize":
			return m.createPreviewEnvironment(ctx, platform, event.PullRequest)
		case "closed":
			return m.deletePreviewEnvironment(ctx, platform, event.PullRequest)
		}
	}

	return nil
}

// handleTagEvent handles git tag events
func (m *Manager) handleTagEvent(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, event *WebhookEvent) error {
	log := m.Log.WithValues("platform", platform.Name, "tag", event.Tag)
	log.V(1).Info("Processing tag event")

	// Check if this matches promotion pattern
	if platform.Spec.GitOps.Promotion != nil && platform.Spec.GitOps.Promotion.Enabled {
		if m.matchesPromotionPattern(event.Tag, platform.Spec.GitOps.Promotion.TagPattern) {
			return m.promoteToEnvironment(ctx, platform, event.Tag)
		}
	}

	return nil
}

// DetectAndRemediateDrift checks for configuration drift
func (m *Manager) DetectAndRemediateDrift(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name)
	log.V(1).Info("Detecting configuration drift")

	// Check if drift detection is enabled
	if platform.Spec.GitOps == nil || platform.Spec.GitOps.DriftDetection == nil || !platform.Spec.GitOps.DriftDetection.Enabled {
		return nil
	}

	// Detect drift
	driftResults, err := m.DriftDetector.DetectDrift(ctx, platform)
	if err != nil {
		return fmt.Errorf("detecting drift: %w", err)
	}

	if len(driftResults) == 0 {
		log.V(1).Info("No drift detected")
		return nil
	}

	log.Info("Drift detected", "items", len(driftResults))

	// Update status with drift information
	platform.Status.GitOps.DriftDetected = true
	platform.Status.GitOps.DriftItems = len(driftResults)
	platform.Status.GitOps.LastDriftCheckTime = &metav1.Time{Time: time.Now()}

	// Auto-remediate if enabled
	if platform.Spec.GitOps.DriftDetection.AutoRemediate {
		log.Info("Auto-remediating drift")
		if err := m.DriftDetector.RemediateDrift(ctx, platform, driftResults); err != nil {
			return fmt.Errorf("remediating drift: %w", err)
		}
		platform.Status.GitOps.DriftDetected = false
		platform.Status.GitOps.DriftItems = 0
	}

	return nil
}

// PromoteEnvironment promotes a platform configuration to another environment
func (m *Manager) PromoteEnvironment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv string) error {
	log := m.Log.WithValues("platform", platform.Name, "targetEnv", targetEnv)
	log.Info("Promoting to environment")

	// Validate promotion is enabled
	if platform.Spec.GitOps == nil || platform.Spec.GitOps.Promotion == nil || !platform.Spec.GitOps.Promotion.Enabled {
		return fmt.Errorf("promotion not enabled for platform")
	}

	// Find target environment configuration
	var targetEnvConfig *observabilityv1.EnvironmentConfig
	for _, env := range platform.Spec.GitOps.Promotion.Environments {
		if env.Name == targetEnv {
			targetEnvConfig = &env
			break
		}
	}

	if targetEnvConfig == nil {
		return fmt.Errorf("target environment %s not found", targetEnv)
	}

	// Perform promotion based on strategy
	switch platform.Spec.GitOps.Promotion.Strategy {
	case observabilityv1.PromotionStrategyManual:
		return m.performManualPromotion(ctx, platform, targetEnvConfig)
	case observabilityv1.PromotionStrategyAutomatic:
		return m.performAutomaticPromotion(ctx, platform, targetEnvConfig)
	case observabilityv1.PromotionStrategyGated:
		return m.performGatedPromotion(ctx, platform, targetEnvConfig)
	default:
		return fmt.Errorf("unknown promotion strategy: %s", platform.Spec.GitOps.Promotion.Strategy)
	}
}

// RollbackDeployment rolls back a failed deployment
func (m *Manager) RollbackDeployment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, reason string) error {
	log := m.Log.WithValues("platform", platform.Name, "reason", reason)
	log.Info("Rolling back deployment")

	// Validate rollback is enabled
	if platform.Spec.GitOps == nil || platform.Spec.GitOps.Rollback == nil || !platform.Spec.GitOps.Rollback.Enabled {
		return fmt.Errorf("rollback not enabled for platform")
	}

	// Get rollback history
	history, err := m.RollbackMgr.GetHistory(ctx, platform)
	if err != nil {
		return fmt.Errorf("getting rollback history: %w", err)
	}

	if len(history) == 0 {
		return fmt.Errorf("no rollback history available")
	}

	// Perform rollback to last known good state
	targetRevision := history[0].Revision
	if err := m.RollbackMgr.RollbackToRevision(ctx, platform, targetRevision, reason); err != nil {
		return fmt.Errorf("rolling back to revision %s: %w", targetRevision, err)
	}

	// Update status
	platform.Status.GitOps.RollbackStatus = &observabilityv1.RollbackStatus{
		InProgress:     true,
		TargetRevision: targetRevision,
		Reason:         reason,
		StartTime:      &metav1.Time{Time: time.Now()},
	}

	return nil
}

// validateGitOpsConfig validates GitOps configuration
func (m *Manager) validateGitOpsConfig(config *observabilityv1.GitOpsConfig) error {
	if config.Repository == nil {
		return fmt.Errorf("repository configuration required")
	}

	if config.Repository.URL == "" {
		return fmt.Errorf("repository URL required")
	}

	if config.Repository.Path == "" {
		config.Repository.Path = "/"
	}

	if config.Repository.Branch == "" {
		config.Repository.Branch = "main"
	}

	return nil
}

// cleanupGitOpsResources removes GitOps resources when disabled
func (m *Manager) cleanupGitOpsResources(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	log := m.Log.WithValues("platform", platform.Name)
	log.V(1).Info("Cleaning up GitOps resources")

	// Clean up based on previous provider
	if platform.Status.GitOps != nil {
		switch platform.Status.GitOps.Provider {
		case observabilityv1.GitOpsProviderArgoCD:
			if err := m.ArgoCDManager.DeleteApplication(ctx, platform); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("deleting ArgoCD application: %w", err)
			}
		case observabilityv1.GitOpsProviderFlux:
			if err := m.FluxManager.DeleteResources(ctx, platform); err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("deleting Flux resources: %w", err)
			}
		}
	}

	// Clear GitOps status
	platform.Status.GitOps = nil
	return nil
}

// findPlatformsByRepo finds platforms using a specific git repository
func (m *Manager) findPlatformsByRepo(ctx context.Context, repoURL string) ([]*observabilityv1.ObservabilityPlatform, error) {
	var platforms observabilityv1.ObservabilityPlatformList
	if err := m.Client.List(ctx, &platforms); err != nil {
		return nil, fmt.Errorf("listing platforms: %w", err)
	}

	var result []*observabilityv1.ObservabilityPlatform
	for i := range platforms.Items {
		platform := &platforms.Items[i]
		if platform.Spec.GitOps != nil && 
		   platform.Spec.GitOps.Repository != nil && 
		   platform.Spec.GitOps.Repository.URL == repoURL {
			result = append(result, platform)
		}
	}

	return result, nil
}

// matchesPromotionPattern checks if a tag matches the promotion pattern
func (m *Manager) matchesPromotionPattern(tag, pattern string) bool {
	// Simple pattern matching, could be enhanced with regex
	if pattern == "" {
		pattern = "v*"
	}
	
	// Basic glob matching
	if pattern == "v*" && len(tag) > 1 && tag[0] == 'v' {
		return true
	}
	
	return tag == pattern
}

// createPreviewEnvironment creates a preview environment for a PR
func (m *Manager) createPreviewEnvironment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, pr *PullRequest) error {
	log := m.Log.WithValues("platform", platform.Name, "pr", pr.Number)
	log.Info("Creating preview environment")

	// Create preview platform
	preview := platform.DeepCopy()
	preview.Name = fmt.Sprintf("%s-pr-%d", platform.Name, pr.Number)
	preview.Namespace = fmt.Sprintf("%s-preview", platform.Namespace)
	preview.Labels["preview"] = "true"
	preview.Labels["pr-number"] = fmt.Sprintf("%d", pr.Number)

	// Adjust resources for preview
	if preview.Spec.Components.Prometheus != nil && preview.Spec.Components.Prometheus.Resources != nil {
		// Reduce resources for preview environments
		preview.Spec.Components.Prometheus.Resources.Requests.Memory().Set(1073741824) // 1Gi
		preview.Spec.Components.Prometheus.Resources.Requests.Cpu().SetMilli(250)
	}

	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: preview.Namespace,
			Labels: map[string]string{
				"preview":   "true",
				"pr-number": fmt.Sprintf("%d", pr.Number),
			},
		},
	}
	if err := m.Client.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("creating preview namespace: %w", err)
	}

	// Create preview platform
	if err := m.Client.Create(ctx, preview); err != nil {
		return fmt.Errorf("creating preview platform: %w", err)
	}

	return nil
}

// deletePreviewEnvironment deletes a preview environment
func (m *Manager) deletePreviewEnvironment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, pr *PullRequest) error {
	log := m.Log.WithValues("platform", platform.Name, "pr", pr.Number)
	log.Info("Deleting preview environment")

	// Delete preview platform
	preview := &observabilityv1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pr-%d", platform.Name, pr.Number),
			Namespace: fmt.Sprintf("%s-preview", platform.Namespace),
		},
	}
	
	if err := m.Client.Delete(ctx, preview); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("deleting preview platform: %w", err)
	}

	// Delete namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: preview.Namespace,
		},
	}
	
	if err := m.Client.Delete(ctx, ns); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("deleting preview namespace: %w", err)
	}

	return nil
}

// promoteToEnvironment promotes a tagged version to an environment
func (m *Manager) promoteToEnvironment(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, tag string) error {
	log := m.Log.WithValues("platform", platform.Name, "tag", tag)
	log.Info("Promoting tag to environment")

	// Determine target environment based on tag
	var targetEnv string
	if platform.Spec.GitOps.Promotion != nil {
		for _, env := range platform.Spec.GitOps.Promotion.Environments {
			if env.AutoPromote && m.matchesPromotionPattern(tag, env.PromotionPattern) {
				targetEnv = env.Name
				break
			}
		}
	}

	if targetEnv == "" {
		log.V(1).Info("No environment matches tag pattern", "tag", tag)
		return nil
	}

	return m.PromoteEnvironment(ctx, platform, targetEnv)
}

// performManualPromotion performs manual promotion
func (m *Manager) performManualPromotion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv *observabilityv1.EnvironmentConfig) error {
	// For manual promotion, we just update the target branch/tag
	return m.SyncManager.UpdateTargetRevision(ctx, platform, targetEnv.TargetRevision)
}

// performAutomaticPromotion performs automatic promotion
func (m *Manager) performAutomaticPromotion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv *observabilityv1.EnvironmentConfig) error {
	// Check if all requirements are met
	if targetEnv.Requirements != nil {
		for _, req := range targetEnv.Requirements.TestsPassed {
			// Check if tests passed (would integrate with CI/CD)
			if !m.checkTestsPassed(ctx, platform, req) {
				return fmt.Errorf("test requirement not met: %s", req)
			}
		}
		
		// Check approval requirements
		if targetEnv.Requirements.Approvals > 0 {
			approvals, err := m.getApprovals(ctx, platform, targetEnv.Name)
			if err != nil {
				return fmt.Errorf("checking approvals: %w", err)
			}
			if len(approvals) < targetEnv.Requirements.Approvals {
				return fmt.Errorf("insufficient approvals: %d/%d", len(approvals), targetEnv.Requirements.Approvals)
			}
		}
	}

	// Perform promotion
	return m.SyncManager.PromoteToEnvironment(ctx, platform, targetEnv)
}

// performGatedPromotion performs gated promotion with checks
func (m *Manager) performGatedPromotion(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv *observabilityv1.EnvironmentConfig) error {
	// Similar to automatic but with additional gates
	if err := m.performAutomaticPromotion(ctx, platform, targetEnv); err != nil {
		return err
	}

	// Additional gate checks (e.g., canary analysis, smoke tests)
	if targetEnv.Gates != nil {
		for _, gate := range targetEnv.Gates {
			if err := m.checkGate(ctx, platform, gate); err != nil {
				// Rollback on gate failure
				m.RollbackMgr.RollbackPromotion(ctx, platform, targetEnv.Name, fmt.Sprintf("gate failed: %s", gate))
				return fmt.Errorf("gate check failed: %w", err)
			}
		}
	}

	return nil
}

// checkTestsPassed checks if required tests have passed
func (m *Manager) checkTestsPassed(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, testName string) bool {
	// This would integrate with CI/CD systems
	// For now, return true as placeholder
	return true
}

// getApprovals gets approvals for promotion
func (m *Manager) getApprovals(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, envName string) ([]string, error) {
	// This would integrate with approval systems (GitHub PRs, etc.)
	// For now, return empty list
	return []string{}, nil
}

// checkGate checks a promotion gate
func (m *Manager) checkGate(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, gate string) error {
	// This would implement various gate checks (canary analysis, etc.)
	// For now, return nil
	return nil
}
