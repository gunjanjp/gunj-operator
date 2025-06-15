package gitops

import (
	"context"
	"fmt"
	"strings"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FluxManager implements GitOps integration with Flux
type FluxManager struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewFluxManager creates a new Flux manager
func NewFluxManager(client client.Client, scheme *runtime.Scheme) (GitOpsManager, error) {
	return &FluxManager{
		client: client,
		scheme: scheme,
	}, nil
}

// SetupGitOps configures Flux for the platform
func (m *FluxManager) SetupGitOps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := ctrl.LoggerFrom(ctx).WithValues("gitops", "flux")
	log.Info("Setting up Flux for platform")

	gitOpsConfig := platform.Spec.GitOps
	if gitOpsConfig == nil {
		return nil
	}

	// Create GitRepository source
	if err := m.createGitRepository(ctx, platform); err != nil {
		return fmt.Errorf("creating GitRepository: %w", err)
	}

	// Create Kustomization for each environment
	for _, env := range gitOpsConfig.Environments {
		if err := m.createKustomization(ctx, platform, env); err != nil {
			return fmt.Errorf("creating Kustomization for environment %s: %w", env.Name, err)
		}
	}

	// Set up image automation if configured
	if platform.Spec.GitOps.ImageAutomation != nil && platform.Spec.GitOps.ImageAutomation.Enabled {
		if err := m.setupImageAutomation(ctx, platform); err != nil {
			return fmt.Errorf("setting up image automation: %w", err)
		}
	}

	return nil
}

// createGitRepository creates a Flux GitRepository resource
func (m *FluxManager) createGitRepository(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := ctrl.LoggerFrom(ctx)

	interval := "1m"
	if platform.Spec.GitOps.Repository.Interval != "" {
		interval = platform.Spec.GitOps.Repository.Interval
	}

	gitRepo := &sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-source", platform.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/platform":   platform.Name,
			},
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: platform.Spec.GitOps.Repository.URL,
			Reference: &sourcev1.GitRepositoryRef{
				Branch: platform.Spec.GitOps.Repository.Branch,
			},
			Interval: metav1.Duration{
				Duration: parseDuration(interval),
			},
		},
	}

	// Add secret reference if provided
	if platform.Spec.GitOps.Repository.SecretRef != "" {
		gitRepo.Spec.SecretRef = &corev1.LocalObjectReference{
			Name: platform.Spec.GitOps.Repository.SecretRef,
		}
	}

	// Set owner reference
	if err := ctrl.SetControllerReference(platform, gitRepo, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	// Create or update
	existing := &sourcev1.GitRepository{}
	err := m.client.Get(ctx, client.ObjectKeyFromObject(gitRepo), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating GitRepository")
			return m.client.Create(ctx, gitRepo)
		}
		return fmt.Errorf("getting GitRepository: %w", err)
	}

	// Update existing
	existing.Spec = gitRepo.Spec
	log.Info("Updating GitRepository")
	return m.client.Update(ctx, existing)
}

// createKustomization creates a Flux Kustomization for an environment
func (m *FluxManager) createKustomization(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, env observabilityv1beta1.GitOpsEnvironment) error {
	log := ctrl.LoggerFrom(ctx).WithValues("environment", env.Name)

	kustomization := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", platform.Name, env.Name),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/platform":   platform.Name,
				"observability.io/environment": env.Name,
			},
		},
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind:      "GitRepository",
				Name:      fmt.Sprintf("%s-source", platform.Name),
				Namespace: platform.Namespace,
			},
			Path: m.getPathForEnvironment(platform.Spec.GitOps.Repository, env),
			Prune: true,
			Interval: metav1.Duration{
				Duration: parseDuration("5m"),
			},
			TargetNamespace: env.Namespace,
		},
	}

	// Configure health checks
	kustomization.Spec.HealthChecks = []kustomizev1.HealthCheck{
		{
			Kind:      "Deployment",
			Namespace: env.Namespace,
		},
		{
			Kind:      "StatefulSet",
			Namespace: env.Namespace,
		},
	}

	// Configure dependencies if this is not the first environment
	if env.PromotionPolicy != nil && env.PromotionPolicy.DependsOn != "" {
		kustomization.Spec.DependsOn = []kustomizev1.Dependency{
			{
				Name:      fmt.Sprintf("%s-%s", platform.Name, env.PromotionPolicy.DependsOn),
				Namespace: platform.Namespace,
			},
		}
	}

	// Add post-build substitutions for environment-specific values
	kustomization.Spec.PostBuild = &kustomizev1.PostBuild{
		Substitute: map[string]string{
			"environment": env.Name,
			"platform":    platform.Name,
			"namespace":   env.Namespace,
		},
	}

	// Set owner reference
	if err := ctrl.SetControllerReference(platform, kustomization, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	// Create or update
	existing := &kustomizev1.Kustomization{}
	err := m.client.Get(ctx, client.ObjectKeyFromObject(kustomization), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Kustomization")
			return m.client.Create(ctx, kustomization)
		}
		return fmt.Errorf("getting Kustomization: %w", err)
	}

	// Update existing
	existing.Spec = kustomization.Spec
	log.Info("Updating Kustomization")
	return m.client.Update(ctx, existing)
}

