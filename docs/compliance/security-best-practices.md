# Security Best Practices Guide

**Version**: 1.0  
**Date**: June 12, 2025  
**Project**: Gunj Operator  
**Author**: Security Engineering Team  
**Status**: Active Guidelines  

---

## 📋 Overview

This guide provides security best practices for all team members working on the Gunj Operator project. Following these practices helps prevent security vulnerabilities and ensures our code, infrastructure, and operations remain secure.

## 🔐 Core Security Principles

### 1. Defense in Depth
- **Multiple Layers**: Never rely on a single security control
- **Fail Secure**: Default to secure state when failures occur
- **Redundancy**: Backup security controls for critical functions

### 2. Least Privilege
- **Minimal Access**: Grant only necessary permissions
- **Time-Limited**: Use temporary credentials when possible
- **Regular Review**: Audit and revoke unused permissions

### 3. Zero Trust
- **Verify Everything**: Never trust, always verify
- **Assume Breach**: Design assuming attackers are already inside
- **Continuous Validation**: Re-authenticate and re-authorize

---

## 💻 Development Security

### Secure Coding Practices

#### Input Validation
```go
// GOOD: Validate all inputs
func CreatePlatform(name string, namespace string) error {
    // Validate name
    if err := validation.ValidateName(name); err != nil {
        return fmt.Errorf("invalid platform name: %w", err)
    }
    
    // Validate namespace
    if err := validation.ValidateNamespace(namespace); err != nil {
        return fmt.Errorf("invalid namespace: %w", err)
    }
    
    // Sanitize inputs
    name = sanitize.Name(name)
    namespace = sanitize.Namespace(namespace)
    
    // Continue with safe inputs...
}

// BAD: Direct use without validation
func CreatePlatformUnsafe(name string, namespace string) error {
    // DON'T DO THIS - no validation!
    return k8sClient.Create(name, namespace)
}
```

#### Authentication & Authorization
```go
// GOOD: Proper authentication and authorization
func (s *APIServer) handleRequest(w http.ResponseWriter, r *http.Request) {
    // Authenticate
    user, err := s.authenticate(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Authorize
    if !s.authorize(user, r.Method, r.URL.Path) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // Audit log
    s.auditLog(user, r.Method, r.URL.Path)
    
    // Process request...
}
```

#### Secret Management
```go
// GOOD: Use environment variables or secret management
func getAPIKey() (string, error) {
    // From environment
    key := os.Getenv("API_KEY")
    if key != "" {
        return key, nil
    }
    
    // From Kubernetes secret
    secret, err := k8sClient.CoreV1().Secrets("gunj-system").Get("api-credentials", metav1.GetOptions{})
    if err != nil {
        return "", err
    }
    
    return string(secret.Data["api-key"]), nil
}

// BAD: Hardcoded secrets
const APIKey = "sk_live_abcd1234" // NEVER DO THIS!
```

#### Error Handling
```go
// GOOD: Safe error messages
func Login(username, password string) error {
    user, err := db.GetUser(username)
    if err != nil {
        log.Errorf("Login failed for user %s: %v", username, err)
        return errors.New("invalid credentials") // Generic message
    }
    
    if !user.CheckPassword(password) {
        log.Errorf("Invalid password for user %s", username)
        return errors.New("invalid credentials") // Same generic message
    }
    
    return nil
}

// BAD: Information leakage
func LoginUnsafe(username, password string) error {
    user, err := db.GetUser(username)
    if err != nil {
        return fmt.Errorf("user %s not found", username) // Leaks user existence
    }
    
    if !user.CheckPassword(password) {
        return errors.New("incorrect password") // Different error message
    }
    
    return nil
}
```

### Dependency Security

#### Package Management
```yaml
# go.mod - Pin dependencies
module github.com/gunjanjp/gunj-operator

go 1.21

require (
    k8s.io/api v0.28.4
    k8s.io/apimachinery v0.28.4
    sigs.k8s.io/controller-runtime v0.16.3
)

// Replace vulnerable versions
replace (
    github.com/vulnerable/package v1.0.0 => github.com/vulnerable/package v1.0.1
)
```

#### Dependency Scanning
```yaml
# .github/workflows/security.yml
name: Security Scan

on: [push, pull_request]

jobs:
  dependency-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Run Snyk
      uses: snyk/actions/golang@master
      env:
        SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
      with:
        args: --severity-threshold=high
        
    - name: Run Nancy
      run: |
        go list -json -m all | nancy sleuth
```

---

## 🐳 Container Security

### Dockerfile Best Practices

```dockerfile
# GOOD: Secure Dockerfile
# Use specific version, not latest
FROM golang:1.21-alpine AS builder

# Run as non-root during build
RUN adduser -D -g '' appuser

# Copy only necessary files
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o app

# Use distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy from builder
COPY --from=builder /build/app /app

# Run as non-root user
USER nonroot:nonroot

# No shell, no package manager
ENTRYPOINT ["/app"]
```

