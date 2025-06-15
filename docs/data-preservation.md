# Data Preservation in API Conversions

## Overview

The Gunj Operator implements comprehensive data preservation mechanisms during API conversions to ensure no data is lost when converting between different API versions (e.g., v1alpha1 to v1beta1). This document describes how data preservation works and provides guidelines for using it.

## Key Features

### 1. Status Field Preservation
The conversion process automatically preserves all status fields during conversion:
- Component statuses (Ready, Failed, etc.)
- Last reconciliation time
- Observed generation
- Custom status fields

### 2. Annotation and Label Preservation
User-defined annotations and labels are preserved, while system annotations are handled separately:
- User annotations are maintained across conversions
- System annotations (kubernetes.io/*, kubectl.kubernetes.io/*) are filtered
- Conversion metadata is stored in dedicated annotations

### 3. Custom Field Mapping
Fields that exist in the source version but not in the target version are:
- Detected automatically
- Stored in annotations for recovery
- Mapped to equivalent fields when possible

### 4. Resource Reference Updates
References to other Kubernetes resources are automatically updated:
- API versions are updated to match the target version
- Namespaced references are preserved
- Cross-namespace references are validated

### 5. Metadata Preservation
Critical metadata is always preserved:
- Creation timestamp
- UID
- Owner references
- Finalizers
- Resource version (for optimistic concurrency)

### 6. Data Integrity
Every conversion includes:
- SHA256 hash calculation for integrity verification
- Conversion history tracking
- Rollback capability

## Architecture

### Data Preservation Flow

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Source Object   │────▶│ Preserve Data   │────▶│ Preserved Data  │
│ (v1alpha1)      │     │                 │     │ Structure       │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                         │
                                                         ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Target Object   │◀────│ Restore Data    │◀────│ Conversion      │
│ (v1beta1)       │     │                 │     │ Process         │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### Preserved Data Structure

```go
type PreservedData struct {
    // Status fields that should be maintained
    Status map[string]interface{}
    
    // User annotations that should be preserved
    Annotations map[string]string
    
    // User labels that should be preserved
    Labels map[string]string
    
    // Custom fields not present in target version
    CustomFields map[string]interface{}
    
    // Resource references that need updating
    ResourceReferences []ResourceReference
    
    // Metadata to preserve
    Metadata PreservedMetadata
    
    // Field mappings for data transformation
    FieldMappings map[string]string
}
```

## Usage Examples

### Example 1: Basic Conversion with Status Preservation

```yaml
# Source object (v1alpha1)
apiVersion: observability.io/v1alpha1
kind: ObservabilityPlatform
metadata:
  name: production-platform
  namespace: monitoring
  annotations:
    team: platform-team
    environment: production
status:
  phase: Ready
  componentStatus:
    prometheus:
      phase: Ready
      version: v2.48.0
      readyReplicas: 3

# After conversion to v1beta1
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production-platform
  namespace: monitoring
  annotations:
    team: platform-team              # Preserved
    environment: production          # Preserved
    observability.io/conversion-data: {...}  # Added
    observability.io/last-conversion-version: v1alpha1
status:
  phase: Ready                       # Preserved
  componentStatuses:                 # Mapped from componentStatus
    prometheus:
      ready: true                    # Converted from phase
      version: v2.48.0              # Preserved
      replicas: 3                   # Mapped from readyReplicas
```

### Example 2: Custom Field Preservation

```yaml
# v1alpha1 with custom fields
spec:
  components:
    prometheus:
      customConfig:                  # Field not in v1beta1
        scrapeInterval: 30s
        evaluationInterval: 30s

# After conversion - custom fields preserved in annotation
metadata:
  annotations:
    observability.io/preserved-fields: "customConfig"
    observability.io/conversion-data: |
      {
        "customFields": {
          "customConfig": {
            "scrapeInterval": "30s",
            "evaluationInterval": "30s"
          }
        }
      }
```

### Example 3: Field Mapping

```yaml
# v1alpha1
spec:
  backup:
    retentionDays: 7              # int32 field

# After conversion to v1beta1
spec:
  backup:
    retention: "7d"               # Transformed to duration string
```

## Preservation Rules

### Default Rules

1. **Status Fields**: Always preserved as-is
2. **Metadata**: Finalizers and owner references always preserved
3. **Annotations**: User annotations preserved, system annotations filtered
4. **Labels**: User labels preserved, system labels filtered

### Custom Rules

You can define custom preservation rules:

```go
rules := []PreservationRule{
    {
        FieldPath:    "spec.backup.retentionDays",
        PreserveType: PreserveWithTransform,
        Transform: func(v interface{}) interface{} {
            if days, ok := v.(int32); ok {
                return fmt.Sprintf("%dd", days)
            }
            return v
        },
    },
    {
        FieldPath:    "spec.components.prometheus.version",
        PreserveType: PreserveWithValidation,
        Validator: func(v interface{}) error {
            if version, ok := v.(string); ok {
                if !strings.HasPrefix(version, "v") {
                    return fmt.Errorf("version must start with 'v'")
                }
            }
            return nil
        },
    },
}
```

## Conversion History

Each conversion is tracked in the object's annotations:

```yaml
metadata:
  annotations:
    observability.io/conversion-history: |
      [{
        "timestamp": "2025-06-14T10:00:00Z",
        "sourceVersion": "v1alpha1",
        "targetVersion": "v1beta1",
        "success": true,
        "dataPreserved": true,
        "fieldsModified": ["spec.backup.retention"]
      }]
```

## Best Practices

### 1. Always Test Conversions
Before deploying a new API version:
- Test conversion with production-like data
- Verify all fields are preserved correctly
- Check that status remains consistent

### 2. Use Dry-Run Mode
Test conversions without making changes:
```bash
kubectl convert -f platform.yaml --output-version=observability.io/v1beta1 --dry-run
```

### 3. Monitor Conversion Metrics
The operator exposes metrics for conversion operations:
- `gunj_operator_conversion_total`: Total conversions by source/target version
- `gunj_operator_conversion_errors_total`: Failed conversions
- `gunj_operator_conversion_field_errors_total`: Field-level conversion errors

### 4. Handle Custom Fields
If you have custom fields in your CRDs:
- Document them clearly
- Provide migration guides
- Consider adding field mappings

### 5. Validate After Conversion
Always validate converted objects:
```go
preserved := &PreservedData{...}
err := dataPreserver.ValidatePreservedData(preserved)
if err != nil {
    // Handle validation error
}
```

## Troubleshooting

### Common Issues

1. **Missing Status After Conversion**
   - Check if status preservation is enabled
   - Verify the object had status before conversion
   - Look for errors in operator logs

2. **Lost Annotations**
   - Check if annotations are user-defined (not system)
   - Verify annotation size limits aren't exceeded
   - Check for special characters in annotation values

3. **Field Mapping Failures**
   - Ensure field types are compatible
   - Check transformation functions for errors
   - Verify target field exists and is settable

### Debug Information

Enable debug logging for detailed conversion information:
```yaml
env:
  - name: LOG_LEVEL
    value: "debug"
```

Check conversion annotations:
```bash
kubectl get observabilityplatform production-platform -o jsonpath='{.metadata.annotations.observability\.io/conversion-data}' | jq .
```

## API Reference

### DataPreserver Methods

```go
// PreserveData extracts data that needs to be preserved during conversion
func (dp *DataPreserver) PreserveData(ctx context.Context, obj runtime.Object, targetVersion string) (*PreservedData, error)

// RestoreData restores preserved data after conversion
func (dp *DataPreserver) RestoreData(ctx context.Context, obj runtime.Object, preserved *PreservedData) error

// ValidatePreservedData validates that preserved data can be safely restored
func (dp *DataPreserver) ValidatePreservedData(preserved *PreservedData) error

// GetPreservationRules returns preservation rules for specific fields
func (dp *DataPreserver) GetPreservationRules(sourceVersion, targetVersion string) []PreservationRule
```

## Security Considerations

1. **Annotation Size Limits**: Kubernetes limits annotation size to 256KB total
2. **Sensitive Data**: Avoid storing secrets in preserved data
3. **Access Control**: Conversion webhooks require appropriate RBAC permissions
4. **Data Validation**: Always validate preserved data before restoration

## Future Enhancements

1. **Compression**: Compress large preserved data before storing
2. **External Storage**: Option to store preserved data in ConfigMaps for large datasets
3. **Selective Preservation**: Allow users to specify which fields to preserve
4. **Automated Recovery**: Automatic recovery from failed conversions
5. **Migration Tools**: CLI tools for bulk conversions with preservation
