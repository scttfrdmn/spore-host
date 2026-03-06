#!/bin/bash
# Create initial CLI user record with email mapping for scott-admin

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <email>"
    echo "Example: $0 scttfrdmn@gmail.com"
    exit 1
fi

EMAIL="$1"
PROFILE="spore-host-infra"
REGION="us-east-1"
TABLE_NAME="spawn-user-accounts"

# Fixed values for scott-admin
USER_ID="arn:aws:iam::435415984226:user/scott-admin"
CLI_IAM_ARN="arn:aws:iam::435415984226:user/scott-admin"
IDENTITY_TYPE="cli"
AWS_ACCOUNT_ID="435415984226"
ACCOUNT_BASE36="c8s8u"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "Creating user mapping for scott-admin..."
echo "  User ID: $USER_ID"
echo "  CLI IAM ARN: $CLI_IAM_ARN"
echo "  Email: $EMAIL"
echo "  Identity Type: $IDENTITY_TYPE"
echo ""

# Create/update user record
aws dynamodb put-item \
  --profile "$PROFILE" \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --item "{
    \"user_id\": {\"S\": \"$USER_ID\"},
    \"cli_iam_arn\": {\"S\": \"$CLI_IAM_ARN\"},
    \"email\": {\"S\": \"$EMAIL\"},
    \"identity_type\": {\"S\": \"$IDENTITY_TYPE\"},
    \"aws_account_id\": {\"S\": \"$AWS_ACCOUNT_ID\"},
    \"account_base36\": {\"S\": \"$ACCOUNT_BASE36\"},
    \"linked_at\": {\"S\": \"$TIMESTAMP\"},
    \"created_at\": {\"S\": \"$TIMESTAMP\"},
    \"last_access\": {\"S\": \"$TIMESTAMP\"}
  }"

echo "✓ User mapping created successfully"
echo ""
echo "Verification:"
aws dynamodb get-item \
  --profile "$PROFILE" \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --key "{\"user_id\": {\"S\": \"$USER_ID\"}}" \
  --query 'Item.{UserID:user_id.S,Email:email.S,CliIamArn:cli_iam_arn.S,IdentityType:identity_type.S}' \
  --output table
