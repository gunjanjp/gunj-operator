# Security Policy Enforcement

This package provides comprehensive security policy enforcement for the Gunj Operator, ensuring that all ObservabilityPlatform resources comply with security best practices and standards.

## Features

### 1. Pod Security Standards (PSS) Enforcement

The security validator enforces three levels of Pod Security Standards:

- **Privileged**: Minimal restrictions (not recommended for production)
- **Baseline**: Prevents known privilege escalations
- **Restricted**: Heavily restricted, following security best practices

### 2. Security Context Validation

#### Pod Security Context
- Enforces non-root user execution (UID >= 1000)
- Requires specific FSGroup settings
- Validates seccomp profiles
- Ensures proper user/group configurations

#### Container Security Context
- Prevents privilege escalation
- Enforces read-only root filesystem
- Validates capability management
- Requires dropping ALL capabilities (with limited exceptions)

### 3. Network Policy Enforcement

- Requires network policies to be defined
- Validates ingress and egress rules
- Prevents overly permissive configurations
- Ensures proper traffic isolation

### 4. Security Annotations

Required annotations:
- `security.gunj-operator.io/pod-security-level`: Specifies the PSS level
- `security.gunj-operator.io/compliance-profile`: Indicates compliance profile (cis, nist, pci-dss, etc.)

### 5. Environment Variable Security

- Detects sensitive data patterns in environment variables
- Enforces use of secrets for sensitive values
- Prevents plaintext passwords and API keys

## Usage

### Configuration

The security validator is automatically integrated with the webhook system. To configure:

```go
validator := security.NewSecurityValidator(client)
validator.DefaultSecurityLevel = security.PodSecurityLevelRestricted
validator.EnforceNonRoot = true
validator.NetworkPolicyRequired = true
```

### Platform Configuration Example

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production-platform
  annotations:
    security.gunj-operator.io/pod-security-level: "restricted"
    security.gunj-operator.io/compliance-profile: "cis"
spec:
  components:
    prometheus:
      enabled: true
      securityContext:
        podSecurityContext:
          runAsNonRoot: true
          runAsUser: 1000
          runAsGroup: 1000
          fsGroup: 1000
          seccompProfile:
            type: RuntimeDefault
        containerSecurityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          capabilities:
            drop:
            - ALL
          seccompProfile:
            type: RuntimeDefault
  security:
    networkPolicy:
      enabled: true
      ingress:
      - from:
        - namespaceSelector:
            matchLabels:
              name: gunj-system
      egress:
      - to:
        - namespaceSelector: {}
```

## Security Levels

### Restricted Level (Recommended)

Most secure configuration:
- Non-root execution mandatory
- Read-only root filesystem
- No privilege escalation
- All capabilities dropped (except NET_BIND_SERVICE if needed)
- Seccomp profiles required
- User/Group IDs >= 1000

### Baseline Level

Moderate security:
- Prevents privileged containers
- Blocks dangerous capabilities
- Prevents host namespace sharing
- Allows some flexibility for legacy applications

### Privileged Level

Minimal restrictions:
- Use only for special cases
- Not recommended for production
- Requires explicit justification

## Compliance Profiles

Supported compliance profiles:
- **cis**: CIS Kubernetes Benchmark
- **nist**: NIST Cybersecurity Framework
- **pci-dss**: PCI Data Security Standard
- **hipaa**: Health Insurance Portability and Accountability Act
- **soc2**: Service Organization Control 2
- **custom**: Custom security requirements

## Security Recommendations

The validator can generate security recommendations:

```go
recommendations := validator.GenerateSecurityRecommendations(platform)
// Returns suggestions like:
// - "Enable read-only root filesystem for Prometheus"
// - "Consider upgrading to 'restricted' security level"
// - "Enable network policies to control traffic"
```

## Testing

Run security validation tests:

```bash
go test ./internal/webhook/security/...
```

## Error Messages

Common validation errors:

1. **Missing Security Context**
   ```
   Prometheus must have security context defined for restricted security level
   ```

2. **Root User Execution**
   ```
   spec.components.prometheus.securityContext.podSecurityContext.runAsUser: 
   Invalid value: 0: must be >= 1000 for restricted security level
   ```

3. **Missing Network Policies**
   ```
   spec.security.networkPolicy: Required value: 
   network policy configuration required for security compliance
   ```

4. **Sensitive Environment Variables**
   ```
   spec.components.prometheus.extraEnvVars[0].value: Invalid value: "[REDACTED]": 
   environment variable 'ADMIN_PASSWORD' appears to contain sensitive data 
   and should use valueFrom.secretKeyRef instead of plaintext value
   ```

## Integration with CI/CD

Include security validation in your CI/CD pipeline:

```yaml
# .github/workflows/security.yml
- name: Validate Security Policies
  run: |
    kubectl apply --dry-run=server -f platform.yaml
```

## Best Practices

1. **Always use restricted level for production**
2. **Enable network policies for all platforms**
3. **Use secrets for sensitive data**
4. **Regularly review security recommendations**
5. **Keep security annotations up to date**
6. **Test security policies in development first**

## Support

For security-related questions or issues:
- Email: gunjanjp@gmail.com
- GitHub Issues: https://github.com/gunjanjp/gunj-operator/issues
- Security Vulnerabilities: gunjanjp@gmail.com (do not create public issues)
