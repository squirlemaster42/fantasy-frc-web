#!/bin/bash

# CSRF Testing Script for Fantasy FRC Web
# Tests CSRF protection on all endpoints

set -e

BASE_URL="http://localhost:3000"
COOKIE_JAR="csrf_test_cookies.txt"
TEMP_DIR="csrf_test_temp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    rm -f "$COOKIE_JAR"
    rm -rf "$TEMP_DIR"
}

# Setup cleanup on exit
trap cleanup EXIT

# Create temp directory
mkdir -p "$TEMP_DIR"

# Function to extract CSRF token from HTML
extract_csrf_token() {
    local url=$1
    local response_file="$TEMP_DIR/response.html"
    
    curl -s -c "$COOKIE_JAR" "$BASE_URL$url" -o "$response_file"
    
    # Extract CSRF token using grep and sed
    local token=$(grep -o 'name="csrf_token" value="[^"]*"' "$response_file" | sed 's/name="csrf_token" value="\([^"]*\)"/\1/')
    
    if [ -z "$token" ]; then
        echo -e "${RED}❌ No CSRF token found on $url${NC}"
        return 1
    else
        echo "$token"
        return 0
    fi
}

# Function to test CSRF protected endpoint
test_csrf_endpoint() {
    local endpoint=$1
    local data=$2
    local token=$3
    local test_name=$4
    
    echo -e "\n${YELLOW}Testing $test_name ($endpoint)...${NC}"
    
    local all_passed=true
    
    # Test 1: Valid CSRF token
    if [ -n "$token" ]; then
        echo "  Testing with valid CSRF token..."
        local response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/valid_response.html" \
            -b "$COOKIE_JAR" \
            -X POST \
            -H "Content-Type: application/x-www-form-urlencoded" \
            -d "$data&csrf_token=$token" \
            "$BASE_URL$endpoint")
        
        if [ "$response" = "200" ] || [ "$response" = "302" ] || [ "$response" = "201" ]; then
            echo -e "    ${GREEN}✅ Valid CSRF token: PASS${NC}"
        else
            echo -e "    ${RED}❌ Valid CSRF token: FAIL ($response)${NC}"
            all_passed=false
        fi
    else
        echo -e "    ${YELLOW}⚠️  No CSRF token available for valid test${NC}"
    fi
    
    # Test 2: Missing CSRF token
    echo "  Testing without CSRF token..."
    local response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/missing_response.html" \
        -b "$COOKIE_JAR" \
        -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$data" \
        "$BASE_URL$endpoint")
    
    if [ "$response" = "403" ]; then
        echo -e "    ${GREEN}✅ Missing CSRF token: PASS${NC}"
    else
        echo -e "    ${RED}❌ Missing CSRF token: FAIL ($response)${NC}"
        all_passed=false
    fi
    
    # Test 3: Invalid CSRF token
    echo "  Testing with invalid CSRF token..."
    local response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/invalid_response.html" \
        -b "$COOKIE_JAR" \
        -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$data&csrf_token=invalid_token_12345" \
        "$BASE_URL$endpoint")
    
    if [ "$response" = "403" ]; then
        echo -e "    ${GREEN}✅ Invalid CSRF token: PASS${NC}"
    else
        echo -e "    ${RED}❌ Invalid CSRF token: FAIL ($response)${NC}"
        all_passed=false
    fi
    
    # Test 4: Cross-origin request
    echo "  Testing cross-origin request..."
    local response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/cross_origin_response.html" \
        -b "$COOKIE_JAR" \
        -X POST \
        -H "Origin: https://evil-site.com" \
        -H "Referer: https://evil-site.com/attack" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "$data&csrf_token=$token" \
        "$BASE_URL$endpoint")
    
    if [ "$response" = "403" ]; then
        echo -e "    ${GREEN}✅ Cross-origin protection: PASS${NC}"
    else
        echo -e "    ${RED}❌ Cross-origin protection: FAIL ($response)${NC}"
        all_passed=false
    fi
    
    # Return result
    if [ "$all_passed" = true ]; then
        return 0
    else
        return 1
    fi
}

# Function to check if server is running
check_server() {
    echo "Checking if server is running on $BASE_URL..."
    if curl -s "$BASE_URL" > /dev/null; then
        echo -e "${GREEN}✅ Server is running${NC}"
        return 0
    else
        echo -e "${RED}❌ Server is not running on $BASE_URL${NC}"
        echo "Please start the server with: cd server && make"
        exit 1
    fi
}

