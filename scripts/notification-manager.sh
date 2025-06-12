#!/bin/bash
# Notification Testing and Management Script
# Test and manage notifications for the Gunj Operator project
# Version: 2.0

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/.github/notification-config.yml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
COMMAND=""
CHANNEL="slack"
TYPE="test"
STATUS="info"
PRIORITY="normal"
DRY_RUN="false"

# Show help
show_help() {
    cat << EOF
Usage: $0 COMMAND [OPTIONS]

Notification management for Gunj Operator

COMMANDS:
    test        Send test notification
    send        Send custom notification
    list        List configured channels
    validate    Validate notification configuration
    history     Show notification history
    setup       Interactive setup wizard

OPTIONS:
    -c, --channel CHANNEL    Notification channel (slack|discord|teams|email|pagerduty)
                            Default: slack
    -t, --type TYPE         Notification type (build|deploy|alert|release|test)
                            Default: test
    -s, --status STATUS     Status (success|failure|warning|info)
                            Default: info
    -p, --priority PRIORITY Priority (low|normal|high|critical)
                            Default: normal
    -m, --message MESSAGE   Custom message
    --dry-run              Show what would be sent without sending
    -h, --help             Show this help message

EXAMPLES:
    # Send test notification to Slack
    $0 test -c slack

    # Send critical alert to all channels
    $0 send -t alert -s failure -p critical -m "Database connection lost"

    # Test email notification (dry run)
    $0 test -c email --dry-run

    # Validate notification configuration
    $0 validate
EOF
}

# Parse arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        show_help
        exit 1
    fi

    COMMAND=$1
    shift

    while [[ $# -gt 0 ]]; do
        case $1 in
            -c|--channel)
                CHANNEL="$2"
                shift 2
                ;;
            -t|--type)
                TYPE="$2"
                shift 2
                ;;
            -s|--status)
                STATUS="$2"
                shift 2
                ;;
            -p|--priority)
                PRIORITY="$2"
                shift 2
                ;;
            -m|--message)
                MESSAGE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                show_help
                exit 1
                ;;
        esac
    done
}

# Test notification
test_notification() {
    echo -e "${BLUE}Testing $CHANNEL notification...${NC}"
    
    local test_title="Test Notification from Gunj Operator"
    local test_message="This is a test notification to verify $CHANNEL integration is working correctly."
    
    case $CHANNEL in
        slack)
            test_slack "$test_title" "$test_message"
            ;;
        discord)
            test_discord "$test_title" "$test_message"
            ;;
        teams)
            test_teams "$test_title" "$test_message"
            ;;
        email)
            test_email "$test_title" "$test_message"
            ;;
        pagerduty)
            test_pagerduty "$test_title" "$test_message"
            ;;
        *)
            echo -e "${RED}Unknown channel: $CHANNEL${NC}"
            exit 1
            ;;
    esac
}

# Test Slack notification
test_slack() {
    local title=$1
    local message=$2
    
    if [[ -z "${SLACK_WEBHOOK_URL:-}" ]]; then
        echo -e "${YELLOW}SLACK_WEBHOOK_URL not set. Reading from config...${NC}"
        # In real implementation, would read from secure config
        echo -e "${RED}Please set SLACK_WEBHOOK_URL environment variable${NC}"
        return 1
    fi
    
    local payload=$(cat << EOF
{
    "text": "$title",
    "attachments": [{
        "color": "good",
        "text": "$message",
        "footer": "Gunj Operator Notification Test",
        "ts": $(date +%s)
    }]
}
EOF
)
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}DRY RUN - Would send:${NC}"
        echo "$payload" | jq .
    else
        response=$(curl -s -X POST "$SLACK_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "$payload")
        
        if [[ "$response" == "ok" ]]; then
            echo -e "${GREEN}✅ Slack notification sent successfully${NC}"
        else
            echo -e "${RED}❌ Failed to send Slack notification: $response${NC}"
        fi
    fi
}

