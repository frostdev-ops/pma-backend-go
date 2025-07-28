#!/bin/bash

# Memory Leak Test Script for PMA Backend
# This script monitors memory usage over time to detect potential leaks

set -e

# Configuration
SERVER_URL="http://localhost:3001"
TEST_DURATION=300  # 5 minutes
CHECK_INTERVAL=30  # 30 seconds
LOG_FILE="memory_test_$(date +%Y%m%d_%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 PMA Backend Memory Leak Test"
echo "================================="
echo "Duration: ${TEST_DURATION}s"
echo "Check interval: ${CHECK_INTERVAL}s"
echo "Log file: ${LOG_FILE}"
echo ""

# Function to get memory stats
get_memory_stats() {
    local response
    response=$(curl -s "${SERVER_URL}/api/v1/memory/stats" 2>/dev/null || echo "{}")
    echo "$response"
}

# Function to get system status
get_system_status() {
    local response
    response=$(curl -s "${SERVER_URL}/api/v1/system/status" 2>/dev/null || echo "{}")
    echo "$response"
}

# Function to check if server is running
check_server() {
    if curl -s "${SERVER_URL}/api/health" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to log with timestamp
log_message() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] ${message}" | tee -a "${LOG_FILE}"
}

# Check if server is running
log_message "Checking if server is running..."
if ! check_server; then
    echo -e "${RED}❌ Server is not running on ${SERVER_URL}${NC}"
    echo "Please start the server first: ./server -config configs/config.yaml"
    exit 1
fi

echo -e "${GREEN}✅ Server is running${NC}"
echo ""

# Initialize test
start_time=$(date +%s)
test_end_time=$((start_time + TEST_DURATION))
check_count=0
initial_memory=""
final_memory=""

log_message "Starting memory leak test..."

# Main test loop
while [ $(date +%s) -lt $test_end_time ]; do
    check_count=$((check_count + 1))
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    
    # Get memory stats
    memory_stats=$(get_memory_stats)
    system_status=$(get_system_status)
    
    # Extract memory information
    heap_alloc=$(echo "$memory_stats" | jq -r '.heap_alloc // 0' 2>/dev/null || echo "0")
    goroutines=$(echo "$memory_stats" | jq -r '.goroutines // 0' 2>/dev/null || echo "0")
    memory_usage=$(echo "$memory_stats" | jq -r '.memory_usage_percent // 0' 2>/dev/null || echo "0")
    
    # Store initial memory stats
    if [ -z "$initial_memory" ]; then
        initial_memory="$heap_alloc"
        log_message "Initial memory: ${heap_alloc} bytes"
    fi
    
    # Calculate memory increase
    if [ "$initial_memory" != "0" ] && [ "$heap_alloc" != "0" ]; then
        memory_increase=$((heap_alloc - initial_memory))
        memory_increase_mb=$((memory_increase / 1024 / 1024))
        log_message "Check #${check_count} (${elapsed}s elapsed):"
        log_message "  Memory: ${heap_alloc} bytes (${memory_increase_mb}MB change)"
        log_message "  Goroutines: ${goroutines}"
        log_message "  Usage: ${memory_usage}%"
        
        # Check for significant memory increase
        if [ $memory_increase_mb -gt 100 ]; then
            echo -e "${YELLOW}⚠️  Significant memory increase detected: ${memory_increase_mb}MB${NC}"
        fi
        
        # Check for high goroutine count
        if [ $goroutines -gt 1000 ]; then
            echo -e "${YELLOW}⚠️  High goroutine count: ${goroutines}${NC}"
        fi
    else
        log_message "Check #${check_count} (${elapsed}s elapsed): Unable to get memory stats"
    fi
    
    # Store final memory stats
    final_memory="$heap_alloc"
    
    # Sleep until next check
    sleep $CHECK_INTERVAL
done

# Test completed
log_message "Memory leak test completed"
log_message "Final memory: ${final_memory} bytes"

# Calculate total memory change
if [ "$initial_memory" != "0" ] && [ "$final_memory" != "0" ]; then
    total_increase=$((final_memory - initial_memory))
    total_increase_mb=$((total_increase / 1024 / 1024))
    
    echo ""
    echo "📊 Test Results:"
    echo "================="
    echo "Initial memory: ${initial_memory} bytes"
    echo "Final memory: ${final_memory} bytes"
    echo "Total change: ${total_increase} bytes (${total_increase_mb}MB)"
    echo "Checks performed: ${check_count}"
    
    # Determine if there's a memory leak
    if [ $total_increase_mb -gt 50 ]; then
        echo -e "${RED}❌ MEMORY LEAK DETECTED!${NC}"
        echo "Memory increased by ${total_increase_mb}MB over ${TEST_DURATION} seconds"
        echo "This indicates a potential memory leak in the application."
    elif [ $total_increase_mb -gt 20 ]; then
        echo -e "${YELLOW}⚠️  POSSIBLE MEMORY LEAK${NC}"
        echo "Memory increased by ${total_increase_mb}MB over ${TEST_DURATION} seconds"
        echo "Monitor the application for longer periods to confirm."
    else
        echo -e "${GREEN}✅ NO MEMORY LEAK DETECTED${NC}"
        echo "Memory usage is stable within acceptable limits."
    fi
else
    echo -e "${YELLOW}⚠️  Unable to determine memory change - check server logs${NC}"
fi

echo ""
echo "Log file: ${LOG_FILE}"
echo "Test completed at $(date)" 