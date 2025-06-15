# Upgrade Procedures and Best Practices

## Table of Contents

- [Overview](#overview)
- [Pre-Upgrade Planning](#pre-upgrade-planning)
- [Upgrade Methods](#upgrade-methods)
- [Step-by-Step Procedures](#step-by-step-procedures)
- [Best Practices](#best-practices)
- [Rollback Procedures](#rollback-procedures)
- [Troubleshooting](#troubleshooting)
- [Post-Upgrade Validation](#post-upgrade-validation)
- [Automation Scripts](#automation-scripts)

## Overview

This guide provides comprehensive procedures for upgrading Gunj Operator deployments from v1alpha1 to v1beta1. Choose the upgrade method that best fits your requirements for downtime, risk tolerance, and complexity.

### Upgrade Methods Comparison

| Method | Downtime | Risk | Complexity | Best For |
|--------|----------|------|------------|----------|
| In-Place | 5-10 min | Medium | Low | Development, small deployments |
| Rolling | Zero | Low | Medium | Production with single cluster |
| Blue-Green | Zero | Very Low | High | Critical production systems |
| Canary | Zero | Very Low | Very High | Large-scale deployments |
| GitOps | Zero | Low | Medium | GitOps-managed environments |

## Pre-Upgrade Planning

### 1. Compatibility Assessment

```bash
#!/bin/bash
# compatibility-check.sh

echo "=== Gunj Operator Upgrade Compatibility Check ==="
echo ""

# Check Kubernetes version
K8S_VERSION=$(kubectl version -o json | jq -r '.serverVersion.gitVersion')
echo "Kubernetes Version: $K8S_VERSION"

# Check operator version
OPERATOR_VERSION=$(kubectl get deployment -n gunj-system gunj-operator -o jsonpath='{.spec.template.spec.containers[0].image}' | cut -d: -f2)
echo "Current Operator Version: $OPERATOR_VERSION"

# Check API versions
echo ""
echo "Available API Versions:"
kubectl api-versions | grep observability

# Check for platforms
echo ""
echo "ObservabilityPlatforms:"
kubectl get observabilityplatforms.observability.io --all-namespaces -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,VERSION:.apiVersion,STATUS:.status.phase

# Check for deprecated fields
echo ""
echo "Checking for deprecated fields..."
./gunj-migrate check --quiet || echo "No deprecated fields found"

# Resource usage
echo ""
echo "Current Resource Usage:"
kubectl top nodes
kubectl top pods -n gunj-system

# Storage check
echo ""
echo "Persistent Volumes:"
kubectl get pv | grep -E "prometheus|grafana|loki|tempo"

echo ""
echo "=== Assessment Complete ==="
```

### 2. Risk Assessment Matrix

| Risk Factor | Low Risk | Medium Risk | High Risk | Mitigation |
|-------------|----------|-------------|-----------|------------|
| Cluster Size | < 50 nodes | 50-200 nodes | > 200 nodes | Use canary deployment |
| Data Volume | < 100GB | 100GB-1TB | > 1TB | Ensure backup complete |
| User Impact | Dev only | Internal users | External customers | Blue-green deployment |
| Customization | Minimal | Moderate | Extensive | Extended testing |
| Integration | Standalone | Few integrations | Many integrations | Phased approach |

### 3. Upgrade Checklist

- [ ] **Backups**
  - [ ] Platform configurations backed up
  - [ ] Persistent data backed up
  - [ ] Operator configuration saved
  - [ ] RBAC settings exported
  
- [ ] **Testing**
  - [ ] Upgrade tested in dev environment
  - [ ] Rollback tested
  - [ ] Performance benchmarks collected
  - [ ] Integration tests passed
  
- [ ] **Communication**
  - [ ] Maintenance window scheduled
  - [ ] Stakeholders notified
  - [ ] Runbooks updated
  - [ ] Support team briefed
  
- [ ] **Resources**
  - [ ] Additional resources allocated
  - [ ] Quota limits checked
  - [ ] Storage capacity verified
  - [ ] Network bandwidth available

## Upgrade Methods

### Method 1: In-Place Upgrade

**Best for**: Development environments, small deployments

**Pros**: Simple, fast, minimal resources
**Cons**: Downtime required, higher risk

```bash
#!/bin/bash
# in-place-upgrade.sh

set -euo pipefail

NAMESPACE="gunj-system"
BACKUP_DIR="./backups/$(date +%Y%m%d-%H%M%S)"

echo "Starting in-place upgrade..."

# Step 1: Create backups
echo "Creating backups..."
mkdir -p "$BACKUP_DIR"
kubectl get observabilityplatforms.observability.io --all-namespaces -o yaml > "$BACKUP_DIR/platforms.yaml"
kubectl get all -n "$NAMESPACE" -o yaml > "$BACKUP_DIR/operator.yaml"

# Step 2: Scale down operator
echo "Scaling down operator..."
kubectl scale deployment -n "$NAMESPACE" gunj-operator --replicas=0

# Step 3: Update CRDs
echo "Updating CRDs..."
kubectl apply -f https://github.com/gunjanjp/gunj-operator/releases/download/v1.5.0/crds.yaml

# Step 4: Update operator
echo "Updating operator..."
kubectl set image deployment/gunj-operator -n "$NAMESPACE" \
  operator=gunjanjp/gunj-operator:v1.5.0

# Step 5: Scale up operator
echo "Scaling up operator..."
kubectl scale deployment -n "$NAMESPACE" gunj-operator --replicas=1

# Step 6: Wait for operator
echo "Waiting for operator to be ready..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/gunj-operator -n "$NAMESPACE"

# Step 7: Trigger conversions
echo "Triggering platform conversions..."
kubectl get observabilityplatforms.observability.io --all-namespaces -o json | \
  jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"' | \
  while read platform; do
    namespace=$(echo $platform | cut -d/ -f1)
    name=$(echo $platform | cut -d/ -f2)
    kubectl annotate observabilityplatform "$name" -n "$namespace" \
      observability.io/trigger-conversion=true --overwrite
  done

echo "In-place upgrade complete!"
```

### Method 2: Rolling Upgrade

**Best for**: Production with moderate availability requirements

**Pros**: Zero downtime, moderate complexity
**Cons**: Temporary resource doubling

```yaml
# rolling-upgrade-strategy.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gunj-operator
  namespace: gunj-system
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
      - name: operator
        image: gunjanjp/gunj-operator:v1.5.0
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
```

```bash
#!/bin/bash
# rolling-upgrade.sh

# Step 1: Increase replicas for HA
kubectl scale deployment -n gunj-system gunj-operator --replicas=3

# Step 2: Apply rolling update
kubectl apply -f rolling-upgrade-strategy.yaml

# Step 3: Monitor rollout
kubectl rollout status deployment/gunj-operator -n gunj-system

# Step 4: Verify all pods are running new version
kubectl get pods -n gunj-system -l app=gunj-operator -o jsonpath='{.items[*].spec.containers[0].image}'
```

### Method 3: Blue-Green Upgrade

**Best for**: Critical production systems

**Pros**: Instant rollback, zero downtime, full testing
**Cons**: Double resources required

```yaml
# blue-green-setup.yaml
---
# Green deployment (new version)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gunj-operator-green
  namespace: gunj-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gunj-operator
      version: green
  template:
    metadata:
      labels:
        app: gunj-operator
        version: green
    spec:
      containers:
      - name: operator
        image: gunjanjp/gunj-operator:v1.5.0
---
# Service selector will be switched
apiVersion: v1
kind: Service
metadata:
  name: gunj-operator
  namespace: gunj-system
spec:
  selector:
    app: gunj-operator
    version: blue  # Currently pointing to blue
  ports:
  - port: 8080
    targetPort: 8080
```

```bash
#!/bin/bash
# blue-green-upgrade.sh

# Step 1: Deploy green version
echo "Deploying green version..."
kubectl apply -f blue-green-setup.yaml

# Step 2: Wait for green deployment
echo "Waiting for green deployment..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/gunj-operator-green -n gunj-system

# Step 3: Test green version
echo "Testing green version..."
GREEN_POD=$(kubectl get pod -n gunj-system -l version=green -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n gunj-system $GREEN_POD -- gunj-operator version

# Step 4: Switch traffic to green
echo "Switching traffic to green..."
kubectl patch service gunj-operator -n gunj-system -p '{"spec":{"selector":{"version":"green"}}}'

# Step 5: Verify switch
echo "Verifying traffic switch..."
sleep 10
kubectl get endpoints gunj-operator -n gunj-system

# Step 6: Keep blue for rollback (optional)
echo "Blue deployment kept for quick rollback"
echo "To rollback: kubectl patch service gunj-operator -n gunj-system -p '{\"spec\":{\"selector\":{\"version\":\"blue\"}}}'"
```

### Method 4: Canary Upgrade

**Best for**: Large-scale deployments with gradual rollout

**Pros**: Minimal blast radius, gradual rollout
**Cons**: Complex setup, extended timeline

```yaml
# canary-deployment.yaml
apiVersion: v1
kind: Service
metadata:
  name: gunj-operator
  namespace: gunj-system
spec:
  selector:
    app: gunj-operator
  ports:
  - port: 8080
---
# Stable version (90% traffic)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gunj-operator-stable
  namespace: gunj-system
spec:
  replicas: 9
  selector:
    matchLabels:
      app: gunj-operator
      version: stable
  template:
    metadata:
      labels:
        app: gunj-operator
        version: stable
    spec:
      containers:
      - name: operator
        image: gunjanjp/gunj-operator:v1.4.5
---
# Canary version (10% traffic)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gunj-operator-canary
  namespace: gunj-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gunj-operator
      version: canary
  template:
    metadata:
      labels:
        app: gunj-operator
        version: canary
    spec:
      containers:
      - name: operator
        image: gunjanjp/gunj-operator:v1.5.0
```

```bash
#!/bin/bash
# canary-upgrade.sh

# Function to adjust canary percentage
adjust_canary() {
  local stable_replicas=$1
  local canary_replicas=$2
  
  kubectl scale deployment -n gunj-system gunj-operator-stable --replicas=$stable_replicas
  kubectl scale deployment -n gunj-system gunj-operator-canary --replicas=$canary_replicas
  
  echo "Traffic split: Stable=$stable_replicas, Canary=$canary_replicas"
}

# Step 1: Start with 10% canary
echo "Starting canary deployment (10%)..."
adjust_canary 9 1

# Step 2: Monitor metrics
echo "Monitoring canary metrics for 1 hour..."
sleep 3600

# Step 3: Check canary health
ERROR_RATE=$(kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant 'rate(gunj_operator_errors_total{version="canary"}[5m])' | \
  jq -r '.data.result[0].value[1]')

if (( $(echo "$ERROR_RATE > 0.01" | bc -l) )); then
  echo "High error rate detected! Rolling back..."
  adjust_canary 10 0
  exit 1
fi

# Step 4: Increase to 50%
echo "Increasing canary to 50%..."
adjust_canary 5 5
sleep 3600

# Step 5: Full rollout
echo "Completing canary rollout..."
adjust_canary 0 10

# Step 6: Rename canary to stable
kubectl patch deployment gunj-operator-canary -n gunj-system \
  --type='json' -p='[{"op": "replace", "path": "/metadata/name", "value": "gunj-operator-stable"}]'
```

### Method 5: GitOps-Based Upgrade

**Best for**: GitOps-managed environments (ArgoCD/Flux)

**Pros**: Auditable, repeatable, integrated with CI/CD
**Cons**: Requires GitOps setup

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: gunj-system

resources:
  - https://github.com/gunjanjp/gunj-operator/releases/download/v1.5.0/operator.yaml

images:
  - name: gunjanjp/gunj-operator
    newTag: v1.5.0

patches:
  - target:
      kind: Deployment
      name: gunj-operator
    patch: |-
      - op: add
        path: /spec/template/metadata/annotations
        value:
          upgraded-at: "2024-06-15"
          upgraded-to: "v1.5.0"

configMapGenerator:
  - name: upgrade-config
    literals:
      - ENABLE_CONVERSION_WEBHOOK=true
      - LOG_LEVEL=debug
```

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: gunj-operator-upgrade
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/yourorg/gitops-configs
    targetRevision: feature/upgrade-v1.5.0
    path: gunj-operator
  destination:
    server: https://kubernetes.default.svc
    namespace: gunj-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

## Step-by-Step Procedures

### Comprehensive Upgrade Procedure

```bash
#!/bin/bash
# comprehensive-upgrade.sh

set -euo pipefail

# Configuration
OPERATOR_NAMESPACE="gunj-system"
TARGET_VERSION="v1.5.0"
BACKUP_RETENTION_DAYS=30
LOG_FILE="upgrade-$(date +%Y%m%d-%H%M%S).log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE"
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" | tee -a "$LOG_FILE"
}

# Pre-flight checks
preflight_checks() {
    log "Running pre-flight checks..."
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
    fi
    
    # Check operator exists
    if ! kubectl get deployment -n "$OPERATOR_NAMESPACE" gunj-operator &> /dev/null; then
        error "Gunj operator not found in namespace $OPERATOR_NAMESPACE"
    fi
    
    # Check for sufficient resources
    local nodes=$(kubectl get nodes -o json | jq '.items | length')
    if [ "$nodes" -lt 3 ]; then
        warning "Less than 3 nodes available. Consider adding more nodes for HA."
    fi
    
    log "Pre-flight checks completed successfully"
}

# Create comprehensive backup
create_backup() {
    local backup_dir="./backups/upgrade-$(date +%Y%m%d-%H%M%S)"
    log "Creating comprehensive backup in $backup_dir..."
    
    mkdir -p "$backup_dir"
    
    # Backup CRDs
    kubectl get crd -o yaml | grep observability > "$backup_dir/crds.yaml"
    
    # Backup all platforms
    kubectl get observabilityplatforms.observability.io --all-namespaces -o yaml > "$backup_dir/platforms.yaml"
    
    # Backup operator deployment
    kubectl get all -n "$OPERATOR_NAMESPACE" -o yaml > "$backup_dir/operator-resources.yaml"
    
    # Backup RBAC
    kubectl get clusterrole,clusterrolebinding -o yaml | grep gunj > "$backup_dir/rbac.yaml"
    
    # Backup secrets and configmaps
    kubectl get secrets,configmaps -n "$OPERATOR_NAMESPACE" -o yaml > "$backup_dir/configs.yaml"
    
    # Create backup manifest
    cat > "$backup_dir/manifest.json" <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "operator_version": "$(kubectl get deployment -n $OPERATOR_NAMESPACE gunj-operator -o jsonpath='{.spec.template.spec.containers[0].image}')",
  "kubernetes_version": "$(kubectl version -o json | jq -r .serverVersion.gitVersion)",
  "platform_count": $(kubectl get observabilityplatforms.observability.io --all-namespaces --no-headers | wc -l)
}
EOF
    
    # Compress backup
    tar -czf "$backup_dir.tar.gz" -C "$(dirname "$backup_dir")" "$(basename "$backup_dir")"
    
    log "Backup completed: $backup_dir.tar.gz"
}

# Health check function
health_check() {
    local component=$1
    local namespace=${2:-$OPERATOR_NAMESPACE}
    
    log "Checking health of $component..."
    
    # Check pods
    local pods=$(kubectl get pods -n "$namespace" -l app="$component" -o json)
    local ready=$(echo "$pods" | jq '[.items[] | select(.status.phase == "Running")] | length')
    local total=$(echo "$pods" | jq '.items | length')
    
    if [ "$ready" -eq "$total" ] && [ "$total" -gt 0 ]; then
        log "✓ $component: $ready/$total pods ready"
        return 0
    else
        warning "✗ $component: only $ready/$total pods ready"
        return 1
    fi
}

# Upgrade CRDs
upgrade_crds() {
    log "Upgrading CRDs..."
    
    # Download new CRDs
    curl -sL "https://github.com/gunjanjp/gunj-operator/releases/download/$TARGET_VERSION/crds.yaml" -o /tmp/crds.yaml
    
    # Apply CRDs
    kubectl apply -f /tmp/crds.yaml
    
    # Wait for CRD establishment
    kubectl wait --for condition=established --timeout=60s crd/observabilityplatforms.observability.io
    
    log "CRDs upgraded successfully"
}

# Upgrade operator
upgrade_operator() {
    log "Upgrading operator to $TARGET_VERSION..."
    
    # Update operator image
    kubectl set image deployment/gunj-operator -n "$OPERATOR_NAMESPACE" \
        operator="gunjanjp/gunj-operator:$TARGET_VERSION"
    
    # Wait for rollout
    kubectl rollout status deployment/gunj-operator -n "$OPERATOR_NAMESPACE" --timeout=600s
    
    # Verify new version
    local new_version=$(kubectl get deployment -n "$OPERATOR_NAMESPACE" gunj-operator \
        -o jsonpath='{.spec.template.spec.containers[0].image}' | cut -d: -f2)
    
    if [ "$new_version" != "$TARGET_VERSION" ]; then
        error "Operator version mismatch. Expected $TARGET_VERSION, got $new_version"
    fi
    
    log "Operator upgraded successfully"
}

# Convert platforms
convert_platforms() {
    log "Converting platforms to v1beta1..."
    
    local platforms=$(kubectl get observabilityplatforms.observability.io --all-namespaces -o json)
    local count=$(echo "$platforms" | jq '.items | length')
    
    log "Found $count platforms to convert"
    
    echo "$platforms" | jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"' | while read platform; do
        local namespace=$(echo "$platform" | cut -d/ -f1)
        local name=$(echo "$platform" | cut -d/ -f2)
        
        log "Converting platform $namespace/$name..."
        
        # Trigger conversion
        kubectl annotate observabilityplatform "$name" -n "$namespace" \
            observability.io/convert-to-v1beta1=true \
            observability.io/conversion-timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            --overwrite
        
        # Wait for conversion
        local retries=0
        while [ $retries -lt 30 ]; do
            local version=$(kubectl get observabilityplatform "$name" -n "$namespace" -o jsonpath='{.apiVersion}')
            if [[ "$version" == *"v1beta1"* ]]; then
                log "✓ Platform $namespace/$name converted successfully"
                break
            fi
            sleep 2
            ((retries++))
        done
        
        if [ $retries -eq 30 ]; then
            warning "Platform $namespace/$name conversion timeout"
        fi
    done
    
    log "Platform conversion completed"
}

# Validate upgrade
validate_upgrade() {
    log "Validating upgrade..."
    
    local all_good=true
    
    # Check operator health
    if ! health_check "gunj-operator"; then
        all_good=false
    fi
    
    # Check conversion webhook
    if ! health_check "gunj-operator-webhook"; then
        warning "Conversion webhook not healthy"
    fi
    
    # Check all platforms
    kubectl get observabilityplatforms.observability.io --all-namespaces -o json | \
        jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)/\(.status.phase)"' | \
        while IFS='/' read namespace name phase; do
            if [ "$phase" != "Ready" ]; then
                warning "Platform $namespace/$name is not ready (phase: $phase)"
                all_good=false
            fi
        done
    
    # Check component health
    kubectl get observabilityplatforms.observability.io --all-namespaces -o json | \
        jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"' | \
        while IFS='/' read namespace name; do
            for component in prometheus grafana loki tempo; do
                if kubectl get deployment -n "$namespace" "$component" &> /dev/null; then
                    if ! health_check "$component" "$namespace"; then
                        all_good=false
                    fi
                fi
            done
        done
    
    if [ "$all_good" = true ]; then
        log "✓ Upgrade validation passed"
    else
        error "✗ Upgrade validation failed"
    fi
}

# Performance validation
performance_validation() {
    log "Running performance validation..."
    
    # Query Prometheus for key metrics
    local prometheus_pod=$(kubectl get pod -n monitoring -l app=prometheus -o jsonpath='{.items[0].metadata.name}')
    
    if [ -n "$prometheus_pod" ]; then
        # Check reconciliation rate
        local recon_rate=$(kubectl exec -n monitoring "$prometheus_pod" -- \
            promtool query instant 'rate(gunj_operator_reconciliations_total[5m])' | \
            jq -r '.data.result[0].value[1]' || echo "0")
        
        log "Reconciliation rate: $recon_rate/sec"
        
        # Check error rate
        local error_rate=$(kubectl exec -n monitoring "$prometheus_pod" -- \
            promtool query instant 'rate(gunj_operator_reconciliation_errors_total[5m])' | \
            jq -r '.data.result[0].value[1]' || echo "0")
        
        if (( $(echo "$error_rate > 0.1" | bc -l) )); then
            warning "High error rate detected: $error_rate/sec"
        else
            log "Error rate: $error_rate/sec (acceptable)"
        fi
    fi
}

# Cleanup old resources
cleanup_old_resources() {
    log "Cleaning up old resources..."
    
    # Remove old CRD versions if no longer needed
    if kubectl get crd observabilityplatforms.observability.io -o json | jq -e '.spec.versions[] | select(.name == "v1alpha1" and .served == false)' > /dev/null; then
        log "Removing v1alpha1 from CRD"
        # This would require patching the CRD to remove the version
    fi
    
    # Clean up old backups
    find ./backups -name "*.tar.gz" -mtime +$BACKUP_RETENTION_DAYS -delete
    
    log "Cleanup completed"
}

# Main upgrade flow
main() {
    log "Starting Gunj Operator upgrade to $TARGET_VERSION"
    log "Upgrade log: $LOG_FILE"
    
    # Phase 1: Preparation
    preflight_checks
    create_backup
    
    # Confirmation
    echo ""
    read -p "Ready to proceed with upgrade? (yes/no): " confirm
    if [ "$confirm" != "yes" ]; then
        log "Upgrade cancelled by user"
        exit 0
    fi
    
    # Phase 2: Upgrade
    upgrade_crds
    upgrade_operator
    
    # Phase 3: Conversion
    convert_platforms
    
    # Phase 4: Validation
    validate_upgrade
    performance_validation
    
    # Phase 5: Cleanup
    cleanup_old_resources
    
    log "✓ Upgrade completed successfully!"
    log "Next steps:"
    log "1. Monitor platform health for 24 hours"
    log "2. Review performance metrics"
    log "3. Update documentation and runbooks"
    log "4. Remove old version deployments if using blue-green"
}

# Run main function
main "$@"
```

## Best Practices

### 1. Timing and Scheduling

#### Optimal Upgrade Windows

| Environment | Best Time | Duration | Frequency |
|-------------|-----------|----------|-----------|
| Development | Anytime | 30 min | Weekly |
| Staging | Off-hours | 1 hour | Bi-weekly |
| Production | Weekend night | 2-4 hours | Monthly |

#### Scheduling Considerations

```yaml
# maintenance-window.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: maintenance-windows
  namespace: gunj-system
data:
  schedule: |
    - name: development
      window: "* * * * *"  # Anytime
      duration: 30m
      
    - name: staging  
      window: "0 2 * * 1-5"  # 2 AM weekdays
      duration: 1h
      
    - name: production
      window: "0 2 * * 0"  # 2 AM Sunday
      duration: 4h
      notify_before: 48h
```

### 2. Communication Plan

#### Stakeholder Notification Template

```markdown
Subject: Gunj Operator Upgrade - [Environment] - [Date]

Dear Team,

We will be upgrading the Gunj Operator in the [Environment] environment.

**Details:**
- Date: [Date]
- Time: [Start Time] - [End Time] [Timezone]
- Version: v1.4.5 → v1.5.0
- Expected Impact: [None/Minimal/Brief interruption]

**What's Changing:**
- Improved performance and stability
- New features: [List key features]
- Security updates

**Action Required:**
- No action required for most users
- [Specific actions if any]

**During the Upgrade:**
- Monitoring dashboards may show brief gaps
- API responses may be slower temporarily

**Rollback Plan:**
We have tested rollback procedures and can revert within 15 minutes if needed.

**Contact:**
- Primary: [Name] - [Email/Phone]
- Secondary: [Name] - [Email/Phone]

Thank you for your patience.
```

### 3. Monitoring During Upgrade

#### Key Metrics to Watch

```promql
# Operator health
up{job="gunj-operator"}

# Reconciliation rate
rate(gunj_operator_reconciliations_total[5m])

# Error rate
rate(gunj_operator_reconciliation_errors_total[5m])

# API latency
histogram_quantile(0.99, gunj_operator_api_request_duration_seconds_bucket)

# Platform status
count by (phase) (gunj_operator_platform_status)

# Resource usage
container_memory_usage_bytes{pod=~"gunj-operator.*"}
rate(container_cpu_usage_seconds_total{pod=~"gunj-operator.*"}[5m])
```

#### Monitoring Dashboard

```yaml
# upgrade-monitoring-dashboard.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: upgrade-monitoring-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  upgrade-dashboard.json: |
    {
      "dashboard": {
        "title": "Gunj Operator Upgrade Monitoring",
        "panels": [
          {
            "title": "Operator Status",
            "targets": [{
              "expr": "up{job='gunj-operator'}"
            }]
          },
          {
            "title": "Platform Health",
            "targets": [{
              "expr": "count by (phase) (gunj_operator_platform_status)"
            }]
          },
          {
            "title": "Conversion Progress",
            "targets": [{
              "expr": "gunj_operator_conversions_total"
            }]
          },
          {
            "title": "Error Rate",
            "targets": [{
              "expr": "rate(gunj_operator_errors_total[5m])"
            }]
          }
        ]
      }
    }
```

### 4. Testing Strategy

#### Test Categories

1. **Smoke Tests** (5 minutes)
   - Operator pod running
   - API responding
   - CRDs accessible

2. **Functional Tests** (30 minutes)
   - Create new platform
   - Update existing platform
   - Delete platform
   - Check conversions

3. **Integration Tests** (1 hour)
   - Prometheus scraping
   - Grafana datasources
   - Alert routing
   - Log collection

4. **Performance Tests** (2 hours)
   - Load testing
   - Stress testing
   - Longevity testing

#### Automated Test Suite

```bash
#!/bin/bash
# upgrade-tests.sh

run_smoke_tests() {
    echo "Running smoke tests..."
    
    # Test 1: Operator health
    kubectl get pod -n gunj-system -l app=gunj-operator &> /dev/null || return 1
    
    # Test 2: API health
    kubectl run test-api --image=curlimages/curl --rm -it --restart=Never -- \
        curl -s http://gunj-operator.gunj-system:8080/healthz | grep -q "ok" || return 1
    
    # Test 3: CRD access
    kubectl api-resources | grep observabilityplatforms &> /dev/null || return 1
    
    echo "✓ Smoke tests passed"
}

run_functional_tests() {
    echo "Running functional tests..."
    
    # Test platform creation
    cat <<EOF | kubectl apply -f - || return 1
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: test-platform
  namespace: default
spec:
  components:
    prometheus:
      enabled: true
      version: v2.48.0
EOF
    
    # Wait for platform
    kubectl wait --for=condition=Ready observabilityplatform/test-platform -n default --timeout=300s || return 1
    
    # Cleanup
    kubectl delete observabilityplatform test-platform -n default
    
    echo "✓ Functional tests passed"
}

# Run all tests
run_smoke_tests && run_functional_tests && echo "All tests passed!" || echo "Tests failed!"
```

## Rollback Procedures

### Emergency Rollback

```bash
#!/bin/bash
# emergency-rollback.sh

PREVIOUS_VERSION="v1.4.5"
BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file>"
    exit 1
fi

echo "EMERGENCY ROLLBACK INITIATED"
echo "This will restore to version $PREVIOUS_VERSION"
read -p "Are you sure? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    exit 0
fi

# Step 1: Stop current operator
kubectl scale deployment -n gunj-system gunj-operator --replicas=0

# Step 2: Restore CRDs
tar -xzf "$BACKUP_FILE" -O "*/crds.yaml" | kubectl apply -f -

# Step 3: Restore operator
kubectl set image deployment/gunj-operator -n gunj-system \
    operator="gunjanjp/gunj-operator:$PREVIOUS_VERSION"

# Step 4: Restore platforms
tar -xzf "$BACKUP_FILE" -O "*/platforms.yaml" | kubectl apply -f -

# Step 5: Scale up operator
kubectl scale deployment -n gunj-system gunj-operator --replicas=1

echo "Rollback completed. Please verify system health."
```

## Troubleshooting

### Common Issues

#### Issue: Conversion Webhook Timeout

```bash
# Fix conversion webhook timeout
kubectl delete validatingwebhookconfiguration gunj-operator-webhook
kubectl delete mutatingwebhookconfiguration gunj-operator-webhook

# Restart webhook
kubectl rollout restart deployment -n gunj-system gunj-operator-webhook
```

#### Issue: Platform Stuck in Converting

```bash
# Force reconversion
kubectl patch observabilityplatform <name> -n <namespace> --type merge -p '
{
  "metadata": {
    "annotations": {
      "observability.io/force-conversion": "true"
    }
  }
}'
```

#### Issue: High Memory Usage

```yaml
# Temporary resource increase
kubectl patch deployment gunj-operator -n gunj-system --type merge -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "operator",
          "resources": {
            "limits": {
              "memory": "1Gi"
            }
          }
        }]
      }
    }
  }
}'
```

## Post-Upgrade Validation

### Validation Checklist

```bash
#!/bin/bash
# post-upgrade-validation.sh

