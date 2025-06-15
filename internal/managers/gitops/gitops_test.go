package gitops

import (
	"context"
	"fmt"
	"testing"
	"time"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestArgoCDManager_SetupGitOps(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = argov1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name           string
		platform       *observabilityv1beta1.ObservabilityPlatform
		existingObjs   []client.Object
		wantErr        bool
		expectedApps   int
		expectedProj   bool
	}{
		{
			name: "create new ArgoCD applications",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-ns",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Provider: "argocd",
						Repository: observabilityv1beta1.GitRepository{
							URL:    "https://github.com/test/repo.git",
							Branch: "main",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{
							{
								Name:      "dev",
								Namespace: "platform-dev",
							},
							{
								Name:      "prod",
								Namespace: "platform-prod",
							},
						},
					},
				},
			},
			wantErr:      false,
			expectedApps: 2,
			expectedProj: true,
		},
		{
			name: "update existing ArgoCD applications",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-ns",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Provider: "argocd",
						Repository: observabilityv1beta1.GitRepository{
							URL:    "https://github.com/test/repo.git",
							Branch: "main",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{
							{
								Name:      "dev",
								Namespace: "platform-dev",
							},
						},
					},
				},
			},
			existingObjs: []client.Object{
				&argov1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-platform-dev",
						Namespace: "argocd",
						Labels: map[string]string{
							"observability.io/platform": "test-platform",
						},
					},
				},
			},
			wantErr:      false,
			expectedApps: 1,
			expectedProj: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjs...).
				Build()

			manager, err := NewArgoCDManager(client, scheme)
			if err != nil {
				t.Fatalf("Failed to create ArgoCD manager: %v", err)
			}

			err = manager.SetupGitOps(ctx, tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetupGitOps() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify applications created
			appList := &argov1alpha1.ApplicationList{}
			if err := client.List(ctx, appList); err != nil {
				t.Fatalf("Failed to list applications: %v", err)
			}

			if len(appList.Items) != tt.expectedApps {
				t.Errorf("Expected %d applications, got %d", tt.expectedApps, len(appList.Items))
			}

			// Verify project created
			if tt.expectedProj {
				project := &argov1alpha1.AppProject{}
				projKey := types.NamespacedName{
					Name:      fmt.Sprintf("%s-project", tt.platform.Name),
					Namespace: "argocd",
				}
				if err := client.Get(ctx, projKey, project); err != nil {
					t.Errorf("Expected project to be created: %v", err)
				}
			}
		})
	}
}

func TestFluxManager_SetupGitOps(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = sourcev1.AddToScheme(scheme)
	_ = kustomizev1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name                string
		platform            *observabilityv1beta1.ObservabilityPlatform
		existingObjs        []client.Object
		wantErr             bool
		expectedGitRepo     bool
		expectedKustomizations int
	}{
		{
			name: "create Flux resources",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-ns",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Provider: "flux",
						Repository: observabilityv1beta1.GitRepository{
							URL:      "https://github.com/test/repo.git",
							Branch:   "main",
							Interval: "5m",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{
							{
								Name:      "dev",
								Namespace: "platform-dev",
								Path:      "environments/dev",
							},
							{
								Name:      "staging",
								Namespace: "platform-staging",
								Path:      "environments/staging",
							},
						},
					},
				},
			},
			wantErr:                false,
			expectedGitRepo:        true,
			expectedKustomizations: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existingObjs...).
				Build()

			manager, err := NewFluxManager(client, scheme)
			if err != nil {
				t.Fatalf("Failed to create Flux manager: %v", err)
			}

			err = manager.SetupGitOps(ctx, tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetupGitOps() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify GitRepository created
			if tt.expectedGitRepo {
				gitRepo := &sourcev1.GitRepository{}
				gitRepoKey := types.NamespacedName{
					Name:      fmt.Sprintf("%s-source", tt.platform.Name),
					Namespace: tt.platform.Namespace,
				}
				if err := client.Get(ctx, gitRepoKey, gitRepo); err != nil {
					t.Errorf("Expected GitRepository to be created: %v", err)
				}
			}

			// Verify Kustomizations created
			kustomizationList := &kustomizev1.KustomizationList{}
			if err := client.List(ctx, kustomizationList); err != nil {
				t.Fatalf("Failed to list Kustomizations: %v", err)
			}

			if len(kustomizationList.Items) != tt.expectedKustomizations {
				t.Errorf("Expected %d Kustomizations, got %d", tt.expectedKustomizations, len(kustomizationList.Items))
			}
		})
	}
}

