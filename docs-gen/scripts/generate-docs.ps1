# Documentation generation script for Gunj Operator (Windows)

param(
    [string]$Command = "all"
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
$DocsGenDir = Join-Path $ProjectRoot "docs-gen"
$OutputDir = Join-Path $DocsGenDir "output"

# Helper functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Check required tools
function Test-Requirements {
    Write-Info "Checking requirements..."
    
    $requirements = @{
        "go" = "Go compiler"
        "node" = "Node.js runtime"
        "npm" = "Node package manager"
        "python" = "Python 3"
        "pip" = "Python package manager"
    }
    
    $allFound = $true
    foreach ($cmd in $requirements.Keys) {
        $found = Get-Command $cmd -ErrorAction SilentlyContinue
        if (-not $found) {
            Write-Error "$($requirements[$cmd]) ($cmd) is not installed"
            $allFound = $false
        }
    }
    
    if (-not $allFound) {
        exit 1
    }
    
    Write-Info "All requirements satisfied"
}

# Install documentation tools
function Install-Tools {
    Write-Info "Installing documentation tools..."
    
    # Install MkDocs and plugins
    Write-Info "Installing MkDocs..."
    pip install --user mkdocs mkdocs-material mkdocs-minify-plugin `
        mkdocs-redirects mkdocs-git-revision-date-localized-plugin `
        mike pymdown-extensions
    
    # Install TypeDoc
    Write-Info "Installing TypeDoc..."
    Push-Location "$ProjectRoot\ui"
    npm install --save-dev typedoc typedoc-plugin-markdown typedoc-plugin-missing-exports
    Pop-Location
    
    # Install godoc
    Write-Info "Installing godoc..."
    go install golang.org/x/tools/cmd/godoc@latest
    
    # Install Swagger CLI
    Write-Info "Installing Swagger CLI..."
    npm install -g @apidevtools/swagger-cli
    
    Write-Info "Documentation tools installed"
}

# Generate Go documentation
function New-GoDocs {
    Write-Info "Generating Go documentation..."
    
    New-Item -ItemType Directory -Force -Path "$OutputDir\go" | Out-Null
    
    # Start godoc server
    Push-Location $ProjectRoot
    $godocProcess = Start-Process -FilePath "godoc" -ArgumentList "-http=:6060" -PassThru
    Start-Sleep -Seconds 5
    
    # Download package documentation
    $url = "http://localhost:6060/pkg/github.com/gunjanjp/gunj-operator/"
    $output = "$OutputDir\go"
    
    # Use Invoke-WebRequest to download the documentation
    try {
        Invoke-WebRequest -Uri $url -OutFile "$output\index.html"
    } catch {
        Write-Warn "Could not download Go documentation"
    }
    
    # Stop godoc server
    Stop-Process -Id $godocProcess.Id -Force
    Pop-Location
    
    Write-Info "Go documentation generated"
}

# Generate TypeScript documentation
function New-TypeScriptDocs {
    Write-Info "Generating TypeScript documentation..."
    
    Push-Location $DocsGenDir
    npx typedoc
    Pop-Location
    
    Write-Info "TypeScript documentation generated"
}

# Generate API documentation
function New-ApiDocs {
    Write-Info "Generating API documentation..."
    
    New-Item -ItemType Directory -Force -Path "$OutputDir\api" | Out-Null
    
    # Validate OpenAPI spec
    $openApiPath = Join-Path $ProjectRoot "api\openapi.yaml"
    if (Test-Path $openApiPath) {
        swagger-cli validate $openApiPath
        
        # Generate HTML documentation
        npx @redocly/openapi-cli build-docs `
            $openApiPath `
            -o "$OutputDir\api\rest.html"
    } else {
        Write-Warn "OpenAPI spec not found"
    }
    
    # Generate GraphQL documentation if schema exists
    $graphqlPath = Join-Path $ProjectRoot "api\schema.graphql"
    if (Test-Path $graphqlPath) {
        npx spectaql $graphqlPath `
            -t "$OutputDir\api\graphql"
    } else {
        Write-Warn "GraphQL schema not found"
    }
    
    Write-Info "API documentation generated"
}

# Build MkDocs site
function Build-MkDocs {
    Write-Info "Building MkDocs site..."
    
    Push-Location $DocsGenDir
    
    # Copy documentation files
    $siteContent = Join-Path $DocsGenDir "site_content"
    New-Item -ItemType Directory -Force -Path $siteContent | Out-Null
    Copy-Item -Path "$ProjectRoot\docs\*" -Destination $siteContent -Recurse -Force
    
    # Build the site
    mkdocs build -d "$OutputDir\site"
    
    Pop-Location
    
    Write-Info "MkDocs site built"
}

# Generate architecture diagrams
function New-Diagrams {
    Write-Info "Generating architecture diagrams..."
    
    New-Item -ItemType Directory -Force -Path "$OutputDir\diagrams" | Out-Null
    
    # Generate PlantUML diagrams if available
    $plantuml = Get-Command plantuml -ErrorAction SilentlyContinue
    if ($plantuml) {
        Get-ChildItem -Path "$ProjectRoot\docs" -Filter "*.puml" -Recurse | ForEach-Object {
            plantuml -tsvg -o "$OutputDir\diagrams" $_.FullName
        }
    } else {
        Write-Warn "PlantUML not installed, skipping diagram generation"
    }
    
    Write-Info "Architecture diagrams generated"
}

# Create documentation index
function New-Index {
    Write-Info "Creating documentation index..."
    
    $indexHtml = @"
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gunj Operator Documentation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 { color: #2c3e50; }
        .section {
            background: #f4f4f4;
            padding: 20px;
            margin: 20px 0;
            border-radius: 8px;
        }
        .section h2 { margin-top: 0; }
        .links { display: flex; flex-wrap: wrap; gap: 10px; }
        .link {
            background: #3498db;
            color: white;
            padding: 10px 20px;
            text-decoration: none;
            border-radius: 5px;
            transition: background 0.3s;
        }
        .link:hover { background: #2980b9; }
    </style>
</head>
<body>
    <h1>Gunj Operator Documentation</h1>
    
    <div class="section">
        <h2>User Documentation</h2>
        <div class="links">
            <a href="site/index.html" class="link">User Guide</a>
            <a href="site/getting-started/index.html" class="link">Getting Started</a>
            <a href="site/api/index.html" class="link">API Reference</a>
        </div>
    </div>
    
    <div class="section">
        <h2>Developer Documentation</h2>
        <div class="links">
            <a href="go/index.html" class="link">Go Documentation</a>
            <a href="typescript/index.html" class="link">TypeScript Documentation</a>
            <a href="api/rest.html" class="link">REST API</a>
            <a href="api/graphql/index.html" class="link">GraphQL API</a>
        </div>
    </div>
    
    <div class="section">
        <h2>Architecture</h2>
        <div class="links">
            <a href="site/architecture/index.html" class="link">Architecture Overview</a>
            <a href="diagrams/index.html" class="link">Architecture Diagrams</a>
        </div>
    </div>
    
    <div class="section">
        <h2>External Links</h2>
        <div class="links">
            <a href="https://github.com/gunjanjp/gunj-operator" class="link">GitHub Repository</a>
            <a href="https://github.com/gunjanjp/gunj-operator/issues" class="link">Issue Tracker</a>
            <a href="https://github.com/gunjanjp/gunj-operator/discussions" class="link">Discussions</a>
        </div>
    </div>
</body>
</html>
"@
    
    Set-Content -Path "$OutputDir\index.html" -Value $indexHtml
    
    Write-Info "Documentation index created"
}

# Main execution
function Main {
    Write-Info "Starting documentation generation..."
    
    # Check requirements
    Test-Requirements
    
    switch ($Command) {
        "install" {
            Install-Tools
        }
        "go" {
            New-GoDocs
        }
        "typescript" {
            New-TypeScriptDocs
        }
        "api" {
            New-ApiDocs
        }
        "mkdocs" {
            Build-MkDocs
        }
        "diagrams" {
            New-Diagrams
        }
        "all" {
            # Clean output directory
            if (Test-Path $OutputDir) {
                Remove-Item -Path $OutputDir -Recurse -Force
            }
            New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
            
            # Generate all documentation
            New-GoDocs
            New-TypeScriptDocs
            New-ApiDocs
            Build-MkDocs
            New-Diagrams
            New-Index
            
            Write-Info "All documentation generated successfully!"
            Write-Info "Output directory: $OutputDir"
        }
        default {
            Write-Host "Usage: .\generate-docs.ps1 [-Command <install|go|typescript|api|mkdocs|diagrams|all>]"
            exit 1
        }
    }
}

# Run main function
Main
