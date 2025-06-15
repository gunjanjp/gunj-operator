package gitops_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/config"
	"github.com/gunjanjp/gunj-operator/internal/gitops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Mock implementations for testing

type mockGitOpsController struct {
	mock.Mock
}

func (m *mockGitOpsController) Reconcile(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	args := m.Called(ctx, platform)
	return args.Error(0)
}

func (m *mockGitOpsController) Sync(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) error {
	args := m.Called(ctx, platform)
	return args.Error(0)
}

func (m *mockGitOpsController) GetStatus(ctx context.Context, platform *observabilityv1.ObservabilityPlatform) (*gitops.GitOpsStatus, error) {
	args := m.Called(ctx, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gitops.GitOpsStatus), args.Error(1)
}

func (m *mockGitOpsController) Rollback(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, revision string) error {
	args := m.Called(ctx, platform, revision)
	return args.Error(0)
}

func (m *mockGitOpsController) Promote(ctx context.Context, platform *observabilityv1.ObservabilityPlatform, targetEnv string) error {
	args := m.Called(ctx, platform, targetEnv)
	return args.Error(0)
}

type mockGitSynchronizer struct {
	mock.Mock
}

func (m *mockGitSynchronizer) Clone(ctx context.Context, repo gitops.GitRepository) (string, error) {
	args := m.Called(ctx, repo)
	return args.String(0), args.Error(1)
}

func (m *mockGitSynchronizer) Pull(ctx context.Context, repoPath string) error {
	args := m.Called(ctx, repoPath)
	return args.Error(0)
}

func (m *mockGitSynchronizer) GetRevision(ctx context.Context, repoPath string) (string, error) {
	args := m.Called(ctx, repoPath)
	return args.String(0), args.Error(1)
}

func (m *mockGitSynchronizer) GetFiles(ctx context.Context, repoPath string, pattern string) ([]string, error) {
	args := m.Called(ctx, repoPath, pattern)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockGitSynchronizer) Cleanup(ctx context.Context, repoPath string) error {
	args := m.Called(ctx, repoPath)
	return args.Error(0)
}

// Test suite

func TestGitOpsManager_Reconcile(t *testing.T) {
	// Setup scheme
	s := scheme.Scheme
	_ = observabilityv1.AddToScheme(s)

	tests := []struct {
		name           string
		platform       *observabilityv1.ObservabilityPlatform
		expectError    bool
		expectedCalls  func(*mockGitOpsController)
		additionalObjs []runtime.Object
	}{
		{
			name: "GitOps disabled",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					GitOps: nil,
				},
			},
			expectError:   false,
			expectedCalls: func(m *mockGitOpsController) {},
		},
		{
			name: "ArgoCD GitOps enabled",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1.GitOpsSpec{
						Enabled:  true,
						Provider: "argocd",
						Repository: observabilityv1.GitRepositorySpec{
							URL:    "https://github.com/example/config.git",
							Branch: "main",
							Path:   "environments/prod",
						},
						SyncPolicy: observabilityv1.SyncPolicySpec{
							Automated: true,
							Prune:     true,
							SelfHeal:  true,
						},
					},
				},
			},
			expectError: false,
			expectedCalls: func(m *mockGitOpsController) {
				m.On("Reconcile", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "Flux GitOps with drift detection",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1.GitOpsSpec{
						Enabled:  true,
						Provider: "flux",
						Repository: observabilityv1.GitRepositorySpec{
							URL:      "git@github.com:example/config.git",
							Branch:   "main",
							Path:     "clusters/production",
							Interval: metav1.Duration{Duration: 5 * time.Minute},
						},
						SyncPolicy: observabilityv1.SyncPolicySpec{
							Automated: true,
							Prune:     true,
							SelfHeal:  true,
						},
						DriftDetection: &observabilityv1.DriftDetectionSpec{
							Enabled:  true,
							Interval: metav1.Duration{Duration: 10 * time.Minute},
							Action:   "remediate",
						},
					},
				},
			},
			expectError: false,
			expectedCalls: func(m *mockGitOpsController) {
				m.On("Reconcile", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "Platform being deleted",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-platform",
					Namespace:         "default",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{gitops.GitOpsFinalizer},
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1.GitOpsSpec{
						Enabled:  true,
						Provider: "argocd",
					},
				},
			},
			expectError:   false,
			expectedCalls: func(m *mockGitOpsController) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			objs := append(tt.additionalObjs, tt.platform)
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithRuntimeObjects(objs...).
				WithStatusSubresource(&observabilityv1.ObservabilityPlatform{}).
				Build()

			// Create mock controllers
			mockArgoCD := &mockGitOpsController{}
			mockFlux := &mockGitOpsController{}

			// Setup expected calls
			if tt.expectedCalls != nil {
				if tt.platform.Spec.GitOps != nil {
					if tt.platform.Spec.GitOps.Provider == "argocd" {
						tt.expectedCalls(mockArgoCD)
					} else if tt.platform.Spec.GitOps.Provider == "flux" {
						tt.expectedCalls(mockFlux)
					}
				}
			}

			// Create manager with mocks
			manager := createTestManager(t, fakeClient, s, mockArgoCD, mockFlux)

			// Test reconcile
			err := manager.Reconcile(context.Background(), tt.platform)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockArgoCD.AssertExpectations(t)
			mockFlux.AssertExpectations(t)
		})
	}
}

