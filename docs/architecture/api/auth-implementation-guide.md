# Authentication Implementation Guide

This guide provides practical implementation details for the Gunj Operator authentication and authorization system.

## JWT Implementation

### Token Structure

```go
// pkg/auth/jwt/claims.go
package jwt

import (
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    jwt.RegisteredClaims
    
    // User identification
    Email      string   `json:"email"`
    Name       string   `json:"name"`
    
    // Authorization
    Roles       []string          `json:"roles"`
    Permissions []string          `json:"permissions"`
    Namespaces  []string          `json:"namespaces"`
    
    // Tenant information
    TenantID   string            `json:"tenant_id,omitempty"`
    TenantName string            `json:"tenant_name,omitempty"`
    
    // Session metadata
    SessionID  string            `json:"session_id"`
    DeviceID   string            `json:"device_id,omitempty"`
    IPAddress  string            `json:"ip_address,omitempty"`
}

// RefreshTokenClaims for refresh tokens
type RefreshTokenClaims struct {
    jwt.RegisteredClaims
    
    SessionID   string `json:"session_id"`
    DeviceID    string `json:"device_id"`
    TokenFamily string `json:"token_family"`
    Used        bool   `json:"used"`
}
```

### Token Service

```go
// pkg/auth/jwt/service.go
package jwt

import (
    "crypto/rsa"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type TokenService struct {
    privateKey     *rsa.PrivateKey
    publicKey      *rsa.PublicKey
    issuer         string
    audience       []string
    accessExpiry   time.Duration
    refreshExpiry  time.Duration
}

func (s *TokenService) GenerateTokenPair(user *User) (*TokenPair, error) {
    now := time.Now()
    sessionID := uuid.New().String()
    
    // Create access token
    accessClaims := &Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.issuer,
            Subject:   user.ID,
            Audience:  s.audience,
            ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
            NotBefore: jwt.NewNumericDate(now),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        uuid.New().String(),
        },
        Email:       user.Email,
        Name:        user.Name,
        Roles:       user.Roles,
        Permissions: user.Permissions,
        Namespaces:  user.Namespaces,
        TenantID:    user.TenantID,
        SessionID:   sessionID,
    }
    
    accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
    accessTokenString, err := accessToken.SignedString(s.privateKey)
    if err != nil {
        return nil, fmt.Errorf("signing access token: %w", err)
    }
    
    // Create refresh token
    refreshClaims := &RefreshTokenClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.issuer,
            Subject:   user.ID,
            ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExpiry)),
            IssuedAt:  jwt.NewNumericDate(now),
            ID:        uuid.New().String(),
        },
        SessionID:   sessionID,
        TokenFamily: uuid.New().String(),
    }
    
    refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
    refreshTokenString, err := refreshToken.SignedString(s.privateKey)
    if err != nil {
        return nil, fmt.Errorf("signing refresh token: %w", err)
    }
    
    return &TokenPair{
        AccessToken:  accessTokenString,
        RefreshToken: refreshTokenString,
        ExpiresIn:    int(s.accessExpiry.Seconds()),
        TokenType:    "Bearer",
    }, nil
}

func (s *TokenService) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.publicKey, nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("parsing token: %w", err)
    }
    
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }
    
    // Additional validation
    if err := s.validateClaims(claims); err != nil {
        return nil, err
    }
    
    return claims, nil
}
```

## OIDC Integration

### OIDC Configuration

```go
// pkg/auth/oidc/config.go
package oidc

type ProviderConfig struct {
    Name         string
    Issuer       string
    ClientID     string
    ClientSecret string
    RedirectURI  string
    Scopes       []string
    
    // Claim mappings
    UserIDClaim    string
    EmailClaim     string
    NameClaim      string
    GroupsClaim    string
    
    // Advanced options
    SkipIssuerCheck   bool
    SkipClientIDCheck bool
    RequiredClaims    map[string]interface{}
}

type Config struct {
    Providers map[string]*ProviderConfig
    
    // Group to role mappings
    GroupMappings map[string]string
    
    // Default role for new users
    DefaultRole string
    
    // Session configuration
    SessionKey    []byte
    SessionMaxAge int
}
```

### OIDC Service

