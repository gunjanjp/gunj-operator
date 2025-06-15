package gitops

import (
	"context"
	"fmt"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MultiProviderManager supports multiple GitOps providers
type MultiProviderManager struct {
	client    client.Client
	scheme    *runtime.Scheme
	providers map[GitOpsProvider]GitOpsManager
}

// NewMultiProviderManager creates a manager that can work with multiple providers
func NewMultiProviderManager(client client.Client, scheme *runtime.Scheme) (GitOpsManager, error) {
	manager := &MultiProviderManager{
		client:    client,
		scheme:    scheme,
		providers: make(map[GitOpsProvider]GitOpsManager),
	}

	// Initialize available providers
	argoManager, err := NewArgoCDManager(client, scheme)
	if err != nil {
		return nil, fmt.Errorf("creating ArgoCD manager: %w", err)
	}
	manager.providers[ArgoCD] = argoManager

	fluxManager, err := NewFluxManager(client, scheme)
	if err != nil {
		return nil, fmt.Errorf("creating Flux manager: %w", err)
	}
	manager.providers[Flux] = fluxManager

	return manager, nil
}

// SetupGitOps configures GitOps for the platform
func (m *MultiProviderManager) SetupGitOps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := ctrl.LoggerFrom(ctx)
	
	if platform.Spec.GitOps == nil {
		log.V(1).Info("No GitOps configuration found")
		return nil
	}

	provider := m.getProvider(platform)
	log.Info("Setting up GitOps", "provider", provider)

	mgr, err := m.getManager(provider)
	if err != nil {
		return err
	}

	return mgr.SetupGitOps(ctx, platform)
}

// SyncWithGit synchronizes the platform configuration with Git repository
func (m *MultiProviderManager) SyncWithGit(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	mgr, err := m.getManagerForPlatform(platform)
	if err != nil {
		return err
	}

	return mgr.SyncWithGit(ctx, platform)
}

// PromoteEnvironment promotes configuration between environments
func (m *MultiProviderManager) PromoteEnvironment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, from, to string) error {
	mgr, err := m.getManagerForPlatform(platform)
	if err != nil {
		return err
	}

	// Validate environments exist
	if err := m.validateEnvironments(platform, from, to); err != nil {
		return err
	}

	// Check promotion policy
	if err := m.checkPromotionPolicy(ctx, platform, from, to); err != nil {
		return err
	}

	return mgr.PromoteEnvironment(ctx, platform, from, to)
}

// Rollback rolls back to a previous configuration
func (m *MultiProviderManager) Rollback(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("revision", revision)
	
	mgr, err := m.getManagerForPlatform(platform)
	if err != nil {
		return err
	}

	// Check rollback policy
	if platform.Spec.GitOps.RollbackConfig != nil {
		if !platform.Spec.GitOps.RollbackConfig.AutoRollback {
			log.Info("Manual rollback requested")
		}
	}

	// Perform rollback
	if err := mgr.Rollback(ctx, platform, revision); err != nil {
		return fmt.Errorf("performing rollback: %w", err)
	}

	// Update platform status
	if err := m.updateRollbackStatus(ctx, platform, revision); err != nil {
		return fmt.Errorf("updating rollback status: %w", err)
	}

	return nil
}

// GetSyncStatus returns the current GitOps sync status
func (m *MultiProviderManager) GetSyncStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*GitOpsSyncStatus, error) {
	mgr, err := m.getManagerForPlatform(platform)
	if err != nil {
		return nil, err
	}

	return mgr.GetSyncStatus(ctx, platform)
}

// ValidateGitOpsConfig validates GitOps configuration
func (m *MultiProviderManager) ValidateGitOpsConfig(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.GitOps == nil {
		return nil
	}

	// Validate provider
	provider := m.getProvider(platform)
	mgr, exists := m.providers[provider]
	if !exists {
		return fmt.Errorf("unsupported GitOps provider: %s", provider)
	}

	// Provider-specific validation
	if err := mgr.ValidateGitOpsConfig(platform); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	// Common validation
	return m.validateCommonConfig(platform)
}

