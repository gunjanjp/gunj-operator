package gitops

import (
	"context"
	"fmt"
	"time"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
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

// ArgoCDManager implements GitOps integration with ArgoCD
type ArgoCDManager struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewArgoCDManager creates a new ArgoCD manager
func NewArgoCDManager(client client.Client, scheme *runtime.Scheme) (GitOpsManager, error) {
	return &ArgoCDManager{
		client: client,
		scheme: scheme,
	}, nil
}

// SetupGitOps configures ArgoCD for the platform
func (m *ArgoCDManager) SetupGitOps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := ctrl.LoggerFrom(ctx).WithValues("gitops", "argocd")
	log.Info("Setting up ArgoCD for platform")

	gitOpsConfig := platform.Spec.GitOps
	if gitOpsConfig == nil {
		return nil
	}

	// Create ArgoCD applications for each environment
	for _, env := range gitOpsConfig.Environments {
		if err := m.createArgoCDApplication(ctx, platform, env); err != nil {
			return fmt.Errorf("creating ArgoCD application for environment %s: %w", env.Name, err)
		}
	}

	// Create AppProject if needed
	if err := m.createAppProject(ctx, platform); err != nil {
		return fmt.Errorf("creating ArgoCD AppProject: %w", err)
	}

	return nil
}

// createArgoCDApplication creates an ArgoCD Application for an environment
func (m *ArgoCDManager) createArgoCDApplication(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, env observabilityv1beta1.GitOpsEnvironment) error {
	log := ctrl.LoggerFrom(ctx).WithValues("environment", env.Name)

	app := &argov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", platform.Name, env.Name),
			Namespace: "argocd", // ArgoCD namespace
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/platform":   platform.Name,
				"observability.io/environment": env.Name,
			},
			Annotations: map[string]string{
				"observability.io/platform-namespace": platform.Namespace,
			},
		},
		Spec: argov1alpha1.ApplicationSpec{
			Project: fmt.Sprintf("%s-project", platform.Name),
			Source: &argov1alpha1.ApplicationSource{
				RepoURL:        platform.Spec.GitOps.Repository.URL,
				Path:           m.getPathForEnvironment(platform.Spec.GitOps.Repository, env),
				TargetRevision: m.getBranchForEnvironment(platform.Spec.GitOps.Repository, env),
			},
			Destination: argov1alpha1.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: env.Namespace,
			},
			SyncPolicy: m.buildSyncPolicy(platform, env),
		},
	}

	// Set owner reference if ArgoCD is in the same cluster
	if err := ctrl.SetControllerReference(platform, app, m.scheme); err != nil {
		log.V(1).Info("Could not set owner reference, ArgoCD might be in different cluster")
	}

	// Create or update the application
	existing := &argov1alpha1.Application{}
	err := m.client.Get(ctx, client.ObjectKeyFromObject(app), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating ArgoCD application")
			return m.client.Create(ctx, app)
		}
		return fmt.Errorf("getting ArgoCD application: %w", err)
	}

	// Update existing application
	existing.Spec = app.Spec
	log.Info("Updating ArgoCD application")
	return m.client.Update(ctx, existing)
}

// createAppProject creates an ArgoCD AppProject
func (m *ArgoCDManager) createAppProject(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	project := &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-project", platform.Name),
			Namespace: "argocd",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "gunj-operator",
				"observability.io/platform":   platform.Name,
			},
		},
		Spec: argov1alpha1.AppProjectSpec{
			Description: fmt.Sprintf("Project for observability platform %s", platform.Name),
			SourceRepos: []string{platform.Spec.GitOps.Repository.URL},
			Destinations: m.buildDestinations(platform),
			ClusterResourceWhitelist: []metav1.GroupKind{
				{Group: "*", Kind: "*"},
			},
			NamespaceResourceWhitelist: []metav1.GroupKind{
				{Group: "*", Kind: "*"},
			},
		},
	}

	existing := &argov1alpha1.AppProject{}
	err := m.client.Get(ctx, client.ObjectKeyFromObject(project), existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return m.client.Create(ctx, project)
		}
		return fmt.Errorf("getting ArgoCD project: %w", err)
	}

	existing.Spec = project.Spec
	return m.client.Update(ctx, existing)
}

