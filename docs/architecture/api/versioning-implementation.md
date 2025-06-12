# API Versioning Implementation Guide

This guide provides practical implementation details for the Gunj Operator API versioning strategy.

## REST API Version Implementation

### Version Router Setup

```go
// pkg/api/router/version_router.go
package router

import (
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
    v1 "github.com/gunjanjp/gunj-operator/pkg/api/v1"
    v2 "github.com/gunjanjp/gunj-operator/pkg/api/v2"
)

type VersionRouter struct {
    v1Handler *v1.Handler
    v2Handler *v2.Handler
    config    *Config
}

func (vr *VersionRouter) Setup(router *gin.Engine) {
    // Version discovery endpoint
    router.GET("/api/version", vr.handleVersionDiscovery)
    router.GET("/api/versions", vr.handleVersionList)
    
    // Version-specific groups
    vr.setupV1Routes(router.Group("/api/v1"))
    vr.setupV2Routes(router.Group("/api/v2"))
    
    // Version negotiation for unversioned paths
    router.Any("/api/*path", vr.handleVersionNegotiation)
}

func (vr *VersionRouter) setupV1Routes(group *gin.RouterGroup) {
    // Add deprecation middleware
    group.Use(DeprecationMiddleware("v1", "2026-06-12"))
    
    // V1 routes
    group.GET("/platforms", vr.v1Handler.ListPlatforms)
    group.POST("/platforms", vr.v1Handler.CreatePlatform)
    group.GET("/platforms/:name", vr.v1Handler.GetPlatform)
    group.PUT("/platforms/:name", vr.v1Handler.UpdatePlatform)
    group.DELETE("/platforms/:name", vr.v1Handler.DeletePlatform)
}

func (vr *VersionRouter) setupV2Routes(group *gin.RouterGroup) {
    // V2 routes with new features
    group.GET("/platforms", vr.v2Handler.ListPlatforms)
    group.POST("/platforms", vr.v2Handler.CreatePlatform)
    group.GET("/platforms/:name", vr.v2Handler.GetPlatform)
    group.PATCH("/platforms/:name", vr.v2Handler.PatchPlatform) // New in v2
    group.DELETE("/platforms/:name", vr.v2Handler.DeletePlatform)
    
    // V2-only endpoints
    group.GET("/platforms/:name/cost", vr.v2Handler.GetPlatformCost)
    group.GET("/platforms/:name/recommendations", vr.v2Handler.GetRecommendations)
}
```

### Version Negotiation Middleware

```go
// pkg/api/middleware/version.go
package middleware

import (
    "fmt"
    "net/http"
    "regexp"
    "strings"
    "time"
    
    "github.com/gin-gonic/gin"
)

var versionRegex = regexp.MustCompile(`application/vnd\.gunj\.v(\d+)(?:\.(\d+))?(?:\+json)?`)

func VersionNegotiationMiddleware(supportedVersions []string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Try to extract version from Accept header
        accept := c.GetHeader("Accept")
        version := extractVersionFromAccept(accept)
        
        // Fallback to query parameter
        if version == "" {
            version = c.Query("version")
        }
        
        // Fallback to header
        if version == "" {
            version = c.GetHeader("API-Version")
        }
        
        // Validate version
        if version != "" && !isVersionSupported(version, supportedVersions) {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "unsupported_version",
                "message": fmt.Sprintf("Version %s is not supported", version),
                "supported_versions": supportedVersions,
            })
            c.Abort()
            return
        }
        
        // Store version in context
        c.Set("api_version", version)
        c.Next()
    }
}

func DeprecationMiddleware(version string, sunsetDate string) gin.HandlerFunc {
    sunset, _ := time.Parse("2006-01-02", sunsetDate)
    
    return func(c *gin.Context) {
        // Add deprecation headers
        c.Header("Deprecation", "true")
        c.Header("Sunset", sunset.Format(time.RFC1123))
        c.Header("Link", fmt.Sprintf(
            `<https://docs.gunj-operator.com/api/migrations/%s>; rel="deprecation"`,
            version,
        ))
        
        // Add warning header
        daysUntilSunset := int(time.Until(sunset).Hours() / 24)
        c.Header("Warning", fmt.Sprintf(
            `299 - "This API version is deprecated and will be removed on %s (%d days remaining)"`,
            sunsetDate,
            daysUntilSunset,
        ))
        
        c.Next()
        
        // Add deprecation notice to response
        if c.GetHeader("Content-Type") == "application/json" {
            // Inject deprecation notice into JSON response
            addDeprecationNotice(c, version, sunsetDate)
        }
    }
}