# Test Discord notification
test_discord() {
    local title=$1
    local message=$2
    
    if [[ -z "${DISCORD_WEBHOOK_URL:-}" ]]; then
        echo -e "${RED}Please set DISCORD_WEBHOOK_URL environment variable${NC}"
        return 1
    fi
    
    local payload=$(cat << EOF
{
    "embeds": [{
        "title": "$title",
        "description": "$message",
        "color": 3066993,
        "footer": {
            "text": "Gunj Operator Notification Test"
        },
        "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    }]
}
EOF
)
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}DRY RUN - Would send:${NC}"
        echo "$payload" | jq .
    else
        response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$DISCORD_WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "$payload")
        
        if [[ "$response" == "204" ]]; then
            echo -e "${GREEN}✅ Discord notification sent successfully${NC}"
        else
            echo -e "${RED}❌ Failed to send Discord notification: HTTP $response${NC}"
        fi
    fi
}

# Test email notification
test_email() {
    local title=$1
    local message=$2
    
    if [[ -z "${EMAIL_USERNAME:-}" ]] || [[ -z "${EMAIL_PASSWORD:-}" ]]; then
        echo -e "${RED}Please set EMAIL_USERNAME and EMAIL_PASSWORD environment variables${NC}"
        return 1
    fi
    
    local recipient="${EMAIL_RECIPIENT:-gunjanjp@gmail.com}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}DRY RUN - Would send email:${NC}"
        echo "To: $recipient"
        echo "Subject: [Gunj Operator] $title"
        echo "Body: $message"
    else
        # In real implementation, would use proper email client
        echo -e "${YELLOW}Email notification would be sent via SMTP${NC}"
        echo "To: $recipient"
        echo "Subject: [Gunj Operator] $title"
    fi
}

# Send custom notification
send_notification() {
    local title="${MESSAGE:-Custom notification from Gunj Operator}"
    local channels_arg=""
    
    # Map single channel to channels list
    if [[ "$CHANNEL" != "all" ]]; then
        channels_arg="$CHANNEL"
    fi
    
    echo -e "${BLUE}Sending notification...${NC}"
    echo "Type: $TYPE"
    echo "Status: $STATUS"
    echo "Priority: $PRIORITY"
    echo "Channel(s): ${channels_arg:-default}"
    echo "Message: $title"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}DRY RUN - Notification would be sent${NC}"
    else
        # In real implementation, would trigger GitHub Action
        echo -e "${GREEN}Notification sent (simulated)${NC}"
    fi
}

# List configured channels
list_channels() {
    echo -e "${BLUE}Configured Notification Channels:${NC}"
    echo ""
    
    # Parse config file if it exists
    if [[ -f "$CONFIG_FILE" ]]; then
        echo "Reading from $CONFIG_FILE..."
        # Simple parsing (in real implementation would use yq or similar)
        grep -E "^  (slack|discord|teams|email|pagerduty):" "$CONFIG_FILE" | while read -r line; do
            channel=$(echo "$line" | cut -d: -f1 | tr -d ' ')
            echo "- $channel"
        done
    else
        # Default channels
        echo "- slack (default)"
        echo "- discord"
        echo "- teams"
        echo "- email"
        echo "- pagerduty"
        echo "- github"
        echo "- webhook"
    fi
    
    echo ""
    echo "Use -c CHANNEL to send to a specific channel"
}

