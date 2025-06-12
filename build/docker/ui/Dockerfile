# syntax=docker/dockerfile:1.4

# Build stage
FROM node:20-alpine AS builder

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG REACT_APP_API_URL=/api
ARG REACT_APP_VERSION=${VERSION}

# Install build dependencies
RUN apk add --no-cache git python3 make g++

# Set working directory
WORKDIR /app

# Copy package files
COPY ui/package*.json ./

# Install dependencies with cache mount
RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline --no-audit

# Copy source code
COPY ui/ ./

# Set environment variables for build
ENV REACT_APP_VERSION=${REACT_APP_VERSION}
ENV REACT_APP_API_URL=${REACT_APP_API_URL}
ENV REACT_APP_BUILD_DATE=${BUILD_DATE}
ENV REACT_APP_GIT_COMMIT=${GIT_COMMIT}
ENV NODE_ENV=production

# Build the application
RUN npm run build

# Runtime stage - nginx
FROM nginx:1.25-alpine

# Labels
LABEL org.opencontainers.image.title="Gunj Operator UI"
LABEL org.opencontainers.image.description="React-based web interface for Gunj Operator"
LABEL org.opencontainers.image.url="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.source="https://github.com/gunjanjp/gunj-operator"
LABEL org.opencontainers.image.vendor="gunjanjp@gmail.com"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Install runtime dependencies
RUN apk add --no-cache curl

# Remove default nginx config
RUN rm -rf /etc/nginx/conf.d/*

# Copy nginx configuration
COPY build/docker/ui/nginx.conf /etc/nginx/nginx.conf
COPY build/docker/ui/default.conf /etc/nginx/conf.d/default.conf

# Copy built application
COPY --from=builder /app/build /usr/share/nginx/html

# Copy runtime configuration script
COPY build/docker/ui/runtime-config.sh /docker-entrypoint.d/40-runtime-config.sh
RUN chmod +x /docker-entrypoint.d/40-runtime-config.sh

# Create non-root user
RUN adduser -D -g '' -u 65532 nonroot && \
    chown -R nonroot:nonroot /usr/share/nginx/html && \
    chown -R nonroot:nonroot /var/cache/nginx && \
    chown -R nonroot:nonroot /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R nonroot:nonroot /var/run/nginx.pid

# Use non-root user
USER 65532:65532

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
