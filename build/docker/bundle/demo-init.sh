#!/bin/bash
# Demo initialization script for Gunj Operator Bundle
set -e

echo "==================================="
echo "Gunj Operator Demo Mode"
echo "==================================="
echo ""
echo "Initializing demo environment..."

# Source the main entrypoint
source /entrypoint.sh &

# Wait for services to start
sleep 10

# Function to create demo platform
create_demo_platform() {
    echo "Creating demo observability platform..."
    
    # Wait for API server
    wait_for_service "API Server" 8090
    
    # Create demo platform via API
    curl -X POST http://localhost:8090/api/v1/platforms \
        -H "Content-Type: application/json" \
        -d '{
            "name": "demo-platform",
            "namespace": "default",
            "spec": {
                "components": {
                    "prometheus": {
                        "enabled": true,
                        "version": "v2.48.0",
                        "replicas": 1,
                        "resources": {
                            "requests": {
                                "memory": "512Mi",
                                "cpu": "250m"
                            }
                        }
                    },
                    "grafana": {
                        "enabled": true,
                        "version": "10.2.0",
                        "adminPassword": "demo123"
                    },
                    "loki": {
                        "enabled": true,
                        "version": "2.9.0"
                    },
                    "tempo": {
                        "enabled": true,
                        "version": "2.3.0"
                    }
                }
            }
        }' || true
}

# Function to generate mock metrics
generate_mock_metrics() {
    echo "Generating mock metrics..."
    
    while true; do
        # Generate random metrics
        cpu_usage=$(awk -v min=10 -v max=90 'BEGIN{srand(); print int(min+rand()*(max-min+1))}')
        memory_usage=$(awk -v min=20 -v max=80 'BEGIN{srand(); print int(min+rand()*(max-min+1))}')
        request_rate=$(awk -v min=100 -v max=1000 'BEGIN{srand(); print int(min+rand()*(max-min+1))}')
        error_rate=$(awk -v min=0 -v max=5 'BEGIN{srand(); print int(min+rand()*(max-min+1))}')
        
        # Send metrics to API
        curl -X POST http://localhost:8090/api/v1/metrics/demo \
            -H "Content-Type: application/json" \
            -d "{
                \"cpu_usage\": $cpu_usage,
                \"memory_usage\": $memory_usage,
                \"request_rate\": $request_rate,
                \"error_rate\": $error_rate,
                \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
            }" 2>/dev/null || true
        
        sleep 30
    done
}

# Print demo information
print_demo_info() {
    echo ""
    echo "==================================="
    echo "Demo Mode Initialized!"
    echo "==================================="
    echo ""
    echo "Access the Gunj Operator UI at:"
    echo "  http://localhost:3000"
    echo ""
    echo "Default credentials:"
    echo "  Username: admin"
    echo "  Password: demo123"
    echo ""
    echo "API endpoints:"
    echo "  REST API: http://localhost:8090/api/v1"
    echo "  GraphQL: http://localhost:8090/graphql"
    echo "  Metrics: http://localhost:8080/metrics"
    echo ""
    echo "Demo features enabled:"
    echo "  - Sample observability platform"
    echo "  - Mock metrics generation"
    echo "  - Pre-configured dashboards"
    echo "  - Sample alerts"
    echo ""
    echo "To stop the demo:"
    echo "  docker stop gunj-operator-demo"
    echo ""
    echo "==================================="
}

# Initialize demo
if [ "$DEMO_MODE" = "true" ]; then
    # Create demo platform after a delay
    (sleep 30 && create_demo_platform) &
    
    # Start mock metrics generation
    if [ "$ENABLE_MOCK_METRICS" = "true" ]; then
        (sleep 60 && generate_mock_metrics) &
    fi
    
    # Print demo information
    (sleep 15 && print_demo_info) &
fi

# Keep the container running
tail -f /var/log/supervisor/*.log