```go
// pkg/auth/oidc/service.go
package oidc

import (
    "context"
    "encoding/json"
    
    "github.com/coreos/go-oidc/v3/oidc"
    "golang.org/x/oauth2"
)

type Service struct {
    providers map[string]*Provider
    config    *Config
}

type Provider struct {
    config   *ProviderConfig
    verifier *oidc.IDTokenVerifier
    oauth2   *oauth2.Config
}

func (s *Service) HandleLogin(provider string) (string, error) {
    p, ok := s.providers[provider]
    if !ok {
        return "", ErrProviderNotFound
    }
    
    state := generateSecureState()
    nonce := generateSecureNonce()
    
    // Store state and nonce in cache
    s.storeAuthState(state, &AuthState{
        Provider: provider,
        Nonce:    nonce,
        Created:  time.Now(),
    })
    
    // Generate authorization URL
    return p.oauth2.AuthCodeURL(state,
        oidc.Nonce(nonce),
        oauth2.AccessTypeOffline,
    ), nil
}

func (s *Service) HandleCallback(ctx context.Context, code, state string) (*User, error) {
    // Validate state
    authState, err := s.validateState(state)
    if err != nil {
        return nil, err
    }
    
    p := s.providers[authState.Provider]
    
    // Exchange code for tokens
    token, err := p.oauth2.Exchange(ctx, code)
    if err != nil {
        return nil, fmt.Errorf("exchanging code: %w", err)
    }
    
    // Extract ID token
    rawIDToken, ok := token.Extra("id_token").(string)
    if !ok {
        return nil, ErrNoIDToken
    }
    
    // Verify ID token
    idToken, err := p.verifier.Verify(ctx, rawIDToken)
    if err != nil {
        return nil, fmt.Errorf("verifying ID token: %w", err)
    }
    
    // Validate nonce
    if idToken.Nonce != authState.Nonce {
        return nil, ErrInvalidNonce
    }
    
    // Extract claims
    var claims map[string]interface{}
    if err := idToken.Claims(&claims); err != nil {
        return nil, fmt.Errorf("extracting claims: %w", err)
    }
    
    // Map claims to user
    return s.mapClaimsToUser(p.config, claims)
}

func (s *Service) mapClaimsToUser(config *ProviderConfig, claims map[string]interface{}) (*User, error) {
    user := &User{
        ID:    extractClaim(claims, config.UserIDClaim),
        Email: extractClaim(claims, config.EmailClaim),
        Name:  extractClaim(claims, config.NameClaim),
    }
    
    // Map groups to roles
    if groups := extractStringSlice(claims, config.GroupsClaim); len(groups) > 0 {
        user.Roles = s.mapGroupsToRoles(groups)
    } else {
        user.Roles = []string{s.config.DefaultRole}
    }
    
    // Derive permissions from roles
    user.Permissions = s.derivePermissions(user.Roles)
    
    return user, nil
}
```

## API Key Management

### API Key Model

```go
// pkg/auth/apikey/model.go
package apikey

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "strings"
)

type APIKey struct {
    ID          string
    KeyHash     string
    Name        string
    Description string
    UserID      string
    
    // Permissions
    Scopes      []string
    Namespaces  []string
    
    // Restrictions
    RateLimit   int
    IPWhitelist []string
    
    // Metadata
    CreatedAt   time.Time
    ExpiresAt   *time.Time
    LastUsedAt  *time.Time
    RevokedAt   *time.Time
    RevokedBy   string
    RevokeReason string
}

func GenerateAPIKey(environment string) (string, string, error) {
    // Generate random bytes
    randomBytes := make([]byte, 32)
    if _, err := rand.Read(randomBytes); err != nil {
        return "", "", err
    }
    
    // Format: gop_[env]_[random]_[checksum]
    random := hex.EncodeToString(randomBytes[:16])
    key := fmt.Sprintf("gop_%s_%s", environment, random)
    
    // Add checksum
    checksum := calculateChecksum(key)
    fullKey := fmt.Sprintf("%s_%s", key, checksum)
    
    // Calculate hash for storage
    hash := sha256.Sum256([]byte(fullKey))
    keyHash := hex.EncodeToString(hash[:])
    
    return fullKey, keyHash, nil
}

func ValidateAPIKey(key string) error {
    parts := strings.Split(key, "_")
    if len(parts) != 4 {
        return ErrInvalidFormat
    }
    
    if parts[0] != "gop" {
        return ErrInvalidPrefix
    }
    
    // Validate checksum
    baseKey := strings.Join(parts[:3], "_")
    expectedChecksum := calculateChecksum(baseKey)
    if parts[3] != expectedChecksum {
        return ErrInvalidChecksum
    }
    
    return nil
}
```

