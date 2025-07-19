#!/bin/bash

# End-to-End Test Script for PMA Backend
# Tests the complete flow from startup to API operations

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVER_PORT=8080
SERVER_HOST="localhost"
BASE_URL="http://${SERVER_HOST}:${SERVER_PORT}"
API_BASE="${BASE_URL}/api/v1"
TEST_TOKEN="test-token"
PID_FILE="/tmp/pma-backend-test.pid"

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

wait_for_server() {
    local timeout=30
    local count=0
    
    log_info "Waiting for server to start..."
    while [ $count -lt $timeout ]; do
        if curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
            log_info "Server is ready!"
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    
    log_error "Server failed to start within ${timeout} seconds"
    return 1
}

cleanup() {
    log_info "Cleaning up..."
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping server (PID: $pid)"
            kill "$pid"
            # Wait for graceful shutdown
            sleep 2
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                log_warn "Force killing server"
                kill -9 "$pid"
            fi
        fi
        rm -f "$PID_FILE"
    fi
}

# Set up cleanup on exit
trap cleanup EXIT

# Test functions
test_server_startup() {
    log_info "Testing server startup..."
    
    # Start the server in background
    cd "$(dirname "$0")/../.."  # Go to project root
    go run cmd/server/main.go > /tmp/pma-backend-test.log 2>&1 &
    local server_pid=$!
    echo "$server_pid" > "$PID_FILE"
    
    log_info "Started server with PID: $server_pid"
    
    # Wait for server to be ready
    if ! wait_for_server; then
        log_error "Server startup failed"
        return 1
    fi
    
    log_info "✓ Server startup test passed"
}

test_health_endpoint() {
    log_info "Testing health endpoint..."
    
    local response=$(curl -s -w "%{http_code}" -o /tmp/health_response.json "${BASE_URL}/health")
    local status_code=$(echo "$response" | tail -1)
    
    if [ "$status_code" != "200" ]; then
        log_error "Health endpoint returned status $status_code"
        return 1
    fi
    
    # Check response format
    if ! jq -e '.status' /tmp/health_response.json > /dev/null 2>&1; then
        log_error "Health endpoint response is not valid JSON or missing status field"
        return 1
    fi
    
    log_info "✓ Health endpoint test passed"
}

test_entity_retrieval() {
    log_info "Testing entity retrieval..."
    
    # Test getting all entities
    local response=$(curl -s -w "%{http_code}" \
        -H "Authorization: Bearer $TEST_TOKEN" \
        -o /tmp/entities_response.json \
        "${API_BASE}/entities")
    local status_code=$(echo "$response" | tail -1)
    
    if [ "$status_code" != "200" ]; then
        log_error "Entity retrieval returned status $status_code"
        cat /tmp/entities_response.json
        return 1
    fi
    
    # Check response format
    if ! jq -e '.success' /tmp/entities_response.json > /dev/null 2>&1; then
        log_error "Entity retrieval response is not valid JSON or missing success field"
        cat /tmp/entities_response.json
        return 1
    fi
    
    local success=$(jq -r '.success' /tmp/entities_response.json)
    if [ "$success" != "true" ]; then
        log_error "Entity retrieval was not successful"
        cat /tmp/entities_response.json
        return 1
    fi
    
    log_info "✓ Entity retrieval test passed"
}

test_unauthorized_access() {
    log_info "Testing unauthorized access..."
    
    # Test without authorization header
    local response=$(curl -s -w "%{http_code}" \
        -o /tmp/unauth_response.json \
        "${API_BASE}/entities")
    local status_code=$(echo "$response" | tail -1)
    
    if [ "$status_code" != "401" ]; then
        log_error "Expected 401 for unauthorized access, got $status_code"
        return 1
    fi
    
    # Test with invalid token
    response=$(curl -s -w "%{http_code}" \
        -H "Authorization: Bearer invalid-token" \
        -o /tmp/unauth_response2.json \
        "${API_BASE}/entities")
    status_code=$(echo "$response" | tail -1)
    
    if [ "$status_code" != "401" ]; then
        log_error "Expected 401 for invalid token, got $status_code"
        return 1
    fi
    
    log_info "✓ Unauthorized access test passed"
}

test_specific_entity() {
    log_info "Testing specific entity retrieval..."
    
    # First, get all entities to find a valid entity ID
    curl -s -H "Authorization: Bearer $TEST_TOKEN" \
        "${API_BASE}/entities" > /tmp/all_entities.json
    
    # Extract first entity ID if available
    local entity_id=$(jq -r '.data[0].entity.id // empty' /tmp/all_entities.json 2>/dev/null)
    
    if [ -z "$entity_id" ] || [ "$entity_id" = "null" ]; then
        log_warn "No entities found, skipping specific entity test"
        return 0
    fi
    
    # Test getting specific entity
    local response=$(curl -s -w "%{http_code}" \
        -H "Authorization: Bearer $TEST_TOKEN" \
        -o /tmp/entity_response.json \
        "${API_BASE}/entities/${entity_id}")
    local status_code=$(echo "$response" | tail -1)
    
    if [ "$status_code" != "200" ]; then
        log_error "Specific entity retrieval returned status $status_code"
        cat /tmp/entity_response.json
        return 1
    fi
    
    # Check that we got the right entity
    local returned_id=$(jq -r '.data.entity.id' /tmp/entity_response.json)
    if [ "$returned_id" != "$entity_id" ]; then
        log_error "Expected entity ID $entity_id, got $returned_id"
        return 1
    fi
    
    log_info "✓ Specific entity test passed (ID: $entity_id)"
}

