package rollback

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// RollbackManager handles automated rollback operations
type RollbackManager struct {
	client        client.Client
	scheme        *runtime.Scheme
	log           logr.Logger
	healthChecker *HealthChecker
}

// NewRollbackManager creates a new RollbackManager
func NewRollbackManager(client client.Client, scheme *runtime.Scheme, log logr.Logger) *RollbackManager {
	return &RollbackManager{
		client:        client,
		scheme:        scheme,
		log:           log.WithName("rollback-manager"),
		healthChecker: NewHealthChecker(client, log),
	}
}

// HealthChecker performs health checks on deployments
type HealthChecker struct {
	client client.Client
	log    logr.Logger
}

// NewHealthChecker creates a new HealthChecker
func NewHealthChecker(client client.Client, log logr.Logger) *HealthChecker {
	return &HealthChecker{
		client: client,
		log:    log.WithName("health-checker"),
	}
}

// CheckDeploymentHealth checks if a deployment is healthy
func (h *HealthChecker) CheckDeploymentHealth(ctx context.Context, gitops *observabilityv1.GitOpsDeployment) (*HealthStatus, error) {
	h.log.Info("Checking deployment health", "name", gitops.Name, "namespace", gitops.Namespace)

	status := &HealthStatus{
		Healthy:         true,
		CheckedAt:       metav1.Now(),
		FailedChecks:    []string{},
		PassedChecks:    []string{},
		ResourceStatus:  make(map[string]ResourceHealth),
	}

	// Get health check configuration
	healthConfig := gitops.Spec.HealthChecks
	if healthConfig == nil {
		healthConfig = &observabilityv1.HealthCheckConfig{
			Enabled:  true,
			Interval: "1m",
			Timeout:  "5m",
		}
	}

	// Check each resource specified in health checks
	for _, check := range healthConfig.Resources {
		resourceHealth, err := h.checkResource(ctx, check)
		if err != nil {
			h.log.Error(err, "Failed to check resource health", 
				"resource", check.Kind, 
				"name", check.Name)
			status.Healthy = false
			status.FailedChecks = append(status.FailedChecks, 
				fmt.Sprintf("%s/%s: %v", check.Kind, check.Name, err))
			resourceHealth = ResourceHealth{
				Healthy: false,
				Message: err.Error(),
			}
		} else if !resourceHealth.Healthy {
			status.Healthy = false
			status.FailedChecks = append(status.FailedChecks, 
				fmt.Sprintf("%s/%s: %s", check.Kind, check.Name, resourceHealth.Message))
		} else {
			status.PassedChecks = append(status.PassedChecks, 
				fmt.Sprintf("%s/%s", check.Kind, check.Name))
		}
		
		status.ResourceStatus[fmt.Sprintf("%s/%s", check.Kind, check.Name)] = resourceHealth
	}

	// Check custom health endpoints if configured
	if len(healthConfig.HTTPChecks) > 0 {
		for _, httpCheck := range healthConfig.HTTPChecks {
			healthy, err := h.checkHTTPEndpoint(ctx, httpCheck)
			checkName := fmt.Sprintf("HTTP: %s", httpCheck.URL)
			
			if err != nil || !healthy {
				status.Healthy = false
				msg := fmt.Sprintf("HTTP check failed: %s", httpCheck.URL)
				if err != nil {
					msg += fmt.Sprintf(" - %v", err)
				}
				status.FailedChecks = append(status.FailedChecks, msg)
			} else {
				status.PassedChecks = append(status.PassedChecks, checkName)
			}
		}
	}

	// Check metrics if configured
	if len(healthConfig.Metrics) > 0 {
		for _, metric := range healthConfig.Metrics {
			value, err := h.checkMetric(ctx, metric)
			checkName := fmt.Sprintf("Metric: %s", metric.Query)
			
			if err != nil {
				status.Healthy = false
				status.FailedChecks = append(status.FailedChecks, 
					fmt.Sprintf("%s - Error: %v", checkName, err))
			} else {
				passed := h.evaluateMetricThreshold(value, metric.Threshold, metric.Operator)
				if !passed {
					status.Healthy = false
					status.FailedChecks = append(status.FailedChecks, 
						fmt.Sprintf("%s - Value: %f, Expected: %s %f", 
							checkName, value, metric.Operator, metric.Threshold))
				} else {
					status.PassedChecks = append(status.PassedChecks, checkName)
				}
			}
		}
	}

	return status, nil
}