### API Key Service

```go
// pkg/auth/apikey/service.go
package apikey

type Service struct {
    store Store
    cache Cache
}

func (s *Service) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*APIKey, string, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, "", err
    }
    
    // Generate key
    key, keyHash, err := GenerateAPIKey(req.Environment)
    if err != nil {
        return nil, "", err
    }
    
    // Create API key record
    apiKey := &APIKey{
        ID:          uuid.New().String(),
        KeyHash:     keyHash,
        Name:        req.Name,
        Description: req.Description,
        UserID:      req.UserID,
        Scopes:      req.Scopes,
        Namespaces:  req.Namespaces,
        RateLimit:   req.RateLimit,
        IPWhitelist: req.IPWhitelist,
        CreatedAt:   time.Now(),
    }
    
    if req.ExpiresIn > 0 {
        expiresAt := time.Now().Add(req.ExpiresIn)
        apiKey.ExpiresAt = &expiresAt
    }
    
    // Store in database
    if err := s.store.Create(ctx, apiKey); err != nil {
        return nil, "", err
    }
    
    // Audit log
    s.auditLog(ctx, "api_key.created", apiKey)
    
    return apiKey, key, nil
}

func (s *Service) ValidateAPIKey(ctx context.Context, key string) (*APIKeyContext, error) {
    // Validate format
    if err := ValidateAPIKey(key); err != nil {
        return nil, err
    }
    
    // Calculate hash
    hash := sha256.Sum256([]byte(key))
    keyHash := hex.EncodeToString(hash[:])
    
    // Check cache
    if cached, ok := s.cache.Get(keyHash); ok {
        return cached.(*APIKeyContext), nil
    }
    
    // Load from database
    apiKey, err := s.store.GetByHash(ctx, keyHash)
    if err != nil {
        return nil, err
    }
    
    // Validate status
    if err := s.validateKeyStatus(apiKey); err != nil {
        return nil, err
    }
    
    // Check IP whitelist
    clientIP := GetClientIP(ctx)
    if !s.checkIPWhitelist(apiKey.IPWhitelist, clientIP) {
        return nil, ErrIPNotAllowed
    }
    
    // Create context
    keyCtx := &APIKeyContext{
        KeyID:      apiKey.ID,
        UserID:     apiKey.UserID,
        Scopes:     apiKey.Scopes,
        Namespaces: apiKey.Namespaces,
        RateLimit:  apiKey.RateLimit,
    }
    
    // Cache for future requests
    s.cache.Set(keyHash, keyCtx, 5*time.Minute)
    
    // Update last used asynchronously
    go s.updateLastUsed(apiKey.ID)
    
    return keyCtx, nil
}
```

## RBAC Implementation

### Permission Service

```go
// pkg/auth/rbac/service.go
package rbac

type Service struct {
    store  Store
    cache  Cache
    config *Config
}

func (s *Service) HasPermission(ctx context.Context, user *User, resource, action string) bool {
    // Check super admin
    if s.isSuperAdmin(user) {
        return true
    }
    
    // Build permission string
    permission := fmt.Sprintf("%s:%s", resource, action)
    
    // Check direct permissions
    if contains(user.Permissions, permission) {
        return true
    }
    
    // Check wildcard permissions
    if contains(user.Permissions, fmt.Sprintf("%s:*", resource)) ||
       contains(user.Permissions, "*") {
        return true
    }
    
    // Check role-based permissions
    for _, role := range user.Roles {
        if s.roleHasPermission(role, permission) {
            return true
        }
    }
    
    // Check namespace-scoped permissions
    if namespace := GetNamespace(ctx); namespace != "" {
        if s.hasNamespacePermission(user, namespace, permission) {
            return true
        }
    }
    
    // Audit the denial
    s.auditDenial(ctx, user, resource, action)
    
    return false
}

func (s *Service) EnforcePermission(permission string) gin.HandlerFunc {
    parts := strings.Split(permission, ":")
    if len(parts) != 2 {
        panic("invalid permission format")
    }
    
    resource, action := parts[0], parts[1]
    
    return func(c *gin.Context) {
        user, ok := GetUser(c)
        if !ok {
            c.AbortWithStatusJSON(401, gin.H{
                "error": "Unauthenticated",
            })
            return
        }
        
        if !s.HasPermission(c.Request.Context(), user, resource, action) {
            c.AbortWithStatusJSON(403, gin.H{
                "error": "Insufficient permissions",
                "required": permission,
                "user_permissions": user.Permissions,
            })
            return
        }
        
        c.Next()
    }
}
```

