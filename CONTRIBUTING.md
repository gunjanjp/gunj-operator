# Contributing to Gunj Operator

First off, thank you for considering contributing to Gunj Operator! It's people like you that make Gunj Operator such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by the [Gunj Operator Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to gunjanjp@gmail.com.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible.

**Before Submitting A Bug Report:**
- Check the [troubleshooting guide](docs/troubleshooting.md)
- Check the [FAQs](docs/faq.md)
- Search existing [issues](https://github.com/gunjanjp/gunj-operator/issues)

**How Do I Submit A Good Bug Report?**

Bugs are tracked as [GitHub issues](https://github.com/gunjanjp/gunj-operator/issues). Create an issue and provide the following information:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples to demonstrate the steps**
- **Describe the behavior you observed after following the steps**
- **Explain which behavior you expected to see instead and why**
- **Include screenshots and animated GIFs if possible**
- **Include logs and error messages**

### Suggesting Enhancements

Enhancement suggestions are tracked as [GitHub issues](https://github.com/gunjanjp/gunj-operator/issues).

**Before Submitting An Enhancement Suggestion:**
- Check the [roadmap](ROADMAP.md)
- Search existing [issues](https://github.com/gunjanjp/gunj-operator/issues)

**How Do I Submit A Good Enhancement Suggestion?**

- **Use a clear and descriptive title**
- **Provide a step-by-step description of the suggested enhancement**
- **Provide specific examples to demonstrate the steps**
- **Describe the current behavior and expected behavior**
- **Explain why this enhancement would be useful**
- **List some other projects where this enhancement exists**

### Your First Code Contribution

Unsure where to begin contributing? You can start by looking through these issues:

- [Good first issues](https://github.com/gunjanjp/gunj-operator/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
- [Help wanted issues](https://github.com/gunjanjp/gunj-operator/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)

### Pull Requests

Please follow these steps to have your contribution considered by the maintainers:

1. Fork the repository
2. Create a new branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes using [conventional commits](#commit-messages)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Development Process

### Setting Up Your Development Environment

See our [Developer Getting Started Guide](docs/development/getting-started.md) for detailed instructions.

### Code Style

#### Go Code Style

We follow the standard Go formatting guidelines:

```go
// Good
func (r *Reconciler) ReconcilePlatform(ctx context.Context, platform *v1beta1.ObservabilityPlatform) error {
    log := ctrl.LoggerFrom(ctx)
    
    if platform == nil {
        return fmt.Errorf("platform cannot be nil")
    }
    
    // Implementation
    return nil
}
```

See [Go Coding Standards](docs/development/coding-standards-go.md) for more details.

#### TypeScript/React Code Style

We use ESLint and Prettier for TypeScript/React code:

```typescript
// Good
export const PlatformCard: React.FC<PlatformCardProps> = ({ platform, onSelect }) => {
  const handleClick = useCallback(() => {
    onSelect?.(platform);
  }, [platform, onSelect]);

  return (
    <Card onClick={handleClick}>
      <CardContent>{platform.name}</CardContent>
    </Card>
  );
};
```

See [TypeScript/React Coding Standards](docs/development/coding-standards-typescript.md) for more details.

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

#### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that don't affect code meaning
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system
- **ci**: Changes to our CI configuration files and scripts
- **chore**: Other changes that don't modify src or test files

#### Examples

```
feat(operator): add support for Tempo tracing

- Implement Tempo component manager
- Add Tempo CRD fields
- Create Tempo deployment logic
- Add integration tests

Closes #123
```

```
fix(ui): correct platform status color

The platform status indicator was showing green for failed
platforms. This fixes the color mapping to correctly show
red for failed status.

Fixes #456
```

### Testing

All code changes must include appropriate tests:

#### Go Testing

```go
func TestReconcilePlatform(t *testing.T) {
    tests := []struct {
        name      string
        platform  *v1beta1.ObservabilityPlatform
        wantErr   bool
        wantPhase string
    }{
        {
            name: "successful reconciliation",
            platform: &v1beta1.ObservabilityPlatform{
                ObjectMeta: metav1.ObjectMeta{
                    Name: "test-platform",
                },
            },
            wantErr:   false,
            wantPhase: "Ready",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

#### React Testing

```typescript
describe('PlatformCard', () => {
  it('should render platform name', () => {
    const platform = { name: 'test-platform' };
    render(<PlatformCard platform={platform} />);
    
    expect(screen.getByText('test-platform')).toBeInTheDocument();
  });
});
```

### Documentation

- Update documentation for any user-facing changes
- Add code comments for complex logic
- Update API documentation for new endpoints
- Include examples where appropriate

## Review Process

### What We Look For

1. **Code Quality**
   - Follows coding standards
   - Well-structured and readable
   - Appropriate abstractions

2. **Testing**
   - Adequate test coverage
   - Tests are meaningful
   - Edge cases covered

3. **Performance**
   - No obvious performance issues
   - Resource usage considered
   - Scalability implications

4. **Security**
   - No security vulnerabilities
   - Follows security best practices
   - Appropriate access controls

5. **Documentation**
   - Code is well-commented
   - User documentation updated
   - API changes documented

### Review Timeline

- First response: Within 2 business days
- Review completion: Within 5 business days
- Merge decision: After 2 approvals

## Community

### Communication Channels

- **GitHub Discussions**: For questions and ideas
- **Slack**: [#gunj-operator](https://kubernetes.slack.com/archives/gunj-operator)
- **Community Calls**: Bi-weekly Thursdays at 10 AM PT

### Recognition

We value all contributions! Contributors are:
- Listed in [CONTRIBUTORS.md](CONTRIBUTORS.md)
- Eligible for project swag
- Invited to contributor events
- Given credit in release notes

## Getting Help

If you need help, you can:

1. Check the [documentation](https://gunjanjp.github.io/gunj-operator)
2. Ask in [GitHub Discussions](https://github.com/gunjanjp/gunj-operator/discussions)
3. Join our [Slack channel](https://kubernetes.slack.com/archives/gunj-operator)
4. Attend a [community call](docs/community/meetings.md)

## License

By contributing to Gunj Operator, you agree that your contributions will be licensed under its [MIT License](LICENSE).

## Acknowledgments

Thank you to all our contributors! Your efforts make this project possible.

Special thanks to:
- The Kubernetes community for the operator pattern
- The CNCF for cloud-native best practices
- All the observability tools we integrate with

---

**Happy Contributing! 🎉**
