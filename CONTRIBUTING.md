# Contributing to Gunj Operator

Thank you for your interest in contributing to Gunj Operator! This document provides guidelines and conventions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Commit Message Conventions](#commit-message-conventions)
- [Pull Request Process](#pull-request-process)
- [Development Guidelines](#development-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

This project adheres to a Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to gunjanjp@gmail.com.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/gunj-operator.git`
3. Add upstream remote: `git remote add upstream https://github.com/gunjanjp/gunj-operator.git`
4. Create a feature branch: `git checkout -b feature/your-feature-name`
5. Make your changes following our guidelines
6. Commit your changes using our commit conventions
7. Push to your fork: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Commit Message Conventions

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification with some project-specific extensions. This leads to more readable messages that are easy to follow when looking through the project history and enables automatic changelog generation.

### Commit Message Format

Each commit message consists of a **header**, a **body**, and a **footer**. The header has a special format that includes a **type**, a **scope**, and a **subject**:

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

The **header** is mandatory and the **scope** of the header is optional.

### Type

Must be one of the following:

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system or external dependencies
- **ci**: Changes to our CI configuration files and scripts
- **chore**: Other changes that don't modify src or test files
- **revert**: Reverts a previous commit

### Scope

The scope should be the name of the package or component affected:

- **operator**: Changes to the operator core
- **api**: Changes to the API server
- **ui**: Changes to the React UI
- **controllers**: Changes to Kubernetes controllers
- **crd**: Changes to Custom Resource Definitions
- **webhooks**: Changes to admission/validation webhooks
- **helm**: Changes to Helm charts
- **docs**: Changes to documentation
- **deps**: Dependency updates
- **security**: Security-related changes

### Subject

The subject contains a succinct description of the change:

- Use the imperative, present tense: "change" not "changed" nor "changes"
- Don't capitalize the first letter
- No dot (.) at the end
- Limit to 72 characters

### Body

The body should include the motivation for the change and contrast this with previous behavior:

- Use the imperative, present tense
- Include motivation for the change
- Contrast this with previous behavior
- Wrap at 72 characters

### Footer

The footer should contain any information about **Breaking Changes** and is also the place to reference GitHub issues that this commit **Closes**.

**Breaking Changes** should start with the word `BREAKING CHANGE:` with a space or two newlines. The rest of the commit message is then used for this.

### Examples

#### Simple feature addition
```
feat(operator): add prometheus scrape interval configuration

Allow users to configure the scrape interval for Prometheus
through the ObservabilityPlatform CRD.

Closes #123
```

#### Bug fix with breaking change
```
fix(api): correct platform status endpoint response

Previously, the status endpoint returned inconsistent data structure
when components were in error state. This change standardizes the
response format.

BREAKING CHANGE: The status endpoint now returns errors as an array
instead of a string. Clients need to update their parsing logic.

Fixes #456
```

#### Documentation update
```
docs(operator): update installation instructions for v2.0

- Add prerequisites section
- Update minimum Kubernetes version to 1.26
- Include new RBAC requirements
- Add troubleshooting section
```

#### Performance improvement
```
perf(controllers): optimize reconciliation loop for large clusters

Implement caching layer for frequently accessed resources to reduce
API server load. Performance testing shows 60% reduction in API calls
during steady state.

Benchmarks:
- Before: 150ms average reconciliation time
- After: 60ms average reconciliation time
```

#### Complex change with multiple effects
```
feat(ui): implement real-time platform status dashboard

Add WebSocket-based real-time updates for platform status monitoring.
The dashboard shows component health, resource usage, and alerts in
real-time with automatic reconnection on connection loss.

- Add WebSocket client with exponential backoff
- Create status dashboard component with charts
- Implement state management using Zustand
- Add connection status indicator
- Include error boundary for graceful failures

This change requires the API server to be running v2.0+ with
WebSocket support enabled.

Closes #789, #790
```

### Commit Message Validation

All commit messages are validated using commitlint. The configuration can be found in `.commitlintrc.json`.

To test your commit message before committing:

```bash
echo "feat(operator): add new feature" | npx commitlint
```

### DCO Sign-off

All commits must be signed off according to the DCO (Developer Certificate of Origin):

```
Signed-off-by: Your Name <your.email@example.com>
```

You can automatically sign-off commits using:

```bash
git commit -s -m "your commit message"
```

## Pull Request Process

### Before Submitting a PR

1. **Check the PR checklist**: Review our [PR Submission Checklist](docs/development/pr-checklist.md)
2. **Ensure your branch is up to date**: Rebase on the latest main branch
3. **Run all tests locally**: `make test`
4. **Run linters**: `make lint`
5. **Update documentation**: Include any necessary documentation changes

### Submitting a PR

1. **Choose the right PR template**: We have templates for features, bug fixes, and documentation
2. **Write a clear PR title**: Follow the conventional commit format
3. **Provide a detailed description**: Explain what, why, and how
4. **Link related issues**: Use keywords like "Closes #123" or "Related to #456"
5. **Add screenshots**: For UI changes, include before/after screenshots
6. **Request reviews**: The CODEOWNERS file will auto-assign reviewers

### PR Review Process

#### Review Timeline
- **First response**: Within 24 hours
- **Complete review**: Within 48 hours
- **Follow-up reviews**: Within 24 hours

#### Review Stages
1. **Automated checks**: CI/CD, linting, tests, security scans
2. **Code review**: At least 2 maintainers review the code
3. **Testing**: Reviewers may test changes locally
4. **Approval**: 2 approvals required for merge

#### Review Feedback
- **🔴 Blocking**: Must be addressed before merge
- **🟡 Important**: Should be addressed
- **🟢 Suggestion**: Optional improvements
- **❓ Question**: Clarification needed
- **👍 Praise**: Acknowledgment of good work

### After Review

1. **Address feedback**: Respond to all comments
2. **Update PR**: Push changes to the same branch
3. **Re-request review**: Ask reviewers to take another look
4. **Wait for approval**: 2 approvals needed
5. **Merge**: Maintainers will merge when ready

### PR Commands

You can use these commands in PR comments:
- `/lgtm` - Looks good to me (informal approval)
- `/hold` - Prevent merge
- `/hold cancel` - Remove hold
- `/retest` - Re-run failed tests

For detailed review guidelines, see [PR Review Guidelines](docs/development/pr-review-guidelines.md).

## Development Guidelines

Please refer to [Development Guidelines](docs/development/guidelines.md) for detailed coding standards and best practices.

## Testing

Please refer to [Testing Guide](docs/development/testing.md) for information on running and writing tests.

## Documentation

Please refer to [Documentation Guide](docs/development/documentation.md) for guidelines on writing and maintaining documentation.
