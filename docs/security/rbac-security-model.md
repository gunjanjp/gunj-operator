# RBAC and Security Model

## Overview

This document defines the Role-Based Access Control (RBAC) and security model for the Gunj Operator, ensuring secure multi-tenant operation and defense in depth.

## Security Principles

1. **Principle of Least Privilege**: Grant minimum required permissions
2. **Defense in Depth**: Multiple layers of security controls
3. **Zero Trust**: Verify everything, trust nothing
4. **Audit Everything**: Comprehensive logging and monitoring
5. **Secure by Default**: Security enabled out-of-the-box

## RBAC Architecture

### Personas and Access Levels

#### 1. Cluster Administrator
- Full access to operator and all platforms
- Can manage operator configuration
- Access to all namespaces

#### 2. Platform Administrator
- Manage ObservabilityPlatform resources
- Configure components within assigned namespaces
- View logs and metrics

#### 3. Platform Viewer
- Read-only access to platforms
- View dashboards and metrics
- No configuration changes

#### 4. Developer
- View metrics and logs for their applications
- Create ServiceMonitors and PrometheusRules
- Limited to specific namespaces

### Operator Service Account

```yaml
# Operator service account with cluster-wide permissions
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gunj-operator
  namespace: gunj-system
  labels:
    app.kubernetes.io/name: gunj-operator
    app.kubernetes.io/instance: gunj-operator
    app.kubernetes.io/component: controller
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gunj-operator-role
rules:
# ObservabilityPlatform management
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms/status"]
  verbs: ["get", "update", "patch"]
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms/finalizers"]
  verbs: ["update"]

# Core resources
- apiGroups: [""]
  resources: ["services", "endpoints", "pods", "configmaps", "secrets", "serviceaccounts", "persistentvolumeclaims"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Apps resources
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Networking resources
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# RBAC resources (for creating component service accounts)
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["roles", "rolebindings"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Events
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]

# Pod logs (for troubleshooting)
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: gunj-operator-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gunj-operator-role
subjects:
- kind: ServiceAccount
  name: gunj-operator
  namespace: gunj-system
```

### User-Facing Roles

#### Platform Administrator Role
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gunj-platform-admin
  labels:
    app.kubernetes.io/name: gunj-operator
    rbac.gunj.io/aggregate-to-admin: "true"
rules:
# Full access to ObservabilityPlatforms
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Read access to component resources
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["services", "configmaps", "persistentvolumeclaims", "pods"]
  verbs: ["get", "list", "watch"]

# Access to logs
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get", "list"]

# Access to metrics
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]
```

#### Platform Viewer Role
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gunj-platform-viewer
  labels:
    app.kubernetes.io/name: gunj-operator
    rbac.gunj.io/aggregate-to-view: "true"
rules:
# Read-only access to ObservabilityPlatforms
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms"]
  verbs: ["get", "list", "watch"]

# Read access to component resources
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list"]
- apiGroups: [""]
  resources: ["services", "configmaps", "pods"]
  verbs: ["get", "list"]
```

#### Developer Role
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gunj-developer
  labels:
    app.kubernetes.io/name: gunj-operator
