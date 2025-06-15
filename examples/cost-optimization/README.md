# Cost Optimization Examples

This directory contains examples of how to configure cost optimization features in the Gunj Operator.

## Overview

The Gunj Operator provides comprehensive cost optimization capabilities:
- **Resource Right-sizing**: Automatically adjusts resource allocations based on actual usage
- **Spot Instance Support**: Uses spot instances for cost savings on eligible components
- **Auto-scaling**: Scales components based on load to optimize costs
- **Budget Management**: Sets budgets and alerts when thresholds are exceeded
- **Cost Allocation**: Tags resources for chargeback and showback

## Examples

### 1. Full Cost Optimization (`platform-with-cost-optimization.yaml`)

This example shows a production platform with all cost optimization features enabled:
- Resource optimization based on usage patterns
- Spot instances for non-critical components (Grafana, Loki queriers)
- Auto-scaling with sensible limits
- Budget alerts at multiple thresholds
- Automated actions when budget thresholds are reached
- Cost allocation tags for chargeback

**Key Features:**
- Monthly budget: $1000
- Alerts at 80%, 90%, and 100% of budget
- Automatic scaling down at 90% budget usage
- Spot instances for Grafana with proper tolerations
- S3 backend for Loki to reduce storage costs

### 2. Environment-Specific Optimization (`environment-specific-cost-optimization.yaml`)

Shows different optimization strategies for different environments:

**Development Environment:**
- Minimal resources (single replicas)
- Short retention periods (7 days)
- No auto-scaling
- Budget: $200/month
- Some components disabled (Tempo)

**Staging Environment:**
- 100% spot instances for maximum savings
- Moderate resources
- Auto-scaling enabled but with lower limits
- Budget: $500/month
- All components use spot instances

### 3. Budget Monitoring Only (`budget-monitoring-only.yaml`)

For organizations that want cost visibility without automatic optimization:
- No automatic resource changes
- Comprehensive budget alerting
- Multiple notification channels
- Detailed cost allocation tags
- Custom Prometheus rules for cost trend analysis

**Includes:**
- Cost trending alerts
- Cost anomaly detection
- Storage growth monitoring

## Cost Optimization Configuration

### Basic Structure

```yaml
spec:
  costOptimization:
    enabled: true
    resourceOptimization: true    # Enable resource right-sizing
    autoScaling: true            # Enable auto-scaling
    spotInstances:
      enabled: true
      allowedComponents:         # Components that can use spot
        - grafana
        - loki-querier
      maxSpotPercentage: 70     # Max % of instances as spot
    budget:
      monthlyLimit: 1000.0      # Monthly budget in USD
      alertThresholds: [80, 90] # Alert at these percentages
    costAllocation:
      enabled: true
      department: Engineering
      team: Platform Team
```

### Budget Actions

Available actions when budget thresholds are reached:
- `alert`: Send notifications only
- `scaleDown`: Reduce replicas to minimum
- `disableNonEssential`: Disable non-critical features
- `preventNewDeployments`: Block new platform deployments

### Alert Channels

Supported notification channels:
- `email`: Send email alerts
- `slack`: Send Slack notifications
- `webhook`: Call custom webhook endpoints

## Best Practices

### 1. Start with Monitoring
Begin with budget monitoring only to understand your costs before enabling automatic optimization.

### 2. Test in Non-Production
Test optimization settings in development/staging before applying to production.

### 3. Gradual Optimization
Enable optimizations gradually:
1. Start with resource right-sizing
2. Add auto-scaling
3. Finally, enable spot instances

### 4. Set Realistic Budgets
Set budgets slightly above your expected costs to avoid frequent alerts.

### 5. Use Cost Allocation
Always configure cost allocation tags for proper chargeback/showback.

## Cost Metrics

The operator exposes these Prometheus metrics for cost monitoring:

```prometheus
# Current cost estimate by platform and component
gunj_operator_platform_cost_estimate_dollars{platform="...", namespace="...", component="..."}

# Resource utilization percentage
gunj_operator_resource_utilization_percent{platform="...", resource_type="cpu|memory"}

# Potential savings from optimizations
gunj_operator_potential_savings_dollars{platform="...", optimization_type="..."}
```

## Grafana Dashboard

Import the cost optimization dashboard to visualize:
- Current costs by component
- Cost trends over time
- Budget usage and projections
- Potential savings
- Resource utilization

## Troubleshooting

### Costs Higher Than Expected
1. Check resource utilization metrics
2. Review retention policies
3. Verify storage class usage
4. Check for unused resources

### Spot Instance Issues
1. Ensure nodes have proper labels
2. Verify spot instance tolerations
3. Check node availability
4. Review eviction events

### Budget Alerts Not Working
1. Verify PrometheusRule creation
2. Check Alertmanager configuration
3. Review webhook endpoints
4. Test notification channels

## Additional Resources

- [Cost Optimization Guide](../../docs/user-guide/cost-optimization.md)
- [Budget Management](../../docs/user-guide/budget-management.md)
- [Spot Instance Configuration](../../docs/user-guide/spot-instances.md)
- [Grafana Cost Dashboard](../dashboards/cost-optimization-dashboard.json)
