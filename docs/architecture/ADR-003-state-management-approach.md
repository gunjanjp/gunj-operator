# ADR-003: State Management Approach

**Date**: 2025-06-12  
**Status**: Accepted  
**Context**: Gunj Operator Development  

## Context

Kubernetes operators need to manage state effectively to ensure reliable reconciliation. We need to decide how the Gunj Operator will manage state for:
- Platform lifecycle (Pending → Installing → Ready → Failed → Upgrading)
- Component status tracking
- Configuration drift detection
- Failure recovery
- Upgrade coordination

## Decision

We will use **Kubernetes Resources as the Single Source of Truth** with the following approach:
- All state stored in CRD status subresource
- No in-memory state between reconciliations
- Event-driven reconciliation only
- Deterministic state transitions

## Rationale

### Options Considered

1. **In-Memory State Cache**
   - Pros: Fast access, complex state machines
   - Cons: Lost on restart, complexity, drift issues

2. **External Database** (PostgreSQL/etcd)
   - Pros: Rich queries, complex state
   - Cons: Additional dependency, complexity

3. **ConfigMaps/Secrets for State**
   - Pros: Native Kubernetes
   - Cons: Not designed for this, no transactions

4. **CRD Status Only** ✓
   - Pros: Kubernetes native, survives restarts, versioned
   - Cons: Limited by K8s API

### State Storage Design

```go
type ObservabilityPlatformStatus struct {
    // Core state
    Phase              string                        `json:"phase,omitempty"`
    Message            string                        `json:"message,omitempty"`
    ObservedGeneration int64                         `json:"observedGeneration,omitempty"`
    LastReconcileTime  *metav1.Time                  `json:"lastReconcileTime,omitempty"`
    
    // Component tracking
    ComponentStatus    map[string]ComponentStatus    `json:"componentStatus,omitempty"`
    
    // Kubernetes standard conditions
    Conditions         []metav1.Condition            `json:"conditions,omitempty"`
    
    // Operation tracking
    CurrentOperation   *OperationStatus              `json:"currentOperation,omitempty"`
    
    // Resource tracking
    Resources          map[string]ResourceReference `json:"resources,omitempty"`
}
```

### State Transition Rules

1. **Idempotent Transitions**: Same input → same output
2. **No Hidden State**: Everything visible in status
3. **Atomic Updates**: Use status subresource
4. **Optimistic Concurrency**: Use resourceVersion

### Reconciliation Pattern

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch current state
    platform := &v1alpha1.ObservabilityPlatform{}
    if err := r.Get(ctx, req.NamespacedName, platform); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // 2. Make decisions based on current state
    desired := r.computeDesiredState(platform)
    
    // 3. Apply changes
    if err := r.applyDesiredState(ctx, platform, desired); err != nil {
        return ctrl.Result{}, err
    }
    
    // 4. Update status
    if err := r.updateStatus(ctx, platform); err != nil {
        return ctrl.Result{}, err
    }
    
    // 5. Requeue if needed
    return r.determineRequeue(platform), nil
}
```

### Failure Recovery

```yaml
status:
  phase: Failed
  lastFailure:
    time: "2025-06-12T10:00:00Z"
    reason: "PrometheusDeploymentFailed"
    message: "Failed to create Prometheus deployment: insufficient resources"
    retryCount: 3
  conditions:
  - type: Ready
    status: "False"
    reason: ComponentFailure
    message: "Prometheus deployment failed"
```

## Consequences

### Positive
- Survives operator restarts
- No state synchronization issues
- Kubernetes-native debugging
- Built-in versioning and history
- Works with GitOps

### Negative
- Limited by Kubernetes API rate limits
- Status size limitations (1MB)
- No complex queries
- Status update conflicts possible

### Mitigation
- Efficient status updates (patch vs update)
- Status compression for large states
- Proper error handling for conflicts
- Rate limiting awareness

## Implementation Guidelines

### Status Update Pattern
```go
// Use patches for efficient updates
patch := client.MergeFrom(platform.DeepCopy())
platform.Status.Phase = "Ready"
if err := r.Status().Patch(ctx, platform, patch); err != nil {
    if errors.IsConflict(err) {
        // Requeue for retry
        return ctrl.Result{Requeue: true}, nil
    }
    return ctrl.Result{}, err
}
```

### Condition Management
```go
// Use standard Kubernetes conditions
meta.SetStatusCondition(&platform.Status.Conditions, metav1.Condition{
    Type:               "Ready",
    Status:             metav1.ConditionTrue,
    ObservedGeneration: platform.Generation,
    LastTransitionTime: metav1.Now(),
    Reason:             "AllComponentsReady",
    Message:            "All observability components are running",
})
```

### Resource Tracking
```go
// Track created resources for cleanup
platform.Status.Resources["prometheus-deployment"] = ResourceReference{
    APIVersion: "apps/v1",
    Kind:       "Deployment",
    Name:       "prometheus-server",
    Namespace:  platform.Namespace,
    UID:        deployment.UID,
}
```

## References
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Status Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#status-subresource)
- [Conditions Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)
