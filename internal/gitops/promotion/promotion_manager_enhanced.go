package promotion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// PromotionManager handles multi-environment promotions with approval gates
type PromotionManager struct {
	client      client.Client
	scheme      *runtime.Scheme
	log         logr.Logger
	approvalMgr *ApprovalManager
}

// NewPromotionManager creates a new PromotionManager
func NewPromotionManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *PromotionManager {
	return &PromotionManager{
		client:      client,
		scheme:      scheme,
		log:         log.WithName("promotion-manager"),
		approvalMgr: NewApprovalManager(client, scheme, log),
	}
}

// ApprovalManager handles approval workflows
type ApprovalManager struct {
	client client.Client
	scheme *runtime.Scheme
	log    logr.Logger
}

// NewApprovalManager creates a new ApprovalManager
func NewApprovalManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *ApprovalManager {
	return &ApprovalManager{
		client: client,
		scheme: scheme,
		log:    log.WithName("approval-manager"),
	}
}

// PromoteDeployment promotes a deployment from one environment to another
func (m *PromotionManager) PromoteDeployment(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	m.log.Info("Starting promotion process", 
		"promotion", promotion.Name,
		"from", promotion.Spec.FromEnvironment,
		"to", promotion.Spec.ToEnvironment)

	// Validate promotion
	if err := m.validatePromotion(ctx, promotion); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check approval requirements
	if promotion.Spec.ApprovalPolicy != nil && promotion.Spec.ApprovalPolicy.Required {
		approved, err := m.approvalMgr.CheckApproval(ctx, promotion)
		if err != nil {
			return fmt.Errorf("checking approval: %w", err)
		}

		if !approved {
			m.log.Info("Promotion requires approval", "promotion", promotion.Name)
			if err := m.createApprovalRequest(ctx, promotion); err != nil {
				return fmt.Errorf("creating approval request: %w", err)
			}
			
			// Update status to pending approval
			promotion.Status.Phase = observabilityv1.PromotionPhasePendingApproval
			promotion.Status.Message = "Waiting for approval"
			if err := m.client.Status().Update(ctx, promotion); err != nil {
				return fmt.Errorf("updating status: %w", err)
			}
			
			return nil
		}
	}

	// Execute pre-promotion checks
	if len(promotion.Spec.PrePromotionChecks) > 0 {
		if err := m.runPrePromotionChecks(ctx, promotion); err != nil {
			return fmt.Errorf("pre-promotion checks failed: %w", err)
		}
	}

	// Perform the actual promotion
	if err := m.executePromotion(ctx, promotion); err != nil {
		return fmt.Errorf("executing promotion: %w", err)
	}

	// Run post-promotion validation
	if len(promotion.Spec.PostPromotionChecks) > 0 {
		if err := m.runPostPromotionChecks(ctx, promotion); err != nil {
			// Rollback on post-promotion check failure
			m.log.Error(err, "Post-promotion checks failed, initiating rollback")
			if rbErr := m.rollbackPromotion(ctx, promotion); rbErr != nil {
				m.log.Error(rbErr, "Failed to rollback promotion")
			}
			return fmt.Errorf("post-promotion checks failed: %w", err)
		}
	}

	// Update promotion status
	promotion.Status.Phase = observabilityv1.PromotionPhaseCompleted
	promotion.Status.Message = "Promotion completed successfully"
	promotion.Status.CompletedAt = &metav1.Time{Time: time.Now()}
	if err := m.client.Status().Update(ctx, promotion); err != nil {
		return fmt.Errorf("updating final status: %w", err)
	}

	// Create success event
	if err := m.createEvent(ctx, promotion, "PromotionCompleted", 
		fmt.Sprintf("Successfully promoted from %s to %s", 
			promotion.Spec.FromEnvironment, promotion.Spec.ToEnvironment)); err != nil {
		m.log.Error(err, "Failed to create success event")
	}

	return nil
}