### Container Runtime Security

```yaml
# GOOD: Secure Pod specification
apiVersion: v1
kind: Pod
metadata:
  name: secure-pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 10001
    fsGroup: 10001
    seccompProfile:
      type: RuntimeDefault
  
  containers:
  - name: app
    image: myapp:1.0.0
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
    resources:
      limits:
        cpu: "1"
        memory: "512Mi"
      requests:
        cpu: "100m"
        memory: "128Mi"
    volumeMounts:
    - name: tmp
      mountPath: /tmp
    - name: cache
      mountPath: /app/cache
  
  volumes:
  - name: tmp
    emptyDir: {}
  - name: cache
    emptyDir: {}
```

---

## ☸️ Kubernetes Security

### RBAC Best Practices

```yaml
# GOOD: Least privilege RBAC
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: gunj-operator
  namespace: gunj-system
rules:
# Only what's needed
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: ["observability.io"]
  resources: ["observabilityplatforms/status"]
  verbs: ["update", "patch"]

---
# BAD: Overly permissive
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: too-permissive
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]  # NEVER DO THIS!
```

### Network Policies

```yaml
# GOOD: Restrictive network policy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: gunj-operator-netpol
  namespace: gunj-system
spec:
  podSelector:
    matchLabels:
      app: gunj-operator
  policyTypes:
  - Ingress
  - Egress
  
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: gunj-system
    - podSelector:
        matchLabels:
          app: gunj-api
    ports:
    - protocol: TCP
      port: 8443
  
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: TCP
      port: 443
  - to:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 9090
```

### Secret Management

```yaml
# GOOD: Using Sealed Secrets
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: api-credentials
  namespace: gunj-system
spec:
  encryptedData:
    api-key: AgB2L6VvM6E... # Encrypted value
    
---
# GOOD: Using external secret operator
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: vault-secret
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: app-secret
  data:
  - secretKey: api-key
    remoteRef:
      key: secret/data/api
      property: key
```

---

## 🌐 API Security

### REST API Security

```go
// GOOD: Secure API implementation
func (s *APIServer) setupMiddleware() {
    // Rate limiting
    s.router.Use(middleware.RateLimit(100, time.Minute))
    
    // CORS with specific origins
    s.router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"https://app.gunj-operator.io"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Authorization", "Content-Type"},
        ExposeHeaders:    []string{"X-Request-ID"},
        AllowCredentials: true,
        MaxAge:          12 * time.Hour,
    }))
    
    // Security headers
    s.router.Use(func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Next()
    })
    
    // Authentication
    s.router.Use(s.authMiddleware())
    
    // Audit logging
    s.router.Use(s.auditMiddleware())
}
```

### GraphQL Security

```graphql
# GOOD: GraphQL with security directives
type Query {
  # Rate limited query
  platforms(limit: Int = 10, offset: Int = 0): [Platform!]! 
    @rateLimit(window: "1m", max: 100)
    @auth(requires: USER)
  
  # Authorized query with field-level security
  platform(id: ID!): Platform 
    @auth(requires: USER)
    @hasPermission(resource: "platform", action: "read")
}

type Platform {
  id: ID!
  name: String!
  # Sensitive field requires additional permission
  secrets: [Secret!]! @auth(requires: ADMIN)
}

type Mutation {
  # Input validation and authorization
  createPlatform(input: CreatePlatformInput!): Platform! 
    @auth(requires: ADMIN)
    @validate
    @audit(action: "create_platform")
}
```

---

## 📊 Monitoring & Logging

### Security Logging

```go
// GOOD: Structured security logging
type SecurityLogger struct {
    logger *zap.Logger
}

func (s *SecurityLogger) LogSecurityEvent(event SecurityEvent) {
    fields := []zap.Field{
        zap.String("event_type", event.Type),
        zap.String("user_id", event.UserID),
        zap.String("ip_address", event.IPAddress),
        zap.String("action", event.Action),
        zap.String("resource", event.Resource),
        zap.Bool("success", event.Success),
        zap.Time("timestamp", event.Timestamp),
    }
    
    if event.Success {
        s.logger.Info("Security event", fields...)
    } else {
        fields = append(fields, zap.String("reason", event.FailureReason))
        s.logger.Warn("Security event failed", fields...)
    }
}

// Usage
s.LogSecurityEvent(SecurityEvent{
    Type:      "authentication",
    UserID:    user.ID,
    IPAddress: getClientIP(r),
    Action:    "login",
    Resource:  "api",
    Success:   true,
    Timestamp: time.Now(),
})
```

### Security Metrics

```go
// GOOD: Security metrics collection
var (
    authAttempts = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "auth_attempts_total",
            Help: "Total authentication attempts",
        },
        []string{"method", "success"},
    )
    
    authLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "auth_duration_seconds",
            Help:    "Authentication duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )
    
    suspiciousActivities = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "suspicious_activities_total",
            Help: "Total suspicious activities detected",
        },
        []string{"type", "action"},
    )
)
```

