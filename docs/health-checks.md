# Health Check Configuration for Gunj Operator

This document describes how to configure health checks for the Gunj Operator.

## Overview

The Gunj Operator implements comprehensive health checking for both the operator itself and the observability platform components it manages.

### Operator Health Checks

The operator exposes the following health endpoints:

- `/healthz` - Liveness probe endpoint
- `/readyz` - Readiness probe endpoint
- `/metrics/health` - Detailed health metrics for all components

These endpoints are served on port 8081 by default.

### Component Health Checks

The operator monitors the health of all deployed components:

- **Prometheus**: Checks deployment status and `//-/healthy` endpoint
- **Grafana**: Checks deployment status and `/api/health` endpoint
- **Loki**: Checks statefulset status and `/ready` endpoint
- **Tempo**: Checks deployment status and `/ready` endpoint

## Deployment Configuration

### Operator Deployment with Health Probes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gunj-operator
  namespace: gunj-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gunj-operator
  template:
    metadata:
      labels:
        app: gunj-operator
    spec:
      containers:
      - name: operator
        image: gunj-operator:latest
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 8081
          name: health
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 10
          successThreshold: 1
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 3
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        env:
        - name: HEALTH_CHECK_INTERVAL
          value: "30s"
        - name: HEALTH_CHECK_TIMEOUT
          value: "10s"
```

## Health Status in CRD

The ObservabilityPlatform CRD includes detailed health information in its status:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
status:
  health:
    healthy: true
    lastChecked: "2025-06-13T10:30:00Z"
    message: "All components are healthy"
    status: "Healthy"
    healthyCount: 4
    totalCount: 4
    components:
      prometheus:
        healthy: true
        message: "Healthy"
        lastChecked: "2025-06-13T10:30:00Z"
        availableReplicas: 3
        desiredReplicas: 3
      grafana:
        healthy: true
        message: "Healthy"
        lastChecked: "2025-06-13T10:30:00Z"
        availableReplicas: 2
        desiredReplicas: 2
      loki:
        healthy: true
        message: "Healthy"
        lastChecked: "2025-06-13T10:30:00Z"
        availableReplicas: 1
        desiredReplicas: 1
      tempo:
        healthy: true
        message: "Healthy"
        lastChecked: "2025-06-13T10:30:00Z"
        availableReplicas: 1
        desiredReplicas: 1
```

## Health Metrics

The operator exposes the following Prometheus metrics:

```
# Component health status (1 = healthy, 0 = unhealthy)
gunj_operator_component_health{platform="production",namespace="monitoring",component="prometheus"} 1

# Component ready replicas
gunj_operator_component_ready_replicas{platform="production",namespace="monitoring",component="prometheus"} 3

# Component desired replicas
gunj_operator_component_desired_replicas{platform="production",namespace="monitoring",component="prometheus"} 3

# Health check duration
gunj_operator_health_check_duration_seconds{platform="production",namespace="monitoring"} 0.125

# Health check errors
gunj_operator_health_check_errors_total{platform="production",namespace="monitoring",component="prometheus",error_type="timeout"} 0

# Last health check timestamp
gunj_operator_last_health_check_timestamp{platform="production",namespace="monitoring",component="prometheus"} 1718275800
```

## Configuration Options

### Environment Variables

- `HEALTH_CHECK_INTERVAL` - How often to perform health checks (default: 30s)
- `HEALTH_CHECK_TIMEOUT` - Timeout for individual health checks (default: 10s)
- `HEALTH_SERVER_PORT` - Port for health endpoints (default: 8081)

### Command Line Flags

```bash
gunj-operator \
  --health-check-interval=30s \
  --health-check-timeout=10s \
  --health-server-port=8081
```

## Monitoring Health

### Using kubectl

```bash
# Check platform health status
kubectl get observabilityplatform production -n monitoring -o jsonpath='{.status.health}'

# Check operator health
kubectl exec -n gunj-system deployment/gunj-operator -- wget -qO- http://localhost:8081/healthz

# Get detailed health metrics
kubectl exec -n gunj-system deployment/gunj-operator -- wget -qO- http://localhost:8081/metrics/health
```

### Grafana Dashboard

Import the provided dashboard to visualize:
- Component health status over time
- Replica availability
- Health check duration trends
- Error rates

### Alerting Rules

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: gunj-operator-health
spec:
  groups:
  - name: operator_health
    interval: 30s
    rules:
    - alert: OperatorUnhealthy
      expr: up{job="gunj-operator"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Gunj Operator is down"
        
    - alert: ComponentUnhealthy
      expr: gunj_operator_component_health == 0
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Component {{ $labels.component }} is unhealthy"
        
    - alert: HealthCheckFailing
      expr: rate(gunj_operator_health_check_errors_total[5m]) > 0.1
      for: 15m
      labels:
        severity: warning
      annotations:
        summary: "Health checks are failing for {{ $labels.component }}"
```

## Troubleshooting

### Common Issues

1. **Health check timeouts**
   - Increase `HEALTH_CHECK_TIMEOUT`
   - Check network connectivity between operator and components
   - Verify component endpoints are accessible

2. **Components reported as unhealthy**
   - Check component logs
   - Verify resource availability
   - Check persistent volume status

3. **Operator not ready**
   - Check operator logs for startup errors
   - Verify RBAC permissions
   - Ensure CRDs are installed

### Debug Commands

```bash
# Enable debug logging for health checks
kubectl set env deployment/gunj-operator -n gunj-system LOG_LEVEL=debug

# Check health server logs
kubectl logs -n gunj-system deployment/gunj-operator | grep health

# Test component endpoints from operator pod
kubectl exec -n gunj-system deployment/gunj-operator -- \
  wget -qO- http://production-prometheus.monitoring.svc.cluster.local:9090/-/healthy
```
