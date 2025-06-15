# Resource Quota Validation

## Overview

The Gunj Operator includes comprehensive resource quota validation to ensure that ObservabilityPlatform deployments respect namespace and cluster-wide resource limits. This feature helps prevent resource exhaustion, supports multi-tenant environments, and provides clear feedback when quota limits would be exceeded.

## Features

### 1. Namespace Quota Validation

The operator automatically checks against Kubernetes ResourceQuota objects in the namespace:

- **CPU and Memory**: Validates requested resources against available quota
- **Storage**: Checks PVC storage requirements
- **Object Counts**: Validates pod and PVC counts
- **Automatic Detection**: Discovers and validates against all ResourceQuotas in the namespace

### 2. Cluster-Wide Resource Limits

Administrators can configure cluster-wide limits through the operator configuration:

```go
ClusterResourceLimits: &quota.ClusterResourceLimits{
    MaxCPUPerNamespace:    resource.MustParse("100"),     // 100 CPU cores per namespace
    MaxMemoryPerNamespace: resource.MustParse("200Gi"),   // 200Gi memory per namespace
    MaxStoragePerPVC:      resource.MustParse("1Ti"),     // 1Ti max per PVC
    MaxPVCPerNamespace:    50,                            // 50 PVCs per namespace
    MaxPodsPerNamespace:   100,                           // 100 pods per namespace
}
```

### 3. Multi-Tenant Support

The validator includes special handling for multi-tenant environments:

- **Tenant Isolation**: Validates nodeSelector requirements for isolated tenants
- **Platform Limits**: Enforces maximum platforms per tenant
- **Resource Fairness**: Prevents any namespace from using more than 3x the average resources
- **Tenant Labels**: Recognizes and enforces tenant-specific policies

### 4. Resource Calculation

The validator accurately calculates total resource requirements:

- **Component Resources**: CPU, memory for each component × replicas
- **Storage Requirements**: PVC sizes for each component
- **Pod Counts**: Total pods across all components
- **PVC Counts**: Accounts for per-replica PVCs (e.g., Prometheus)

### 5. Update Validation

During updates, the validator:

- Only validates quota if resources are being increased
- Checks for new components being added
- Validates scaling operations
- Provides warnings for resource increases

## Error Messages

The validator provides detailed, actionable error messages:

```
spec.components: exceeds available CPU quota in namespace monitoring. 
Required: 1500m, Available: 800m, Quota: 2
```

```
spec.components: exceeds cluster CPU limit for namespace monitoring. 
Required: 150, Available: 50, Limit: 100
```

## Configuration Examples

### Basic Platform with Quota

```yaml
# ResourceQuota in namespace
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
  namespace: monitoring
spec:
  hard:
    requests.cpu: "10"
    requests.memory: "20Gi"
    requests.storage: "100Gi"
    persistentvolumeclaims: "10"
    pods: "50"

---
# ObservabilityPlatform that respects quota
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      replicas: 2
      resources:
        requests:
          cpu: "500m"
          memory: "2Gi"
      storage:
        size: "10Gi"
```

### Multi-Tenant Platform

```yaml
# Namespace with tenant label
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-a-prod
  labels:
    tenant: tenant-a
    require-isolation: "true"

---
# Platform with required nodeSelector
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: tenant-a-platform
  namespace: tenant-a-prod
spec:
  nodeSelector:
    tenant: tenant-a
  components:
    prometheus:
      enabled: true
```

## Quota Summary

The validator can provide a helpful summary of current quota usage:

```
Resource Quota Summary for namespace 'monitoring':
----------------------------------------

Quota: compute-quota
  CPU: 4500m / 10 (45.0% used)
  Memory: 8Gi / 20Gi (40.0% used)
  Storage: 35Gi / 100Gi (35.0% used)
  Pods: 12 / 50 (24.0% used)
  PVCs: 4 / 10 (40.0% used)
```

## Best Practices

1. **Set Appropriate Quotas**: Define ResourceQuotas that reflect actual capacity
2. **Monitor Usage**: Regularly review quota utilization
3. **Plan for Growth**: Leave headroom for scaling operations
4. **Use Defaults**: Leverage the operator's intelligent defaulting
5. **Test Updates**: Validate that updates won't exceed quotas before applying

## Troubleshooting

### Platform Creation Fails Due to Quota

1. Check current quota usage:
   ```bash
   kubectl describe resourcequota -n <namespace>
   ```

2. Review platform resource requirements:
   ```bash
   kubectl describe observabilityplatform <platform-name> -n <namespace>
   ```

3. Either:
   - Reduce resource requests in the platform spec
   - Clean up unused resources in the namespace
   - Request quota increase from administrators

### Multi-Tenant Validation Fails

1. Ensure namespace has correct labels:
   ```bash
   kubectl get namespace <namespace> -o yaml
   ```

2. Add required nodeSelector to platform spec:
   ```yaml
   spec:
     nodeSelector:
       tenant: <tenant-name>
   ```

### Fairness Policy Violations

If you receive a "resource allocation violates fairness policy" error:

1. Review namespace resource usage across the cluster
2. Consider distributing workloads more evenly
3. Request exemption from fairness policy if justified

## Implementation Details

The resource quota validation is implemented in:
- `internal/webhook/quota/resource_quota_validator.go`
- Integrated into the validating webhook
- Runs during both Create and Update operations
- Configurable cluster limits via webhook setup

## Future Enhancements

- Dynamic quota recommendations based on historical usage
- Integration with cluster autoscaler for quota adjustments
- Quota reservation system for planned deployments
- Cross-namespace quota sharing for organizations