# Function to test form rendering
test_form_rendering() {
    echo -e "\n${YELLOW}Testing form rendering...${NC}"
    
    local forms=(
        "/login:Login Form"
        "/register:Register Form"
        "/u/createDraft:Create Draft Form"
    )
    
    for form in "${forms[@]}"; do
        local url="${form%%:*}"
        local name="${form##*:}"
        
        echo "  Testing $name..."
        local token=$(extract_csrf_token "$url")
        
        if [ -n "$token" ]; then
            echo -e "    ${GREEN}✅ $name has CSRF token${NC}"
        else
            echo -e "    ${RED}❌ $name missing CSRF token${NC}"
        fi
    done
}

# Main test execution
main() {
    echo -e "${YELLOW}=== CSRF Protection Tests ===${NC}"
    
    # Check if server is running
    check_server
    
    # Test form rendering
    test_form_rendering
    
    # Test login endpoint
    echo -e "\n${YELLOW}=== Testing Login Endpoint ===${NC}"
    CSRF_TOKEN=$(extract_csrf_token "/login")
    if [ -n "$CSRF_TOKEN" ]; then
        echo "Extracted CSRF token: ${CSRF_TOKEN:0:20}..."
        
        test_csrf_endpoint "/login" "username=test&password=test" "$CSRF_TOKEN" "Login"
    else
        echo -e "${RED}❌ Could not extract CSRF token from login page${NC}"
    fi
    
    # Test registration endpoint
    echo -e "\n${YELLOW}=== Testing Registration Endpoint ===${NC}"
    CSRF_TOKEN=$(extract_csrf_token "/register")
    if [ -n "$CSRF_TOKEN" ]; then
        test_csrf_endpoint "/register" "username=testuser$(date +%s)&password=test123&confirmPassword=test123" "$CSRF_TOKEN" "Register"
    else
        echo -e "${RED}❌ Could not extract CSRF token from register page${NC}"
    fi
    
    # Test webhook exemption
    echo -e "\n${YELLOW}=== Testing Webhook Exemption ===${NC}"
    echo "Testing /tbaWebhook endpoint (should not require CSRF)..."
    local response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/webhook_response.html" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"test":"data"}' \
        "$BASE_URL/tbaWebhook")
    
    if [ "$response" = "200" ] || [ "$response" = "204" ]; then
        echo -e "${GREEN}✅ Webhook CSRF exemption: PASS${NC}"
    else
        echo -e "${RED}❌ Webhook CSRF exemption: FAIL ($response)${NC}"
    fi
    
    # Test authenticated endpoints (if we can authenticate)
    echo -e "\n${YELLOW}=== Testing Authenticated Endpoints ===${NC}"
    
    # Try to authenticate first
    CSRF_TOKEN=$(extract_csrf_token "/login")
    if [ -n "$CSRF_TOKEN" ]; then
        echo "Attempting to authenticate..."
        local response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/auth_response.html" \
            -c "$COOKIE_JAR" \
            -X POST \
            -H "Content-Type: application/x-www-form-urlencoded" \
            -d "username=test&password=test&csrf_token=$CSRF_TOKEN" \
            "$BASE_URL/login")
        
        if [ "$response" = "302" ] || [ "$response" = "200" ]; then
            echo -e "${GREEN}✅ Authentication successful${NC}"
            
            # Test protected endpoints
            CSRF_TOKEN=$(extract_csrf_token "/u/createDraft")
            if [ -n "$CSRF_TOKEN" ]; then
                test_csrf_endpoint "/u/createDraft" "displayName=Test Draft&description=Test Description" "$CSRF_TOKEN" "Create Draft"
            else
                echo -e "${YELLOW}⚠️  Could not test protected endpoints (authentication may have failed)${NC}"
            fi
        else
            echo -e "${YELLOW}⚠️  Authentication failed (response: $response)${NC}"
            echo "Skipping authenticated endpoint tests"
        fi
    fi
    
    # Test summary
    echo -e "\n${YELLOW}=== Test Summary ===${NC}"
    echo "CSRF testing completed. Review the results above."
    echo -e "${GREEN}✅ PASS${NC} = CSRF protection working correctly"
    echo -e "${RED}❌ FAIL${NC} = CSRF protection missing or broken"
    echo -e "${YELLOW}⚠️  WARNING${NC} = Could not test (missing prerequisites)"
    
    echo -e "\n${YELLOW}=== Manual Testing Recommendations ===${NC}"
    echo "1. Open the application in a browser"
    echo "2. Use browser dev tools to inspect forms"
    echo "3. Try submitting forms without CSRF tokens"
    echo "4. Test with different browsers"
    echo "5. Verify AJAX/HTMX requests include CSRF tokens"
}

# Run main function
main "$@"