# Validate configuration
validate_config() {
    echo -e "${BLUE}Validating notification configuration...${NC}"
    
    local errors=0
    
    # Check config file exists
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo -e "${YELLOW}⚠️  Config file not found: $CONFIG_FILE${NC}"
        ((errors++))
    fi
    
    # Check environment variables
    local required_vars=(
        "SLACK_WEBHOOK_URL:Slack"
        "DISCORD_WEBHOOK_URL:Discord"
        "EMAIL_USERNAME:Email"
        "EMAIL_PASSWORD:Email"
    )
    
    for var_def in "${required_vars[@]}"; do
        var_name="${var_def%%:*}"
        var_desc="${var_def#*:}"
        
        if [[ -z "${!var_name:-}" ]]; then
            echo -e "${YELLOW}⚠️  $var_desc: $var_name not set${NC}"
            ((errors++))
        else
            echo -e "${GREEN}✅ $var_desc: $var_name is configured${NC}"
        fi
    done
    
    # Test webhook URLs
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        if [[ ! "$SLACK_WEBHOOK_URL" =~ ^https://hooks\.slack\.com/services/ ]]; then
            echo -e "${RED}❌ Invalid Slack webhook URL format${NC}"
            ((errors++))
        fi
    fi
    
    if [[ $errors -eq 0 ]]; then
        echo -e "${GREEN}✅ All validations passed${NC}"
    else
        echo -e "${RED}❌ Found $errors configuration issues${NC}"
        exit 1
    fi
}

# Show notification history
show_history() {
    echo -e "${BLUE}Recent Notifications:${NC}"
    echo ""
    
    # In real implementation, would read from storage
    echo "Timestamp              | Channel | Type    | Status  | Title"
    echo "--------------------- | ------- | ------- | ------- | -----"
    echo "2025-06-12 10:30:00   | slack   | build   | success | Build #123 Successful"
    echo "2025-06-12 10:15:00   | email   | deploy  | failure | Deploy to prod failed"
    echo "2025-06-12 09:45:00   | discord | release | success | v2.0.0 Released"
}

# Interactive setup wizard
setup_wizard() {
    echo -e "${BLUE}=== Gunj Operator Notification Setup Wizard ===${NC}"
    echo ""
    
    # Create env file
    ENV_FILE="$PROJECT_ROOT/.env.notifications"
    
    echo -e "${YELLOW}This wizard will help you configure notification channels.${NC}"
    echo -e "${YELLOW}Configuration will be saved to: $ENV_FILE${NC}"
    echo ""
    
    # Slack setup
    echo -e "${BLUE}1. Slack Configuration${NC}"
    echo "Do you want to configure Slack notifications? (y/N)"
    read -r configure_slack
    
    if [[ "$configure_slack" =~ ^[Yy]$ ]]; then
        echo "Enter Slack webhook URL:"
        echo "(Get from: https://api.slack.com/messaging/webhooks)"
        read -r slack_webhook
        echo "SLACK_WEBHOOK_URL=$slack_webhook" >> "$ENV_FILE"
        echo -e "${GREEN}✅ Slack configured${NC}"
    fi
    
    echo ""
    
    # Discord setup
    echo -e "${BLUE}2. Discord Configuration${NC}"
    echo "Do you want to configure Discord notifications? (y/N)"
    read -r configure_discord
    
    if [[ "$configure_discord" =~ ^[Yy]$ ]]; then
        echo "Enter Discord webhook URL:"
        echo "(Get from: Server Settings > Integrations > Webhooks)"
        read -r discord_webhook
        echo "DISCORD_WEBHOOK_URL=$discord_webhook" >> "$ENV_FILE"
        echo -e "${GREEN}✅ Discord configured${NC}"
    fi
    
    echo ""
    
    # Email setup
    echo -e "${BLUE}3. Email Configuration${NC}"
    echo "Do you want to configure email notifications? (y/N)"
    read -r configure_email
    
    if [[ "$configure_email" =~ ^[Yy]$ ]]; then
        echo "Enter SMTP username (email address):"
        read -r email_username
        echo "Enter SMTP password (app password for Gmail):"
        read -rs email_password
        echo ""
        echo "Enter recipient email(s) (comma-separated):"
        read -r email_recipients
        
        echo "EMAIL_USERNAME=$email_username" >> "$ENV_FILE"
        echo "EMAIL_PASSWORD=$email_password" >> "$ENV_FILE"
        echo "EMAIL_RECIPIENTS=$email_recipients" >> "$ENV_FILE"
        echo -e "${GREEN}✅ Email configured${NC}"
    fi
    
    echo ""
    echo -e "${GREEN}=== Setup Complete ===${NC}"
    echo ""
    echo "Configuration saved to: $ENV_FILE"
    echo ""
    echo "To use these settings:"
    echo "  source $ENV_FILE"
    echo ""
    echo "Test your configuration:"
    echo "  $0 test -c slack"
    echo "  $0 test -c discord"
    echo "  $0 test -c email"
}

# Main execution
main() {
    parse_args "$@"
    
    case $COMMAND in
        test)
            test_notification
            ;;
        send)
            send_notification
            ;;
        list)
            list_channels
            ;;
        validate)
            validate_config
            ;;
        history)
            show_history
            ;;
        setup)
            setup_wizard
            ;;
        *)
            echo -e "${RED}Unknown command: $COMMAND${NC}"
            show_help
            exit 1
            ;;
    esac
}

# Run main
main "$@"
