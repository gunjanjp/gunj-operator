# validate-examples.ps1 - PowerShell version of the validation script
# Validates all example CR manifests
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ExamplesDir = $ScriptDir
$FailedFiles = @()
$PassedFiles = @()
Write-Host "≡ƒöì Validating ObservabilityPlatform example manifests..." -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
function Validate-Yaml {
    param($File)
    Write-Host -NoNewline "Checking $(Split-Path -Leaf $File)... "
    if (-not (Test-Path $File)) {
        Write-Host "Γ£ù File not found" -ForegroundColor Red
        $script:FailedFiles += $File
        return $false
    }
    try {
        # Basic validation using kubectl
        $output = kubectl apply --dry-run=client -f $File 2>&1
        if ($LASTEXITCODE -eq 0) {
            # Check for required fields
            $content = Get-Content $File -Raw
            if ($content -match "apiVersion: observability.io/v1beta1" -and
                $content -match "kind: ObservabilityPlatform" -and
                $content -match "metadata:" -and
                $content -match "spec:") {
                Write-Host "Γ£ô Valid" -ForegroundColor Green
                $script:PassedFiles += $File
                return $true
            } else {
                Write-Host "Γ£ù Missing required fields" -ForegroundColor Red
                $script:FailedFiles += $File
                return $false
            }
        } else {
            Write-Host "Γ£ù Invalid YAML syntax" -ForegroundColor Red
            Write-Host $output -ForegroundColor Yellow
            $script:FailedFiles += $File
            return $false
        }
    } catch {
        Write-Host "Γ£ù Error: $_" -ForegroundColor Red
        $script:FailedFiles += $File
        return $false
    }
}
# Find all YAML files
Write-Host "Finding example files..."
$YamlFiles = Get-ChildItem -Path $ExamplesDir -Recurse -Include "*.yaml", "*.yml" | 
             Where-Object { $_.DirectoryName -notmatch "node_modules" } | 
             Sort-Object FullName
Write-Host "Found $($YamlFiles.Count) YAML files to validate"
Write-Host ""
# Validate each file
foreach ($file in $YamlFiles) {
    Validate-Yaml -File $file.FullName
}
Write-Host ""
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host "Validation Summary:"
Write-Host "  Total files: $($YamlFiles.Count)"
Write-Host "  Passed: $($PassedFiles.Count)" -ForegroundColor Green
Write-Host "  Failed: $($FailedFiles.Count)" -ForegroundColor Red
if ($FailedFiles.Count -gt 0) {
    Write-Host ""
    Write-Host "Failed files:" -ForegroundColor Red
    foreach ($file in $FailedFiles) {
        Write-Host "  - $file"
    }
    exit 1
} else {
    Write-Host ""
    Write-Host "Γ£à All example manifests are valid!" -ForegroundColor Green
}
# Check if CRD is installed
try {
    kubectl get crd observabilityplatforms.observability.io 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "≡ƒôï CRD is installed. Running additional validation..." -ForegroundColor Cyan
        foreach ($file in $PassedFiles) {
            Write-Host -NoNewline "  Validating against CRD schema: $(Split-Path -Leaf $file)... "
            $output = kubectl apply --dry-run=server -f $file 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host "Γ£ô" -ForegroundColor Green
            } else {
                Write-Host "ΓÜá Schema validation failed" -ForegroundColor Yellow
            }
        }
    }
} catch {
    Write-Host ""
    Write-Host "Γä╣∩╕Å  CRD not installed. Skipping schema validation." -ForegroundColor Yellow
    Write-Host "   Install the operator to enable full validation."
}