// validatePromotion validates the promotion request
func (m *PromotionManager) validatePromotion(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	// Get source GitOpsDeployment
	sourceGitOps := &observabilityv1.GitOpsDeployment{}
	sourceKey := types.NamespacedName{
		Name:      promotion.Spec.SourceRef.Name,
		Namespace: promotion.Spec.SourceRef.Namespace,
	}
	
	if err := m.client.Get(ctx, sourceKey, sourceGitOps); err != nil {
		return fmt.Errorf("getting source GitOpsDeployment: %w", err)
	}

	// Verify source is in a healthy state
	if sourceGitOps.Status.Phase != observabilityv1.GitOpsPhaseReady {
		return fmt.Errorf("source deployment is not ready: %s", sourceGitOps.Status.Phase)
	}

	// Check if target environment exists
	if promotion.Spec.TargetRef != nil {
		targetGitOps := &observabilityv1.GitOpsDeployment{}
		targetKey := types.NamespacedName{
			Name:      promotion.Spec.TargetRef.Name,
			Namespace: promotion.Spec.TargetRef.Namespace,
		}
		
		if err := m.client.Get(ctx, targetKey, targetGitOps); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("getting target GitOpsDeployment: %w", err)
			}
			// Target doesn't exist, will be created during promotion
		}
	}

	// Validate promotion strategy
	if promotion.Spec.Strategy == "" {
		promotion.Spec.Strategy = observabilityv1.PromotionStrategyDirect
	}

	return nil
}

