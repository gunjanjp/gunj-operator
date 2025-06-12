#!/bin/bash

# Setup script for Git commit conventions
# This script configures Git to use the project's commit message template

echo "🔧 Setting up Git commit conventions for Gunj Operator..."

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Configure Git to use the commit message template
echo "📝 Configuring Git commit template..."
git config --local commit.template "${PROJECT_ROOT}/.gitmessage.txt"

# Install npm dependencies if not already installed
if [ ! -d "node_modules" ]; then
    echo "📦 Installing npm dependencies..."
    npm install
else
    echo "✅ npm dependencies already installed"
fi

# Initialize husky if not already initialized
if [ ! -d ".husky/_" ]; then
    echo "🐕 Initializing Husky..."
    npx husky install
else
    echo "✅ Husky already initialized"
fi

# Make hooks executable
echo "🔑 Making Git hooks executable..."
chmod +x .husky/commit-msg
chmod +x .husky/pre-commit

# Verify commitlint is working
echo "🧪 Testing commitlint configuration..."
echo "feat(operator): test commit message" | npx commitlint

if [ $? -eq 0 ]; then
    echo "✅ Commitlint is configured correctly!"
else
    echo "❌ Commitlint configuration error. Please check commitlint.config.js"
    exit 1
fi

echo ""
echo "✨ Git commit conventions setup complete!"
echo ""
echo "📚 Quick Reference:"
echo "  - Commit format: <type>(<scope>): <subject>"
echo "  - Example: feat(operator): add health check endpoint"
echo "  - Run 'git commit' without -m to use the template"
echo "  - See docs/development/commit-conventions.md for full guidelines"
echo ""
echo "💡 Tips:"
echo "  - Use 'npm run commitlint' to test commit messages"
echo "  - Invalid commits will be rejected by the pre-commit hook"
echo "  - For breaking changes, add 'BREAKING CHANGE:' in the footer"
echo ""
