# PR Review Configuration

This document outlines the PR review settings and branch protection rules for the Gunj Operator project.

## Branch Protection Rules

### `main` branch

**Protection Settings:**
- ✅ Require pull request reviews before merging
  - Required approving reviews: **2**
  - Dismiss stale pull request approvals when new commits are pushed
  - Require review from CODEOWNERS
  - Restrict who can dismiss pull request reviews: `maintainer-team`
- ✅ Require status checks to pass before merging
  - Required status checks:
    - `lint`
    - `test`
    - `build`
    - `security-scan`
    - `commitlint`
    - `dco-check`
  - Require branches to be up to date before merging
- ✅ Require conversation resolution before merging
- ✅ Require signed commits
- ✅ Require linear history
- ✅ Include administrators
- ✅ Restrict who can push to matching branches: `maintainer-team`
- ❌ Allow force pushes (disabled)
- ❌ Allow deletions (disabled)

### `develop` branch

**Protection Settings:**
- ✅ Require pull request reviews before merging
  - Required approving reviews: **1**
  - Require review from CODEOWNERS
- ✅ Require status checks to pass before merging
  - Required status checks:
    - `lint`
    - `test`
    - `build`
- ✅ Require branches to be up to date before merging
- ✅ Require signed commits
- ✅ Include administrators

### `release/*` branches

**Protection Settings:**
- ✅ Require pull request reviews before merging
  - Required approving reviews: **3**
  - Require review from CODEOWNERS
  - Restrict who can dismiss pull request reviews: `release-team`
- ✅ Require status checks to pass before merging
  - All status checks required
- ✅ Require conversation resolution before merging
- ✅ Require signed commits
- ✅ Restrict who can push to matching branches: `release-team`
- ❌ Allow force pushes (disabled)
- ❌ Allow deletions (disabled)

## Auto-merge Settings

### Allowed for:
- Dependabot PRs (patch and minor updates only)
- Documentation-only PRs
- PRs with `auto-merge` label and 2+ approvals

### Requirements:
- All status checks passing
- No changes requested
- No merge conflicts
- PR is up to date with base branch

## Review Assignment

### Automatic Assignment Rules:
1. **Round-robin assignment** among team members
2. **Load balancing** - max 5 active reviews per person
3. **Expertise-based** - based on CODEOWNERS

### Teams:
- `operator-team`: 3-5 members
- `api-team`: 2-3 members
- `frontend-team`: 2-3 members
- `docs-team`: 2 members
- `security-team`: 2 members
- `devops-team`: 2 members

## PR Labels

### Size Labels (auto-applied):
- `size/XS`: < 10 lines
- `size/S`: 10-99 lines
- `size/M`: 100-499 lines
- `size/L`: 500-999 lines
- `size/XL`: 1000+ lines

### Priority Labels:
- `priority/critical`: Security fixes, major bugs
- `priority/high`: Important features, significant bugs
- `priority/medium`: Standard features and fixes
- `priority/low`: Nice-to-have improvements

### Status Labels:
- `ready-for-review`: PR is ready for review
- `work-in-progress`: PR is still being worked on
- `needs-changes`: Changes requested by reviewers
- `ready-to-merge`: All checks passed, awaiting merge
- `blocked`: Blocked by external factors

### Type Labels:
- `breaking-change`: Contains breaking changes
- `security`: Security-related changes
- `performance`: Performance improvements
- `documentation`: Documentation updates
- `dependencies`: Dependency updates

### Review Labels:
- `needs-review`: Awaiting initial review
- `needs-second-review`: One approval, needs another
- `lgtm`: Looks good to me (informal approval)
- `approved`: Formally approved
- `do-not-merge/hold`: Prevent automatic merging

## Merge Queue

### Configuration:
- **Maximum PRs in queue**: 5
- **Merge method**: Squash and merge (default)
- **Wait time**: 5 minutes after approval
- **Batch size**: 1 (no batching)

### Priority Order:
1. `priority/critical` PRs
2. `priority/high` PRs
3. Security fixes
4. Bug fixes
5. Features
6. Other changes

## SLA (Service Level Agreement)

### Review Times:
- **First response**: Within 24 hours
- **Initial review**: Within 48 hours
- **Subsequent reviews**: Within 24 hours
- **Critical fixes**: Within 4 hours

### Escalation:
- After 48 hours: Notify team lead
- After 72 hours: Notify project maintainer
- After 1 week: Escalate to steering committee

## Stale PR Policy

### Timeframes:
- **Warning**: After 7 days of inactivity
- **Stale label**: After 14 days of inactivity
- **Close warning**: After 28 days of inactivity
- **Auto-close**: After 30 days of inactivity

### Exemptions:
- PRs with `do-not-close` label
- PRs assigned to milestone
- PRs with active discussion

## Review Metrics

### Tracked Metrics:
- Average time to first review
- Average time to merge
- Number of review iterations
- Review workload distribution
- PR success rate

### Goals:
- First review: < 24 hours (90% of PRs)
- Time to merge: < 72 hours (80% of PRs)
- Review iterations: < 3 (85% of PRs)

## GitHub Settings Commands

```bash
# Configure branch protection (requires admin access)
gh api repos/gunjanjp/gunj-operator/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["lint","test","build","security-scan","commitlint","dco-check"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":2,"dismiss_stale_reviews":true,"require_code_owner_reviews":true}' \
  --field restrictions='{"users":[],"teams":["maintainer-team"]}' \
  --field allow_force_pushes=false \
  --field allow_deletions=false \
  --field required_conversation_resolution=true \
  --field required_linear_history=true \
  --field required_signatures=true

# Configure PR review settings
gh api repos/gunjanjp/gunj-operator \
  --method PATCH \
  --field allow_auto_merge=true \
  --field allow_merge_commit=true \
  --field allow_squash_merge=true \
  --field allow_rebase_merge=true \
  --field delete_branch_on_merge=true \
  --field allow_update_branch=true
```

---

For questions about PR review configuration, contact @gunjanjp or the maintainer team.
