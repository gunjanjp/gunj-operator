# ObservabilityPlatform API

This directory contains the API definitions for the Gunj Operator's Custom Resource Definitions (CRDs).

## API Version: v1beta1

The `v1beta1` API version includes the following types:

### ObservabilityPlatform

The main CRD that defines an observability platform deployment with the following components:

- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Loki**: Log aggregation
- **Tempo**: Distributed tracing
- **OpenTelemetry Collector**: Unified telemetry collection

### Key Features

1. **Component Management**
   - Individual enable/disable for each component
   - Version management with validation
   - Resource allocation and limits
   - Persistent storage configuration
   - High availability support

2. **Global Configuration**
   - External labels for all telemetry data
   - Unified log levels
   - Node selection and tolerations
   - Pod affinity rules
   - Image pull secrets

3. **Security**
   - TLS/SSL support with auto-generation
   - Multiple authentication methods (OIDC, LDAP, Basic)
   - Pod Security Policies
   - Network Policies

4. **Operations**
   - Automated backups with configurable retention
   - Alerting with Alertmanager integration
   - Custom alerting rules
   - Multi-cluster support ready

5. **Integration**
   - S3/GCS/Azure storage backends
   - Remote write for Prometheus
   - External data sources for Grafana
   - SMTP configuration for notifications

### Usage Example

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
  namespace: observability
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
    grafana:
      enabled: true
      version: "10.2.0"
```

### Development

To regenerate the CRD manifests after modifying the Go types:

```bash
make generate
make manifests
```

### Validation

The CRD includes comprehensive validation:
- Version format validation (semantic versioning)
- Resource limit validation
- Enum validation for phases and log levels
- Required field enforcement
- Pattern matching for specific fields

### Status Management

The operator maintains detailed status information:
- Overall platform phase
- Per-component status
- Health indicators
- Endpoint discovery
- Condition tracking
- Last reconciliation time

For more examples, see the [examples directory](../../examples/).
