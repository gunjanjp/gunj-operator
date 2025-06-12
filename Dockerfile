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

# Create non-root user for runtime
RUN adduser -D -g '' -u 65532 nonroot

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source code
COPY . .

# Build the operator with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s \
    -X 'main.version=${VERSION}' \
    -X 'main.gitCommit=${GIT_COMMIT}' \
    -X 'main.buildDate=${BUILD_DATE}'" \
    -a -installsuffix cgo \
    -o gunj-operator cmd/operator/main.go

# Runtime stage - using distroless for security
FROM gcr.io/distroless/static:nonroot

# Labels
LABEL org.opencontainers.image.title="Gunj Operator"
LABEL org.opencontainers.image.description="Enterprise Kubernetes operator for observability platform management"
LABEL org.opencontainers.image.url="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.source="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.vendor="gunjanjp@gmail.com"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Copy timezone data for proper time handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates for TLS verification
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the operator binary
COPY --from=builder /workspace/gunj-operator /gunj-operator

# Use non-root user
USER 65532:65532

# Expose metrics port
EXPOSE 8080

# Expose health port
EXPOSE 8081

# Expose webhook port
EXPOSE 9443

# Set entrypoint
ENTRYPOINT ["/gunj-operator"]

# Default arguments
CMD ["--leader-elect=true", "--metrics-bind-addr=:8080", "--health-addr=:8081"]
