# Commit Message Conventions Guide

This guide provides detailed instructions and best practices for writing commit messages in the Gunj Operator project.

## Table of Contents

- [Overview](#overview)
- [Quick Reference](#quick-reference)
- [Detailed Format](#detailed-format)
- [Types](#types)
- [Scopes](#scopes)
- [Examples](#examples)
- [Tools and Automation](#tools-and-automation)
- [FAQ](#faq)

## Overview

The Gunj Operator project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification. This provides several benefits:

- **Automated Changelog Generation**: Commit messages are used to generate release notes
- **Semantic Versioning**: Commit types determine version bumps
- **Better History**: Consistent format makes git history more readable
- **Team Communication**: Clear commits help team members understand changes
- **CI/CD Integration**: Automated validation ensures consistency

## Quick Reference

### Basic Format

```
type(scope): subject

body

footer
```

### Quick Examples

```bash
# Feature
git commit -m "feat(operator): add support for custom metrics"

# Bug fix
git commit -m "fix(api): correct authentication token validation"

# Documentation
git commit -m "docs(ui): update installation instructions"

# With sign-off (required)
git commit -s -m "feat(operator): add prometheus integration"
```

## Detailed Format

### Structure

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

### Rules

1. **Header** (first line):
   - Maximum 100 characters
   - Type is mandatory
   - Scope is optional but recommended
   - Subject is mandatory

2. **Body**:
   - Separated from header by blank line
   - Maximum 72 characters per line
   - Explain what and why, not how
   - Use imperative mood

3. **Footer**:
   - Separated from body by blank line
   - Contains breaking changes and issue references
   - Must include DCO sign-off

## Types

### Primary Types

| Type | Description | Version Impact |
|------|-------------|----------------|
| `feat` | New feature | Minor |
| `fix` | Bug fix | Patch |
| `docs` | Documentation changes | None |
| `style` | Code style changes (formatting, etc.) | None |
| `refactor` | Code refactoring | None |
| `perf` | Performance improvements | Patch |
| `test` | Test additions or corrections | None |
| `build` | Build system or dependencies | None |
| `ci` | CI/CD configuration | None |
| `chore` | Other changes | None |
| `revert` | Revert a previous commit | Varies |

### When to Use Each Type

#### `feat` - Features
```bash
# Adding new functionality
feat(operator): add multi-cluster support
feat(ui): implement dark mode theme
feat(api): add GraphQL endpoint for metrics
```

#### `fix` - Bug Fixes
```bash
# Fixing broken functionality
fix(controllers): resolve memory leak in reconciliation loop
fix(ui): correct dropdown selection behavior
fix(webhooks): handle nil pointer in validation
```

#### `docs` - Documentation
```bash
# Documentation updates
docs(operator): add troubleshooting guide
docs(api): update REST API examples
docs: fix typos in README
```

#### `refactor` - Code Refactoring
```bash
# Code restructuring without changing behavior
refactor(controllers): extract common logic to utils
refactor(ui): convert class components to hooks
refactor: rename variables for clarity
```

## Scopes

### Available Scopes

| Scope | Description |
|-------|-------------|
| `operator` | Core operator functionality |
| `api` | API server (REST/GraphQL) |
| `ui` | React user interface |
| `controllers` | Kubernetes controllers |
| `crd` | Custom Resource Definitions |
| `webhooks` | Admission/validation webhooks |
| `helm` | Helm charts |
| `docs` | Documentation |
| `deps` | Dependencies |
| `security` | Security-related changes |

### Scope Guidelines

- Use scope to indicate the area of change
- Omit scope for changes affecting multiple areas
- Be consistent with existing scopes
- Keep scopes short and meaningful

## Examples

### Simple Examples

```bash
# Feature with scope
feat(operator): add prometheus scrape interval configuration

# Bug fix without body
fix(ui): correct platform status display

# Docs update
docs: update kubernetes version requirements

# Build changes
build(deps): bump golang from 1.20 to 1.21

# Style changes
style(operator): format imports according to goimports
```

### Complete Examples

#### Feature with Full Details
```
feat(operator): implement automated backup functionality

Add automated backup capability for observability platforms. Backups can be 
scheduled or triggered manually, with support for S3, GCS, and Azure Blob 
storage backends.

- Add backup CRD and controller
- Implement storage backend abstraction  
- Create backup scheduling logic
- Add restoration functionality
- Include progress tracking

The backup data includes:
- Prometheus data (compressed)
- Grafana dashboards and settings
- Loki logs (with retention)
- Custom configurations

Closes #234
Closes #235

Signed-off-by: Jane Developer <jane@example.com>
```

#### Bug Fix with Breaking Change
```
fix(api): standardize error response format

Previously, API errors were returned in inconsistent formats depending on 
the endpoint. This made client error handling difficult and error-prone.

This change standardizes all error responses to follow the format:
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "details": {}
  }
}

BREAKING CHANGE: API error responses now use a different JSON structure.
Clients must update their error parsing logic to handle the new format.

Fixes #567

Signed-off-by: John Developer <john@example.com>
```

#### Performance Improvement
```
perf(controllers): optimize reconciliation for large clusters

Improve reconciliation performance by implementing intelligent caching and
reducing unnecessary API calls. This significantly improves operator 
performance in clusters with 100+ platforms.

Changes:
- Add in-memory cache for frequently accessed resources
- Implement batch processing for status updates  
- Use pagination for large list operations
- Add circuit breaker for failing API calls

Benchmarks show:
- 70% reduction in API calls during steady state
- 50% improvement in reconciliation latency
- 80% reduction in memory allocations

Fixes #789

Signed-off-by: Alice Engineer <alice@example.com>
```

## Tools and Automation

### Commit Helper Script

Use the provided commit helper for guided commit creation:

```bash
# Run the interactive commit helper
./scripts/commit-helper.sh
```

### Git Configuration

Set up git to use the commit template:

```bash
# Configure locally for this project
git config commit.template .gitmessage

# Sign commits automatically  
git config user.signingkey YOUR_GPG_KEY
git config commit.gpgsign true
```

### Validation

Commits are automatically validated by:

1. **Pre-commit hook**: Validates before commit
2. **CI Pipeline**: Validates in pull requests
3. **Manual validation**:
   ```bash
   # Validate last commit
   npx commitlint --from=HEAD~1
   
   # Validate range
   npx commitlint --from=origin/main --to=HEAD
   ```

### IDE Integration

#### VS Code

1. Install "Conventional Commits" extension
2. Use Command Palette: `Conventional Commits`

#### IntelliJ IDEA

1. Install "Conventional Commit" plugin
2. Use commit dialog with convention support

## FAQ

### Q: What if I make a mistake in a commit message?

For the last commit:
```bash
git commit --amend
```

For older commits (use with caution):
```bash
git rebase -i HEAD~n  # n = number of commits back
```

### Q: How do I reference multiple issues?

```
feat(operator): add monitoring dashboard

Implement comprehensive monitoring dashboard with real-time metrics,
alerts, and resource usage visualization.

Closes #123
Closes #124
Fixes #125
```

### Q: What about work-in-progress commits?

For WIP commits in feature branches:
```bash
# WIP commits are fine in feature branches
git commit -m "WIP: implementing validation logic"

# Squash before merging to main
git rebase -i main
```

### Q: Do I need to sign-off every commit?

Yes, DCO (Developer Certificate of Origin) sign-off is required:
```bash
# Automatic sign-off
git commit -s

# Add to existing commit
git commit --amend -s
```

### Q: What if my change doesn't fit any type?

Use `chore` for changes that don't fit other categories:
```bash
chore: update git ignores
chore: configure editor settings
```

### Q: How detailed should the body be?

Include enough detail so that:
- Future developers understand why the change was made
- The change can be reviewed without looking at code
- Any special considerations are documented

## Best Practices

1. **Write for your future self**: Will you understand this in 6 months?
2. **Be specific**: "fix bug" vs "fix nil pointer in webhook validation"
3. **Use present tense**: "add" not "added"
4. **Reference issues**: Always link to related issues
5. **Explain why**: Code shows what, commits explain why
6. **Keep it atomic**: One logical change per commit
7. **Test before committing**: Ensure tests pass
8. **Review before pushing**: Read your commit message

## Additional Resources

- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)
- [Angular Commit Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md)
- [Semantic Versioning](https://semver.org/)

---

For questions or clarifications, please refer to the main [CONTRIBUTING.md](../CONTRIBUTING.md) or contact the maintainers.