func TestGitOpsManager_Sync(t *testing.T) {
	// Setup scheme
	s := scheme.Scheme
	_ = observabilityv1.AddToScheme(s)

	tests := []struct {
		name          string
		platform      *observabilityv1.ObservabilityPlatform
		expectError   bool
		expectedCalls func(*mockGitOpsController, *mockGitSynchronizer)
	}{
		{
			name: "Successful sync",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1.GitOpsSpec{
						Enabled:  true,
						Provider: "argocd",
						Repository: observabilityv1.GitRepositorySpec{
							URL:    "https://github.com/example/config.git",
							Branch: "main",
						},
					},
				},
			},
			expectError: false,
			expectedCalls: func(controller *mockGitOpsController, sync *mockGitSynchronizer) {
				controller.On("Sync", mock.Anything, mock.Anything).Return(nil)
				sync.On("GetRevision", mock.Anything, mock.Anything).Return("abc123", nil)
			},
		},
		{
			name: "Sync with rollback trigger",
			platform: &observabilityv1.ObservabilityPlatform{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-platform",
					Namespace: "default",
				},
				Spec: observabilityv1.ObservabilityPlatformSpec{
					GitOps: &observabilityv1.GitOpsSpec{
						Enabled:  true,
						Provider: "flux",
						Repository: observabilityv1.GitRepositorySpec{
							URL:    "https://github.com/example/config.git",
							Branch: "main",
						},
						Rollback: &observabilityv1.RollbackSpec{
							Enabled:    true,
							MaxHistory: 10,
						},
					},
				},
			},
			expectError: true,
			expectedCalls: func(controller *mockGitOpsController, sync *mockGitSynchronizer) {
				controller.On("Sync", mock.Anything, mock.Anything).Return(assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithRuntimeObjects(tt.platform).
				WithStatusSubresource(&observabilityv1.ObservabilityPlatform{}).
				Build()

			// Create mocks
			mockArgoCD := &mockGitOpsController{}
			mockFlux := &mockGitOpsController{}
			mockSync := &mockGitSynchronizer{}

			// Setup expected calls
			if tt.expectedCalls != nil {
				if tt.platform.Spec.GitOps.Provider == "argocd" {
					tt.expectedCalls(mockArgoCD, mockSync)
				} else if tt.platform.Spec.GitOps.Provider == "flux" {
					tt.expectedCalls(mockFlux, mockSync)
				}
			}

			// Create manager
			manager := createTestManager(t, fakeClient, s, mockArgoCD, mockFlux)
			manager.synchronizer = mockSync

			// Test sync
			err := manager.Sync(context.Background(), tt.platform)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mocks
			mockArgoCD.AssertExpectations(t)
			mockFlux.AssertExpectations(t)
			mockSync.AssertExpectations(t)
		})
	}
}

func TestGitOpsManager_Promote(t *testing.T) {
	// Setup scheme
	s := scheme.Scheme
	_ = observabilityv1.AddToScheme(s)

	platform := &observabilityv1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-platform",
			Namespace: "monitoring-dev",
			Annotations: map[string]string{
				"promotion.observability.io/environment": "dev",
			},
		},
		Spec: observabilityv1.ObservabilityPlatformSpec{
			GitOps: &observabilityv1.GitOpsSpec{
				Enabled:  true,
				Provider: "argocd",
				Promotion: &observabilityv1.PromotionSpec{
					Strategy: "progressive",
					Environments: []observabilityv1.EnvironmentSpec{
						{
							Name:        "dev",
							Namespace:   "monitoring-dev",
							Branch:      "dev",
							AutoPromote: true,
						},
						{
							Name:        "staging",
							Namespace:   "monitoring-staging",
							Branch:      "staging",
							AutoPromote: true,
						},
						{
							Name:        "production",
							Namespace:   "monitoring-prod",
							Branch:      "main",
							AutoPromote: false,
						},
					},
				},
			},
		},
		Status: observabilityv1.ObservabilityPlatformStatus{
			Phase: observabilityv1.PhaseReady,
			Components: map[string]observabilityv1.ComponentStatus{
				"prometheus": {
					Status:            observabilityv1.ComponentStatusReady,
					Replicas:          3,
					AvailableReplicas: 3,
				},
			},
		},
	}

	tests := []struct {
		name        string
		targetEnv   string
		expectError bool
		setupMocks  func(*testing.T, client.Client)
	}{
		{
			name:        "Promote to staging",
			targetEnv:   "staging",
			expectError: false,
		},
		{
			name:        "Invalid promotion - skip environment",
			targetEnv:   "production",
			expectError: true,
		},
		{
			name:        "Unknown environment",
			targetEnv:   "unknown",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithRuntimeObjects(platform.DeepCopy()).
				WithStatusSubresource(&observabilityv1.ObservabilityPlatform{}).
				Build()

			// Setup additional mocks if needed
			if tt.setupMocks != nil {
				tt.setupMocks(t, fakeClient)
			}

			// Create manager
			manager := createTestManager(t, fakeClient, s, nil, nil)

			// Test promote
			err := manager.Promote(context.Background(), platform, tt.targetEnv)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to create test manager
func createTestManager(t *testing.T, client client.Client, scheme *runtime.Scheme, argoCD, flux gitops.GitOpsController) *gitops.Manager {
	// Create config
	cfg := &config.Config{
		Namespace: "gunj-system",
	}

	// Create recorder
	recorder := record.NewFakeRecorder(100)

	// Create manager
	manager, err := gitops.NewManager(client, scheme, recorder, cfg)
	assert.NoError(t, err)

	// Replace controllers with mocks if provided
	if argoCD != nil {
		manager.argoCDController = argoCD
	}
	if flux != nil {
		manager.fluxController = flux
	}

	return manager
}
