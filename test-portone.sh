#!/bin/bash

# Quick PortOne Integration Test Script
# This verifies the PortOne integration is working

echo "========================================"
echo "  PortOne Integration Test"
echo "========================================"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL="${1:-http://localhost:8080}"

echo "Testing against: $BASE_URL"
echo ""

# Test 1: Health Check
echo -n "1. Health check... "
if curl -s "${BASE_URL}/ping" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
else
    echo -e "${RED}✗ FAIL${NC}"
    echo "   Make sure myCart is running: ./dev-start.sh -d"
    exit 1
fi

# Test 2: Payment Methods API
echo -n "2. Payment methods API... "
RESPONSE=$(curl -s "${BASE_URL}/api/cart/payment")
if echo "$RESPONSE" | grep -q "success"; then
    echo -e "${GREEN}✓ PASS${NC}"
    
    # Check if portone is in the list
    echo -n "   - PortOne in list... "
    if echo "$RESPONSE" | grep -q "portone"; then
        echo -e "${GREEN}✓ PASS${NC}"
        
        # Check if portone is active
        echo -n "   - PortOne active... "
        if echo "$RESPONSE" | grep -q '"portone":\s*true'; then
            echo -e "${GREEN}✓ PASS${NC}"
        else
            echo -e "${YELLOW}⚠ WARN${NC} (not activated - configure in admin panel)"
        fi
    else
        echo -e "${YELLOW}⚠ WARN${NC} (not in payment methods)"
    fi
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# Test 3: PortOne Config Endpoint
echo -n "3. PortOne config endpoint... "
CONFIG_RESPONSE=$(curl -s "${BASE_URL}/api/cart/portone-config")
if echo "$CONFIG_RESPONSE" | grep -q "store_id"; then
    echo -e "${GREEN}✓ PASS${NC}"
    
    # Check if configured
    STORE_ID=$(echo "$CONFIG_RESPONSE" | grep -o '"store_id":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$STORE_ID" ] && [ "$STORE_ID" != "" ]; then
        echo "   - Store ID configured: ${STORE_ID:0:20}..."
    else
        echo -e "   - ${YELLOW}⚠ Store ID not configured${NC}"
    fi
else
    echo -e "${RED}✗ FAIL${NC}"
fi

# Test 4: Webhook Endpoint
echo -n "4. Webhook endpoint exists... "
WEBHOOK_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "PortOne-Signature: test" \
    -w "%{http_code}" \
    -o /dev/null \
    "${BASE_URL}/api/payment/portone/webhook")

if [ "$WEBHOOK_RESPONSE" = "401" ] || [ "$WEBHOOK_RESPONSE" = "400" ]; then
    echo -e "${GREEN}✓ PASS${NC} (endpoint exists, signature validation working)"
else
    echo -e "${YELLOW}⚠ Response: ${WEBHOOK_RESPONSE}${NC}"
fi

# Summary
echo ""
echo "========================================"
echo "  Test Summary"
echo "========================================"
echo ""
echo "If all tests pass:"
echo "  1. Go to ${BASE_URL}/admin"
echo "  2. Settings → Payment → PortOne"
echo "  3. Enter your PortOne credentials"
echo "  4. Toggle Active ON"
echo "  5. Test payment at ${BASE_URL}"
echo ""
echo "For detailed testing guide, see:"
echo "  TESTING_PORTONE.md"
echo ""