echo "=== Post-Upgrade Validation ==="

# 1. Version verification
echo -n "Operator version: "
kubectl get deployment -n gunj-system gunj-operator -o jsonpath='{.spec.template.spec.containers[0].image}'
echo ""

# 2. Platform status
echo "Platform status:"
kubectl get observabilityplatforms.observability.io --all-namespaces

# 3. Component health
echo "Component health:"
for ns in $(kubectl get ns -o jsonpath='{.items[*].metadata.name}'); do
    for component in prometheus grafana loki tempo; do
        if kubectl get deployment -n $ns $component &> /dev/null; then
            ready=$(kubectl get deployment -n $ns $component -o jsonpath='{.status.readyReplicas}')
            desired=$(kubectl get deployment -n $ns $component -o jsonpath='{.spec.replicas}')
            echo "$ns/$component: $ready/$desired ready"
        fi
    done
done

# 4. API validation
echo "API versions:"
kubectl api-versions | grep observability

# 5. Metrics validation
echo "Key metrics:"
kubectl exec -n monitoring prometheus-0 -- promtool query instant 'up{job=~"prometheus|grafana|loki|tempo"}'

echo "=== Validation Complete ==="
```

## Automation Scripts

### Complete Automation Suite

Create a directory structure for upgrade automation:

```
upgrade-automation/
├── scripts/
│   ├── pre-upgrade.sh
│   ├── upgrade.sh
│   ├── post-upgrade.sh
│   └── rollback.sh
├── configs/
│   ├── upgrade-config.yaml
│   └── validation-rules.yaml
├── templates/
│   ├── notifications.md
│   └── reports.md
└── Makefile
```

```makefile
# Makefile for upgrade automation

.PHONY: all pre-upgrade upgrade post-upgrade rollback

OPERATOR_NAMESPACE ?= gunj-system
TARGET_VERSION ?= v1.5.0
UPGRADE_METHOD ?= rolling

all: pre-upgrade upgrade post-upgrade

pre-upgrade:
	@echo "Running pre-upgrade tasks..."
	./scripts/pre-upgrade.sh

upgrade:
	@echo "Running upgrade with method: $(UPGRADE_METHOD)..."
	./scripts/upgrade.sh $(UPGRADE_METHOD) $(TARGET_VERSION)

post-upgrade:
	@echo "Running post-upgrade validation..."
	./scripts/post-upgrade.sh

rollback:
	@echo "Running rollback..."
	./scripts/rollback.sh

report:
	@echo "Generating upgrade report..."
	./scripts/generate-report.sh > upgrade-report-$(date +%Y%m%d).md
```

---

**Document Version**: 1.0  
**Last Updated**: June 15, 2025  
**Next Review**: July 15, 2025

For support during upgrades, contact gunjanjp@gmail.com or join #gunj-operator-upgrades on Slack.
