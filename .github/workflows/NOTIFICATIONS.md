# CI/CD Notification Configuration

This document describes how to configure notifications for the Gunj Operator CI/CD pipeline.

## Overview

The Gunj Operator CI/CD pipeline supports multiple notification channels:
- Slack
- Discord
- Microsoft Teams
- Email (via SendGrid)
- GitHub Issues (for failures)

## Configuration

### 1. Setting up Secrets

Navigate to your GitHub repository settings → Secrets and variables → Actions, and add the following secrets:

#### Slack Notifications
- `SLACK_WEBHOOK_URL`: Webhook URL from your Slack app
  - Create a Slack app at https://api.slack.com/apps
  - Add "Incoming Webhooks" feature
  - Create a webhook for your desired channel

#### Discord Notifications
- `DISCORD_WEBHOOK_URL`: Webhook URL from your Discord server
  - Go to Server Settings → Integrations → Webhooks
  - Create a new webhook and copy the URL

#### Microsoft Teams Notifications
- `TEAMS_WEBHOOK_URL`: Webhook URL from your Teams channel
  - In Teams, go to channel → Connectors
  - Add "Incoming Webhook" connector
  - Configure and copy the webhook URL

#### Email Notifications
- `SENDGRID_API_KEY`: Your SendGrid API key
- `NOTIFICATION_EMAIL_TO`: Recipient email address
- `NOTIFICATION_EMAIL_FROM`: Sender email address (must be verified in SendGrid)

### 2. Notification Triggers

Notifications are sent for the following events:

| Event | Channels | Condition |
|-------|----------|-----------|
| Build Success | All configured channels | All jobs complete successfully |
| Build Failure | All configured channels | Any job fails |
| Critical Failure | GitHub Issue + All channels | Failure on main branch |
| Recurring Failures | GitHub Issue update | 3+ failures of same workflow |

### 3. Customizing Notifications

#### Modify Notification Content
Edit `.github/workflows/notifications.yml` to customize:
- Message format
- Fields displayed
- Colors and emojis
- Additional context

#### Add New Channels
To add a new notification channel:

1. Add webhook secret to repository
2. Add new step in `notifications.yml`
3. Update the workflow inputs/secrets

Example for adding PagerDuty:
```yaml
- name: Send PagerDuty alert
  if: ${{ secrets.PAGERDUTY_TOKEN != '' && inputs.status == 'failure' }}
  env:
    PAGERDUTY_TOKEN: ${{ secrets.PAGERDUTY_TOKEN }}
  run: |
    curl -X POST https://api.pagerduty.com/incidents \
      -H "Authorization: Token token=$PAGERDUTY_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "incident": {
          "type": "incident",
          "title": "CI Build Failed: ${{ inputs.workflow_name }}",
          "service": {
            "id": "YOUR_SERVICE_ID",
            "type": "service_reference"
          },
          "urgency": "high"
        }
      }'
```

### 4. Notification Rules

#### Filtering Notifications
You can add conditions to control when notifications are sent:

```yaml
# Only notify on main branch failures
if: ${{ failure() && github.ref == 'refs/heads/main' }}

# Only notify for specific workflows
if: ${{ contains(fromJSON('["ci", "release", "deploy"]'), github.workflow) }}

# Time-based notifications (business hours only)
if: ${{ github.event.schedule == '0 9-17 * * 1-5' }}
```

#### Escalation Rules
For critical failures, the system automatically:
1. Creates a GitHub issue
2. Escalates severity after 3 failures
3. Adds "recurring-failure" label
4. Notifies all channels

### 5. Testing Notifications

To test your notification configuration:

1. **Manual Test**: Run the notification workflow manually
   ```bash
   gh workflow run notifications.yml \
     -f status=success \
     -f workflow_name="Test Notification" \
     -f branch=main
   ```

2. **Dry Run**: Comment out actual notification sends and log instead

3. **Test Webhook**: Use webhook testing services like webhook.site

### 6. Troubleshooting

Common issues and solutions:

#### Notifications not sending
- Check secret values are correctly set
- Verify webhook URLs are valid
- Check workflow permissions

#### Incorrect formatting
- Validate JSON payloads
- Check for special characters in messages
- Verify timestamp formats

#### Rate limiting
- Implement exponential backoff
- Use queuing for high-volume notifications
- Consider batching notifications

## Best Practices

1. **Security**
   - Never commit webhook URLs or tokens
   - Rotate webhooks periodically
   - Use least-privilege access

2. **Reliability**
   - Implement retry logic
   - Handle network failures gracefully
   - Log notification attempts

3. **Usability**
   - Keep messages concise
   - Include actionable information
   - Provide direct links to issues

4. **Performance**
   - Run notifications in parallel
   - Set reasonable timeouts
   - Cache frequently used data

## Monitoring

Monitor your notification system:

1. **Success Rate**: Track successful vs failed notifications
2. **Delivery Time**: Measure notification latency
3. **Channel Health**: Monitor each channel's availability

## Support

For issues with notifications:
1. Check the workflow logs
2. Verify webhook configuration
3. Contact: gunjanjp@gmail.com
