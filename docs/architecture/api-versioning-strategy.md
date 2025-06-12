# Gunj Operator API Versioning Strategy

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Architecture Decision  
**Contact**: gunjanjp@gmail.com  

---

## Executive Summary

This document defines the API versioning strategy for the Gunj Operator, covering both RESTful and GraphQL APIs. The strategy ensures backward compatibility, clear deprecation paths, and seamless evolution of APIs while maintaining enterprise-grade stability.

---

## 🎯 Versioning Principles

### Core Principles

1. **Backward Compatibility**: Breaking changes require new major versions
2. **Clear Communication**: Version changes are well-documented and communicated
3. **Graceful Deprecation**: Minimum 6-month deprecation period for major versions
4. **Consistency**: Unified versioning approach across all API types
5. **Predictability**: Clear rules for what constitutes breaking changes

### Non-Breaking Changes

- Adding new optional fields to requests
- Adding new fields to responses
- Adding new endpoints or operations
- Adding new optional query parameters
- Adding new enum values (when client can handle unknowns)

### Breaking Changes

- Removing or renaming fields, endpoints, or operations
- Changing field types or formats
- Changing authentication mechanisms
- Modifying validation rules to be more restrictive
- Changing default behaviors
- Removing enum values

---

## 🔄 REST API Versioning

### URL-Based Versioning Strategy

```
https://api.gunj-operator.yourdomain.com/api/{version}/{resource}
```

#### Version Format
- **Pattern**: `v{major}` (e.g., v1, v2, v3)
- **Location**: URL path segment after `/api/`
- **Examples**:
  ```
  /api/v1/platforms
  /api/v1/platforms/{name}/components
  /api/v2/platforms  (new version with breaking changes)
  ```

#### Version Lifecycle

1. **Alpha** (`v1alpha1`, `v1alpha2`)
   - Experimental features
   - No stability guarantees
   - Can be removed without deprecation
   - Not recommended for production

2. **Beta** (`v1beta1`, `v1beta2`)
   - Feature complete but may change
   - Deprecation period: 3 months
   - Suitable for non-critical use

3. **Stable** (`v1`, `v2`)
   - Production ready
   - Deprecation period: 6 months minimum
   - Full backward compatibility within major version

### Implementation Details

```go
// router/api.go
func SetupAPIRoutes(router *gin.Engine) {
    // v1 API routes (stable)
    v1 := router.Group("/api/v1")
    {
        v1.Use(middleware.APIVersion("v1"))
        setupV1Routes(v1)
    }
    
    // v1beta1 API routes (beta)
    v1beta1 := router.Group("/api/v1beta1")
    {
        v1beta1.Use(middleware.APIVersion("v1beta1"))
        v1beta1.Use(middleware.BetaWarning())
        setupV1Beta1Routes(v1beta1)
    }
    
    // v2 API routes (next stable version)
    v2 := router.Group("/api/v2")
    {
        v2.Use(middleware.APIVersion("v2"))
        setupV2Routes(v2)
    }
}
```

### Version Negotiation

#### Request Headers
```http
GET /api/v1/platforms HTTP/1.1
Host: api.gunj-operator.yourdomain.com
Accept: application/vnd.gunj-operator.v1+json
Accept-Version: v1
```

#### Response Headers
```http
HTTP/1.1 200 OK
Content-Type: application/vnd.gunj-operator.v1+json
X-API-Version: v1
X-API-Deprecated: false
X-API-Sunset: 2026-06-12T00:00:00Z  (if deprecated)
Sunset: Thu, 12 Jun 2026 00:00:00 GMT  (RFC 7231)
```

### Deprecation Process

1. **Announcement** (T-6 months)
   - Add deprecation headers
   - Update documentation
   - Notify users via channels

2. **Warning Period** (T-3 months)
   - Add warning logs
   - Include migration guides
   - Provide tooling for migration

3. **End of Life** (T-0)
   - Return 410 Gone for deprecated endpoints
   - Provide helpful error messages with migration path

