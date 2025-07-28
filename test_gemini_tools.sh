#!/bin/bash

# Test script for Gemini tool calling functionality
echo "🧪 Testing Gemini Tool Calling with PMA Backend"
echo "================================================="

# Test 1: Check if backend starts properly
echo "1️⃣  Testing backend startup..."
timeout 10s /tmp/pma-server-with-gemini-tools &
BACKEND_PID=$!
sleep 5

# Check if backend is running
if kill -0 $BACKEND_PID 2>/dev/null; then
    echo "✅ Backend started successfully"
else
    echo "❌ Backend failed to start"
    exit 1
fi

# Test 2: Create a conversation with Gemini
echo ""
echo "2️⃣  Creating conversation with Gemini..."
CONVERSATION_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/conversations \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Tool Calling Test",
    "provider": "gemini",
    "model": "gemini-2.5-flash"
  }')

CONVERSATION_ID=$(echo $CONVERSATION_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -n "$CONVERSATION_ID" ]; then
    echo "✅ Conversation created: $CONVERSATION_ID"
else
    echo "❌ Failed to create conversation"
    echo "Response: $CONVERSATION_RESPONSE"
    kill $BACKEND_PID
    exit 1
fi

# Test 3: Send a message that should trigger tool calling
echo ""
echo "3️⃣  Testing tool calling with home automation request..."
CHAT_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/conversations/$CONVERSATION_ID/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Turn on the lights in the living room"
  }')

echo "Response: $CHAT_RESPONSE"

# Check for tool calls in the response
if echo "$CHAT_RESPONSE" | grep -q "tool_calls"; then
    echo "✅ Tool calls detected in response!"
    TOOL_COUNT=$(echo "$CHAT_RESPONSE" | grep -o '"tool_calls":[0-9]*' | cut -d':' -f2)
    echo "🔧 Number of tool calls: $TOOL_COUNT"
else
    echo "❌ No tool calls detected"
    echo "Full response: $CHAT_RESPONSE"
fi

# Test 4: Check AI providers
echo ""
echo "4️⃣  Checking AI providers configuration..."
PROVIDERS_RESPONSE=$(curl -s http://localhost:8080/api/v1/ai/providers)
echo "Providers: $PROVIDERS_RESPONSE"

if echo "$PROVIDERS_RESPONSE" | grep -q "gemini"; then
    echo "✅ Gemini provider found"
else
    echo "❌ Gemini provider not found"
fi

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $BACKEND_PID
wait $BACKEND_PID 2>/dev/null

echo ""
echo "�� Test completed!" 