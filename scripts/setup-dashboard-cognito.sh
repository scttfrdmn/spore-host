#!/bin/bash
set -e

# Setup Cognito Identity Pool for Dashboard with Multiple OIDC Providers
# Supports: Globus Auth, Google, GitHub
# Usage: ./setup-dashboard-cognito.sh [aws-profile]

PROFILE=${1:-spore-host-infra}
IDENTITY_POOL_NAME="spawn-dashboard-identity-pool"
REGION="us-east-1"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up Cognito Identity Pool (Multi-OIDC)"
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
echo -e "  ${GREEN}✓${NC} Account: $ACCOUNT_ID"
echo ""

# OIDC Provider Configuration
echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  OIDC Provider Setup Instructions${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Before proceeding, you need to register OAuth applications:${NC}"
echo ""
echo -e "${BLUE}1. Globus Auth:${NC}"
echo "   → Go to: https://developers.globus.org/"
echo "   → Register new application"
echo "   → Redirect URI: https://spore.host/callback"
echo "   → Copy Client ID"
echo ""
echo -e "${BLUE}2. Google:${NC}"
echo "   → Go to: https://console.cloud.google.com/apis/credentials"
echo "   → Create OAuth 2.0 Client ID (Web application)"
echo "   → Authorized redirect URIs: https://spore.host/callback"
echo "   → Copy Client ID"
echo ""
echo -e "${BLUE}3. GitHub:${NC}"
echo "   → Go to: https://github.com/settings/developers"
echo "   → New OAuth App"
echo "   → Homepage URL: https://spore.host"
echo "   → Authorization callback URL: https://spore.host/callback"
echo "   → Copy Client ID"
echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
echo ""

# Prompt for OIDC provider details
echo -e "${YELLOW}Enter your OIDC provider Client IDs (press Enter to skip):${NC}"
echo ""

read -p "Globus Auth Client ID: " GLOBUS_CLIENT_ID
read -p "Google Client ID: " GOOGLE_CLIENT_ID
read -p "GitHub Client ID: " GITHUB_CLIENT_ID

echo ""

# Validate at least one provider
if [ -z "$GLOBUS_CLIENT_ID" ] && [ -z "$GOOGLE_CLIENT_ID" ] && [ -z "$GITHUB_CLIENT_ID" ]; then
    echo -e "${RED}✗ Error: At least one OIDC provider is required${NC}"
    echo "Please register an OAuth application and provide a Client ID"
    exit 1
fi

# Build OpenID Connect providers JSON
OIDC_PROVIDERS="{"
FIRST=true

if [ -n "$GLOBUS_CLIENT_ID" ]; then
    if [ "$FIRST" = true ]; then FIRST=false; else OIDC_PROVIDERS+=","; fi
    OIDC_PROVIDERS+="\"auth.globus.org\": \"$GLOBUS_CLIENT_ID\""
fi

if [ -n "$GOOGLE_CLIENT_ID" ]; then
    if [ "$FIRST" = true ]; then FIRST=false; else OIDC_PROVIDERS+=","; fi
    OIDC_PROVIDERS+="\"accounts.google.com\": \"$GOOGLE_CLIENT_ID\""
fi

if [ -n "$GITHUB_CLIENT_ID" ]; then
    if [ "$FIRST" = true ]; then FIRST=false; else OIDC_PROVIDERS+=","; fi
    # GitHub requires special handling - we'll use GitHub OAuth via API Gateway
    echo -e "${YELLOW}⚠ Note: GitHub OAuth will be handled separately (not pure OIDC)${NC}"
fi

OIDC_PROVIDERS+="}"

echo ""
echo "→ Creating Cognito Identity Pool..."

# Check if identity pool already exists
EXISTING_POOL=$(aws cognito-identity list-identity-pools \
    --max-results 60 \
    --profile "$PROFILE" \
    --region "$REGION" \
    --query "IdentityPools[?IdentityPoolName=='$IDENTITY_POOL_NAME'].IdentityPoolId" \
    --output text 2>/dev/null || echo "")

if [ -n "$EXISTING_POOL" ]; then
    echo -e "  ${YELLOW}⚠${NC} Identity Pool already exists: $EXISTING_POOL"
    IDENTITY_POOL_ID="$EXISTING_POOL"

    # Update existing pool
    echo "→ Updating existing Identity Pool..."
    aws cognito-identity update-identity-pool \
        --identity-pool-id "$IDENTITY_POOL_ID" \
        --identity-pool-name "$IDENTITY_POOL_NAME" \
        --allow-unauthenticated-identities \
        --allow-classic-flow \
        --open-id-connect-provider-arns \
        --supported-login-providers "$OIDC_PROVIDERS" \
        --profile "$PROFILE" \
        --region "$REGION" \
        --output json > /dev/null

    echo -e "  ${GREEN}✓${NC} Identity Pool updated"
else
    # Create new pool
    IDENTITY_POOL_ID=$(aws cognito-identity create-identity-pool \
        --identity-pool-name "$IDENTITY_POOL_NAME" \
        --allow-unauthenticated-identities \
        --allow-classic-flow \
        --supported-login-providers "$OIDC_PROVIDERS" \
        --profile "$PROFILE" \
        --region "$REGION" \
        --query 'IdentityPoolId' \
        --output text)

    echo -e "  ${GREEN}✓${NC} Identity Pool created: $IDENTITY_POOL_ID"
fi

echo ""
echo "→ Creating IAM roles for authenticated users..."

# Authenticated role trust policy
cat > /tmp/cognito-auth-trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "cognito-identity.amazonaws.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "cognito-identity.amazonaws.com:aud": "$IDENTITY_POOL_ID"
        },
        "ForAnyValue:StringLike": {
          "cognito-identity.amazonaws.com:amr": "authenticated"
        }
      }
    }
  ]
}
EOF