func extractVersionFromAccept(accept string) string {
    matches := versionRegex.FindStringSubmatch(accept)
    if len(matches) > 1 {
        return "v" + matches[1]
    }
    return ""
}
```

### Model Versioning

```go
// pkg/api/v1/models/platform.go
package v1

type Platform struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
    Status    string `json:"status"` // Simple string in v1
    Created   string `json:"created"`
}

// pkg/api/v2/models/platform.go
package v2

type Platform struct {
    ID        string           `json:"id"`
    Name      string           `json:"name"`
    Namespace string           `json:"namespace"`
    Status    *PlatformStatus  `json:"status"`    // Object in v2
    Metadata  *PlatformMetadata `json:"metadata"`  // New in v2
    Created   time.Time        `json:"created"`   // Proper time type
    Updated   time.Time        `json:"updated"`   // New in v2
}

type PlatformStatus struct {
    Phase      string    `json:"phase"`
    Message    string    `json:"message"`
    Conditions []Condition `json:"conditions"`
}

type PlatformMetadata struct {
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    Finalizers  []string         `json:"finalizers"`
}
```

### Version Converters

```go
// pkg/api/converters/v1_to_v2.go
package converters

import (
    "time"
    
    v1 "github.com/gunjanjp/gunj-operator/pkg/api/v1/models"
    v2 "github.com/gunjanjp/gunj-operator/pkg/api/v2/models"
)

// V1ToV2Platform converts a v1 Platform to v2
func V1ToV2Platform(v1Platform *v1.Platform) *v2.Platform {
    // Parse time
    created, _ := time.Parse(time.RFC3339, v1Platform.Created)
    
    // Map status string to status object
    status := &v2.PlatformStatus{
        Phase: mapV1StatusToV2Phase(v1Platform.Status),
        Message: generateStatusMessage(v1Platform.Status),
    }
    
    return &v2.Platform{
        ID:        v1Platform.ID,
        Name:      v1Platform.Name,
        Namespace: v1Platform.Namespace,
        Status:    status,
        Metadata: &v2.PlatformMetadata{
            Labels:      make(map[string]string),
            Annotations: make(map[string]string),
        },
        Created: created,
        Updated: created, // Use created time as initial updated time
    }
}

// V2ToV1Platform converts a v2 Platform to v1 (with potential data loss)
func V2ToV1Platform(v2Platform *v2.Platform) *v1.Platform {
    // Simplify status object to string
    status := "unknown"
    if v2Platform.Status != nil {
        status = v2Platform.Status.Phase
    }
    
    return &v1.Platform{
        ID:        v2Platform.ID,
        Name:      v2Platform.Name,
        Namespace: v2Platform.Namespace,
        Status:    status,
        Created:   v2Platform.Created.Format(time.RFC3339),
    }
}

func mapV1StatusToV2Phase(v1Status string) string {
    statusMap := map[string]string{
        "running":   "Ready",
        "pending":   "Installing",
        "failed":    "Failed",
        "unknown":   "Unknown",
    }
    
    if phase, ok := statusMap[v1Status]; ok {
        return phase
    }
    return "Unknown"
}
```

## GraphQL Version Implementation

### Schema Evolution

```graphql
# schema/schema.graphql
type Query {
  # Version-aware platform query
  platforms(
    version: ApiVersion
    namespace: String
    limit: Int
  ): PlatformConnection!
  
  # Version-specific queries (deprecated)
  platformsV1: [PlatformV1!]! @deprecated(reason: "Use platforms(version: V1)")
  platformsV2: [PlatformV2!]!
}

