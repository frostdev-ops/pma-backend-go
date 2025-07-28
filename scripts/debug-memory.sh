#!/bin/bash

# Memory debugging script for PMA Backend
# This script helps diagnose memory leaks using pprof

echo "=== PMA Backend Memory Debugging Script ==="

# Check if the backend is running
if ! curl -s http://localhost:3001/debug/pprof/ > /dev/null; then
    echo "Error: Debug server not accessible on port 3001"
    echo "Make sure the backend is running with debug enabled"
    exit 1
fi

echo "Debug server is accessible. Starting memory analysis..."

# Create debug output directory
DEBUG_DIR="debug-output-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$DEBUG_DIR"

echo "Output directory: $DEBUG_DIR"

# Capture initial heap profile
echo "Capturing initial heap profile..."
curl -s http://localhost:3001/debug/pprof/heap > "$DEBUG_DIR/heap-initial.prof"

# Capture goroutine profile
echo "Capturing goroutine profile..."
curl -s http://localhost:3001/debug/pprof/goroutine > "$DEBUG_DIR/goroutine.prof"

# Get current memory stats
echo "Getting memory stats..."
curl -s http://localhost:3001/debug/pprof/heap?debug=1 > "$DEBUG_DIR/heap-debug.txt"

# Get goroutine debug info
curl -s http://localhost:3001/debug/pprof/goroutine?debug=1 > "$DEBUG_DIR/goroutine-debug.txt"

echo ""
echo "=== Current Memory Usage ==="
curl -s http://localhost:3001/debug/vars | jq '.memstats' 2>/dev/null || curl -s http://localhost:3001/debug/vars

echo ""
echo "=== Goroutine Count ==="
curl -s http://localhost:3001/debug/pprof/goroutine?debug=1 | head -n 1

echo ""
echo "=== Analysis Results ==="
echo "Files saved to: $DEBUG_DIR/"
echo ""
echo "To analyze the heap profile:"
echo "  go tool pprof $DEBUG_DIR/heap-initial.prof"
echo ""
echo "To analyze goroutines:"
echo "  go tool pprof $DEBUG_DIR/goroutine.prof"
echo ""
echo "To monitor continuous memory growth:"
echo "  watch -n 5 'curl -s http://localhost:3001/debug/vars | jq .memstats.Alloc'"
echo ""
echo "=== Quick Analysis ==="
echo "Checking for common memory leak patterns..."

# Check for excessive goroutines
GOROUTINE_COUNT=$(curl -s http://localhost:3001/debug/pprof/goroutine?debug=1 | head -n 1 | grep -o '[0-9]\+')
if [ "$GOROUTINE_COUNT" -gt 100 ]; then
    echo "⚠️  WARNING: High goroutine count ($GOROUTINE_COUNT) - potential goroutine leak"
else
    echo "✅ Goroutine count looks normal ($GOROUTINE_COUNT)"
fi

# Check for heap growth patterns
HEAP_SIZE=$(curl -s http://localhost:3001/debug/vars | jq -r '.memstats.Alloc // "unknown"')
echo "📊 Current heap allocation: $HEAP_SIZE bytes"

echo ""
echo "Run this script multiple times to monitor memory growth patterns."