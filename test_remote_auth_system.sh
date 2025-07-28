#!/bin/bash

# Test script for PMA Remote Authentication System
# This script tests both localhost bypass and remote authentication

BASE_URL="http://localhost:3001"
API_BASE="$BASE_URL/api/v1"

echo "🧪 Testing PMA Remote Authentication System"
echo "=========================================="

# Test 1: Check remote auth status from localhost (should bypass auth)
echo ""
echo "1. Testing remote auth status from localhost (should bypass auth)..."
curl -s -X GET "$API_BASE/auth/remote-status" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  | jq '.' 2>/dev/null || echo "Response received"

# Test 2: Check if users exist
echo ""
echo "2. Checking if users exist in system..."
curl -s -X GET "$API_BASE/users" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  | jq '.' 2>/dev/null || echo "Response received"

# Test 3: Create first admin user (if no users exist)
echo ""
echo "3. Creating first admin user..."
curl -s -X POST "$API_BASE/auth/user/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin123!",
    "email": "admin@pma.local"
  }' \
  -w "\nHTTP Status: %{http_code}\n" \
  | jq '.' 2>/dev/null || echo "Response received"

# Test 4: Login with created user
echo ""
echo "4. Testing user login..."
curl -s -X POST "$API_BASE/auth/user/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin123!"
  }' \
  -w "\nHTTP Status: %{http_code}\n" \
  | jq '.' 2>/dev/null || echo "Response received"

# Test 5: Test invalid login
echo ""
echo "5. Testing invalid login (should fail)..."
curl -s -X POST "$API_BASE/auth/user/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "wrongpassword"
  }' \
  -w "\nHTTP Status: %{http_code}\n" \
  | jq '.' 2>/dev/null || echo "Response received"

# Test 6: Test PIN authentication (legacy)
echo ""
echo "6. Testing PIN authentication (legacy)..."
curl -s -X POST "$API_BASE/auth/verify-pin" \
  -H "Content-Type: application/json" \
  -d '{
    "pin": "1234"
  }' \
  -w "\nHTTP Status: %{http_code}\n" \
  | jq '.' 2>/dev/null || echo "Response received"

# Test 7: Check system status (should work without auth from localhost)
echo ""
echo "7. Testing system status (localhost should work without auth)..."
curl -s -X GET "$API_BASE/status" \
  -H "Content-Type: application/json" \
  -w "\nHTTP Status: %{http_code}\n" \
  | jq '.' 2>/dev/null || echo "Response received"

echo ""
echo "✅ Remote Authentication System Test Complete"
echo "============================================"
echo ""
echo "Summary:"
echo "- Localhost access should bypass authentication"
echo "- Remote access should require user/password login"
echo "- First user registration should work when no users exist"
echo "- PIN authentication should still work for legacy compatibility"
echo ""
echo "Frontend should now:"
echo "1. Show registration screen if no users exist"
echo "2. Show login screen for remote access"
echo "3. Bypass authentication for localhost access"
echo "4. Maintain PIN authentication for local access" 