---

## 🚨 Incident Response

### Security Incident Checklist

#### Immediate Response (0-15 minutes)
- [ ] Identify and classify the incident
- [ ] Activate incident response team
- [ ] Contain the immediate threat
- [ ] Start incident documentation
- [ ] Notify key stakeholders

#### Investigation (15-60 minutes)
- [ ] Collect relevant logs
- [ ] Identify affected systems
- [ ] Determine attack vector
- [ ] Assess data exposure
- [ ] Document timeline

#### Containment (1-4 hours)
- [ ] Isolate affected systems
- [ ] Revoke compromised credentials
- [ ] Block malicious IPs
- [ ] Patch vulnerabilities
- [ ] Enhance monitoring

#### Recovery (4-24 hours)
- [ ] Restore from clean backups
- [ ] Rebuild compromised systems
- [ ] Reset all related credentials
- [ ] Verify system integrity
- [ ] Resume normal operations

#### Post-Incident (1-7 days)
- [ ] Complete incident report
- [ ] Conduct lessons learned
- [ ] Update security controls
- [ ] Improve detection rules
- [ ] Train team on findings

---

## 🎓 Security Training

### Required Training Modules

| Role | Training Requirements | Frequency |
|------|---------------------|-----------|
| **All Staff** |
| Security Awareness | Basic security principles | Annual |
| Phishing Awareness | Recognizing attacks | Quarterly |
| Data Handling | Secure data practices | Annual |
| **Developers** |
| Secure Coding | OWASP Top 10, best practices | Bi-annual |
| Container Security | Docker/K8s security | Annual |
| DevSecOps | Security in CI/CD | Annual |
| **Operations** |
| Incident Response | Handling security events | Quarterly |
| Security Tools | Using security tools | On-boarding |
| Compliance | Regulatory requirements | Annual |

### Security Champions Program

```yaml
security_champions:
  requirements:
    - complete_advanced_security_training
    - demonstrate_security_leadership
    - contribute_to_security_initiatives
    - mentor_team_members
  
  responsibilities:
    - security_point_of_contact
    - code_review_participation
    - threat_modeling_sessions
    - security_awareness_promotion
  
  benefits:
    - additional_training_budget
    - conference_attendance
    - certification_support
    - recognition_program
```

---

## 📝 Security Checklist for PRs

### Before Submitting PR

- [ ] **Code Security**
  - [ ] No hardcoded secrets
  - [ ] Input validation implemented
  - [ ] Error handling doesn't leak info
  - [ ] Authentication/authorization checked
  
- [ ] **Dependencies**
  - [ ] No new vulnerable dependencies
  - [ ] Dependencies are necessary
  - [ ] Licenses are compatible
  
- [ ] **Tests**
  - [ ] Security tests included
  - [ ] Edge cases covered
  - [ ] Negative tests present
  
- [ ] **Documentation**
  - [ ] Security implications documented
  - [ ] API changes noted
  - [ ] Configuration options explained

### Review Focus Areas

```yaml
security_review_checklist:
  authentication:
    - proper_token_validation
    - session_management
    - password_policies
    - mfa_support
  
  authorization:
    - rbac_implementation
    - permission_checks
    - privilege_escalation_prevention
    - audit_logging
  
  data_protection:
    - encryption_at_rest
    - encryption_in_transit
    - pii_handling
    - data_retention
  
  input_validation:
    - sql_injection_prevention
    - xss_prevention
    - xxe_prevention
    - command_injection_prevention
```

---

## 🔄 Continuous Improvement

### Security Metrics Dashboard

Monitor these key security metrics:
1. **Vulnerability Metrics**: Open vulnerabilities by severity
2. **Patch Metrics**: Time to patch by severity
3. **Incident Metrics**: MTTD and MTTR
4. **Compliance Metrics**: Policy adherence rate
5. **Training Metrics**: Completion rates

### Regular Security Activities

| Activity | Frequency | Owner |
|----------|-----------|-------|
| Dependency updates | Weekly | Dependabot |
| Security scans | Daily | CI/CD |
| Access review | Monthly | Security team |
| Penetration test | Quarterly | External |
| Security training | Quarterly | All teams |
| Incident drills | Semi-annual | Response team |

---

## 📚 Resources

### Security References
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)

### Tools & Libraries
- **Go Security**: `gosec`, `nancy`, `go-sec-tools`
- **Container Security**: `trivy`, `grype`, `syft`
- **Kubernetes Security**: `kubesec`, `kube-bench`, `falco`
- **API Security**: `OWASP ZAP`, `swagger-validator`

---

*Security is everyone's responsibility. When in doubt, ask the security team!*

**Questions?** Contact security@gunj-operator.io

**Last Updated**: June 12, 2025  
**Next Review**: September 12, 2025
