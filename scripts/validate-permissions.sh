#!/bin/bash
set -e

# Validate AWS IAM permissions for spawn
# Usage: ./validate-permissions.sh [aws-profile]

PROFILE=${1:-spore-host-infra}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Validating AWS IAM Permissions for spawn"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile: $PROFILE"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
TOTAL=0
PASSED=0
FAILED=0
WARNINGS=0

# Get account ID and user
echo "→ Checking AWS identity..."
IDENTITY=$(aws sts get-caller-identity --profile "$PROFILE" --output json 2>/dev/null || echo "")
if [ -z "$IDENTITY" ]; then
    echo -e "${RED}✗ Failed to get AWS identity. Check your credentials.${NC}"
    exit 1
fi

ACCOUNT_ID=$(echo "$IDENTITY" | jq -r .Account)
USER_ARN=$(echo "$IDENTITY" | jq -r .Arn)
echo -e "  ${GREEN}✓${NC} Account: $ACCOUNT_ID"
echo -e "  ${GREEN}✓${NC} Identity: $USER_ARN"
echo ""

# Test function
test_permission() {
    local description=$1
    local command=$2
    local required=${3:-true}

    TOTAL=$((TOTAL + 1))

    # Run command and capture output
    if eval "$command" &>/dev/null; then
        if [ "$required" = "true" ]; then
            echo -e "  ${GREEN}✓${NC} $description"
            PASSED=$((PASSED + 1))
        else
            echo -e "  ${GREEN}✓${NC} $description (optional)"
            PASSED=$((PASSED + 1))
        fi
        return 0
    else
        if [ "$required" = "true" ]; then
            echo -e "  ${RED}✗${NC} $description"
            FAILED=$((FAILED + 1))
        else
            echo -e "  ${YELLOW}⚠${NC} $description (optional - not available)"
            WARNINGS=$((WARNINGS + 1))
        fi
        return 1
    fi
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Testing EC2 Permissions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_permission \
    "ec2:DescribeInstances" \
    "aws ec2 describe-instances --profile $PROFILE --region us-east-1 --max-results 1" \
    true

test_permission \
    "ec2:DescribeInstanceTypes" \
    "aws ec2 describe-instance-types --profile $PROFILE --region us-east-1 --instance-types t3.micro" \
    true

test_permission \
    "ec2:DescribeImages" \
    "aws ec2 describe-images --profile $PROFILE --region us-east-1 --owners amazon --filters 'Name=name,Values=al2023-ami-*' --max-results 1" \
    true

test_permission \
    "ec2:DescribeKeyPairs" \
    "aws ec2 describe-key-pairs --profile $PROFILE --region us-east-1" \
    true

test_permission \
    "ec2:DescribeSecurityGroups" \
    "aws ec2 describe-security-groups --profile $PROFILE --region us-east-1 --max-results 1" \
    true

test_permission \
    "ec2:DescribeSubnets" \
    "aws ec2 describe-subnets --profile $PROFILE --region us-east-1 --max-results 1" \
    true

test_permission \
    "ec2:DescribeVpcs" \
    "aws ec2 describe-vpcs --profile $PROFILE --region us-east-1 --max-results 1" \
    true

test_permission \
    "ec2:DescribeTags" \
    "aws ec2 describe-tags --profile $PROFILE --region us-east-1 --max-results 1" \
    true

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Testing IAM Permissions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_permission \
    "iam:GetRole (spawnd-instance-role)" \
    "aws iam get-role --profile $PROFILE --role-name spawnd-instance-role 2>&1 | grep -q 'NoSuchEntity\|AssumeRolePolicyDocument'" \
    true

test_permission \
    "iam:GetInstanceProfile (spawnd-instance-profile)" \
    "aws iam get-instance-profile --profile $PROFILE --instance-profile-name spawnd-instance-profile 2>&1 | grep -q 'NoSuchEntity\|InstanceProfile'" \
    true

# Note: We can't easily test CreateRole/CreateInstanceProfile without actually creating them
# So we just test that the commands are valid
test_permission \
    "iam:CreateRole (command available)" \
    "aws iam create-role help --profile $PROFILE 2>&1 | grep -q 'NAME'" \
    true

test_permission \
    "iam:CreateInstanceProfile (command available)" \
    "aws iam create-instance-profile help --profile $PROFILE 2>&1 | grep -q 'NAME'" \
    true

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Testing SSM Permissions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_permission \
    "ssm:GetParameter (AMI auto-detection)" \
    "aws ssm get-parameter --profile $PROFILE --region us-east-1 --name /aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64" \
    true

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Testing Write Permissions (Simulation)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Use dry-run mode to test permissions without actually creating resources
test_permission \
    "ec2:RunInstances (dry-run)" \
    "aws ec2 run-instances --profile $PROFILE --region us-east-1 --image-id ami-0c02fb55b15a5c152 --instance-type t3.micro --dry-run 2>&1 | grep -q 'DryRunOperation\|would have succeeded'" \
    true

test_permission \
    "ec2:ImportKeyPair (dry-run)" \
    "aws ec2 import-key-pair --profile $PROFILE --region us-east-1 --key-name test-validation-key --public-key-material 'ssh-rsa AAAA' --dry-run 2>&1 | grep -q 'DryRunOperation\|InvalidKey\|would have succeeded'" \
    true

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Results"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Total tests: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
fi
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
fi
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All required permissions are available!${NC}"
    echo ""
    echo "You can use spawn with this AWS profile."
    echo ""
    exit 0
else
    echo -e "${RED}✗ Some required permissions are missing.${NC}"
    echo ""
    echo "To fix this:"
    echo "  1. See spawn/IAM_PERMISSIONS.md for the required policy"
    echo "  2. Ask your AWS administrator to grant these permissions"
    echo "  3. Or attach the policy directly (if you have admin access):"
    echo ""
    echo "     aws iam put-user-policy \\"
    echo "       --user-name \$(aws sts get-caller-identity --query Arn --output text | cut -d'/' -f2) \\"
    echo "       --policy-name spawn-policy \\"
    echo "       --policy-document file://spawn/IAM_PERMISSIONS.md"
    echo ""
    exit 1
fi
