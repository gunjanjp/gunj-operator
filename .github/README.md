# GitHub Actions Workflows

This directory contains the CI/CD workflows for the Gunj Operator project.

## Workflows

### 🔐 secret-test.yml
- **Purpose**: Validates that all required secrets are properly configured
- **Trigger**: Manual, on change, or weekly schedule
- **Use Case**: Run this after setting up or rotating secrets

### 🏗️ ci.yml
- **Purpose**: Main CI pipeline for building, testing, and publishing
- **Trigger**: Push to main/develop, pull requests, releases
- **Features**:
  - Multi-architecture builds (amd64, arm64)
  - Docker Hub and GHCR publishing
  - Security scanning with Snyk
  - Slack notifications

### 🔄 secret-rotation.yml
- **Purpose**: Creates GitHub issues for secret rotation reminders
- **Trigger**: Monthly (1st of each month)
- **Features**:
  - Monthly reminders for Docker Hub credentials
  - Quarterly reminders for API tokens
  - Automatic issue creation

## Required Secrets

Set these in Settings → Secrets and variables → Actions:

| Secret | Required | Description |
|--------|----------|-------------|
| DOCKER_USERNAME | ✅ | Docker Hub username |
| DOCKER_PASSWORD | ✅ | Docker Hub access token |
| SLACK_WEBHOOK_URL | ❌ | Slack notifications |
| GPG_PRIVATE_KEY | ❌ | Code signing key |
| GPG_PASSPHRASE | ❌ | GPG key passphrase |
| SNYK_TOKEN | ❌ | Security scanning |
| SONAR_TOKEN | ❌ | Code quality analysis |

## Quick Start

1. Fork/clone the repository
2. Set up required secrets (see docs/security/secret-management.md)
3. Run the secret test workflow to verify configuration
4. Make changes and watch the CI pipeline run automatically

## Security Notes

- Never commit secrets to the repository
- Use access tokens instead of passwords
- Rotate secrets regularly (see secret-rotation.yml)
- Follow the security guidelines in docs/security/

## Troubleshooting

If workflows fail:
1. Check the workflow logs in the Actions tab
2. Verify secrets are set correctly
3. Run the secret-test.yml workflow
4. Consult docs/security/secret-management.md

## Contributing

When adding new workflows:
- Use secret references, never hardcode values
- Add new secrets to secret-test.yml
- Update documentation
- Test thoroughly before merging