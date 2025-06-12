# Create main directory structure for Gunj Operator monorepo
Write-Host "Setting up Gunj Operator monorepo directory structure..."
# Define all directories
$directories = @(
    "api\v1beta1",
    "cmd\operator",
    "cmd\api-server",
    "cmd\cli",
    "config\crd\bases",
    "config\default",
    "config\manager",
    "config\prometheus",
    "config\rbac",
    "config\webhook",
    "controllers",
    "internal\api\handlers",
    "internal\api\middleware",
    "internal\api\graphql",
    "internal\managers",
    "internal\metrics",
    "internal\utils",
    "internal\webhooks",
    "pkg\apis",
    "pkg\client",
    "pkg\sdk",
    "ui\public",
    "ui\src\components",
    "ui\src\pages",
    "ui\src\hooks",
    "ui\src\store",
    "ui\src\api",
    "ui\src\utils",
    "ui\tests",
    "hack",
    "test\e2e",
    "test\integration",
    "test\performance",
    "docs\api",
    "docs\development",
    "docs\user",
    "docs\architecture",
    "examples",
    "charts\gunj-operator\templates",
    ".github\workflows",
    ".github\ISSUE_TEMPLATE",
    ".github\PULL_REQUEST_TEMPLATE"
)
# Create each directory
foreach ($dir in $directories) {
    $fullPath = Join-Path -Path $PSScriptRoot -ChildPath $dir
    if (!(Test-Path -Path $fullPath)) {
        New-Item -ItemType Directory -Path $fullPath -Force | Out-Null
        Write-Host "Created: $dir"
    }
}
Write-Host "Directory structure created successfully!"