### Role Management

```go
// pkg/auth/rbac/roles.go
package rbac

type Role struct {
    Name        string
    Description string
    Permissions []string
    Inherited   []string // Inherited roles
}

var defaultRoles = map[string]*Role{
    "admin": {
        Name:        "admin",
        Description: "Full system administrator",
        Permissions: []string{"*"},
    },
    "platform-operator": {
        Name:        "platform-operator",
        Description: "Manage observability platforms",
        Permissions: []string{
            "platform:create",
            "platform:read",
            "platform:update",
            "platform:delete",
            "component:*",
            "backup:create",
            "backup:read",
        },
    },
    "platform-viewer": {
        Name:        "platform-viewer",
        Description: "View platforms and metrics",
        Permissions: []string{
            "platform:read",
            "component:read",
            "metrics:read",
            "logs:read",
        },
    },
    "cost-analyst": {
        Name:        "cost-analyst",
        Description: "View cost and optimization data",
        Permissions: []string{
            "platform:read",
            "cost:read",
            "recommendations:read",
        },
        Inherited: []string{"platform-viewer"},
    },
}

func (s *Service) GetRolePermissions(roleName string) []string {
    role, ok := s.roles[roleName]
    if !ok {
        return nil
    }
    
    permissions := make([]string, 0, len(role.Permissions))
    permissions = append(permissions, role.Permissions...)
    
    // Add inherited permissions
    for _, inherited := range role.Inherited {
        permissions = append(permissions, s.GetRolePermissions(inherited)...)
    }
    
    return unique(permissions)
}
```

## Middleware Implementation

### Authentication Middleware

```go
// pkg/middleware/auth.go
package middleware

func AuthenticationMiddleware(authService *auth.Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Try multiple auth methods in order
        
        // 1. Bearer token
        if token := extractBearerToken(c); token != "" {
            claims, err := authService.ValidateJWT(token)
            if err == nil {
                setUserContext(c, claims)
                c.Next()
                return
            }
        }
        
        // 2. API Key
        if apiKey := extractAPIKey(c); apiKey != "" {
            keyCtx, err := authService.ValidateAPIKey(c.Request.Context(), apiKey)
            if err == nil {
                setAPIKeyContext(c, keyCtx)
                c.Next()
                return
            }
        }
        
        // 3. Session cookie (for web UI)
        if session := extractSessionCookie(c); session != "" {
            claims, err := authService.ValidateSession(session)
            if err == nil {
                setUserContext(c, claims)
                c.Next()
                return
            }
        }
        
        // 4. mTLS client certificate
        if cert := extractClientCert(c); cert != nil {
            claims, err := authService.ValidateCertificate(cert)
            if err == nil {
                setUserContext(c, claims)
                c.Next()
                return
            }
        }
        
        // No valid authentication
        c.AbortWithStatusJSON(401, gin.H{
            "error": "Authentication required",
            "details": "No valid authentication credentials provided",
        })
    }
}

func extractBearerToken(c *gin.Context) string {
    auth := c.GetHeader("Authorization")
    if auth == "" {
        return ""
    }
    
    parts := strings.Split(auth, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        return ""
    }
    
    return parts[1]
}
```

### GraphQL Auth Directive

