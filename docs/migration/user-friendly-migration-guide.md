# Migration Guide: v1alpha1 to v1beta1 (User-Friendly)

## 🚀 Quick Start

**Estimated Time**: 30 minutes per platform  
**Difficulty**: Medium  
**Downtime**: Zero (with proper planning)

### What's Changing?

The Gunj Operator is evolving! Version v1beta1 brings:
- ✨ Better Kubernetes integration
- 🔒 Enhanced security by default
- 📊 Improved monitoring capabilities
- 🚀 Automatic high availability
- 💰 Cost optimization features

### Do I Need to Migrate?

**Yes, if you're using v1alpha1.** The old version will stop working on **January 15, 2025**.

### When Should I Migrate?

**Now!** The sooner you migrate, the more time you have to test and adjust.

---

## 📋 Pre-Migration Checklist

Before you start, make sure you have:

- [ ] Access to your Kubernetes cluster
- [ ] `kubectl` command-line tool installed
- [ ] Current configuration files backed up
- [ ] 30 minutes of uninterrupted time
- [ ] A test environment (recommended)

---

## 🔧 Migration Steps

### Step 1: Check Your Current Version

```bash
# See what version you're running
kubectl get observabilityplatforms.observability.io --all-namespaces
```

If you see `v1alpha1` in the output, you need to migrate.

### Step 2: Back Up Everything

```bash
# Create a backup (IMPORTANT!)
kubectl get observabilityplatforms.observability.io --all-namespaces -o yaml > backup.yaml

# Save it somewhere safe
cp backup.yaml ~/important-backups/gunj-backup-$(date +%Y%m%d).yaml
```

### Step 3: Download Migration Tool

```bash
# Download the migration tool
curl -LO https://github.com/gunjanjp/gunj-operator/releases/latest/download/gunj-migrate
chmod +x gunj-migrate

# Check it works
./gunj-migrate --version
```

### Step 4: Run Migration Check

```bash
# This won't change anything, just checks for issues
./gunj-migrate check --file backup.yaml
```

You'll see output like:
```
✓ Platform 'production' can be migrated automatically
⚠ Platform 'development' has custom configuration that needs attention
✗ Platform 'legacy' has incompatible settings
```

### Step 5: Fix Any Issues

If you see warnings or errors, here's what to do:

#### Custom Configuration Warning
Your old configuration:
```yaml
prometheus:
  customConfig: |
    global:
      scrape_interval: 15s
```

Needs to become:
```yaml
prometheus:
  global:
    scrapeInterval: 15s
```

#### Resource Format Warning
Your old configuration:
```yaml
prometheus:
  cpuRequest: "1"
  memoryRequest: "4Gi"
```

Needs to become:
```yaml
prometheus:
  resources:
    requests:
      cpu: "1"
      memory: "4Gi"
```

### Step 6: Apply Migration

```bash
# First, do a dry run (safe - no changes)
./gunj-migrate apply --dry-run --file backup.yaml

# If everything looks good, do it for real
./gunj-migrate apply --file backup.yaml
```

### Step 7: Verify Everything Works

```bash
# Check platform status
kubectl get observabilityplatforms.observability.io --all-namespaces

# Check if Prometheus is running
kubectl get pods -n monitoring | grep prometheus

# Check if Grafana is accessible
kubectl get svc -n monitoring | grep grafana
```

---

## 🎯 Common Scenarios

### Scenario 1: "I just have basic Prometheus and Grafana"

Great! Your migration will be simple:

1. Run the migration tool
2. It will automatically update your configuration
3. No manual changes needed

### Scenario 2: "I have custom Prometheus configuration"

You'll need to restructure your configuration:

**Before:**
```yaml
spec:
  components:
    prometheus:
      customConfig: |
        # Your prometheus.yml content here
```

**After:**
```yaml
spec:
  components:
    prometheus:
      # Your configuration in structured format
      # The migration tool helps with this!
```

### Scenario 3: "I use GitOps (ArgoCD/Flux)"

1. Update your Git repository with the new format
2. Test in a branch first
3. Use the migration tool locally to convert files
4. Commit and let GitOps handle the rest

### Scenario 4: "I have multiple clusters"

Migrate one cluster at a time:
1. Start with development
2. Move to staging
3. Finally update production
4. Use the same process for each

---

## ⚡ Quick Fixes

### Problem: "Field 'customConfig' is not supported"

**Solution**: Use the migration tool to convert it automatically, or manually restructure following the examples above.

### Problem: "Resource quota exceeded"

**Solution**: The new version uses more resources for better reliability. Either:
- Increase your resource quota
- Temporarily disable high availability:
  ```yaml
  prometheus:
    replicas: 1  # Instead of 2
  ```

### Problem: "Version format is invalid"

**Solution**: Add a 'v' prefix to versions:
- Old: `version: "2.45.0"`
- New: `version: "v2.45.0"`

### Problem: "Storage configuration not recognized"

**Solution**: Update the structure:
```yaml
# Old
prometheus:
  storageSize: 100Gi

# New
prometheus:
  storage:
    size: 100Gi
```

---

## 🆘 Getting Help

### Self-Help Resources

1. **Check the logs**:
   ```bash
   kubectl logs -n gunj-system deployment/gunj-operator
   ```

2. **Read the detailed guide**: [Breaking Changes Guide](breaking-changes-v2.md)

3. **Watch the video tutorial**: [YouTube - Gunj Migration](https://youtube.com/watch?v=...)

### Community Help

- **Slack**: Join #gunj-operator-help
- **GitHub**: Open an issue with the `migration` label
- **Forum**: https://discuss.gunj.io

### Professional Help

For critical systems or complex setups:
- Email: gunjanjp@gmail.com
- Emergency: support-emergency@gunj.io

---

## ✅ Post-Migration Checklist

After migrating, verify:

- [ ] All platforms show "Ready" status
- [ ] Metrics are still being collected
- [ ] Dashboards are accessible
- [ ] Alerts are working
- [ ] No errors in operator logs
- [ ] Resource usage is acceptable

---

## 🎉 Congratulations!

You've successfully migrated to v1beta1! Here's what you get:

- **Better Performance**: Optimized resource usage
- **Enhanced Security**: Secure by default
- **New Features**: Multi-cluster support, cost management
- **Future Proof**: Ready for upcoming features

### What's Next?

1. **Explore new features**: Check out multi-cluster support
2. **Optimize costs**: Use the cost analysis dashboard
3. **Join the community**: Share your migration experience
4. **Stay updated**: Follow @GunjOperator on Twitter

---

## 📚 Appendix

### Minimal Migration Example

**Your current file** (`platform-old.yaml`):
```yaml
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: my-platform
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      version: "2.45.0"
      storageSize: 50Gi
```

**After migration** (`platform-new.yaml`):
```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: my-platform
  namespace: monitoring
spec:
  components:
    prometheus:
      enabled: true
      version: "v2.48.0"  # Note: version updated and 'v' prefix added
      storage:
        size: 50Gi      # Note: nested under 'storage'
```

### Migration Command Reference

```bash
# Check if migration is needed
./gunj-migrate check

# Convert a single file
./gunj-migrate convert -f old-platform.yaml -o new-platform.yaml

# Migrate all platforms in a cluster
./gunj-migrate apply --all-namespaces

# Migrate with custom operator namespace
./gunj-migrate apply --operator-namespace custom-system

# Generate migration report
./gunj-migrate report --output migration-report.html
```

---

**Remember**: Migration might seem daunting, but we're here to help! The tools do most of the work, and the community is always ready to assist.

Good luck! 🚀