// SyncWithGit triggers a sync with the Git repository
func (m *ArgoCDManager) SyncWithGit(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	log := ctrl.LoggerFrom(ctx).WithValues("gitops", "argocd")
	log.Info("Syncing with Git repository")

	// Get all ArgoCD applications for this platform
	appList := &argov1alpha1.ApplicationList{}
	if err := m.client.List(ctx, appList, client.InNamespace("argocd"), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return fmt.Errorf("listing ArgoCD applications: %w", err)
	}

	// Trigger sync for each application
	for _, app := range appList.Items {
		log.Info("Triggering sync", "application", app.Name)
		
		// Update sync operation
		app.Operation = &argov1alpha1.Operation{
			Sync: &argov1alpha1.SyncOperation{
				Revision: app.Spec.Source.TargetRevision,
				Prune:    true,
				DryRun:   false,
			},
		}
		
		if err := m.client.Update(ctx, &app); err != nil {
			return fmt.Errorf("triggering sync for application %s: %w", app.Name, err)
		}
	}

	return nil
}

// PromoteEnvironment promotes configuration between environments
func (m *ArgoCDManager) PromoteEnvironment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, from, to string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("from", from, "to", to)
	log.Info("Promoting environment")

	// Get source application
	sourceApp := &argov1alpha1.Application{}
	sourceKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", platform.Name, from),
		Namespace: "argocd",
	}
	if err := m.client.Get(ctx, sourceKey, sourceApp); err != nil {
		return fmt.Errorf("getting source application: %w", err)
	}

	// Get target application
	targetApp := &argov1alpha1.Application{}
	targetKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", platform.Name, to),
		Namespace: "argocd",
	}
	if err := m.client.Get(ctx, targetKey, targetApp); err != nil {
		return fmt.Errorf("getting target application: %w", err)
	}

	// Update target to use source revision
	targetApp.Spec.Source.TargetRevision = sourceApp.Status.Sync.Revision
	
	// Add promotion annotations
	if targetApp.Annotations == nil {
		targetApp.Annotations = make(map[string]string)
	}
	targetApp.Annotations["observability.io/promoted-from"] = from
	targetApp.Annotations["observability.io/promoted-at"] = time.Now().Format(time.RFC3339)
	targetApp.Annotations["observability.io/promoted-revision"] = sourceApp.Status.Sync.Revision

	// Update target application
	if err := m.client.Update(ctx, targetApp); err != nil {
		return fmt.Errorf("updating target application: %w", err)
	}

	// Trigger sync
	targetApp.Operation = &argov1alpha1.Operation{
		Sync: &argov1alpha1.SyncOperation{
			Revision: sourceApp.Status.Sync.Revision,
			Prune:    true,
		},
	}
	
	return m.client.Update(ctx, targetApp)
}

// Rollback rolls back to a previous configuration
func (m *ArgoCDManager) Rollback(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("revision", revision)
	log.Info("Rolling back to revision")

	// Get all applications for this platform
	appList := &argov1alpha1.ApplicationList{}
	if err := m.client.List(ctx, appList, client.InNamespace("argocd"), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return fmt.Errorf("listing ArgoCD applications: %w", err)
	}

	// Rollback each application
	for _, app := range appList.Items {
		log.Info("Rolling back application", "application", app.Name)
		
		// Find the history entry for the revision
		var historyID int64 = -1
		for _, h := range app.Status.History {
			if h.Revision == revision {
				historyID = h.ID
				break
			}
		}
		
		if historyID == -1 {
			log.Info("Revision not found in history, syncing to revision directly", "application", app.Name)
			app.Operation = &argov1alpha1.Operation{
				Sync: &argov1alpha1.SyncOperation{
					Revision: revision,
					Prune:    true,
				},
			}
		} else {
			// Use history rollback
			app.Spec.Source.TargetRevision = revision
		}
		
		if err := m.client.Update(ctx, &app); err != nil {
			return fmt.Errorf("rolling back application %s: %w", app.Name, err)
		}
	}

	return nil
}

