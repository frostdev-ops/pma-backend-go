#!/bin/bash

# Test script to verify tool execution with fallback handling

set -e

BACKEND_URL="http://localhost:3001"
CONVERSATION_ID="tool_fallback_test_$(date +%s)"

echo "🔧 TESTING TOOL EXECUTION WITH FALLBACK HANDLING"
echo "================================================"
echo ""

echo "Testing analyze_system_setup tool with fallback handling..."
echo ""

# Test the specific tool that was failing
response=$(curl -s -X POST "$BACKEND_URL/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Analyze my smart home setup using the analyze_system_setup tool",
    "role": "user"
  }')

echo "Response received:"
echo "$response" | jq '.'
echo ""

# Check if response contains content
content=$(echo "$response" | jq -r '.content // empty')
if [ -n "$content" ] && [ "$content" != "null" ]; then
    echo "✅ SUCCESS: Tool execution returned content"
    echo "Content: $content"
else
    echo "❌ FAILURE: Tool execution returned empty content"
    echo "Full response: $response"
fi

echo ""
echo "================================================"
echo "Test complete!" 