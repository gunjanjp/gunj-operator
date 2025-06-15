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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/gitops/git"
)

var _ = Describe("Promotion Manager", func() {
	var (
		manager    *Manager
		ctx        context.Context
		platform   *observabilityv1.ObservabilityPlatform
		gitOps     *gitopsv1beta1.GitOpsIntegrationSpec
		testScheme *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		// Setup scheme
		testScheme = scheme.Scheme
		Expect(observabilityv1.AddToScheme(testScheme)).To(Succeed())
		
		// Create fake client
		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		
		// Create mock git manager
		mockGitManager := &MockGitManager{}
		
		// Create manager
		manager = NewManager(fakeClient, ctrl.Log.WithName("test"), mockGitManager)
		
		// Create test platform
		platform = &observabilityv1.ObservabilityPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-platform",
				Namespace: "default",
			},
		}
		
		// Create GitOps spec with environments
		gitOps = &gitopsv1beta1.GitOpsIntegrationSpec{
			Repository: gitopsv1beta1.GitRepositorySpec{
				URL:    "https://github.com/test/repo",
				Branch: "main",
				Path:   "/",
			},
			Environments: []gitopsv1beta1.EnvironmentSpec{
				{
					Name:     "dev",
					Branch:   "develop",
					Path:     "/dev",
					AutoSync: true,
				},
				{
					Name:     "staging",
					Branch:   "staging",
					Path:     "/staging",
					AutoSync: true,
				},
				{
					Name:     "production",
					Branch:   "main",
					Path:     "/production",
					AutoSync: false,
				},
			},
			Promotion: &gitopsv1beta1.PromotionSpec{
				Strategy:      "Progressive",
				AutoPromotion: true,
				PromotionPolicy: []gitopsv1beta1.PromotionPolicy{
					{
						From:              "dev",
						To:                "staging",
						RequiredApprovals: 0,
						AutoPromoteAfter:  "1h",
					},
					{
						From:              "staging",
						To:                "production",
						RequiredApprovals: 2,
						AutoPromoteAfter:  "24h",
					},
				},
			},
		}
	})

	Describe("Environment Promotion", func() {
		Context("with valid environments", func() {
			It("should promote from dev to staging", func() {
				// Create mock git manager
				mockGit := &MockGitManager{}
				manager.GitManager = mockGit
				
				// Setup expectations
				mockGit.On("CloneRepository", mock.Anything, gitOps.Repository).
					Return(&git.Repository{}, "/tmp/repo", nil)
				mockGit.On("GetCurrentRevision", mock.Anything).
					Return("abc123", nil)
				mockGit.On("GetFilesAtPath", mock.Anything, "/dev").
					Return(map[string][]byte{
						"deployment.yaml": []byte("test-content"),
					}, nil)
				mockGit.On("WriteFiles", mock.Anything, "/staging", mock.Anything).
					Return(nil)
				mockGit.On("CommitAndPush", mock.Anything, mock.Anything, gitOps.Repository, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGit.On("Cleanup").Return(nil)
				
				// Create promotion request
				req := &PromotionRequest{
					Platform:      platform,
					GitOps:        gitOps,
					FromEnv:       "dev",
					ToEnv:         "staging",
					User:          "test-user",
					AutoPromotion: true,
				}
				
				// Promote
				result, err := manager.Promote(ctx, req)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Success).To(BeTrue())
				Expect(result.FromRevision).To(Equal("abc123"))
				Expect(result.Message).To(ContainSubstring("Successfully promoted"))
				
				// Verify expectations
				mockGit.AssertExpectations(GinkgoT())
			})
		})

		Context("with invalid environments", func() {
			It("should fail for non-existent source environment", func() {
				req := &PromotionRequest{
					Platform: platform,
					GitOps:   gitOps,
					FromEnv:  "non-existent",
					ToEnv:    "staging",
					User:     "test-user",
				}
				
				result, err := manager.Promote(ctx, req)
				
				Expect(err).To(HaveOccurred())
				Expect(result.Success).To(BeFalse())
				Expect(result.Message).To(ContainSubstring("source environment non-existent not found"))
			})

			It("should fail for non-existent target environment", func() {
				req := &PromotionRequest{
					Platform: platform,
					GitOps:   gitOps,
					FromEnv:  "dev",
					ToEnv:    "non-existent",
					User:     "test-user",
				}
				
				result, err := manager.Promote(ctx, req)
				
				Expect(err).To(HaveOccurred())
				Expect(result.Success).To(BeFalse())
				Expect(result.Message).To(ContainSubstring("target environment non-existent not found"))
			})
		})

		Context("with promotion gates", func() {
			BeforeEach(func() {
				// Add promotion gates
				gitOps.Environments[1].PromotionGates = []gitopsv1beta1.PromotionGate{
					{
						Type: "Manual",
						Config: map[string]string{
							"approvers": "2",
						},
					},
					{
						Type: "Time",
						Config: map[string]string{
							"minAge": "1h",
						},
					},
				}
			})

			It("should check manual approval gates", func() {
				// Create mock approval store
				mockApprovals := &MockApprovalStore{}
				manager.ApprovalStore = mockApprovals
				
				// No approvals yet
				mockApprovals.On("GetApprovals", mock.Anything, mock.Anything).
					Return([]Approval{}, nil)
				
				req := &PromotionRequest{
					Platform:      platform,
					GitOps:        gitOps,
					FromEnv:       "dev",
					ToEnv:         "staging",
					User:          "test-user",
					AutoPromotion: false,
				}
				
				result, err := manager.Promote(ctx, req)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Success).To(BeFalse())
				Expect(result.Message).To(Equal("One or more promotion gates failed"))
				Expect(result.GatesChecked).To(HaveLen(2))
				Expect(result.GatesChecked[0].GateType).To(Equal("Manual"))
				Expect(result.GatesChecked[0].Passed).To(BeFalse())
			})

			It("should pass gates with sufficient approvals", func() {
				// Create mock approval store
				mockApprovals := &MockApprovalStore{}
				manager.ApprovalStore = mockApprovals
				
				// Return sufficient approvals
				mockApprovals.On("GetApprovals", mock.Anything, mock.Anything).
					Return([]Approval{
						{User: "user1", Timestamp: time.Now()},
						{User: "user2", Timestamp: time.Now()},
					}, nil)
				
				// Create mock git manager
				mockGit := &MockGitManager{}
				manager.GitManager = mockGit
				
				// Setup Git expectations
				mockGit.On("CloneRepository", mock.Anything, gitOps.Repository).
					Return(&git.Repository{}, "/tmp/repo", nil)
				mockGit.On("GetCurrentRevision", mock.Anything).
					Return("abc123", nil)
				mockGit.On("GetFilesAtPath", mock.Anything, "/dev").
					Return(map[string][]byte{}, nil)
				mockGit.On("WriteFiles", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGit.On("CommitAndPush", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGit.On("Cleanup").Return(nil)
				
				req := &PromotionRequest{
					Platform: platform,
					GitOps:   gitOps,
					FromEnv:  "dev",
					ToEnv:    "staging",
					User:     "test-user",
				}
				
				result, err := manager.Promote(ctx, req)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Success).To(BeTrue())
				Expect(result.GatesChecked[0].Passed).To(BeTrue())
			})
		})

		Context("with metrics recording", func() {
			It("should record promotion metrics", func() {
				// Create mock metrics recorder
				mockMetrics := &MockPromotionMetrics{}
				manager.MetricsRecorder = mockMetrics
				
				// Expect metrics calls
				mockMetrics.On("RecordPromotionStarted", "dev", "staging").Once()
				mockMetrics.On("RecordPromotionCompleted", "dev", "staging", true, mock.Anything).Once()
				
				// Create mock git manager
				mockGit := &MockGitManager{}
				manager.GitManager = mockGit
				
				// Setup Git expectations
				mockGit.On("CloneRepository", mock.Anything, gitOps.Repository).
					Return(&git.Repository{}, "/tmp/repo", nil)
				mockGit.On("GetCurrentRevision", mock.Anything).
					Return("abc123", nil)
				mockGit.On("GetFilesAtPath", mock.Anything, "/dev").
					Return(map[string][]byte{}, nil)
				mockGit.On("WriteFiles", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGit.On("CommitAndPush", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				mockGit.On("Cleanup").Return(nil)
				
				req := &PromotionRequest{
					Platform:      platform,
					GitOps:        gitOps,
					FromEnv:       "dev",
					ToEnv:         "staging",
					User:          "test-user",
					AutoPromotion: true,
				}
				
				_, err := manager.Promote(ctx, req)
				
				Expect(err).NotTo(HaveOccurred())
				mockMetrics.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("File Processing", func() {
		It("should process files for target environment", func() {
			content := []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  namespace: dev
  labels:
    environment: dev
spec:
  template:
    spec:
      containers:
      - name: app
        image: app-dev.example.com/app:latest`)
			
			processed := manager.processFileForEnvironment(content, "dev", "production")
			processedStr := string(processed)
			
			// Check namespace was updated
			Expect(processedStr).To(ContainSubstring("namespace: production"))
			Expect(processedStr).NotTo(ContainSubstring("namespace: dev"))
			
			// Check environment label was updated
			Expect(processedStr).To(ContainSubstring("environment: production"))
			Expect(processedStr).NotTo(ContainSubstring("environment: dev"))
			
			// Check ingress host was updated
			Expect(processedStr).To(ContainSubstring("app-production.example.com"))
			Expect(processedStr).NotTo(ContainSubstring("app-dev.example.com"))
		})
	})

	Describe("Promotion Policy", func() {
		It("should find correct promotion policy", func() {
			policy := manager.findPromotionPolicy(gitOps, "dev", "staging")
			
			Expect(policy).NotTo(BeNil())
			Expect(policy.From).To(Equal("dev"))
			Expect(policy.To).To(Equal("staging"))
			Expect(policy.RequiredApprovals).To(Equal(0))
			Expect(policy.AutoPromoteAfter).To(Equal("1h"))
		})

		It("should return nil for non-existent policy", func() {
			policy := manager.findPromotionPolicy(gitOps, "dev", "production")
			
			Expect(policy).To(BeNil())
		})
	})

	Describe("Promotion Eligibility", func() {
		It("should check if promotion is allowed", func() {
			canPromote, reason := manager.CanPromote(ctx, platform, gitOps, "dev", "staging")
			
			Expect(canPromote).To(BeTrue())
			Expect(reason).To(BeEmpty())
		})

		It("should deny promotion without defined path", func() {
			// Set strategy to manual
			gitOps.Promotion.Strategy = "Manual"
			
			canPromote, reason := manager.CanPromote(ctx, platform, gitOps, "dev", "production")
			
			Expect(canPromote).To(BeFalse())
			Expect(reason).To(Equal("No promotion path defined from dev to production"))
		})
	})

	Describe("Promotion History", func() {
		It("should get promotion history", func() {
			// Initially no history
			history, err := manager.GetPromotionHistory(ctx, platform, "dev", "staging")
			
			Expect(err).NotTo(HaveOccurred())
			Expect(history).To(BeEmpty())
		})
	})
})

// Mock types

type MockGitManager struct {
	mock.Mock
}

func (m *MockGitManager) CloneRepository(ctx context.Context, repoSpec gitopsv1beta1.GitRepositorySpec) (*git.Repository, string, error) {
	args := m.Called(ctx, repoSpec)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*git.Repository), args.String(1), args.Error(2)
}

func (m *MockGitManager) GetCurrentRevision(repo *git.Repository) (string, error) {
	args := m.Called(repo)
	return args.String(0), args.Error(1)
}

func (m *MockGitManager) GetFilesAtPath(repo *git.Repository, path string) (map[string][]byte, error) {
	args := m.Called(repo, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (m *MockGitManager) WriteFiles(repo *git.Repository, basePath string, files map[string][]byte) error {
	args := m.Called(repo, basePath, files)
	return args.Error(0)
}

func (m *MockGitManager) CommitAndPush(ctx context.Context, repo *git.Repository, repoSpec gitopsv1beta1.GitRepositorySpec, message string, author string, email string) error {
	args := m.Called(ctx, repo, repoSpec, message, author, email)
	return args.Error(0)
}

func (m *MockGitManager) Cleanup() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitManager) CheckoutRevision(repo *git.Repository, revision string) error {
	args := m.Called(repo, revision)
	return args.Error(0)
}

type MockApprovalStore struct {
	mock.Mock
}

func (m *MockApprovalStore) GetApprovals(ctx context.Context, promotionID string) ([]Approval, error) {
	args := m.Called(ctx, promotionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Approval), args.Error(1)
}

func (m *MockApprovalStore) AddApproval(ctx context.Context, promotionID string, approval Approval) error {
	args := m.Called(ctx, promotionID, approval)
	return args.Error(0)
}

func (m *MockApprovalStore) ClearApprovals(ctx context.Context, promotionID string) error {
	args := m.Called(ctx, promotionID)
	return args.Error(0)
}

type MockPromotionMetrics struct {
	mock.Mock
}

func (m *MockPromotionMetrics) RecordPromotionStarted(from, to string) {
	m.Called(from, to)
}

func (m *MockPromotionMetrics) RecordPromotionCompleted(from, to string, success bool, duration time.Duration) {
	m.Called(from, to, success, duration)
}

func (m *MockPromotionMetrics) RecordPromotionGatePassed(from, to, gateType string) {
	m.Called(from, to, gateType)
}

func (m *MockPromotionMetrics) RecordPromotionGateFailed(from, to, gateType string) {
	m.Called(from, to, gateType)
}
