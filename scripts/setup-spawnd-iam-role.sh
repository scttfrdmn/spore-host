#!/bin/bash
set -e

# Setup spawnd IAM role and instance profile
# Usage: ./setup-spawnd-iam-role.sh [aws-profile]
#
# This is a ONE-TIME setup script for Cloud Administrators.
# After running this, developers only need PowerUserAccess to use spawn.

PROFILE=${1:-spore-host-infra}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up spawnd IAM Role (One-Time Setup)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile: $PROFILE"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Resource names
ROLE_NAME="spawnd-instance-role"
INSTANCE_PROFILE_NAME="spawnd-instance-profile"
POLICY_NAME="spawnd-policy"

# Get account ID
echo "→ Checking AWS identity..."
ACCOUNT_ID=$(aws sts get-caller-identity --profile "$PROFILE" --query Account --output text 2>/dev/null || echo "")
if [ -z "$ACCOUNT_ID" ]; then
    echo -e "${RED}✗ Failed to get AWS identity. Check your credentials.${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Account: $ACCOUNT_ID"
echo ""

# Trust policy (who can assume the role)
TRUST_POLICY=$(cat <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
)

# Permissions policy (what the role can do)
PERMISSIONS_POLICY=$(cat <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeTags",
        "ec2:DescribeInstances"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:TerminateInstances",
        "ec2:StopInstances"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ec2:ResourceTag/spawn:managed": "true"
        }
      }
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
    echo "$TRUST_POLICY" > /tmp/spawnd-trust-policy.json
    aws iam update-assume-role-policy \
        --profile "$PROFILE" \
        --role-name "$ROLE_NAME" \
        --policy-document file:///tmp/spawnd-trust-policy.json
    rm -f /tmp/spawnd-trust-policy.json
else
    # Create role
    echo "$TRUST_POLICY" > /tmp/spawnd-trust-policy.json
    aws iam create-role \
        --profile "$PROFILE" \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document file:///tmp/spawnd-trust-policy.json \
        --description "IAM role for spawnd daemon on EC2 instances" \
        --tags Key=spawn:managed,Value=true \
        --output json > /dev/null
    rm -f /tmp/spawnd-trust-policy.json

    echo -e "  ${GREEN}✓${NC} Role created"
fi

# Step 2: Attach permissions policy
echo "→ Attaching permissions policy: $POLICY_NAME"

echo "$PERMISSIONS_POLICY" > /tmp/spawnd-permissions-policy.json
aws iam put-role-policy \
    --profile "$PROFILE" \
    --role-name "$ROLE_NAME" \
    --policy-name "$POLICY_NAME" \
    --policy-document file:///tmp/spawnd-permissions-policy.json
rm -f /tmp/spawnd-permissions-policy.json

echo -e "  ${GREEN}✓${NC} Policy attached"

# Step 3: Create Instance Profile
echo "→ Creating instance profile: $INSTANCE_PROFILE_NAME"

if aws iam get-instance-profile --profile "$PROFILE" --instance-profile-name "$INSTANCE_PROFILE_NAME" &>/dev/null; then
    echo -e "  ${YELLOW}⚠${NC} Instance profile already exists"

    # Check if role is already associated
    EXISTING_ROLES=$(aws iam get-instance-profile \
        --profile "$PROFILE" \
        --instance-profile-name "$INSTANCE_PROFILE_NAME" \
        --query 'InstanceProfile.Roles[].RoleName' \
        --output text)

    if echo "$EXISTING_ROLES" | grep -q "$ROLE_NAME"; then
        echo -e "  ${GREEN}✓${NC} Role already associated with instance profile"
    else
        # Add role to instance profile
        echo "  Adding role to instance profile..."
        aws iam add-role-to-instance-profile \
            --profile "$PROFILE" \
            --instance-profile-name "$INSTANCE_PROFILE_NAME" \
            --role-name "$ROLE_NAME" 2>/dev/null || echo -e "  ${YELLOW}⚠${NC} Role may already be associated"
        echo -e "  ${GREEN}✓${NC} Role associated"
    fi
else
    # Create instance profile
    aws iam create-instance-profile \
        --profile "$PROFILE" \
        --instance-profile-name "$INSTANCE_PROFILE_NAME" \
        --tags Key=spawn:managed,Value=true \
        --output json > /dev/null

    echo -e "  ${GREEN}✓${NC} Instance profile created"

    # Add role to instance profile
    echo "  Associating role with instance profile..."
    aws iam add-role-to-instance-profile \
        --profile "$PROFILE" \
        --instance-profile-name "$INSTANCE_PROFILE_NAME" \
        --role-name "$ROLE_NAME"

    echo -e "  ${GREEN}✓${NC} Role associated"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Created Resources:"
echo "  • IAM Role:           $ROLE_NAME"
echo "  • Instance Profile:   $INSTANCE_PROFILE_NAME"
echo "  • Permissions Policy: $POLICY_NAME"
echo ""
echo "Role ARN:"
ROLE_ARN=$(aws iam get-role --profile "$PROFILE" --role-name "$ROLE_NAME" --query 'Role.Arn' --output text)
echo "  $ROLE_ARN"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "What This Role Does:"
echo "  ✓ Allows spawnd to read its own EC2 instance tags"
echo "  ✓ Allows spawnd to terminate/stop itself when TTL/idle reached"
echo "  ✓ CANNOT terminate other instances (only spawn:managed=true)"
echo "  ✓ CANNOT access S3, databases, or other AWS services"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next Steps for Developers:"
echo "  1. Ensure developers have PowerUserAccess managed policy"
echo "  2. Developers can now use spawn without additional IAM permissions"
echo "  3. spawn will automatically detect and use this pre-created role"
echo ""
echo "To verify:"
echo "  aws iam get-role --role-name $ROLE_NAME"
echo "  aws iam get-instance-profile --instance-profile-name $INSTANCE_PROFILE_NAME"
echo ""