// checkResource checks the health of a specific Kubernetes resource
func (h *HealthChecker) checkResource(ctx context.Context, check observabilityv1.ResourceCheck) (ResourceHealth, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(check.APIVersion)
	obj.SetKind(check.Kind)
	obj.SetName(check.Name)
	obj.SetNamespace(check.Namespace)

	if err := h.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Resource not found: %v", err),
		}, err
	}

	// Check resource-specific health
	switch check.Kind {
	case "Deployment":
		return h.checkDeploymentResource(obj)
	case "StatefulSet":
		return h.checkStatefulSetResource(obj)
	case "DaemonSet":
		return h.checkDaemonSetResource(obj)
	case "Pod":
		return h.checkPodResource(obj)
	case "Service":
		return h.checkServiceResource(obj)
	default:
		// For custom resources, check conditions
		return h.checkCustomResource(obj, check.HealthConditions)
	}
}

// checkDeploymentResource checks if a Deployment is healthy
func (h *HealthChecker) checkDeploymentResource(obj *unstructured.Unstructured) (ResourceHealth, error) {
	replicas, found, err := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if err != nil || !found {
		replicas = 1
	}

	readyReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	updatedReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "updatedReplicas")
	availableReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "availableReplicas")

	if readyReplicas < replicas {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Only %d/%d replicas are ready", readyReplicas, replicas),
			Ready:   int(readyReplicas),
			Total:   int(replicas),
		}, nil
	}

	if updatedReplicas < replicas {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Only %d/%d replicas are updated", updatedReplicas, replicas),
			Ready:   int(readyReplicas),
			Total:   int(replicas),
		}, nil
	}

	if availableReplicas < replicas {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Only %d/%d replicas are available", availableReplicas, replicas),
			Ready:   int(readyReplicas),
			Total:   int(replicas),
		}, nil
	}

	// Check conditions
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		
		condType, _, _ := unstructured.NestedString(cond, "type")
		condStatus, _, _ := unstructured.NestedString(cond, "status")
		condMessage, _, _ := unstructured.NestedString(cond, "message")
		
		if condType == "Progressing" && condStatus == "False" {
			return ResourceHealth{
				Healthy: false,
				Message: fmt.Sprintf("Deployment is not progressing: %s", condMessage),
				Ready:   int(readyReplicas),
				Total:   int(replicas),
			}, nil
		}
	}

	return ResourceHealth{
		Healthy: true,
		Message: "Deployment is healthy",
		Ready:   int(readyReplicas),
		Total:   int(replicas),
	}, nil
}

