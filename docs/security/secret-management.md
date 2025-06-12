# Secret Management Guide

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Classification**: Internal  
**Owner**: DevOps Team  

## Overview

This document outlines the secret management procedures for the Gunj Operator project. It covers the creation, storage, rotation, and usage of secrets in our CI/CD pipeline.

## Table of Contents

1. [Required Secrets](#required-secrets)
2. [Secret Categories](#secret-categories)
3. [Setting Up Secrets](#setting-up-secrets)
4. [Secret Rotation Schedule](#secret-rotation-schedule)
5. [Security Best Practices](#security-best-practices)
6. [Emergency Procedures](#emergency-procedures)
7. [Audit and Compliance](#audit-and-compliance)

## Required Secrets

### Critical Secrets (Required)

| Secret Name | Purpose | Type | Rotation Frequency |
|------------|---------|------|-------------------|
| `DOCKER_USERNAME` | Docker Hub authentication | Username | As needed |
| `DOCKER_PASSWORD` | Docker Hub authentication | Access Token | Monthly |
| `GITHUB_TOKEN` | GitHub API access | Automatic | Managed by GitHub |

### Optional Secrets (Recommended)

| Secret Name | Purpose | Type | Rotation Frequency |
|------------|---------|------|-------------------|
| `SLACK_WEBHOOK_URL` | Build notifications | Webhook URL | Quarterly |
| `GPG_PRIVATE_KEY` | Code/Release signing | GPG Key | Yearly |
| `GPG_PASSPHRASE` | GPG key passphrase | Password | With key |
| `SNYK_TOKEN` | Security scanning | API Token | Quarterly |
| `SONAR_TOKEN` | Code quality analysis | API Token | Quarterly |

## Secret Categories

### 1. Authentication Secrets
- **Docker Hub**: Used for pushing container images
- **GitHub Container Registry**: Automatic via GITHUB_TOKEN
- **NPM Registry**: For publishing JavaScript packages (future)

### 2. Notification Secrets
- **Slack Webhooks**: For build status notifications
- **Email Configuration**: For failure alerts (future)
- **Discord/Teams**: Alternative notification channels

### 3. Security Scanning Secrets
- **Snyk**: Vulnerability scanning
- **Sonar**: Code quality and security
- **Trivy**: Container scanning (uses anonymous access)

### 4. Signing Secrets
- **GPG Keys**: For signing commits and releases
- **Cosign**: For container image signing (future)

## Setting Up Secrets

### Step 1: Access Repository Settings

1. Navigate to your GitHub repository
2. Click on **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**

### Step 2: Docker Hub Setup

```bash
# 1. Log in to Docker Hub
# https://hub.docker.com/

# 2. Go to Account Settings → Security → Access Tokens

# 3. Create a new access token with permissions:
#    - Read access to public repos
#    - Write access to private repos
#    - Delete access (optional, for cleanup)

# 4. Copy the token immediately (shown only once)

# 5. In GitHub, create secrets:
#    Name: DOCKER_USERNAME
#    Value: your-docker-username
#
#    Name: DOCKER_PASSWORD  
#    Value: your-access-token
```

### Step 3: Slack Webhook Setup (Optional)

```bash
# 1. Go to Slack App Directory
# https://api.slack.com/apps

# 2. Create a new app or use existing

# 3. Add "Incoming Webhooks" feature

# 4. Create webhook for your channel

# 5. In GitHub, create secret:
#    Name: SLACK_WEBHOOK_URL
#    Value: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

### Step 4: GPG Key Setup (Optional but Recommended)

```bash
# 1. Generate GPG key pair
gpg --full-generate-key

# 2. Export private key
gpg --armor --export-secret-keys YOUR_KEY_ID > private.key

# 3. Create a passphrase for the key

# 4. In GitHub, create secrets:
#    Name: GPG_PRIVATE_KEY
#    Value: Contents of private.key
#
#    Name: GPG_PASSPHRASE
#    Value: Your passphrase

# 5. Securely delete local private key file
shred -vfz private.key
```

### Step 5: Verify Configuration

Run the secret test workflow:

```bash
# Via GitHub UI:
# Actions → Secret Configuration Test → Run workflow

# Via GitHub CLI:
gh workflow run secret-test.yml
```

## Secret Rotation Schedule

### Monthly Rotation
- **Docker Hub Access Token**: Rotate on the 1st of each month
- Automated reminder via GitHub Issues

### Quarterly Rotation
- **API Tokens** (Snyk, Sonar): Rotate at the start of each quarter
- **Slack Webhooks**: Review and rotate if needed
- **Other Integration Tokens**: Review for active use

### Annual Rotation
- **GPG Keys**: Rotate yearly or on team changes
- **SSH Keys**: If used for deployments
- **Long-lived Credentials**: Full security review

### Event-Driven Rotation
Rotate immediately when:
- Team member leaves the project
- Security breach is suspected
- Secret is accidentally exposed
- Unusual activity is detected

## Security Best Practices

### 1. Secret Creation
- ✅ Use strong, randomly generated values
- ✅ Use access tokens instead of passwords when possible
- ✅ Limit permissions to minimum required
- ✅ Set expiration dates when available
- ❌ Never commit secrets to the repository
- ❌ Never log secret values

### 2. Secret Storage
- ✅ Use GitHub's encrypted secrets
- ✅ Document secret purposes (not values)
- ✅ Maintain secret inventory
- ❌ Don't store secrets in code or config files
- ❌ Don't share secrets via email or chat

### 3. Secret Usage
```yaml
# ✅ CORRECT: Reference secrets properly
- name: Login to Docker
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKER_USERNAME }}
    password: ${{ secrets.DOCKER_PASSWORD }}

# ❌ INCORRECT: Never echo secrets
- name: Debug
  run: echo ${{ secrets.DOCKER_PASSWORD }}  # NEVER DO THIS
```

### 4. Access Control
- Limit secret access to required workflows
- Use environment-specific secrets
- Implement least privilege principle
- Regular access reviews
- Enable audit logging

## Emergency Procedures

### Secret Exposure Response

1. **Immediate Actions** (Within 1 hour)
   ```bash
   # 1. Rotate the exposed secret immediately
   # 2. Update GitHub secret
   # 3. Audit recent usage
   
   # Check for unauthorized usage
   gh api /repos/:owner/:repo/actions/runs --paginate | \
     jq '.workflow_runs[] | select(.created_at > "2024-01-01")'
   ```

2. **Investigation** (Within 24 hours)
   - Review access logs
   - Check for unauthorized access
   - Identify exposure scope
   - Document incident

3. **Remediation** (Within 48 hours)
   - Implement additional controls
   - Update security procedures
   - Team security training
   - Post-mortem review

### Lost Access Recovery

1. **Docker Hub Recovery**
   - Use account recovery options
   - Contact Docker support if needed
   - Create new access tokens
   - Update all dependent systems

2. **GPG Key Recovery**
   - Use backup keys if available
   - Generate new signing keys
   - Update public key distribution
   - Re-sign recent releases

## Audit and Compliance

### Monthly Audit Checklist

- [ ] Review all active secrets
- [ ] Check secret last-used dates
- [ ] Verify rotation compliance
- [ ] Remove unused secrets
- [ ] Update documentation
- [ ] Review access logs

### Compliance Tracking

```yaml
# Secret Audit Record Template
date: 2025-06-12
auditor: DevOps Team
secrets_reviewed:
  - name: DOCKER_PASSWORD
    last_rotated: 2025-06-01
    next_rotation: 2025-07-01
    status: compliant
  - name: SLACK_WEBHOOK_URL
    last_rotated: 2025-04-01
    next_rotation: 2025-07-01
    status: compliant
issues_found: none
actions_taken: none
```

### Security Metrics

Track and report:
- Secret rotation compliance rate
- Average secret age
- Unauthorized access attempts
- Time to rotate after exposure
- Audit completion rate

## Tools and Scripts

### Secret Validation Script

```bash
#!/bin/bash
# validate-secrets.sh - Run locally to check secret configuration

echo "Checking required environment variables..."

required_secrets=(
  "DOCKER_USERNAME"
  "DOCKER_PASSWORD"
)

optional_secrets=(
  "SLACK_WEBHOOK_URL"
  "GPG_PRIVATE_KEY"
  "SNYK_TOKEN"
  "SONAR_TOKEN"
)

for secret in "${required_secrets[@]}"; do
  if [ -z "${!secret}" ]; then
    echo "❌ Missing required secret: $secret"
    exit 1
  else
    echo "✅ Found required secret: $secret"
  fi
done

for secret in "${optional_secrets[@]}"; do
  if [ -z "${!secret}" ]; then
    echo "⚠️  Missing optional secret: $secret"
  else
    echo "✅ Found optional secret: $secret"
  fi
done
```

### Secret Rotation Script

```bash
#!/bin/bash
# rotate-docker-token.sh - Helper script for Docker token rotation

echo "Docker Hub Token Rotation Helper"
echo "================================"
echo ""
echo "1. Log in to Docker Hub: https://hub.docker.com/"
echo "2. Go to: Account Settings → Security → Access Tokens"
echo "3. Create new token with required permissions"
echo "4. Copy the token value"
echo ""
read -p "Paste the new token here: " new_token
echo ""
echo "5. Go to: https://github.com/gunjanjp/gunj-operator/settings/secrets/actions"
echo "6. Update DOCKER_PASSWORD with the new token"
echo "7. Run the secret test workflow to verify"
echo ""
echo "Token preview: ${new_token:0:10}..."
echo ""
echo "Remember to:"
echo "- Delete the old token from Docker Hub"
echo "- Update any local .env files"
echo "- Document the rotation in the security log"
```

## Quick Reference

### GitHub CLI Commands

```bash
# List all secrets
gh secret list

# Set a secret
gh secret set SECRET_NAME

# Remove a secret
gh secret remove SECRET_NAME

# View secret details (not value)
gh api /repos/:owner/:repo/actions/secrets/SECRET_NAME
```

### Troubleshooting

| Issue | Solution |
|-------|----------|
| Workflow fails with authentication error | Check secret names and values |
| Docker push fails | Verify token permissions |
| Slack notifications not working | Test webhook URL manually |
| GPG signing fails | Check key format and passphrase |

## Contact and Support

- **Security Issues**: security@gunjanjp.com
- **DevOps Team**: devops@gunjanjp.com
- **Emergency**: Use #security-emergency Slack channel

---

*This document contains sensitive security information. Handle with appropriate care.*