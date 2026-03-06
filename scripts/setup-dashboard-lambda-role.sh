#!/bin/bash
set -e

# Setup IAM role for Dashboard API Lambda function
# Usage: ./setup-dashboard-lambda-role.sh [aws-profile]
#
# This is a ONE-TIME setup script for the default account (752123829273)
# where the Lambda function runs.

PROFILE=${1:-spore-host-infra}
AWS_ACCOUNT_ID="942542972736"  # Account with EC2 instances

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up Dashboard Lambda IAM Role"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile: $PROFILE"
echo "EC2 Account: $AWS_ACCOUNT_ID"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Resource names
ROLE_NAME="SpawnDashboardLambdaRole"

# Get account ID
echo "→ Checking AWS identity..."
ACCOUNT_ID=$(aws sts get-caller-identity --profile "$PROFILE" --query Account --output text 2>/dev/null || echo "")
if [ -z "$ACCOUNT_ID" ]; then
    echo -e "${RED}✗ Failed to get AWS identity. Check your credentials.${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Account: $ACCOUNT_ID"

if [ "$ACCOUNT_ID" = "$AWS_ACCOUNT_ID" ]; then
    echo -e "${RED}✗ ERROR: You're using the aws account profile!${NC}"
    echo "This role should be created in the default account (752123829273)"
    echo "Use: ./setup-dashboard-lambda-role.sh default"
    exit 1
fi
echo ""

# Trust policy (allow Lambda to assume)
TRUST_POLICY=$(cat <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
)

# Permissions policy
PERMISSIONS_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CloudWatchLogs",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:${ACCOUNT_ID}:log-group:/aws/lambda/spawn-dashboard-api*"
    },
    {
      "Sid": "DynamoDBAccess",
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:Query"
      ],
      "Resource": "arn:aws:dynamodb:*:${ACCOUNT_ID}:table/spawn-user-accounts"
    },
    {
      "Sid": "STSAssumeRole",
      "Effect": "Allow",
      "Action": [
        "sts:AssumeRole"
      ],
      "Resource": "arn:aws:iam::${AWS_ACCOUNT_ID}:role/SpawnDashboardCrossAccountRole"
    },
    {
      "Sid": "STSGetCallerIdentity",
      "Effect": "Allow",
      "Action": [
        "sts:GetCallerIdentity"
      ],
      "Resource": "*"
    }
  ]
}
EOF
)

# Step 1: Create IAM Role
echo "→ Creating IAM role: $ROLE_NAME"

if aws iam get-role --profile "$PROFILE" --role-name "$ROLE_NAME" &>/dev/null; then
    echo -e "  ${YELLOW}⚠${NC} Role already exists, updating trust policy..."

    # Update trust policy
    echo "$TRUST_POLICY" > /tmp/dashboard-lambda-trust-policy.json
    aws iam update-assume-role-policy \
        --profile "$PROFILE" \
        --role-name "$ROLE_NAME" \
        --policy-document file:///tmp/dashboard-lambda-trust-policy.json
    rm -f /tmp/dashboard-lambda-trust-policy.json
else
    # Create role
    echo "$TRUST_POLICY" > /tmp/dashboard-lambda-trust-policy.json
    aws iam create-role \
        --profile "$PROFILE" \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document file:///tmp/dashboard-lambda-trust-policy.json \
        --description "IAM role for spawn-dashboard-api Lambda function" \
        --tags Key=spawn:managed,Value=true \
        --output json > /dev/null
    rm -f /tmp/dashboard-lambda-trust-policy.json

    echo -e "  ${GREEN}✓${NC} Role created"
fi

# Step 2: Attach permissions policy
echo "→ Attaching permissions policy: SpawnDashboardLambdaPolicy"

echo "$PERMISSIONS_POLICY" > /tmp/dashboard-lambda-permissions-policy.json
aws iam put-role-policy \
    --profile "$PROFILE" \
    --role-name "$ROLE_NAME" \
    --policy-name "SpawnDashboardLambdaPolicy" \
    --policy-document file:///tmp/dashboard-lambda-permissions-policy.json
rm -f /tmp/dashboard-lambda-permissions-policy.json

echo -e "  ${GREEN}✓${NC} Policy attached"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Created Resources:"
echo "  • IAM Role: $ROLE_NAME"
echo "  • Account:  $ACCOUNT_ID"
echo ""
echo "Role ARN:"
ROLE_ARN=$(aws iam get-role --profile "$PROFILE" --role-name "$ROLE_NAME" --query 'Role.Arn' --output text)
echo "  $ROLE_ARN"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "What This Role Does:"
echo "  ✓ Allows Lambda to write CloudWatch Logs"
echo "  ✓ Allows Lambda to read/write DynamoDB (spawn-user-accounts table)"
echo "  ✓ Allows Lambda to assume cross-account role for EC2 access"
echo "  ✓ Allows Lambda to call STS GetCallerIdentity"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next Steps:"
echo "  1. Create DynamoDB table: spawn-user-accounts"
echo "  2. Build and deploy Lambda function: spawn-dashboard-api"
echo "  3. Create API Gateway integration"
echo ""
echo "To verify:"
echo "  aws iam get-role --role-name $ROLE_NAME --profile $PROFILE"
echo ""
