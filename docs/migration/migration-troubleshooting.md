# Migration Troubleshooting Guide

This comprehensive guide helps diagnose and resolve issues that may arise during ObservabilityPlatform API version migrations.

## Table of Contents

- [Common Issues](#common-issues)
- [Diagnostic Tools](#diagnostic-tools)
- [Error Messages](#error-messages)
- [Performance Issues](#performance-issues)
- [Data Integrity](#data-integrity)
- [Recovery Procedures](#recovery-procedures)
- [Advanced Debugging](#advanced-debugging)
- [FAQ](#faq)

## Common Issues

### 1. Conversion Webhook Failures

#### Symptoms
- Error: `conversion webhook for observability.io/v1beta1 failed`
- Resources stuck in old API version
- kubectl commands timing out

#### Diagnosis
```bash
# Check webhook status
kubectl get validatingwebhookconfigurations gunj-operator-webhook -o yaml
kubectl get mutatingwebhookconfigurations gunj-operator-webhook -o yaml

# Check webhook pods
kubectl get pods -n gunj-system -l app=gunj-operator-webhook
kubectl logs -n gunj-system -l app=gunj-operator-webhook --tail=100

# Test webhook connectivity
kubectl run webhook-test --rm -it --image=curlimages/curl -- \
  curl -k https://gunj-operator-webhook.gunj-system.svc:443/healthz
```

#### Solutions

**Solution 1: Restart webhook pods**
```bash
kubectl rollout restart deployment/gunj-operator-webhook -n gunj-system
kubectl rollout status deployment/gunj-operator-webhook -n gunj-system
```

**Solution 2: Regenerate webhook certificates**
```bash
# Delete old certificates
kubectl delete secret gunj-operator-webhook-cert -n gunj-system

# Trigger certificate regeneration
kubectl delete pod -n gunj-system -l app=gunj-operator-webhook

# Wait for new certificates
kubectl wait --for=condition=ready pod -n gunj-system -l app=gunj-operator-webhook --timeout=60s
```

**Solution 3: Bypass webhook temporarily (emergency only)**
```bash
# Add label to skip conversion
kubectl label namespace <namespace> conversion.observability.io/skip=true

# Remove label after fixing issue
kubectl label namespace <namespace> conversion.observability.io/skip-
```

### 2. Validation Errors

#### Symptoms
- Error: `error validating data: ValidationError`
- Fields rejected during migration
- Resources fail to apply

#### Common Validation Errors

**Resource Quantities**
```yaml
# ERROR: Invalid quantity format
resources:
  requests:
    memory: 4GB  # Wrong: Should be 4Gi
    cpu: 1 core  # Wrong: Should be 1000m or 1

# CORRECT
resources:
  requests:
    memory: 4Gi
    cpu: 1000m
```

**Version Format**
```yaml
# ERROR: Invalid version format
version: 2.48.0  # Wrong: Missing 'v' prefix

# CORRECT
version: v2.48.0
```

**Label Format**
```yaml
# ERROR: Invalid label format
labels:
  my-label: value with spaces  # Wrong: Spaces not allowed
  _private: value  # Wrong: Cannot start with underscore

# CORRECT
labels:
  my-label: value-with-dashes
  private-label: value
```

#### Solutions

**Solution 1: Use validation tool**
```bash
# Validate before applying
gunj-migrate validate -f platform.yaml --strict

# Get detailed validation errors
kubectl apply -f platform.yaml --dry-run=server -o yaml 2>&1 | grep -A5 error
```

**Solution 2: Fix common issues**
```bash
#!/bin/bash
# fix-validation-errors.sh

# Fix resource quantities
yq e '.spec.components.*.resources.requests.memory |= sub("GB$", "Gi")' -i platform.yaml
yq e '.spec.components.*.resources.requests.cpu |= sub(" core$", "")' -i platform.yaml

# Fix version format
yq e '.spec.components.*.version |= "v" + .' -i platform.yaml

# Fix label format
yq e '.metadata.labels |= with_entries(.key |= gsub(" "; "-") | .value |= gsub(" "; "-"))' -i platform.yaml
```

### 3. Field Mapping Errors

#### Symptoms
- Warning: `unknown field`
- Fields missing after migration
- Unexpected behavior

#### Diagnosis
```bash
# Compare fields before and after
diff -u <(kubectl get observabilityplatform <name> -o yaml) \
        <(kubectl get observabilityplatform <name> -o yaml --output-version=observability.io/v1beta1)

# Check for deprecated fields
kubectl get observabilityplatform <name> -o yaml | grep -E "(customConfig|legacyConfig|deprecated)"
```

#### Solutions

**Solution 1: Manual field mapping**
```yaml
# Map customConfig to specific fields
# Before (v1alpha1)
customConfig:
  externalLabels: '{"cluster": "prod"}'
  
# After (v1beta1)
externalLabels:
  cluster: prod
```

**Solution 2: Use transformation script**
```python
# transform_fields.py
import yaml
import json

def transform_custom_config(custom_config):
    """Transform customConfig to v1beta1 fields."""
    result = {}
    
    if isinstance(custom_config, str):
        # Parse YAML or JSON string
        try:
            config = yaml.safe_load(custom_config)
        except:
            config = json.loads(custom_config)
    else:
        config = custom_config
    
    # Extract known fields
    if 'externalLabels' in config:
        labels = config['externalLabels']
        if isinstance(labels, str):
            result['externalLabels'] = json.loads(labels)
        else:
            result['externalLabels'] = labels
    
    if 'remoteWrite' in config:
        result['remoteWrite'] = config['remoteWrite']
    
    return result
```

### 4. Status Update Failures

#### Symptoms
- Status stuck in "Installing" or "Upgrading"
- Components show as not ready
- Reconciliation loops

#### Diagnosis
```bash
# Check platform status
kubectl get observabilityplatform <name> -o jsonpath='{.status}'

# Check operator logs
kubectl logs -n gunj-system deployment/gunj-operator --since=5m | grep <platform-name>

# Check events
kubectl get events --field-selector involvedObject.name=<platform-name> --sort-by='.lastTimestamp'
```

#### Solutions

**Solution 1: Force reconciliation**
```bash
# Add annotation to trigger reconciliation
kubectl annotate observabilityplatform <name> \
  observability.io/force-reconcile=$(date +%s) --overwrite

# Watch status updates
kubectl get observabilityplatform <name> -w
```

**Solution 2: Reset status**
```bash
# Patch status to trigger update
kubectl patch observabilityplatform <name> --type=json -p='[
  {"op": "remove", "path": "/status/conditions"},
  {"op": "replace", "path": "/status/phase", "value": "Pending"}
]'
```

### 5. Resource Conflicts

#### Symptoms
- Error: `Operation cannot be fulfilled on observabilityplatforms`
- Conflicting resource versions
- Update failures

#### Diagnosis
```bash
# Check resource version
kubectl get observabilityplatform <name> -o jsonpath='{.metadata.resourceVersion}'

# Check for multiple controllers
kubectl get pods -A -l app=gunj-operator

# Check for conflicting resources
kubectl get all -l observability.io/platform=<name>
```

#### Solutions

**Solution 1: Resolve conflicts**
```bash
# Get latest version
kubectl get observabilityplatform <name> -o yaml > latest.yaml

# Update and reapply
kubectl apply -f latest.yaml --force-conflicts
```

**Solution 2: Leader election issues**
```bash
# Check leader election
kubectl get configmap -n gunj-system gunj-operator-leader-election -o yaml

# Force new leader
kubectl delete configmap -n gunj-system gunj-operator-leader-election
```

## Diagnostic Tools

### Migration Health Check Script

```bash
#!/bin/bash
# migration-health-check.sh

echo "=== Migration Health Check ==="
echo

# Check operator health
echo "1. Checking Operator Health..."
operator_pods=$(kubectl get pods -n gunj-system -l app=gunj-operator -o json)
operator_ready=$(echo "$operator_pods" | jq '[.items[] | select(.status.phase == "Running")] | length')
operator_total=$(echo "$operator_pods" | jq '.items | length')
echo "   Operator Pods: $operator_ready/$operator_total ready"

# Check webhook health
echo "2. Checking Webhook Health..."
webhook_pods=$(kubectl get pods -n gunj-system -l app=gunj-operator-webhook -o json)
webhook_ready=$(echo "$webhook_pods" | jq '[.items[] | select(.status.phase == "Running")] | length')
webhook_total=$(echo "$webhook_pods" | jq '.items | length')
echo "   Webhook Pods: $webhook_ready/$webhook_total ready"

# Check conversion metrics
echo "3. Checking Conversion Metrics..."
if kubectl port-forward -n gunj-system svc/gunj-operator-metrics 8080:8080 > /dev/null 2>&1 &; then
    PF_PID=$!
    sleep 2
    
    conversions=$(curl -s http://localhost:8080/metrics | grep -E "gunj_operator_conversion_total" | grep -v "#" | awk '{sum += $2} END {print sum}')
    errors=$(curl -s http://localhost:8080/metrics | grep -E "gunj_operator_conversion_errors_total" | grep -v "#" | awk '{sum += $2} END {print sum}')
    
    echo "   Total Conversions: ${conversions:-0}"
    echo "   Conversion Errors: ${errors:-0}"
    
    kill $PF_PID 2>/dev/null
fi

# Check platform versions
echo "4. Checking Platform Versions..."
v1alpha1_count=$(kubectl get observabilityplatforms -A -o json | jq '[.items[] | select(.apiVersion == "observability.io/v1alpha1")] | length')
v1beta1_count=$(kubectl get observabilityplatforms -A -o json | jq '[.items[] | select(.apiVersion == "observability.io/v1beta1")] | length')
echo "   v1alpha1 Platforms: $v1alpha1_count"
echo "   v1beta1 Platforms: $v1beta1_count"

# Check for stuck migrations
echo "5. Checking for Stuck Migrations..."
stuck_platforms=$(kubectl get observabilityplatforms -A -o json | jq -r '.items[] | select(.status.phase == "Upgrading" and (.metadata.annotations."observability.io/last-transition-time" | fromdateiso8601) < (now - 300)) | "\(.metadata.namespace)/\(.metadata.name)"')
if [[ -n "$stuck_platforms" ]]; then
    echo "   ⚠️  Stuck platforms detected:"
    echo "$stuck_platforms" | sed 's/^/      - /'
else
    echo "   ✓ No stuck platforms"
fi

# Check for validation errors
echo "6. Checking Recent Errors..."
recent_errors=$(kubectl get events -A --field-selector type=Warning --sort-by='.lastTimestamp' | grep -i "observabilityplatform" | tail -5)
if [[ -n "$recent_errors" ]]; then
    echo "   Recent warnings:"
    echo "$recent_errors" | sed 's/^/      /'
else
    echo "   ✓ No recent warnings"
fi

echo
echo "=== Health Check Complete ==="
```

### Debug Information Collector

```python
#!/usr/bin/env python3
"""
collect_debug_info.py - Collects comprehensive debug information for migration issues
"""

import subprocess
import json
import yaml
import os
from datetime import datetime
from pathlib import Path

class DebugCollector:
    def __init__(self, output_dir: str = "./debug-info"):
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(exist_ok=True)
        
    def collect_all(self):
        """Collect all debug information."""
        print("Collecting debug information...")
        
        self.collect_platform_info()
        self.collect_operator_logs()
        self.collect_webhook_info()
        self.collect_events()
        self.collect_metrics()
        self.create_summary()
        
        print(f"\nDebug information collected in: {self.output_dir}")
        
    def collect_platform_info(self):
        """Collect platform information."""
        print("  - Collecting platform information...")
        
        # Get all platforms
        result = subprocess.run([
            "kubectl", "get", "observabilityplatforms", "-A", "-o", "json"
        ], capture_output=True, text=True)
        
        if result.returncode == 0:
            platforms = json.loads(result.stdout)
            
            # Save full dump
            with open(self.output_dir / "platforms.json", "w") as f:
                json.dump(platforms, f, indent=2)
                
            # Extract key information
            platform_summary = []
            for platform in platforms.get('items', []):
                summary = {
                    'name': platform['metadata']['name'],
                    'namespace': platform['metadata'].get('namespace', 'default'),
                    'apiVersion': platform['apiVersion'],
                    'phase': platform.get('status', {}).get('phase', 'Unknown'),
                    'message': platform.get('status', {}).get('message', ''),
                    'annotations': platform['metadata'].get('annotations', {}),
                    'generation': platform['metadata'].get('generation'),
                    'observedGeneration': platform.get('status', {}).get('observedGeneration')
                }
                platform_summary.append(summary)
                
            with open(self.output_dir / "platform-summary.yaml", "w") as f:
                yaml.dump(platform_summary, f, default_flow_style=False)
                
    def collect_operator_logs(self):
        """Collect operator logs."""
        print("  - Collecting operator logs...")
        
        # Get operator pods
        result = subprocess.run([
            "kubectl", "get", "pods", "-n", "gunj-system",
            "-l", "app=gunj-operator", "-o", "json"
        ], capture_output=True, text=True)
        
        if result.returncode == 0:
            pods = json.loads(result.stdout)
            
            for pod in pods.get('items', []):
                pod_name = pod['metadata']['name']
                
                # Get logs
                log_result = subprocess.run([
                    "kubectl", "logs", "-n", "gunj-system",
                    pod_name, "--tail=1000"
                ], capture_output=True, text=True)
                
                if log_result.returncode == 0:
                    with open(self.output_dir / f"operator-logs-{pod_name}.log", "w") as f:
                        f.write(log_result.stdout)
                        
    def collect_webhook_info(self):
        """Collect webhook information."""
        print("  - Collecting webhook information...")
        
        # Get webhook configurations
        for resource in ["validatingwebhookconfigurations", "mutatingwebhookconfigurations"]:
            result = subprocess.run([
                "kubectl", "get", resource,
                "gunj-operator-webhook", "-o", "yaml"
            ], capture_output=True, text=True)
            
            if result.returncode == 0:
                with open(self.output_dir / f"{resource}.yaml", "w") as f:
                    f.write(result.stdout)
                    
        # Get webhook pods logs
        result = subprocess.run([
            "kubectl", "get", "pods", "-n", "gunj-system",
            "-l", "app=gunj-operator-webhook", "-o", "json"
        ], capture_output=True, text=True)
        
        if result.returncode == 0:
            pods = json.loads(result.stdout)
            
            for pod in pods.get('items', []):
                pod_name = pod['metadata']['name']
                
                log_result = subprocess.run([
                    "kubectl", "logs", "-n", "gunj-system",
                    pod_name, "--tail=500"
                ], capture_output=True, text=True)
                
                if log_result.returncode == 0:
                    with open(self.output_dir / f"webhook-logs-{pod_name}.log", "w") as f:
                        f.write(log_result.stdout)
                        
    def collect_events(self):
        """Collect recent events."""
        print("  - Collecting events...")
        
        result = subprocess.run([
            "kubectl", "get", "events", "-A",
            "--sort-by='.lastTimestamp'", "-o", "json"
        ], capture_output=True, text=True)
        
        if result.returncode == 0:
            events = json.loads(result.stdout)
            
            # Filter relevant events
            relevant_events = []
            for event in events.get('items', []):
                if 'observability' in str(event).lower():
                    relevant_events.append({
                        'timestamp': event.get('lastTimestamp'),
                        'type': event.get('type'),
                        'reason': event.get('reason'),
                        'message': event.get('message'),
                        'object': f"{event['involvedObject']['kind']}/{event['involvedObject']['name']}",
                        'namespace': event['involvedObject'].get('namespace', 'default')
                    })
                    
            with open(self.output_dir / "events.yaml", "w") as f:
                yaml.dump(relevant_events, f, default_flow_style=False)
                
    def collect_metrics(self):
        """Collect conversion metrics."""
        print("  - Collecting metrics...")
        
        # Try to get metrics
        try:
            # Port forward in background
            pf_process = subprocess.Popen([
                "kubectl", "port-forward", "-n", "gunj-system",
                "svc/gunj-operator-metrics", "8080:8080"
            ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            
            import time
            time.sleep(2)  # Wait for port forward
            
            # Get metrics
            import urllib.request
            response = urllib.request.urlopen("http://localhost:8080/metrics")
            metrics = response.read().decode('utf-8')
            
            # Filter relevant metrics
            relevant_metrics = []
            for line in metrics.split('\n'):
                if 'gunj_operator' in line and not line.startswith('#'):
                    relevant_metrics.append(line)
                    
            with open(self.output_dir / "metrics.txt", "w") as f:
                f.write('\n'.join(relevant_metrics))
                
            pf_process.terminate()
            
        except Exception as e:
            print(f"    Warning: Could not collect metrics: {e}")
            
    def create_summary(self):
        """Create a summary report."""
        print("  - Creating summary report...")
        
        summary = {
            'timestamp': datetime.now().isoformat(),
            'cluster_info': self._get_cluster_info(),
            'operator_version': self._get_operator_version(),
            'platform_stats': self._get_platform_stats(),
            'health_status': self._get_health_status()
        }
        
        with open(self.output_dir / "summary.yaml", "w") as f:
            yaml.dump(summary, f, default_flow_style=False)
            
    def _get_cluster_info(self):
        """Get cluster information."""
        result = subprocess.run([
            "kubectl", "version", "-o", "json"
        ], capture_output=True, text=True)
        
        if result.returncode == 0:
            version_info = json.loads(result.stdout)
            return {
                'client_version': version_info.get('clientVersion', {}).get('gitVersion'),
                'server_version': version_info.get('serverVersion', {}).get('gitVersion')
            }
        return {}
        
    def _get_operator_version(self):
        """Get operator version."""
        result = subprocess.run([
            "kubectl", "get", "deployment", "-n", "gunj-system",
            "gunj-operator", "-o", "jsonpath={.spec.template.spec.containers[0].image}"
        ], capture_output=True, text=True)
        
        return result.stdout.strip() if result.returncode == 0 else "unknown"
        
    def _get_platform_stats(self):
        """Get platform statistics."""
        try:
            with open(self.output_dir / "platforms.json", "r") as f:
                platforms = json.load(f)
                
            stats = {
                'total': len(platforms.get('items', [])),
                'by_version': {},
                'by_phase': {}
            }
            
            for platform in platforms.get('items', []):
                version = platform['apiVersion'].split('/')[-1]
                phase = platform.get('status', {}).get('phase', 'Unknown')
                
                stats['by_version'][version] = stats['by_version'].get(version, 0) + 1
                stats['by_phase'][phase] = stats['by_phase'].get(phase, 0) + 1
                
            return stats
        except:
            return {}
            
    def _get_health_status(self):
        """Get overall health status."""
        issues = []
        
        # Check for webhook issues
        webhook_result = subprocess.run([
            "kubectl", "get", "pods", "-n", "gunj-system",
            "-l", "app=gunj-operator-webhook", "-o", "json"
        ], capture_output=True, text=True)
        
        if webhook_result.returncode == 0:
            pods = json.loads(webhook_result.stdout)
            not_ready = [p for p in pods.get('items', []) if p['status']['phase'] != 'Running']
            if not_ready:
                issues.append(f"Webhook pods not ready: {len(not_ready)}")
                
        # Check for stuck platforms
        try:
            with open(self.output_dir / "platform-summary.yaml", "r") as f:
                platforms = yaml.safe_load(f)
                
            stuck = [p for p in platforms if p['phase'] in ['Installing', 'Upgrading']]
            if stuck:
                issues.append(f"Platforms stuck in transition: {len(stuck)}")
        except:
            pass
            
        return {
            'healthy': len(issues) == 0,
            'issues': issues
        }


if __name__ == '__main__':
    collector = DebugCollector()
    collector.collect_all()
```

## Error Messages

### Comprehensive Error Reference

#### E001: Conversion Failed

**Error Message:**
```
conversion webhook for observability.io/v1beta1, Kind=ObservabilityPlatform failed: 
conversion failed: cannot convert customConfig: invalid JSON format
```

**Cause:** The customConfig field contains invalid JSON or YAML

**Solution:**
1. Validate JSON/YAML syntax
2. Use proper quoting for strings
3. Escape special characters

**Example Fix:**
```yaml
# Before (invalid)
customConfig:
  externalLabels: {"cluster": prod}  # Missing quotes

# After (valid)
customConfig:
  externalLabels: '{"cluster": "prod"}'
```

#### E002: Field Not Supported

**Error Message:**
```
error validating data: ValidationError(ObservabilityPlatform.spec.components.prometheus): 
unknown field "legacyConfig" in io.observability.v1beta1.PrometheusSpec
```

**Cause:** Using fields that don't exist in the target version

**Solution:**
1. Check field mapping documentation
2. Remove or transform unsupported fields
3. Use correct field names for target version

#### E003: Type Mismatch

**Error Message:**
```
error validating data: ValidationError(ObservabilityPlatform.spec.components.prometheus.externalLabels): 
invalid type for io.observability.v1beta1.PrometheusSpec.externalLabels: 
got "string", expected "map"
```

**Cause:** Field type changed between versions

**Solution:**
```bash
# Transform string to map
yq e '.spec.components.prometheus.externalLabels |= from_json' -i platform.yaml
```

#### E004: Resource Conflict

**Error Message:**
```
Operation cannot be fulfilled on observabilityplatforms.observability.io "production": 
the object has been modified; please apply your changes to the latest version and try again
```

**Cause:** Concurrent modifications to the resource

**Solution:**
1. Get latest version: `kubectl get observabilityplatform production -o yaml > latest.yaml`
2. Apply your changes to latest.yaml
3. Reapply: `kubectl apply -f latest.yaml`

#### E005: Webhook Timeout

**Error Message:**
```
Error from server (Timeout): error when creating "platform.yaml": 
Timeout: request did not complete within requested timeout 30s
```

**Cause:** Webhook not responding or overloaded

**Solution:**
1. Check webhook pod health
2. Increase webhook timeout
3. Scale webhook deployment

## Performance Issues

### Slow Conversion

#### Symptoms
- Conversions taking > 10 seconds
- Timeouts during migration
- High webhook CPU usage

#### Diagnosis
```bash
# Check webhook performance
kubectl top pods -n gunj-system -l app=gunj-operator-webhook

# Check conversion metrics
curl http://localhost:8080/metrics | grep conversion_duration_seconds

# Profile webhook
kubectl exec -n gunj-system deployment/gunj-operator-webhook -- \
  go tool pprof -http=:6060 http://localhost:6060/debug/pprof/profile
```

#### Solutions

**Solution 1: Scale webhook**
```bash
kubectl scale deployment/gunj-operator-webhook -n gunj-system --replicas=3
```

**Solution 2: Increase resources**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gunj-operator-webhook
spec:
  template:
    spec:
      containers:
      - name: webhook
        resources:
          requests:
            cpu: 500m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 512Mi
```

**Solution 3: Batch migrations**
```bash
# Migrate in smaller batches
for namespace in $(kubectl get ns -o name | cut -d/ -f2); do
    echo "Migrating namespace: $namespace"
    gunj-migrate convert --namespace "$namespace"
    sleep 30  # Allow webhook to recover
done
```

### Memory Issues

#### Symptoms
- OOMKilled webhook pods
- Large platforms failing to convert
- Memory spikes during migration

#### Solutions

**Solution 1: Increase memory limits**
```bash
kubectl patch deployment gunj-operator-webhook -n gunj-system --type=json -p='[
  {
    "op": "replace",
    "path": "/spec/template/spec/containers/0/resources/limits/memory",
    "value": "1Gi"
  }
]'
```

**Solution 2: Optimize large configs**
```python
# split_large_configs.py
def split_scrape_configs(configs, max_size=100):
    """Split large scrape configs into chunks."""
    chunks = []
    current_chunk = []
    
    for config in configs:
        if len(current_chunk) >= max_size:
            chunks.append(current_chunk)
            current_chunk = []
        current_chunk.append(config)
    
    if current_chunk:
        chunks.append(current_chunk)
    
    return chunks
```

## Data Integrity

### Verification Procedures

#### Pre-Migration Checksum

```bash
#!/bin/bash
# checksum-platforms.sh

for platform in $(kubectl get observabilityplatforms -A -o name); do
    namespace=$(echo "$platform" | cut -d/ -f2)
    name=$(echo "$platform" | cut -d/ -f3)
    
    # Calculate checksum of spec
    checksum=$(kubectl get "$platform" -o jsonpath='{.spec}' | sha256sum | cut -d' ' -f1)
    
    # Store checksum
    kubectl annotate "$platform" "observability.io/spec-checksum=$checksum" --overwrite
    
    echo "$namespace/$name: $checksum"
done
```

#### Post-Migration Verification

```python
#!/usr/bin/env python3
# verify_migration_integrity.py

import subprocess
import json
import hashlib

def verify_platform_integrity(name, namespace):
    """Verify platform data integrity after migration."""
    
    # Get current platform
    result = subprocess.run([
        "kubectl", "get", "observabilityplatform", name,
        "-n", namespace, "-o", "json"
    ], capture_output=True, text=True)
    
    if result.returncode != 0:
        return False, "Failed to get platform"
    
    platform = json.loads(result.stdout)
    
    # Check for data loss indicators
    issues = []
    
    # Check if security config was lost (v1beta1 feature)
    if 'security' not in platform['spec']:
        issues.append("Security configuration missing")
    
    # Check if monitoring config was lost
    if 'monitoring' not in platform['spec']:
        issues.append("Monitoring configuration missing")
    
    # Verify component counts
    components = platform['spec'].get('components', {})
    enabled_components = sum(1 for c in components.values() if c.get('enabled'))
    
    if enabled_components == 0:
        issues.append("No components enabled")
    
    # Check for null values
    def check_nulls(obj, path=""):
        if isinstance(obj, dict):
            for k, v in obj.items():
                if v is None:
                    issues.append(f"Null value at {path}.{k}")
                else:
                    check_nulls(v, f"{path}.{k}")
        elif isinstance(obj, list):
            for i, item in enumerate(obj):
                check_nulls(item, f"{path}[{i}]")
    
    check_nulls(platform['spec'])
    
    return len(issues) == 0, issues

# Verify all platforms
platforms = subprocess.run([
    "kubectl", "get", "observabilityplatforms", "-A",
    "-o", "custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name",
    "--no-headers"
], capture_output=True, text=True).stdout.strip().split('\n')

for platform_line in platforms:
    if platform_line:
        namespace, name = platform_line.split()
        valid, issues = verify_platform_integrity(name, namespace)
        
        if not valid:
            print(f"❌ {namespace}/{name}: {', '.join(issues)}")
        else:
            print(f"✅ {namespace}/{name}: Integrity verified")
```

### Data Recovery

#### Recover from Backup

```bash
#!/bin/bash
# recover-platform.sh

PLATFORM=$1
NAMESPACE=$2
BACKUP_DIR=$3

if [[ -z "$PLATFORM" || -z "$NAMESPACE" || -z "$BACKUP_DIR" ]]; then
    echo "Usage: $0 <platform> <namespace> <backup-dir>"
    exit 1
fi

backup_file="$BACKUP_DIR/${NAMESPACE}-${PLATFORM}.yaml"

if [[ ! -f "$backup_file" ]]; then
    echo "Backup file not found: $backup_file"
    exit 1
fi

# Delete corrupted platform
kubectl delete observabilityplatform "$PLATFORM" -n "$NAMESPACE" --wait=false

# Restore from backup
kubectl apply -f "$backup_file"

# Verify restoration
kubectl wait --for=condition=Ready observabilityplatform/"$PLATFORM" -n "$NAMESPACE" --timeout=300s
```

#### Reconstruct Missing Data

```python
# reconstruct_data.py
def reconstruct_security_config(platform):
    """Reconstruct default security configuration."""
    if 'security' not in platform['spec']:
        platform['spec']['security'] = {
            'tls': {
                'enabled': True,
                'certManager': {
                    'enabled': True
                }
            },
            'podSecurityPolicy': True,
            'networkPolicy': True,
            'rbac': {
                'create': True
            }
        }
    return platform

def reconstruct_monitoring_config(platform):
    """Reconstruct default monitoring configuration."""
    if 'monitoring' not in platform['spec']:
        platform['spec']['monitoring'] = {
            'selfMonitoring': True,
            'serviceMonitor': {
                'enabled': True
            },
            'prometheusRule': {
                'enabled': True
            },
            'grafanaDashboard': {
                'enabled': True
            }
        }
    return platform
```

## Recovery Procedures

### Emergency Rollback

```bash
#!/bin/bash
# emergency-rollback.sh

echo "🚨 EMERGENCY ROLLBACK INITIATED 🚨"

# Stop operator to prevent further changes
kubectl scale deployment/gunj-operator -n gunj-system --replicas=0
kubectl scale deployment/gunj-operator-webhook -n gunj-system --replicas=0

echo "✓ Operator stopped"

# Find latest backup
LATEST_BACKUP=$(ls -t backups/*.tar.gz | head -1)

if [[ -z "$LATEST_BACKUP" ]]; then
    echo "❌ No backup found!"
    exit 1
fi

echo "📦 Using backup: $LATEST_BACKUP"

# Extract backup
TEMP_DIR=$(mktemp -d)
tar -xzf "$LATEST_BACKUP" -C "$TEMP_DIR"

# Delete all current platforms
kubectl delete observabilityplatforms --all -A --wait=false

# Restore from backup
find "$TEMP_DIR" -name "*.yaml" -type f | while read -r file; do
    echo "Restoring: $(basename "$file")"
    kubectl apply -f "$file"
done

# Restart operator
kubectl scale deployment/gunj-operator -n gunj-system --replicas=1
kubectl scale deployment/gunj-operator-webhook -n gunj-system --replicas=2

echo "✅ Rollback complete"

# Cleanup
rm -rf "$TEMP_DIR"
```

### Partial Recovery

```python
#!/usr/bin/env python3
# partial_recovery.py

import argparse
import subprocess
import yaml
import json

def recover_component(platform_name, namespace, component, backup_file):
    """Recover a single component from backup."""
    
    # Load backup
    with open(backup_file, 'r') as f:
        backup = yaml.safe_load(f)
    
    # Extract component config
    component_config = backup['spec']['components'].get(component)
    
    if not component_config:
        print(f"Component {component} not found in backup")
        return False
    
    # Create patch
    patch = [{
        "op": "replace",
        "path": f"/spec/components/{component}",
        "value": component_config
    }]
    
    # Apply patch
    result = subprocess.run([
        "kubectl", "patch", "observabilityplatform", platform_name,
        "-n", namespace, "--type=json", "-p", json.dumps(patch)
    ], capture_output=True, text=True)
    
    if result.returncode == 0:
        print(f"✓ Recovered {component} configuration")
        return True
    else:
        print(f"✗ Failed to recover {component}: {result.stderr}")
        return False

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('platform', help='Platform name')
    parser.add_argument('namespace', help='Namespace')
    parser.add_argument('component', help='Component to recover')
    parser.add_argument('backup', help='Backup file')
    
    args = parser.parse_args()
    
    success = recover_component(
        args.platform,
        args.namespace,
        args.component,
        args.backup
    )
    
    exit(0 if success else 1)
```

## Advanced Debugging

### Enable Debug Logging

```bash
# Enable debug logging for operator
kubectl set env deployment/gunj-operator -n gunj-system LOG_LEVEL=debug

# Enable debug logging for webhook
kubectl set env deployment/gunj-operator-webhook -n gunj-system LOG_LEVEL=debug

# Enable verbose conversion logging
kubectl annotate observabilityplatform <name> \
  observability.io/debug-conversion=true --overwrite
```

### Trace Conversion Process

```go
// Add to webhook for detailed tracing
func (w *WebhookServer) convertWithTrace(req *v1.ConversionRequest) (*v1.ConversionResponse, error) {
    trace := &ConversionTrace{
        RequestID: uuid.New().String(),
        StartTime: time.Now(),
        Steps:     []TraceStep{},
    }
    
    // Log each conversion step
    trace.AddStep("Starting conversion", map[string]interface{}{
        "from_version": req.DesiredAPIVersion,
        "object_count": len(req.Objects),
    })
    
    // ... conversion logic with trace points ...
    
    // Save trace for debugging
    w.saveTrace(trace)
    
    return response, nil
}
```

### Webhook Debugging Proxy

```python
#!/usr/bin/env python3
# webhook_debug_proxy.py

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import requests
import logging

class WebhookDebugProxy(BaseHTTPRequestHandler):
    def do_POST(self):
        # Read request
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        # Log request
        logging.info(f"Webhook request: {post_data.decode()[:200]}...")
        
        # Forward to actual webhook
        response = requests.post(
            'https://gunj-operator-webhook.gunj-system.svc:443/convert',
            data=post_data,
            headers=dict(self.headers),
            verify=False
        )
        
        # Log response
        logging.info(f"Webhook response: {response.status_code}")
        
        # Return response
        self.send_response(response.status_code)
        for key, value in response.headers.items():
            self.send_header(key, value)
        self.end_headers()
        self.wfile.write(response.content)

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    server = HTTPServer(('localhost', 8443), WebhookDebugProxy)
    print("Debug proxy listening on port 8443")
    server.serve_forever()
```

### Live Debugging Session

```bash
# Start debug pod
kubectl run debug-pod --rm -it --image=gunjanjp/gunj-debug:latest \
  --namespace=gunj-system \
  --overrides='{"spec":{"serviceAccount":"gunj-operator"}}'

# Inside debug pod
# Install debugging tools
apt-get update && apt-get install -y curl jq dnsutils

# Test webhook connectivity
curl -k https://gunj-operator-webhook:443/healthz

# Test conversion endpoint
curl -k -X POST https://gunj-operator-webhook:443/convert \
  -H "Content-Type: application/json" \
  -d @test-conversion.json

# Check DNS resolution
nslookup gunj-operator-webhook.gunj-system.svc.cluster.local

# Trace network path
traceroute gunj-operator-webhook.gunj-system.svc.cluster.local
```

## FAQ

### Q: Can I skip the webhook entirely?

**A:** Yes, but it's not recommended. In emergency:
```bash
# Temporarily disable webhook
kubectl delete validatingwebhookconfiguration gunj-operator-webhook
kubectl delete mutatingwebhookconfiguration gunj-operator-webhook

# Perform migration manually
# ... 

# Restore webhooks
kubectl apply -f webhook-config.yaml
```

### Q: How do I migrate a single field that's failing?

**A:** Use targeted patching:
```bash
# Patch specific field
kubectl patch observabilityplatform <name> --type='json' -p='[{
  "op": "remove",
  "path": "/spec/components/prometheus/customConfig"
},{
  "op": "add",
  "path": "/spec/components/prometheus/externalLabels",
  "value": {"cluster": "prod"}
}]'
```

### Q: What if my platform is too large to convert?

**A:** Split into smaller configs:
1. Export platform: `kubectl get observabilityplatform <name> -o yaml > platform.yaml`
2. Split components into separate files
3. Migrate each component separately
4. Recombine after migration

### Q: Can I test conversion without applying?

**A:** Yes, use dry-run:
```bash
# Test conversion
kubectl apply -f platform.yaml --dry-run=server -o yaml

# Test with webhook
curl -k -X POST https://localhost:8443/convert \
  -H "Content-Type: application/json" \
  -d '{"desiredAPIVersion":"observability.io/v1beta1","objects":[...]}'
```

### Q: How do I know if conversion succeeded?

**A:** Check multiple indicators:
1. Platform phase: Should be "Ready"
2. All components running: `kubectl get pods -l observability.io/platform=<name>`
3. No error events: `kubectl get events --field-selector involvedObject.name=<name>`
4. Conversion metrics show success

### Q: What's the safest migration approach?

**A:** Follow this order:
1. Test in dev environment
2. Backup production
3. Migrate during maintenance window
4. Use canary approach (one platform at a time)
5. Monitor for 24 hours before proceeding

## Getting Help

### Self-Help Resources
- Check this troubleshooting guide
- Review error messages in [Error Reference](#error-messages)
- Run diagnostic scripts
- Check operator logs

### Community Support
- Slack: #gunj-operator-help
- GitHub Discussions: https://github.com/gunjanjp/gunj-operator/discussions
- Stack Overflow: Tag `gunj-operator`

### Commercial Support
- Email: gunjanjp@gmail.com
- Enterprise support contracts available
- On-site migration assistance
- Custom tooling development

### Reporting Issues
When reporting issues, include:
1. Output of `migration-health-check.sh`
2. Debug bundle from `collect_debug_info.py`
3. Specific error messages
4. Steps to reproduce
5. Expected vs actual behavior
