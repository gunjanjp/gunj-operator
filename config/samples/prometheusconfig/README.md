# PrometheusConfig Examples

This directory contains example PrometheusConfig custom resources demonstrating various configuration scenarios for advanced Prometheus setups.

## Examples Overview

### 1. Basic Configuration (`prometheusconfig_v1beta1_basic.yaml`)
A minimal PrometheusConfig showing:
- Basic global configuration
- Simple Kubernetes service discovery for pods
- External labels configuration

**Use Case**: Getting started with custom Prometheus configuration

### 2. Remote Storage (`prometheusconfig_v1beta1_remote_storage.yaml`)
Demonstrates remote write and read configurations:
- Multiple remote write endpoints (Thanos, Grafana Cloud)
- Remote read configuration for federated queries
- Queue tuning for optimal performance
- TLS and authentication configuration
- Write relabeling to control data flow

**Use Case**: Setting up long-term storage and federation

### 3. Recording and Alerting Rules (`prometheusconfig_v1beta1_rules.yaml`)
Shows how to configure:
- Recording rules for pre-aggregated metrics
- Multi-group alerting rules
- Infrastructure and application alerts
- Proper use of labels and annotations
- Keep firing duration for alerts

**Use Case**: Implementing monitoring rules and alerts

### 4. Service Discovery (`prometheusconfig_v1beta1_service_discovery.yaml`)
Comprehensive service discovery examples:
- Kubernetes SD (pods, services, endpoints)
- Consul service discovery
- DNS-based discovery
- File-based discovery
- Cloud provider discovery (AWS EC2, Azure VMs, GCE)
- Static configurations
- Advanced relabeling configurations

**Use Case**: Discovering and monitoring targets across different platforms

### 5. Advanced Configuration (`prometheusconfig_v1beta1_advanced.yaml`)
Advanced Prometheus features:
- TSDB configuration with retention and compression
- Native histogram support
- Out-of-order sample handling
- Distributed tracing integration
- Exemplar storage configuration
- Query optimization settings
- WAL tuning
- Multiple remote storage backends

**Use Case**: Production-grade Prometheus with advanced features

## How to Use

1. **Choose an example** that matches your use case
2. **Copy the example** to your namespace:
   ```bash
   kubectl apply -f prometheusconfig_v1beta1_basic.yaml -n your-namespace
   ```
3. **Modify the targetPlatform** reference to match your ObservabilityPlatform:
   ```yaml
   spec:
     targetPlatform:
       name: your-platform-name
   ```
4. **Customize** the configuration according to your needs

## Configuration Hierarchy

PrometheusConfig settings override the default Prometheus configuration from the ObservabilityPlatform. The merge strategy is:
- Global config: Merged with precedence to PrometheusConfig
- Service discovery: Appended to existing configurations
- Remote write/read: Appended to existing endpoints
- Rules: Merged by group name
- Storage settings: Completely overridden

## Best Practices

1. **Start Simple**: Begin with basic configuration and add complexity as needed
2. **Test Relabeling**: Use Prometheus UI to test relabeling rules before applying
3. **Monitor Performance**: Watch queue metrics when using remote write
4. **Secure Credentials**: Always use Kubernetes secrets for sensitive data
5. **Version Control**: Keep your PrometheusConfig in git for change tracking

## Validation

The operator validates PrometheusConfig before applying:
- Syntax validation for PromQL expressions
- TLS certificate validation
- Remote endpoint connectivity checks
- Rule validation for recording and alerting rules

Check the PrometheusConfig status for validation results:
```bash
kubectl get prometheusconfig -n your-namespace
kubectl describe prometheusconfig config-name -n your-namespace
```

## Troubleshooting

Common issues and solutions:

1. **Config Not Applied**: Check if the target platform exists and is ready
2. **Invalid Rules**: Validate PromQL syntax using `promtool`
3. **Remote Write Failing**: Check network policies and credentials
4. **High Memory Usage**: Tune queue configurations and sample limits
5. **Service Discovery Not Working**: Verify RBAC permissions for Prometheus

For more information, see the [Gunj Operator documentation](https://gunjanjp.github.io/gunj-operator/docs/).
