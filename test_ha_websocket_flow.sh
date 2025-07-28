#!/bin/bash

# Test script for Home Assistant WebSocket integration flow
# This script tests the complete integration from HA WebSocket -> Unified Service -> Frontend WebSocket

echo "🧪 Home Assistant WebSocket Integration Flow Test"
echo "================================================"

# Configuration
PMA_BACKEND_URL="http://localhost:3000"
HA_URL="http://192.168.100.2:8123"
WEBSOCKET_URL="ws://localhost:3000/ws"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Test 1: Check if backend is running
test_backend_running() {
    log_info "Test 1: Checking if PMA backend is running..."
    
    if curl -s "$PMA_BACKEND_URL/api/v1/health" > /dev/null 2>&1; then
        log_success "PMA backend is running"
        return 0
    else
        log_error "PMA backend is not running. Please start it first."
        return 1
    fi
}

# Test 2: Check Home Assistant connectivity
test_ha_connectivity() {
    log_info "Test 2: Checking Home Assistant connectivity..."
    
    if curl -s "$HA_URL/api/" > /dev/null 2>&1; then
        log_success "Home Assistant is accessible"
        return 0
    else
        log_warning "Home Assistant is not accessible at $HA_URL"
        return 1
    fi
}

# Test 3: Test entity synchronization
test_entity_sync() {
    log_info "Test 3: Testing entity synchronization..."
    
    # Trigger manual sync
    SYNC_RESPONSE=$(curl -s -X POST "$PMA_BACKEND_URL/api/v1/entities/sync?source=homeassistant")
    
    if echo "$SYNC_RESPONSE" | grep -q "success"; then
        log_success "Entity sync completed successfully"
        
        # Check entity count
        ENTITY_COUNT=$(curl -s "$PMA_BACKEND_URL/api/v1/entities" | jq '.data | length' 2>/dev/null || echo "0")
        log_info "Total entities synchronized: $ENTITY_COUNT"
        
        if [ "$ENTITY_COUNT" -gt 0 ]; then
            log_success "Entities found and synchronized"
        else
            log_warning "No entities found after sync"
        fi
        return 0
    else
        log_error "Entity sync failed"
        echo "Response: $SYNC_RESPONSE"
        return 1
    fi
}

# Test 4: Test WebSocket connectivity
test_websocket_connectivity() {
    log_info "Test 4: Testing WebSocket connectivity..."
    
    # Create a simple WebSocket test client
    cat > /tmp/websocket_test.js << 'EOF'
const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:3000/ws');

ws.on('open', function open() {
    console.log('✅ WebSocket connection established');
    
    // Subscribe to entity state changes
    ws.send(JSON.stringify({
        type: 'subscribe',
        topic: 'entity_state_change'
    }));
    
    setTimeout(() => {
        console.log('ℹ️  WebSocket test completed');
        ws.close();
        process.exit(0);
    }, 2000);
});

ws.on('message', function message(data) {
    const msg = JSON.parse(data);
    console.log('📨 WebSocket message received:', msg.type);
});

ws.on('error', function error(err) {
    console.error('❌ WebSocket error:', err.message);
    process.exit(1);
});

ws.on('close', function close() {
    console.log('ℹ️  WebSocket connection closed');
});

// Timeout after 5 seconds
setTimeout(() => {
    console.error('❌ WebSocket connection timeout');
    process.exit(1);
}, 5000);
EOF

    if command -v node > /dev/null 2>&1; then
        if npm list ws > /dev/null 2>&1 || npm install ws > /dev/null 2>&1; then
            node /tmp/websocket_test.js
            return $?
        else
            log_warning "Cannot install ws package, skipping WebSocket test"
            return 0
        fi
    else
        log_warning "Node.js not available, skipping WebSocket test"
        return 0
    fi
}