```go
// middleware/deprecation.go
func DeprecationMiddleware(version string, sunsetDate time.Time) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-API-Deprecated", "true")
        c.Header("X-API-Sunset", sunsetDate.Format(time.RFC3339))
        c.Header("Sunset", sunsetDate.Format(time.RFC1123))
        c.Header("Link", fmt.Sprintf("</api/%s/docs>; rel=\"sunset\"", version))
        
        // Add warning to response
        c.Header("Warning", fmt.Sprintf("299 - \"API version %s is deprecated. Please migrate to %s\"", 
            version, getLatestVersion()))
        
        c.Next()
    }
}
```

---

## 📊 GraphQL API Versioning

### Field-Level Evolution Strategy

GraphQL versioning follows a different approach focusing on field-level evolution rather than URL versioning.

#### Schema Evolution Rules

1. **Additive Changes Only**
   - New fields can be added anytime
   - New types can be introduced
   - New arguments with defaults are allowed

2. **Deprecation Over Removal**
   ```graphql
   type Platform {
     id: ID!
     name: String!
     # Deprecated in favor of metadata.namespace
     namespace: String! @deprecated(reason: "Use metadata.namespace instead")
     metadata: PlatformMetadata!
   }
   ```

3. **Versioned Types When Necessary**
   ```graphql
   type Query {
     # Original query
     platforms: [Platform!]! @deprecated(reason: "Use platformsV2 for enhanced features")
     
     # New version with different structure
     platformsV2(filter: PlatformFilter): PlatformConnection!
   }
   ```

### Implementation Strategy

```typescript
// schema/platform.graphql
"""
Platform represents an observability platform instance.
API Version: v1
"""
type Platform {
  id: ID!
  metadata: PlatformMetadata!
  spec: PlatformSpec!
  status: PlatformStatus!
  
  # Version-specific fields
  apiVersion: String!
  
  # Deprecated fields
  namespace: String! @deprecated(reason: "Use metadata.namespace instead. Will be removed in v2")
}

"""
Enhanced platform type for v2 features.
API Version: v2
"""
type PlatformV2 {
  id: ID!
  metadata: PlatformMetadataV2!
  spec: PlatformSpecV2!
  status: PlatformStatusV2!
  apiVersion: String!
  
  # New v2 features
  cost: CostAnalysis!
  recommendations: [Recommendation!]!
}
```

### Client Version Detection

```typescript
// resolvers/platform.ts
export const platformResolvers = {
  Query: {
    platforms: async (parent, args, context) => {
      const clientVersion = context.clientVersion || 'v1';
      
      if (clientVersion === 'v2') {
        return await getPlatformsV2(args);
      }
      
      // Default to v1 behavior
      return await getPlatformsV1(args);
    }
  }
};

// middleware/graphql-version.ts
export function extractClientVersion(req: Request): string {
  // Check custom header
  const headerVersion = req.headers['x-graphql-client-version'];
  if (headerVersion) return headerVersion;
  
  // Check query extension
  const query = req.body.query;
  const match = query.match(/#\s*@version\((\w+)\)/);
  if (match) return match[1];
  
  // Default version
  return 'v1';
}
```

---

## 🔗 CRD API Versioning

### Kubernetes Native Versioning

Following Kubernetes conventions for Custom Resource Definitions:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: observabilityplatforms.observability.io
spec:
  group: observability.io
  versions:
  - name: v1beta1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
  - name: v1alpha1
    served: true
    storage: false
    deprecated: true
    deprecationWarning: "v1alpha1 is deprecated; use v1beta1"
    schema:
      openAPIV3Schema:
        type: object
  
  conversion:
    strategy: Webhook
    webhook:
      clientConfig:
        service:
          name: gunj-operator-webhook
          namespace: gunj-system
          path: "/convert"
      conversionReviewVersions: ["v1", "v1beta1"]
```

### Version Conversion

```go
// api/v1beta1/conversion.go
func (dst *ObservabilityPlatform) ConvertFrom(srcRaw conversion.Hub) error {
    switch src := srcRaw.(type) {
    case *v1alpha1.ObservabilityPlatform:
        // Convert from v1alpha1
        dst.ObjectMeta = src.ObjectMeta
        if err := convertV1Alpha1SpecToV1Beta1(&src.Spec, &dst.Spec); err != nil {
            return err
        }
    case *v1.ObservabilityPlatform:
        // Convert from v1
        return fmt.Errorf("downgrade from v1 to v1beta1 not supported")
    default:
        return fmt.Errorf("unsupported type %T", src)
    }
    return nil
}
```

---

## 🛡️ Client SDK Versioning

### SDK Release Strategy

```go
// github.com/gunjanjp/gunj-operator-sdk-go
module github.com/gunjanjp/gunj-operator-sdk-go/v2