enum ApiVersion {
  V1
  V2
  V3_ALPHA @deprecated(reason: "Experimental - may change")
}

# Versioned types using interfaces
interface Platform {
  id: ID!
  name: String!
  namespace: String!
}

type PlatformV1 implements Platform {
  id: ID!
  name: String!
  namespace: String!
  status: String! # Simple string in v1
}

type PlatformV2 implements Platform {
  id: ID!
  name: String!
  namespace: String!
  status: PlatformStatus! # Rich object in v2
  metadata: PlatformMetadata # New in v2
  cost: CostAnalysis # New in v2
}

# Union for version-specific returns
union PlatformResult = PlatformV1 | PlatformV2
```

### GraphQL Resolvers with Versioning

```go
// pkg/graphql/resolvers/platform.go
package resolvers

import (
    "context"
    
    "github.com/gunjanjp/gunj-operator/pkg/graphql/generated"
    "github.com/gunjanjp/gunj-operator/pkg/graphql/model"
)

type Resolver struct {
    v1Service *v1.Service
    v2Service *v2.Service
}

func (r *Resolver) Platforms(ctx context.Context, version *model.ApiVersion, namespace *string, limit *int) (*model.PlatformConnection, error) {
    // Default to v2
    if version == nil {
        v := model.ApiVersionV2
        version = &v
    }
    
    switch *version {
    case model.ApiVersionV1:
        return r.platformsV1(ctx, namespace, limit)
    case model.ApiVersionV2:
        return r.platformsV2(ctx, namespace, limit)
    case model.ApiVersionV3Alpha:
        // Check if experimental features enabled
        if !r.config.EnableExperimental {
            return nil, ErrExperimentalNotEnabled
        }
        return r.platformsV3Alpha(ctx, namespace, limit)
    default:
        return nil, ErrUnsupportedVersion
    }
}

// Field-level deprecation handling
func (r *Resolver) PlatformStatus(ctx context.Context, obj *model.Platform) (interface{}, error) {
    // Log deprecation warning if old field accessed
    if fieldContext := graphql.GetFieldContext(ctx); fieldContext != nil {
        if fieldContext.Field.Field.Definition.Directives.ForName("deprecated") != nil {
            r.metrics.DeprecatedFieldAccessed(fieldContext.Field.Name)
        }
    }
    
    // Return appropriate version
    switch platform := obj.(type) {
    case *model.PlatformV1:
        return platform.Status, nil
    case *model.PlatformV2:
        return platform.Status, nil
    default:
        return nil, ErrUnknownType
    }
}
```

### GraphQL Schema Stitching

```go
// pkg/graphql/schema/stitcher.go
package schema

import (
    "github.com/graph-gophers/graphql-go"
)

type SchemaStitcher struct {
    v1Schema string
    v2Schema string
    v3Schema string
}

func (s *SchemaStitcher) BuildSchema(versions []string) (*graphql.Schema, error) {
    var schemaStr string
    
    // Base schema
    schemaStr = baseSchema
    
    // Add version-specific schemas
    for _, version := range versions {
        switch version {
        case "v1":
            schemaStr += "\n" + s.v1Schema
        case "v2":
            schemaStr += "\n" + s.v2Schema
        case "v3alpha":
            if config.EnableExperimental {
                schemaStr += "\n" + s.v3Schema
            }
        }
    }
    
    // Add version union types
    schemaStr += s.buildUnionTypes(versions)
    
    return graphql.MustParseSchema(schemaStr, s.resolver)
}

