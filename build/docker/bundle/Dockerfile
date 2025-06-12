# syntax=docker/dockerfile:1.4

# Build all components first
FROM golang:1.21-alpine AS go-builder

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source code
COPY . .

# Build operator
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s \
    -X 'main.version=${VERSION}' \
    -X 'main.gitCommit=${GIT_COMMIT}' \
    -X 'main.buildDate=${BUILD_DATE}'" \
    -o gunj-operator cmd/operator/main.go

# Build API server
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s \
    -X 'main.version=${VERSION}' \
    -X 'main.gitCommit=${GIT_COMMIT}' \
    -X 'main.buildDate=${BUILD_DATE}'" \
    -o gunj-api-server cmd/api-server/main.go

# Build UI
FROM node:20-alpine AS ui-builder

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Set working directory
WORKDIR /app

# Copy package files
COPY ui/package*.json ./

# Install dependencies
RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline --no-audit

# Copy source code
COPY ui/ ./

# Build UI
ENV REACT_APP_VERSION=${VERSION}
ENV REACT_APP_API_URL=/api
ENV NODE_ENV=production
RUN npm run build

# Final runtime stage
FROM alpine:3.19

# Labels
LABEL org.opencontainers.image.title="Gunj Operator Bundle"
LABEL org.opencontainers.image.description="All-in-one Gunj Operator deployment"
LABEL org.opencontainers.image.url="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.source="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.vendor="gunjanjp@gmail.com"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    nginx \
    supervisor \
    curl \
    bash

# Create non-root user
RUN adduser -D -g '' -u 65532 nonroot

# Copy timezone data
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binaries
COPY --from=go-builder /workspace/gunj-operator /usr/local/bin/gunj-operator
COPY --from=go-builder /workspace/gunj-api-server /usr/local/bin/gunj-api-server

# Copy UI build
COPY --from=ui-builder /app/build /usr/share/nginx/html

# Copy configuration files
COPY build/docker/bundle/nginx.conf /etc/nginx/nginx.conf
COPY build/docker/bundle/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY build/docker/bundle/entrypoint.sh /entrypoint.sh

# Make entrypoint executable
RUN chmod +x /entrypoint.sh

# Create necessary directories
RUN mkdir -p /var/log/supervisor /var/run/supervisor && \
    chown -R nonroot:nonroot /var/log/supervisor /var/run/supervisor && \
    chown -R nonroot:nonroot /usr/share/nginx/html && \
    chown -R nonroot:nonroot /var/cache/nginx && \
    chown -R nonroot:nonroot /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R nonroot:nonroot /var/run/nginx.pid

# Expose ports
EXPOSE 3000 8080 8081 8090 8091 9443

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:3000/health && \
        curl -f http://localhost:8090/health && \
        curl -f http://localhost:8081/healthz || exit 1

# Use non-root user
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]
