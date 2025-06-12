# Notification System Documentation

## Overview

The Gunj Operator notification system provides comprehensive alerting and communication capabilities across multiple channels. It ensures teams stay informed about build status, security issues, deployments, and releases.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ GitHub Actions  │────▶│ Notification     │────▶│ Channel         │
│ Workflows       │     │ Manager          │     │ Handlers        │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │                           │
                               ▼                           ▼
                        ┌──────────────┐          ┌──────────────┐
                        │ Rate Limiter │          │ Slack        │
                        └──────────────┘          │ Discord      │
                               │                   │ Teams        │
                               ▼                   │ Email        │
                        ┌──────────────┐          │ PagerDuty    │
                        │ Audit Logger │          └──────────────┘
                        └──────────────┘
```

## Supported Channels

### 1. Slack
- **Use Cases**: Team notifications, build status, alerts
- **Features**: Rich formatting, mentions, threads
- **Configuration**: Webhook URL required

### 2. Discord
- **Use Cases**: Community updates, releases
- **Features**: Embeds, roles mentions
- **Configuration**: Webhook URL required

### 3. Microsoft Teams
- **Use Cases**: Enterprise notifications
- **Features**: Adaptive cards, actions
- **Configuration**: Incoming webhook

### 4. Email
- **Use Cases**: Critical alerts, reports
- **Features**: HTML formatting, attachments
- **Configuration**: SMTP settings

### 5. PagerDuty
- **Use Cases**: Critical incidents, on-call
- **Features**: Escalation policies, scheduling
- **Configuration**: API token, service ID

### 6. GitHub
- **Use Cases**: PR status, issues
- **Features**: Commit status, comments
- **Configuration**: Automatic with token

## Configuration

### Setting Up Notifications

1. **Configure Secrets**
   ```bash
   # Run the setup wizard
   ./scripts/notification-manager.sh setup
   
   # Or manually set environment variables
   export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
   export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
   export EMAIL_USERNAME="notifications@example.com"
   export EMAIL_PASSWORD="app-specific-password"
   ```

2. **GitHub Actions Secrets**
   
   Navigate to Settings → Secrets and variables → Actions:
   
   | Secret Name | Description | Required |
   |------------|-------------|----------|
   | SLACK_WEBHOOK_URL | Slack incoming webhook | Yes |
   | DISCORD_WEBHOOK_URL | Discord webhook | No |
   | TEAMS_WEBHOOK_URL | Teams incoming webhook | No |
   | EMAIL_USERNAME | SMTP username | No |
   | EMAIL_PASSWORD | SMTP password | No |
   | PAGERDUTY_TOKEN | PagerDuty API token | No |
   | PAGERDUTY_SERVICE_ID | PagerDuty service ID | No |

3. **Channel Configuration**
   
   Edit `.github/notification-config.yml`:
   ```yaml
   channels:
     slack:
       enabled: true
       default_channel: "#gunj-operator"
       channels:
         builds: "#gunj-operator-builds"
         alerts: "#gunj-operator-alerts"
   ```

## Notification Types

### Build Notifications
- Triggered on build completion
- Includes success/failure status
- Links to logs and artifacts

### Security Notifications
- CVE alerts
- Dependency vulnerabilities
- Security scan results

### Deployment Notifications
- Environment deployments
- Rollback alerts
- Health status updates

### Release Notifications
- New version announcements
- Changelog summaries
- Download links

## Usage

### Sending Notifications via Workflow

```yaml
jobs:
  notify:
    uses: ./.github/workflows/notification-manager.yml
    with:
      notification_type: build
      status: failure
      title: "Build Failed"
      message: "The build for branch main has failed"
      priority: high
      channels: "slack,email"
```

### Manual Notifications

```bash
# Send test notification
./scripts/notification-manager.sh test -c slack

# Send custom notification
./scripts/notification-manager.sh send \
  -t alert \
  -s failure \
  -p critical \
  -m "Database connection lost"

# Validate configuration
./scripts/notification-manager.sh validate
```

## Notification Rules

Rules are defined in `.github/notification-config.yml`:

```yaml
rules:
  - name: build-failure-main
    conditions:
      - type: build
      - status: failure
      - branch: main
    channels:
      - slack:alerts
      - pagerduty
    priority: critical
```

### Rule Conditions
- `type`: Notification type (build, deploy, etc.)
- `status`: Event status (success, failure, warning)
- `branch`: Git branch pattern
- `environment`: Deployment environment
- `severity`: Security severity level

### Rule Actions
- `channels`: Where to send notifications
- `priority`: Notification priority
- `template`: Message template to use
- `escalate`: Escalation policy

## Templates

Message templates provide consistent formatting:

```yaml
templates:
  build_failure:
    title: "❌ Build Failed"
    message: |
      Build failed for ${GITHUB_REPOSITORY}
      
      Branch: ${GITHUB_REF_NAME}
      Commit: ${GITHUB_SHA:0:7}
      Error: ${ERROR_MESSAGE}
    color: danger
