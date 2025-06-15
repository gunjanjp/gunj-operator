#!/bin/bash
# Commit helper script for Gunj Operator

echo "Gunj Operator Commit Helper"
echo "=========================="
echo ""
echo "Commit Types:"
echo "  feat     - A new feature"
echo "  fix      - A bug fix"
echo "  docs     - Documentation only changes"
echo "  style    - Code style changes (formatting, etc)"
echo "  refactor - Code refactoring"
echo "  perf     - Performance improvements"
echo "  test     - Adding or updating tests"
echo "  build    - Build system or dependency changes"
echo "  ci       - CI/CD changes"
echo "  chore    - Other changes"
echo "  revert   - Revert a previous commit"
echo ""
echo "Scopes:"
echo "  operator, api, ui, controllers, crd, webhooks, helm, docs, deps, security"
echo ""

read -p "Type: " TYPE
read -p "Scope (optional): " SCOPE
read -p "Subject (imperative mood, no capital, no period): " SUBJECT

if [ -n "$SCOPE" ]; then
    HEADER="$TYPE($SCOPE): $SUBJECT"
else
    HEADER="$TYPE: $SUBJECT"
fi

echo ""
echo "Enter commit body (press Ctrl+D when done):"
BODY=$(cat)

echo ""
read -p "Breaking change? (y/N): " BREAKING
if [ "$BREAKING" = "y" ] || [ "$BREAKING" = "Y" ]; then
    read -p "Describe breaking change: " BREAKING_DESC
    FOOTER="BREAKING CHANGE: $BREAKING_DESC"
fi

read -p "Closes issue? (enter issue number or press Enter to skip): " ISSUE
if [ -n "$ISSUE" ]; then
    if [ -n "$FOOTER" ]; then
        FOOTER="$FOOTER\nCloses #$ISSUE"
    else
        FOOTER="Closes #$ISSUE"
    fi
fi

# Get git user info for sign-off
GIT_NAME=$(git config user.name)
GIT_EMAIL=$(git config user.email)
SIGNOFF="Signed-off-by: $GIT_NAME <$GIT_EMAIL>"

# Construct the commit message
COMMIT_MSG="$HEADER"
if [ -n "$BODY" ]; then
    COMMIT_MSG="$COMMIT_MSG\n\n$BODY"
fi
if [ -n "$FOOTER" ]; then
    COMMIT_MSG="$COMMIT_MSG\n\n$FOOTER"
fi
COMMIT_MSG="$COMMIT_MSG\n\n$SIGNOFF"

# Show the commit message
echo ""
echo "=== Commit Message Preview ==="
echo -e "$COMMIT_MSG"
echo "=============================="
echo ""

read -p "Commit with this message? (Y/n): " CONFIRM
if [ "$CONFIRM" != "n" ] && [ "$CONFIRM" != "N" ]; then
    echo -e "$COMMIT_MSG" | git commit -F -
    echo "Commit successful!"
else
    echo "Commit cancelled."
fi
