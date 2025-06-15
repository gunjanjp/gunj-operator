# Cost Optimization Guide

## Overview

The Gunj Operator includes comprehensive cost optimization features that help you reduce the operational costs of your observability platform while maintaining performance and reliability.

## Features

### 1. Cost Analysis

The operator continuously analyzes the costs of running your observability platform:

- **Resource Costs**: CPU, memory usage costs based on cloud provider pricing
- **Storage Costs**: Persistent volume costs with tiered pricing support
- **Network Costs**: Egress costs for remote write and data transfer
- **Component Breakdown**: Detailed cost analysis per component

### 2. Resource Right-Sizing

Automatically adjusts resource allocations based on actual usage patterns:

- Analyzes historical usage data (P50, P90, P95, P99 percentiles)
- Recommends optimal resource allocations
- Applies changes automatically when enabled
- Maintains safety margins to prevent under-provisioning

### 3. Spot Instance Support

Reduces costs by using spot instances for eligible components:

- Automatically identifies spot-eligible components
- Configures proper node selectors and tolerations
- Handles spot instance interruptions gracefully
- Supports mixed instance deployments

### 4. Auto-Scaling

Implements intelligent scaling policies:

- CPU and memory-based scaling
- Custom metrics support
- Prevents flapping with stabilization windows
- Cost-aware scaling decisions

### 5. Budget Management

Set and monitor budgets with automated actions:

- Monthly budget limits
- Multi-threshold alerts (50%, 80%, 90%, 100%)
- Multiple notification channels (email, Slack, webhooks)
- Automated actions (scale down, disable features)

### 6. Cost Allocation

Track costs by team, department, or project:

- Tag-based cost allocation
- Chargeback/showback support
- Detailed cost reports
- Integration with cloud billing

## Configuration

### Basic Configuration

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
spec:
  costOptimization:
    enabled: true
    resourceOptimization: true
    autoScaling: true
```

### Advanced Configuration

```yaml
spec:
  costOptimization:
    enabled: true
    resourceOptimization: true
    autoScaling: true
    
    spotInstances:
      enabled: true
      allowedComponents:
        - grafana
        - loki-querier
        - tempo-querier
      maxSpotPercentage: 70
    
    budget:
      monthlyLimit: 1000.0
      alertThresholds: [80, 90, 100]
      alertChannels:
        - type: email
          target: team@example.com
        - type: slack
          target: https://hooks.slack.com/services/xxx
      actions:
        - threshold: 90
          action: scaleDown
        - threshold: 100
          action: disableNonEssential
    
    costAllocation:
      enabled: true
      department: Engineering
      costCenter: CC-1234
      team: Platform Team
      tags:
        project: observability
        environment: production
```

## Cost Optimization Strategies

### 1. Storage Optimization

- Use object storage (S3) for long-term retention
- Implement data lifecycle policies
- Compress data where possible
- Use appropriate storage classes

### 2. Resource Optimization

- Start with baseline resources and scale up
- Use P95 utilization for sizing
- Implement request/limit ratios properly
- Regular review and adjustment

### 3. Architecture Optimization

- Use remote write for federation
- Implement proper data retention
- Optimize cardinality
- Use recording rules

### 4. Spot Instance Best Practices

- Use for stateless components only
- Implement proper disruption handling
- Mix spot and on-demand instances
- Monitor spot availability

## Monitoring Cost Optimization

### Prometheus Metrics

```prometheus
# Current platform cost
gunj_operator_platform_cost_estimate_dollars{platform="my-platform"}

# Resource utilization
gunj_operator_resource_utilization_percent{resource_type="cpu"}

# Potential savings
gunj_operator_potential_savings_dollars{optimization_type="resource"}

# Budget usage
gunj_operator_budget_usage_percent{platform="my-platform"}
```

### Grafana Dashboard

The cost optimization dashboard provides:
- Real-time cost tracking
- Budget usage visualization
- Savings opportunities
- Resource utilization trends
- Component cost breakdown

### Alerts

Example cost-related alerts:

```yaml
- alert: BudgetThresholdExceeded
  expr: gunj_operator_budget_usage_percent > 80
  annotations:
    summary: "Platform budget usage above 80%"

- alert: HighResourceWaste
  expr: |
    (1 - gunj_operator_resource_utilization_percent) > 0.5
  annotations:
    summary: "Resource utilization below 50%"

- alert: SpotInstanceUnavailable
  expr: gunj_operator_spot_instance_available == 0
  annotations:
    summary: "Spot instances unavailable"
```

## Cost Reports

### Monthly Cost Report

Generate monthly cost reports:

```bash
kubectl exec -n gunj-system deployment/gunj-operator -- \
  gunj-cli cost report --platform my-platform --month 2025-06
```

### Cost Allocation Report

Generate chargeback reports:

```bash
kubectl exec -n gunj-system deployment/gunj-operator -- \
  gunj-cli cost allocation --platform my-platform --by team
```

## Troubleshooting

### High Costs

1. Check resource utilization metrics
2. Review component configurations
3. Analyze storage usage
4. Check for cardinality issues

### Optimization Not Applied

1. Verify cost optimization is enabled
2. Check operator logs for errors
3. Review recommendations
4. Validate permissions

### Budget Alerts Not Working

1. Check PrometheusRule creation
2. Verify Alertmanager configuration
3. Test notification channels
4. Review alert thresholds

## Best Practices

1. **Start Conservative**: Begin with monitoring only before enabling auto-optimization
2. **Test in Staging**: Validate optimization settings in non-production first
3. **Regular Reviews**: Schedule monthly cost reviews
4. **Document Changes**: Track all optimization decisions
5. **Monitor Impact**: Watch for performance impacts after optimization

## API Reference

### Get Cost Analysis

```bash
curl -X GET https://api.gunj-operator.example.com/api/v1/platforms/my-platform/cost-analysis \
  -H "Authorization: Bearer $TOKEN"
```

### Get Recommendations

```bash
curl -X GET https://api.gunj-operator.example.com/api/v1/platforms/my-platform/cost-recommendations \
  -H "Authorization: Bearer $TOKEN"
```

### Apply Optimizations

```bash
curl -X POST https://api.gunj-operator.example.com/api/v1/platforms/my-platform/cost-optimizations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "applyResourceOptimization": true,
    "enableSpotInstances": true,
    "applyScalingPolicies": true
  }'
```

### Set Budget

```bash
curl -X PUT https://api.gunj-operator.example.com/api/v1/platforms/my-platform/budget \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "monthlyLimit": 1000.0,
    "alertThresholds": [80, 90, 100]
  }'
```