# Authenticated role permissions (EC2 read for client-side queries)
cat > /tmp/cognito-auth-permissions.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EC2ReadAccess",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeRegions",
        "ec2:DescribeTags"
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

# Create authenticated role
AUTH_ROLE_NAME="Cognito_SpawnDashboard_Auth_Role"

if aws iam get-role --role-name "$AUTH_ROLE_NAME" --profile "$PROFILE" &>/dev/null; then
    echo -e "  ${YELLOW}⚠${NC} Role already exists: $AUTH_ROLE_NAME"

    # Update trust policy
    aws iam update-assume-role-policy \
        --role-name "$AUTH_ROLE_NAME" \
        --policy-document file:///tmp/cognito-auth-trust-policy.json \
        --profile "$PROFILE"
else
    aws iam create-role \
        --role-name "$AUTH_ROLE_NAME" \
        --assume-role-policy-document file:///tmp/cognito-auth-trust-policy.json \
        --description "Role for authenticated Cognito Identity Pool users" \
        --profile "$PROFILE" \
        --output json > /dev/null

    echo -e "  ${GREEN}✓${NC} Authenticated role created"
fi

# Attach permissions policy
aws iam put-role-policy \
    --role-name "$AUTH_ROLE_NAME" \
    --policy-name "SpawnDashboardAPIAccess" \
    --policy-document file:///tmp/cognito-auth-permissions.json \
    --profile "$PROFILE"

echo -e "  ${GREEN}✓${NC} Permissions attached"

# Get role ARN
AUTH_ROLE_ARN=$(aws iam get-role --role-name "$AUTH_ROLE_NAME" --profile "$PROFILE" --query 'Role.Arn' --output text)

# Unauthenticated role (minimal permissions)
UNAUTH_ROLE_NAME="Cognito_SpawnDashboard_Unauth_Role"

cat > /tmp/cognito-unauth-trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "cognito-identity.amazonaws.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "cognito-identity.amazonaws.com:aud": "$IDENTITY_POOL_ID"
        },
        "ForAnyValue:StringLike": {
          "cognito-identity.amazonaws.com:amr": "unauthenticated"
        }
      }
    }
  ]
}
EOF

if aws iam get-role --role-name "$UNAUTH_ROLE_NAME" --profile "$PROFILE" &>/dev/null; then
    echo -e "  ${YELLOW}⚠${NC} Unauthenticated role already exists"
else
    aws iam create-role \
        --role-name "$UNAUTH_ROLE_NAME" \
        --assume-role-policy-document file:///tmp/cognito-unauth-trust-policy.json \
        --description "Role for unauthenticated Cognito Identity Pool users" \
        --profile "$PROFILE" \
        --output json > /dev/null

    echo -e "  ${GREEN}✓${NC} Unauthenticated role created"
fi

UNAUTH_ROLE_ARN=$(aws iam get-role --role-name "$UNAUTH_ROLE_NAME" --profile "$PROFILE" --query 'Role.Arn' --output text)

# Set identity pool roles
echo ""
echo "→ Configuring Identity Pool roles..."

aws cognito-identity set-identity-pool-roles \
    --identity-pool-id "$IDENTITY_POOL_ID" \
    --roles authenticated="$AUTH_ROLE_ARN",unauthenticated="$UNAUTH_ROLE_ARN" \
    --profile "$PROFILE" \
    --region "$REGION"

echo -e "  ${GREEN}✓${NC} Roles configured"

# Cleanup temp files
rm -f /tmp/cognito-*.json

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Cognito Identity Pool Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Identity Pool Details:"
echo "  Pool ID:  $IDENTITY_POOL_ID"
echo "  Region:   $REGION"
echo ""
echo "Configured Providers:"
[ -n "$GLOBUS_CLIENT_ID" ] && echo "  ✓ Globus Auth (auth.globus.org)"
[ -n "$GOOGLE_CLIENT_ID" ] && echo "  ✓ Google (accounts.google.com)"
[ -n "$GITHUB_CLIENT_ID" ] && echo "  ⚠ GitHub (requires custom OAuth flow)"
echo ""
echo "IAM Roles:"
echo "  Authenticated:   $AUTH_ROLE_ARN"
echo "  Unauthenticated: $UNAUTH_ROLE_ARN"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next Steps:"
echo "  1. Update web/js/auth.js with Identity Pool ID"
echo "  2. Configure OAuth redirect URLs in each provider"
echo "  3. Test authentication flow for each provider"
echo ""
echo "Configuration for frontend:"
echo ""
echo "const AWS_CONFIG = {"
echo "  region: '$REGION',"
echo "  identityPoolId: '$IDENTITY_POOL_ID'"
echo "};"
echo ""