rules:
# Create monitoring resources
- apiGroups: ["monitoring.coreos.com"]
  resources: ["servicemonitors", "prometheusrules"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Read platform status
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms"]
  verbs: ["get", "list"]

# Access to metrics and logs
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get", "list"]
```

### Component Service Accounts

Each component runs with its own service account:

#### Prometheus Service Account
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: {{ .Namespace }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus-{{ .Namespace }}
rules:
- apiGroups: [""]
  resources: ["nodes", "nodes/proxy", "services", "endpoints", "pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["extensions", "networking.k8s.io"]
  resources: ["ingresses"]
  verbs: ["get", "list", "watch"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
```

#### Grafana Service Account
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: grafana
  namespace: {{ .Namespace }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: grafana
  namespace: {{ .Namespace }}
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
```

## Security Policies

### Pod Security Standards

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production-tenant-a
  labels:
    # Enforce restricted security standard
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### Pod Security Policy (if enabled)

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: gunj-restricted
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
  - ALL
  volumes:
  - 'configMap'
  - 'emptyDir'
  - 'projected'
  - 'secret'
  - 'downwardAPI'
  - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
```

### Network Policies

#### Default Deny All
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: {{ .Namespace }}
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
```

#### Component-Specific Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: prometheus-network-policy
  namespace: {{ .Namespace }}
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: prometheus
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app.kubernetes.io/name: grafana
    - podSelector:
        matchLabels:
          app.kubernetes.io/name: gunj-operator
    ports:
    - protocol: TCP
      port: 9090
  egress:
  - to:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 443
```

## Secrets Management

### Secret Types

1. **Component Credentials**
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: grafana-admin
     namespace: {{ .Namespace }}
   type: Opaque
   data:
     admin-user: YWRtaW4=  # base64 encoded
     admin-password: {{ .GeneratedPassword | b64enc }}
   ```

2. **External Storage Credentials**
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: s3-credentials
     namespace: {{ .Namespace }}
   type: Opaque
   data:
     access-key-id: {{ .AccessKeyID | b64enc }}
     secret-access-key: {{ .SecretAccessKey | b64enc }}
   ```

3. **TLS Certificates**
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: platform-tls
     namespace: {{ .Namespace }}
   type: kubernetes.io/tls
   data:
     tls.crt: {{ .Certificate | b64enc }}
     tls.key: {{ .PrivateKey | b64enc }}
   ```

### Secret Rotation

```go
// Automatic secret rotation
func (r *SecretRotator) RotateSecrets(ctx context.Context, platform *v1alpha1.ObservabilityPlatform) error {
    secrets := []string{"grafana-admin", "prometheus-remote-write"}
    
    for _, secretName := range secrets {
        secret := &corev1.Secret{}
        if err := r.Get(ctx, types.NamespacedName{
            Name:      secretName,
            Namespace: platform.Namespace,
        }, secret); err != nil {
            continue
        }
        
        // Check rotation annotation
        if lastRotation, ok := secret.Annotations["gunj.io/last-rotation"]; ok {
            rotationTime, _ := time.Parse(time.RFC3339, lastRotation)
            if time.Since(rotationTime) > 90*24*time.Hour { // 90 days
                // Rotate secret
                if err := r.rotateSecret(ctx, secret); err != nil {
                    return err
                }
            }
        }
    }
    
    return nil
}
```

## Audit Logging

### Audit Configuration

```yaml
apiVersion: audit.k8s.io/v1
kind: Policy
metadata:
  name: gunj-operator-audit-policy
rules:
# Log all operations on ObservabilityPlatforms
- level: RequestResponse
  omitStages:
  - RequestReceived
  resources:
  - group: "observability.io"
    resources: ["observabilityplatforms"]
  namespaces: ["*"]

# Log secret access
- level: Metadata
  omitStages:
  - RequestReceived
  resources:
  - group: ""
    resources: ["secrets"]
  namespaces: ["*"]

# Log RBAC changes
- level: RequestResponse
  omitStages:
  - RequestReceived
  resources:
  - group: "rbac.authorization.k8s.io"
    resources: ["roles", "rolebindings", "clusterroles", "clusterrolebindings"]
```

### Audit Event Processing

```go
type AuditEvent struct {
    Timestamp   time.Time         `json:"timestamp"`
    User        string            `json:"user"`
    Action      string            `json:"action"`
    Resource    string            `json:"resource"`
    Namespace   string            `json:"namespace"`
    Success     bool              `json:"success"`
    Reason      string            `json:"reason,omitempty"`
    Details     map[string]string `json:"details,omitempty"`
}

func (a *AuditLogger) LogEvent(event AuditEvent) {
    // Send to audit backend
    // Could be: Elasticsearch, Splunk, CloudWatch, etc.
}
```

## Security Scanning

### Image Scanning Policy

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: image-policy
  namespace: gunj-system
data:
  policy.yaml: |
    rules:
    - pattern: "*"
      requirements:
        maxSeverity: HIGH
        maxFixable: 10
        blockedCVEs: []
    - pattern: "prometheus:*"
      requirements:
        maxSeverity: CRITICAL
        allowedLicenses: ["Apache-2.0", "MIT"]
```

### Runtime Security

Using Falco rules for runtime protection:

```yaml
- rule: Unauthorized Process in Container
  desc: Detect unauthorized process execution
  condition: >
    spawned_process and container and container.id != host and
    proc.name != in (allowed_processes) and
    not trusted_containers
  output: >
    Unauthorized process started in container
    (user=%user.name container_id=%container.id container_name=%container.name
     process=%proc.name parent=%proc.pname)
  priority: WARNING
  tags: [container, process]
```

## Compliance

### CIS Kubernetes Benchmark

Compliance checks for:
- API server configuration
- Controller manager settings
- Scheduler configuration
- etcd security
- Network policies
- Pod security standards

### NIST Controls

Mapping to NIST 800-53 controls:
- AC-2: Account Management
- AC-3: Access Enforcement
- AU-2: Audit Events
- SC-7: Boundary Protection
- SI-3: Malicious Code Protection

## Security Hardening Checklist

- [ ] Enable RBAC
- [ ] Configure Pod Security Standards
- [ ] Deploy Network Policies
- [ ] Enable audit logging
- [ ] Configure secret encryption at rest
- [ ] Enable admission controllers
- [ ] Set resource quotas
- [ ] Configure TLS for all communications
- [ ] Enable image scanning
- [ ] Deploy runtime security monitoring
- [ ] Regular security assessments
- [ ] Incident response plan

## References

- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [RBAC Good Practices](https://kubernetes.io/docs/concepts/security/rbac-good-practices/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
