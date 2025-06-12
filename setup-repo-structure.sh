#!/bin/bash

# Create Gunj Operator repository structure
echo "🚀 Creating Gunj Operator repository structure..."

cd /mnt/d/claude/gunj-operator

# Create main directories
echo "📁 Creating main directories..."
mkdir -p {api,cmd,config,controllers,docs,hack,internal,pkg,test,ui,.github}

# API directory structure (CRD definitions)
echo "📦 Setting up API structure..."
mkdir -p api/v1beta1

# CMD directory structure (entry points)
echo "🔧 Setting up CMD structure..."
mkdir -p cmd/{operator,api-server,cli}

# Config directory structure (Kubernetes configurations)
echo "⚙️ Setting up Config structure..."
mkdir -p config/{crd,default,manager,prometheus,rbac,webhook,samples,certmanager,manifests}
mkdir -p config/crd/{bases,patches,kustomization}
mkdir -p config/rbac/{auth_proxy,leader_election}

# Controllers directory
echo "🎮 Setting up Controllers structure..."
mkdir -p controllers/component_managers

# Docs directory structure
echo "📚 Setting up Documentation structure..."
mkdir -p docs/{api,architecture,development,user,tutorials,images}

# Hack directory (development scripts)
echo "🛠️ Setting up Hack structure..."
mkdir -p hack

# Internal directory structure
echo "🏗️ Setting up Internal structure..."
mkdir -p internal/{api,managers,metrics,utils,webhooks,version}
mkdir -p internal/api/{handlers,middleware,graphql,validators}
mkdir -p internal/managers/{prometheus,grafana,loki,tempo}

# Pkg directory (public packages)
echo "📦 Setting up Pkg structure..."
mkdir -p pkg/{apis,client,sdk,constants,errors}

# Test directory structure
echo "🧪 Setting up Test structure..."
mkdir -p test/{e2e,integration,performance,fixtures}
mkdir -p test/e2e/{framework,scenarios}

# UI directory structure
echo "💻 Setting up UI structure..."
mkdir -p ui/{public,src,tests}
mkdir -p ui/src/{components,pages,hooks,store,api,utils,types,styles,assets}
mkdir -p ui/src/components/{common,platform,monitoring,settings}
mkdir -p ui/tests/{unit,integration,e2e}

# GitHub directory structure
echo "🐙 Setting up GitHub structure..."
mkdir -p .github/{workflows,ISSUE_TEMPLATE,PULL_REQUEST_TEMPLATE}

# Charts directory (Helm charts)
echo "⎈ Setting up Helm charts structure..."
mkdir -p charts/gunj-operator/{templates,crds}

# Additional project directories
echo "📂 Creating additional directories..."
mkdir -p {examples,scripts,build,deployments}

echo "✅ Directory structure created successfully!"
