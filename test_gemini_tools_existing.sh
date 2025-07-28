#!/bin/bash

# Test script for Gemini tool calling functionality using existing backend
echo "🧪 Testing Gemini Tool Calling with Existing PMA Backend"
echo "========================================================="

# Test 1: Check if backend is running
echo "1️⃣  Checking if backend is running on port 3001..."
if curl -s http://localhost:3001/api/health > /dev/null; then
    echo "✅ Backend is running and accessible"
else
    echo "❌ Backend is not accessible on port 3001"
    exit 1
fi

# Test 2: Check AI providers
echo ""
echo "2️⃣  Checking AI providers configuration..."
PROVIDERS_RESPONSE=$(curl -s http://localhost:3001/api/v1/ai/providers)
echo "Providers response: $PROVIDERS_RESPONSE"

if echo "$PROVIDERS_RESPONSE" | grep -q "gemini"; then
    echo "✅ Gemini provider found"
else
    echo "❌ Gemini provider not found"
fi

# Test 3: Create a conversation with Gemini
echo ""
echo "3️⃣  Creating conversation with Gemini..."
CONVERSATION_RESPONSE=$(curl -s -X POST http://localhost:3001/api/v1/conversations \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Gemini Tool Calling Test",
    "provider": "gemini",
    "model": "gemini-2.5-flash"
  }')

echo "Conversation response: $CONVERSATION_RESPONSE"

CONVERSATION_ID=$(echo "$CONVERSATION_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$CONVERSATION_ID" ]; then
    echo "✅ Conversation created: $CONVERSATION_ID"
else
    echo "❌ Failed to create conversation"
    echo "Response: $CONVERSATION_RESPONSE"
    exit 1
fi

# Test 4: Send a message that should trigger tool calling
echo ""
echo "4️⃣  Testing tool calling with home automation request..."
echo "Sending: 'Turn on the lights in the living room and check the temperature'"

CHAT_RESPONSE=$(curl -s -X POST http://localhost:3001/api/v1/conversations/$CONVERSATION_ID/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Turn on the lights in the living room and check the temperature"
  }')

echo ""
echo "📝 Full Chat Response:"
echo "$CHAT_RESPONSE"
echo ""

# Check for tool calls in the response
if echo "$CHAT_RESPONSE" | grep -q "tool_calls"; then
    echo "✅ Tool calls detected in response!"
    
    # Try to extract tool call count
    TOOL_COUNT=$(echo "$CHAT_RESPONSE" | grep -o '"tool_calls":\[[^]]*\]' | grep -o '\[.*\]' | grep -o ',' | wc -l)
    TOOL_COUNT=$((TOOL_COUNT + 1))
    echo "🔧 Estimated number of tool calls: $TOOL_COUNT"
else
    echo "❌ No tool calls detected in response"
    echo ""
    echo "🔍 Let's check what the AI actually responded with..."
    
    # Try to extract the content from the response
    CONTENT=$(echo "$CHAT_RESPONSE" | grep -o '"content":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$CONTENT" ]; then
        echo "📄 AI Response Content: $CONTENT"
    fi
    
    # Check if it's an error response
    if echo "$CHAT_RESPONSE" | grep -q "error"; then
        ERROR_MSG=$(echo "$CHAT_RESPONSE" | grep -o '"error":"[^"]*"' | cut -d'"' -f4)
        echo "❗ Error: $ERROR_MSG"
    fi
fi

# Test 5: Test with a simpler prompt to verify basic functionality
echo ""
echo "5️⃣  Testing with a simpler prompt..."
SIMPLE_RESPONSE=$(curl -s -X POST http://localhost:3001/api/v1/conversations/$CONVERSATION_ID/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hello, can you help me with home automation?"
  }')

echo "Simple response: $SIMPLE_RESPONSE"

echo ""
echo "🏁 Test completed!"
echo ""
echo "💡 If tool calls were detected, Gemini is successfully using the MCP tools!"
echo "💡 If no tool calls were detected, check the backend logs for any errors." 