func TestGitOpsManager_ValidateGitOpsConfig(t *testing.T) {
	tests := []struct {
		name     string
		platform *observabilityv1beta1.ObservabilityPlatform
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid configuration",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Repository: observabilityv1beta1.GitRepository{
							URL:    "https://github.com/test/repo.git",
							Branch: "main",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{
							{
								Name:      "dev",
								Namespace: "dev-ns",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing repository URL",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Repository: observabilityv1beta1.GitRepository{
							Branch: "main",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{
							{
								Name:      "dev",
								Namespace: "dev-ns",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "repository URL is required",
		},
		{
			name: "no environments",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Repository: observabilityv1beta1.GitRepository{
							URL:    "https://github.com/test/repo.git",
							Branch: "main",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one environment must be configured",
		},
		{
			name: "duplicate environment names",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Repository: observabilityv1beta1.GitRepository{
							URL:    "https://github.com/test/repo.git",
							Branch: "main",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{
							{
								Name:      "dev",
								Namespace: "dev-ns",
							},
							{
								Name:      "dev",
								Namespace: "dev-ns-2",
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate environment name: dev",
		},
		{
			name: "invalid promotion strategy",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						Repository: observabilityv1beta1.GitRepository{
							URL:    "https://github.com/test/repo.git",
							Branch: "main",
						},
						Environments: []observabilityv1beta1.GitOpsEnvironment{
							{
								Name:      "dev",
								Namespace: "dev-ns",
							},
						},
						AutoPromotion: &observabilityv1beta1.AutoPromotionConfig{
							Enabled:  true,
							Strategy: "invalid",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid promotion strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = observabilityv1beta1.AddToScheme(scheme)
			
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			manager, _ := NewArgoCDManager(client, scheme)
			err := manager.ValidateGitOpsConfig(tt.platform)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGitOpsConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Expected error message %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestRollbackManager_CheckAndRollback(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = observabilityv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	tests := []struct {
		name         string
		platform     *observabilityv1beta1.ObservabilityPlatform
		events       []corev1.Event
		configMap    *corev1.ConfigMap
		shouldRollback bool
	}{
		{
			name: "rollback on failure threshold",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-ns",
					UID:       "test-uid",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						RollbackConfig: &observabilityv1beta1.RollbackConfig{
							AutoRollback:     true,
							FailureThreshold: 3,
							Window:           "30m",
							MaxHistory:       10,
						},
					},
				},
				Status: observabilityv1beta1.ObservabilityPlatformStatus{
					Health: observabilityv1beta1.HealthStatus{
						Healthy: false,
					},
				},
			},
			events: []corev1.Event{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "event1",
						Namespace: "test-ns",
					},
					InvolvedObject: corev1.ObjectReference{
						Name: "test-platform",
					},
					Type:          corev1.EventTypeWarning,
					Reason:        "Failed",
					LastTimestamp: metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "event2",
						Namespace: "test-ns",
					},
					InvolvedObject: corev1.ObjectReference{
						Name: "test-platform",
					},
					Type:          corev1.EventTypeWarning,
					Reason:        "CrashLoopBackOff",
					LastTimestamp: metav1.Time{Time: time.Now().Add(-3 * time.Minute)},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "event3",
						Namespace: "test-ns",
					},
					InvolvedObject: corev1.ObjectReference{
						Name: "test-platform",
					},
					Type:          corev1.EventTypeWarning,
					Reason:        "BackOff",
					LastTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
				},
			},
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform-gitops-history",
					Namespace: "test-ns",
				},
				Data: map[string]string{
					"lastKnownGood": "abc123",
				},
			},
			shouldRollback: true,
		},
		{
			name: "no rollback when disabled",
			platform: &observabilityv1beta1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "test-ns",
				},
				Spec: observabilityv1beta1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1beta1.GitOpsConfig{
						RollbackConfig: &observabilityv1beta1.RollbackConfig{
							AutoRollback: false,
						},
					},
				},
			},
			shouldRollback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			objs := []client.Object{}
			if tt.configMap != nil {
				objs = append(objs, tt.configMap)
			}
			for i := range tt.events {
				objs = append(objs, &tt.events[i])
			}
			
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			gitOpsManager, _ := NewArgoCDManager(client, scheme)
			rollbackManager := NewRollbackManager(client, scheme, gitOpsManager)

			// Mock the GitOps manager rollback
			mockGitOpsManager := &mockGitOpsManager{
				rollbackCalled: false,
			}
			rollbackManager.gitOpsManager = mockGitOpsManager

			err := rollbackManager.CheckAndRollback(ctx, tt.platform)
			if err != nil {
				t.Errorf("CheckAndRollback() unexpected error: %v", err)
			}

			if mockGitOpsManager.rollbackCalled != tt.shouldRollback {
				t.Errorf("Expected rollback called = %v, got %v", tt.shouldRollback, mockGitOpsManager.rollbackCalled)
			}
		})
	}
}

// Mock GitOps manager for testing
type mockGitOpsManager struct {
	rollbackCalled bool
	syncStatus     *GitOpsSyncStatus
}

func (m *mockGitOpsManager) SetupGitOps(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return nil
}

func (m *mockGitOpsManager) SyncWithGit(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) error {
	return nil
}

func (m *mockGitOpsManager) PromoteEnvironment(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, from, to string) error {
	return nil
}

func (m *mockGitOpsManager) Rollback(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform, revision string) error {
	m.rollbackCalled = true
	return nil
}

func (m *mockGitOpsManager) GetSyncStatus(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) (*GitOpsSyncStatus, error) {
	if m.syncStatus != nil {
		return m.syncStatus, nil
	}
	return &GitOpsSyncStatus{
		Provider:  ArgoCD,
		SyncState: SyncStateSynced,
		Revision:  "test-revision",
	}, nil
}

func (m *mockGitOpsManager) ValidateGitOpsConfig(platform *observabilityv1beta1.ObservabilityPlatform) error {
	return nil
}
