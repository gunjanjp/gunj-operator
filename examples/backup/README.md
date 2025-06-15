# Backup and Restore Examples

This directory contains examples of backup and restore configurations for the Gunj Operator.

## Examples

1. **basic-backup.yaml** - Simple backup configuration
2. **scheduled-backup.yaml** - Scheduled backup with retention policy
3. **encrypted-backup.yaml** - Backup with encryption enabled
4. **selective-backup.yaml** - Backup with namespace and resource filtering
5. **restore-example.yaml** - Basic restore configuration
6. **cross-namespace-restore.yaml** - Restore with namespace mapping

## Usage

### Creating a Backup

```bash
# Apply the backup configuration
kubectl apply -f basic-backup.yaml

# Check backup status
kubectl get backups -n gunj-system

# View backup details
kubectl describe backup manual-backup-1 -n gunj-system
```

### Creating a Scheduled Backup

```bash
# Apply the scheduled backup
kubectl apply -f scheduled-backup.yaml

# Check schedule status
kubectl get backupschedules -n gunj-system

# View schedule details
kubectl describe backupschedule daily-backup -n gunj-system
```

### Performing a Restore

```bash
# Apply the restore configuration
kubectl apply -f restore-example.yaml

# Check restore status
kubectl get restores -n gunj-system

# View restore progress
kubectl describe restore platform-restore-1 -n gunj-system
```

## Storage Configuration

Before using these examples, ensure you have configured your storage backend:

### S3 Storage

```bash
# Create secret with AWS credentials
kubectl create secret generic s3-credentials \
  --from-literal=access-key-id=YOUR_ACCESS_KEY \
  --from-literal=secret-access-key=YOUR_SECRET_KEY \
  -n gunj-system
```

### GCS Storage

```bash
# Create secret with GCS service account
kubectl create secret generic gcs-credentials \
  --from-file=service-account.json=path/to/service-account.json \
  -n gunj-system
```

### Azure Blob Storage

```bash
# Create secret with Azure storage account
kubectl create secret generic azure-credentials \
  --from-literal=storage-account=YOUR_STORAGE_ACCOUNT \
  --from-literal=storage-key=YOUR_STORAGE_KEY \
  -n gunj-system
```

## Encryption Setup

For encrypted backups:

```bash
# Create encryption key secret
kubectl create secret generic backup-encryption-key \
  --from-literal=key=$(openssl rand -base64 32) \
  -n gunj-system
```

## Monitoring Backups

View backup metrics:

```bash
# Port-forward to Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Query backup metrics
# - gunj_backup_total
# - gunj_backup_duration_seconds
# - gunj_backup_size_bytes
# - gunj_backup_errors_total
```

## Troubleshooting

Check operator logs for backup/restore issues:

```bash
# View backup controller logs
kubectl logs -n gunj-system deployment/gunj-operator -c manager | grep backup-controller

# View restore controller logs
kubectl logs -n gunj-system deployment/gunj-operator -c manager | grep restore-controller
```
