#!/bin/bash
set -e

# Setup API Gateway for Dashboard API
# Usage: ./setup-dashboard-api-gateway.sh [aws-profile]

PROFILE=${1:-spore-host-infra}
API_NAME="spawn-dashboard-api"
FUNCTION_NAME="spawn-dashboard-api"
REGION="us-east-1"
STAGE_NAME="prod"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up API Gateway for Dashboard API"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile:  $PROFILE"
echo "API Name: $API_NAME"
echo "Region:   $REGION"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get account ID
ACCOUNT_ID=$(aws sts get-caller-identity --profile "$PROFILE" --query Account --output text)
echo -e "  ${GREEN}✓${NC} Account: $ACCOUNT_ID"

# Get Lambda function ARN
FUNCTION_ARN=$(aws lambda get-function --profile "$PROFILE" --region "$REGION" --function-name "$FUNCTION_NAME" --query 'Configuration.FunctionArn' --output text 2>/dev/null || echo "")
if [ -z "$FUNCTION_ARN" ]; then
    echo -e "${RED}✗ Lambda function not found: $FUNCTION_NAME${NC}"
    echo "Deploy Lambda first: cd spawn/lambda/dashboard-api && ./deploy.sh"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Lambda function exists"
echo ""

# Check if API already exists
echo "→ Checking if API Gateway exists..."
API_ID=$(aws apigateway get-rest-apis --profile "$PROFILE" --region "$REGION" --query "items[?name=='$API_NAME'].id" --output text 2>/dev/null || echo "")

if [ -n "$API_ID" ]; then
    echo -e "  ${YELLOW}⚠${NC} API already exists: $API_ID"
    echo ""
    echo "Existing API Endpoint:"
    echo "  https://$API_ID.execute-api.$REGION.amazonaws.com/$STAGE_NAME"
    echo ""
    echo "To delete and recreate:"
    echo "  aws apigateway delete-rest-api --rest-api-id $API_ID --profile $PROFILE --region $REGION"
    echo ""
    exit 0
fi

# Create REST API
echo "→ Creating REST API..."
API_ID=$(aws apigateway create-rest-api \
    --profile "$PROFILE" \
    --region "$REGION" \
    --name "$API_NAME" \
    --description "Spawn Dashboard API - read-only instance viewer" \
    --endpoint-configuration types=REGIONAL \
    --query 'id' \
    --output text)
echo -e "  ${GREEN}✓${NC} API created: $API_ID"

# Get root resource ID
ROOT_RESOURCE_ID=$(aws apigateway get-resources \
    --profile "$PROFILE" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --query 'items[?path==`/`].id' \
    --output text)

# Create /api resource
echo "→ Creating /api resource..."
API_RESOURCE_ID=$(aws apigateway create-resource \
    --profile "$PROFILE" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --parent-id "$ROOT_RESOURCE_ID" \
    --path-part "api" \
    --query 'id' \
    --output text)
echo -e "  ${GREEN}✓${NC} Resource created: /api"

# Create /api/instances resource
echo "→ Creating /api/instances resource..."
INSTANCES_RESOURCE_ID=$(aws apigateway create-resource \
    --profile "$PROFILE" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --parent-id "$API_RESOURCE_ID" \
    --path-part "instances" \
    --query 'id' \
    --output text)
echo -e "  ${GREEN}✓${NC} Resource created: /api/instances"

# Create /api/instances/{id} resource
echo "→ Creating /api/instances/{id} resource..."
INSTANCE_ID_RESOURCE_ID=$(aws apigateway create-resource \
    --profile "$PROFILE" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --parent-id "$INSTANCES_RESOURCE_ID" \
    --path-part "{id}" \
    --query 'id' \
    --output text)
echo -e "  ${GREEN}✓${NC} Resource created: /api/instances/{id}"

# Create /api/user resource
echo "→ Creating /api/user resource..."
USER_RESOURCE_ID=$(aws apigateway create-resource \
    --profile "$PROFILE" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --parent-id "$API_RESOURCE_ID" \
    --path-part "user" \
    --query 'id' \
    --output text)
echo -e "  ${GREEN}✓${NC} Resource created: /api/user"

# Create /api/user/profile resource
echo "→ Creating /api/user/profile resource..."
PROFILE_RESOURCE_ID=$(aws apigateway create-resource \
    --profile "$PROFILE" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --parent-id "$USER_RESOURCE_ID" \
    --path-part "profile" \
    --query 'id' \
    --output text)
echo -e "  ${GREEN}✓${NC} Resource created: /api/user/profile"

# Helper function to create method and integration
create_method() {
    local RESOURCE_ID=$1
    local HTTP_METHOD=$2
    local PATH=$3

    echo "→ Creating $HTTP_METHOD method for $PATH..."

    # Create method with IAM authorization
    aws apigateway put-method \
        --profile "$PROFILE" \
        --region "$REGION" \
        --rest-api-id "$API_ID" \
        --resource-id "$RESOURCE_ID" \
        --http-method "$HTTP_METHOD" \
        --authorization-type "AWS_IAM" \
        --request-parameters method.request.path.id=false \
        --output json > /dev/null

    # Create Lambda integration
    aws apigateway put-integration \
        --profile "$PROFILE" \
        --region "$REGION" \
        --rest-api-id "$API_ID" \
        --resource-id "$RESOURCE_ID" \
        --http-method "$HTTP_METHOD" \
        --type AWS_PROXY \
        --integration-http-method POST \
        --uri "arn:aws:apigateway:$REGION:lambda:path/2015-03-31/functions/$FUNCTION_ARN/invocations" \
        --output json > /dev/null

    echo -e "  ${GREEN}✓${NC} Method and integration created"
}

