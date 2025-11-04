#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}TreasureHunt AI - API Test Script${NC}\n"

# Check if server is running
echo -e "${YELLOW}Checking if server is running...${NC}"
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${RED}Error: Server is not running on port 8080${NC}"
    echo -e "${YELLOW}Please start the server with: go run main.go${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}\n"

# Test 1: Health Check
echo -e "${YELLOW}Test 1: Health Check${NC}"
HEALTH=$(curl -s http://localhost:8080/health)
echo "Response: $HEALTH"
if [[ $HEALTH == *"ok"* ]]; then
    echo -e "${GREEN}✓ Health check passed${NC}\n"
else
    echo -e "${RED}✗ Health check failed${NC}\n"
fi

# Test 2: Signup
echo -e "${YELLOW}Test 2: User Signup${NC}"
SIGNUP_RESPONSE=$(curl -s -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"test$(date +%s)@example.com\",
    \"password\": \"password123\",
    \"name\": \"Test User\"
  }")

echo "Response: $SIGNUP_RESPONSE"

# Extract token using grep and sed (more portable than jq)
TOKEN=$(echo $SIGNUP_RESPONSE | grep -o '"token":"[^"]*' | sed 's/"token":"//')

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ Signup failed - no token received${NC}\n"
    exit 1
else
    echo -e "${GREEN}✓ Signup successful${NC}"
    echo -e "${GREEN}Token: ${TOKEN:0:30}...${NC}\n"
fi

# Test 3: Get Current User
echo -e "${YELLOW}Test 3: Get Current User (Protected Route)${NC}"
ME_RESPONSE=$(curl -s -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer $TOKEN")

echo "Response: $ME_RESPONSE"
if [[ $ME_RESPONSE == *"email"* ]]; then
    echo -e "${GREEN}✓ Get current user passed${NC}\n"
else
    echo -e "${RED}✗ Get current user failed${NC}\n"
fi

# Test 4: Image Analysis - Multiple Images and Prompts
echo -e "${YELLOW}Test 4: Image Analysis - Multiple Images and Prompts${NC}"

# Define test cases as an associative array
declare -a IMAGE_URLS=(
    "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg"
    "https://images.unsplash.com/photo-1495521821757-a1efb6729352?w=400"
    "https://upload.wikimedia.org/wikipedia/commons/thumb/6/6d/Good_Food_Display_-_NCI_Visuals_Online.jpg/1200px-Good_Food_Display_-_NCI_Visuals_Online.jpg"
)

declare -a PROMPTS=(
    "Is there nature or outdoor scenery in this image?"
    "Is there a person or people in this image?"
    "Is there food or edible items in this image?"
)

IMAGE_ANALYSIS_PASS=0
IMAGE_ANALYSIS_TOTAL=${#IMAGE_URLS[@]}

for i in "${!IMAGE_URLS[@]}"; do
    echo -e "${YELLOW}  Test 4.$((i+1)): Image URL $((i+1)) - Prompt: \"${PROMPTS[$i]}\"${NC}"
    
    IMAGE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/getImageContents \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d "{
        \"imageUrl\": \"${IMAGE_URLS[$i]}\",
        \"prompt\": \"${PROMPTS[$i]}\"
      }")
    
    echo "  Response: $IMAGE_RESPONSE"
    
    if [[ $IMAGE_RESPONSE == *"result"* ]]; then
        # Extract the result value
        RESULT=$(echo $IMAGE_RESPONSE | grep -o '"result":[^,}]*' | sed 's/"result"://')
        echo -e "  ${GREEN}✓ Image analysis passed (result: $RESULT)${NC}"
        ((IMAGE_ANALYSIS_PASS++))
    else
        echo -e "  ${RED}✗ Image analysis failed${NC}"
    fi
    echo ""
done

echo -e "${YELLOW}Image Analysis Summary: $IMAGE_ANALYSIS_PASS/$IMAGE_ANALYSIS_TOTAL tests passed${NC}"
if [[ $IMAGE_ANALYSIS_PASS -eq $IMAGE_ANALYSIS_TOTAL ]]; then
    echo -e "${GREEN}✓ All image analysis tests passed${NC}\n"
else
    echo -e "${RED}✗ Some image analysis tests failed${NC}"
    echo -e "${YELLOW}Note: Make sure OPENAI_API_KEY is configured in .env${NC}\n"
fi

# Test 5: Unauthorized Access
echo -e "${YELLOW}Test 5: Unauthorized Access (Should Fail)${NC}"
UNAUTHORIZED=$(curl -s -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer invalid_token")

echo "Response: $UNAUTHORIZED"
if [[ $UNAUTHORIZED == *"error"* ]]; then
    echo -e "${GREEN}✓ Unauthorized access properly rejected${NC}\n"
else
    echo -e "${RED}✗ Unauthorized access test failed${NC}\n"
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}All tests completed!${NC}"
echo -e "${GREEN}========================================${NC}"
