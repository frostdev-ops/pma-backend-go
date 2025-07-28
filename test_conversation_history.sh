#!/bin/bash

# Test script to verify conversation history functionality
# This will test that AI remembers information from earlier in the same conversation

echo "🧪 Testing conversation history functionality..."

# Backend URL
BASE_URL="http://localhost:3001"

echo "📝 Step 1: Creating a new conversation..."
CONV_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/conversations" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Memory Test Conversation",
    "system_prompt": null
  }')

CONV_ID=$(echo "$CONV_RESPONSE" | grep -o '"id":"[^"]*"' | sed 's/"id":"//; s/"//')
echo "✅ Created conversation: $CONV_ID"

echo ""
echo "💬 Step 2: Sending first message - telling AI my name..."
MSG1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/conversations/$CONV_ID/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hi! My name is James and I live in Toronto. Please remember this information."
  }')

echo "🤖 AI Response 1:"
echo "$MSG1_RESPONSE" | jq -r '.message.content' 2>/dev/null || echo "Failed to parse response"

echo ""
echo "💬 Step 3: Sending second message - asking AI what my name is..."
MSG2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/conversations/$CONV_ID/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "What is my name and where do I live?"
  }')

echo "🤖 AI Response 2:"
echo "$MSG2_RESPONSE" | jq -r '.message.content' 2>/dev/null || echo "Failed to parse response"

echo ""
echo "🔍 Step 4: Checking conversation history from API..."
HISTORY_RESPONSE=$(curl -s "$BASE_URL/api/v1/conversations/$CONV_ID/messages?limit=10")

echo "📜 Conversation History:"
echo "$HISTORY_RESPONSE" | jq -r '.data[] | "\(.role): \(.content)"' 2>/dev/null || echo "Failed to parse history"

echo ""
echo "✅ Test completed! Check if the AI remembered the name 'James' and location 'Toronto' in the second response." 