// SyncWithGit triggers a sync with the Git repository
func (m *FluxManager) SyncWithGit(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := ctrl.LoggerFrom(ctx).WithValues("gitops", "flux")
	log.Info("Syncing with Git repository")

	// Annotate GitRepository to trigger reconciliation
	gitRepo := &sourcev1.GitRepository{}
	gitRepoKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-source", platform.Name),
		Namespace: platform.Namespace,
	}

	if err := m.client.Get(ctx, gitRepoKey, gitRepo); err != nil {
		return fmt.Errorf("getting GitRepository: %w", err)
	}

	// Add reconciliation annotation
	if gitRepo.Annotations == nil {
		gitRepo.Annotations = make(map[string]string)
	}
	gitRepo.Annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)

	if err := m.client.Update(ctx, gitRepo); err != nil {
		return fmt.Errorf("updating GitRepository: %w", err)
	}

	// Also trigger reconciliation for all Kustomizations
	kustomizationList := &kustomizev1.KustomizationList{}
	if err := m.client.List(ctx, kustomizationList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return fmt.Errorf("listing Kustomizations: %w", err)
	}

	for _, k := range kustomizationList.Items {
		if k.Annotations == nil {
			k.Annotations = make(map[string]string)
		}
		k.Annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)
		
		if err := m.client.Update(ctx, &k); err != nil {
			return fmt.Errorf("updating Kustomization %s: %w", k.Name, err)
		}
	}

	return nil
}

// PromoteEnvironment promotes configuration between environments
func (m *FluxManager) PromoteEnvironment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, from, to string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("from", from, "to", to)
	log.Info("Promoting environment")

	// Get source Kustomization
	sourceKustomization := &kustomizev1.Kustomization{}
	sourceKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", platform.Name, from),
		Namespace: platform.Namespace,
	}
	if err := m.client.Get(ctx, sourceKey, sourceKustomization); err != nil {
		return fmt.Errorf("getting source Kustomization: %w", err)
	}

	// Check if source is ready
	if !isKustomizationReady(sourceKustomization) {
		return fmt.Errorf("source environment %s is not ready", from)
	}

	// Get target Kustomization
	targetKustomization := &kustomizev1.Kustomization{}
	targetKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", platform.Name, to),
		Namespace: platform.Namespace,
	}
	if err := m.client.Get(ctx, targetKey, targetKustomization); err != nil {
		return fmt.Errorf("getting target Kustomization: %w", err)
	}

	// Update target to use source revision
	if targetKustomization.Annotations == nil {
		targetKustomization.Annotations = make(map[string]string)
	}
	targetKustomization.Annotations["observability.io/promoted-from"] = from
	targetKustomization.Annotations["observability.io/promoted-at"] = time.Now().Format(time.RFC3339)
	targetKustomization.Annotations["observability.io/promoted-revision"] = sourceKustomization.Status.LastAppliedRevision
	
	// Trigger reconciliation
	targetKustomization.Annotations["reconcile.fluxcd.io/requestedAt"] = time.Now().Format(time.RFC3339)

	return m.client.Update(ctx, targetKustomization)
}

// Rollback rolls back to a previous configuration
func (m *FluxManager) Rollback(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("revision", revision)
	log.Info("Rolling back to revision")

	// Update GitRepository to specific revision
	gitRepo := &sourcev1.GitRepository{}
	gitRepoKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-source", platform.Name),
		Namespace: platform.Namespace,
	}

	if err := m.client.Get(ctx, gitRepoKey, gitRepo); err != nil {
		return fmt.Errorf("getting GitRepository: %w", err)
	}

	// Create a new reference for the specific revision
	gitRepo.Spec.Reference = &sourcev1.GitRepositoryRef{
		Commit: revision,
	}

	// Add rollback annotation
	if gitRepo.Annotations == nil {
		gitRepo.Annotations = make(map[string]string)
	}
	gitRepo.Annotations["observability.io/rollback-revision"] = revision
	gitRepo.Annotations["observability.io/rollback-at"] = time.Now().Format(time.RFC3339)

	if err := m.client.Update(ctx, gitRepo); err != nil {
		return fmt.Errorf("updating GitRepository: %w", err)
	}

	// Trigger reconciliation for all Kustomizations
	return m.SyncWithGit(ctx, platform)
}

