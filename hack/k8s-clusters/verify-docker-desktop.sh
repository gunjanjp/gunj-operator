#!/bin/bash
# Script to verify Docker Desktop Kubernetes is running

set -e

echo "🔍 Verifying Docker Desktop Kubernetes..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl."
    exit 1
fi

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker Desktop."
    exit 1
fi

# Check Kubernetes context
CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "none")
echo "📍 Current context: $CURRENT_CONTEXT"

# Check if docker-desktop context exists
if kubectl config get-contexts docker-desktop &> /dev/null; then
    echo "✅ docker-desktop context found"
    
    # Switch to docker-desktop if not already
    if [ "$CURRENT_CONTEXT" != "docker-desktop" ]; then
        echo "🔄 Switching to docker-desktop context..."
        kubectl config use-context docker-desktop
    fi
else
    echo "❌ docker-desktop context not found. Please enable Kubernetes in Docker Desktop settings."
    exit 1
fi

# Verify cluster is accessible
if kubectl cluster-info &> /dev/null; then
    echo "✅ Kubernetes cluster is accessible"
    kubectl cluster-info
else
    echo "❌ Cannot access Kubernetes cluster"
    exit 1
fi

# Check cluster version
echo ""
echo "📊 Cluster information:"
kubectl version --short
echo ""
echo "🔧 Nodes:"
kubectl get nodes
echo ""
echo "📦 Namespaces:"
kubectl get namespaces

echo ""
echo "✅ Docker Desktop Kubernetes is ready for development!"
