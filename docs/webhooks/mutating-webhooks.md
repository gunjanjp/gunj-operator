# Mutating Webhooks Documentation

## Overview

The Gunj Operator implements mutating admission webhooks to automatically set sensible defaults for ObservabilityPlatform resources. This ensures consistent configurations across deployments and reduces the amount of boilerplate YAML users need to write.

## What are Mutating Webhooks?

Mutating admission webhooks are HTTP callbacks that receive admission requests and can modify objects before they are persisted to the Kubernetes API. The Gunj Operator uses these webhooks to:

1. Set default values for optional fields
2. Add default labels and annotations
3. Normalize resource specifications
4. Set sensible defaults for components
5. Auto-generate secure passwords
6. Apply organization-specific defaults

## Default Values Applied

### Metadata Defaults

#### Labels
- `app.kubernetes.io/managed-by`: `gunj-operator`
- `app.kubernetes.io/name`: `observability-platform`
- `app.kubernetes.io/instance`: `<platform-name>`
- `app.kubernetes.io/version`: `v2.0.0`

#### Annotations
- `observability.io/created-by`: `gunj-operator`

### Component Defaults

#### Prometheus
- **Version**: `v2.48.0`
- **Replicas**: `1` (or `3` if HA is enabled)
- **Resources**:
  - Requests: `250m` CPU, `512Mi` memory
  - Limits: `1` CPU, `2Gi` memory
- **Storage**:
  - Size: `10Gi`
  - Retention: `15d`
- **External Labels**:
  - `cluster`: `default`

#### Grafana
- **Version**: `10.2.0`
- **Replicas**: `1` (or `2` if HA is enabled)
- **Admin User**: `admin`
- **Admin Password**: Auto-generated 16-character secure password
- **Resources**:
  - Requests: `100m` CPU, `256Mi` memory
  - Limits: `500m` CPU, `1Gi` memory
- **Persistence**:
  - Enabled: `true`
  - Size: `1Gi`

#### Loki
- **Version**: `2.9.0`
- **Replicas**: `1`
- **Resources**:
  - Requests: `100m` CPU, `256Mi` memory
  - Limits: `500m` CPU, `1Gi` memory
- **Storage**:
  - Size: `10Gi`
- **Retention**:
  - Days: `7`
  - Deletes Enabled: `true`
  - Compaction Interval: `10m`

#### Tempo
- **Version**: `2.3.0`
- **Replicas**: `1`
- **Resources**:
  - Requests: `100m` CPU, `256Mi` memory
  - Limits: `500m` CPU, `1Gi` memory
- **Storage**:
  - Size: `5Gi`

### Global Settings Defaults

- **Log Level**: `info`
- **External Labels**:
  - `organization`: `default`
  - `environment`: Inferred from namespace:
    - `production` or `prod` → `production`
    - `staging` or `stage` → `staging`
    - `development` or `dev` → `development`
    - Others → `default`
- **Retention Policies**:
  - Metrics: `30d`
  - Logs: `7d`
  - Traces: `3d`

### High Availability Defaults

When `highAvailability.enabled` is `true`:
- Prometheus replicas are set to minimum `3`
- Grafana replicas are set to minimum `2`
- Anti-affinity type is set to `soft`

### Backup Defaults

When `backup.enabled` is `true`:
- **Schedule**: `0 2 * * *` (daily at 2 AM)
- **Retention**: `7` days
- **Storage Type**: `s3`
- **Storage Location**: `observability-backups/<namespace>/<platform-name>`

### Alerting Defaults

When `alerting.alertmanager.enabled` is `true`:
- **Version**: `v0.26.0`
- **Replicas**: `1` (or `3` if HA is enabled)
- **Resources**:
  - Requests: `10m` CPU, `32Mi` memory
  - Limits: `100m` CPU, `128Mi` memory
- **Storage**:
  - Size: `1Gi`
- **Default Configuration**:
  ```yaml
  route:
    groupBy: ["alertname", "cluster", "service"]
    groupWait: "10s"
    groupInterval: "10s"
    repeatInterval: "12h"
    receiver: "default-receiver"
  receivers:
    - name: "default-receiver"
  ```

## Behavior

### Preservation of Existing Values

The mutating webhook **never** overrides values that are already set. It only adds missing values. For example:

- If you specify `prometheus.version: v2.45.0`, it won't be changed to the default `v2.48.0`
- If you set custom labels, they are preserved and default labels are added only if missing
- Resource specifications are only added for missing fields

### Resource Normalization

When partial resource specifications are provided, the webhook completes them:

```yaml
# Input
resources:
  requests:
    cpu: 500m
    # memory is missing

# After mutation
resources:
  requests:
    cpu: 500m      # preserved
    memory: 512Mi  # added default
  limits:          # added defaults
    cpu: "1"
    memory: 2Gi
```

### Password Generation

For Grafana, if no `adminPassword` is specified, a cryptographically secure 16-character password is generated containing:
- Lowercase letters
- Uppercase letters
- Numbers
- Special characters

The generated password is logged (at info level) for initial retrieval.

### Environment Inference

The webhook attempts to infer the environment from the namespace name:
- Namespaces named `production` or `prod` → environment label `production`
- Namespaces named `staging` or `stage` → environment label `staging`
- Namespaces named `development` or `dev` → environment label `development`
- All other namespaces → environment label `default`

## Examples

### Minimal Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
```

This minimal configuration will be expanded with all the defaults mentioned above.

### High Availability Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: ha-platform
  namespace: production
spec:
  components:
    prometheus:
      enabled: true
    grafana:
      enabled: true
  highAvailability:
    enabled: true
```

This will automatically set:
- Prometheus replicas to 3
- Grafana replicas to 2
- Environment label to "production"
- Anti-affinity rules

## Webhook Configuration

The mutating webhook is configured to:
- Intercept CREATE and UPDATE operations
- Apply to `observabilityplatforms` resources in the `observability.io` API group
- Fail closed (if the webhook is unavailable, the operation fails)
- Have no side effects

## Troubleshooting

### Webhook Not Applied

If defaults are not being applied:
1. Check that the webhook is registered: `kubectl get mutatingwebhookconfigurations`
2. Verify the webhook service is running: `kubectl get svc -n gunj-system gunj-operator-webhook-service`
3. Check operator logs for webhook errors: `kubectl logs -n gunj-system -l app=gunj-operator`

### Debugging Mutations

To see what mutations were applied:
1. Create a resource with minimal spec
2. Get the resource and observe the added fields: `kubectl get observabilityplatform <name> -o yaml`
3. Check operator logs for mutation details

### Disabling Mutations

If you need to disable the mutating webhook temporarily:
```bash
kubectl delete mutatingwebhookconfiguration gunj-operator-mutating-webhook-configuration
```

Note: This will prevent defaults from being applied to new resources.

## Best Practices

1. **Leverage Defaults**: Start with minimal configurations and let the webhook apply sensible defaults
2. **Override When Needed**: Only specify values that differ from defaults
3. **Review Generated Passwords**: Check operator logs for auto-generated Grafana passwords
4. **Use Namespace Conventions**: Name namespaces according to environment for automatic labeling
5. **Test in Non-Production**: Always test configurations in non-production environments first

## Security Considerations

1. **Password Storage**: Auto-generated passwords are only logged once. Store them securely.
2. **Webhook Certificates**: Managed by cert-manager for automatic rotation
3. **RBAC**: The webhook runs with the operator's service account permissions
4. **Fail Closed**: If the webhook fails, resource creation/updates are blocked for safety