// MonitorAndRollback monitors deployment health and performs automatic rollback
func (m *RollbackManager) MonitorAndRollback(ctx context.Context, gitops *observabilityv1.GitOpsDeployment) error {
	m.log.Info("Starting health monitoring", "name", gitops.Name, "namespace", gitops.Namespace)

	if gitops.Spec.RollbackConfig == nil || !gitops.Spec.RollbackConfig.AutoRollback {
		m.log.Info("Auto-rollback is disabled", "name", gitops.Name)
		return nil
	}

	// Create rollback state
	state := &RollbackState{
		GitOpsName:      gitops.Name,
		GitOpsNamespace: gitops.Namespace,
		StartTime:       time.Now(),
		LastHealthCheck: time.Now(),
	}

	// Save initial state before deployment
	if err := m.saveDeploymentState(ctx, gitops, state); err != nil {
		return fmt.Errorf("saving deployment state: %w", err)
	}

	// Start monitoring loop
	timeout := 5 * time.Minute
	if gitops.Spec.HealthChecks != nil && gitops.Spec.HealthChecks.Timeout != "" {
		duration, err := time.ParseDuration(gitops.Spec.HealthChecks.Timeout)
		if err == nil {
			timeout = duration
		}
	}

	checkInterval := 30 * time.Second
	if gitops.Spec.HealthChecks != nil && gitops.Spec.HealthChecks.Interval != "" {
		duration, err := time.ParseDuration(gitops.Spec.HealthChecks.Interval)
		if err == nil {
			checkInterval = duration
		}
	}

	// Wait for initial stabilization
	time.Sleep(10 * time.Second)

	// Monitor health with timeout
	err := wait.PollImmediate(checkInterval, timeout, func() (bool, error) {
		healthStatus, err := m.healthChecker.CheckDeploymentHealth(ctx, gitops)
		if err != nil {
			m.log.Error(err, "Error checking deployment health")
			return false, nil // Continue polling
		}

		state.LastHealthCheck = time.Now()
		state.ConsecutiveFailures = 0

		if !healthStatus.Healthy {
			state.ConsecutiveFailures++
			m.log.Info("Health check failed", 
				"failures", state.ConsecutiveFailures,
				"threshold", gitops.Spec.RollbackConfig.FailureThreshold,
				"failed_checks", healthStatus.FailedChecks)

			// Check if we've exceeded the failure threshold
			if state.ConsecutiveFailures >= gitops.Spec.RollbackConfig.FailureThreshold {
				m.log.Info("Failure threshold exceeded, initiating rollback",
					"failures", state.ConsecutiveFailures,
					"threshold", gitops.Spec.RollbackConfig.FailureThreshold)
				
				// Perform rollback
				if err := m.performRollback(ctx, gitops, state, healthStatus); err != nil {
					return false, fmt.Errorf("performing rollback: %w", err)
				}
				
				return true, nil // Stop monitoring after rollback
			}
		} else {
			m.log.V(1).Info("Health check passed", "passed_checks", healthStatus.PassedChecks)
			state.ConsecutiveFailures = 0
		}

		// Continue monitoring
		return false, nil
	})

	if err != nil && err != wait.ErrWaitTimeout {
		return fmt.Errorf("monitoring deployment health: %w", err)
	}

	if err == wait.ErrWaitTimeout {
		m.log.Info("Health monitoring completed successfully", "name", gitops.Name)
	}

	return nil
}

// performRollback executes the rollback operation
func (m *RollbackManager) performRollback(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, state *RollbackState, healthStatus *HealthStatus) error {
	m.log.Info("Performing rollback", "name", gitops.Name, "namespace", gitops.Namespace)

	// Create rollback event
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-rollback-%d", gitops.Name, time.Now().Unix()),
			Namespace: gitops.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "GitOpsDeployment",
			Name:       gitops.Name,
			Namespace:  gitops.Namespace,
			UID:        gitops.UID,
			APIVersion: observabilityv1.GroupVersion.String(),
		},
		Reason:  "AutomaticRollback",
		Message: fmt.Sprintf("Automatic rollback triggered due to health check failures: %v", healthStatus.FailedChecks),
		Type:    corev1.EventTypeWarning,
		Source: corev1.EventSource{
			Component: "gunj-operator",
		},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}

	if err := m.client.Create(ctx, event); err != nil {
		m.log.Error(err, "Failed to create rollback event")
	}

	// Update GitOps status to indicate rollback
	gitops.Status.RollbackStatus = &observabilityv1.RollbackStatus{
		InProgress:    true,
		StartTime:     &metav1.Time{Time: time.Now()},
		Reason:        fmt.Sprintf("Health check failures: %v", healthStatus.FailedChecks),
		PreviousState: state.SavedState,
	}

	if err := m.client.Status().Update(ctx, gitops); err != nil {
		m.log.Error(err, "Failed to update rollback status")
	}

	// Execute rollback based on sync provider
	switch gitops.Spec.SyncProvider {
	case observabilityv1.ArgoCD:
		return m.rollbackArgoCD(ctx, gitops, state)
	case observabilityv1.Flux:
		return m.rollbackFlux(ctx, gitops, state)
	default:
		return m.rollbackGit(ctx, gitops, state)
	}
}

