# API Versioning Strategy

## Overview

This document defines the API versioning strategy for the Gunj Operator, including version progression, compatibility guarantees, and migration paths.

## Version Progression Plan

### v1alpha1 (Current) - Development Phase
**Timeline**: June 2025 - December 2025  
**Status**: Unstable, rapid iteration  
**Compatibility**: No guarantees, breaking changes allowed  

**Goals**:
- Rapid prototyping
- Feature exploration
- Community feedback collection
- API shape refinement

**Features**:
- Basic CRUD operations
- Core components (Prometheus, Grafana, Loki, Tempo)
- Simple configuration
- Basic status reporting

### v1beta1 - Beta Phase
**Timeline**: January 2026 - June 2026  
**Status**: Feature complete, stabilizing  
**Compatibility**: Backward compatible within beta  

**Goals**:
- API stabilization
- Production testing
- Feature freeze
- Performance optimization

**New Features**:
- Advanced configuration options
- Multi-cluster support
- Cost optimization
- Enhanced monitoring
- Webhook validations

**Migration from v1alpha1**:
- Automated conversion
- Deprecation warnings
- Migration tools provided

### v1 - GA Phase
**Timeline**: July 2026+  
**Status**: Stable, production-ready  
**Compatibility**: Long-term support  

**Goals**:
- Production deployments
- Enterprise features
- Long-term stability
- Certification ready

**Guarantees**:
- No breaking changes
- 3-year support minimum
- Security patches
- Bug fixes only

## API Versioning Rules

### 1. Field Addition
- **v1alpha1**: Allowed anytime
- **v1beta1**: Allowed with defaults
- **v1**: Allowed with defaults, must be optional

### 2. Field Removal
- **v1alpha1**: Allowed with notice
- **v1beta1**: Deprecated first, removed in next version
- **v1**: Not allowed, only deprecation

### 3. Field Modification
- **v1alpha1**: Allowed
- **v1beta1**: Only if backward compatible
- **v1**: Not allowed

### 4. Behavior Changes
- **v1alpha1**: Allowed
- **v1beta1**: Only additions
- **v1**: Not allowed

## Conversion Strategy

### Conversion Webhook Implementation

```go
// +kubebuilder:webhook:path=/convert,mutating=false,failurePolicy=fail,groups=observability.io,resources=observabilityplatforms,verbs=create;update,versions=v1;v1beta1;v1alpha1,name=cobservabilityplatform.kb.io

type ObservabilityPlatformConverter struct{}

func (c *ObservabilityPlatformConverter) ConvertTo(src, dst runtime.Object) error {
    switch s := src.(type) {
    case *v1alpha1.ObservabilityPlatform:
        switch d := dst.(type) {
        case *v1beta1.ObservabilityPlatform:
            return convertV1alpha1ToV1beta1(s, d)
        case *v1.ObservabilityPlatform:
            // Convert through v1beta1
            intermediate := &v1beta1.ObservabilityPlatform{}
            if err := convertV1alpha1ToV1beta1(s, intermediate); err != nil {
                return err
            }
            return convertV1beta1ToV1(intermediate, d)
        }
    case *v1beta1.ObservabilityPlatform:
        switch d := dst.(type) {
        case *v1alpha1.ObservabilityPlatform:
            return convertV1beta1ToV1alpha1(s, d)
        case *v1.ObservabilityPlatform:
            return convertV1beta1ToV1(s, d)
        }
    case *v1.ObservabilityPlatform:
        // v1 is the hub version
        return fmt.Errorf("v1 is the storage version")
    }
    return fmt.Errorf("unsupported conversion")
}
```

### Storage Version Migration

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate-to-v1beta1
spec:
  template:
    spec:
      containers:
      - name: migrator
        image: gunj-operator:migration-tool
        command:
        - /migrate
        - --from-version=v1alpha1
        - --to-version=v1beta1
        - --dry-run=false
