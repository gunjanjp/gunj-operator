# Commit Message Examples

This file contains real-world examples of commit messages following our conventions.

## ✅ Good Examples

### Feature Commits

```
feat(operator): add multi-cluster support

Implemented federation controller to manage ObservabilityPlatforms
across multiple Kubernetes clusters. Includes service discovery
and cross-cluster metric aggregation.

Closes #234
```

```
feat(ui): implement real-time metric streaming

Added WebSocket connection for live metric updates in the dashboard.
Updates are throttled to 1Hz to prevent overwhelming the browser.

Co-authored-by: Jane Smith <jane@example.com>
```

```
feat(api): add rate limiting middleware

Protect API endpoints from abuse by implementing token bucket
rate limiting. Configurable per endpoint with defaults:
- 100 requests/minute for authenticated users
- 20 requests/minute for anonymous users
```

### Bug Fixes

```
fix(controller): resolve race condition in resource creation

Multiple goroutines were attempting to create the same ConfigMap
simultaneously, causing intermittent failures. Added mutex locking
around the creation logic.

Fixes #456
Related to #123
```

```
fix(ui): correct timezone display in log viewer

Logs were displaying in UTC regardless of user's local timezone.
Now properly converts timestamps using the browser's timezone.
```

### Documentation

```
docs(readme): update installation instructions for v2.0

- Added prerequisites section
- Updated Helm chart values
- Added troubleshooting for common issues
- Included migration guide from v1.x
```

```
docs(api): add OpenAPI annotations to all endpoints

Generated comprehensive API documentation by adding OpenAPI
annotations. This enables auto-generated client SDKs and
interactive API exploration via Swagger UI.
```

### Performance Improvements

```
perf(operator): optimize reconciliation loop

Reduced reconciliation time by 60% through:
- Implementing resource caching with 5-minute TTL
- Batching API requests where possible
- Skipping reconciliation for unchanged resources

Benchmark results: pkg/benchmark/reconcile_test.go
```

```
perf(ui): lazy load dashboard components

Implemented code splitting for dashboard panels, reducing
initial bundle size by 40%. Components now load on-demand
as users navigate to different sections.
```

### Refactoring

```
refactor(api): extract authentication into middleware

Moved authentication logic from individual handlers into
reusable middleware. This reduces code duplication and
ensures consistent auth behavior across all endpoints.
```

```
refactor(controller): split monolithic reconciler

Broke down the 1000+ line reconciler into focused sub-reconcilers:
- ComponentReconciler for Prometheus/Grafana/Loki/Tempo
- ConfigReconciler for configuration management
- StatusReconciler for status updates

This improves testability and maintainability.
```

### Breaking Changes

```
refactor(api)!: change API versioning from URL to header

API version is now specified via 'API-Version' header instead
of URL path. This aligns with REST best practices and simplifies
routing.

BREAKING CHANGE: Clients must include 'API-Version: v1' header
in all requests. URL-based versioning (/v1/platforms) is no
longer supported. See migration guide: docs/migration/api-v2.md
```

```
feat(crd)!: rename 'metrics' field to 'monitoring'

Standardized field naming across all CRDs for consistency.

BREAKING CHANGE: The 'spec.metrics' field in ObservabilityPlatform
has been renamed to 'spec.monitoring'. Existing resources must be
updated before upgrading the operator.

Migration script: hack/migrate-v2.sh
```

### Build/CI Changes

```
build(docker): optimize image layers for caching

Restructured Dockerfile to maximize layer caching:
- Separate layers for dependencies and source code
- Use multi-stage build to reduce final image size
- Cache Go modules between builds

Image size reduced from 156MB to 87MB
```

```
ci(github): add matrix testing for Kubernetes versions

Test against Kubernetes 1.26, 1.27, 1.28, and 1.29 to ensure
compatibility. Also added ARM64 builds to the release pipeline.
```

### Chores

```
chore(deps): update Go dependencies

Updated all Go modules to latest stable versions:
- k8s.io/* modules to v0.29.0
- controller-runtime to v0.17.0
- gin to v1.9.1

All tests passing, no breaking changes detected.
```

```
chore(tools): add pre-commit hooks for code quality

Added hooks for:
- gofmt and goimports for Go files
- ESLint and Prettier for TypeScript/JavaScript
- yamllint for YAML files
- markdownlint for documentation
```

### Reverts

```
revert: feat(operator): add experimental caching layer

This reverts commit 3a4b5c6d.

The caching layer introduced memory leaks under high load.
Reverting until the issue is properly addressed.

See #789 for details
```

## ❌ Bad Examples (What NOT to do)

### Wrong Type
```
❌ update(api): add new endpoint
✅ feat(api): add new endpoint
```

### Missing Scope
```
❌ feat: add health check endpoint
✅ feat(operator): add health check endpoint
```

### Past Tense
```
❌ fix(controller): fixed memory leak
✅ fix(controller): fix memory leak
```

### Capitalized Subject
```
❌ feat(ui): Add dark mode toggle
✅ feat(ui): add dark mode toggle
```

### Subject Too Long
```
❌ feat(operator): implement comprehensive health checking system with multiple probe types
✅ feat(operator): add advanced health check system
```

### Unclear Subject
```
❌ fix(api): fix bug
✅ fix(api): correct JWT token expiration handling
```

### Multiple Changes in One Commit
```
❌ feat(operator): add health checks and fix memory leak and update docs

Better: Split into three commits:
✅ feat(operator): add health check endpoint
✅ fix(operator): resolve memory leak in cache
✅ docs(operator): update health check documentation
```

### Missing Context in Body
```
❌ fix(controller): fix reconciliation

✅ fix(controller): prevent reconciliation infinite loop

When errors occur during status updates, the controller was
immediately requeuing, causing high CPU usage. Now uses
exponential backoff with maximum 5 retries.
```

## 📝 Template for Complex Commits

```
<type>(<scope>): <concise summary of change>

<detailed explanation of what changed and why>
<include technical details if relevant>
<mention any side effects or important notes>

<breaking changes if any>
<issue references>
<co-authors if applicable>
```

### Real Example Using Template:

```
feat(operator): implement automated backup scheduling

Added CronJob-based backup automation for ObservabilityPlatforms.
Users can now configure scheduled backups via the CRD:

  spec:
    backup:
      schedule: "0 2 * * *"  # Daily at 2 AM
      retention: 7           # Keep 7 backups
      destination: s3://backups/

The operator creates a CronJob that triggers the backup controller,
which exports Prometheus data, Grafana dashboards, and Loki indices
to the specified destination.

Includes validation webhook to ensure valid cron expressions and
accessible backup destinations.

Closes #345
Closes #402
```