# Test 5: Test Home Assistant adapter status
test_ha_adapter_status() {
    log_info "Test 5: Checking Home Assistant adapter status..."
    
    ADAPTER_STATUS=$(curl -s "$PMA_BACKEND_URL/api/v1/adapters/homeassistant/status")
    
    if echo "$ADAPTER_STATUS" | grep -q "connected.*true"; then
        log_success "Home Assistant adapter is connected"
        return 0
    else
        log_warning "Home Assistant adapter status unclear"
        echo "Response: $ADAPTER_STATUS"
        return 1
    fi
}

# Test 6: Test entity state change simulation
test_entity_state_change() {
    log_info "Test 6: Testing entity state change simulation..."
    
    # Get a test entity
    TEST_ENTITY=$(curl -s "$PMA_BACKEND_URL/api/v1/entities?type=switch&limit=1" | jq -r '.data[0].id // empty' 2>/dev/null)
    
    if [ -n "$TEST_ENTITY" ] && [ "$TEST_ENTITY" != "null" ]; then
        log_info "Testing with entity: $TEST_ENTITY"
        
        # Try to toggle the entity state
        TOGGLE_RESPONSE=$(curl -s -X POST "$PMA_BACKEND_URL/api/v1/entities/$TEST_ENTITY/action" \
            -H "Content-Type: application/json" \
            -d '{"action":"toggle"}')
        
        if echo "$TOGGLE_RESPONSE" | grep -q "success"; then
            log_success "Entity state change action completed"
            return 0
        else
            log_warning "Entity state change action failed or not supported"
            echo "Response: $TOGGLE_RESPONSE"
            return 1
        fi
    else
        log_warning "No test entity found for state change test"
        return 1
    fi
}

# Test 7: Test debug endpoints
test_debug_endpoints() {
    log_info "Test 7: Testing debug endpoints..."
    
    # Test entity registry debug
    REGISTRY_DEBUG=$(curl -s "$PMA_BACKEND_URL/api/v1/debug/entity-registry")
    
    if echo "$REGISTRY_DEBUG" | grep -q "total_entities"; then
        TOTAL_ENTITIES=$(echo "$REGISTRY_DEBUG" | jq -r '.total_entities // 0' 2>/dev/null)
        log_success "Entity registry debug endpoint working - Total entities: $TOTAL_ENTITIES"
        return 0
    else
        log_warning "Debug endpoints not available or not working"
        return 1
    fi
}

# Run all tests
main() {
    echo ""
    log_info "Starting Home Assistant WebSocket integration tests..."
    echo ""
    
    FAILED_TESTS=0
    
    test_backend_running || ((FAILED_TESTS++))
    echo ""
    
    test_ha_connectivity || ((FAILED_TESTS++))
    echo ""
    
    test_entity_sync || ((FAILED_TESTS++))
    echo ""
    
    test_ha_adapter_status || ((FAILED_TESTS++))
    echo ""
    
    test_websocket_connectivity || ((FAILED_TESTS++))
    echo ""
    
    test_entity_state_change || ((FAILED_TESTS++))
    echo ""
    
    test_debug_endpoints || ((FAILED_TESTS++))
    echo ""
    
    # Summary
    echo "================================================"
    if [ $FAILED_TESTS -eq 0 ]; then
        log_success "All tests completed successfully! 🎉"
        echo ""
        log_info "Next steps:"
        echo "  1. Monitor backend logs for HA WebSocket events"
        echo "  2. Test physical switch toggles in Home Assistant"
        echo "  3. Verify frontend receives real-time updates"
        echo ""
        log_info "To monitor WebSocket events in real-time:"
        echo "  tail -f pma-backend-go/logs/websocket.log"
        echo ""
    else
        log_error "$FAILED_TESTS test(s) failed"
        echo ""
        log_info "Troubleshooting steps:"
        echo "  1. Check backend logs: tail -f logs/app.log"
        echo "  2. Verify Home Assistant configuration"
        echo "  3. Check network connectivity"
        echo "  4. Restart backend if needed"
        echo ""
    fi
    
    # Cleanup
    rm -f /tmp/websocket_test.js
    
    exit $FAILED_TESTS
}

# Run the main function
main "$@" 