func (s *SchemaStitcher) buildUnionTypes(versions []string) string {
    unionTypes := "union PlatformResult = "
    types := []string{}
    
    for _, v := range versions {
        switch v {
        case "v1":
            types = append(types, "PlatformV1")
        case "v2":
            types = append(types, "PlatformV2")
        case "v3alpha":
            types = append(types, "PlatformV3Alpha")
        }
    }
    
    return unionTypes + strings.Join(types, " | ")
}
```

## CRD Version Implementation

### Multi-Version CRD

```go
// api/v1beta1/observabilityplatform_types.go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=observabilityplatforms,shortName=op
// +kubebuilder:deprecated:warning="v1beta1 is deprecated, use v1"

type ObservabilityPlatform struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    
    Spec   ObservabilityPlatformSpec   `json:"spec,omitempty"`
    Status ObservabilityPlatformStatus `json:"status,omitempty"`
}

// api/v1/observabilityplatform_types.go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=observabilityplatforms,shortName=op
// +kubebuilder:storageversion

type ObservabilityPlatform struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    
    Spec   ObservabilityPlatformSpec   `json:"spec,omitempty"`
    Status ObservabilityPlatformStatus `json:"status,omitempty"`
}

// Hub marks this type as a conversion hub
func (*ObservabilityPlatform) Hub() {}
```

### Conversion Webhook Implementation

```go
// pkg/webhook/observabilityplatform_conversion.go
package webhook

import (
    "context"
    "encoding/json"
    
    "k8s.io/apimachinery/pkg/runtime"
    "sigs.k8s.io/controller-runtime/pkg/conversion"
    
    v1 "github.com/gunjanjp/gunj-operator/api/v1"
    v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

type ObservabilityPlatformConversionWebhook struct {
    decoder *runtime.Decoder
}

func (w *ObservabilityPlatformConversionWebhook) ConvertTo(ctx context.Context, obj runtime.Object, toVersion string) (runtime.Object, error) {
    switch src := obj.(type) {
    case *v1beta1.ObservabilityPlatform:
        switch toVersion {
        case "v1":
            dst := &v1.ObservabilityPlatform{}
            if err := w.convertV1Beta1ToV1(src, dst); err != nil {
                return nil, err
            }
            return dst, nil
        }
    }
    
    return nil, fmt.Errorf("unsupported conversion from %T to %s", obj, toVersion)
}

func (w *ObservabilityPlatformConversionWebhook) convertV1Beta1ToV1(src *v1beta1.ObservabilityPlatform, dst *v1.ObservabilityPlatform) error {
    // Copy metadata
    dst.ObjectMeta = src.ObjectMeta
    
    // Convert spec
    dst.Spec.Components = w.convertComponents(src.Spec.Components)
    
    // Add new fields with defaults
    if src.Spec.Monitoring == nil {
        dst.Spec.Monitoring = &v1.MonitoringConfig{
            Enabled: true,
            SelfMonitoring: true,
        }
    }
    
    // Handle removed fields - store in annotations
    if src.Spec.LegacyField != "" {
        if dst.Annotations == nil {
            dst.Annotations = make(map[string]string)
        }
        dst.Annotations["observability.io/legacy-field"] = src.Spec.LegacyField
    }
    
    // Convert status
    dst.Status = w.convertStatus(src.Status)
    
    return nil
}
```

### Version-Aware Controller

```go
// controllers/observabilityplatform_controller.go
package controllers

import (
    "context"
    "fmt"
    
    "k8s.io/apimachinery/pkg/runtime/schema"
    ctrl "sigs.k8s.io/controller-runtime"
    
    v1 "github.com/gunjanjp/gunj-operator/api/v1"
    v1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

type ObservabilityPlatformReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    
    // Version-specific handlers
    v1Handler      PlatformHandler
    v1beta1Handler PlatformHandler
}

func (r *ObservabilityPlatformReconciler) SetupWithManager(mgr ctrl.Manager) error {
    // Watch both versions
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1.ObservabilityPlatform{}).
        Watches(
            &source.Kind{Type: &v1beta1.ObservabilityPlatform{}},
            handler.EnqueueRequestsFromMapFunc(r.findObjectsForV1Beta1),
        ).
        Complete(r)
}

