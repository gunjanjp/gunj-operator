# Migration Helpers Documentation

## Overview

The Gunj Operator provides comprehensive migration helpers to facilitate smooth transitions between API versions. These helpers ensure data integrity, provide rollback capabilities, and offer detailed insights into the migration process.

## Table of Contents

1. [Architecture](#architecture)
2. [Core Components](#core-components)
3. [Migration Process](#migration-process)
4. [CLI Usage](#cli-usage)
5. [API Integration](#api-integration)
6. [Best Practices](#best-practices)
7. [Troubleshooting](#troubleshooting)

## Architecture

The migration helpers are designed with the following principles:

- **Safety First**: All migrations create snapshots for rollback
- **Transparency**: Detailed progress tracking and reporting
- **Performance**: Optimized batch processing and caching
- **Flexibility**: Support for dry-run and manual interventions

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Migration Manager                         │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │   Schema    │  │ Conversion   │  │    Lifecycle     │   │
│  │  Evolution  │  │  Optimizer   │  │   Integration    │   │
│  │   Tracker   │  │              │  │    Manager       │   │
│  └─────────────┘  └──────────────┘  └──────────────────┘   │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │    Batch    │  │   Status     │  │    Rollback      │   │
│  │  Processor  │  │  Reporter    │  │    Manager       │   │
│  └─────────────┘  └──────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Migration Manager

The central component that orchestrates the migration process.

```go
// Create migration manager
config := migration.MigrationConfig{
    MaxConcurrentMigrations: 5,
    BatchSize:               10,
    RetryAttempts:           3,
    RetryInterval:           5 * time.Second,
    EnableOptimizations:     true,
    DryRun:                  false,
    ProgressReportInterval:  5 * time.Second,
}

migrationManager := migration.NewMigrationManager(client, scheme, logger, config)
```

#### Key Features:
- Single resource and batch migrations
- Concurrent processing with configurable limits
- Automatic retry with exponential backoff
- Progress tracking and estimation

### 2. Schema Evolution Tracker

Tracks schema changes and migration paths between versions.

```go
tracker := migration.NewSchemaEvolutionTracker(logger)

// Get migration path
path, err := tracker.GetMigrationPath("v1alpha1", "v1beta1")

// Check for data loss risk
if path.DataLossRisk {
    log.Warn("Migration may result in data loss")
}
```

#### Features:
- Version compatibility matrix
- Field change tracking
- Migration complexity assessment
- Analytics and reporting

### 3. Conversion Optimizer

Optimizes conversion performance through various strategies.

```go
optimizer := migration.NewConversionOptimizer(logger)

// Optimize conversion
optimized, err := optimizer.OptimizeConversion(source, targetVersion)
```

#### Optimization Strategies:
- **Field Skipping**: Removes unnecessary fields
- **Lazy Loading**: Defers loading of large fields
- **Batching**: Groups similar conversions
- **Compression**: Compresses large configuration data
- **Parallelization**: Processes independent fields concurrently

### 4. Lifecycle Integration Manager

Manages integration with the operator lifecycle.

```go
lifecycleMgr := migration.NewLifecycleIntegrationManager(client, logger)

// Pre-migration checks
err := lifecycleMgr.PreMigrationCheck(ctx, resource, targetVersion)

// Apply migration hooks
err := lifecycleMgr.ApplyMigrationHooks(ctx, convertedResource)

// Post-migration validation
err := lifecycleMgr.PostMigrationValidation(ctx, resource, targetVersion)
```

#### Lifecycle Phases:
1. **Pre-Migration**: Health checks, dependency validation
2. **Migration**: Snapshot creation, data preservation
3. **Post-Migration**: Integrity verification, functionality tests
4. **Rollback**: Automatic rollback on failure

### 5. Batch Conversion Processor

Handles efficient batch processing of multiple resources.

```go
processor := migration.NewBatchConversionProcessor(client, scheme, logger, batchSize)

// Process batch
results, err := processor.ProcessBatch(ctx, resources, targetVersion)
```

#### Features:
- Worker pool for concurrent processing
- Rate-limited queue with retry
- Progress tracking per resource
- Detailed result reporting

### 6. Migration Status Reporter

Provides comprehensive migration reporting and analytics.

```go
reporter := migration.NewMigrationStatusReporter(logger)

// Get migration report
report, err := reporter.GetReport(taskID)

// Generate HTML report
htmlReport, err := reporter.GenerateHTMLReport(taskID)

// Export as JSON
jsonData, err := reporter.ExportReportJSON(taskID)
```

#### Report Contents:
- Migration summary
- Resource-level details
- Performance metrics
- Event timeline
- Recommendations

## Migration Process

### Step 1: Analysis

Before migration, analyze your resources:

```bash
gunj-migrate analyze --target-version v1beta1 --all-namespaces
```

Output:
```
Migration Analysis Report
========================

Target Version: v1beta1
Resources Found: 25

Resources Requiring Migration: 20
Resources Already at Target Version: 5

Warnings:
  - prod-platform/production: Migration may result in data loss
  - staging-platform/staging: Manual intervention required

Migration Complexity Assessment:
  - Medium: Moderate number of resources

Recommendations:
  - Run migration in dry-run mode first
  - Backup resources before migration
  - Monitor migration progress closely
```

### Step 2: Dry Run

Test the migration without making changes:

```bash
gunj-migrate migrate --target-version v1beta1 --all-namespaces --dry-run
```

### Step 3: Execute Migration

Perform the actual migration:

```bash
gunj-migrate migrate --target-version v1beta1 --all-namespaces
```

### Step 4: Monitor Progress

Check migration status:

```bash
gunj-migrate status migrate-12345
```

### Step 5: Generate Report

Create a detailed report:

```bash
gunj-migrate report migrate-12345 --format html --output report.html
```

## CLI Usage

### Installation

```bash
go install github.com/gunjanjp/gunj-operator/cmd/migrate@latest
```

### Common Commands

#### Migrate Single Resource
```bash
gunj-migrate migrate my-platform \
  --namespace production \
  --target-version v1beta1
```

#### Batch Migration with Options
```bash
gunj-migrate migrate \
  --all-namespaces \
  --target-version v1beta1 \
  --batch-size 20 \
  --max-concurrent 10 \
  --enable-optimization
```

#### Check Migration Status
```bash
# List all active migrations
gunj-migrate status

# Check specific migration
gunj-migrate status migrate-12345
```

#### Rollback Migration
```bash
gunj-migrate rollback migrate-12345
```

## API Integration

### Using Migration Manager in Code

```go
package main

import (
    "context"
    "github.com/gunjanjp/gunj-operator/api/v1beta1/migration"
    "k8s.io/apimachinery/pkg/types"
)

func migrateResources(ctx context.Context) error {
    // Create migration manager
    config := migration.MigrationConfig{
        MaxConcurrentMigrations: 5,
        BatchSize:               10,
        RetryAttempts:           3,
        EnableOptimizations:     true,
    }
    
    migrationManager := migration.NewMigrationManager(
        k8sClient, 
        scheme, 
        logger, 
        config,
    )
    
    // Single resource migration
    resource := types.NamespacedName{
        Name:      "my-platform",
        Namespace: "production",
    }
    
    err := migrationManager.MigrateResource(ctx, resource, "v1beta1")
    if err != nil {
        return fmt.Errorf("migration failed: %w", err)
    }
    
    // Batch migration
    resources := []types.NamespacedName{
        {Name: "platform-1", Namespace: "default"},
        {Name: "platform-2", Namespace: "default"},
    }
    
    task, err := migrationManager.MigrateBatch(ctx, resources, "v1beta1")
    if err != nil {
        return fmt.Errorf("batch migration failed: %w", err)
    }
    
    // Monitor progress
    for {
        status, err := migrationManager.GetMigrationStatus(task.ID)
        if err != nil {
            return err
        }
        
        if status.Status != migration.MigrationStatusInProgress {
            break
        }
        
        fmt.Printf("Progress: %d/%d\n", 
            status.Progress.MigratedResources,
            status.Progress.TotalResources)
        
        time.Sleep(5 * time.Second)
    }
    
    return nil
}
```

### Webhook Integration

The conversion webhook automatically uses migration helpers:

```go
// Webhook automatically:
// 1. Tracks schema evolution
// 2. Optimizes conversions
// 3. Manages lifecycle hooks
// 4. Creates rollback snapshots
// 5. Reports migration status

webhook := v1beta1.NewConversionWebhook(client, scheme)
```

## Best Practices

### 1. Always Run Analysis First

Before migrating, analyze your resources to identify potential issues:

```bash
gunj-migrate analyze --target-version v1beta1 --all-namespaces
```

### 2. Use Dry Run Mode

Test migrations without making changes:

```bash
gunj-migrate migrate --dry-run --target-version v1beta1
```

### 3. Migrate in Batches

For large deployments, migrate in smaller batches:

```bash
# Migrate by namespace
gunj-migrate migrate --namespace development --target-version v1beta1
gunj-migrate migrate --namespace staging --target-version v1beta1
gunj-migrate migrate --namespace production --target-version v1beta1
```

### 4. Monitor Progress

Keep track of migration progress:

```bash
# Watch migration status
watch -n 5 gunj-migrate status migrate-12345
```

### 5. Review Reports

Always review migration reports:

```bash
gunj-migrate report migrate-12345 --format html --output report.html
open report.html
```

### 6. Plan for Rollback

Be prepared to rollback if issues occur:

```bash
# Save task IDs
export MIGRATION_TASK=migrate-12345

# Rollback if needed
gunj-migrate rollback $MIGRATION_TASK
```

## Troubleshooting

### Common Issues

#### 1. Migration Stuck in Progress

**Symptom**: Migration shows "InProgress" for extended time

**Solution**:
```bash
# Check detailed status
gunj-migrate status migrate-12345

# Check operator logs
kubectl logs -n gunj-system deployment/gunj-operator

# Force retry
kubectl annotate observabilityplatform my-platform \
  observability.io/force-reconcile=$(date +%s)
```

#### 2. Validation Failures

**Symptom**: Migration fails with validation errors

**Solution**:
```bash
# Run dry-run to see validation details
gunj-migrate migrate my-platform --dry-run --target-version v1beta1

# Fix issues in resource spec
kubectl edit observabilityplatform my-platform

# Retry migration
gunj-migrate migrate my-platform --target-version v1beta1
```

#### 3. Data Loss Warnings

**Symptom**: Warning about potential data loss

**Solution**:
```bash
# Export resource before migration
kubectl get observabilityplatform my-platform -o yaml > backup.yaml

# Review field changes
gunj-migrate analyze --target-version v1beta1

# Proceed with caution
gunj-migrate migrate my-platform --target-version v1beta1
```

#### 4. Performance Issues

**Symptom**: Slow migration performance

**Solution**:
```bash
# Increase concurrency
gunj-migrate migrate --max-concurrent 20 --batch-size 50

# Enable optimizations
gunj-migrate migrate --enable-optimization

# Check resource limits
kubectl describe deployment gunj-operator -n gunj-system
```

### Debug Mode

Enable detailed logging:

```bash
# Set log level
export GUNJ_LOG_LEVEL=debug

# Run with verbose output
gunj-migrate migrate --verbose --target-version v1beta1
```

### Getting Help

1. **Check logs**:
   ```bash
   kubectl logs -n gunj-system deployment/gunj-operator -f
   ```

2. **Generate diagnostic bundle**:
   ```bash
   gunj-migrate diagnose --output diagnostic-bundle.tar.gz
   ```

3. **Community support**:
   - GitHub Issues: https://github.com/gunjanjp/gunj-operator/issues
   - Slack: #gunj-operator

## Performance Considerations

### Optimization Strategies

1. **Caching**: Conversion results are cached for repeated operations
2. **Batching**: Similar resources are grouped for efficiency
3. **Parallelization**: Independent operations run concurrently
4. **Compression**: Large fields are compressed during conversion

### Performance Metrics

Monitor migration performance:

```bash
# Prometheus metrics
gunj_operator_migration_duration_seconds
gunj_operator_migration_batch_size
gunj_operator_migration_cache_hit_rate
gunj_operator_migration_optimization_rate
```

### Tuning Parameters

Adjust for your environment:

```yaml
# High-performance configuration
config:
  maxConcurrentMigrations: 20
  batchSize: 100
  enableOptimizations: true
  cacheSize: 5000
  cacheTTL: 10m
  parallelWorkers: 10
```

## Security Considerations

### RBAC Requirements

The migration tool requires appropriate permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gunj-migration-operator
rules:
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms"]
  verbs: ["get", "list", "update", "patch"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
```

### Audit Trail

All migrations are logged for audit:

```bash
# View migration events
kubectl get events --field-selector reason=MigrationCompleted

# Check audit logs
kubectl logs -n gunj-system deployment/gunj-operator | grep -i migration
```

## Conclusion

The Gunj Operator migration helpers provide a robust, safe, and efficient way to manage API version transitions. By following the best practices and using the provided tools, you can ensure smooth migrations with minimal risk and downtime.

For more information:
- [API Reference](../api/README.md)
- [Architecture Guide](../architecture/README.md)
- [Troubleshooting Guide](../troubleshooting/README.md)
