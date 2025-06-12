# Quick verification script for Docker Desktop Kubernetes
# This script uses kubectl from Docker Desktop installation

$kubectlPath = "C:\Program Files\Docker\Docker\resources\bin\kubectl.exe"

if (Test-Path $kubectlPath) {
    Write-Host "✅ kubectl found at Docker Desktop location" -ForegroundColor Green
    
    Write-Host "`n📍 Checking Kubernetes cluster..." -ForegroundColor Blue
    & $kubectlPath cluster-info
    
    Write-Host "`n🔧 Checking nodes..." -ForegroundColor Blue
    & $kubectlPath get nodes
    
    Write-Host "`n📦 Checking namespaces..." -ForegroundColor Blue
    & $kubectlPath get namespaces
    
} else {
    Write-Host "❌ kubectl not found at expected Docker Desktop location" -ForegroundColor Red
    Write-Host "Please ensure Docker Desktop is installed and Kubernetes is enabled" -ForegroundColor Yellow
}
