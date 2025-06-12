# Architecture Decision Record: Golang Framework Selection

**Date**: June 12, 2025  
**Status**: Accepted  
**Deciders**: Gunjan JP, Development Team  

## Context

The Gunj Operator requires a Golang framework for building a Kubernetes operator that manages complex observability components (Prometheus, Grafana, Loki, Tempo). We need to choose between Kubebuilder and Operator SDK.

## Decision

We will use **Kubebuilder v3.14.0** as our operator framework.

## Comparison Matrix

| Criteria | Kubebuilder | Operator SDK | Winner |
|----------|-------------|--------------|---------|
| **Code Structure** | Clean, idiomatic Go | More opinionated | Kubebuilder ✅ |
| **Learning Curve** | Steeper | Moderate | Operator SDK |
| **Webhook Support** | Excellent native support | Good support | Kubebuilder ✅ |
| **CRD Versioning** | Superior conversion support | Basic support | Kubebuilder ✅ |
| **Testing Framework** | Envtest (excellent) | Scorecard + Envtest | Kubebuilder ✅ |
| **Performance** | Optimal (pure Go) | Good (depends on type) | Kubebuilder ✅ |
| **OLM Integration** | Manual setup | Built-in | Operator SDK |
| **Multi-language Support** | Go only | Go, Ansible, Helm | Operator SDK |
| **CNCF Alignment** | Strong (k8s-sigs) | Good | Kubebuilder ✅ |
| **Community & Docs** | Excellent | Excellent | Tie |

## Rationale

### Why Kubebuilder?

1. **Pure Go Implementation**
   - No overhead from Ansible/Helm operators
   - Better performance for managing resource-intensive components
   - Direct control over reconciliation logic

2. **Superior CRD Management**
   - Advanced conversion webhook support
   - Multi-version API handling
   - Better schema validation

3. **CNCF Alignment**
   - Part of kubernetes-sigs
   - Follows Kubernetes patterns closely
   - Better for CNCF certification

4. **Testing Excellence**
   - Comprehensive envtest framework
   - Better integration testing support
   - Easier to achieve 85%+ coverage

5. **Clean Architecture**
   - Less opinionated structure
   - More flexibility for complex logic
   - Better separation of concerns

### Trade-offs

1. **OLM Integration**: We'll need to manually create OLM bundles
2. **Learning Curve**: Steeper initial learning curve
3. **No Ansible/Helm**: Can't reuse existing Helm charts directly

## Implementation Details

### Project Structure
```
gunj-operator/
├── api/v1beta1/          # CRD definitions
├── controllers/          # Reconciliation logic
├── config/              # Kustomize manifests
│   ├── crd/            # CRD bases
│   ├── manager/        # Controller manager
│   ├── rbac/           # RBAC definitions
│   └── webhook/        # Webhook configs
├── hack/               # Scripts
└── main.go            # Entry point
```

### Key Dependencies
```go
// go.mod
module github.com/gunjanjp/gunj-operator

go 1.21

require (
    k8s.io/apimachinery v0.29.0
    k8s.io/client-go v0.29.0
    sigs.k8s.io/controller-runtime v0.17.0
)
```

### Scaffolding Commands
```bash
# Initialize project
kubebuilder init --domain observability.io --repo github.com/gunjanjp/gunj-operator

# Create API
kubebuilder create api --group observability --version v1beta1 --kind ObservabilityPlatform

# Add webhook
kubebuilder create webhook --group observability --version v1beta1 --kind ObservabilityPlatform --defaulting --validation --conversion
```

## Consequences

### Positive
- Clean, maintainable Go code
- Excellent webhook and conversion support
- Strong testing capabilities
- CNCF-aligned patterns
- Better performance

### Negative
- Steeper learning curve initially
- Manual OLM bundle creation
- No direct Helm chart reuse

### Mitigation
- Comprehensive documentation
- Team training on Kubebuilder
- Create OLM generation tooling
- Build Helm chart import utilities

## References
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [CNCF Operator Best Practices](https://github.com/cncf/tag-app-delivery/blob/main/operator-whitepaper/v1/operator-whitepaper.md)
