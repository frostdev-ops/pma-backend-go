#!/bin/bash

# Debug script to see exactly what's happening with conversation API responses

echo "🔍 Debugging conversation API responses..."

# Backend URL
BASE_URL="http://localhost:3001"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📝 Step 1: Creating a new conversation..."
CONV_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/conversations" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Debug Test Conversation",
    "system_prompt": null
  }')

echo "🔍 RAW CONVERSATION RESPONSE:"
echo "$CONV_RESPONSE"
echo ""

CONV_ID=$(echo "$CONV_RESPONSE" | jq -r '.conversation.id // .id // .data.id // empty' 2>/dev/null)
echo "🆔 Extracted conversation ID: '$CONV_ID'"

if [ -z "$CONV_ID" ] || [ "$CONV_ID" = "null" ]; then
    echo "❌ Failed to create conversation or extract ID. Exiting."
    exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💬 Step 2: Sending first message..."
MSG1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/conversations/$CONV_ID/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hello! My name is James and I like pizza. Please remember this."
  }')

echo "🔍 RAW MESSAGE 1 RESPONSE:"
echo "$MSG1_RESPONSE"
echo ""

MSG1_CONTENT=$(echo "$MSG1_RESPONSE" | jq -r '.message.content // .data.message.content // empty' 2>/dev/null)
echo "📝 Extracted message 1 content: '$MSG1_CONTENT'"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💬 Step 3: Sending second message..."
MSG2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/conversations/$CONV_ID/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "What is my name and what food do I like?"
  }')

echo "🔍 RAW MESSAGE 2 RESPONSE:"
echo "$MSG2_RESPONSE"
echo ""

MSG2_CONTENT=$(echo "$MSG2_RESPONSE" | jq -r '.message.content // .data.message.content // empty' 2>/dev/null)
echo "📝 Extracted message 2 content: '$MSG2_CONTENT'"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📜 Step 4: Checking conversation history..."
HISTORY_RESPONSE=$(curl -s "$BASE_URL/api/v1/conversations/$CONV_ID/messages?limit=10")

echo "🔍 RAW HISTORY RESPONSE:"
echo "$HISTORY_RESPONSE"
echo ""

echo "📊 FORMATTED HISTORY:"
echo "$HISTORY_RESPONSE" | jq -r '.data[]? | "\(.role): \(.content)"' 2>/dev/null || echo "Failed to parse history"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧐 ANALYSIS:"
echo "- Did conversation creation work? $([ -n "$CONV_ID" ] && echo "✅ YES" || echo "❌ NO")"
echo "- Did message 1 get a response? $([ -n "$MSG1_CONTENT" ] && echo "✅ YES" || echo "❌ NO")"
echo "- Did message 2 get a response? $([ -n "$MSG2_CONTENT" ] && echo "❌ NO" || echo "✅ YES")"
echo "- Does the AI remember the name from message 1? $(echo "$MSG2_CONTENT" | grep -i "james" >/dev/null 2>&1 && echo "✅ YES" || echo "❌ NO")"
echo "- Does the AI remember the food preference? $(echo "$MSG2_CONTENT" | grep -i "pizza" >/dev/null 2>&1 && echo "✅ YES" || echo "❌ NO")" 