// createApprovalRequest creates an approval request for the promotion
func (m *PromotionManager) createApprovalRequest(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	approval := &observabilityv1.ApprovalRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-approval", promotion.Name),
			Namespace: promotion.Namespace,
			Labels: map[string]string{
				"observability.io/promotion": promotion.Name,
				"observability.io/type":      "promotion",
			},
		},
		Spec: observabilityv1.ApprovalRequestSpec{
			PromotionRef: &corev1.ObjectReference{
				Kind:       "GitOpsPromotion",
				Name:       promotion.Name,
				Namespace:  promotion.Namespace,
				UID:        promotion.UID,
				APIVersion: observabilityv1.GroupVersion.String(),
			},
			Description: fmt.Sprintf("Approve promotion from %s to %s",
				promotion.Spec.FromEnvironment, promotion.Spec.ToEnvironment),
			RequiredApprovers: promotion.Spec.ApprovalPolicy.Approvers,
			MinApprovals:      promotion.Spec.ApprovalPolicy.MinApprovals,
			Timeout:           promotion.Spec.ApprovalPolicy.Timeout,
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(promotion, approval, m.scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	if err := m.client.Create(ctx, approval); err != nil {
		return fmt.Errorf("creating approval request: %w", err)
	}

	// Send notifications if configured
	if len(promotion.Spec.ApprovalPolicy.NotificationChannels) > 0 {
		if err := m.sendApprovalNotifications(ctx, promotion, approval); err != nil {
			m.log.Error(err, "Failed to send approval notifications")
		}
	}

	return nil
}

// CheckApproval checks if a promotion has been approved
func (am *ApprovalManager) CheckApproval(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) (bool, error) {
	// Find approval request
	approvalList := &observabilityv1.ApprovalRequestList{}
	if err := am.client.List(ctx, approvalList, 
		client.InNamespace(promotion.Namespace),
		client.MatchingLabels{
			"observability.io/promotion": promotion.Name,
		}); err != nil {
		return false, fmt.Errorf("listing approval requests: %w", err)
	}

	if len(approvalList.Items) == 0 {
		return false, nil // No approval request found
	}

	approval := &approvalList.Items[0]

	// Check if approved
	if approval.Status.Approved {
		am.log.Info("Promotion approved", 
			"promotion", promotion.Name,
			"approvers", approval.Status.Approvers)
		return true, nil
	}

	// Check if timeout exceeded
	if approval.Spec.Timeout != "" {
		timeout, err := time.ParseDuration(approval.Spec.Timeout)
		if err == nil {
			if time.Since(approval.CreationTimestamp.Time) > timeout {
				am.log.Info("Approval timeout exceeded", "promotion", promotion.Name)
				approval.Status.Approved = false
				approval.Status.Reason = "Timeout exceeded"
				if err := am.client.Status().Update(ctx, approval); err != nil {
					am.log.Error(err, "Failed to update approval status")
				}
				return false, fmt.Errorf("approval timeout exceeded")
			}
		}
	}

	return false, nil
}

// runPrePromotionChecks executes pre-promotion validation checks
func (m *PromotionManager) runPrePromotionChecks(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	m.log.Info("Running pre-promotion checks", "promotion", promotion.Name)

	for _, check := range promotion.Spec.PrePromotionChecks {
		m.log.V(1).Info("Executing check", "type", check.Type, "name", check.Name)
		
		switch check.Type {
		case observabilityv1.CheckTypeTest:
			if err := m.runTestCheck(ctx, check); err != nil {
				return fmt.Errorf("test check %s failed: %w", check.Name, err)
			}
		case observabilityv1.CheckTypeHealth:
			if err := m.runHealthCheck(ctx, check); err != nil {
				return fmt.Errorf("health check %s failed: %w", check.Name, err)
			}
		case observabilityv1.CheckTypeMetric:
			if err := m.runMetricCheck(ctx, check); err != nil {
				return fmt.Errorf("metric check %s failed: %w", check.Name, err)
			}
		case observabilityv1.CheckTypeScript:
			if err := m.runScriptCheck(ctx, check); err != nil {
				return fmt.Errorf("script check %s failed: %w", check.Name, err)
			}
		default:
			m.log.Info("Unknown check type, skipping", "type", check.Type)
		}
	}

	return nil
}

// executePromotion performs the actual promotion
func (m *PromotionManager) executePromotion(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	m.log.Info("Executing promotion", "promotion", promotion.Name, "strategy", promotion.Spec.Strategy)

	// Update promotion status
	promotion.Status.Phase = observabilityv1.PromotionPhaseProgressing
	promotion.Status.Message = "Promotion in progress"
	promotion.Status.StartedAt = &metav1.Time{Time: time.Now()}
	if err := m.client.Status().Update(ctx, promotion); err != nil {
		m.log.Error(err, "Failed to update promotion status")
	}

	switch promotion.Spec.Strategy {
	case observabilityv1.PromotionStrategyDirect:
		return m.executeDirect(ctx, promotion)
	case observabilityv1.PromotionStrategyBlueGreen:
		return m.executeBlueGreen(ctx, promotion)
	case observabilityv1.PromotionStrategyCanary:
		return m.executeCanary(ctx, promotion)
	case observabilityv1.PromotionStrategyProgressive:
		return m.executeProgressive(ctx, promotion)
	default:
		return fmt.Errorf("unknown promotion strategy: %s", promotion.Spec.Strategy)
	}
}

// executeDirect performs a direct promotion (immediate switch)
func (m *PromotionManager) executeDirect(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	// Get source deployment
	sourceGitOps := &observabilityv1.GitOpsDeployment{}
	if err := m.client.Get(ctx, types.NamespacedName{
		Name:      promotion.Spec.SourceRef.Name,
		Namespace: promotion.Spec.SourceRef.Namespace,
	}, sourceGitOps); err != nil {
		return fmt.Errorf("getting source deployment: %w", err)
	}

	// Create or update target deployment
	targetGitOps := &observabilityv1.GitOpsDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      promotion.Spec.TargetRef.Name,
			Namespace: promotion.Spec.TargetRef.Namespace,
			Labels: map[string]string{
				"observability.io/environment": promotion.Spec.ToEnvironment,
				"observability.io/promoted-by": promotion.Name,
			},
		},
		Spec: sourceGitOps.Spec.DeepCopy(),
	}

	// Apply environment-specific overrides
	if promotion.Spec.Overrides != nil {
		if err := m.applyOverrides(targetGitOps, promotion.Spec.Overrides); err != nil {
			return fmt.Errorf("applying overrides: %w", err)
		}
	}

	// Create or update the target
	op, err := controllerutil.CreateOrUpdate(ctx, m.client, targetGitOps, func() error {
		targetGitOps.Spec = *sourceGitOps.Spec.DeepCopy()
		
		// Update with promoted commit/tag
		if promotion.Spec.TargetRevision != "" {
			targetGitOps.Spec.Branch = ""
			targetGitOps.Spec.Tag = promotion.Spec.TargetRevision
		} else {
			targetGitOps.Spec.Tag = sourceGitOps.Status.LastSyncedCommit
		}
		
		// Apply overrides again after spec copy
		if promotion.Spec.Overrides != nil {
			return m.applyOverrides(targetGitOps, promotion.Spec.Overrides)
		}
		
		return nil
	})

	if err != nil {
		return fmt.Errorf("creating/updating target deployment: %w", err)
	}

	m.log.Info("Target deployment updated", "operation", op, "target", targetGitOps.Name)
	
	// Record promotion in status
	promotion.Status.PromotedRevision = sourceGitOps.Status.LastSyncedCommit
	promotion.Status.TargetDeployment = fmt.Sprintf("%s/%s", targetGitOps.Namespace, targetGitOps.Name)

	return nil
}

// executeBlueGreen performs a blue-green promotion
func (m *PromotionManager) executeBlueGreen(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	// TODO: Implement blue-green deployment strategy
	// 1. Deploy to inactive environment
	// 2. Run validation
	// 3. Switch traffic
	// 4. Monitor
	// 5. Cleanup old version
	return fmt.Errorf("blue-green strategy not yet implemented")
}

