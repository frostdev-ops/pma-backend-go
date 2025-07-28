#!/bin/bash

# Test script for new smart home control tools
# Tests the enhanced AI capabilities for device discovery and control

set -e

BACKEND_URL="http://localhost:3001"
CONVERSATION_ID=$(date +%s)_test

echo "🏠 Testing Smart Home AI Tools"
echo "==============================="
echo ""

# Test 1: Find devices by name
echo "🔍 Test 1: Finding devices by name (bedroom lights)"
curl -X POST "$BACKEND_URL/api/v1/conversations/$CONVERSATION_ID/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Are there any bedroom lights? Use the find devices tool to search for them.",
    "role": "user"
  }' | jq '.content' || echo "Request failed"

echo ""
echo "---"
echo ""

# Test 2: Get all rooms
echo "🏡 Test 2: Getting all available rooms"
curl -X POST "$BACKEND_URL/api/v1/conversations/${CONVERSATION_ID}_rooms/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "What rooms are available in the smart home? Show me all rooms.",
    "role": "user"
  }' | jq '.content' || echo "Request failed"

echo ""
echo "---"
echo ""

# Test 3: Search devices
echo "🔎 Test 3: Advanced device search"
curl -X POST "$BACKEND_URL/api/v1/conversations/${CONVERSATION_ID}_search/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Search for all light devices in the house. Use the search devices tool.",
    "role": "user"
  }' | jq '.content' || echo "Request failed"

echo ""
echo "---"
echo ""

# Test 4: Room status
echo "🛏️ Test 4: Getting room status"
curl -X POST "$BACKEND_URL/api/v1/conversations/${CONVERSATION_ID}_status/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "What is the status of devices in the bedroom? Show me what devices are on or off.",
    "role": "user"
  }' | jq '.content' || echo "Request failed"

echo ""
echo "=============================="
echo "✅ Smart Home Tools Test Complete"
echo ""
echo "The AI now has access to 19 tools across 7 categories:"
echo "• Device Discovery: find_devices_by_name, search_devices, get_device_details"
echo "• Room Control: get_all_rooms, get_room_status, control_room"  
echo "• Device Control: control_multiple_devices, toggle_devices, set_brightness"
echo "• Monitoring: get_sensor_readings, check_device_connectivity"
echo "• Home Assistant: get_entity_state, set_entity_state, get_room_entities, execute_scene"
echo "• Energy: get_energy_data"
echo "• Automation: create_automation"
echo "• Analytics: analyze_patterns"
echo "• System: get_system_status"
echo ""
echo "🎯 Key new capabilities:"
echo "✓ Natural device discovery: 'Are the bedroom lights on?'"
echo "✓ Room-based control: 'Turn off all living room lights'"
echo "✓ Advanced search: 'Find all motion sensors'"
echo "✓ Bulk operations: Control multiple devices at once"
echo "✓ Comprehensive monitoring: Device health, sensor data" 