# Create OPTIONS method for CORS preflight (all resources)
create_options_method() {
    local RESOURCE_ID=$1
    local PATH=$2

    echo "→ Creating OPTIONS method for $PATH (CORS)..."

    aws apigateway put-method \
        --profile "$PROFILE" \
        --region "$REGION" \
        --rest-api-id "$API_ID" \
        --resource-id "$RESOURCE_ID" \
        --http-method OPTIONS \
        --authorization-type NONE \
        --output json > /dev/null

    aws apigateway put-integration \
        --profile "$PROFILE" \
        --region "$REGION" \
        --rest-api-id "$API_ID" \
        --resource-id "$RESOURCE_ID" \
        --http-method OPTIONS \
        --type MOCK \
        --request-templates '{"application/json":"{\"statusCode\":200}"}' \
        --output json > /dev/null

    aws apigateway put-method-response \
        --profile "$PROFILE" \
        --region "$REGION" \
        --rest-api-id "$API_ID" \
        --resource-id "$RESOURCE_ID" \
        --http-method OPTIONS \
        --status-code 200 \
        --response-parameters \
            method.response.header.Access-Control-Allow-Headers=false,\
method.response.header.Access-Control-Allow-Methods=false,\
method.response.header.Access-Control-Allow-Origin=false \
        --output json > /dev/null

    aws apigateway put-integration-response \
        --profile "$PROFILE" \
        --region "$REGION" \
        --rest-api-id "$API_ID" \
        --resource-id "$RESOURCE_ID" \
        --http-method OPTIONS \
        --status-code 200 \
        --response-parameters \
            method.response.header.Access-Control-Allow-Headers="'Content-Type,Authorization,X-Amz-Date,X-Api-Key,X-Amz-Security-Token'",\
method.response.header.Access-Control-Allow-Methods="'GET,OPTIONS'",\
method.response.header.Access-Control-Allow-Origin="'*'" \
        --output json > /dev/null

    echo -e "  ${GREEN}✓${NC} CORS preflight configured"
}

# Create methods and integrations
create_method "$INSTANCES_RESOURCE_ID" "GET" "/api/instances"
create_options_method "$INSTANCES_RESOURCE_ID" "/api/instances"

create_method "$INSTANCE_ID_RESOURCE_ID" "GET" "/api/instances/{id}"
create_options_method "$INSTANCE_ID_RESOURCE_ID" "/api/instances/{id}"

create_method "$PROFILE_RESOURCE_ID" "GET" "/api/user/profile"
create_options_method "$PROFILE_RESOURCE_ID" "/api/user/profile"

# Grant API Gateway permission to invoke Lambda
echo "→ Granting API Gateway permission to invoke Lambda..."
aws lambda add-permission \
    --profile "$PROFILE" \
    --region "$REGION" \
    --function-name "$FUNCTION_NAME" \
    --statement-id "apigateway-invoke-$API_ID" \
    --action lambda:InvokeFunction \
    --principal apigateway.amazonaws.com \
    --source-arn "arn:aws:execute-api:$REGION:$ACCOUNT_ID:$API_ID/*/*" \
    --output json > /dev/null 2>&1 || echo -e "  ${YELLOW}⚠${NC} Permission may already exist"
echo -e "  ${GREEN}✓${NC} Permission granted"

# Deploy API to stage
echo "→ Deploying API to $STAGE_NAME stage..."
aws apigateway create-deployment \
    --profile "$PROFILE" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --stage-name "$STAGE_NAME" \
    --description "Initial deployment" \
    --output json > /dev/null
echo -e "  ${GREEN}✓${NC} API deployed"

# Get API endpoint
API_ENDPOINT="https://$API_ID.execute-api.$REGION.amazonaws.com/$STAGE_NAME"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ API Gateway Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "API Details:"
echo "  API ID:   $API_ID"
echo "  Stage:    $STAGE_NAME"
echo "  Region:   $REGION"
echo ""
echo "API Endpoint:"
echo "  $API_ENDPOINT"
echo ""
echo "Endpoints:"
echo "  GET $API_ENDPOINT/api/instances"
echo "  GET $API_ENDPOINT/api/instances/{id}"
echo "  GET $API_ENDPOINT/api/user/profile"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next Steps:"
echo "  1. Save API endpoint for frontend integration"
echo "  2. Test endpoints with AWS CLI or curl (requires IAM auth)"
echo "  3. Update web/js/main.js with API_ENDPOINT"
echo ""
echo "To test (requires AWS credentials):"
echo "  aws apigateway test-invoke-method \\"
echo "    --rest-api-id $API_ID \\"
echo "    --resource-id $INSTANCES_RESOURCE_ID \\"
echo "    --http-method GET \\"
echo "    --path-with-query-string /api/instances"
echo ""
