# ADR-001: Operator Framework Selection

**Date**: 2025-06-12  
**Status**: Accepted  
**Context**: Gunj Operator Development  

## Context

We need to select an operator framework for building the Gunj Operator. The main contenders are:

1. **Kubebuilder**: Official Kubernetes SIG project
2. **Operator SDK**: Red Hat's operator framework (now part of Operator Framework)
3. **Custom controller-runtime**: Direct use of controller-runtime library
4. **Kopf**: Python-based operator framework
5. **KUDO**: Declarative operator framework

## Decision

We will use **Kubebuilder v3.14.0** as our operator framework.

## Rationale

### Kubebuilder Advantages
1. **Official Kubernetes Project**: Maintained by Kubernetes SIG API Machinery
2. **Best Practices Built-in**: Follows Kubernetes API conventions by default
3. **Code Generation**: Excellent scaffolding and code generation tools
4. **Testing Framework**: Integrated with Ginkgo/Gomega for testing
5. **CRD Management**: Superior CRD versioning and conversion webhook support
6. **Documentation**: Comprehensive documentation and tutorials
7. **Community**: Large, active community with good support

### Comparison Matrix

| Feature | Kubebuilder | Operator SDK | Custom | Kopf | KUDO |
|---------|-------------|--------------|--------|------|------|
| Language | Go | Go | Go | Python | YAML |
| Learning Curve | Medium | Medium | High | Low | Low |
| Flexibility | High | High | Highest | Medium | Low |
| Code Generation | Excellent | Good | None | None | N/A |
| Testing Support | Excellent | Good | Manual | Basic | Limited |
| Production Ready | Yes | Yes | Depends | Yes | Beta |
| CNCF Alignment | Perfect | Good | Perfect | Good | Fair |

### Why Not Others?

- **Operator SDK**: While excellent, it adds additional abstractions that we don't need
- **Custom controller-runtime**: Too much boilerplate code to maintain
- **Kopf**: Python is not ideal for high-performance operators
- **KUDO**: Too restrictive for our complex use case

## Consequences

### Positive
- Faster development with scaffolding tools
- Guaranteed Kubernetes API compliance
- Easy CRD versioning and migration
- Excellent testing infrastructure
- Strong typing with Go

### Negative
- Requires learning Kubebuilder conventions
- Some generated code might seem verbose
- Tied to Kubebuilder's project structure

### Mitigation
- Team training on Kubebuilder best practices
- Custom templates for repetitive patterns
- Clear documentation of our conventions

## References
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
