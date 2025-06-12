# PR Review Checklist

## 🚀 Quick Review Checklist

Use this checklist for quick PR reviews. For detailed guidelines, see [PR Review Process](./pr-review-process.md).

### 🏗️ General
- [ ] PR has descriptive title following conventional commits
- [ ] PR description is complete and clear
- [ ] Related issues are linked
- [ ] Appropriate labels applied
- [ ] Target branch is correct

### 💻 Code Quality
- [ ] Code follows project style guide
- [ ] No unnecessary complexity
- [ ] Functions are focused and single-purpose
- [ ] Variable/function names are clear
- [ ] No code duplication
- [ ] Comments explain "why" not "what"

### 🧪 Testing
- [ ] All new code has tests
- [ ] Tests cover happy path and edge cases
- [ ] Tests are independent and repeatable
- [ ] Test names clearly describe what they test
- [ ] No tests are skipped without explanation
- [ ] Coverage meets requirements (≥80%)

### 🔒 Security
- [ ] No hardcoded secrets or credentials
- [ ] Input validation is present
- [ ] Authentication/authorization checks in place
- [ ] No SQL injection vulnerabilities
- [ ] Dependencies are secure
- [ ] Error messages don't leak sensitive info

### ⚡ Performance
- [ ] No obvious performance bottlenecks
- [ ] Database queries are optimized
- [ ] No memory leaks
- [ ] Appropriate caching is used
- [ ] Resource cleanup is handled
- [ ] Concurrent code is thread-safe

### 📚 Documentation
- [ ] Code is self-documenting where possible
- [ ] Complex logic has explanatory comments
- [ ] Public APIs are documented
- [ ] README updated if needed
- [ ] CHANGELOG entry added
- [ ] User documentation updated

### 🔄 Kubernetes Specific
- [ ] CRDs follow Kubernetes conventions
- [ ] Resource limits and requests defined
- [ ] RBAC permissions are minimal
- [ ] Finalizers clean up resources
- [ ] Status updates are idempotent
- [ ] Events are recorded appropriately

### 🌐 API Specific
- [ ] Endpoints follow REST conventions
- [ ] Error responses are consistent
- [ ] API versioning is maintained
- [ ] OpenAPI spec is updated
- [ ] Rate limiting considered
- [ ] CORS headers appropriate

### 🎨 UI Specific
- [ ] Components are accessible (ARIA)
- [ ] Mobile responsive
- [ ] Cross-browser tested
- [ ] Loading states present
- [ ] Error states handled
- [ ] Internationalization supported

### ✅ Final Checks
- [ ] CI/CD passes all checks
- [ ] No merge conflicts
- [ ] Dependencies are up to date
- [ ] Breaking changes are documented
- [ ] Deployment notes included
- [ ] Rollback plan documented

## 🏷️ Quick Labels Reference

Apply these labels as appropriate:

- `bug` - Bug fixes
- `feature` - New features
- `enhancement` - Improvements
- `documentation` - Doc updates
- `security` - Security related
- `performance` - Performance improvements
- `breaking-change` - Breaking changes
- `needs-discussion` - Requires team discussion
- `blocked` - Blocked by dependency
- `ready-to-merge` - All checks passed

## 💬 Review Comment Templates

### Requesting Changes
```markdown
[MUST] This needs to be addressed before merge:
- Issue: [describe the problem]
- Suggestion: [how to fix it]
- Example: [code example if helpful]
```

### Suggesting Improvements
```markdown
[CONSIDER] Non-blocking suggestion:
This works, but consider [alternative approach] for [benefit].
See [reference/example] for more details.
```

### Asking Questions
```markdown
[QUESTION] Could you clarify:
- Why was this approach chosen over [alternative]?
- What happens when [edge case]?
- How does this interact with [component]?
```

### Giving Praise
```markdown
[PRAISE] Great work on:
- [Specific thing that was done well]
- This is a pattern we should use elsewhere!
```

---

**Remember**: Reviews are about improving code quality and sharing knowledge, not finding fault. Be kind, be helpful, be constructive! 🤝