func (r *ObservabilityPlatformReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)
    
    // Try v1 first (storage version)
    v1Platform := &v1.ObservabilityPlatform{}
    if err := r.Get(ctx, req.NamespacedName, v1Platform); err == nil {
        log.Info("Reconciling v1 platform", "version", "v1")
        return r.v1Handler.Handle(ctx, v1Platform)
    }
    
    // Fallback to v1beta1
    v1beta1Platform := &v1beta1.ObservabilityPlatform{}
    if err := r.Get(ctx, req.NamespacedName, v1beta1Platform); err == nil {
        log.Info("Reconciling v1beta1 platform", "version", "v1beta1")
        
        // Log deprecation warning
        log.Info("WARNING: v1beta1 is deprecated, please migrate to v1",
            "platform", req.NamespacedName,
            "migration_guide", "https://docs.gunj-operator.com/migration/v1",
        )
        
        return r.v1beta1Handler.Handle(ctx, v1beta1Platform)
    }
    
    // Not found
    return ctrl.Result{}, nil
}
```

## Client SDK Versioning

### Go Client with Version Support

```go
// sdk/go/client/client.go
package client

import (
    "context"
    "fmt"
    "net/http"
    
    v1 "github.com/gunjanjp/gunj-operator/sdk/go/v1"
    v2 "github.com/gunjanjp/gunj-operator/sdk/go/v2"
)

type Client struct {
    httpClient *http.Client
    baseURL    string
    version    string
    
    // Version-specific clients
    V1 *v1.Client
    V2 *v2.Client
}

func NewClient(baseURL string, opts ...Option) (*Client, error) {
    c := &Client{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        baseURL:    baseURL,
    }
    
    // Apply options
    for _, opt := range opts {
        opt(c)
    }
    
    // Auto-discover version if not specified
    if c.version == "" {
        version, err := c.discoverVersion()
        if err != nil {
            return nil, fmt.Errorf("version discovery failed: %w", err)
        }
        c.version = version
    }
    
    // Initialize version-specific clients
    switch c.version {
    case "v1":
        c.V1 = v1.NewClient(c.httpClient, c.baseURL)
    case "v2":
        c.V2 = v2.NewClient(c.httpClient, c.baseURL)
    default:
        return nil, fmt.Errorf("unsupported version: %s", c.version)
    }
    
    return c, nil
}

func (c *Client) discoverVersion() (string, error) {
    resp, err := c.httpClient.Get(c.baseURL + "/api/version")
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var info VersionInfo
    if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
        return "", err
    }
    
    // Use current stable version
    return info.Current, nil
}

// Version-agnostic interface
type PlatformClient interface {
    ListPlatforms(ctx context.Context) ([]Platform, error)
    GetPlatform(ctx context.Context, name string) (*Platform, error)
    CreatePlatform(ctx context.Context, platform *Platform) error
}

// Implement version adapter
func (c *Client) Platforms() PlatformClient {
    switch c.version {
    case "v1":
        return &v1Adapter{client: c.V1}
    case "v2":
        return &v2Adapter{client: c.V2}
    default:
        return nil
    }
}
```

### TypeScript Client with Version Support

```typescript
// sdk/typescript/src/client.ts
import { V1Client } from './v1/client';
import { V2Client } from './v2/client';
import { VersionDiscovery, ClientOptions } from './types';

export class GunjOperatorClient {
  private version: string;
  private v1Client?: V1Client;
  private v2Client?: V2Client;
  
  constructor(
    private baseURL: string,
    private options: ClientOptions = {}
  ) {}
  
