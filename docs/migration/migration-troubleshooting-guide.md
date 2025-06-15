# Migration Troubleshooting Guide

## Table of Contents

- [Overview](#overview)
- [Common Issues Quick Reference](#common-issues-quick-reference)
- [Detailed Troubleshooting](#detailed-troubleshooting)
- [Diagnostic Tools](#diagnostic-tools)
- [Issue Categories](#issue-categories)
- [Resolution Procedures](#resolution-procedures)
- [Prevention Strategies](#prevention-strategies)
- [Getting Help](#getting-help)

## Overview

This guide helps troubleshoot common issues encountered during migration from v1alpha1 to v1beta1 of the Gunj Operator. Each issue includes symptoms, root causes, and step-by-step resolution procedures.

### How to Use This Guide

1. **Identify symptoms** in the Quick Reference
2. **Find your issue** in the detailed sections
3. **Follow resolution steps** in order
4. **Verify the fix** using provided commands
5. **Prevent recurrence** with best practices

## Common Issues Quick Reference

| Symptom | Likely Cause | Section |
|---------|--------------|---------|
| "field not supported" error | Deprecated field usage | [Field Migration Errors](#field-migration-errors) |
| Platform stuck in "Converting" | Webhook timeout | [Conversion Issues](#conversion-issues) |
| "connection refused" webhook error | Webhook not running | [Webhook Problems](#webhook-problems) |
| High memory usage after upgrade | HA enabled by default | [Resource Issues](#resource-issues) |
| Prometheus data missing | PVC migration needed | [Data Persistence](#data-persistence) |
| API version not found | CRD not updated | [API Version Issues](#api-version-issues) |
| Validation errors | Stricter v1beta1 rules | [Validation Failures](#validation-failures) |
| Performance degradation | Conversion overhead | [Performance Problems](#performance-problems) |

## Detailed Troubleshooting

### Field Migration Errors

#### Issue: "unknown field" Errors

**Symptoms:**
```
error validating data: ValidationError(ObservabilityPlatform.spec.components.prometheus): 
unknown field "customConfig" in io.observability.v1beta1.PrometheusSpec
```

**Root Cause:** Using deprecated v1alpha1 fields in v1beta1 API

**Resolution:**

1. **Identify deprecated fields:**
```bash
# Check for common deprecated fields
kubectl get observabilityplatform <name> -n <namespace> -o yaml | grep -E "customConfig|cpuRequest|memoryRequest|storageSize|retentionDays"
```

2. **Use migration tool:**
```bash
# Download and run migration tool
curl -LO https://github.com/gunjanjp/gunj-operator/releases/latest/download/gunj-migrate
chmod +x gunj-migrate

# Convert configuration
./gunj-migrate convert -f old-config.yaml -o new-config.yaml
```

3. **Manual field mapping:**
```yaml
# Before (v1alpha1)
spec:
  components:
    prometheus:
      customConfig: |
        global:
          scrape_interval: 15s
      cpuRequest: "1"
      memoryRequest: "4Gi"
      storageSize: 100Gi

# After (v1beta1)
spec:
  components:
    prometheus:
      global:
        scrapeInterval: 15s
      resources:
        requests:
          cpu: "1"
          memory: "4Gi"
      storage:
        size: 100Gi
```

4. **Apply fixed configuration:**
```bash
kubectl apply -f new-config.yaml
```

#### Issue: Type Mismatch Errors

**Symptoms:**
```
error validating data: ValidationError(ObservabilityPlatform.spec.components.prometheus.externalLabels): 
invalid type for io.observability.v1beta1.PrometheusSpec.externalLabels: 
got "string", expected "map"
```

**Resolution:**

1. **Convert string to map types:**
```bash
# For JSON string labels
echo '{"cluster": "prod", "region": "us-east-1"}' | jq -r 'to_entries | map("  \(.key): \(.value)") | .[]'

# Result:
#   cluster: prod
#   region: us-east-1
```

2. **Update configuration:**
```yaml
# Wrong (string)
externalLabels: '{"cluster": "prod", "region": "us-east-1"}'

# Correct (map)
externalLabels:
  cluster: prod
  region: us-east-1
```

### Conversion Issues

#### Issue: Platform Stuck in "Converting" State

**Symptoms:**
- Platform phase shows "Converting" for extended period
- No progress in conversion
- Operator logs show timeout errors

**Root Cause:** Conversion webhook timeout or failure

**Resolution:**

1. **Check webhook status:**
```bash
# Check webhook pods
kubectl get pods -n gunj-system -l app=gunj-operator-webhook

# Check webhook logs
kubectl logs -n gunj-system -l app=gunj-operator-webhook --tail=50
```

2. **Restart webhook if needed:**
```bash
# Delete webhook pods to force restart
kubectl delete pods -n gunj-system -l app=gunj-operator-webhook

# Wait for new pods
kubectl wait --for=condition=ready pod -l app=gunj-operator-webhook -n gunj-system --timeout=60s
```

3. **Force reconversion:**
```bash
# Add force-conversion annotation
kubectl annotate observabilityplatform <name> -n <namespace> \
  observability.io/force-conversion=true \
  observability.io/conversion-retry-count=0 \
  --overwrite

# Remove and re-add to trigger
kubectl patch observabilityplatform <name> -n <namespace> \
  --type='json' -p='[{"op": "remove", "path": "/metadata/annotations/observability.io~1convert-to-v1beta1"}]'

kubectl annotate observabilityplatform <name> -n <namespace> \
  observability.io/convert-to-v1beta1=true
```

4. **Check webhook certificate:**
```bash
# Verify webhook certificate is valid
kubectl get secret -n gunj-system gunj-operator-webhook-cert -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -text -noout | grep -A2 "Validity"
```

#### Issue: Partial Conversion

**Symptoms:**
- Some fields converted, others remain in old format
- Mixed v1alpha1 and v1beta1 syntax

**Resolution:**

1. **Export current state:**
```bash
kubectl get observabilityplatform <name> -n <namespace> -o yaml > current-state.yaml
```

2. **Clean and reconvert:**
```bash
# Remove status and metadata fields
cat current-state.yaml | \
  yq eval 'del(.status) | del(.metadata.resourceVersion) | del(.metadata.uid) | del(.metadata.generation)' \
  > clean-config.yaml

# Run through converter
./gunj-migrate convert -f clean-config.yaml -o converted-config.yaml

# Apply
kubectl apply -f converted-config.yaml
```

### Webhook Problems

#### Issue: Webhook Connection Refused

**Symptoms:**
```
Internal error occurred: failed calling webhook "vobservabilityplatform.kb.io": 
Post "https://gunj-operator-webhook-service.gunj-system.svc:443/convert": 
dial tcp 10.96.x.x:443: connect: connection refused
```

**Resolution:**

1. **Check webhook service:**
```bash
# Verify service exists
kubectl get svc -n gunj-system gunj-operator-webhook-service

# Check endpoints
kubectl get endpoints -n gunj-system gunj-operator-webhook-service
```

2. **Recreate webhook configuration:**
```bash
# Delete existing webhook configs
kubectl delete validatingwebhookconfiguration gunj-operator-validating-webhook-configuration
kubectl delete mutatingwebhookconfiguration gunj-operator-mutating-webhook-configuration

# Reapply webhook manifests
kubectl apply -f https://github.com/gunjanjp/gunj-operator/releases/download/v1.5.0/webhooks.yaml
```

3. **Check network policies:**
```bash
# List network policies
kubectl get networkpolicy -n gunj-system

# Temporarily disable if blocking
kubectl delete networkpolicy -n gunj-system --all
```

#### Issue: Webhook Certificate Invalid

**Symptoms:**
```
x509: certificate signed by unknown authority
```

**Resolution:**

1. **Regenerate certificates:**
```bash
# Delete old certificate
kubectl delete secret -n gunj-system gunj-operator-webhook-cert

# Trigger cert regeneration
kubectl rollout restart deployment -n gunj-system gunj-operator-webhook
```

2. **Update CA bundle:**
```bash
# Get CA certificate
CA_BUNDLE=$(kubectl get secret -n gunj-system gunj-operator-webhook-cert -o jsonpath='{.data.ca\.crt}')

# Update webhook configuration
kubectl patch validatingwebhookconfiguration gunj-operator-validating-webhook-configuration \
  --type='json' -p='[{"op": "replace", "path": "/webhooks/0/clientConfig/caBundle", "value":"'${CA_BUNDLE}'"}]'
```

### Resource Issues

#### Issue: Resource Quota Exceeded

**Symptoms:**
```
Error creating: pods "prometheus-1" is forbidden: 
exceeded quota: team-quota, requested: requests.memory=8Gi, 
used: requests.memory=32Gi, limited: requests.memory=40Gi
```

**Root Cause:** v1beta1 defaults to HA mode with higher resource requirements

**Resolution:**

1. **Check current usage:**
```bash
# View quota status
kubectl describe resourcequota -n <namespace>

# List resource requests
kubectl get pods -n <namespace> -o json | \
  jq -r '.items[] | "\(.metadata.name): CPU:\(.spec.containers[].resources.requests.cpu) Memory:\(.spec.containers[].resources.requests.memory)"'
```

2. **Temporarily disable HA:**
```yaml
# Reduce replicas and disable HA
spec:
  components:
    prometheus:
      replicas: 1  # Reduce from default 2
      highAvailability:
        enabled: false
```

3. **Optimize resource requests:**
```yaml
spec:
  components:
    prometheus:
      resources:
        requests:
          cpu: "500m"    # Reduced from 1
          memory: "2Gi"  # Reduced from 4Gi
        limits:
          cpu: "1"
          memory: "4Gi"
```

4. **Request quota increase:**
```bash
# For cluster admin
kubectl patch resourcequota team-quota -n <namespace> --type merge -p '
{
  "spec": {
    "hard": {
      "requests.memory": "80Gi",
      "requests.cpu": "40"
    }
  }
}'
```

#### Issue: Pod OOMKilled

**Symptoms:**
- Pods restarting with OOMKilled status
- Memory usage spikes during conversion

**Resolution:**

1. **Increase memory limits temporarily:**
```bash
# Patch deployment with higher memory
kubectl patch deployment <component> -n <namespace> --type merge -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "prometheus",
          "resources": {
            "limits": {
              "memory": "16Gi"
            }
          }
        }]
      }
    }
  }
}'
```

2. **Enable memory optimization:**
```yaml
spec:
  components:
    prometheus:
      # Add memory optimization settings
      extraArgs:
        - --storage.tsdb.wal-compression
        - --storage.tsdb.max-block-chunk-segment-size=32MB
        - --query.max-concurrency=10
```

### Data Persistence

#### Issue: Data Loss After Migration

**Symptoms:**
- Historical metrics missing
- Dashboards show "No Data"
- Queries return empty results

**Root Cause:** PVC not properly migrated or mounted

**Resolution:**

1. **Verify PVC exists:**
```bash
# List PVCs
kubectl get pvc -n <namespace> | grep prometheus

# Check PV binding
kubectl get pv | grep <namespace>
```

2. **Create migration job:**
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: prometheus-data-migration
  namespace: <namespace>
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: busybox:latest
        command:
        - sh
        - -c
        - |
          echo "Starting data migration..."
          # Check source data
          if [ -d /source/prometheus ]; then
            echo "Found source data, copying..."
            cp -Rpv /source/* /destination/
            echo "Setting permissions..."
            chown -R 65534:65534 /destination
            echo "Migration complete"
          else
            echo "No source data found"
            exit 1
          fi
        volumeMounts:
        - name: source-data
          mountPath: /source
        - name: dest-data
          mountPath: /destination
      volumes:
      - name: source-data
        persistentVolumeClaim:
          claimName: prometheus-data-v1alpha1
      - name: dest-data
        persistentVolumeClaim:
          claimName: prometheus-data-v1beta1
      restartPolicy: Never
```

3. **Verify data integrity:**
```bash
# Query historical data
kubectl exec -n <namespace> prometheus-0 -- \
  promtool query instant 'up[30d]' | jq '.data.result | length'
```

#### Issue: Wrong Storage Class

**Symptoms:**
```
Warning  ProvisioningFailed  persistentvolumeclaim/prometheus-data  
storageclass.storage.k8s.io "fast-ssd" not found
```

**Resolution:**

1. **List available storage classes:**
```bash
kubectl get storageclass
```

2. **Update platform configuration:**
```yaml
spec:
  components:
    prometheus:
      storage:
        storageClassName: standard  # Use available class
```

3. **Migrate existing data if needed:**
```bash
# Create new PVC with correct storage class
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: prometheus-data-new
  namespace: <namespace>
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
  storageClassName: standard
EOF

# Copy data using job (see above)
```

### API Version Issues

#### Issue: No matches for kind "ObservabilityPlatform"

**Symptoms:**
```
error: unable to recognize "platform.yaml": no matches for kind "ObservabilityPlatform" 
in version "observability.io/v1beta1"
```

**Resolution:**

1. **Check installed CRDs:**
```bash
# List CRD versions
kubectl get crd observabilityplatforms.observability.io -o jsonpath='{.spec.versions[*].name}'

# Verify v1beta1 is served
kubectl get crd observabilityplatforms.observability.io -o jsonpath='{.spec.versions[?(@.name=="v1beta1")].served}'
```

2. **Update CRDs if needed:**
```bash
# Apply latest CRDs
kubectl apply -f https://github.com/gunjanjp/gunj-operator/releases/download/v1.5.0/crds.yaml

# Wait for establishment
kubectl wait --for condition=established crd/observabilityplatforms.observability.io --timeout=60s
```

3. **Verify API availability:**
```bash
# Check API versions
kubectl api-versions | grep observability

# Test API access
kubectl explain observabilityplatform.spec --api-version=observability.io/v1beta1
```

### Validation Failures

#### Issue: Strict Validation Rejecting Valid Config

**Symptoms:**
```
error validating data: ValidationError(ObservabilityPlatform.spec.components.prometheus.version): 
invalid value: "2.48.0", must match pattern "^v?\d+\.\d+\.\d+$"
```

**Resolution:**

1. **Fix version format:**
```yaml
# Add 'v' prefix to versions
spec:
  components:
    prometheus:
      version: "v2.48.0"  # Added 'v' prefix
```

2. **Fix resource quantities:**
```yaml
# Use Kubernetes quantity format
resources:
  requests:
    cpu: "1000m"    # or "1"
    memory: "4Gi"   # not "4GB"
```

3. **Fix duration formats:**
```yaml
# Use duration strings
retention: "30d"          # not retentionDays: 30
scrapeInterval: "30s"     # not scrapeInterval: 30
```

#### Issue: Label Validation Errors

**Symptoms:**
```
invalid label key: "app.kubernetes.io/very-long-label-name-that-exceeds-the-limit": 
must be no more than 63 characters
```

**Resolution:**

1. **Shorten label keys and values:**
```yaml
# Before
labels:
  app.kubernetes.io/very-long-label-name-that-exceeds-the-limit: value

# After
labels:
  app.kubernetes.io/component: value
```

2. **Remove invalid characters:**
```bash
# Valid label format: [a-z0-9]([-a-z0-9]*[a-z0-9])?
# Fix using sed
echo "my_invalid-LABEL" | sed 's/_/-/g' | tr '[:upper:]' '[:lower:]'
```

### Performance Problems

#### Issue: Slow Reconciliation After Upgrade

**Symptoms:**
- Reconciliation taking minutes instead of seconds
- High operator CPU usage
- Timeouts in operator logs

**Resolution:**

1. **Check reconciliation metrics:**
```bash
# Query reconciliation duration
kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant 'histogram_quantile(0.99, gunj_operator_reconcile_duration_seconds_bucket)'
```

2. **Increase operator resources:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gunj-operator
  namespace: gunj-system
spec:
  template:
    spec:
      containers:
      - name: operator
        resources:
          requests:
            cpu: "500m"     # Increased
            memory: "512Mi" # Increased
          limits:
            cpu: "2"
            memory: "2Gi"
```

3. **Optimize reconciliation frequency:**
```bash
# Add reconciliation interval annotation
kubectl annotate observabilityplatform <name> -n <namespace> \
  observability.io/reconcile-frequency=5m
```

4. **Enable caching:**
```yaml
# In operator deployment
env:
- name: ENABLE_CACHE
  value: "true"
- name: CACHE_TTL
  value: "300"
```

## Diagnostic Tools

### Comprehensive Diagnostics Script

```bash
#!/bin/bash
# diagnose-migration-issues.sh

echo "=== Gunj Operator Migration Diagnostics ==="
echo "Timestamp: $(date)"
echo ""

# Function to check component
check_component() {
    local component=$1
    local namespace=$2
    
    echo "Checking $component in $namespace..."
    
    # Pod status
    kubectl get pods -n "$namespace" -l "app=$component" -o wide
    
    # Recent events
    kubectl get events -n "$namespace" --field-selector "involvedObject.name=$component" \
      --sort-by='.lastTimestamp' | tail -5
    
    # Logs (last 20 lines)
    kubectl logs -n "$namespace" -l "app=$component" --tail=20 --all-containers
    
    echo ""
}

# Check operator
check_component "gunj-operator" "gunj-system"

# Check webhook
check_component "gunj-operator-webhook" "gunj-system"

# Check CRDs
echo "CRD Status:"
kubectl get crd observabilityplatforms.observability.io -o json | \
  jq -r '.spec.versions[] | "\(.name): served=\(.served), storage=\(.storage)"'

# Check platforms
echo ""
echo "Platform Status:"
kubectl get observabilityplatforms.observability.io --all-namespaces -o custom-columns=\
NAMESPACE:.metadata.namespace,\
NAME:.metadata.name,\
VERSION:.apiVersion,\
PHASE:.status.phase,\
MESSAGE:.status.message

# Check for deprecated fields
echo ""
echo "Checking for deprecated field usage..."
for platform in $(kubectl get observabilityplatforms.observability.io --all-namespaces -o json | \
  jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"'); do
    
    namespace=$(echo "$platform" | cut -d/ -f1)
    name=$(echo "$platform" | cut -d/ -f2)
    
    deprecated=$(kubectl get observabilityplatform "$name" -n "$namespace" -o json | \
      jq -r '[.. | objects | keys[]] | map(select(. == "customConfig" or . == "cpuRequest" or . == "memoryRequest")) | unique | join(", ")')
    
    if [ -n "$deprecated" ]; then
        echo "  $platform uses deprecated fields: $deprecated"
    fi
done

# Resource usage
echo ""
echo "Resource Usage:"
kubectl top nodes
kubectl top pods -n gunj-system

# Recent errors from operator logs
echo ""
echo "Recent Operator Errors:"
kubectl logs -n gunj-system -l app=gunj-operator --since=1h | grep -E "ERROR|WARN" | tail -20

echo ""
echo "=== Diagnostics Complete ==="
```

### Migration Health Check

```bash
#!/bin/bash
# migration-health-check.sh

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "Running migration health check..."

# Check 1: Operator version
OPERATOR_VERSION=$(kubectl get deployment -n gunj-system gunj-operator -o jsonpath='{.spec.template.spec.containers[0].image}' | cut -d: -f2)
if [[ "$OPERATOR_VERSION" == "v1.5.0" ]] || [[ "$OPERATOR_VERSION" > "v1.5.0" ]]; then
    echo -e "${GREEN}✓${NC} Operator version: $OPERATOR_VERSION"
else
    echo -e "${RED}✗${NC} Operator version: $OPERATOR_VERSION (upgrade required)"
fi

# Check 2: CRD versions
if kubectl get crd observabilityplatforms.observability.io -o jsonpath='{.spec.versions[?(@.name=="v1beta1")].served}' | grep -q true; then
    echo -e "${GREEN}✓${NC} v1beta1 API available"
else
    echo -e "${RED}✗${NC} v1beta1 API not available"
fi

# Check 3: Webhook status
WEBHOOK_READY=$(kubectl get pods -n gunj-system -l app=gunj-operator-webhook -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}')
if [[ "$WEBHOOK_READY" == "True" ]]; then
    echo -e "${GREEN}✓${NC} Conversion webhook ready"
else
    echo -e "${YELLOW}⚠${NC}  Conversion webhook not ready"
fi

# Check 4: Platform migration status
TOTAL_PLATFORMS=$(kubectl get observabilityplatforms.observability.io --all-namespaces --no-headers | wc -l)
V1BETA1_PLATFORMS=$(kubectl get observabilityplatforms.observability.io --all-namespaces -o json | jq -r '.items[] | select(.apiVersion | contains("v1beta1"))' | jq -s 'length')

echo -e "Platform migration: $V1BETA1_PLATFORMS/$TOTAL_PLATFORMS completed"

# Check 5: Component health
UNHEALTHY_COMPONENTS=$(kubectl get pods --all-namespaces -l "observability.io/managed=true" -o json | \
  jq -r '.items[] | select(.status.phase != "Running") | "\(.metadata.namespace)/\(.metadata.name)"')

if [ -z "$UNHEALTHY_COMPONENTS" ]; then
    echo -e "${GREEN}✓${NC} All components healthy"
else
    echo -e "${RED}✗${NC} Unhealthy components:"
    echo "$UNHEALTHY_COMPONENTS"
fi

echo ""
echo "Health check complete!"
```

## Issue Categories

### Critical Issues (Immediate Action)

1. **Data Loss Risk**
   - Missing PVCs
   - Incorrect volume mounts
   - Failed data migration

2. **Service Disruption**
   - Webhook failures
   - API unavailability
   - Component crashes

3. **Security Vulnerabilities**
   - Exposed secrets
   - Incorrect RBAC
   - Missing network policies

### Major Issues (Plan Resolution)

1. **Performance Degradation**
   - Slow reconciliation
   - High resource usage
   - API latency

2. **Configuration Problems**
   - Validation failures
   - Type mismatches
   - Missing required fields

3. **Integration Failures**
   - Prometheus scraping issues
   - Grafana datasource problems
   - Alert routing failures

### Minor Issues (Monitor)

1. **Cosmetic Problems**
   - Warning messages
   - Deprecated field usage
   - Non-critical validation

2. **Optimization Opportunities**
   - Resource rightsizing
   - Cache tuning
   - Batch processing

## Resolution Procedures

### Standard Resolution Flow

1. **Identify** the issue category
2. **Gather** diagnostic information
3. **Isolate** the root cause
4. **Apply** the fix
5. **Verify** resolution
6. **Document** for future reference

### Emergency Response

For critical production issues:

1. **Stabilize** the system
   - Scale down problematic components
   - Increase resources if needed
   - Disable non-critical features

2. **Diagnose** quickly
   - Run diagnostic scripts
   - Check recent changes
   - Review error logs

3. **Fix or Rollback**
   - Apply hotfix if available
   - Rollback if fix will take time
   - Communicate status

4. **Post-Incident**
   - Root cause analysis
   - Update runbooks
   - Implement prevention

## Prevention Strategies

### Pre-Migration Prevention

1. **Thorough Testing**
   - Test in dev environment
   - Validate configurations
   - Check resource availability

2. **Proper Planning**
   - Review breaking changes
   - Plan maintenance windows
   - Prepare rollback procedures

3. **Tool Usage**
   - Use migration tool
   - Validate with dry-run
   - Check deprecation warnings

### Post-Migration Prevention

1. **Monitoring**
   - Set up alerts
   - Track key metrics
   - Regular health checks

2. **Documentation**
   - Update runbooks
   - Document customizations
   - Maintain change log

3. **Training**
   - Team knowledge sharing
   - Regular drills
   - Stay updated on changes

## Getting Help

### Self-Service Resources

1. **Documentation**
   - [Breaking Changes Guide](breaking-changes-v2.md)
   - [Migration Guide](user-friendly-migration-guide.md)
   - [API Reference](https://docs.gunj.io/api/v1beta1)

2. **Tools**
   - Migration tool: `gunj-migrate`
   - Diagnostic script: `diagnose-migration-issues.sh`
   - Health check: `migration-health-check.sh`

3. **Examples**
   - [Migration examples](migration-examples.md)
   - [Configuration samples](https://github.com/gunjanjp/gunj-operator/tree/main/examples)

### Community Support

- **Slack**: #gunj-operator-help
- **GitHub Issues**: [Report a problem](https://github.com/gunjanjp/gunj-operator/issues/new)
- **Stack Overflow**: Tag `gunj-operator`

### Commercial Support

For production issues:
- **Email**: gunjanjp@gmail.com
- **Support Portal**: https://support.gunj.io
- **Phone**: +1-555-GUNJ-OPS (business hours)

### Emergency Contacts

For critical production issues:
- **Pager**: on-call@gunj.io
- **Emergency Line**: +1-555-GUNJ-911
- **Response Time**: 15 minutes for P1 issues

---

**Document Version**: 1.0  
**Last Updated**: June 15, 2025  
**Next Review**: Monthly

**Remember**: Most migration issues are known and documented. Check this guide first before escalating!
