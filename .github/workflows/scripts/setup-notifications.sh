#!/bin/bash

# Notification System Setup and Test Script
# For Gunj Operator CI/CD Pipeline

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_color() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to validate webhook URL format
validate_webhook_url() {
    local url=$1
    local service=$2
    
    case $service in
        slack)
            if [[ $url =~ ^https://hooks\.slack\.com/services/.+ ]]; then
                return 0
            fi
            ;;
        discord)
            if [[ $url =~ ^https://discord\.com/api/webhooks/.+ ]] || [[ $url =~ ^https://discordapp\.com/api/webhooks/.+ ]]; then
                return 0
            fi
            ;;
        teams)
            if [[ $url =~ ^https://.*\.webhook\.office\.com/.+ ]]; then
                return 0
            fi
            ;;
        *)
            return 1
            ;;
    esac
    return 1
}

# Function to test webhook
test_webhook() {
    local service=$1
    local webhook_url=$2
    local test_message="Test notification from Gunj Operator CI/CD setup script"
    
    print_color $BLUE "Testing $service webhook..."
    
    case $service in
        slack)
            response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$webhook_url" \
                -H "Content-Type: application/json" \
                -d "{\"text\":\"$test_message\"}")
            ;;
        discord)
            response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$webhook_url" \
                -H "Content-Type: application/json" \
                -d "{\"content\":\"$test_message\"}")
            ;;
        teams)
            response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$webhook_url" \
                -H "Content-Type: application/json" \
                -d "{\"text\":\"$test_message\"}")
            ;;
    esac
    
    if [ "$response" = "200" ] || [ "$response" = "204" ]; then
        print_color $GREEN "✓ $service webhook test successful!"
        return 0
    else
        print_color $RED "✗ $service webhook test failed (HTTP $response)"
        return 1
    fi
}

# Function to setup GitHub secrets
setup_github_secret() {
    local secret_name=$1
    local secret_value=$2
    
    if command_exists gh; then
        print_color $BLUE "Setting up GitHub secret: $secret_name"
        echo "$secret_value" | gh secret set "$secret_name"
        print_color $GREEN "✓ Secret $secret_name configured"
    else
        print_color $YELLOW "GitHub CLI not found. Please set the following secret manually:"
        print_color $YELLOW "  Name: $secret_name"
        print_color $YELLOW "  Value: [hidden]"
    fi
}

# Main menu
show_menu() {
    echo
    print_color $BLUE "=== Gunj Operator CI/CD Notification Setup ==="
    echo "1. Configure Slack notifications"
    echo "2. Configure Discord notifications"
    echo "3. Configure Teams notifications"
    echo "4. Configure Email notifications"
    echo "5. Test all configured notifications"
    echo "6. Generate notification documentation"
    echo "7. Check notification system health"
    echo "8. Exit"
    echo
}

# Configure Slack
configure_slack() {
    print_color $BLUE "\n=== Configuring Slack Notifications ==="
    echo "To get a Slack webhook URL:"
    echo "1. Go to https://api.slack.com/apps"
    echo "2. Create a new app or select existing"
    echo "3. Add 'Incoming Webhooks' feature"
    echo "4. Create webhook for your channel"
    echo
    read -p "Enter Slack webhook URL (or press Enter to skip): " webhook_url
    
    if [ -n "$webhook_url" ]; then
        if validate_webhook_url "$webhook_url" "slack"; then
            if test_webhook "slack" "$webhook_url"; then
                setup_github_secret "SLACK_WEBHOOK_URL" "$webhook_url"
            fi
        else
            print_color $RED "Invalid Slack webhook URL format"
        fi
    fi
}

# Configure Discord
configure_discord() {
    print_color $BLUE "\n=== Configuring Discord Notifications ==="
    echo "To get a Discord webhook URL:"
    echo "1. Go to Server Settings → Integrations → Webhooks"
    echo "2. Create New Webhook"
    echo "3. Copy Webhook URL"
    echo
    read -p "Enter Discord webhook URL (or press Enter to skip): " webhook_url
    
    if [ -n "$webhook_url" ]; then
        if validate_webhook_url "$webhook_url" "discord"; then
            if test_webhook "discord" "$webhook_url"; then
                setup_github_secret "DISCORD_WEBHOOK_URL" "$webhook_url"
            fi
        else
            print_color $RED "Invalid Discord webhook URL format"
        fi
    fi
}

# Configure Teams
configure_teams() {
    print_color $BLUE "\n=== Configuring Microsoft Teams Notifications ==="
    echo "To get a Teams webhook URL:"
    echo "1. In Teams channel, click ... → Connectors"
    echo "2. Search and add 'Incoming Webhook'"
    echo "3. Configure and copy webhook URL"
    echo
    read -p "Enter Teams webhook URL (or press Enter to skip): " webhook_url
    
    if [ -n "$webhook_url" ]; then
        if validate_webhook_url "$webhook_url" "teams"; then
            if test_webhook "teams" "$webhook_url"; then
                setup_github_secret "TEAMS_WEBHOOK_URL" "$webhook_url"
            fi
        else
            print_color $RED "Invalid Teams webhook URL format"
        fi
    fi
}

# Configure Email
configure_email() {
    print_color $BLUE "\n=== Configuring Email Notifications ==="
    echo "Email notifications require SendGrid account"
    echo "1. Sign up at https://sendgrid.com"
    echo "2. Create API key with 'Mail Send' permission"
    echo "3. Verify sender email address"
    echo
    read -p "Enter SendGrid API Key (or press Enter to skip): " api_key
    
    if [ -n "$api_key" ]; then
        read -p "Enter FROM email address: " from_email
        read -p "Enter TO email address: " to_email
        
        if [ -n "$from_email" ] && [ -n "$to_email" ]; then
            setup_github_secret "SENDGRID_API_KEY" "$api_key"
            setup_github_secret "NOTIFICATION_EMAIL_FROM" "$from_email"
            setup_github_secret "NOTIFICATION_EMAIL_TO" "$to_email"
            print_color $GREEN "✓ Email notifications configured"
        else
            print_color $RED "Email addresses are required"
        fi
    fi
}

