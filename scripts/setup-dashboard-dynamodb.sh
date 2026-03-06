#!/bin/bash
set -e

# Setup DynamoDB table for Dashboard API user account mapping
# Usage: ./setup-dashboard-dynamodb.sh [aws-profile]
#
# This is a ONE-TIME setup script for the default account (752123829273)

PROFILE=${1:-spore-host-infra}
TABLE_NAME="spawn-user-accounts"
REGION="us-east-1"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up Dashboard DynamoDB Table"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile:    $PROFILE"
echo "Table Name: $TABLE_NAME"
echo "Region:     $REGION"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get account ID
echo "→ Checking AWS identity..."
ACCOUNT_ID=$(aws sts get-caller-identity --profile "$PROFILE" --query Account --output text 2>/dev/null || echo "")
if [ -z "$ACCOUNT_ID" ]; then
    echo -e "${RED}✗ Failed to get AWS identity. Check your credentials.${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Account: $ACCOUNT_ID"
echo ""

# Check if table already exists
echo "→ Checking if table exists..."
if aws dynamodb describe-table \
    --profile "$PROFILE" \
    --region "$REGION" \
    --table-name "$TABLE_NAME" &>/dev/null; then
    echo -e "  ${YELLOW}⚠${NC} Table already exists!"
    echo ""

    # Show table info
    TABLE_STATUS=$(aws dynamodb describe-table \
        --profile "$PROFILE" \
        --region "$REGION" \
        --table-name "$TABLE_NAME" \
        --query 'Table.TableStatus' \
        --output text)

    ITEM_COUNT=$(aws dynamodb describe-table \
        --profile "$PROFILE" \
        --region "$REGION" \
        --table-name "$TABLE_NAME" \
        --query 'Table.ItemCount' \
        --output text)

    echo "Table Status: $TABLE_STATUS"
    echo "Item Count:   $ITEM_COUNT"
    echo ""
    echo "No changes made."
    exit 0
fi

# Create table
echo "→ Creating DynamoDB table..."

aws dynamodb create-table \
    --profile "$PROFILE" \
    --region "$REGION" \
    --table-name "$TABLE_NAME" \
    --attribute-definitions \
        AttributeName=user_id,AttributeType=S \
    --key-schema \
        AttributeName=user_id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --tags \
        Key=spawn:managed,Value=true \
        Key=spawn:purpose,Value=dashboard-api \
    --output json > /dev/null

echo -e "  ${GREEN}✓${NC} Table creation initiated"

# Wait for table to become active
echo "→ Waiting for table to become active..."
aws dynamodb wait table-exists \
    --profile "$PROFILE" \
    --region "$REGION" \
    --table-name "$TABLE_NAME"

echo -e "  ${GREEN}✓${NC} Table is active"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Created Resource:"
echo "  • DynamoDB Table: $TABLE_NAME"
echo "  • Region:         $REGION"
echo "  • Billing:        PAY_PER_REQUEST (on-demand)"
echo ""

# Show table ARN
TABLE_ARN=$(aws dynamodb describe-table \
    --profile "$PROFILE" \
    --region "$REGION" \
    --table-name "$TABLE_NAME" \
    --query 'Table.TableArn' \
    --output text)
echo "Table ARN:"
echo "  $TABLE_ARN"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Table Schema:"
echo "  Primary Key: user_id (String)"
echo "  Attributes:"
echo "    - user_id (PK)         : IAM user ARN or Cognito sub"
echo "    - aws_account_id       : AWS account ID (decimal)"
echo "    - account_base36       : Base36 encoded account ID"
echo "    - email                : User email address"
echo "    - created_at           : ISO8601 timestamp"
echo "    - last_access          : ISO8601 timestamp"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Usage:"
echo "  Lambda will auto-populate this table on first user access"
echo "  Each user's AWS account is detected via STS GetCallerIdentity"
echo ""
echo "To view table:"
echo "  aws dynamodb scan --table-name $TABLE_NAME --profile $PROFILE"
echo ""
echo "To delete table (if needed):"
echo "  aws dynamodb delete-table --table-name $TABLE_NAME --profile $PROFILE"
echo ""
