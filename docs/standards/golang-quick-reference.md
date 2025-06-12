# Golang Standards Quick Reference - Gunj Operator

## 🚀 Quick Setup

```bash
# Install required tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Format code
make fmt

# Run linters
make lint

# Run tests
make test
```

## 📋 Key Standards

### Import Order
```go
import (
    // 1. Standard library
    "context"
    "fmt"
    
    // 2. Third-party
    "k8s.io/apimachinery/pkg/api/errors"
    
    // 3. Internal
    "github.com/gunjanjp/gunj-operator/api/v1beta1"
)
```

### Naming
- Variables: `camelCase`
- Exported: `PascalCase`
- Constants: `PascalCase` (not ALL_CAPS)
- Interfaces: End with `-er` when possible

### Error Handling
```go
// Always wrap errors with context
if err := r.Create(ctx, obj); err != nil {
    return fmt.Errorf("creating object: %w", err)
}
```

### Testing
```go
// Use table-driven tests
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    // test cases
}
```

### Comments
- Package: Document the package purpose
- Exported: Start with the name being documented
- Complete sentences with punctuation

### Common Commands
```bash
# Format
gofmt -w .
goimports -w .

# Lint
golangci-lint run

# Test with coverage
go test -v -race -coverprofile=coverage.out ./...

# Security scan
gosec ./...
```

## 🔍 Quick Checks

Before committing:
- [ ] Code formatted with `gofmt`
- [ ] Imports organized with `goimports`
- [ ] All tests pass
- [ ] No linter warnings
- [ ] Exported items documented
- [ ] Errors wrapped with context
- [ ] No sensitive data in logs

## 📚 Full Documentation

See [golang-coding-standards.md](./golang-coding-standards.md) for complete guidelines.