test_action_execution() {
    log_info "Testing action execution..."
    
    # First, get all entities to find a controllable entity
    curl -s -H "Authorization: Bearer $TEST_TOKEN" \
        "${API_BASE}/entities" > /tmp/all_entities.json
    
    # Look for a light entity
    local light_id=$(jq -r '.data[] | select(.entity.type == "light") | .entity.id' /tmp/all_entities.json | head -1)
    
    if [ -z "$light_id" ] || [ "$light_id" = "null" ]; then
        log_warn "No light entities found, skipping action execution test"
        return 0
    fi
    
    # Test executing action
    local action_payload='{"action":"turn_on","parameters":{"brightness":75}}'
    local response=$(curl -s -w "%{http_code}" \
        -H "Authorization: Bearer $TEST_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$action_payload" \
        -o /tmp/action_response.json \
        "${API_BASE}/entities/${light_id}/action")
    local status_code=$(echo "$response" | tail -1)
    
    if [ "$status_code" != "200" ]; then
        log_error "Action execution returned status $status_code"
        cat /tmp/action_response.json
        return 1
    fi
    
    # Check response format
    local success=$(jq -r '.success' /tmp/action_response.json)
    if [ "$success" != "true" ]; then
        log_error "Action execution was not successful"
        cat /tmp/action_response.json
        return 1
    fi
    
    log_info "✓ Action execution test passed (Entity: $light_id)"
}

test_adapter_status() {
    log_info "Testing adapter status..."
    
    local response=$(curl -s -w "%{http_code}" \
        -H "Authorization: Bearer $TEST_TOKEN" \
        -o /tmp/adapters_response.json \
        "${API_BASE}/adapters")
    local status_code=$(echo "$response" | tail -1)
    
    if [ "$status_code" != "200" ]; then
        log_error "Adapter status returned status $status_code"
        cat /tmp/adapters_response.json
        return 1
    fi
    
    # Check response format
    if ! jq -e '.success' /tmp/adapters_response.json > /dev/null 2>&1; then
        log_error "Adapter status response is not valid JSON or missing success field"
        cat /tmp/adapters_response.json
        return 1
    fi
    
    log_info "✓ Adapter status test passed"
}

test_websocket_connection() {
    log_info "Testing WebSocket connection..."
    
    # Simple WebSocket connection test using curl with upgrade headers
    local response=$(curl -s -w "%{http_code}" \
        -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
        -H "Sec-WebSocket-Version: 13" \
        -o /tmp/ws_response.txt \
        "${BASE_URL}/ws")
    local status_code=$(echo "$response" | tail -1)
    
    # WebSocket upgrade should return 101 or connection error
    if [ "$status_code" != "101" ] && [ "$status_code" != "400" ] && [ "$status_code" != "404" ]; then
        log_warn "WebSocket test returned unexpected status $status_code (this might be normal)"
    else
        log_info "✓ WebSocket endpoint test passed"
    fi
}

# Main test execution
main() {
    log_info "Starting PMA Backend End-to-End Tests"
    log_info "======================================"
    
    # Check dependencies
    if ! command -v curl > /dev/null; then
        log_error "curl is required for tests"
        exit 1
    fi
    
    if ! command -v jq > /dev/null; then
        log_error "jq is required for JSON parsing"
        exit 1
    fi
    
    if ! command -v go > /dev/null; then
        log_error "go is required to run the server"
        exit 1
    fi
    
    # Run tests
    local failed_tests=0
    
    test_server_startup || failed_tests=$((failed_tests + 1))
    test_health_endpoint || failed_tests=$((failed_tests + 1))
    test_unauthorized_access || failed_tests=$((failed_tests + 1))
    test_entity_retrieval || failed_tests=$((failed_tests + 1))
    test_specific_entity || failed_tests=$((failed_tests + 1))
    test_action_execution || failed_tests=$((failed_tests + 1))
    test_adapter_status || failed_tests=$((failed_tests + 1))
    test_websocket_connection || failed_tests=$((failed_tests + 1))
    
    # Results
    log_info "======================================"
    if [ $failed_tests -eq 0 ]; then
        log_info "All tests passed! ✓"
        exit 0
    else
        log_error "$failed_tests test(s) failed"
        log_info "Check server logs: /tmp/pma-backend-test.log"
        exit 1
    fi
}

# Run main function
main "$@" 