# Test all notifications
test_all_notifications() {
    print_color $BLUE "\n=== Testing All Configured Notifications ==="
    
    if command_exists gh; then
        print_color $BLUE "Triggering test notification workflow..."
        gh workflow run notifications.yml \
            -f status=success \
            -f workflow_name="Notification Test" \
            -f branch=main \
            -f commit_sha=$(git rev-parse HEAD 2>/dev/null || echo "test-commit")
        
        print_color $GREEN "✓ Test workflow triggered. Check your notification channels!"
    else
        print_color $YELLOW "GitHub CLI not found. Cannot trigger test workflow."
        print_color $YELLOW "Install with: https://cli.github.com/"
    fi
}

# Generate documentation
generate_docs() {
    print_color $BLUE "\n=== Generating Notification Documentation ==="
    
    local doc_file="${REPO_ROOT}/docs/notifications-setup.md"
    mkdir -p "${REPO_ROOT}/docs"
    
    cat > "$doc_file" << 'EOF'
# Notification System Setup Guide

This guide was auto-generated on $(date)

## Quick Start

1. Run the setup script:
   ```bash
   .github/workflows/scripts/setup-notifications.sh
   ```

2. Follow the interactive prompts to configure each notification channel

3. Test your configuration:
   ```bash
   gh workflow run notifications.yml -f status=success -f workflow_name="Test"
   ```

## Manual Configuration

### GitHub Secrets Required

| Secret Name | Description | Required |
|------------|-------------|----------|
| SLACK_WEBHOOK_URL | Slack incoming webhook URL | Optional |
| DISCORD_WEBHOOK_URL | Discord webhook URL | Optional |
| TEAMS_WEBHOOK_URL | Microsoft Teams webhook URL | Optional |
| SENDGRID_API_KEY | SendGrid API key for emails | Optional |
| NOTIFICATION_EMAIL_FROM | Sender email address | Optional* |
| NOTIFICATION_EMAIL_TO | Recipient email address | Optional* |

*Required if using email notifications

### Webhook URL Formats

- **Slack**: `https://hooks.slack.com/services/XXX/YYY/ZZZ`
- **Discord**: `https://discord.com/api/webhooks/XXX/YYY`
- **Teams**: `https://xxx.webhook.office.com/webhookb2/YYY`

## Troubleshooting

1. **Notifications not received**
   - Check webhook URL is correct
   - Verify secrets are set in repository
   - Check workflow logs for errors

2. **Test webhook failing**
   - Ensure webhook is active
   - Check network connectivity
   - Verify webhook permissions

3. **GitHub CLI issues**
   - Install from https://cli.github.com/
   - Authenticate with `gh auth login`
   - Ensure you have repository access

## Support

For issues, contact: gunjanjp@gmail.com
EOF

    print_color $GREEN "✓ Documentation generated at: $doc_file"
}

# Check health
check_health() {
    print_color $BLUE "\n=== Checking Notification System Health ==="
    
    local health_good=true
    
    # Check for required files
    print_color $BLUE "Checking workflow files..."
    for file in "notifications.yml" "issue-on-failure.yml" "ci.yml"; do
        if [ -f "${REPO_ROOT}/.github/workflows/$file" ]; then
            print_color $GREEN "✓ $file exists"
        else
            print_color $RED "✗ $file missing"
            health_good=false
        fi
    done
    
    # Check GitHub CLI
    print_color $BLUE "\nChecking tools..."
    if command_exists gh; then
        print_color $GREEN "✓ GitHub CLI installed"
        if gh auth status >/dev/null 2>&1; then
            print_color $GREEN "✓ GitHub CLI authenticated"
        else
            print_color $YELLOW "⚠ GitHub CLI not authenticated (run: gh auth login)"
        fi
    else
        print_color $YELLOW "⚠ GitHub CLI not installed"
    fi
    
    # Check secrets (if gh available)
    if command_exists gh && gh auth status >/dev/null 2>&1; then
        print_color $BLUE "\nChecking secrets..."
        for secret in "SLACK_WEBHOOK_URL" "DISCORD_WEBHOOK_URL" "TEAMS_WEBHOOK_URL" "SENDGRID_API_KEY"; do
            if gh secret list | grep -q "^$secret"; then
                print_color $GREEN "✓ $secret is set"
            else
                print_color $YELLOW "⚠ $secret not set (optional)"
            fi
        done
    fi
    
    if $health_good; then
        print_color $GREEN "\n✓ Notification system is healthy!"
    else
        print_color $YELLOW "\n⚠ Some issues found, but system may still work"
    fi
}

# Main loop
main() {
    print_color $GREEN "Welcome to Gunj Operator CI/CD Notification Setup!"
    
    while true; do
        show_menu
        read -p "Enter your choice (1-8): " choice
        
        case $choice in
            1) configure_slack ;;
            2) configure_discord ;;
            3) configure_teams ;;
            4) configure_email ;;
            5) test_all_notifications ;;
            6) generate_docs ;;
            7) check_health ;;
            8) 
                print_color $GREEN "Thank you for using the setup script!"
                exit 0
                ;;
            *)
                print_color $RED "Invalid choice. Please try again."
                ;;
        esac
    done
}

# Run main function
main
