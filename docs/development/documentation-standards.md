# Documentation Standards

This document establishes the documentation standards for the Gunj Operator project. All documentation must follow these guidelines to ensure consistency, clarity, and maintainability.

## Table of Contents

- [Overview](#overview)
- [Documentation Structure](#documentation-structure)
- [Writing Style Guide](#writing-style-guide)
- [Markdown Standards](#markdown-standards)
- [Code Documentation](#code-documentation)
- [API Documentation](#api-documentation)
- [Documentation Types](#documentation-types)
- [Review Process](#review-process)
- [Tools and Automation](#tools-and-automation)

## Overview

Good documentation is crucial for the success of the Gunj Operator project. It helps users understand and use the operator effectively, assists developers in contributing, and ensures long-term maintainability.

### Documentation Principles

1. **Clear and Concise**: Use simple language and avoid jargon
2. **Complete**: Cover all features and use cases
3. **Accurate**: Keep documentation up-to-date with code
4. **Accessible**: Consider readers with different skill levels
5. **Searchable**: Use clear headings and keywords
6. **Visual**: Include diagrams and examples where helpful
7. **Tested**: Verify all examples and commands work

## Documentation Structure

```
docs/
├── README.md                    # Project overview and quick start
├── getting-started/            # New user guides
│   ├── installation.md         # Installation instructions
│   ├── quick-start.md         # 5-minute quick start
│   ├── concepts.md            # Core concepts
│   └── first-platform.md      # Creating first platform
├── user-guide/                # Detailed user documentation
│   ├── configuration.md       # Configuration reference
│   ├── platforms.md          # Platform management
│   ├── components.md         # Component details
│   ├── monitoring.md         # Monitoring setup
│   ├── alerting.md          # Alerting configuration
│   ├── backup-restore.md    # Backup and restore
│   ├── upgrades.md          # Upgrade procedures
│   ├── troubleshooting.md   # Common issues
│   └── faq.md              # Frequently asked questions
├── api/                      # API documentation
│   ├── rest-api.md          # REST API reference
│   ├── graphql-api.md       # GraphQL API reference
│   ├── crd-reference.md     # CRD specifications
│   └── webhooks.md          # Webhook documentation
├── architecture/            # Technical architecture
│   ├── overview.md         # Architecture overview
│   ├── operator-design.md  # Operator design
│   ├── api-design.md      # API design
│   ├── security.md        # Security architecture
│   ├── scalability.md     # Scalability design
│   └── decisions/         # Architecture Decision Records
├── development/            # Developer documentation
│   ├── setup.md           # Development setup
│   ├── guidelines.md      # Coding guidelines
│   ├── testing.md        # Testing guide
│   ├── debugging.md      # Debugging guide
│   ├── contributing.md   # Contribution guide
│   └── releasing.md      # Release process
├── operations/           # Operations guide
│   ├── deployment.md    # Deployment options
│   ├── production.md    # Production setup
│   ├── monitoring.md    # Monitoring setup
│   ├── security.md      # Security hardening
│   └── disaster-recovery.md
├── tutorials/           # Step-by-step tutorials
│   ├── multi-cluster.md
│   ├── gitops-integration.md
│   ├── custom-dashboards.md
│   └── advanced-alerting.md
└── reference/          # Reference materials
    ├── cli.md         # CLI reference
    ├── metrics.md     # Metrics reference
    ├── events.md      # Events reference
    ├── glossary.md    # Terms and definitions
    └── resources.md   # External resources
```

## Writing Style Guide

### General Guidelines

1. **Voice and Tone**
   - Use active voice: "The operator creates a deployment" not "A deployment is created"
   - Be direct and clear: "You must" not "It is recommended that you"
   - Be friendly but professional
   - Use "we" for the project, "you" for the reader

2. **Language**
   - Use American English spelling
   - Define acronyms on first use: "Custom Resource Definition (CRD)"
   - Avoid idioms and cultural references
   - Use inclusive language

3. **Structure**
   - Start with the most important information
   - Use short paragraphs (3-5 sentences)
   - Use bulleted lists for multiple items
   - Use numbered lists for sequential steps
   - Include a summary for long documents

### Headings

- **H1 (#)**: Document title only, one per document
- **H2 (##)**: Major sections
- **H3 (###)**: Subsections
- **H4 (####)**: Sub-subsections (avoid if possible)

Example:
```markdown
# Platform Configuration Guide

## Overview

This guide explains how to configure observability platforms...

## Basic Configuration

### Component Selection

Configure which components to deploy...

### Resource Allocation

Set resource limits and requests...
```

### Common Terms

Use these terms consistently:

| Use | Don't Use |
|-----|-----------|
| Kubernetes | k8s |
| namespace | Namespace |
| Pod | pod |
| click | click on |
| select | check |
| enter | type |
| API | api |
| URL | url |

## Markdown Standards

### Basic Formatting

1. **Bold** for UI elements: "Click **Create Platform**"
2. *Italic* for emphasis: "This is *very* important"
3. `Code` for commands, files, values: "Run `kubectl apply`"
4. Use blank lines between elements
5. Limit line length to 120 characters

### Code Blocks

Always specify the language:

````markdown
```bash
kubectl apply -f platform.yaml
```

```yaml
apiVersion: observability.io/v1beta1
kind: ObservabilityPlatform
metadata:
  name: production
```

```go
func main() {
    fmt.Println("Hello, Gunj!")
}
```
````

### Tables

Use tables for structured data:

```markdown
| Component | Version | Required |
|-----------|---------|----------|
| Prometheus | v2.48.0 | Yes |
| Grafana | 10.2.0 | Yes |
| Loki | 2.9.0 | No |
```

### Links

- Use descriptive link text: `[Installation Guide](install.md)` not `[Click here](install.md)`
- Use relative links for internal documents
- Use HTTPS for external links
- Check links regularly

### Images and Diagrams

```markdown
![Platform Architecture](../images/architecture.png)
*Figure 1: Gunj Operator Architecture Overview*
```

- Use PNG for screenshots
- Use SVG for diagrams
- Include alt text
- Add captions
- Keep file sizes reasonable (<1MB)

## Code Documentation

### Go Documentation

Follow [Effective Go](https://golang.org/doc/effective_go) guidelines:

```go
// Package operator provides the core operator functionality for managing
// observability platforms in Kubernetes environments.
package operator

// ReconcileResult represents the result of a reconciliation operation.
// It contains information about whether reconciliation should be requeued
// and any error that occurred during the process.
type ReconcileResult struct {
    // Requeue indicates whether the reconciliation should be requeued
    Requeue bool
    
    // RequeueAfter specifies the duration to wait before requeuing
    RequeueAfter time.Duration
    
    // Error contains any error that occurred during reconciliation
    Error error
}

// Reconcile processes the ObservabilityPlatform resource and ensures
// all components are in the desired state. It returns a ReconcileResult
// indicating whether reconciliation should be retried.
//
// The reconciliation process includes:
//   - Validating the platform specification
//   - Creating or updating Prometheus deployments
//   - Configuring Grafana with appropriate datasources
//   - Setting up Loki for log aggregation
//   - Updating the platform status
//
// Example:
//
//	result, err := r.Reconcile(ctx, platform)
//	if err != nil {
//	    return ctrl.Result{}, err
//	}
func (r *Reconciler) Reconcile(ctx context.Context, platform *v1beta1.ObservabilityPlatform) (ReconcileResult, error) {
    // Implementation
}
```

### TypeScript/JavaScript Documentation

Use JSDoc format:

```typescript
/**
 * Represents a platform configuration in the UI.
 * @interface PlatformConfig
 */
export interface PlatformConfig {
  /** The unique name of the platform */
  name: string;
  
  /** The Kubernetes namespace */
  namespace: string;
  
  /** Component configuration */
  components: ComponentConfig;
}

/**
 * Creates a new observability platform with the specified configuration.
 * 
 * @param {PlatformConfig} config - The platform configuration
 * @returns {Promise<Platform>} The created platform
 * @throws {ValidationError} If the configuration is invalid
 * @throws {APIError} If the API request fails
 * 
 * @example
 * const platform = await createPlatform({
 *   name: 'production',
 *   namespace: 'monitoring',
 *   components: {
 *     prometheus: { enabled: true }
 *   }
 * });
 */
export async function createPlatform(config: PlatformConfig): Promise<Platform> {
  // Implementation
}
```

## API Documentation

### REST API Documentation

Use OpenAPI 3.0 specification:

```yaml
openapi: 3.0.0
info:
  title: Gunj Operator API
  version: 1.0.0
  description: |
    The Gunj Operator API provides programmatic access to manage
    observability platforms in Kubernetes.

paths:
  /api/v1/platforms:
    get:
      summary: List all platforms
      description: |
        Returns a list of all observability platforms in the cluster.
        Results can be filtered by namespace and labels.
      parameters:
        - name: namespace
          in: query
          description: Filter by namespace
          schema:
            type: string
        - name: labelSelector
          in: query
          description: Filter by labels
          schema:
            type: string
      responses:
        '200':
          description: List of platforms
          content:
            application/json:
              schema:
                type: object
                properties:
                  items:
                    type: array
                    items:
                      $ref: '#/components/schemas/Platform'
              example:
                items:
                  - name: production
                    namespace: monitoring
                    status: Ready
```

### GraphQL Documentation

Document schemas with descriptions:

```graphql
"""
Represents an observability platform managed by the Gunj Operator.
"""
type Platform {
  """
  The unique name of the platform within its namespace.
  """
  name: String!
  
  """
  The Kubernetes namespace containing the platform.
  """
  namespace: String!
  
  """
  The current status of the platform.
  """
  status: PlatformStatus!
  
  """
  The components deployed as part of this platform.
  """
  components: [Component!]!
  
  """
  Metrics for the platform over the specified time range.
  """
  metrics(range: TimeRange!): PlatformMetrics!
}

"""
Input for creating a new observability platform.
"""
input CreatePlatformInput {
  """
  The name for the new platform. Must be unique within the namespace.
  """
  name: String!
  
  """
  The namespace where the platform will be created.
  Default: "default"
  """
  namespace: String = "default"
  
  """
  Component configuration for the platform.
  """
  components: ComponentConfigInput!
}
```

## Documentation Types

### User Documentation

**Purpose**: Help users deploy and manage the operator

**Audience**: DevOps engineers, SREs, platform engineers

**Style**: Task-oriented, practical examples

**Template**: [User Guide Template](templates/user-guide-template.md)

### API Reference

**Purpose**: Detailed technical reference

**Audience**: Developers integrating with the API

**Style**: Precise, complete, with examples

**Template**: [API Reference Template](templates/api-reference-template.md)

### Architecture Documentation

**Purpose**: Explain design decisions and system structure

**Audience**: Contributors, architects

**Style**: Technical, with diagrams

**Template**: [Architecture Doc Template](templates/architecture-template.md)

### Tutorial

**Purpose**: Step-by-step learning

**Audience**: New users

**Style**: Friendly, detailed, progressive

**Template**: [Tutorial Template](templates/tutorial-template.md)

## Review Process

### Documentation Review Checklist

- [ ] **Accuracy**: Information is correct and up-to-date
- [ ] **Completeness**: All features/options documented
- [ ] **Clarity**: Easy to understand
- [ ] **Examples**: Working examples provided
- [ ] **Structure**: Logical organization
- [ ] **Style**: Follows style guide
- [ ] **Links**: All links work
- [ ] **Images**: Clear and relevant
- [ ] **Code**: Examples are tested
- [ ] **Grammar**: No spelling/grammar errors

### Review Workflow

1. **Author**: Write documentation following standards
2. **Technical Review**: Verify technical accuracy
3. **Editorial Review**: Check style and clarity
4. **Testing**: Verify examples work
5. **Approval**: Merge when criteria met

## Tools and Automation

### Documentation Linting

Use markdownlint for consistency:

```bash
# Check all markdown files
markdownlint '**/*.md'

# Fix common issues
markdownlint --fix '**/*.md'
```

### Link Checking

```bash
# Check for broken links
npm run check-links

# Specific file
linkcheck docs/user-guide/installation.md
```

### Documentation Generation

```bash
# Generate API docs from code
make generate-api-docs

# Generate CRD reference
make generate-crd-docs

# Build documentation site
make build-docs
```

### Vale for Style Checking

Configure Vale for consistent style:

```ini
# .vale.ini
StylesPath = docs/styles
MinAlertLevel = warning

[*.md]
BasedOnStyles = Vale, Google, write-good
```

## Documentation Maintenance

### Regular Reviews

- **Weekly**: Check for broken links
- **Monthly**: Review for accuracy
- **Quarterly**: Major documentation review
- **Release**: Update for new features

### Documentation Debt

Track documentation debt in issues:

```markdown
**Documentation Debt: Installation Guide**

The installation guide needs updates for:
- [ ] New authentication options
- [ ] Kubernetes 1.29 support
- [ ] Troubleshooting section

Priority: High
Effort: 4 hours
```

### Metrics

Track documentation quality:

- Documentation coverage (features documented)
- Link rot percentage
- Time to first contribution
- User feedback scores
- Search success rate

---

## Summary

Good documentation is essential for the success of the Gunj Operator. By following these standards, we ensure our documentation is:

- **Consistent**: Same style throughout
- **Clear**: Easy to understand
- **Complete**: Covers all features
- **Current**: Up-to-date with code
- **Helpful**: Solves real problems

Remember: If it's not documented, it doesn't exist!

For questions about documentation standards, contact the documentation team or post in #gunj-operator-docs.
