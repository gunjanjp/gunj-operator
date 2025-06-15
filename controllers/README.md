# Controllers Package

This package contains the Kubernetes controllers for the Gunj Operator.

## ObservabilityPlatformReconciler

The main controller that manages ObservabilityPlatform resources.

### Features

- **Controller-Runtime Client Integration**: Full integration with controller-runtime client for Kubernetes API operations
- **Reconciliation Loop**: Implements idempotent reconciliation logic
- **Finalizer Handling**: Proper cleanup on resource deletion
- **Status Management**: Updates resource status with current phase and conditions
- **Event Recording**: Records Kubernetes events for significant actions
- **Metrics Collection**: Prometheus metrics for monitoring operator performance
- **Component Management**: Delegates to specialized managers for each component
- **Error Handling**: Comprehensive error handling with exponential backoff
- **Leader Election**: Support for high availability deployments
- **Namespace Management**: Creates and manages namespaces with proper labels

### Controller Client Usage

The controller uses the controller-runtime client for all Kubernetes API operations:

```go
// Create a resource
err := r.Create(ctx, resource)

// Get a resource
err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, resource)

// Update a resource
err := r.Update(ctx, resource)

// Update status
err := r.Status().Update(ctx, resource)

// Delete a resource
err := r.Delete(ctx, resource)

// List resources
list := &corev1.ServiceList{}
err := r.List(ctx, list, client.InNamespace(namespace))

// Patch a resource
patch := client.MergeFrom(original.DeepCopy())
err := r.Patch(ctx, modified, patch)
```

### Reconciliation Flow

1. **Fetch Resource**: Get the ObservabilityPlatform resource
2. **Handle Deletion**: Check for deletion timestamp and process cleanup
3. **Add Finalizer**: Ensure finalizer is present for cleanup
4. **Check Paused**: Skip reconciliation if paused
5. **Ensure Namespace**: Create namespace if needed
6. **Reconcile Components**: Process each enabled component
7. **Update Status**: Update resource status and conditions
8. **Requeue**: Schedule next reconciliation

### Component Managers

The controller delegates component-specific logic to managers:

- `PrometheusManager`: Manages Prometheus StatefulSets
- `GrafanaManager`: Manages Grafana Deployments
- `LokiManager`: Manages Loki StatefulSets
- `TempoManager`: Manages Tempo Deployments

### Metrics

The controller exposes the following metrics:

- `gunj_operator_reconcile_total`: Total reconciliations
- `gunj_operator_reconcile_errors_total`: Failed reconciliations
- `gunj_operator_reconcile_duration_seconds`: Reconciliation duration
- `gunj_operator_platforms_total`: Platforms by phase
- `gunj_operator_component_status`: Component readiness status

### Testing

Run controller tests:

```bash
go test ./controllers/... -v
```

Run with Ginkgo:

```bash
ginkgo ./controllers/...
```

### RBAC Requirements

The controller requires the following RBAC permissions:

- Full access to ObservabilityPlatform resources
- Create/update/delete for workloads (Deployments, StatefulSets)
- Create/update/delete for configuration (ConfigMaps, Secrets)
- Create/update/delete for networking (Services, Ingresses)
- Create events for audit trail
- Read namespaces for multi-namespace support
