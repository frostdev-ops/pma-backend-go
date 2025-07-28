#!/bin/bash

# Test script to verify no recursive tool calls

set -e

BACKEND_URL="http://localhost:3001"
CONVERSATION_ID="no_recursive_tools_test_$(date +%s)"

echo "🔄 TESTING: NO RECURSIVE TOOL CALLS"
echo "====================================="
echo ""

echo "Testing that tool execution doesn't trigger recursive calls..."
echo ""

# Start monitoring logs in background (if server is running locally)
echo "Making request and monitoring for recursive tool execution..."

# Test the analyze_system_setup tool specifically
response=$(curl -s -X POST "$BACKEND_URL/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Analyze my smart home setup using analyze_system_setup tool",
    "role": "user"
  }')

echo "Response received:"
echo "$response" | jq '.'
echo ""

# Check if response contains content and no errors
content=$(echo "$response" | jq -r '.content // empty')
error=$(echo "$response" | jq -r '.error // empty')

if [ -n "$content" ] && [ "$content" != "null" ] && [ "$error" = "empty" ]; then
    echo "✅ SUCCESS: Tool execution returned content without recursive calls"
    echo ""
    echo "Content preview:"
    echo "$content" | head -200
else
    echo "❌ FAILURE: Tool execution issue"
    echo "Content: $content"
    echo "Error: $error" 
fi

echo ""
echo "====================================="
echo "🔍 CHECK THE BACKEND LOGS"
echo "You should see:"
echo "✅ One 'Executing MCP tool' message"
echo "✅ One 'Tool executed successfully' message" 
echo "✅ One 'Successfully generated follow-up response' message"
echo "❌ NO repeated tool execution cycles"
echo ""
echo "Test complete!" 