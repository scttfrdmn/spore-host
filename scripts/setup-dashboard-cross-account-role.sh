#!/bin/bash
set -e

# Setup Cross-Account IAM Role for Dashboard in Development Account
# This role allows Cognito authenticated users from the infrastructure account
# to query EC2 instances in the development account.
#
# Usage: ./setup-dashboard-cross-account-role.sh [profile]

PROFILE=${1:-spore-host-dev}
INFRA_ACCOUNT_ID="966362334030"  # spore-host-infra account
REGION="us-east-1"
ROLE_NAME="SpawnDashboardCrossAccountReadRole"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up Cross-Account Dashboard Role"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile: $PROFILE"
echo "Region:  $REGION"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get account ID
echo "→ Checking AWS identity..."
ACCOUNT_ID=$(aws sts get-caller-identity --profile "$PROFILE" --query Account --output text 2>/dev/null || echo "")
if [ -z "$ACCOUNT_ID" ]; then
    echo -e "${RED}✗ Failed to get AWS identity. Check your credentials.${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Account: $ACCOUNT_ID (spore-host-dev)"
echo ""

# Get Cognito Identity Pool ID from infrastructure account
echo "→ Getting Cognito Identity Pool ID from infrastructure account..."
IDENTITY_POOL_ID=$(aws cognito-identity list-identity-pools \
    --max-results 60 \
    --profile spore-host-infra \
    --region "$REGION" \
    --query "IdentityPools[?IdentityPoolName=='spawn-dashboard-identity-pool'].IdentityPoolId" \
    --output text 2>/dev/null || echo "")

if [ -z "$IDENTITY_POOL_ID" ]; then
    echo -e "${RED}✗ Cognito Identity Pool not found in infrastructure account${NC}"
    echo "Run setup-dashboard-cognito.sh first to create the Identity Pool"
    exit 1
fi

echo -e "  ${GREEN}✓${NC} Identity Pool: $IDENTITY_POOL_ID"
echo ""

# Create trust policy that allows Cognito authenticated IAM role from infra account
echo "→ Creating IAM role trust policy..."

cat > /tmp/dashboard-cross-account-trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::$INFRA_ACCOUNT_ID:role/Cognito_SpawnDashboard_Auth_Role"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

echo -e "  ${GREEN}✓${NC} Trust policy created"

# Create permissions policy for EC2 read access
echo "→ Creating IAM role permissions policy..."

cat > /tmp/dashboard-cross-account-permissions.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EC2DescribeAll",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeRegions",
        "ec2:DescribeTags",
        "ec2:DescribeInstanceStatus"
      ],
      "Resource": "*"
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

echo -e "  ${GREEN}✓${NC} Permissions policy created"

# Create or update the IAM role
echo "→ Creating IAM role..."

if aws iam get-role --role-name "$ROLE_NAME" --profile "$PROFILE" &>/dev/null; then
    echo -e "  ${YELLOW}⚠${NC} Role already exists: $ROLE_NAME"

    # Update trust policy
    echo "→ Updating trust policy..."
    aws iam update-assume-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-document file:///tmp/dashboard-cross-account-trust-policy.json \
        --profile "$PROFILE"

    echo -e "  ${GREEN}✓${NC} Trust policy updated"
else
    # Create new role
    aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document file:///tmp/dashboard-cross-account-trust-policy.json \
        --description "Cross-account role for Spawn dashboard to query EC2 instances" \
        --profile "$PROFILE" \
        --output json > /dev/null

    echo -e "  ${GREEN}✓${NC} Role created: $ROLE_NAME"
fi

# Attach or update permissions policy
echo "→ Attaching permissions policy..."

aws iam put-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-name "SpawnDashboardEC2ReadAccess" \
    --policy-document file:///tmp/dashboard-cross-account-permissions.json \
    --profile "$PROFILE"

echo -e "  ${GREEN}✓${NC} Permissions attached"

# Get role ARN
ROLE_ARN=$(aws iam get-role --role-name "$ROLE_NAME" --profile "$PROFILE" --query 'Role.Arn' --output text)

# Cleanup temp files
rm -f /tmp/dashboard-cross-account-*.json

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Cross-Account Role Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Role Details:"
echo "  Name: $ROLE_NAME"
echo "  ARN:  $ROLE_ARN"
echo ""
echo "Permissions:"
echo "  ✓ EC2 read access (all instances)"
echo "  ✓ STS GetCallerIdentity"
echo ""
echo "Trust Policy:"
echo "  ✓ Cognito Identity Pool: $IDENTITY_POOL_ID"
echo "  ✓ Infrastructure Account: $INFRA_ACCOUNT_ID"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
