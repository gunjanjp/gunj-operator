#!/bin/bash
# Setup development environment on Docker Desktop Kubernetes

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "🚀 Setting up Gunj Operator development environment on Docker Desktop..."

# Verify Docker Desktop K8s is running
"${SCRIPT_DIR}/verify-docker-desktop.sh"

echo ""
echo "📦 Installing required components..."

# Install NGINX Ingress Controller
echo "🔧 Installing NGINX Ingress Controller..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml

# Wait for ingress controller to be ready
echo "⏳ Waiting for NGINX Ingress Controller to be ready..."
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s

# Install metrics-server for resource monitoring
echo "📊 Installing metrics-server..."
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Patch metrics-server for Docker Desktop
kubectl patch deployment metrics-server -n kube-system --type='json' -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/containers/0/args/-",
    "value": "--kubelet-insecure-tls"
  }
]'

# Create development namespaces
echo "🏗️ Creating development namespaces..."
kubectl create namespace gunj-operator-system --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace gunj-operator-dev --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace gunj-operator-test --dry-run=client -o yaml | kubectl apply -f -

# Label namespaces
kubectl label namespace gunj-operator-system environment=development --overwrite
kubectl label namespace gunj-operator-dev environment=development --overwrite
kubectl label namespace gunj-operator-test environment=test --overwrite

# Create local storage class (if not exists)
echo "💾 Setting up storage class..."
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: local-storage
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
EOF

# Install cert-manager for webhook certificates
echo "🔐 Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

# Wait for cert-manager to be ready
echo "⏳ Waiting for cert-manager to be ready..."
kubectl wait --namespace cert-manager \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/instance=cert-manager \
  --timeout=120s

# Create development secrets
echo "🔑 Creating development secrets..."
kubectl create secret generic gunj-operator-webhook-tls \
  --from-literal=tls.crt="" \
  --from-literal=tls.key="" \
  --namespace gunj-operator-system \
  --dry-run=client -o yaml | kubectl apply -f -

# Setup local registry (optional)
echo "🐳 Setting up local registry..."
kubectl create namespace local-registry --dry-run=client -o yaml | kubectl apply -f -
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: local-registry
  namespace: local-registry
spec:
  type: NodePort
  ports:
  - port: 5000
    nodePort: 30500
    targetPort: 5000
  selector:
    app: local-registry
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: local-registry
  namespace: local-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: local-registry
  template:
    metadata:
      labels:
        app: local-registry
    spec:
      containers:
      - name: registry
        image: registry:2
        ports:
        - containerPort: 5000
        volumeMounts:
        - name: registry-storage
          mountPath: /var/lib/registry
      volumes:
      - name: registry-storage
        emptyDir: {}
EOF

echo ""
echo "✅ Development environment setup complete!"
echo ""
echo "📋 Summary:"
echo "  - NGINX Ingress Controller: Installed"
echo "  - Metrics Server: Installed"
echo "  - Cert Manager: Installed"
echo "  - Development Namespaces: Created"
echo "  - Local Registry: Available at localhost:30500"
echo ""
echo "🔧 Next steps:"
echo "  1. Build operator image: make docker-build"
echo "  2. Deploy operator: make deploy"
echo "  3. Access UI: http://localhost"
