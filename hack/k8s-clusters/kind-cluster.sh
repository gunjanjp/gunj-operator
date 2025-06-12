#!/bin/bash
# Kind cluster management script

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

function print_usage() {
    echo "Usage: $0 <command> [cluster-type]"
    echo ""
    echo "Commands:"
    echo "  create <type>  - Create a kind cluster (dev|ha|ci)"
    echo "  delete <type>  - Delete a kind cluster"
    echo "  list          - List all kind clusters"
    echo "  switch <type> - Switch kubectl context to cluster"
    echo "  setup <type>  - Setup cluster with required components"
    echo ""
    echo "Cluster types:"
    echo "  dev  - Single node development cluster"
    echo "  ha   - Multi-node HA testing cluster"
    echo "  ci   - Minimal CI/CD cluster"
}

function check_kind() {
    if ! command -v kind &> /dev/null; then
        echo -e "${RED}❌ kind not found. Installing...${NC}"
        
        # Detect OS and install kind
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
            chmod +x ./kind
            sudo mv ./kind /usr/local/bin/kind
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            brew install kind
        elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
            curl.exe -Lo kind-windows-amd64.exe https://kind.sigs.k8s.io/dl/v0.20.0/kind-windows-amd64
            Move-Item .\kind-windows-amd64.exe c:\kind.exe -Force
        else
            echo -e "${RED}Unsupported OS. Please install kind manually.${NC}"
            exit 1
        fi
    fi
}

function create_cluster() {
    local cluster_type=$1
    local config_file="${SCRIPT_DIR}/kind-${cluster_type}.yaml"
    
    if [[ ! -f "$config_file" ]]; then
        echo -e "${RED}❌ Configuration file not found: $config_file${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}🚀 Creating kind cluster: gunj-operator-${cluster_type}${NC}"
    
    # Check if cluster already exists
    if kind get clusters | grep -q "gunj-operator-${cluster_type}"; then
        echo -e "${YELLOW}⚠️  Cluster already exists. Delete it first with: $0 delete ${cluster_type}${NC}"
        exit 1
    fi
    
    # Create cluster
    kind create cluster --config="$config_file"
    
    echo -e "${GREEN}✅ Cluster created successfully!${NC}"
    
    # Setup cluster
    setup_cluster "$cluster_type"
}

function delete_cluster() {
    local cluster_type=$1
    local cluster_name="gunj-operator-${cluster_type}"
    
    echo -e "${BLUE}🗑️  Deleting kind cluster: ${cluster_name}${NC}"
    
    if kind get clusters | grep -q "$cluster_name"; then
        kind delete cluster --name="$cluster_name"
        echo -e "${GREEN}✅ Cluster deleted successfully!${NC}"
    else
        echo -e "${YELLOW}⚠️  Cluster not found: ${cluster_name}${NC}"
    fi
}

function list_clusters() {
    echo -e "${BLUE}📋 Kind clusters:${NC}"
    kind get clusters
    echo ""
    echo -e "${BLUE}📍 Current kubectl context:${NC}"
    kubectl config current-context
}

function switch_cluster() {
    local cluster_type=$1
    local context="kind-gunj-operator-${cluster_type}"
    
    echo -e "${BLUE}🔄 Switching to cluster: ${context}${NC}"
    
    if kubectl config get-contexts -o name | grep -q "^${context}$"; then
        kubectl config use-context "$context"
        echo -e "${GREEN}✅ Switched to ${context}${NC}"
    else
        echo -e "${RED}❌ Context not found: ${context}${NC}"
        echo "Available contexts:"
        kubectl config get-contexts -o name | grep kind
    fi
}

function setup_cluster() {
    local cluster_type=$1
    local context="kind-gunj-operator-${cluster_type}"
    
    echo -e "${BLUE}🔧 Setting up cluster: ${context}${NC}"
    
    # Switch to cluster context
    kubectl config use-context "$context"
    
    # Install NGINX Ingress for kind
    echo "📦 Installing NGINX Ingress..."
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    
    # Wait for ingress
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=90s
    
    # Install metrics-server
    echo "📊 Installing metrics-server..."
    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
    
    # Patch metrics-server for kind
    kubectl patch deployment metrics-server -n kube-system --type='json' -p='[
      {
        "op": "add",
        "path": "/spec/template/spec/containers/0/args/-",
        "value": "--kubelet-insecure-tls"
      },
      {
        "op": "add", 
        "path": "/spec/template/spec/containers/0/args/-",
        "value": "--kubelet-preferred-address-types=InternalIP"
      }
    ]'
    
    # Create namespaces
    echo "🏗️  Creating namespaces..."
    kubectl create namespace gunj-operator-system --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace gunj-operator-dev --dry-run=client -o yaml | kubectl apply -f -
    
    # Setup local registry connection
    if [[ "$cluster_type" == "dev" ]]; then
        echo "🐳 Setting up local registry..."
        docker network connect "kind" "kind-registry" 2>/dev/null || true
        
        cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:5000"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
    fi
    
    echo -e "${GREEN}✅ Cluster setup complete!${NC}"
}

# Main script
check_kind

case "$1" in
    create)
        if [[ -z "$2" ]]; then
            echo -e "${RED}❌ Please specify cluster type (dev|ha|ci)${NC}"
            print_usage
            exit 1
        fi
        create_cluster "$2"
        ;;
    delete)
        if [[ -z "$2" ]]; then
            echo -e "${RED}❌ Please specify cluster type (dev|ha|ci)${NC}"
            print_usage
            exit 1
        fi
        delete_cluster "$2"
        ;;
    list)
        list_clusters
        ;;
    switch)
        if [[ -z "$2" ]]; then
            echo -e "${RED}❌ Please specify cluster type (dev|ha|ci)${NC}"
            print_usage
            exit 1
        fi
        switch_cluster "$2"
        ;;
    setup)
        if [[ -z "$2" ]]; then
            echo -e "${RED}❌ Please specify cluster type (dev|ha|ci)${NC}"
            print_usage
            exit 1
        fi
        setup_cluster "$2"
        ;;
    *)
        print_usage
        exit 1
        ;;
esac