```

## Version Compatibility Matrix

| Operator Version | v1alpha1 | v1beta1 | v1 |
|-----------------|----------|---------|-----|
| v0.1.0 - v0.9.0 | ✓ | ✗ | ✗ |
| v1.0.0 - v1.9.0 | ✓ | ✓ | ✗ |
| v2.0.0+ | ✓ | ✓ | ✓ |

## Deprecation Policy

### Deprecation Timeline
1. **Announcement**: Feature marked deprecated in release notes
2. **Warning Period**: Minimum 6 months for beta, 12 months for stable
3. **Final Warning**: Last version with support clearly marked
4. **Removal**: Feature removed in major version bump

### Deprecation Markers

```go
type PrometheusSpec struct {
    // ... other fields ...
    
    // +kubebuilder:deprecatedversion:warning="This field is deprecated and will be removed in v2.0.0. Use 'resources' instead"
    // Deprecated: Use resources.requests instead
    MemoryRequest string `json:"memoryRequest,omitempty"`
}
```

### User Notification

```yaml
apiVersion: v1
kind: Event
metadata:
  name: deprecation-warning
reason: DeprecationWarning
message: "Field 'spec.prometheus.memoryRequest' is deprecated. Please use 'spec.prometheus.resources.requests.memory'"
type: Warning
```

## API Evolution Examples

### Example 1: Adding a Field

**v1alpha1**:
```go
type PrometheusSpec struct {
    Version string `json:"version"`
    Replicas int32 `json:"replicas,omitempty"`
}
```

**v1beta1**:
```go
type PrometheusSpec struct {
    Version string `json:"version"`
    Replicas int32 `json:"replicas,omitempty"`
    
    // New field with default
    // +kubebuilder:default="30d"
    Retention string `json:"retention,omitempty"`
}
```

### Example 2: Restructuring

**v1alpha1**:
```go
type ObservabilityPlatformSpec struct {
    PrometheusEnabled bool `json:"prometheusEnabled"`
    PrometheusVersion string `json:"prometheusVersion"`
}
```

**v1beta1**:
```go
type ObservabilityPlatformSpec struct {
    // Old fields deprecated
    // +kubebuilder:deprecatedversion:warning="Use components.prometheus instead"
    PrometheusEnabled bool `json:"prometheusEnabled,omitempty"`
    // +kubebuilder:deprecatedversion:warning="Use components.prometheus.version instead"
    PrometheusVersion string `json:"prometheusVersion,omitempty"`
    
    // New structure
    Components Components `json:"components"`
}
```

## Testing Strategy

### Version Testing Matrix
- Test all supported versions
- Test upgrade paths
- Test downgrade scenarios
- Test field migrations

### Automated Testing
```go
var _ = Describe("API Conversion", func() {
    It("should convert v1alpha1 to v1beta1", func() {
        v1alpha1Obj := &v1alpha1.ObservabilityPlatform{
            Spec: v1alpha1.ObservabilityPlatformSpec{
                PrometheusEnabled: true,
                PrometheusVersion: "v2.45.0",
            },
        }
        
        v1beta1Obj := &v1beta1.ObservabilityPlatform{}
        Expect(converter.Convert(v1alpha1Obj, v1beta1Obj)).To(Succeed())
        
        Expect(v1beta1Obj.Spec.Components.Prometheus).NotTo(BeNil())
        Expect(v1beta1Obj.Spec.Components.Prometheus.Enabled).To(BeTrue())
        Expect(v1beta1Obj.Spec.Components.Prometheus.Version).To(Equal("v2.45.0"))
    })
})
```

## Documentation Requirements

### For Each Version
1. API Reference (OpenAPI spec)
2. Migration Guide
3. Breaking Changes List
4. Examples for all resources
5. Conversion webhook details

### User Communication
1. Release notes highlighting changes
2. Blog posts for major versions
3. Migration webinars
4. Support during transition

## Tooling

### Version Management Tools
```bash
# Check API compatibility
gunj-operator api-check --from=v1alpha1 --to=v1beta1

# Generate migration guide
gunj-operator generate-migration --from=v1alpha1 --to=v1beta1

# Validate resources against version
gunj-operator validate --version=v1beta1 resource.yaml
```

## References
- [Kubernetes API Versioning](https://kubernetes.io/docs/reference/using-api/api-concepts/#api-versioning)
- [CRD Versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)
- [API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
