# Commit Message Conventions

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Project**: Gunj Operator  
**Status**: Active  

---

## 📋 Overview

The Gunj Operator project follows the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. This ensures consistent commit messages that can be parsed by automated tools for changelog generation, versioning, and release notes.

## 🎯 Commit Message Format

Each commit message consists of a **header**, an optional **body**, and an optional **footer**.

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Example:
```
feat(operator): add reconciliation retry mechanism

Implemented exponential backoff for failed reconciliation attempts.
This improves reliability when temporary failures occur during
resource creation or updates.

Closes #123
```

---

## 📝 Message Components

### Type (Required)

The type must be one of the following:

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat(api): add GraphQL endpoint for metrics` |
| `fix` | Bug fix | `fix(controller): resolve memory leak in reconciler` |
| `docs` | Documentation changes | `docs(readme): update installation instructions` |
| `style` | Code style changes (formatting, missing semicolons, etc.) | `style(ui): fix indentation in components` |
| `refactor` | Code refactoring without feature changes | `refactor(api): simplify authentication middleware` |
| `perf` | Performance improvements | `perf(operator): optimize resource caching` |
| `test` | Adding or updating tests | `test(e2e): add platform creation tests` |
| `build` | Build system or dependencies | `build(docker): optimize container image size` |
| `ci` | CI/CD configuration changes | `ci(github): add security scanning workflow` |
| `chore` | Maintenance tasks | `chore(deps): update Go dependencies` |
| `revert` | Revert a previous commit | `revert: feat(api): add GraphQL endpoint` |
| `wip` | Work in progress (for draft PRs only) | `wip(ui): dashboard redesign` |

### Scope (Required)

The scope indicates the component or area affected:

#### Core Components
- `operator` - Main operator logic
- `api` - API server (REST/GraphQL)
- `ui` - Web user interface
- `cli` - Command-line interface

#### Operator Specifics
- `controller` - Controller logic
- `webhook` - Admission/conversion webhooks
- `crd` - Custom Resource Definitions
- `rbac` - RBAC configurations

#### API Specifics
- `rest` - RESTful API endpoints
- `graphql` - GraphQL schema/resolvers
- `auth` - Authentication/authorization

#### UI Specifics
- `components` - React components
- `pages` - Page components
- `hooks` - Custom hooks
- `store` - State management

#### Infrastructure
- `docker` - Dockerfile changes
- `k8s` - Kubernetes manifests
- `helm` - Helm chart changes
- `ci` - CI/CD pipeline

#### Other
- `docs` - Documentation
- `examples` - Example configurations
- `test` - General testing
- `e2e` - End-to-end tests
- `deps` - Dependencies
- `*` - Multiple scopes or general changes

### Subject (Required)

The subject is a short description of the change:

- Use imperative, present tense: "add" not "added" or "adds"
- Don't capitalize the first letter
- No period (.) at the end
- Maximum 50 characters

✅ Good: `add prometheus service monitor creation`  
❌ Bad: `Added Prometheus service monitor creation.`

### Body (Optional)

The body provides additional context:

- Use imperative, present tense
- Include motivation for the change
- Contrast with previous behavior
- Wrap at 72 characters
- Separate from header with blank line

Example:
```
Implement automatic service monitor creation for Prometheus
to discover and scrape metrics from platform components.

Previously, service monitors had to be created manually,
which was error-prone and inconsistent across deployments.
```

### Footer (Optional)

The footer contains:

#### Breaking Changes
```
BREAKING CHANGE: The 'metrics' field in ObservabilityPlatform CRD
has been renamed to 'monitoring'. Update all existing resources.
```

#### Issue References
```
Closes #123
Fixes #456
Related to #789
```

#### Co-authors
```
Co-authored-by: Jane Doe <jane@example.com>
```

---

## 🔧 Validation & Automation

### Commitlint Setup

The project uses `commitlint` to validate commit messages:

```bash
# Install dependencies
npm install --save-dev @commitlint/cli @commitlint/config-conventional