// executeCanary performs a canary promotion
func (m *PromotionManager) executeCanary(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	// TODO: Implement canary deployment strategy
	// 1. Deploy canary version
	// 2. Route percentage of traffic
	// 3. Monitor metrics
	// 4. Gradually increase traffic
	// 5. Complete or rollback
	return fmt.Errorf("canary strategy not yet implemented")
}

// executeProgressive performs a progressive promotion
func (m *PromotionManager) executeProgressive(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	// TODO: Implement progressive deployment strategy
	// 1. Deploy to subset of instances
	// 2. Monitor health
	// 3. Expand deployment
	// 4. Repeat until complete
	return fmt.Errorf("progressive strategy not yet implemented")
}

// applyOverrides applies environment-specific overrides to the target deployment
func (m *PromotionManager) applyOverrides(gitops *observabilityv1.GitOpsDeployment, overrides *observabilityv1.PromotionOverrides) error {
	// Apply path override
	if overrides.Path != "" {
		gitops.Spec.Path = overrides.Path
	}

	// Apply value overrides
	if len(overrides.Values) > 0 {
		if gitops.Spec.Values == nil {
			gitops.Spec.Values = make(map[string]string)
		}
		for k, v := range overrides.Values {
			gitops.Spec.Values[k] = v
		}
	}

	// Apply config patches
	if len(overrides.Patches) > 0 {
		// TODO: Implement strategic merge patches
		m.log.Info("Config patches not yet implemented", "patches", len(overrides.Patches))
	}

	return nil
}

// runPostPromotionChecks executes post-promotion validation
func (m *PromotionManager) runPostPromotionChecks(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	m.log.Info("Running post-promotion checks", "promotion", promotion.Name)

	// Similar to pre-promotion checks but run after deployment
	for _, check := range promotion.Spec.PostPromotionChecks {
		m.log.V(1).Info("Executing post-check", "type", check.Type, "name", check.Name)
		
		// Add a delay if specified
		if check.Delay != "" {
			delay, err := time.ParseDuration(check.Delay)
			if err == nil {
				m.log.Info("Waiting before check", "delay", delay)
				time.Sleep(delay)
			}
		}

		switch check.Type {
		case observabilityv1.CheckTypeTest:
			if err := m.runTestCheck(ctx, check); err != nil {
				return fmt.Errorf("test check %s failed: %w", check.Name, err)
			}
		case observabilityv1.CheckTypeHealth:
			if err := m.runHealthCheck(ctx, check); err != nil {
				return fmt.Errorf("health check %s failed: %w", check.Name, err)
			}
		case observabilityv1.CheckTypeMetric:
			if err := m.runMetricCheck(ctx, check); err != nil {
				return fmt.Errorf("metric check %s failed: %w", check.Name, err)
			}
		case observabilityv1.CheckTypeScript:
			if err := m.runScriptCheck(ctx, check); err != nil {
				return fmt.Errorf("script check %s failed: %w", check.Name, err)
			}
		}
	}

	return nil
}

// rollbackPromotion rolls back a failed promotion
func (m *PromotionManager) rollbackPromotion(ctx context.Context, promotion *observabilityv1.GitOpsPromotion) error {
	m.log.Info("Rolling back promotion", "promotion", promotion.Name)

	// Update status
	promotion.Status.Phase = observabilityv1.PromotionPhaseRollingBack
	promotion.Status.Message = "Rolling back due to post-promotion check failure"
	if err := m.client.Status().Update(ctx, promotion); err != nil {
		m.log.Error(err, "Failed to update rollback status")
	}

	// TODO: Implement rollback logic based on strategy
	// For now, just mark as failed
	promotion.Status.Phase = observabilityv1.PromotionPhaseFailed
	promotion.Status.Message = "Promotion failed and rollback initiated"
	
	return m.client.Status().Update(ctx, promotion)
}

// Check implementation methods
func (m *PromotionManager) runTestCheck(ctx context.Context, check observabilityv1.PromotionCheck) error {
	// TODO: Implement test execution (e.g., run test Job)
	m.log.V(1).Info("Running test check", "check", check.Name)
	return nil
}

func (m *PromotionManager) runHealthCheck(ctx context.Context, check observabilityv1.PromotionCheck) error {
	// TODO: Implement health check
	m.log.V(1).Info("Running health check", "check", check.Name)
	return nil
}