// GetSyncStatus returns the current GitOps sync status
func (m *ArgoCDManager) GetSyncStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*GitOpsSyncStatus, error) {
	status := &GitOpsSyncStatus{
		Provider:     ArgoCD,
		Environments: make(map[string]EnvironmentStatus),
	}

	// Get all applications for this platform
	appList := &argov1alpha1.ApplicationList{}
	if err := m.client.List(ctx, appList, client.InNamespace("argocd"), client.MatchingLabels{
		"observability.io/platform": platform.Name,
	}); err != nil {
		return nil, fmt.Errorf("listing ArgoCD applications: %w", err)
	}

	// Aggregate status from all applications
	overallState := SyncStateSynced
	var latestSyncTime *metav1.Time

	for _, app := range appList.Items {
		envName := app.Labels["observability.io/environment"]
		
		envStatus := EnvironmentStatus{
			Name:      envName,
			Namespace: app.Spec.Destination.Namespace,
			Revision:  app.Status.Sync.Revision,
			Status:    string(app.Status.Sync.Status),
			Health:    string(app.Status.Health.Status),
		}
		
		// Check promotion info
		if promotedFrom, ok := app.Annotations["observability.io/promoted-from"]; ok {
			envStatus.PromotedFrom = promotedFrom
			envStatus.PromotedAt = app.Annotations["observability.io/promoted-at"]
		}
		
		status.Environments[envName] = envStatus
		
		// Update overall state
		switch app.Status.Sync.Status {
		case argov1alpha1.SyncStatusCodeOutOfSync:
			overallState = SyncStateOutOfSync
		case argov1alpha1.SyncStatusCodeUnknown:
			if overallState == SyncStateSynced {
				overallState = SyncStateUnknown
			}
		}
		
		// Track latest sync time
		if app.Status.OperationState != nil && app.Status.OperationState.FinishedAt != nil {
			if latestSyncTime == nil || app.Status.OperationState.FinishedAt.After(latestSyncTime.Time) {
				latestSyncTime = app.Status.OperationState.FinishedAt
			}
		}
	}

	status.SyncState = overallState
	if latestSyncTime != nil {
		status.LastSyncTime = latestSyncTime.Format(time.RFC3339)
	}

	// Get the latest revision across all apps
	if len(appList.Items) > 0 {
		status.Revision = appList.Items[0].Status.Sync.Revision
	}

	return status, nil
}

// ValidateGitOpsConfig validates GitOps configuration
func (m *ArgoCDManager) ValidateGitOpsConfig(platform *observabilityv1beta1.ObservabilityPlatform) error {
	if platform.Spec.GitOps == nil {
		return nil
	}

	gitOps := platform.Spec.GitOps

	// Validate repository
	if gitOps.Repository.URL == "" {
		return fmt.Errorf("repository URL is required")
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

	// Validate promotion policies
	if gitOps.AutoPromotion != nil && gitOps.AutoPromotion.Enabled {
		if gitOps.AutoPromotion.Strategy != "sequential" && gitOps.AutoPromotion.Strategy != "parallel" {
			return fmt.Errorf("invalid promotion strategy: %s", gitOps.AutoPromotion.Strategy)
		}
	}

	return nil
}

// Helper methods

func (m *ArgoCDManager) getPathForEnvironment(repo observabilityv1beta1.GitRepository, env observabilityv1beta1.GitOpsEnvironment) string {
	if env.Path != "" {
		return env.Path
	}
	if repo.Path != "" {
		return fmt.Sprintf("%s/%s", repo.Path, env.Name)
	}
	return env.Name
}

func (m *ArgoCDManager) getBranchForEnvironment(repo observabilityv1beta1.GitRepository, env observabilityv1beta1.GitOpsEnvironment) string {
	if env.Branch != "" {
		return env.Branch
	}
	return repo.Branch
}

func (m *ArgoCDManager) buildSyncPolicy(platform *observabilityv1beta1.ObservabilityPlatform, env observabilityv1beta1.GitOpsEnvironment) *argov1alpha1.SyncPolicy {
	policy := &argov1alpha1.SyncPolicy{}
	
	if platform.Spec.GitOps.AutoSync {
		policy.Automated = &argov1alpha1.SyncPolicyAutomated{
			Prune:    true,
			SelfHeal: true,
		}
	}
	
	// Add retry policy
	policy.Retry = &argov1alpha1.RetryStrategy{
		Limit: 5,
		Backoff: &argov1alpha1.Backoff{
			Duration:    "5s",
			Factor:      func(i int64) *int64 { return &i }(2),
			MaxDuration: "3m",
		},
	}
	
	return policy
}

func (m *ArgoCDManager) buildDestinations(platform *observabilityv1beta1.ObservabilityPlatform) []argov1alpha1.ApplicationDestination {
	var destinations []argov1alpha1.ApplicationDestination
	
	for _, env := range platform.Spec.GitOps.Environments {
		destinations = append(destinations, argov1alpha1.ApplicationDestination{
			Server:    "https://kubernetes.default.svc",
			Namespace: env.Namespace,
		})
	}
	
	return destinations
}
