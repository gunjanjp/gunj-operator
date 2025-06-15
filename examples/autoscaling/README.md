# Autoscaling Examples

This directory contains examples of how to configure and use autoscaling features in the Gunj Operator.

## Examples Included

1. **basic-hpa.yaml** - Basic horizontal pod autoscaling configuration
2. **advanced-hpa.yaml** - Advanced HPA with custom metrics
3. **vpa-config.yaml** - Vertical pod autoscaling configuration
4. **predictive-scaling.yaml** - Predictive scaling based on historical data
5. **cost-aware-scaling.yaml** - Cost-optimized autoscaling
6. **multi-type-scaling.yaml** - Combined HPA, VPA, and predictive scaling
7. **custom-metrics.yaml** - Using custom metrics for scaling decisions
8. **scaling-policies.yaml** - Different scaling policies for different components

## Quick Start

To enable autoscaling for your ObservabilityPlatform:

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
spec:
  autoscaling:
    enabled: true
    hpa:
      enabled: true
    vpa:
      enabled: true
  components:
    prometheus:
      enabled: true
      autoscaling:
        minReplicas: 2
        maxReplicas: 10
        targetCPUUtilizationPercentage: 70
```

## Monitoring Autoscaling

Check autoscaling status:

```bash
# Get HPA status
kubectl get hpa -n <namespace>

# Get VPA status
kubectl get vpa -n <namespace>

# View scaling events
kubectl get events -n <namespace> --field-selector reason=Scaling

# Check metrics
kubectl exec -n gunj-system deployment/gunj-operator -- curl -s http://localhost:8080/metrics | grep gunj_operator_scaling
```

## Troubleshooting

If autoscaling is not working as expected:

1. Check that metrics-server is installed and running
2. Verify that Prometheus is scraping component metrics
3. Check operator logs for scaling decisions
4. Ensure resource requests are set on containers
5. Verify RBAC permissions for autoscaling resources
