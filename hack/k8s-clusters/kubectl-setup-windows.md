# Setting up kubectl on Windows

Since you have Docker Desktop installed, kubectl should be available but may not be in your PATH.

## Option 1: Add kubectl to PATH (Recommended)

1. Open System Properties → Environment Variables
2. Add to PATH: `C:\Program Files\Docker\Docker\resources\bin`
3. Restart PowerShell/Terminal

## Option 2: Create kubectl alias in PowerShell

Add this to your PowerShell profile:

```powershell
# Open PowerShell profile
notepad $PROFILE

# Add this line:
Set-Alias kubectl "C:\Program Files\Docker\Docker\resources\bin\kubectl.exe"

# Reload profile
. $PROFILE
```

## Option 3: Use kubectl directly

For now, you can use the full path:

```powershell
& "C:\Program Files\Docker\Docker\resources\bin\kubectl.exe" get nodes
```

## Verify Kubernetes is Enabled

1. Open Docker Desktop
2. Go to Settings → Kubernetes
3. Check "Enable Kubernetes"
4. Click "Apply & Restart"

## Test kubectl

After setup, test with:

```bash
kubectl version --client
kubectl cluster-info
kubectl get nodes
```

You should see your docker-desktop node running.