go 1.21

// Major version in module path for breaking changes
```

### Multi-Version Support

```go
// client/client.go
package client

type Client struct {
    // Base configuration
    config     *Config
    httpClient *http.Client
    
    // Version-specific clients
    V1      *V1Client
    V1Beta1 *V1Beta1Client
    V2      *V2Client
}

// NewClient creates a multi-version client
func NewClient(config *Config) (*Client, error) {
    return &Client{
        config:     config,
        httpClient: &http.Client{},
        V1:         NewV1Client(config),
        V1Beta1:    NewV1Beta1Client(config),
        V2:         NewV2Client(config),
    }, nil
}
```

---

## 📊 Version Discovery

### Discovery Endpoint

```json
GET /api/versions

{
  "versions": [
    {
      "version": "v1",
      "status": "stable",
      "deprecated": false,
      "startDate": "2025-01-01T00:00:00Z",
      "endDate": null,
      "links": {
        "self": "/api/v1",
        "docs": "/api/v1/docs",
        "openapi": "/api/v1/openapi.json"
      }
    },
    {
      "version": "v1beta1",
      "status": "beta",
      "deprecated": false,
      "startDate": "2024-10-01T00:00:00Z",
      "endDate": null,
      "links": {
        "self": "/api/v1beta1",
        "docs": "/api/v1beta1/docs",
        "openapi": "/api/v1beta1/openapi.json"
      }
    },
    {
      "version": "v2",
      "status": "alpha",
      "deprecated": false,
      "startDate": "2025-04-01T00:00:00Z",
      "endDate": null,
      "links": {
        "self": "/api/v2",
        "docs": "/api/v2/docs",
        "openapi": "/api/v2/openapi.json"
      }
    }
  ],
  "recommended": "v1",
  "minimum": "v1",
  "graphql": {
    "endpoint": "/graphql",
    "version": "2025-06-12",
    "introspection": true
  }
}
```

---

## 🔄 Migration Support

### Migration Tools

1. **API Migration CLI**
   ```bash
   gunj-cli api migrate --from v1 --to v2 --dry-run
   gunj-cli api validate --version v2 platform.yaml
   ```

2. **Automated Testing**
   ```go
   // test/migration/v1_to_v2_test.go
   func TestV1ToV2Migration(t *testing.T) {
       v1Client := NewV1Client()
       v2Client := NewV2Client()
       
       // Create in v1
       v1Platform := createV1Platform()
       
       // Migrate to v2
       v2Platform := migrateToV2(v1Platform)
       
       // Verify compatibility
       assert.Equal(t, v1Platform.Name, v2Platform.Metadata.Name)
   }
   ```

3. **Documentation**
   - Migration guides for each version
   - API changelog with examples
   - Breaking change notifications
   - Code examples for common migrations

---

## 📋 Version Compatibility Matrix

| Client Version | API v1 | API v1beta1 | API v2 | GraphQL |
|---------------|--------|-------------|---------|---------|
| SDK v1.x      | ✅ Full | ⚠️ Limited | ❌ None | ✅ v1 schema |
| SDK v2.x      | ✅ Full | ✅ Full | ✅ Full | ✅ v1+v2 schema |
| CLI v1.x      | ✅ Full | ❌ None | ❌ None | ⚠️ Limited |
| CLI v2.x      | ✅ Full | ✅ Full | ✅ Full | ✅ Full |

---

## 🎯 Implementation Timeline

1. **Phase 1**: Implement v1 stable API
2. **Phase 2**: Add v1beta1 for new features
3. **Phase 3**: Introduce v2 alpha
4. **Phase 4**: Promote v2 to stable
5. **Phase 5**: Deprecate v1 (6-month notice)
6. **Phase 6**: Remove v1 support

---

This versioning strategy ensures the Gunj Operator APIs can evolve while maintaining stability and providing clear migration paths for users.