# Test a commit message
echo "feat(operator): add new feature" | npx commitlint
```

### Husky Git Hooks

Commit messages are automatically validated using Husky:

```bash
# The hook is already configured in .husky/commit-msg
# It runs commitlint on every commit
```

### Making Commits

```bash
# Stage changes
git add .

# Commit with valid message
git commit -m "feat(operator): add health check endpoint"

# For multi-line commits
git commit

# Then write:
# feat(api): implement rate limiting
#
# Add configurable rate limiting to prevent API abuse.
# Supports both IP-based and token-based limiting.
#
# Closes #234
```

---

## 📚 Examples

### Feature Addition
```
feat(ui): add dark mode support

Implemented a theme toggle that allows users to switch
between light and dark modes. The preference is saved
in localStorage and applied on page load.

Closes #150
```

### Bug Fix
```
fix(controller): prevent duplicate resource creation

Added mutex locking to ensure only one reconciliation
process can create resources at a time. This fixes the
race condition that caused duplicate StatefulSets.

Fixes #201
```

### Breaking Change
```
refactor(api)!: change authentication to JWT

Replaced session-based authentication with JWT tokens.
This improves scalability and enables stateless API servers.

BREAKING CHANGE: API clients must now include JWT tokens
in the Authorization header instead of session cookies.

Migration guide: docs/migration/v2-auth.md
```

### Documentation Update
```
docs(install): add production deployment guide

Created comprehensive guide for deploying the operator
in production environments, including:
- Resource requirements
- Security considerations
- High availability setup
- Monitoring configuration
```

### Performance Improvement
```
perf(operator): optimize reconciliation loop

Reduced reconciliation time by 60% through:
- Caching Kubernetes API responses
- Parallel processing of independent resources
- Skipping unchanged resources

Benchmark results in test/performance/reconcile_bench.go
```

---

## 🚫 Common Mistakes

### ❌ Wrong Type
```
update(api): add new endpoint     # Wrong: 'update' is not a valid type
feat(api): add new endpoint       # Correct
```

### ❌ Missing Scope
```
feat: add new endpoint            # Wrong: missing scope
feat(api): add new endpoint       # Correct
```

### ❌ Capitalized Subject
```
feat(api): Add new endpoint       # Wrong: capitalized
feat(api): add new endpoint       # Correct
```

### ❌ Past Tense
```
feat(api): added new endpoint     # Wrong: past tense
feat(api): add new endpoint       # Correct
```

### ❌ Period at End
```
feat(api): add new endpoint.      # Wrong: period at end
feat(api): add new endpoint       # Correct
```

---

## 🤖 Tooling Integration

### VS Code
Install the "Conventional Commits" extension for commit message assistance.

### IntelliJ IDEA
Enable the Git Commit Template plugin and configure with our format.

### Git Aliases
```bash
# Add to ~/.gitconfig
[alias]
    # Feature commit
    cf = "!f() { git commit -m \"feat($1): $2\"; }; f"
    # Fix commit
    cx = "!f() { git commit -m \"fix($1): $2\"; }; f"
    # Docs commit
    cd = "!f() { git commit -m \"docs($1): $2\"; }; f"
```

Usage:
```bash
git cf operator "add retry logic"
# Creates: feat(operator): add retry logic
```

---

## 📊 Benefits

Following these conventions enables:

1. **Automated Changelog Generation** - Tools can parse commits to generate changelogs
2. **Semantic Versioning** - Determine version bumps based on commit types
3. **Better History** - Easily understand what changed and why
4. **CI/CD Integration** - Trigger different pipelines based on commit types
5. **Team Communication** - Clear, consistent communication of changes

---

## 🔗 Resources

- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Angular Commit Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#-commit-message-format)
- [Commitlint Documentation](https://commitlint.js.org/)
- [Semantic Versioning](https://semver.org/)

---

**Questions?** Reach out on our [Slack channel](https://gunj-operator.slack.com) or create a [discussion](https://github.com/gunjanjp/gunj-operator/discussions).
