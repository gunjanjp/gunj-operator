#!/bin/bash
# Entrypoint script for Gunj Operator Bundle
set -e

# Default values
KUBECONFIG=${KUBECONFIG:-/home/nonroot/.kube/config}
LOG_LEVEL=${LOG_LEVEL:-info}
ENABLE_AUTH=${ENABLE_AUTH:-false}
API_PORT=${API_PORT:-8090}
UI_PORT=${UI_PORT:-3000}

# Function to check if running in Kubernetes
is_in_kubernetes() {
    if [ -f /var/run/secrets/kubernetes.io/serviceaccount/token ]; then
        return 0
    else
        return 1
    fi
}

# Function to wait for service
wait_for_service() {
    local service=$1
    local port=$2
    local max_attempts=30
    local attempt=0
    
    echo "Waiting for $service on port $port..."
    while ! nc -z localhost $port; do
        attempt=$((attempt + 1))
        if [ $attempt -eq $max_attempts ]; then
            echo "Error: $service failed to start after $max_attempts attempts"
            exit 1
        fi
        sleep 2
    done
    echo "$service is ready!"
}

# Configure environment based on deployment context
if is_in_kubernetes; then
    echo "Running in Kubernetes cluster mode"
    export IN_CLUSTER=true
else
    echo "Running in standalone mode"
    export IN_CLUSTER=false
    
    # Check if kubeconfig exists
    if [ ! -f "$KUBECONFIG" ]; then
        echo "Warning: Kubeconfig not found at $KUBECONFIG"
        echo "Operator will run in limited mode without cluster access"
        export OPERATOR_MODE=standalone
    fi
fi

# Create runtime configuration for UI
cat > /usr/share/nginx/html/config.js <<EOF
window._env_ = {
  API_URL: "${API_URL:-/api}",
  WS_URL: "${WS_URL:-/ws}",
  GRAPHQL_URL: "${GRAPHQL_URL:-/graphql}",
  AUTH_ENABLED: ${ENABLE_AUTH},
  VERSION: "${VERSION:-dev}",
  BUILD_DATE: "${BUILD_DATE:-unknown}",
  GIT_COMMIT: "${GIT_COMMIT:-unknown}",
  BUNDLE_MODE: true
};
EOF

# Update index.html to include config.js
if ! grep -q "config.js" /usr/share/nginx/html/index.html; then
    sed -i 's|</head>|<script src="/config.js"></script></head>|' /usr/share/nginx/html/index.html
fi

# Start supervisord
echo "Starting Gunj Operator Bundle..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
