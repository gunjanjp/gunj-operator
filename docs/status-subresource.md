# Status Subresource Implementation

This document describes the status subresource implementation for the Gunj Operator's ObservabilityPlatform CRD.

## Overview

The status subresource is a Kubernetes feature that allows the status of a resource to be updated independently from its spec. This separation provides several benefits:

1. **Security**: Different RBAC permissions can be applied to spec and status
2. **Consistency**: Only controllers should update status, while users update spec
3. **Conflict Reduction**: Spec and status updates don't conflict with each other
4. **Audit Trail**: Status changes can be tracked separately

## Implementation Details

### CRD Configuration

The status subresource is enabled using the Kubebuilder marker:

```go
// +kubebuilder:subresource:status
```

This marker generates the necessary OpenAPI schema with a status subresource in the CRD.

### Status Structure

The ObservabilityPlatformStatus includes:

```go
type ObservabilityPlatformStatus struct {
    // Phase represents the current state
    Phase string
    
    // ObservedGeneration for tracking spec changes
    ObservedGeneration int64
    
    // Conditions for detailed state information
    Conditions []metav1.Condition
    
    // ComponentStatus for individual component tracking
    ComponentStatus map[string]ComponentStatus
    
    // Health for overall platform health
    Health HealthStatus
    
    // Other fields...
}
```

### Status Helper Methods

Helper methods are provided in `observabilityplatform_status.go`:

#### Phase Management
```go
// Update the platform phase
platform.UpdatePhase("Ready", "All components deployed")

// Check phase
if platform.IsReady() {
    // Platform is ready
}
```

#### Condition Management
```go
// Set a condition
platform.SetReadyCondition(metav1.ConditionTrue, "Ready", "All components ready")

// Get a condition
condition := platform.GetCondition("Ready")
```

#### Component Status
```go
// Update component status
platform.UpdateComponentStatus("prometheus", ComponentStatus{
    Phase: "Ready",
    Version: "v2.48.0",
    Ready: 3,
})

// Update overall health
platform.UpdateHealth()
```

### Status Updater Utility

The `status.Updater` provides safe status updates with retry logic:

```go
// Create updater
updater := status.NewUpdater(client)

// Update status with retry on conflicts
err := updater.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
    p.UpdatePhase("Installing", "Installing components")
    p.UpdateComponentStatus("prometheus", status)
})

// Convenience methods
err := updater.SetPhaseV1Beta1(ctx, platform, "Ready", "Installation complete")
err := updater.SetConditionV1Beta1(ctx, platform, "Ready", metav1.ConditionTrue, "Ready", "All good")
```

## RBAC Configuration

### Operator Permissions

The operator needs full access to the status subresource:

```yaml
- apiGroups:
  - observability.io
  resources:
  - observabilityplatforms/status
  verbs:
  - get
  - patch
  - update
```

### User Permissions

Users typically need read-only access to status:

```yaml
- apiGroups:
  - observability.io
  resources:
  - observabilityplatforms/status
  verbs:
  - get
```

### GitOps Permissions

GitOps tools should only manage spec, not status:

```yaml
# Can manage the resource
- apiGroups:
  - observability.io
  resources:
  - observabilityplatforms
  verbs:
  - create
  - update
  - patch
  - delete

# But only read status
- apiGroups:
  - observability.io
  resources:
  - observabilityplatforms/status
  verbs:
  - get
```

## Usage in Controllers

### Reconciliation Pattern

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    platform := &v1beta1.ObservabilityPlatform{}
    if err := r.Get(ctx, req.NamespacedName, platform); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    
    // Initialize status if needed
    if platform.Status.Phase == "" {
        platform.InitializeStatus()
    }
    
    // Mark as reconciling
    if err := r.statusUpdater.MarkReconcilingV1Beta1(ctx, platform); err != nil {
        return ctrl.Result{}, err
    }
    
    // Perform reconciliation...
    
    // Update status based on results
    if err := r.statusUpdater.UpdateObservabilityPlatformV1Beta1(ctx, platform, func(p *v1beta1.ObservabilityPlatform) {
        p.UpdatePhase("Ready", "All components deployed")
        p.SetReadyCondition(metav1.ConditionTrue, "Ready", "Platform is ready")
        p.UpdateHealth()
    }); err != nil {
        return ctrl.Result{}, err
    }
    
    return ctrl.Result{}, nil
}
```

### Component Status Updates

```go
// Update component after deployment
func (r *Reconciler) updateComponentStatus(ctx context.Context, platform *v1beta1.ObservabilityPlatform, component string) error {
    // Get component deployment status
    deployment := r.getDeployment(component)
    
    status := v1beta1.ComponentStatus{
        Phase:    "Ready",
        Version:  deployment.Labels["version"],
        Replicas: *deployment.Spec.Replicas,
        Ready:    deployment.Status.ReadyReplicas,
    }
    
    return r.statusUpdater.UpdateComponentStatusV1Beta1(ctx, platform, component, status)
}
```

## Status Phases

The platform can be in one of these phases:

1. **Pending**: Initial state, resources not yet created
2. **Installing**: Components are being deployed
3. **Ready**: All components are running successfully
4. **Failed**: One or more critical components failed
5. **Upgrading**: Platform is being upgraded
6. **Degraded**: Some components are unhealthy but platform is functional

## Health Status

Health is calculated based on component states:

- **Healthy**: All components are ready
- **Degraded**: Some components are not ready
- **Unhealthy**: Critical components are not ready
- **Unknown**: Unable to determine health

## Best Practices

1. **Always Use Status Updater**: Don't update status directly, use the updater for proper retry handling

2. **Update ObservedGeneration**: Always set ObservedGeneration to track spec changes

3. **Use Conditions**: Provide detailed state information through conditions

4. **Atomic Updates**: Update all related status fields in a single update

5. **Meaningful Messages**: Provide clear, actionable status messages

6. **Regular Health Checks**: Update health status after component changes

7. **Handle Conflicts**: Expect and handle update conflicts gracefully

## Testing

Test the status subresource with:

```go
// Create fake client with status subresource
client := fake.NewClientBuilder().
    WithScheme(scheme).
    WithStatusSubresource(&v1beta1.ObservabilityPlatform{}).
    Build()

// Test status updates
err := client.Status().Update(ctx, platform)
```

## Troubleshooting

### Common Issues

1. **Status Not Updating**: Check RBAC permissions for status subresource

2. **Update Conflicts**: Use the status updater with retry logic

3. **Missing Status Fields**: Ensure InitializeStatus is called

4. **Stale ObservedGeneration**: Always update ObservedGeneration in status updates

### Debugging

Enable verbose logging to see status updates:

```bash
kubectl logs -n gunj-system deployment/gunj-operator -f | grep -i status
```

Check current status:

```bash
kubectl get observabilityplatform my-platform -o jsonpath='{.status}' | jq .
```
