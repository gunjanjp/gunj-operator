#!/bin/sh
# Runtime configuration for Gunj Operator UI
# This script runs at container startup to inject runtime configuration

set -e

# Default values
API_URL="${REACT_APP_API_URL:-/api}"
WS_URL="${REACT_APP_WS_URL:-/ws}"
GRAPHQL_URL="${REACT_APP_GRAPHQL_URL:-/graphql}"
AUTH_ENABLED="${REACT_APP_AUTH_ENABLED:-true}"
OIDC_AUTHORITY="${REACT_APP_OIDC_AUTHORITY:-}"
OIDC_CLIENT_ID="${REACT_APP_OIDC_CLIENT_ID:-gunj-operator-ui}"
THEME_MODE="${REACT_APP_THEME_MODE:-light}"

# Create runtime config file
cat > /usr/share/nginx/html/config.js <<EOF
window._env_ = {
  API_URL: "${API_URL}",
  WS_URL: "${WS_URL}",
  GRAPHQL_URL: "${GRAPHQL_URL}",
  AUTH_ENABLED: ${AUTH_ENABLED},
  OIDC_AUTHORITY: "${OIDC_AUTHORITY}",
  OIDC_CLIENT_ID: "${OIDC_CLIENT_ID}",
  THEME_MODE: "${THEME_MODE}",
  VERSION: "${VERSION}",
  BUILD_DATE: "${BUILD_DATE}",
  GIT_COMMIT: "${GIT_COMMIT}"
};
EOF

# Update index.html to include config.js
if ! grep -q "config.js" /usr/share/nginx/html/index.html; then
  sed -i 's|</head>|<script src="/config.js"></script></head>|' /usr/share/nginx/html/index.html
fi

echo "Runtime configuration completed"