// saveDeploymentState saves the current state before deployment
func (m *RollbackManager) saveDeploymentState(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, state *RollbackState) error {
	savedState := &SavedDeploymentState{
		Timestamp: metav1.Now(),
		Resources: make(map[string]interface{}),
	}

	// Save current Git revision
	if gitops.Status.LastSyncedCommit != "" {
		savedState.GitRevision = gitops.Status.LastSyncedCommit
	}

	// Save current resource states if specified
	if gitops.Spec.RollbackConfig != nil && len(gitops.Spec.RollbackConfig.ResourcesSnapshot) > 0 {
		for _, resource := range gitops.Spec.RollbackConfig.ResourcesSnapshot {
			obj := &unstructured.Unstructured{}
			obj.SetAPIVersion(resource.APIVersion)
			obj.SetKind(resource.Kind)
			obj.SetName(resource.Name)
			obj.SetNamespace(resource.Namespace)

			if err := m.client.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
				if !errors.IsNotFound(err) {
					m.log.Error(err, "Failed to get resource for snapshot",
						"resource", resource.Kind,
						"name", resource.Name)
				}
				continue
			}

			// Save resource spec
			spec, found, _ := unstructured.NestedMap(obj.Object, "spec")
			if found {
				key := fmt.Sprintf("%s/%s/%s", resource.Kind, resource.Namespace, resource.Name)
				savedState.Resources[key] = spec
			}
		}
	}

	state.SavedState = savedState
	return nil
}

// Additional helper types
type HealthStatus struct {
	Healthy        bool
	CheckedAt      metav1.Time
	FailedChecks   []string
	PassedChecks   []string
	ResourceStatus map[string]ResourceHealth
}

type ResourceHealth struct {
	Healthy bool
	Message string
	Ready   int
	Total   int
}

type RollbackState struct {
	GitOpsName          string
	GitOpsNamespace     string
	StartTime           time.Time
	LastHealthCheck     time.Time
	ConsecutiveFailures int
	SavedState          *SavedDeploymentState
}

type SavedDeploymentState struct {
	Timestamp   metav1.Time
	GitRevision string
	Resources   map[string]interface{}
}

// Implement remaining health check methods...
func (h *HealthChecker) checkStatefulSetResource(obj *unstructured.Unstructured) (ResourceHealth, error) {
	// Similar implementation to checkDeploymentResource
	replicas, found, err := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if err != nil || !found {
		replicas = 1
	}

	readyReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	currentReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "currentReplicas")

	if readyReplicas < replicas {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Only %d/%d replicas are ready", readyReplicas, replicas),
			Ready:   int(readyReplicas),
			Total:   int(replicas),
		}, nil
	}

	if currentReplicas < replicas {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Only %d/%d replicas are current", currentReplicas, replicas),
			Ready:   int(readyReplicas),
			Total:   int(replicas),
		}, nil
	}

	return ResourceHealth{
		Healthy: true,
		Message: "StatefulSet is healthy",
		Ready:   int(readyReplicas),
		Total:   int(replicas),
	}, nil
}

func (h *HealthChecker) checkDaemonSetResource(obj *unstructured.Unstructured) (ResourceHealth, error) {
	desiredNumberScheduled, _, _ := unstructured.NestedInt64(obj.Object, "status", "desiredNumberScheduled")
	numberReady, _, _ := unstructured.NestedInt64(obj.Object, "status", "numberReady")

	if numberReady < desiredNumberScheduled {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Only %d/%d pods are ready", numberReady, desiredNumberScheduled),
			Ready:   int(numberReady),
			Total:   int(desiredNumberScheduled),
		}, nil
	}

	return ResourceHealth{
		Healthy: true,
		Message: "DaemonSet is healthy",
		Ready:   int(numberReady),
		Total:   int(desiredNumberScheduled),
	}, nil
}

func (h *HealthChecker) checkPodResource(obj *unstructured.Unstructured) (ResourceHealth, error) {
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	
	if phase != "Running" && phase != "Succeeded" {
		return ResourceHealth{
			Healthy: false,
			Message: fmt.Sprintf("Pod is in %s phase", phase),
		}, nil
	}

	// Check container statuses
	containers, _, _ := unstructured.NestedSlice(obj.Object, "status", "containerStatuses")
	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		
		ready, _, _ := unstructured.NestedBool(container, "ready")
		if !ready {
			name, _, _ := unstructured.NestedString(container, "name")
			return ResourceHealth{
				Healthy: false,
				Message: fmt.Sprintf("Container %s is not ready", name),
			}, nil
		}
	}

	return ResourceHealth{
		Healthy: true,
		Message: "Pod is healthy",
		Ready:   1,
		Total:   1,
	}, nil
}

