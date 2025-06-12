# Quick Setup: Notification System

Follow these steps to set up notifications for your Gunj Operator deployment.

## 🚀 5-Minute Setup

### Step 1: Choose Your Channels

Decide which notification channels you want to use:
- **Slack** - Best for team collaboration
- **Discord** - Great for community updates
- **Email** - Essential for critical alerts
- **PagerDuty** - Required for on-call rotation

### Step 2: Get Webhook URLs

#### Slack
1. Go to https://api.slack.com/apps
2. Create a new app or select existing
3. Add "Incoming Webhooks" feature
4. Create webhook for your channel
5. Copy the webhook URL

#### Discord
1. Open Discord server settings
2. Go to Integrations → Webhooks
3. Create new webhook
4. Copy the webhook URL

#### Microsoft Teams
1. Go to your Teams channel
2. Click ••• → Connectors
3. Add "Incoming Webhook"
4. Configure and copy URL

### Step 3: Configure GitHub Secrets

1. Go to your repository on GitHub
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Add these secrets:

| Secret Name | Your Value |
|-------------|------------|
| `SLACK_WEBHOOK_URL` | `https://hooks.slack.com/services/...` |
| `DISCORD_WEBHOOK_URL` | `https://discord.com/api/webhooks/...` |
| `EMAIL_USERNAME` | `notifications@yourcompany.com` |
| `EMAIL_PASSWORD` | `app-specific-password` |

### Step 4: Test Notifications

Run this workflow to test:

```yaml
name: Test Notifications
on: workflow_dispatch

jobs:
  test:
    uses: ./.github/workflows/notification-manager.yml
    with:
      notification_type: test
      status: info
      title: "Test Notification"
      message: "Testing Gunj Operator notifications!"
      channels: "slack,discord,email"
    secrets: inherit
```

### Step 5: Enable Automated Notifications

The system will automatically send notifications for:
- ✅ Build completions
- 🔒 Security alerts
- 🚀 Deployments
- 🎉 Releases

## 📊 Verify Setup

Check your notification status:
1. Open GitHub Actions tab
2. Look for "Notification Manager" runs
3. Check the summary for delivery status

## 🎯 Next Steps

- **Customize Rules**: Edit `.github/notification-config.yml`
- **Add Channels**: Configure additional notification channels
- **Set Priorities**: Define which events trigger which channels
- **Monitor Usage**: Import Grafana dashboard for metrics

## 🆘 Troubleshooting

**Notifications not working?**
```bash
# Download and run the test script
curl -O https://raw.githubusercontent.com/gunjanjp/gunj-operator/main/scripts/notification-manager.sh
chmod +x notification-manager.sh
./notification-manager.sh validate
```

**Need help?**
- 📚 [Full Documentation](notifications.md)
- 💬 [Community Discord](https://discord.gg/gunj-operator)
- 📧 Email: gunjanjp@gmail.com

---

🎉 **That's it!** Your notifications are now configured. Happy building!
