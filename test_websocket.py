#!/usr/bin/env python3
"""
Simple WebSocket client test for PMA Backend WebSocket functionality
"""

import asyncio
import websockets
import json
import requests
import signal
import sys

# Configuration
SERVER_URL = "ws://localhost:3001/ws"
API_BASE = "http://localhost:3001/api/v1"
JWT_TOKEN = "test-token"  # Simple token for testing

def signal_handler(sig, frame):
    print('\nTest interrupted by user')
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)

async def test_websocket_connection():
    """Test basic WebSocket connection and messaging"""
    print("Testing WebSocket connection...")
    
    try:
        async with websockets.connect(SERVER_URL) as websocket:
            print("✅ Connected to WebSocket")
            
            # Wait for welcome message
            welcome_msg = await websocket.recv()
            welcome_data = json.loads(welcome_msg)
            print(f"✅ Received welcome message: {welcome_data}")
            
            # Send a ping message
            ping_msg = {
                "type": "ping",
                "data": {}
            }
            await websocket.send(json.dumps(ping_msg))
            print("📤 Sent ping message")
            
            # Wait for pong response
            response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
            response_data = json.loads(response)
            print(f"✅ Received pong response: {response_data}")
            
            # Subscribe to room updates
            subscribe_msg = {
                "type": "subscribe_room",
                "data": {"room_id": 1}
            }
            await websocket.send(json.dumps(subscribe_msg))
            print("📤 Subscribed to room 1 updates")
            
            # Wait for heartbeat
            print("⏳ Waiting for heartbeat message...")
            heartbeat = await asyncio.wait_for(websocket.recv(), timeout=35.0)
            heartbeat_data = json.loads(heartbeat)
            print(f"💓 Received heartbeat: {heartbeat_data}")
            
    except websockets.exceptions.ConnectionRefused:
        print("❌ Failed to connect to WebSocket server. Is the server running?")
        return False
    except asyncio.TimeoutError:
        print("⏰ Timeout waiting for WebSocket response")
        return False
    except Exception as e:
        print(f"❌ WebSocket test failed: {e}")
        return False
    
    return True

def test_websocket_api_endpoints():
    """Test WebSocket management API endpoints"""
    print("\nTesting WebSocket API endpoints...")
    
    headers = {"Authorization": f"Bearer {JWT_TOKEN}"}
    
    try:
        # Test WebSocket stats endpoint
        response = requests.get(f"{API_BASE}/websocket/stats", headers=headers)
        if response.status_code == 200:
            stats = response.json()
            print(f"✅ WebSocket stats: {stats}")
        else:
            print(f"❌ WebSocket stats failed: {response.status_code} - {response.text}")
            return False
        
        # Test broadcast endpoint
        broadcast_data = {
            "type": "test_broadcast",
            "data": {
                "message": "Hello from API test",
                "timestamp": "2025-07-16T21:30:00Z"
            }
        }
        
        response = requests.post(f"{API_BASE}/websocket/broadcast", 
                               headers=headers, 
                               json=broadcast_data)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Broadcast successful: {result}")
        else:
            print(f"❌ Broadcast failed: {response.status_code} - {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        print("❌ Failed to connect to API server. Is the server running?")
        return False
    except Exception as e:
        print(f"❌ API test failed: {e}")
        return False
    
    return True

async def main():
    """Main test function"""
    print("🚀 Starting PMA Backend WebSocket Tests")
    print("="*50)
    
    # Test API endpoints first
    api_success = test_websocket_api_endpoints()
    
    # Test WebSocket connection
    ws_success = await test_websocket_connection()
    
    print("\n" + "="*50)
    print("📊 Test Results:")
    print(f"API Endpoints: {'✅ PASS' if api_success else '❌ FAIL'}")
    print(f"WebSocket Connection: {'✅ PASS' if ws_success else '❌ FAIL'}")
    
    if api_success and ws_success:
        print("🎉 All WebSocket tests passed!")
        return True
    else:
        print("⚠️  Some tests failed. Check server logs for details.")
        return False

if __name__ == "__main__":
    try:
        result = asyncio.run(main())
        sys.exit(0 if result else 1)
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
        sys.exit(1) 