func (h *HealthChecker) checkServiceResource(obj *unstructured.Unstructured) (ResourceHealth, error) {
	// Check if service has endpoints
	endpoints := &corev1.Endpoints{}
	endpoints.SetName(obj.GetName())
	endpoints.SetNamespace(obj.GetNamespace())

	if err := h.client.Get(context.Background(), client.ObjectKeyFromObject(endpoints), endpoints); err != nil {
		return ResourceHealth{
			Healthy: false,
			Message: "Service has no endpoints",
		}, nil
	}

	// Count ready endpoints
	readyEndpoints := 0
	totalEndpoints := 0
	for _, subset := range endpoints.Subsets {
		readyEndpoints += len(subset.Addresses)
		totalEndpoints += len(subset.Addresses) + len(subset.NotReadyAddresses)
	}

	if readyEndpoints == 0 {
		return ResourceHealth{
			Healthy: false,
			Message: "Service has no ready endpoints",
			Ready:   0,
			Total:   totalEndpoints,
		}, nil
	}

	return ResourceHealth{
		Healthy: true,
		Message: fmt.Sprintf("Service has %d ready endpoints", readyEndpoints),
		Ready:   readyEndpoints,
		Total:   totalEndpoints,
	}, nil
}

func (h *HealthChecker) checkCustomResource(obj *unstructured.Unstructured, healthConditions []observabilityv1.HealthCondition) (ResourceHealth, error) {
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	
	// If no specific health conditions are defined, check for Ready condition
	if len(healthConditions) == 0 {
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			
			condType, _, _ := unstructured.NestedString(cond, "type")
			condStatus, _, _ := unstructured.NestedString(cond, "status")
			
			if condType == "Ready" && condStatus == "True" {
				return ResourceHealth{
					Healthy: true,
					Message: "Resource is ready",
				}, nil
			}
		}
		
		return ResourceHealth{
			Healthy: false,
			Message: "Resource is not ready",
		}, nil
	}

	// Check specific health conditions
	for _, hc := range healthConditions {
		found := false
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			
			condType, _, _ := unstructured.NestedString(cond, "type")
			if condType == hc.Type {
				found = true
				condStatus, _, _ := unstructured.NestedString(cond, "status")
				if condStatus != hc.Status {
					message, _, _ := unstructured.NestedString(cond, "message")
					return ResourceHealth{
						Healthy: false,
						Message: fmt.Sprintf("Condition %s is %s (expected %s): %s", 
							hc.Type, condStatus, hc.Status, message),
					}, nil
				}
				break
			}
		}
		
		if !found {
			return ResourceHealth{
				Healthy: false,
				Message: fmt.Sprintf("Condition %s not found", hc.Type),
			}, nil
		}
	}

	return ResourceHealth{
		Healthy: true,
		Message: "All health conditions met",
	}, nil
}

// Placeholder implementations for HTTP and metric checks
func (h *HealthChecker) checkHTTPEndpoint(ctx context.Context, check observabilityv1.HTTPCheck) (bool, error) {
	// TODO: Implement HTTP health check
	return true, nil
}

func (h *HealthChecker) checkMetric(ctx context.Context, metric observabilityv1.MetricCheck) (float64, error) {
	// TODO: Implement metric check via Prometheus
	return 0, nil
}

func (h *HealthChecker) evaluateMetricThreshold(value float64, threshold float64, operator string) bool {
	switch operator {
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "==":
		return value == threshold
	default:
		return false
	}
}

// Rollback implementation methods
func (m *RollbackManager) rollbackArgoCD(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, state *RollbackState) error {
	// TODO: Implement ArgoCD-specific rollback
	return nil
}

func (m *RollbackManager) rollbackFlux(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, state *RollbackState) error {
	// TODO: Implement Flux-specific rollback
	return nil
}

func (m *RollbackManager) rollbackGit(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, state *RollbackState) error {
	// TODO: Implement Git-based rollback
	return nil
}
