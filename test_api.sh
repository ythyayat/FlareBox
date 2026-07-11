#!/bin/bash

# Simple Email Server API Test Script
# Make sure the server is running before executing this script

BASE_URL="http://localhost:8080"
API_KEY="your-secret-api-key-here"

echo "==================================="
echo "Testing Simple Email Server API"
echo "==================================="
echo ""

# Test 1: Health Check
echo "1. Testing Health Check..."
curl -s -X GET "$BASE_URL/health"
echo -e "\n"

# Test 2: Send Email via Webhook
echo "2. Sending test email via webhook..."
curl -X POST "$BASE_URL/api/webhook" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "to": "testuser@example.com",
    "from": "sender@company.com",
    "subject": "Test Email - OTP Code",
    "body": "Your verification code is: 123456",
    "html_body": "<p>Your verification code is: <strong>123456</strong></p>",
    "has_attachments": false
  }'
echo -e "\n"

# Test 3: Send Another Email
echo "3. Sending another test email..."
curl -X POST "$BASE_URL/api/webhook" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "to": "testuser@example.com",
    "from": "noreply@service.com",
    "subject": "Welcome Email",
    "body": "Welcome to our service!",
    "html_body": "<h1>Welcome!</h1><p>Welcome to our service!</p>",
    "has_attachments": false
  }'
echo -e "\n"

# Wait a moment for processing
sleep 1

# Test 4: Retrieve Emails
echo "4. Retrieving emails for testuser@example.com..."
curl -s -X GET "$BASE_URL/api/email/example.com/testuser/?page=1&limit=20" | jq '.'
echo -e "\n"

# Test 5: Test Authentication (should fail)
echo "5. Testing authentication with invalid API key (should fail)..."
curl -X POST "$BASE_URL/api/webhook" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: invalid-key" \
  -d '{
    "to": "test@example.com",
    "from": "sender@company.com",
    "subject": "Test",
    "body": "This should fail"
  }'
echo -e "\n"

echo "==================================="
echo "Tests completed!"
echo "==================================="
