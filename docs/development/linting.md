# Gunj Operator - Linting Configuration

This document describes the linting setup for the Gunj Operator project. We use comprehensive linting to ensure code quality, consistency, and adherence to CNCF best practices.

## Overview

The project uses multiple linters for different file types:

- **Go**: golangci-lint with extensive configuration
- **TypeScript/React**: ESLint with TypeScript and React plugins
- **YAML**: yamllint for Kubernetes manifests and configurations
- **Dockerfile**: hadolint for container best practices
- **Markdown**: markdownlint for documentation
- **Shell Scripts**: ShellCheck for bash scripts
- **CSS/SCSS**: Stylelint for styling files

## Quick Start

### Install Dependencies

```bash
# Install all linting tools
make deps-tools

# Install pre-commit hooks
make pre-commit-install
```

### Run All Linters

```bash
# Run all linters
make lint

# Run all linters and fix auto-fixable issues
make ci-fix
```

### Run Specific Linters

```bash
# Go linting
make lint-go
make lint-go-fix    # Auto-fix issues

# UI linting (TypeScript/React)
make lint-ui
make lint-ui-fix    # Auto-fix issues

# YAML linting
make lint-yaml

# Dockerfile linting
make lint-docker

# Markdown linting
make lint-markdown
make lint-markdown-fix    # Auto-fix issues

# Shell script linting
make lint-shell
```

## Go Linting Configuration

The `.golangci.yml` file configures comprehensive Go linting with:

- **Code Quality**: errcheck, govet, revive, gofmt, goimports
- **Complexity**: gocyclo, gocognit, funlen
- **Best Practices**: goconst, gocritic, gosec
- **Style**: godot, gofumpt, whitespace, wsl
- **Security**: gosec for security issues

Key settings:
- Line length: 120 characters
- Cyclomatic complexity: 15
- Cognitive complexity: 20
- Function length: 80 lines

## TypeScript/React Linting

The `ui/.eslintrc.js` file configures ESLint with:

- **TypeScript**: Full type checking and strict rules
- **React**: React 18 and hooks rules
- **Accessibility**: jsx-a11y plugin
- **Imports**: Organized and sorted imports
- **Security**: Basic security checks

Key features:
- Prettier integration for formatting
- Jest and Testing Library support
- Storybook file exceptions
- 80% test coverage requirement

## YAML Linting

The `.yamllint.yml` file ensures consistent YAML formatting:

- 2-space indentation
- Line length: 120 characters
- No trailing spaces
- Consistent quote usage

## Dockerfile Linting

The `.hadolint.yaml` file enforces:

- Best practices for container builds
- Security requirements
- Label schema validation
- Trusted registry checks

## Pre-commit Hooks

The `.pre-commit-config.yaml` file runs checks automatically:

```bash
# Manual run on all files
make pre-commit-run

# Or use pre-commit directly
pre-commit run --all-files
```

Hooks include:
- File size limits (1MB)
- Private key detection
- Branch protection (no direct commits to main)
- All linters mentioned above

## CI/CD Integration

GitHub Actions workflow (`.github/workflows/lint.yml`) runs all linters on:
- Push to main/develop branches
- Pull requests

The workflow includes:
- Parallel execution for faster feedback
- Security scanning with Trivy and gosec
- Results upload to GitHub Security tab

## Editor Integration

### VS Code

Install these extensions:
- Go (official)
- ESLint
- Prettier
- YAML
- markdownlint
- ShellCheck
- Docker
- EditorConfig

Settings are provided in `.vscode/settings.json`.

### IntelliJ IDEA / GoLand

1. Enable golangci-lint file watcher
2. Configure ESLint for TypeScript files
3. Enable EditorConfig support
4. Install Prettier plugin

## Code Formatting

### Format All Code

```bash
# Format all code
make fmt

# Check formatting without changes
make fmt-check
```

### Specific Formatting

```bash
# Format Go code
make fmt-go

# Format UI code
make fmt-ui
```

## Security Scanning

```bash
# Run all security checks
make sec

# Specific security checks
make sec-go      # Go security with gosec
make sec-ui      # npm audit
make sec-docker  # Trivy scan
```

## Troubleshooting

### golangci-lint Issues

If golangci-lint is slow:
```bash
# Run with specific linters
golangci-lint run --disable-all --enable=gofmt,govet,errcheck

# Increase timeout
golangci-lint run --timeout=20m
```

### ESLint Performance

For large codebases:
```bash
# Run on changed files only
cd ui && npx eslint --cache --ext .ts,.tsx src/
```

### Pre-commit Failures

If pre-commit fails:
```bash
# Skip hooks temporarily (not recommended)
git commit --no-verify

# Update hooks
pre-commit autoupdate
```

## Best Practices

1. **Run linters before committing**: Use pre-commit hooks
2. **Fix issues immediately**: Don't let linting errors accumulate
3. **Configure your editor**: Enable real-time linting
4. **Review linter rules**: Understand why rules exist
5. **Use auto-fix**: For formatting and simple issues
6. **Document exceptions**: When disabling rules, explain why

## Adding New Linters

To add a new linter:

1. Add configuration file to project root
2. Update Makefile with new targets
3. Add to pre-commit hooks
4. Update CI workflow
5. Document in this README

## Customizing Rules

Before changing linting rules:

1. Discuss with team
2. Document the reason
3. Update across all environments
4. Ensure CI passes
5. Update documentation

## Resources

- [golangci-lint documentation](https://golangci-lint.run/)
- [ESLint documentation](https://eslint.org/)
- [yamllint documentation](https://yamllint.readthedocs.io/)
- [hadolint documentation](https://github.com/hadolint/hadolint)
- [markdownlint documentation](https://github.com/DavidAnson/markdownlint)
- [ShellCheck documentation](https://www.shellcheck.net/)