```

### Available Variables
- `${GITHUB_*}`: GitHub context variables
- `${BUILD_*}`: Build-specific variables
- `${ERROR_*}`: Error details
- `${CUSTOM_*}`: Custom metadata

## Rate Limiting

Prevents notification spam:

```yaml
rate_limiting:
  global:
    window: 3600  # 1 hour
    max_notifications: 100
  per_channel:
    slack:
      window: 300  # 5 minutes
      max_notifications: 20
```

## Monitoring

### Metrics
- `gunj_notifications_sent_total`: Total notifications sent
- `gunj_notifications_failed_total`: Failed notifications
- `gunj_notification_duration_seconds`: Send duration
- `gunj_notification_queue_size`: Queue backlog

### Grafana Dashboard
Import the dashboard from `config/monitoring/notification-dashboard.yaml`

### Alerts
```yaml
- alert: HighNotificationFailureRate
  expr: rate(gunj_notifications_failed_total[5m]) > 0.1
  annotations:
    summary: "High notification failure rate"
```

## Troubleshooting

### Common Issues

1. **Notifications not sending**
   - Check webhook URLs are correct
   - Verify secrets are set
   - Check rate limits
   - Review workflow logs

2. **Slack notifications failing**
   ```bash
   # Test Slack webhook
   curl -X POST $SLACK_WEBHOOK_URL \
     -H 'Content-Type: application/json' \
     -d '{"text":"Test message"}'
   ```

3. **Email notifications failing**
   - Verify SMTP settings
   - Check app-specific password (Gmail)
   - Test connection:
   ```bash
   openssl s_client -connect smtp.gmail.com:587 -starttls smtp
   ```

4. **Rate limiting active**
   - Check notification frequency
   - Adjust rate limits if needed
   - Enable aggregation

### Debug Mode

Enable debug logging:
```yaml
- uses: ./.github/workflows/notification-manager.yml
  with:
    debug: true
```

## Best Practices

1. **Channel Selection**
   - Use appropriate channels for each type
   - Critical alerts → PagerDuty
   - Team updates → Slack
   - Community → Discord

2. **Message Content**
   - Keep messages concise
   - Include actionable information
   - Add relevant links
   - Use consistent formatting

3. **Priority Levels**
   - `low`: Informational only
   - `normal`: Standard notifications
   - `high`: Requires attention
   - `critical`: Immediate action needed

4. **Avoid Noise**
   - Don't notify on every commit
   - Aggregate similar notifications
   - Use smart routing rules
   - Implement quiet hours

## Advanced Features

### Custom Webhooks
```yaml
webhook:
  url: "${CUSTOM_WEBHOOK_URL}"
  headers:
    X-Custom-Header: "value"
  payload_template: |
    {
      "event": "{{ .Type }}",
      "status": "{{ .Status }}",
      "message": "{{ .Message }}"
    }
```

### Escalation Policies
```yaml
escalation:
  policies:
    critical:
      levels:
        - wait: 0
          channels: [slack, email]
        - wait: 300  # 5 minutes
          channels: [pagerduty]
        - wait: 900  # 15 minutes
          channels: [phone]
```

### Notification Aggregation
```yaml
aggregation:
  enabled: true
  window: 900  # 15 minutes
  max_aggregated: 10
  group_by: [type, component]
```

## Security Considerations

1. **Webhook Security**
   - Use HTTPS webhooks only
   - Rotate webhook URLs regularly
   - Store as encrypted secrets
   - Validate webhook signatures

2. **Access Control**
   - Limit who can trigger notifications
   - Use repository secrets
   - Audit notification usage
   - Monitor for abuse

3. **Content Security**
   - Sanitize user input
   - Avoid exposing secrets
   - Limit sensitive information
   - Use secure templates

## Integration Examples

### ArgoCD Integration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
data:
  service.webhook.gunj: |
    url: https://api.gunj-operator.io/webhooks/argocd
    headers:
    - name: X-Auth-Token
      value: $webhook-token
```

### Prometheus AlertManager
```yaml
receivers:
  - name: gunj-operator
    webhook_configs:
      - url: https://api.gunj-operator.io/webhooks/alertmanager
        send_resolved: true
```

## Roadmap

- [ ] SMS notifications support
- [ ] Jira integration
- [ ] ServiceNow integration
- [ ] Voice call alerts
- [ ] Mobile push notifications
- [ ] Notification analytics dashboard
- [ ] AI-powered alert grouping
- [ ] Customizable notification sounds