  async initialize(): Promise<void> {
    // Discover version if not specified
    if (!this.options.version) {
      const discovery = await this.discoverVersion();
      this.version = this.selectBestVersion(discovery);
    } else {
      this.version = this.options.version;
    }
    
    // Initialize version-specific client
    switch (this.version) {
      case 'v1':
        this.v1Client = new V1Client(this.baseURL, this.options);
        break;
      case 'v2':
        this.v2Client = new V2Client(this.baseURL, this.options);
        break;
      default:
        throw new Error(`Unsupported version: ${this.version}`);
    }
    
    console.log(`Initialized Gunj Operator client with API ${this.version}`);
  }
  
  private async discoverVersion(): Promise<VersionDiscovery> {
    const response = await fetch(`${this.baseURL}/api/version`);
    if (!response.ok) {
      throw new Error('Failed to discover API version');
    }
    return response.json();
  }
  
  private selectBestVersion(discovery: VersionDiscovery): string {
    // Prefer stable versions
    const stable = discovery.supported
      .filter(v => discovery.versions[v].status === 'stable')
      .sort()
      .reverse();
      
    if (stable.length > 0) {
      return stable[0];
    }
    
    return discovery.current;
  }
  
  // Version-agnostic API
  get platforms() {
    if (this.v2Client) {
      return this.v2Client.platforms;
    } else if (this.v1Client) {
      return this.v1Client.platforms;
    }
    throw new Error('Client not initialized');
  }
  
  // Version-specific features
  get costAnalysis() {
    if (this.v2Client) {
      return this.v2Client.costAnalysis;
    }
    throw new Error('Cost analysis requires API v2 or later');
  }
}

// Usage
const client = new GunjOperatorClient('https://api.gunj-operator.com');
await client.initialize();

// Works with any version
const platforms = await client.platforms.list();

// Version-specific feature
try {
  const costs = await client.costAnalysis.get('production');
} catch (e) {
  console.warn('Cost analysis not available in this API version');
}
```

## Testing Version Compatibility

```go
// test/versioning/compatibility_test.go
package versioning

import (
    "testing"
    
    v1client "github.com/gunjanjp/gunj-operator/sdk/go/v1"
    v2client "github.com/gunjanjp/gunj-operator/sdk/go/v2"
)

func TestBackwardCompatibility(t *testing.T) {
    // Start server with v2 API
    server := startTestServer()
    defer server.Close()
    
    // Test v1 client with v2 server
    t.Run("V1 Client Compatibility", func(t *testing.T) {
        client := v1client.NewClient(server.URL)
        
        // Should work with compatibility layer
        platforms, err := client.ListPlatforms()
        assert.NoError(t, err)
        assert.NotEmpty(t, platforms)
        
        // Should get deprecation warning
        assert.Contains(t, client.LastResponse().Header.Get("Warning"), "deprecated")
    })
    
    // Test v2 client with v2 server
    t.Run("V2 Client Full Features", func(t *testing.T) {
        client := v2client.NewClient(server.URL)
        
        // All features should work
        platforms, err := client.ListPlatforms()
        assert.NoError(t, err)
        
        // V2-specific features
        cost, err := client.GetCostAnalysis("platform-1")
        assert.NoError(t, err)
        assert.NotNil(t, cost)
    })
}

func TestVersionNegotiation(t *testing.T) {
    tests := []struct {
        name           string
        acceptHeader   string
        expectedVersion string
    }{
        {
            name:           "Specific version",
            acceptHeader:   "application/vnd.gunj.v2+json",
            expectedVersion: "v2",
        },
        {
            name:           "Version range",
            acceptHeader:   "application/vnd.gunj.v1-v2+json",
            expectedVersion: "v2", // Latest in range
        },
        {
            name:           "No version",
            acceptHeader:   "application/json",
            expectedVersion: "v2", // Default
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/api/platforms", nil)
            req.Header.Set("Accept", tt.acceptHeader)
            
            version := negotiateVersion(req)
            assert.Equal(t, tt.expectedVersion, version)
        })
    }
}
```

This implementation guide provides practical code examples for implementing API versioning across REST APIs, GraphQL, CRDs, and client SDKs.