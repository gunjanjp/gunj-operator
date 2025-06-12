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
RUN apk add --no-cache git make ca-certificates

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source code
COPY . .

# Build the CLI with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s \
    -X 'main.version=${VERSION}' \
    -X 'main.gitCommit=${GIT_COMMIT}' \
    -X 'main.buildDate=${BUILD_DATE}'" \
    -a -installsuffix cgo \
    -o gunj-cli cmd/cli/main.go

# Runtime stage - minimal scratch image
FROM scratch

# Labels
LABEL org.opencontainers.image.title="Gunj Operator CLI"
LABEL org.opencontainers.image.description="Command-line interface for Gunj Operator"
LABEL org.opencontainers.image.url="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.source="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.vendor="gunjanjp@gmail.com"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Copy CA certificates for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the CLI binary
COPY --from=builder /workspace/gunj-cli /gunj-cli

# Set entrypoint
ENTRYPOINT ["/gunj-cli"]

# Default command shows help
CMD ["--help"]