// GetSyncStatus returns the current GitOps sync status
func (m *FluxManager) GetSyncStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*GitOpsSyncStatus, error) {
	status := &GitOpsSyncStatus{
		Provider:     Flux,
		Environments: make(map[string]EnvironmentStatus),
	}

	// Get GitRepository status
	gitRepo := &sourcev1.GitRepository{}
	gitRepoKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-source", platform.Name),
		Namespace: platform.Namespace,
	}

	if err := m.client.Get(ctx, gitRepoKey, gitRepo); err != nil {
		return nil, fmt.Errorf("getting GitRepository: %w", err)
	}

	// Set revision from GitRepository
	if gitRepo.Status.Artifact != nil {
		status.Revision = gitRepo.Status.Artifact.Revision
	}

	// Get all Kustomizations for this platform
	kustomizationList := &kustomizev1.KustomizationList{}
	if err := m.client.List(ctx, kustomizationList, client.InNamespace(platform.Namespace), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return nil, fmt.Errorf("listing Kustomizations: %w", err)
	}

	// Aggregate status from all Kustomizations
	overallState := SyncStateSynced
	var latestSyncTime *metav1.Time

	for _, k := range kustomizationList.Items {
		envName := k.Labels["observability.io/environment"]
		
		envStatus := EnvironmentStatus{
			Name:      envName,
			Namespace: k.Spec.TargetNamespace,
			Revision:  k.Status.LastAppliedRevision,
			Status:    string(getKustomizationConditionStatus(&k)),
			Health:    string(getKustomizationHealth(&k)),
		}
		
		// Check promotion info
		if promotedFrom, ok := k.Annotations["observability.io/promoted-from"]; ok {
			envStatus.PromotedFrom = promotedFrom
			envStatus.PromotedAt = k.Annotations["observability.io/promoted-at"]
		}
		
		status.Environments[envName] = envStatus
		
		// Update overall state
		if !isKustomizationReady(&k) {
			overallState = SyncStateOutOfSync
		}
		
		// Track latest sync time
		if k.Status.LastAppliedTime != nil {
			if latestSyncTime == nil || k.Status.LastAppliedTime.After(latestSyncTime.Time) {
				latestSyncTime = k.Status.LastAppliedTime
			}
		}
	}

	status.SyncState = overallState
	if latestSyncTime != nil {
		status.LastSyncTime = latestSyncTime.Format(time.RFC3339)
	}

	return status, nil
}

// ValidateGitOpsConfig validates GitOps configuration
func (m *FluxManager) ValidateGitOpsConfig(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.GitOps == nil {
		return nil
	}

	gitOps := platform.Spec.GitOps

	// Validate repository
	if gitOps.Repository.URL == "" {
		return fmt.Errorf("repository URL is required")
	}
	if gitOps.Repository.Branch == "" && gitOps.Repository.Tag == "" {
		return fmt.Errorf("either branch or tag must be specified")
	}

	// Validate environments
	if len(gitOps.Environments) == 0 {
		return fmt.Errorf("at least one environment must be configured")
	}

	envNames := make(map[string]bool)
	for _, env := range gitOps.Environments {
		if env.Name == "" {
			return fmt.Errorf("environment name is required")
		}
		if env.Namespace == "" {
			return fmt.Errorf("environment namespace is required for %s", env.Name)
		}
		if envNames[env.Name] {
			return fmt.Errorf("duplicate environment name: %s", env.Name)
		}
		envNames[env.Name] = true
	}

	// Validate interval format
	if gitOps.Repository.Interval != "" {
		if _, err := time.ParseDuration(gitOps.Repository.Interval); err != nil {
			return fmt.Errorf("invalid interval format: %s", gitOps.Repository.Interval)
		}
	}

	return nil
}

// setupImageAutomation sets up Flux image automation
func (m *FluxManager) setupImageAutomation(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	// This would set up ImageRepository, ImagePolicy, and ImageUpdateAutomation resources
	// Implementation depends on specific requirements
	log := ctrl.LoggerFrom(ctx)
	log.Info("Setting up image automation", "platform", platform.Name)
	
	// TODO: Implement image automation setup
	return nil
}

// Helper methods

func (m *FluxManager) getPathForEnvironment(repo observabilityv1beta1.GitRepository, env observabilityv1beta1.GitOpsEnvironment) string {
	if env.Path != "" {
		return env.Path
	}
	if repo.Path != "" {
		return fmt.Sprintf("%s/%s", repo.Path, env.Name)
	}
	return env.Name
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Minute // default
	}
	return d
}

func isKustomizationReady(k *kustomizev1.Kustomization) bool {
	for _, cond := range k.Status.Conditions {
		if cond.Type == "Ready" {
			return cond.Status == metav1.ConditionTrue
		}
	}
	return false
}

func getKustomizationConditionStatus(k *kustomizev1.Kustomization) string {
	for _, cond := range k.Status.Conditions {
		if cond.Type == "Ready" {
			if cond.Status == metav1.ConditionTrue {
				return "Ready"
			}
			return cond.Reason
		}
	}
	return "Unknown"
}

func getKustomizationHealth(k *kustomizev1.Kustomization) string {
	// Check various conditions to determine health
	for _, cond := range k.Status.Conditions {
		if cond.Type == "Ready" && cond.Status != metav1.ConditionTrue {
			return "Degraded"
		}
		if cond.Type == "Stalled" && cond.Status == metav1.ConditionTrue {
			return "Stalled"
		}
	}
	
	if isKustomizationReady(k) {
		return "Healthy"
	}
	
	return "Unknown"
}
