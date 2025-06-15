# Pull Request Review Guidelines

This document provides comprehensive guidelines for reviewing pull requests in the Gunj Operator project.

## Table of Contents

- [Overview](#overview)
- [Review Process](#review-process)
- [Review Criteria](#review-criteria)
- [Code Review Checklist](#code-review-checklist)
- [Review Comments](#review-comments)
- [Approval Process](#approval-process)
- [Merge Requirements](#merge-requirements)
- [Best Practices](#best-practices)

## Overview

Code reviews are a critical part of our development process. They help:
- Maintain code quality and consistency
- Share knowledge across the team
- Catch bugs before they reach production
- Improve overall system design
- Ensure compliance with project standards

### Review Responsibilities

**Authors:**
- Provide clear PR descriptions
- Keep PRs focused and reasonably sized
- Respond to feedback promptly
- Update PR based on review comments
- Ensure all checks pass

**Reviewers:**
- Review promptly (within 2 business days)
- Provide constructive feedback
- Check for compliance with standards
- Verify testing adequacy
- Consider security implications

## Review Process

### 1. Pre-Review Checks

Before starting a detailed review, verify:
- [ ] CI/CD checks are passing
- [ ] PR has a clear description
- [ ] Related issues are linked
- [ ] Commits follow conventions
- [ ] PR size is reasonable (<500 lines preferred)

### 2. Review Stages

#### Stage 1: High-Level Review (5-10 minutes)
- Understand the overall purpose
- Check if the approach makes sense
- Identify any architectural concerns
- Verify PR scope is appropriate

#### Stage 2: Detailed Code Review (20-45 minutes)
- Review code changes line by line
- Check for bugs and edge cases
- Verify coding standards compliance
- Assess test coverage
- Consider performance implications

#### Stage 3: Testing & Verification (10-20 minutes)
- Review test cases
- Check if tests cover the changes
- Verify documentation updates
- Test locally if needed

### 3. Review Timeline

- **First response**: Within 24 hours
- **Complete review**: Within 48 hours
- **Follow-up reviews**: Within 24 hours
- **Urgent fixes**: Within 4 hours

## Review Criteria

### 1. Functionality

- **Correctness**: Does the code do what it's supposed to?
- **Completeness**: Are all requirements addressed?
- **Edge Cases**: Are edge cases handled?
- **Error Handling**: Is error handling comprehensive?

### 2. Code Quality

- **Readability**: Is the code easy to understand?
- **Maintainability**: Can it be easily modified?
- **Simplicity**: Is it as simple as possible?
- **DRY**: Is there unnecessary duplication?

### 3. Design

- **Architecture**: Does it fit the overall architecture?
- **Patterns**: Are appropriate patterns used?
- **Abstractions**: Are abstractions at the right level?
- **Dependencies**: Are dependencies appropriate?

### 4. Performance

- **Efficiency**: Is the code performant?
- **Resource Usage**: Does it use resources efficiently?
- **Scalability**: Will it scale appropriately?
- **Bottlenecks**: Are there potential bottlenecks?

### 5. Security

- **Vulnerabilities**: Are there security issues?
- **Input Validation**: Is input properly validated?
- **Authentication**: Is auth/authz correct?
- **Secrets**: Are secrets handled properly?

### 6. Testing

- **Coverage**: Is test coverage adequate (>80%)?
- **Quality**: Are tests meaningful?
- **Types**: Are different test types included?
- **Edge Cases**: Do tests cover edge cases?

## Code Review Checklist

### General

- [ ] PR description is clear and complete
- [ ] Changes match the PR description
- [ ] No unrelated changes included
- [ ] Code follows project style guide
- [ ] No commented-out code
- [ ] No debug code left in
- [ ] TODOs are tracked in issues

### Go Code

- [ ] Follows Go idioms and best practices
- [ ] Error handling is consistent
- [ ] No panics in library code
- [ ] Proper use of contexts
- [ ] Resource cleanup (defer statements)
- [ ] Concurrent code is safe
- [ ] Interfaces are small and focused

### TypeScript/React Code

- [ ] Follows React best practices
- [ ] No use of `any` type
- [ ] Proper TypeScript types
- [ ] Components are properly memoized
- [ ] No memory leaks
- [ ] Accessibility considered
- [ ] Error boundaries present

### Kubernetes/Operator

- [ ] CRD changes are backward compatible
- [ ] RBAC permissions are minimal
- [ ] Resource limits are set
- [ ] Finalizers handle cleanup
- [ ] Status updates are appropriate
- [ ] Events are recorded
- [ ] Reconciliation is idempotent

### API Changes

- [ ] API changes are backward compatible
- [ ] New endpoints documented
- [ ] Input validation present
- [ ] Error responses consistent
- [ ] Rate limiting considered
- [ ] API versioning followed

### Testing

- [ ] Unit tests for new code
- [ ] Integration tests for features
- [ ] Tests are deterministic
- [ ] Tests are independent
- [ ] Mocks are appropriate
- [ ] Edge cases tested
- [ ] Error cases tested

### Documentation

- [ ] Code comments where needed
- [ ] API documentation updated
- [ ] README updated if needed
- [ ] Architecture docs updated
- [ ] Examples provided
- [ ] Changelog updated

### Security

- [ ] No hardcoded secrets
- [ ] Input validation present
- [ ] SQL injection prevented
- [ ] XSS prevention in UI
- [ ] RBAC properly configured
- [ ] Audit logging added

## Review Comments

### Comment Guidelines

#### DO:
- Be constructive and professional
- Explain the "why" behind suggestions
- Provide examples or links
- Acknowledge good code
- Use "we" instead of "you"
- Ask questions to understand

#### DON'T:
- Be dismissive or rude
- Nitpick unnecessarily
- Block on preferences
- Assume malice
- Make it personal
- Ignore the context

### Comment Types

#### 🔴 **Blocking** (Must Fix)
Issues that must be resolved before merge:
```
🔴 This will cause a nil pointer panic when the platform is not found.
Please add a nil check before accessing platform.Spec.
```

#### 🟡 **Important** (Should Fix)
Issues that should be addressed but aren't blocking:
```
🟡 Consider extracting this logic into a separate function for better testability.
This would also make the reconciliation loop more readable.
```

#### 🟢 **Suggestion** (Nice to Have)
Optional improvements or preferences:
```
🟢 nit: This could be simplified using the strings.Contains function.
```

#### ❓ **Question** (Clarification)
Questions to understand the code better:
```
❓ I'm not familiar with this pattern. Could you explain why we need
to check the cache here instead of querying the API directly?
```

#### 👍 **Praise** (Good Code)
Acknowledge good practices:
```
👍 Great error handling! This makes debugging much easier.
```

### Example Review Comments

#### Constructive Feedback
```markdown
I see what you're trying to accomplish here, but I'm concerned about the performance
impact of calling the API in a loop. 

What if we batch these requests instead? Something like:

```go
func (r *Reconciler) updateStatuses(ctx context.Context, platforms []Platform) error {
    batch := make([]StatusUpdate, 0, len(platforms))
    for _, p := range platforms {
        batch = append(batch, StatusUpdate{
            Name: p.Name,
            Status: p.Status,
        })
    }
    return r.client.BatchUpdateStatus(ctx, batch)
}
```

This would reduce the number of API calls from N to 1.
```

#### Security Concern
```markdown
🔴 Security Issue: This endpoint allows unauthenticated access to sensitive data.

Please add authentication middleware:

```go
router.GET("/api/v1/platforms/:id", 
    authMiddleware.RequireAuth(),
    authMiddleware.RequirePermission("platforms:read"),
    handler.GetPlatform,
)
```

Also, ensure we're not exposing internal fields in the response.
```

## Approval Process

### Approval Requirements

#### Standard PRs
- 2 approvals from maintainers
- All CI checks passing
- No unresolved comments
- Author has responded to all feedback

#### Trivial Changes
- 1 approval sufficient for:
  - Documentation typos
  - Comment updates
  - Dependency updates (patch versions)
  - Test additions only

#### High-Risk Changes
- 3 approvals required for:
  - API breaking changes
  - Security-related changes
  - Core operator logic changes
  - Authentication/authorization changes

### Approval Guidelines

**Before Approving:**
1. All blocking issues are resolved
2. Tests are adequate
3. Documentation is updated
4. You understand the changes
5. You would be comfortable maintaining this code

**Approval Types:**
- ✅ **Approve**: All criteria met
- 💬 **Comment**: Feedback provided, re-review needed
- ❌ **Request Changes**: Blocking issues found

## Merge Requirements

### Pre-Merge Checklist

- [ ] Required approvals obtained
- [ ] All CI checks passing
- [ ] No unresolved conversations
- [ ] Branch is up to date with main
- [ ] Commits are properly formatted
- [ ] DCO sign-off present

### Merge Methods

#### Squash and Merge (Default)
Use for:
- Feature branches with many commits
- PRs with messy commit history
- External contributions

#### Rebase and Merge
Use for:
- Clean commit history
- Each commit is meaningful
- Commits tell a story

#### Create Merge Commit
Use for:
- Large features with sub-features
- When preserving branch history is important
- Release branches

## Best Practices

### For Authors

1. **Keep PRs Small**
   - Aim for <500 lines changed
   - One logical change per PR
   - Split large features into multiple PRs

2. **Write Good Descriptions**
   - Explain what and why
   - Include screenshots for UI changes
   - List testing performed
   - Highlight areas needing attention

3. **Respond Promptly**
   - Address feedback within 24 hours
   - Ask for clarification if needed
   - Update PR based on feedback
   - Re-request review when ready

4. **Test Thoroughly**
   - Run all tests locally
   - Test edge cases
   - Verify in a real environment
   - Update tests for changes

### For Reviewers

1. **Be Timely**
   - Start reviews promptly
   - Complete reviews thoroughly
   - Don't leave PRs hanging
   - Communicate delays

2. **Be Thorough**
   - Review all changed files
   - Consider the bigger picture
   - Test locally if unsure
   - Check for common issues

3. **Be Constructive**
   - Focus on the code, not the person
   - Explain your reasoning
   - Provide alternatives
   - Acknowledge good work

4. **Be Practical**
   - Don't block on preferences
   - Consider the context
   - Balance perfection with progress
   - Pick your battles

### Review Anti-Patterns to Avoid

❌ **Nitpicking**: Focusing on trivial issues
❌ **Design Changes**: Major design feedback on implementation PRs
❌ **Scope Creep**: Requesting unrelated improvements
❌ **Perfectionism**: Blocking on non-critical issues
❌ **Drive-by Reviews**: Superficial reviews without understanding
❌ **Ghosting**: Starting a review but not completing it

## Tools and Automation

### GitHub Features

- **Suggested Changes**: Use for small fixes
- **Review Threads**: Group related comments
- **Draft Reviews**: Batch comments before submitting
- **Code Owners**: Automatic review requests

### PR Commands

```bash
# Common PR commands (in comments)
/lgtm              # Looks good to me
/approve           # Approve the PR
/hold              # Prevent automatic merge
/hold cancel       # Remove hold
/retest            # Re-run failed tests
/assign @reviewer  # Assign a reviewer
/cc @person        # CC someone for visibility
```

### Review Tools

- **GitHub CLI**: Review from terminal
- **VS Code GitHub**: Review in editor
- **Reviewable**: Enhanced review interface
- **CodeStream**: In-IDE reviews

## Summary

Effective code reviews are essential for maintaining high-quality code. By following these guidelines, we ensure:

- Consistent code quality
- Knowledge sharing
- Bug prevention
- Team collaboration
- Continuous improvement

Remember: The goal is to ship high-quality code efficiently while helping each other grow as developers.

---

For questions about the review process, contact the maintainers or discuss in #gunj-operator-dev.
