# Migration Tools Documentation

This document describes the tools available for migrating ObservabilityPlatform resources between API versions. These tools help automate the migration process and reduce the risk of errors.

## Table of Contents

- [Overview](#overview)
- [Built-in Tools](#built-in-tools)
- [CLI Migration Tool](#cli-migration-tool)
- [Conversion Webhooks](#conversion-webhooks)
- [Migration Scripts](#migration-scripts)
- [Validation Tools](#validation-tools)
- [Backup and Restore](#backup-and-restore)
- [Monitoring Migration Progress](#monitoring-migration-progress)
- [Third-Party Tools](#third-party-tools)

## Overview

The Gunj Operator provides several tools to assist with API version migrations:

1. **Built-in conversion**: Automatic conversion via webhooks
2. **CLI tool**: Command-line migration utility
3. **Scripts**: Bash/Python scripts for bulk operations
4. **Validation**: Pre and post-migration validation
5. **Monitoring**: Track migration progress and health

## Built-in Tools

### Automatic Conversion

The operator includes conversion webhooks that automatically handle API version changes:

```bash
# Check if conversion webhook is running
kubectl get validatingwebhookconfigurations | grep gunj-operator
kubectl get mutatingwebhookconfigurations | grep gunj-operator

# View webhook details
kubectl describe mutatingwebhookconfiguration gunj-operator-webhook
```

Features:
- Seamless conversion between versions
- Validation of converted resources
- Detailed error messages
- Conversion metrics

### Dry-Run Mode

Test conversions without applying changes:

```bash
# Dry-run conversion
kubectl apply -f platform-v1alpha1.yaml --dry-run=server -o yaml

# Dry-run with validation
kubectl apply -f platform-v1alpha1.yaml --dry-run=server --validate=true
```

## CLI Migration Tool

### Installation

```bash
# Download the latest release
curl -L https://github.com/gunjanjp/gunj-operator/releases/latest/download/gunj-migrate -o gunj-migrate
chmod +x gunj-migrate
sudo mv gunj-migrate /usr/local/bin/

# Or install via Go
go install github.com/gunjanjp/gunj-operator/cmd/gunj-migrate@latest
```

### Basic Usage

```bash
# Migrate a single resource
gunj-migrate convert -f platform.yaml -o platform-migrated.yaml

# Migrate with target version
gunj-migrate convert -f platform.yaml --to-version v1beta1

# Batch migration
gunj-migrate convert -f ./platforms/ -o ./migrated/

# Validate without converting
gunj-migrate validate -f platform.yaml --version v1beta1
```

### Advanced Features

```bash
# Interactive migration with prompts
gunj-migrate convert -f platform.yaml --interactive

# Generate migration report
gunj-migrate analyze -f ./platforms/ --report migration-report.html

# Rollback to previous version
gunj-migrate rollback -f platform.yaml --to-version v1alpha1

# Compare versions
gunj-migrate diff -f platform-v1alpha1.yaml -f platform-v1beta1.yaml
```

### Configuration File

Create `.gunj-migrate.yaml`:

```yaml
# Migration tool configuration
migration:
  defaultTargetVersion: v1beta1
  backupBeforeMigrate: true
  validateAfterMigrate: true
  
conversion:
  preserveUnknownFields: false
  strictValidation: true
  generateDefaults: true
  
output:
  format: yaml  # yaml or json
  prettyPrint: true
  includeManaged: false
  
backup:
  enabled: true
  location: ./backups
  retention: 30d
```

## Conversion Webhooks

### Understanding Webhook Operation

The conversion webhook automatically converts resources between API versions:

```go
// Webhook implementation (simplified)
func (w *WebhookServer) Convert(ctx context.Context, req *v1.ConversionRequest) (*v1.ConversionResponse, error) {
    switch req.DesiredAPIVersion {
    case "observability.io/v1beta1":
        return w.convertToV1Beta1(req)
    case "observability.io/v1alpha1":
        return w.convertToV1Alpha1(req)
    default:
        return nil, fmt.Errorf("unknown version: %s", req.DesiredAPIVersion)
    }
}
```

### Webhook Metrics

Monitor conversion webhook performance:

```bash
# Port-forward to metrics endpoint
kubectl port-forward -n gunj-system svc/gunj-operator-metrics 8080:8080

# Check conversion metrics
curl http://localhost:8080/metrics | grep conversion

# Key metrics:
# - gunj_operator_conversion_total
# - gunj_operator_conversion_duration_seconds
# - gunj_operator_conversion_errors_total
```

### Troubleshooting Webhooks

```bash
# Check webhook logs
kubectl logs -n gunj-system deployment/gunj-operator-webhook -f

# Test webhook connectivity
kubectl run test-webhook --image=curlimages/curl --rm -it -- \
  curl -k https://gunj-operator-webhook.gunj-system.svc:443/convert

# Bypass webhook if needed (emergency only)
kubectl label namespace monitoring conversion.observability.io/skip=true
```

## Migration Scripts

### Bash Migration Script

```bash
#!/bin/bash
# migrate-platforms.sh - Comprehensive migration script

set -euo pipefail

# Configuration
SOURCE_VERSION="v1alpha1"
TARGET_VERSION="v1beta1"
BACKUP_DIR="./backups/$(date +%Y%m%d-%H%M%S)"
LOG_FILE="./migration.log"
DRY_RUN=${DRY_RUN:-false}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging function
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Error handling
error_exit() {
    echo -e "${RED}ERROR: $1${NC}" >&2
    exit 1
}

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Function to migrate a single platform
migrate_platform() {
    local file=$1
    local name=$(yq e '.metadata.name' "$file")
    local namespace=$(yq e '.metadata.namespace // "default"' "$file")
    
    log "Processing platform: $namespace/$name"
    
    # Backup original
    cp "$file" "$BACKUP_DIR/${namespace}-${name}-original.yaml"
    
    # Convert API version
    local converted_file="$BACKUP_DIR/${namespace}-${name}-converted.yaml"
    
    # Perform conversion
    yq eval "
        .apiVersion = \"observability.io/$TARGET_VERSION\" |
        .metadata.annotations[\"observability.io/migrated-from\"] = \"$SOURCE_VERSION\" |
        .metadata.annotations[\"observability.io/migrated-at\"] = \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
    " "$file" > "$converted_file"
    
    # Apply field transformations
    transform_fields "$converted_file"
    
    # Validate converted file
    if kubectl apply -f "$converted_file" --dry-run=server > /dev/null 2>&1; then
        log "✓ Validation passed for $namespace/$name"
        
        if [[ "$DRY_RUN" == "false" ]]; then
            # Apply the migration
            kubectl apply -f "$converted_file"
            log "✓ Applied migration for $namespace/$name"
        else
            log "✓ Dry-run successful for $namespace/$name"
        fi
    else
        log "✗ Validation failed for $namespace/$name"
        return 1
    fi
}

# Function to transform fields
transform_fields() {
    local file=$1
    
    # Transform Prometheus customConfig
    if yq e '.spec.components.prometheus.customConfig' "$file" > /dev/null 2>&1; then
        local custom_config=$(yq e '.spec.components.prometheus.customConfig' "$file")
        
        # Extract and transform externalLabels
        if echo "$custom_config" | grep -q "externalLabels"; then
            local labels=$(echo "$custom_config" | grep "externalLabels" | cut -d: -f2- | tr -d "'\"")
            yq e -i ".spec.components.prometheus.externalLabels = $labels" "$file"
        fi
        
        # Remove customConfig after transformation
        yq e -i 'del(.spec.components.prometheus.customConfig)' "$file"
    fi
    
    # Transform resource specifications
    for component in prometheus grafana loki tempo; do
        if yq e ".spec.components.$component.cpuRequest" "$file" > /dev/null 2>&1; then
            local cpu_req=$(yq e ".spec.components.$component.cpuRequest" "$file")
            local mem_req=$(yq e ".spec.components.$component.memoryRequest" "$file")
            local cpu_limit=$(yq e ".spec.components.$component.cpuLimit // \"\"" "$file")
            local mem_limit=$(yq e ".spec.components.$component.memoryLimit // \"\"" "$file")
            
            # Create resources structure
            yq e -i "
                .spec.components.$component.resources.requests.cpu = \"$cpu_req\" |
                .spec.components.$component.resources.requests.memory = \"$mem_req\"
            " "$file"
            
            if [[ -n "$cpu_limit" ]]; then
                yq e -i ".spec.components.$component.resources.limits.cpu = \"$cpu_limit\"" "$file"
            fi
            
            if [[ -n "$mem_limit" ]]; then
                yq e -i ".spec.components.$component.resources.limits.memory = \"$mem_limit\"" "$file"
            fi
            
            # Remove old fields
            yq e -i "
                del(.spec.components.$component.cpuRequest) |
                del(.spec.components.$component.memoryRequest) |
                del(.spec.components.$component.cpuLimit) |
                del(.spec.components.$component.memoryLimit)
            " "$file"
        fi
    done
    
    # Transform storage configuration
    if yq e '.spec.components.prometheus.storageSize' "$file" > /dev/null 2>&1; then
        local size=$(yq e '.spec.components.prometheus.storageSize' "$file")
        local class=$(yq e '.spec.components.prometheus.storageClass // ""' "$file")
        
        yq e -i ".spec.components.prometheus.storage.size = \"$size\"" "$file"
        
        if [[ -n "$class" ]]; then
            yq e -i ".spec.components.prometheus.storage.storageClassName = \"$class\"" "$file"
        fi
        
        yq e -i '
            del(.spec.components.prometheus.storageSize) |
            del(.spec.components.prometheus.storageClass)
        ' "$file"
    fi
}

# Main migration process
main() {
    log "Starting migration from $SOURCE_VERSION to $TARGET_VERSION"
    log "Dry-run mode: $DRY_RUN"
    log "Backup directory: $BACKUP_DIR"
    
    # Find all platform files
    local platforms=()
    
    # Get platforms from cluster
    while IFS= read -r line; do
        platforms+=("$line")
    done < <(kubectl get observabilityplatforms -A -o json | jq -r '.items[] | "\(.metadata.namespace)/\(.metadata.name)"')
    
    if [[ ${#platforms[@]} -eq 0 ]]; then
        log "No platforms found to migrate"
        exit 0
    fi
    
    log "Found ${#platforms[@]} platforms to migrate"
    
    # Export platforms to files
    local success=0
    local failed=0
    
    for platform in "${platforms[@]}"; do
        namespace=$(echo "$platform" | cut -d'/' -f1)
        name=$(echo "$platform" | cut -d'/' -f2)
        
        # Export current state
        local export_file="$BACKUP_DIR/${namespace}-${name}-export.yaml"
        kubectl get observabilityplatform "$name" -n "$namespace" -o yaml > "$export_file"
        
        # Attempt migration
        if migrate_platform "$export_file"; then
            ((success++))
        else
            ((failed++))
            echo -e "${YELLOW}Warning: Failed to migrate $platform${NC}"
        fi
    done
    
    # Summary
    log "Migration complete!"
    log "Successful: $success"
    log "Failed: $failed"
    log "Backups saved to: $BACKUP_DIR"
    
    if [[ $failed -gt 0 ]]; then
        echo -e "${YELLOW}Some migrations failed. Check $LOG_FILE for details.${NC}"
        exit 1
    fi
}

# Run main function
main "$@"
```

### Python Migration Tool

```python
#!/usr/bin/env python3
"""
gunj_migrate.py - Advanced migration tool for ObservabilityPlatform resources
"""

import argparse
import json
import logging
import os
import sys
import yaml
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import kubernetes
from kubernetes import client, config
from kubernetes.client.rest import ApiException


class PlatformMigrator:
    """Handles migration of ObservabilityPlatform resources between API versions."""
    
    def __init__(self, source_version: str, target_version: str, dry_run: bool = False):
        self.source_version = source_version
        self.target_version = target_version
        self.dry_run = dry_run
        self.logger = self._setup_logging()
        
        # Load Kubernetes config
        try:
            config.load_incluster_config()
        except:
            config.load_kube_config()
        
        self.api_client = client.ApiClient()
        self.custom_api = client.CustomObjectsApi()
        
    def _setup_logging(self) -> logging.Logger:
        """Configure logging."""
        logger = logging.getLogger(__name__)
        logger.setLevel(logging.INFO)
        
        handler = logging.StreamHandler(sys.stdout)
        formatter = logging.Formatter(
            '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        handler.setFormatter(formatter)
        logger.addHandler(handler)
        
        return logger
        
    def migrate_resource(self, resource: Dict) -> Tuple[bool, Optional[Dict]]:
        """Migrate a single resource to the target version."""
        name = resource['metadata']['name']
        namespace = resource['metadata'].get('namespace', 'default')
        
        self.logger.info(f"Migrating {namespace}/{name}")
        
        try:
            # Create a copy for migration
            migrated = resource.copy()
            
            # Update API version
            migrated['apiVersion'] = f"observability.io/{self.target_version}"
            
            # Add migration annotations
            if 'annotations' not in migrated['metadata']:
                migrated['metadata']['annotations'] = {}
                
            migrated['metadata']['annotations'].update({
                'observability.io/migrated-from': self.source_version,
                'observability.io/migrated-at': datetime.utcnow().isoformat() + 'Z',
                'observability.io/migration-tool': 'gunj-migrate'
            })
            
            # Transform fields based on version
            if self.source_version == 'v1alpha1' and self.target_version == 'v1beta1':
                migrated = self._transform_v1alpha1_to_v1beta1(migrated)
            elif self.source_version == 'v1beta1' and self.target_version == 'v1alpha1':
                migrated = self._transform_v1beta1_to_v1alpha1(migrated)
            else:
                raise ValueError(f"Unsupported migration path: {self.source_version} -> {self.target_version}")
            
            # Validate the migrated resource
            if self._validate_resource(migrated):
                return True, migrated
            else:
                return False, None
                
        except Exception as e:
            self.logger.error(f"Failed to migrate {namespace}/{name}: {str(e)}")
            return False, None
            
    def _transform_v1alpha1_to_v1beta1(self, resource: Dict) -> Dict:
        """Transform fields from v1alpha1 to v1beta1."""
        spec = resource.get('spec', {})
        
        # Transform global configuration
        if 'globalConfig' in spec:
            spec['global'] = spec.pop('globalConfig')
            
        if 'paused' in spec:
            if 'global' not in spec:
                spec['global'] = {}
            spec['global']['paused'] = spec.pop('paused')
            
        # Transform each component
        components = spec.get('components', {})
        
        for component_name, component_spec in components.items():
            # Transform resource specifications
            if 'cpuRequest' in component_spec:
                if 'resources' not in component_spec:
                    component_spec['resources'] = {'requests': {}, 'limits': {}}
                    
                component_spec['resources']['requests']['cpu'] = component_spec.pop('cpuRequest')
                component_spec['resources']['requests']['memory'] = component_spec.pop('memoryRequest', '1Gi')
                
                if 'cpuLimit' in component_spec:
                    component_spec['resources']['limits']['cpu'] = component_spec.pop('cpuLimit')
                if 'memoryLimit' in component_spec:
                    component_spec['resources']['limits']['memory'] = component_spec.pop('memoryLimit')
                    
            # Transform storage configuration
            if 'storageSize' in component_spec:
                if 'storage' not in component_spec:
                    component_spec['storage'] = {}
                    
                component_spec['storage']['size'] = component_spec.pop('storageSize')
                
                if 'storageClass' in component_spec:
                    component_spec['storage']['storageClassName'] = component_spec.pop('storageClass')
                    
            # Transform Prometheus-specific fields
            if component_name == 'prometheus' and 'customConfig' in component_spec:
                custom_config = component_spec.pop('customConfig')
                
                # Parse custom config (assumed to be string with embedded YAML/JSON)
                if isinstance(custom_config, str):
                    try:
                        # Try to parse as YAML first
                        parsed_config = yaml.safe_load(custom_config)
                    except:
                        # Try JSON if YAML fails
                        try:
                            parsed_config = json.loads(custom_config)
                        except:
                            self.logger.warning(f"Could not parse customConfig for {component_name}")
                            parsed_config = {}
                else:
                    parsed_config = custom_config
                    
                # Extract known fields
                if 'externalLabels' in parsed_config:
                    labels = parsed_config['externalLabels']
                    if isinstance(labels, str):
                        component_spec['externalLabels'] = json.loads(labels)
                    else:
                        component_spec['externalLabels'] = labels
                        
                if 'remoteWrite' in parsed_config:
                    component_spec['remoteWrite'] = parsed_config['remoteWrite']
                    
                if 'scrapeConfigs' in parsed_config:
                    component_spec['additionalScrapeConfigs'] = parsed_config['scrapeConfigs']
                    
            # Transform Grafana-specific fields
            if component_name == 'grafana':
                if 'ingressEnabled' in component_spec:
                    if 'ingress' not in component_spec:
                        component_spec['ingress'] = {}
                        
                    component_spec['ingress']['enabled'] = component_spec.pop('ingressEnabled')
                    
                    if 'ingressHost' in component_spec:
                        component_spec['ingress']['host'] = component_spec.pop('ingressHost')
                        
                    if 'ingressTLS' in component_spec:
                        component_spec['ingress']['tls'] = {
                            'enabled': component_spec.pop('ingressTLS')
                        }
                        
                if 'adminPassword' in component_spec:
                    if 'security' not in component_spec:
                        component_spec['security'] = {}
                    component_spec['security']['adminPassword'] = component_spec.pop('adminPassword')
                    
        # Add new v1beta1 fields with defaults
        if 'security' not in spec:
            spec['security'] = {
                'tls': {'enabled': True},
                'podSecurityPolicy': True,
                'networkPolicy': True,
                'rbac': {'create': True}
            }
            
        if 'monitoring' not in spec:
            spec['monitoring'] = {
                'selfMonitoring': True,
                'serviceMonitor': {'enabled': True}
            }
            
        resource['spec'] = spec
        return resource
        
    def _transform_v1beta1_to_v1alpha1(self, resource: Dict) -> Dict:
        """Transform fields from v1beta1 to v1alpha1 (downgrade)."""
        # This is generally not recommended and will lose data
        self.logger.warning("Downgrading from v1beta1 to v1alpha1 will result in data loss!")
        
        spec = resource.get('spec', {})
        
        # Reverse transformations (simplified)
        if 'global' in spec:
            spec['globalConfig'] = spec.pop('global')
            
        # Remove v1beta1-only fields
        for field in ['security', 'monitoring', 'costOptimization', 'serviceMesh', 'multiCluster']:
            if field in spec:
                self.logger.warning(f"Removing v1beta1-only field: {field}")
                spec.pop(field)
                
        resource['spec'] = spec
        return resource
        
    def _validate_resource(self, resource: Dict) -> bool:
        """Validate the migrated resource."""
        # TODO: Implement actual validation logic
        # This would typically involve:
        # 1. Schema validation
        # 2. Semantic validation
        # 3. Dependency checks
        
        required_fields = ['apiVersion', 'kind', 'metadata', 'spec']
        for field in required_fields:
            if field not in resource:
                self.logger.error(f"Missing required field: {field}")
                return False
                
        return True
        
    def migrate_from_cluster(self, namespace: Optional[str] = None) -> Dict[str, int]:
        """Migrate all ObservabilityPlatform resources in the cluster."""
        results = {'success': 0, 'failed': 0, 'skipped': 0}
        
        try:
            # List all ObservabilityPlatform resources
            if namespace:
                platforms = self.custom_api.list_namespaced_custom_object(
                    group="observability.io",
                    version=self.source_version,
                    namespace=namespace,
                    plural="observabilityplatforms"
                )
            else:
                platforms = self.custom_api.list_cluster_custom_object(
                    group="observability.io",
                    version=self.source_version,
                    plural="observabilityplatforms"
                )
                
            items = platforms.get('items', [])
            self.logger.info(f"Found {len(items)} platforms to migrate")
            
            for platform in items:
                name = platform['metadata']['name']
                ns = platform['metadata'].get('namespace', 'default')
                
                # Check if already migrated
                if platform.get('apiVersion', '').endswith(self.target_version):
                    self.logger.info(f"Skipping {ns}/{name} - already at target version")
                    results['skipped'] += 1
                    continue
                    
                # Migrate the resource
                success, migrated = self.migrate_resource(platform)
                
                if success and migrated:
                    if not self.dry_run:
                        try:
                            # Delete old version
                            self.custom_api.delete_namespaced_custom_object(
                                group="observability.io",
                                version=self.source_version,
                                namespace=ns,
                                plural="observabilityplatforms",
                                name=name
                            )
                            
                            # Create new version
                            self.custom_api.create_namespaced_custom_object(
                                group="observability.io",
                                version=self.target_version,
                                namespace=ns,
                                plural="observabilityplatforms",
                                body=migrated
                            )
                            
                            self.logger.info(f"Successfully migrated {ns}/{name}")
                            results['success'] += 1
                            
                        except ApiException as e:
                            self.logger.error(f"Failed to apply migration for {ns}/{name}: {e}")
                            results['failed'] += 1
                    else:
                        self.logger.info(f"Dry-run: Would migrate {ns}/{name}")
                        results['success'] += 1
                else:
                    results['failed'] += 1
                    
        except ApiException as e:
            self.logger.error(f"Failed to list platforms: {e}")
            
        return results
        
    def migrate_from_files(self, input_path: Path, output_path: Path) -> Dict[str, int]:
        """Migrate resources from files."""
        results = {'success': 0, 'failed': 0, 'skipped': 0}
        
        # Create output directory if needed
        output_path.mkdir(parents=True, exist_ok=True)
        
        # Process all YAML files
        yaml_files = list(input_path.glob('*.yaml')) + list(input_path.glob('*.yml'))
        
        for file_path in yaml_files:
            self.logger.info(f"Processing {file_path}")
            
            try:
                with open(file_path, 'r') as f:
                    # Handle multi-document YAML
                    documents = list(yaml.safe_load_all(f))
                    
                migrated_docs = []
                
                for doc in documents:
                    if not doc:
                        continue
                        
                    # Check if it's an ObservabilityPlatform
                    if doc.get('kind') != 'ObservabilityPlatform':
                        migrated_docs.append(doc)
                        continue
                        
                    # Check if already at target version
                    if doc.get('apiVersion', '').endswith(self.target_version):
                        self.logger.info(f"Skipping {file_path} - already at target version")
                        results['skipped'] += 1
                        migrated_docs.append(doc)
                        continue
                        
                    # Migrate the resource
                    success, migrated = self.migrate_resource(doc)
                    
                    if success and migrated:
                        migrated_docs.append(migrated)
                        results['success'] += 1
                    else:
                        # Keep original if migration fails
                        migrated_docs.append(doc)
                        results['failed'] += 1
                        
                # Write migrated documents
                output_file = output_path / file_path.name
                with open(output_file, 'w') as f:
                    yaml.dump_all(migrated_docs, f, default_flow_style=False)
                    
                self.logger.info(f"Wrote migrated resources to {output_file}")
                
            except Exception as e:
                self.logger.error(f"Failed to process {file_path}: {e}")
                results['failed'] += 1
                
        return results


def main():
    """Main entry point for the migration tool."""
    parser = argparse.ArgumentParser(
        description='Migrate ObservabilityPlatform resources between API versions'
    )
    
    parser.add_argument(
        '--source-version',
        default='v1alpha1',
        help='Source API version (default: v1alpha1)'
    )
    
    parser.add_argument(
        '--target-version',
        default='v1beta1',
        help='Target API version (default: v1beta1)'
    )
    
    parser.add_argument(
        '--mode',
        choices=['cluster', 'files'],
        default='cluster',
        help='Migration mode: cluster or files (default: cluster)'
    )
    
    parser.add_argument(
        '--namespace',
        help='Namespace to migrate (cluster mode only, default: all namespaces)'
    )
    
    parser.add_argument(
        '--input-path',
        type=Path,
        help='Input directory for file mode'
    )
    
    parser.add_argument(
        '--output-path',
        type=Path,
        help='Output directory for file mode'
    )
    
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Perform dry-run without making changes'
    )
    
    parser.add_argument(
        '--verbose',
        action='store_true',
        help='Enable verbose logging'
    )
    
    args = parser.parse_args()
    
    # Validate arguments
    if args.mode == 'files':
        if not args.input_path or not args.output_path:
            parser.error("File mode requires --input-path and --output-path")
            
    # Create migrator
    migrator = PlatformMigrator(
        source_version=args.source_version,
        target_version=args.target_version,
        dry_run=args.dry_run
    )
    
    if args.verbose:
        migrator.logger.setLevel(logging.DEBUG)
        
    # Perform migration
    if args.mode == 'cluster':
        results = migrator.migrate_from_cluster(namespace=args.namespace)
    else:
        results = migrator.migrate_from_files(
            input_path=args.input_path,
            output_path=args.output_path
        )
        
    # Print summary
    print("\nMigration Summary:")
    print(f"  Successful: {results['success']}")
    print(f"  Failed: {results['failed']}")
    print(f"  Skipped: {results['skipped']}")
    
    # Exit with appropriate code
    if results['failed'] > 0:
        sys.exit(1)
    else:
        sys.exit(0)


if __name__ == '__main__':
    main()
```

## Validation Tools

### Pre-Migration Validation

```bash
#!/bin/bash
# validate-before-migration.sh

# Function to validate platform readiness
validate_platform() {
    local name=$1
    local namespace=$2
    
    echo "Validating $namespace/$name..."
    
    # Check if platform exists
    if ! kubectl get observabilityplatform "$name" -n "$namespace" > /dev/null 2>&1; then
        echo "✗ Platform not found"
        return 1
    fi
    
    # Check platform status
    local phase=$(kubectl get observabilityplatform "$name" -n "$namespace" -o jsonpath='{.status.phase}')
    if [[ "$phase" != "Ready" ]]; then
        echo "✗ Platform not ready (phase: $phase)"
        return 1
    fi
    
    # Check component health
    local components=$(kubectl get pods -n "$namespace" -l "observability.io/platform=$name" -o json)
    local not_ready=$(echo "$components" | jq '[.items[] | select(.status.phase != "Running")] | length')
    
    if [[ "$not_ready" -gt 0 ]]; then
        echo "✗ Some components not ready ($not_ready pods)"
        return 1
    fi
    
    # Check for pending operations
    local reconciling=$(kubectl get observabilityplatform "$name" -n "$namespace" \
        -o jsonpath='{.metadata.annotations.observability\.io/reconciling}')
    
    if [[ "$reconciling" == "true" ]]; then
        echo "✗ Platform is currently reconciling"
        return 1
    fi
    
    echo "✓ Platform is ready for migration"
    return 0
}

# Main validation
kubectl get observabilityplatforms -A -o json | \
    jq -r '.items[] | "\(.metadata.namespace) \(.metadata.name)"' | \
    while read -r namespace name; do
        validate_platform "$name" "$namespace"
    done
```

### Post-Migration Validation

```python
#!/usr/bin/env python3
"""
validate_migration.py - Post-migration validation tool
"""

import sys
import time
import kubernetes
from kubernetes import client, config

def validate_migrated_platform(name: str, namespace: str) -> bool:
    """Validate a migrated platform."""
    config.load_kube_config()
    custom_api = client.CustomObjectsApi()
    core_api = client.CoreV1Api()
    
    try:
        # Get the platform
        platform = custom_api.get_namespaced_custom_object(
            group="observability.io",
            version="v1beta1",
            namespace=namespace,
            plural="observabilityplatforms",
            name=name
        )
        
        # Check migration annotations
        annotations = platform['metadata'].get('annotations', {})
        if 'observability.io/migrated-from' not in annotations:
            print(f"⚠ Missing migration annotation for {namespace}/{name}")
            
        # Check status
        status = platform.get('status', {})
        phase = status.get('phase')
        
        if phase != 'Ready':
            print(f"✗ Platform {namespace}/{name} not ready: {phase}")
            return False
            
        # Check components
        components = platform['spec'].get('components', {})
        for comp_name, comp_spec in components.items():
            if comp_spec.get('enabled', False):
                # Check if component pods are running
                pods = core_api.list_namespaced_pod(
                    namespace=namespace,
                    label_selector=f"app.kubernetes.io/component={comp_name},observability.io/platform={name}"
                )
                
                if not pods.items:
                    print(f"✗ No pods found for component {comp_name}")
                    return False
                    
                for pod in pods.items:
                    if pod.status.phase != 'Running':
                        print(f"✗ Pod {pod.metadata.name} not running: {pod.status.phase}")
                        return False
                        
        print(f"✓ Platform {namespace}/{name} validated successfully")
        return True
        
    except Exception as e:
        print(f"✗ Failed to validate {namespace}/{name}: {e}")
        return False


if __name__ == '__main__':
    # Get all migrated platforms
    config.load_kube_config()
    custom_api = client.CustomObjectsApi()
    
    platforms = custom_api.list_cluster_custom_object(
        group="observability.io",
        version="v1beta1",
        plural="observabilityplatforms"
    )
    
    failed_validations = []
    
    for platform in platforms.get('items', []):
        name = platform['metadata']['name']
        namespace = platform['metadata'].get('namespace', 'default')
        
        if not validate_migrated_platform(name, namespace):
            failed_validations.append(f"{namespace}/{name}")
            
    if failed_validations:
        print(f"\n❌ Validation failed for {len(failed_validations)} platforms:")
        for platform in failed_validations:
            print(f"  - {platform}")
        sys.exit(1)
    else:
        print(f"\n✅ All platforms validated successfully!")
        sys.exit(0)
```

## Backup and Restore

### Backup Before Migration

```bash
#!/bin/bash
# backup-platforms.sh

BACKUP_DIR="./backups/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup all platforms
echo "Creating backup in $BACKUP_DIR..."

# Export platforms
kubectl get observabilityplatforms -A -o yaml > "$BACKUP_DIR/all-platforms.yaml"

# Export individual platforms
kubectl get observabilityplatforms -A -o json | \
    jq -r '.items[] | "\(.metadata.namespace) \(.metadata.name)"' | \
    while read -r namespace name; do
        kubectl get observabilityplatform "$name" -n "$namespace" -o yaml > \
            "$BACKUP_DIR/${namespace}-${name}.yaml"
        
        # Also backup related resources
        kubectl get configmaps,secrets,services,deployments,statefulsets \
            -n "$namespace" \
            -l "observability.io/platform=$name" \
            -o yaml > "$BACKUP_DIR/${namespace}-${name}-resources.yaml"
    done

# Create restore script
cat > "$BACKUP_DIR/restore.sh" <<'EOF'
#!/bin/bash
# Restore script for platform backups

echo "Restoring platforms from backup..."

# Apply all platforms
kubectl apply -f all-platforms.yaml

# Wait for platforms to be created
sleep 10

# Apply related resources
for file in *-resources.yaml; do
    echo "Restoring resources from $file"
    kubectl apply -f "$file"
done

echo "Restore complete!"
EOF

chmod +x "$BACKUP_DIR/restore.sh"

# Create tarball
tar -czf "$BACKUP_DIR.tar.gz" -C "$(dirname "$BACKUP_DIR")" "$(basename "$BACKUP_DIR")"

echo "Backup complete: $BACKUP_DIR.tar.gz"
```

### Automated Rollback

```python
#!/usr/bin/env python3
"""
rollback_migration.py - Automated rollback for failed migrations
"""

import argparse
import subprocess
import sys
import yaml
from pathlib import Path

class MigrationRollback:
    def __init__(self, backup_dir: str):
        self.backup_dir = Path(backup_dir)
        
    def rollback_platform(self, name: str, namespace: str) -> bool:
        """Rollback a single platform."""
        backup_file = self.backup_dir / f"{namespace}-{name}.yaml"
        
        if not backup_file.exists():
            print(f"✗ Backup not found for {namespace}/{name}")
            return False
            
        try:
            # Delete current version
            subprocess.run([
                'kubectl', 'delete', 'observabilityplatform',
                name, '-n', namespace, '--ignore-not-found'
            ], check=True)
            
            # Restore from backup
            subprocess.run([
                'kubectl', 'apply', '-f', str(backup_file)
            ], check=True)
            
            print(f"✓ Rolled back {namespace}/{name}")
            return True
            
        except subprocess.CalledProcessError as e:
            print(f"✗ Failed to rollback {namespace}/{name}: {e}")
            return False
            
    def rollback_all(self) -> bool:
        """Rollback all platforms from backup."""
        all_platforms_file = self.backup_dir / "all-platforms.yaml"
        
        if not all_platforms_file.exists():
            print("✗ Backup file not found")
            return False
            
        try:
            # Delete all current platforms
            subprocess.run([
                'kubectl', 'delete', 'observabilityplatforms',
                '--all', '-A'
            ], check=True)
            
            # Restore from backup
            subprocess.run([
                'kubectl', 'apply', '-f', str(all_platforms_file)
            ], check=True)
            
            print("✓ Rolled back all platforms")
            return True
            
        except subprocess.CalledProcessError as e:
            print(f"✗ Failed to rollback: {e}")
            return False


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Rollback platform migrations')
    parser.add_argument('--backup-dir', required=True, help='Backup directory')
    parser.add_argument('--platform', help='Specific platform to rollback')
    parser.add_argument('--namespace', help='Platform namespace')
    parser.add_argument('--all', action='store_true', help='Rollback all platforms')
    
    args = parser.parse_args()
    
    rollback = MigrationRollback(args.backup_dir)
    
    if args.all:
        success = rollback.rollback_all()
    elif args.platform and args.namespace:
        success = rollback.rollback_platform(args.platform, args.namespace)
    else:
        parser.error("Specify either --all or both --platform and --namespace")
        
    sys.exit(0 if success else 1)
```

## Monitoring Migration Progress

### Migration Dashboard

```yaml
# migration-dashboard.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: migration-dashboard
  namespace: gunj-system
  labels:
    grafana_dashboard: "1"
data:
  migration-dashboard.json: |
    {
      "dashboard": {
        "title": "ObservabilityPlatform Migration Dashboard",
        "panels": [
          {
            "title": "Migration Progress",
            "targets": [{
              "expr": "sum(gunj_operator_platform_info{version=\"v1beta1\"}) / sum(gunj_operator_platform_info)"
            }]
          },
          {
            "title": "Conversion Rate",
            "targets": [{
              "expr": "rate(gunj_operator_conversion_total[5m])"
            }]
          },
          {
            "title": "Conversion Errors",
            "targets": [{
              "expr": "rate(gunj_operator_conversion_errors_total[5m])"
            }]
          },
          {
            "title": "Platform Status by Version",
            "targets": [{
              "expr": "gunj_operator_platform_info"
            }]
          }
        ]
      }
    }
```

### Migration Metrics

```bash
# Check migration metrics
curl -s http://localhost:8080/metrics | grep -E "(migration|conversion)" | sort

# Key metrics to monitor:
# - gunj_operator_conversion_total
# - gunj_operator_conversion_duration_seconds
# - gunj_operator_conversion_errors_total
# - gunj_operator_platform_info{version="..."}
# - gunj_operator_migration_status
```

## Third-Party Tools

### Integration with CI/CD

```yaml
# .gitlab-ci.yml example
migrate-platforms:
  stage: migrate
  image: gunjanjp/gunj-migrate:latest
  script:
    - gunj-migrate validate -f ./platforms/
    - gunj-migrate convert -f ./platforms/ -o ./migrated/
    - gunj-migrate analyze -f ./migrated/ --report report.html
  artifacts:
    paths:
      - migrated/
      - report.html
    expire_in: 1 week
```

### ArgoCD Integration

```yaml
# argocd-migration-hook.yaml
apiVersion: batch/v1
kind: Job
metadata:
  generateName: platform-migration-
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: gunjanjp/gunj-migrate:latest
        command:
          - gunj-migrate
          - convert
          - --mode=cluster
          - --namespace=$(NAMESPACE)
          - --target-version=v1beta1
      restartPolicy: Never
```

### Terraform Provider

```hcl
# Terraform migration example
resource "gunj_platform_migration" "v1beta1" {
  source_version = "v1alpha1"
  target_version = "v1beta1"
  
  platforms = [
    "production",
    "staging",
    "development"
  ]
  
  dry_run = false
  
  backup {
    enabled = true
    location = "s3://backups/migrations/${timestamp()}"
  }
  
  validation {
    pre_migration = true
    post_migration = true
    timeout = "10m"
  }
}
```

## Best Practices

1. **Always backup before migrating**
   - Use the backup scripts provided
   - Test restore procedures
   - Keep backups for rollback

2. **Test in non-production first**
   - Use dry-run mode
   - Validate thoroughly
   - Monitor after migration

3. **Migrate incrementally**
   - Start with one platform
   - Monitor for issues
   - Proceed with others

4. **Monitor during migration**
   - Watch conversion metrics
   - Check error logs
   - Validate component health

5. **Document custom configurations**
   - Note any manual changes needed
   - Update automation scripts
   - Share knowledge with team

## Troubleshooting Common Issues

### Webhook Certificate Issues

```bash
# Regenerate webhook certificates
kubectl delete secret gunj-operator-webhook-cert -n gunj-system
kubectl rollout restart deployment/gunj-operator-webhook -n gunj-system
```

### Stuck Migrations

```bash
# Force migration retry
kubectl annotate observabilityplatform <name> \
  observability.io/force-migration=true --overwrite
```

### Validation Failures

```bash
# Get detailed validation errors
kubectl logs -n gunj-system deployment/gunj-operator-webhook | \
  grep -A 10 "validation failed"
```

## Getting Help

- Documentation: https://gunjanjp.github.io/gunj-operator/migration
- GitHub Issues: https://github.com/gunjanjp/gunj-operator/issues
- Community Slack: #gunj-operator-migration
- Email: gunjanjp@gmail.com
