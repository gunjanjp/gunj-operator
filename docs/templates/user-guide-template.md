# [Feature/Component Name] User Guide

> **Version**: v1.0  
> **Last Updated**: YYYY-MM-DD  
> **Audience**: DevOps Engineers, Platform Engineers, SREs

## Overview

[Provide a brief introduction to the feature/component. Explain what it does and why users would want to use it. Keep it to 2-3 paragraphs.]

### Key Features

- **Feature 1**: Brief description
- **Feature 2**: Brief description  
- **Feature 3**: Brief description

### Use Cases

This guide is useful when you need to:
- [Use case 1]
- [Use case 2]
- [Use case 3]

## Prerequisites

Before you begin, ensure you have:

- [ ] Gunj Operator v2.0+ installed
- [ ] Kubernetes cluster v1.26+
- [ ] Appropriate RBAC permissions
- [ ] [Any other requirements]

### Required Permissions

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: platform-operator
rules:
  - apiGroups: ["observability.io"]
    resources: ["observabilityplatforms"]
    verbs: ["get", "list", "create", "update", "delete"]
```

## Quick Start

Follow these steps to get started quickly:

1. **Step 1**: [Action]
   ```bash
   kubectl apply -f example.yaml
   ```

2. **Step 2**: [Action]
   ```bash
   kubectl get observabilityplatforms
   ```

3. **Step 3**: [Action]
   ```bash
   kubectl describe observabilityplatform my-platform
   ```

## Configuration

### Basic Configuration

The simplest configuration looks like this:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
spec:
  # Basic configuration here
```

### Advanced Configuration

For production use, consider these additional options:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
spec:
  # Advanced configuration here
```

### Configuration Reference

| Field | Type | Description | Default | Required |
|-------|------|-------------|---------|----------|
| `spec.field1` | string | Description | `"default"` | No |
| `spec.field2` | integer | Description | `10` | Yes |
| `spec.field3` | boolean | Description | `false` | No |

## Common Tasks

### Task 1: [Creating Something]

To create [something], follow these steps:

1. Prepare your configuration file
2. Apply the configuration:
   ```bash
   kubectl apply -f config.yaml
   ```
3. Verify the creation:
   ```bash
   kubectl get [resource]
   ```

**Example**:
```yaml
# Complete example configuration
```

### Task 2: [Updating Something]

To update [something]:

1. Edit your configuration
2. Apply the changes:
   ```bash
   kubectl apply -f updated-config.yaml
   ```
3. Monitor the update:
   ```bash
   kubectl rollout status deployment/[name]
   ```

### Task 3: [Deleting Something]

To safely delete [something]:

1. Check dependencies:
   ```bash
   kubectl get [dependent-resources]
   ```
2. Delete the resource:
   ```bash
   kubectl delete observabilityplatform my-platform
   ```
3. Verify cleanup:
   ```bash
   kubectl get all -n [namespace]
   ```

## Best Practices

### Do's

- ✅ Always set resource limits
- ✅ Use namespaces for isolation
- ✅ Regular backups
- ✅ Monitor resource usage
- ✅ Follow naming conventions

### Don'ts

- ❌ Don't use default passwords
- ❌ Don't skip validation
- ❌ Don't ignore warnings
- ❌ Don't run as root
- ❌ Don't disable security features

### Production Recommendations

For production deployments:

1. **High Availability**: Deploy with at least 3 replicas
2. **Resource Allocation**: Set appropriate limits
   ```yaml
   resources:
     requests:
       memory: "4Gi"
       cpu: "2"
     limits:
       memory: "8Gi"
       cpu: "4"
   ```
3. **Persistence**: Use persistent volumes
4. **Security**: Enable all security features
5. **Monitoring**: Set up comprehensive monitoring

## Troubleshooting

### Common Issues

#### Issue 1: [Platform Not Starting]

**Symptoms**:
- Pods in `Pending` state
- Events show scheduling errors

**Cause**: Insufficient resources

**Solution**:
```bash
# Check node resources
kubectl top nodes

# Check resource requests
kubectl describe pod [pod-name]

# Adjust resource requests
kubectl edit observabilityplatform [name]
```

#### Issue 2: [Connection Errors]

**Symptoms**:
- Service unavailable errors
- Timeout errors

**Cause**: Network policy blocking traffic

**Solution**:
```bash
# Check network policies
kubectl get networkpolicies

# Test connectivity
kubectl exec -it [pod] -- curl [service]
```

### Debug Commands

Useful commands for debugging:

```bash
# Check operator logs
kubectl logs -n gunj-system deployment/gunj-operator

# Check platform status
kubectl get observabilityplatform -o wide

# Describe platform
kubectl describe observabilityplatform [name]

# Check events
kubectl get events --sort-by='.lastTimestamp'
```

### Getting Help

If you're still having issues:

1. Check the [FAQ](../faq.md)
2. Search [existing issues](https://github.com/gunjanjp/gunj-operator/issues)
3. Ask in [Slack channel](https://gunjanjp.slack.com)
4. Create a [new issue](https://github.com/gunjanjp/gunj-operator/issues/new)

## Examples

### Example 1: Development Setup

```yaml
# development-platform.yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: dev-platform
  namespace: development
spec:
  # Minimal resources for development
  components:
    prometheus:
      enabled: true
      replicas: 1
      resources:
        requests:
          memory: "512Mi"
          cpu: "250m"
```

### Example 2: Production Setup

```yaml
# production-platform.yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: prod-platform
  namespace: production
spec:
  # HA configuration for production
  components:
    prometheus:
      enabled: true
      replicas: 3
      resources:
        requests:
          memory: "4Gi"
          cpu: "2"
      storage:
        size: 100Gi
        class: fast-ssd
```

## Related Documentation

- [Architecture Overview](../architecture/overview.md)
- [API Reference](../api/rest-api.md)
- [Configuration Reference](configuration.md)
- [Security Guide](../operations/security.md)

## Feedback

Help us improve this guide:
- 📝 [Edit this page](https://github.com/gunjanjp/gunj-operator/edit/main/docs/user-guide/[filename].md)
- 🐛 [Report an issue](https://github.com/gunjanjp/gunj-operator/issues/new)
- 💡 [Suggest improvement](https://github.com/gunjanjp/gunj-operator/discussions)

---

**Need more help?** Join our [community Slack channel](https://gunjanjp.slack.com) or check our [support options](../support.md).
