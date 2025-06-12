# syntax=docker/dockerfile:1.4

# Build stage
FROM golang:1.21-alpine AS builder

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source code
COPY . .

# Build the API server with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s \
    -X 'main.version=${VERSION}' \
    -X 'main.gitCommit=${GIT_COMMIT}' \
    -X 'main.buildDate=${BUILD_DATE}'" \
    -a -installsuffix cgo \
    -o gunj-api-server cmd/api-server/main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot

# Labels
LABEL org.opencontainers.image.title="Gunj Operator API Server"
LABEL org.opencontainers.image.description="RESTful and GraphQL API server for Gunj Operator"
LABEL org.opencontainers.image.url="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.source="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.vendor="gunjanjp@gmail.com"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the API server binary
COPY --from=builder /workspace/gunj-api-server /gunj-api-server

# Copy API documentation (if needed)
COPY --from=builder /workspace/api/openapi /api/openapi

# Use non-root user
USER 65532:65532

# Expose API port
EXPOSE 8090

# Expose metrics port
EXPOSE 8091

# Expose GraphQL playground port (dev only)
EXPOSE 8092

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/gunj-api-server", "health"]

# Set entrypoint
ENTRYPOINT ["/gunj-api-server"]

# Default arguments
CMD ["--port=8090", "--metrics-port=8091", "--log-level=info"]
