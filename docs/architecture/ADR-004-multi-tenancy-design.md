# ADR-004: Multi-Tenancy Design

**Date**: 2025-06-12  
**Status**: Accepted  
**Context**: Gunj Operator Development  

## Context

The Gunj Operator needs to support multi-tenant deployments where multiple teams or projects can deploy their own observability platforms. We need to design for:
- Tenant isolation
- Resource quotas and limits
- Access control
- Cost allocation
- Noisy neighbor prevention
- Cross-tenant visibility (where allowed)

## Decision

We will implement **Namespace-based Soft Multi-tenancy** with:
- One namespace per tenant
- Namespace-scoped ObservabilityPlatform CRD
- RBAC for access control
- NetworkPolicies for network isolation
- Resource quotas per namespace
- Optional cross-namespace data federation

## Rationale

### Multi-tenancy Models Considered

1. **Cluster per Tenant**
   - Pros: Complete isolation, simple security
   - Cons: Expensive, complex management

2. **Namespace per Tenant** ✓
   - Pros: Good isolation, Kubernetes native, cost-effective
   - Cons: Shared kernel, some noisy neighbor risk

3. **Label-based Separation**
   - Pros: Flexible, easy data sharing
   - Cons: Weak isolation, complex RBAC

4. **Virtual Clusters** (vcluster)
   - Pros: Strong isolation, full Kubernetes API
   - Cons: Additional complexity, overhead

### Architecture

```yaml
# Tenant namespace structure
production-tenant-a/
├── ObservabilityPlatform (production-platform)
├── Prometheus
├── Grafana
├── Loki
└── Tempo

production-tenant-b/
├── ObservabilityPlatform (production-platform)
├── Prometheus
├── Grafana
├── Loki
└── Tempo

# Shared operator namespace
gunj-system/
├── gunj-operator (deployment)
├── gunj-api-server
└── gunj-ui
```

### Isolation Mechanisms

#### 1. RBAC Configuration
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: observability-platform-manager
  namespace: production-tenant-a
rules:
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
```

#### 2. Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tenant-isolation
  namespace: production-tenant-a
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: production-tenant-a
    - namespaceSelector:
        matchLabels:
          name: gunj-system
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: production-tenant-a
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
  - ports:
    - protocol: TCP
      port: 53  # DNS
```

#### 3. Resource Quotas
```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-quota
  namespace: production-tenant-a
spec:
  hard:
    requests.cpu: "100"
    requests.memory: "200Gi"
    requests.storage: "1Ti"
    persistentvolumeclaims: "20"
    services: "10"
    services.loadbalancers: "2"
```

### Tenant Configuration in CRD

```go
type ObservabilityPlatformSpec struct {
    // ... existing fields ...
    
    // Tenant configuration
    TenantSettings *TenantSettings `json:"tenantSettings,omitempty"`
}

type TenantSettings struct {
    // Tenant identifier
    TenantID string `json:"tenantId"`
    
    // Isolation level
    IsolationLevel string `json:"isolationLevel,omitempty"` // strict, moderate, relaxed
    
    // Cost allocation tags
    CostTags map[string]string `json:"costTags,omitempty"`
    
    // Data retention overrides
    DataRetention *DataRetentionPolicy `json:"dataRetention,omitempty"`
    
    // Cross-tenant data sharing
    DataSharing *DataSharingPolicy `json:"dataSharing,omitempty"`
}
```

### Cross-Tenant Features

#### 1. Global View (Admin Only)
```go
// API endpoint for global view
GET /api/v1/platforms?all-namespaces=true

// Requires cluster-wide permissions
```

#### 2. Federated Queries
```yaml
# Grafana datasource for cross-tenant Prometheus
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
data:
  prometheus.yaml: |
    datasources:
    - name: Prometheus-Federation
      type: prometheus
      url: http://prometheus-global.gunj-system:9090
      jsonData:
        httpMethod: POST
```

#### 3. Shared Dashboards
```yaml
# Shared dashboard ConfigMap in gunj-system
apiVersion: v1
kind: ConfigMap
metadata:
  name: shared-dashboards
  namespace: gunj-system
  labels:
    gunj.io/shared-resource: "true"
```

## Consequences

### Positive
- Good isolation with Kubernetes primitives
- Cost-effective resource utilization
- Flexible access control
- Native Kubernetes integration
- Easy tenant onboarding

### Negative
- Shared kernel risks
- Namespace proliferation
- Complex RBAC management
- Limited hard isolation

### Mitigation
- Regular security audits
- Automated RBAC management
- Resource monitoring and alerts
- Clear tenant SLAs

## Implementation Checklist

- [ ] Namespace creation automation
- [ ] RBAC template per tenant type
- [ ] NetworkPolicy templates
- [ ] ResourceQuota templates
- [ ] Tenant onboarding automation
- [ ] Cost allocation reporting
- [ ] Cross-tenant query federation
- [ ] Audit logging per tenant
- [ ] Tenant resource dashboard

## Security Considerations

1. **Pod Security Standards**
   ```yaml
   apiVersion: v1
   kind: Namespace
   metadata:
     name: production-tenant-a
     labels:
       pod-security.kubernetes.io/enforce: restricted
       pod-security.kubernetes.io/audit: restricted
       pod-security.kubernetes.io/warn: restricted
   ```

2. **Secrets Management**
   - Each tenant manages own secrets
   - No cross-namespace secret access
   - Encryption at rest enabled

3. **Audit Logging**
   - All tenant API actions logged
   - Separate audit streams per tenant
   - Compliance reporting support

## References
- [Kubernetes Multi-tenancy SIG](https://github.com/kubernetes-sigs/multi-tenancy)
- [Namespace Security Best Practices](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [NetworkPolicy Recipes](https://github.com/ahmetb/kubernetes-network-policy-recipes)
- [Resource Quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/)