```go
// pkg/graphql/directives/auth.go
package directives

func HasRole(roles []string) func(next graphql.Resolver) graphql.Resolver {
    return func(next graphql.Resolver) graphql.Resolver {
        return func(ctx context.Context, args interface{}) (interface{}, error) {
            user := auth.GetUserFromContext(ctx)
            if user == nil {
                return nil, ErrUnauthenticated
            }
            
            for _, requiredRole := range roles {
                for _, userRole := range user.Roles {
                    if userRole == requiredRole {
                        return next(ctx, args)
                    }
                }
            }
            
            return nil, ErrUnauthorized{
                RequiredRoles: roles,
                UserRoles:     user.Roles,
            }
        }
    }
}

func HasPermission(permissions []string) func(next graphql.Resolver) graphql.Resolver {
    return func(next graphql.Resolver) graphql.Resolver {
        return func(ctx context.Context, args interface{}) (interface{}, error) {
            user := auth.GetUserFromContext(ctx)
            if user == nil {
                return nil, ErrUnauthenticated
            }
            
            rbacService := rbac.GetServiceFromContext(ctx)
            
            for _, permission := range permissions {
                parts := strings.Split(permission, ":")
                if len(parts) != 2 {
                    continue
                }
                
                if rbacService.HasPermission(ctx, user, parts[0], parts[1]) {
                    return next(ctx, args)
                }
            }
            
            return nil, ErrUnauthorized{
                RequiredPermissions: permissions,
                UserPermissions:     user.Permissions,
            }
        }
    }
}
```

## Security Headers

```go
// pkg/middleware/security.go
package middleware

func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Security headers
        headers := map[string]string{
            "X-Content-Type-Options":            "nosniff",
            "X-Frame-Options":                   "DENY",
            "X-XSS-Protection":                  "1; mode=block",
            "Strict-Transport-Security":         "max-age=31536000; includeSubDomains",
            "Referrer-Policy":                   "strict-origin-when-cross-origin",
            "Permissions-Policy":                "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
            "Content-Security-Policy":           buildCSP(),
            "Cross-Origin-Embedder-Policy":      "require-corp",
            "Cross-Origin-Opener-Policy":        "same-origin",
            "Cross-Origin-Resource-Policy":      "same-origin",
        }
        
        for key, value := range headers {
            c.Header(key, value)
        }
        
        c.Next()
    }
}

func buildCSP() string {
    directives := []string{
        "default-src 'self'",
        "script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net",
        "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
        "img-src 'self' data: https:",
        "font-src 'self' https://fonts.gstatic.com",
        "connect-src 'self' wss: https://api.gunj-operator.com",
        "media-src 'none'",
        "object-src 'none'",
        "frame-ancestors 'none'",
        "base-uri 'self'",
        "form-action 'self'",
        "upgrade-insecure-requests",
    }
    
    return strings.Join(directives, "; ")
}
```

## Audit Logging

```go
// pkg/audit/logger.go
package audit

type Logger struct {
    store Store
    queue chan *Event
}

type Event struct {
    ID          string
    Timestamp   time.Time
    EventType   string
    UserID      string
    Username    string
    IPAddress   string
    UserAgent   string
    Resource    string
    Action      string
    Result      string
    Details     map[string]interface{}
    RequestID   string
    SessionID   string
}

func (l *Logger) Log(ctx context.Context, eventType string, details map[string]interface{}) {
    user := auth.GetUserFromContext(ctx)
    request := GetRequestFromContext(ctx)
    
    event := &Event{
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        EventType: eventType,
        UserID:    user.ID,
        Username:  user.Email,
        IPAddress: request.ClientIP,
        UserAgent: request.UserAgent,
        Resource:  request.Resource,
        Action:    request.Action,
        Result:    "success",
        Details:   details,
        RequestID: request.ID,
        SessionID: user.SessionID,
    }
    
    select {
    case l.queue <- event:
    case <-time.After(100 * time.Millisecond):
        // Log to error channel if queue is full
        log.Error("Audit log queue full, dropping event", "event", event)
    }
}

func (l *Logger) processEvents() {
    batch := make([]*Event, 0, 100)
    ticker := time.NewTicker(1 * time.Second)
    
    for {
        select {
        case event := <-l.queue:
            batch = append(batch, event)
            
            if len(batch) >= 100 {
                l.flush(batch)
                batch = batch[:0]
            }
            
        case <-ticker.C:
            if len(batch) > 0 {
                l.flush(batch)
                batch = batch[:0]
            }
        }
    }
}
```

This implementation guide provides the core components needed to build a secure authentication and authorization system for the Gunj Operator.