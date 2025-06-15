# PR Submission Checklist

Use this checklist before submitting your PR to ensure it meets all requirements.

## Pre-Submission Checklist

### Code Quality
- [ ] Code follows the project style guide
- [ ] No linting errors (`make lint`)
- [ ] Code is properly formatted (`make fmt`)
- [ ] No commented-out code
- [ ] No debug/console.log statements
- [ ] Meaningful variable and function names
- [ ] Complex logic is documented with comments

### Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated (if applicable)
- [ ] All tests pass locally (`make test`)
- [ ] Test coverage maintained or improved
- [ ] Edge cases are tested
- [ ] Error scenarios are tested

### Documentation
- [ ] Code comments added where necessary
- [ ] API documentation updated (if applicable)
- [ ] README updated (if applicable)
- [ ] Architecture docs updated (if applicable)
- [ ] CHANGELOG entry added
- [ ] Example usage provided (for new features)

### Commits
- [ ] Commits follow conventional commit format
- [ ] Each commit is atomic and meaningful
- [ ] Commits are signed with DCO (`git commit -s`)
- [ ] No merge commits in feature branch
- [ ] Branch is rebased on latest main

### PR Details
- [ ] PR title follows conventional format
- [ ] PR description is comprehensive
- [ ] Related issues are linked
- [ ] Breaking changes are clearly marked
- [ ] Screenshots added (for UI changes)
- [ ] Testing steps are documented

### Security
- [ ] No hardcoded secrets or credentials
- [ ] Input validation implemented
- [ ] Authentication/authorization checked
- [ ] Security implications considered
- [ ] RBAC permissions are minimal

### Performance
- [ ] No obvious performance issues
- [ ] Database queries are optimized
- [ ] Caching implemented where appropriate
- [ ] Resource usage is reasonable
- [ ] No memory leaks

### Kubernetes/Operator Specific
- [ ] CRD changes are backward compatible
- [ ] Reconciliation is idempotent
- [ ] Proper error handling in controllers
- [ ] Status updates are accurate
- [ ] Events are recorded appropriately
- [ ] Resource cleanup handled (finalizers)

### API Changes
- [ ] API changes are backward compatible
- [ ] API versioning is correct
- [ ] OpenAPI spec updated
- [ ] GraphQL schema updated (if applicable)
- [ ] API documentation updated
- [ ] Client libraries updated

### Dependencies
- [ ] Dependencies are necessary
- [ ] Dependencies are from trusted sources
- [ ] License compatibility checked
- [ ] Security vulnerabilities checked
- [ ] Lock files updated

## Final Checks

- [ ] Self-review completed
- [ ] PR is focused on a single concern
- [ ] No unrelated changes included
- [ ] Branch is up to date with main
- [ ] CI/CD checks are expected to pass

## Ready to Submit?

If all items are checked, your PR is ready for submission! 🎉

Remember:
- Keep PRs small and focused
- Respond to feedback promptly
- Be open to suggestions
- Help review others' PRs too

Thank you for contributing to Gunj Operator!
