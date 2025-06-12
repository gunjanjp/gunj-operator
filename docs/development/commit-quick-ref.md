# Git Commit Quick Reference

## Format
```
<type>(<scope>): <subject>

<body>

<footer>
```

## Types
- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation
- **style**: Formatting
- **refactor**: Code restructuring
- **perf**: Performance
- **test**: Tests
- **build**: Build system
- **ci**: CI/CD
- **chore**: Maintenance
- **revert**: Revert commit

## Scopes
**Core**: operator, api, ui, cli  
**Operator**: controller, webhook, crd, rbac  
**API**: rest, graphql, auth  
**UI**: components, pages, hooks, store  
**Infra**: docker, k8s, helm, ci  
**Other**: docs, examples, test, e2e, deps, *

## Rules
✅ **DO**
- Use present tense ("add" not "added")
- Keep subject ≤ 50 chars
- Separate subject from body with blank line
- Wrap body at 72 chars
- Reference issues ("Closes #123")

❌ **DON'T**
- Capitalize subject first letter
- End subject with period
- Use past tense
- Combine unrelated changes

## Examples
```bash
# Simple
git commit -m "feat(api): add health endpoint"

# With body
git commit -m "fix(controller): resolve memory leak

The reconciler was not releasing cached objects,
causing memory usage to grow unbounded.

Fixes #456"

# Breaking change
git commit -m "feat(crd)!: rename metrics field

BREAKING CHANGE: 'metrics' renamed to 'monitoring'"
```

## Commands
```bash
# Test a message
echo "feat(api): test" | npx commitlint

# Use template
git commit  # Opens editor with template

# Amend last commit
git commit --amend

# Interactive rebase
git rebase -i HEAD~3
```
