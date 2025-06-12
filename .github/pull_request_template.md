## 📋 Description

<!-- Provide a clear and concise description of the changes in this PR -->

### 🎯 Purpose
<!-- Why is this change necessary? What problem does it solve? -->

### 🔍 Changes Made
<!-- List the key changes made in this PR -->
- 
- 
- 

## 📦 Type of Change

<!-- Please delete options that are not relevant -->

- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📚 Documentation update
- [ ] ⚡ Performance improvement
- [ ] ♻️ Code refactoring
- [ ] 🔧 Configuration change
- [ ] 🧪 Test improvement
- [ ] 🎨 UI/UX improvement

## 🧪 Testing

### Test Coverage
<!-- Describe the tests that you ran to verify your changes -->

- [ ] Unit tests pass (`make test-unit`)
- [ ] Integration tests pass (`make test-integration`)
- [ ] E2E tests pass (`make test-e2e`) - if applicable
- [ ] Manual testing completed

### Test Details
<!-- Provide details about test scenarios covered -->
```
# Commands used for testing
make test
make lint
make build
```

## ✅ Checklist

### Code Quality
- [ ] My code follows the [project style guidelines](../docs/development/coding-standards.md)
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes

### Documentation
- [ ] I have updated the documentation accordingly
- [ ] I have updated the CHANGELOG.md if applicable
- [ ] I have updated API documentation if endpoints were modified
- [ ] I have updated configuration examples if needed

### Security
- [ ] I have considered security implications of my changes
- [ ] No sensitive data (passwords, tokens) are exposed in code or logs
- [ ] Input validation is implemented where necessary
- [ ] Dependencies are up to date and vulnerability-free

### Performance
- [ ] My changes do not negatively impact performance
- [ ] I have benchmarked critical code paths if applicable
- [ ] Resource usage (CPU/Memory) is within acceptable limits

## 🔗 Related Issues

<!-- Link to related issues, PRs, or discussions -->

Closes: #
Relates to: #
Blocks: #
Blocked by: #

## 📸 Screenshots/Recordings

<!-- If applicable, add screenshots or recordings to help explain your changes -->

<details>
<summary>UI Changes</summary>

<!-- Add before/after screenshots here -->

</details>

## 🚀 Deployment Notes

<!-- Any special considerations for deployment? -->

- [ ] Database migrations required
- [ ] Configuration changes required
- [ ] Breaking API changes
- [ ] Requires coordination with other services

### Rollback Plan
<!-- How to rollback if issues are discovered post-deployment -->

## 📝 Additional Notes

<!-- Any additional information that reviewers should know -->

---

**Reviewer Guidelines**: Please refer to our [PR Review Process](../docs/development/pr-review-process.md) for detailed review criteria.

<!-- 
PR Title Format: <type>(<scope>): <subject>
Examples:
- feat(operator): add prometheus configuration validation
- fix(ui): resolve dashboard loading issue
- docs(api): update REST API documentation
- chore(deps): upgrade kubernetes client to v0.28.0
-->
