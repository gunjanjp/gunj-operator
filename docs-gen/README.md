# Documentation Generation Configuration

This directory contains configuration and scripts for generating documentation from the Gunj Operator codebase.

## Documentation Types

1. **API Documentation**
   - Generated from OpenAPI/Swagger specs
   - GraphQL schema documentation
   - Go package documentation (godoc)

2. **Code Documentation**
   - TypeScript/React component documentation
   - Storybook for UI components
   - Architecture diagrams

3. **User Documentation**
   - Markdown-based documentation
   - Generated with MkDocs
   - Versioned documentation

## Tools Used

- **MkDocs**: Main documentation site generator
- **godoc**: Go package documentation
- **TypeDoc**: TypeScript API documentation
- **Storybook**: React component documentation
- **Swagger UI**: REST API documentation
- **GraphQL Playground**: GraphQL API documentation

## Quick Start

```bash
# Install documentation tools
make install-docs-tools

# Generate all documentation
make docs-generate

# Serve documentation locally
make docs-serve

# Build documentation for production
make docs-build
```

## Directory Structure

```
docs-gen/
├── mkdocs.yml          # MkDocs configuration
├── typedoc.json        # TypeDoc configuration
├── .storybook/         # Storybook configuration
├── templates/          # Custom templates
├── scripts/            # Generation scripts
└── output/             # Generated documentation
```