// Helper methods

func (m *MultiProviderManager) getProvider(platform *observabilityv1beta1.ObservabilityPlatform) GitOpsProvider {
	if platform.Spec.GitOps == nil || platform.Spec.GitOps.Provider == "" {
		// Default to ArgoCD
		return ArgoCD
	}
	return GitOpsProvider(platform.Spec.GitOps.Provider)
}

func (m *MultiProviderManager) getManager(provider GitOpsProvider) (GitOpsManager, error) {
	mgr, exists := m.providers[provider]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	return mgr, nil
}

func (m *MultiProviderManager) getManagerForPlatform(platform *observabilityv1beta1.ObservabilityPlatform) (GitOpsManager, error) {
	if platform.Spec.GitOps == nil {
		return nil, fmt.Errorf("GitOps not configured for platform")
	}
	
	provider := m.getProvider(platform)
	return m.getManager(provider)
}

func (m *MultiProviderManager) validateEnvironments(platform *observabilityv1beta1.ObservabilityPlatform, envNames ...string) error {
	envMap := make(map[string]bool)
	for _, env := range platform.Spec.GitOps.Environments {
		envMap[env.Name] = true
	}

	for _, name := range envNames {
		if !envMap[name] {
			return fmt.Errorf("environment %s not found", name)
		}
	}

	return nil
}

func (m *MultiProviderManager) checkPromotionPolicy(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, from, to string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("from", from, "to", to)

	// Find target environment
	var targetEnv *observabilityv1beta1.GitOpsEnvironment
	for _, env := range platform.Spec.GitOps.Environments {
		if env.Name == to {
			targetEnv = &env
			break
		}
	}

	if targetEnv == nil {
		return fmt.Errorf("target environment %s not found", to)
	}

	// Check promotion policy
	if targetEnv.PromotionPolicy != nil {
		policy := targetEnv.PromotionPolicy

		// Check if promotion is allowed
		if policy.ApprovalRequired {
			log.Info("Manual approval required for promotion")
			// In a real implementation, this would check for approval
		}

		// Check required tests
		if len(policy.RequiredTests) > 0 {
			log.Info("Checking required tests", "tests", policy.RequiredTests)
			// In a real implementation, this would verify test results
		}

		// Check promotion timing
		if policy.PromoteAfter != "" {
			log.Info("Promotion delay configured", "delay", policy.PromoteAfter)
			// In a real implementation, this would check timing constraints
		}
	}

	return nil
}

func (m *MultiProviderManager) validateCommonConfig(platform *observabilityv1beta1.ObservabilityPlatform) error {
	gitOps := platform.Spec.GitOps

	// Validate auto-promotion configuration
	if gitOps.AutoPromotion != nil && gitOps.AutoPromotion.Enabled {
		if gitOps.AutoPromotion.Strategy == "parallel" && gitOps.AutoPromotion.MaxParallel <= 0 {
			return fmt.Errorf("maxParallel must be greater than 0 for parallel promotion")
		}
	}

	// Validate rollback configuration
	if gitOps.RollbackConfig != nil {
		if gitOps.RollbackConfig.FailureThreshold <= 0 {
			return fmt.Errorf("failureThreshold must be greater than 0")
		}
		if gitOps.RollbackConfig.MaxHistory <= 0 {
			return fmt.Errorf("maxHistory must be greater than 0")
		}
	}

	return nil
}

func (m *MultiProviderManager) updateRollbackStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision string) error {
	// Update platform status to reflect rollback
	platform.Status.LastRollback = &observabilityv1beta1.RollbackStatus{
		Revision:  revision,
		Timestamp: ctrl.Now().Format(ctrl.RFC3339),
		Reason:    "Manual rollback requested",
	}

	return m.client.Status().Update(ctx, platform)
}
