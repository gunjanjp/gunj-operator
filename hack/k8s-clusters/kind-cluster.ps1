# PowerShell script for Kind cluster management on Windows

param(
    [Parameter(Mandatory=$true, Position=0)]
    [ValidateSet("create", "delete", "list", "switch", "setup")]
    [string]$Command,
    
    [Parameter(Position=1)]
    [ValidateSet("dev", "ha", "ci")]
    [string]$ClusterType
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Colors
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Error { Write-Host $args -ForegroundColor Red }
function Write-Info { Write-Host $args -ForegroundColor Blue }
function Write-Warning { Write-Host $args -ForegroundColor Yellow }

# Check if kind is installed
function Test-Kind {
    try {
        kind version | Out-Null
        return $true
    } catch {
        return $false
    }
}

# Install kind if not present
function Install-Kind {
    if (-not (Test-Kind)) {
        Write-Error "❌ kind not found. Installing..."
        
        # Download kind for Windows
        $kindUrl = "https://kind.sigs.k8s.io/dl/v0.20.0/kind-windows-amd64"
        $kindPath = "$env:USERPROFILE\kind.exe"
        
        Write-Info "Downloading kind..."
        Invoke-WebRequest -Uri $kindUrl -OutFile $kindPath
        
        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$env:USERPROFILE*") {
            [Environment]::SetEnvironmentVariable("Path", "$userPath;$env:USERPROFILE", "User")
            Write-Warning "Added kind to PATH. Please restart PowerShell for changes to take effect."
        }
        
        Write-Success "✅ kind installed successfully!"
    }
}

# Create cluster
function New-Cluster {
    param([string]$Type)
    
    $configFile = Join-Path $ScriptDir "kind-$Type.yaml"
    
    if (-not (Test-Path $configFile)) {
        Write-Error "❌ Configuration file not found: $configFile"
        exit 1
    }
    
    Write-Info "🚀 Creating kind cluster: gunj-operator-$Type"
    
    # Check if cluster already exists
    $clusters = kind get clusters 2>$null
    if ($clusters -contains "gunj-operator-$Type") {
        Write-Warning "⚠️  Cluster already exists. Delete it first with: .\kind-cluster.ps1 delete $Type"
        exit 1
    }
    
    # Create cluster
    kind create cluster --config="$configFile"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "✅ Cluster created successfully!"
        Setup-Cluster -Type $Type
    } else {
        Write-Error "❌ Failed to create cluster"
        exit 1
    }
}

# Delete cluster
function Remove-Cluster {
    param([string]$Type)
    
    $clusterName = "gunj-operator-$Type"
    Write-Info "🗑️  Deleting kind cluster: $clusterName"
    
    $clusters = kind get clusters 2>$null
    if ($clusters -contains $clusterName) {
        kind delete cluster --name="$clusterName"
        Write-Success "✅ Cluster deleted successfully!"
    } else {
        Write-Warning "⚠️  Cluster not found: $clusterName"
    }
}

# List clusters
function Get-Clusters {
    Write-Info "📋 Kind clusters:"
    kind get clusters
    
    Write-Info "`n📍 Current kubectl context:"
    kubectl config current-context
}

# Switch cluster context
function Switch-Cluster {
    param([string]$Type)
    
    $context = "kind-gunj-operator-$Type"
    Write-Info "🔄 Switching to cluster: $context"
    
    $contexts = kubectl config get-contexts -o name
    if ($contexts -contains $context) {
        kubectl config use-context $context
        Write-Success "✅ Switched to $context"
    } else {
        Write-Error "❌ Context not found: $context"
        Write-Info "Available contexts:"
        $contexts | Where-Object { $_ -like "kind-*" }
    }
}

# Setup cluster with required components
function Setup-Cluster {
    param([string]$Type)
    
    $context = "kind-gunj-operator-$Type"
    Write-Info "🔧 Setting up cluster: $context"
    
    # Switch to cluster context
    kubectl config use-context $context
    
    # Install NGINX Ingress
    Write-Info "📦 Installing NGINX Ingress..."
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    
    # Wait for ingress
    Write-Info "⏳ Waiting for ingress controller..."
    kubectl wait --namespace ingress-nginx `
        --for=condition=ready pod `
        --selector=app.kubernetes.io/component=controller `
        --timeout=90s
    
    # Install metrics-server
    Write-Info "📊 Installing metrics-server..."
    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
    
    # Patch metrics-server for kind
    $patch = @'
[
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
]
'@
    
    kubectl patch deployment metrics-server -n kube-system --type='json' -p=$patch
    
    # Create namespaces
    Write-Info "🏗️  Creating namespaces..."
    kubectl create namespace gunj-operator-system --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace gunj-operator-dev --dry-run=client -o yaml | kubectl apply -f -
    
    # Setup local registry for dev cluster
    if ($Type -eq "dev") {
        Write-Info "🐳 Setting up local registry..."
        
        # Create registry ConfigMap
        $registryConfig = @"
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:5000"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
"@
        
        $registryConfig | kubectl apply -f -
    }
    
    Write-Success "✅ Cluster setup complete!"
}

# Main execution
Install-Kind

switch ($Command) {
    "create" {
        if (-not $ClusterType) {
            Write-Error "❌ Please specify cluster type (dev|ha|ci)"
            exit 1
        }
        New-Cluster -Type $ClusterType
    }
    "delete" {
        if (-not $ClusterType) {
            Write-Error "❌ Please specify cluster type (dev|ha|ci)"
            exit 1
        }
        Remove-Cluster -Type $ClusterType
    }
    "list" {
        Get-Clusters
    }
    "switch" {
        if (-not $ClusterType) {
            Write-Error "❌ Please specify cluster type (dev|ha|ci)"
            exit 1
        }
        Switch-Cluster -Type $ClusterType
    }
    "setup" {
        if (-not $ClusterType) {
            Write-Error "❌ Please specify cluster type (dev|ha|ci)"
            exit 1
        }
        Setup-Cluster -Type $ClusterType
    }
}