func (m *PromotionManager) runMetricCheck(ctx context.Context, check observabilityv1.PromotionCheck) error {
	// TODO: Implement metric query and threshold check
	m.log.V(1).Info("Running metric check", "check", check.Name)
	return nil
}

func (m *PromotionManager) runScriptCheck(ctx context.Context, check observabilityv1.PromotionCheck) error {
	// TODO: Implement script execution
	m.log.V(1).Info("Running script check", "check", check.Name)
	return nil
}

// sendApprovalNotifications sends notifications for approval requests
func (m *PromotionManager) sendApprovalNotifications(ctx context.Context, promotion *observabilityv1.GitOpsPromotion, approval *observabilityv1.ApprovalRequest) error {
	for _, channel := range promotion.Spec.ApprovalPolicy.NotificationChannels {
		switch channel.Type {
		case "slack":
			// TODO: Send Slack notification
			m.log.V(1).Info("Sending Slack notification", "webhook", channel.URL)
		case "email":
			// TODO: Send email notification
			m.log.V(1).Info("Sending email notification", "addresses", channel.Addresses)
		case "webhook":
			// TODO: Send webhook notification
			m.log.V(1).Info("Sending webhook notification", "url", channel.URL)
		}
	}
	return nil
}

// createEvent creates a Kubernetes event for the promotion
func (m *PromotionManager) createEvent(ctx context.Context, promotion *observabilityv1.GitOpsPromotion, reason, message string) error {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%d", promotion.Name, strings.ToLower(reason), time.Now().Unix()),
			Namespace: promotion.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "GitOpsPromotion",
			Name:       promotion.Name,
			Namespace:  promotion.Namespace,
			UID:        promotion.UID,
			APIVersion: observabilityv1.GroupVersion.String(),
		},
		Reason:  reason,
		Message: message,
		Type:    corev1.EventTypeNormal,
		Source: corev1.EventSource{
			Component: "gunj-operator-promotion",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	return m.client.Create(ctx, event)
}

// ProcessApproval processes an approval decision
func (am *ApprovalManager) ProcessApproval(ctx context.Context, approvalName, namespace, approver string, approved bool, reason string) error {
	approval := &observabilityv1.ApprovalRequest{}
	if err := am.client.Get(ctx, types.NamespacedName{
		Name:      approvalName,
		Namespace: namespace,
	}, approval); err != nil {
		return fmt.Errorf("getting approval request: %w", err)
	}

	// Check if approver is authorized
	authorized := false
	for _, a := range approval.Spec.RequiredApprovers {
		if a == approver {
			authorized = true
			break
		}
	}

	if !authorized {
		return fmt.Errorf("user %s is not authorized to approve this request", approver)
	}

	// Check if already approved/rejected
	if approval.Status.Approved || approval.Status.Rejected {
		return fmt.Errorf("approval request has already been processed")
	}

	// Record approval
	if approval.Status.Approvers == nil {
		approval.Status.Approvers = []string{}
	}
	approval.Status.Approvers = append(approval.Status.Approvers, approver)

	if approved {
		// Check if we have enough approvals
		if len(approval.Status.Approvers) >= approval.Spec.MinApprovals {
			approval.Status.Approved = true
			approval.Status.ApprovedAt = &metav1.Time{Time: time.Now()}
			approval.Status.Reason = reason
		}
	} else {
		approval.Status.Rejected = true
		approval.Status.RejectedAt = &metav1.Time{Time: time.Now()}
		approval.Status.Reason = reason
	}

	if err := am.client.Status().Update(ctx, approval); err != nil {
		return fmt.Errorf("updating approval status: %w", err)
	}

	// If approved, trigger the promotion to continue
	if approval.Status.Approved && approval.Spec.PromotionRef != nil {
		promotion := &observabilityv1.GitOpsPromotion{}
		if err := am.client.Get(ctx, types.NamespacedName{
			Name:      approval.Spec.PromotionRef.Name,
			Namespace: approval.Spec.PromotionRef.Namespace,
		}, promotion); err != nil {
			return fmt.Errorf("getting promotion: %w", err)
		}

		// Update promotion to continue
		promotion.Status.Phase = observabilityv1.PromotionPhaseApproved
		promotion.Status.Message = fmt.Sprintf("Approved by %s", strings.Join(approval.Status.Approvers, ", "))
		if err := am.client.Status().Update(ctx, promotion); err != nil {
			return fmt.Errorf("updating promotion status: %w", err)
		}
	}

	return nil
}
