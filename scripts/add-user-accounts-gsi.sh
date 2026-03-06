#!/bin/bash
# Add email-index and cli_iam_arn-index GSIs to spawn-user-accounts table

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <aws-profile>"
    echo "Example: $0 spore-host-infra"
    exit 1
fi

PROFILE="$1"
TABLE_NAME="spawn-user-accounts"
REGION="us-east-1"

echo "Adding GSIs to $TABLE_NAME..."
echo ""

# Create email-index GSI first
echo "1. Creating email-index GSI..."
aws dynamodb update-table \
  --profile "$PROFILE" \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --attribute-definitions AttributeName=email,AttributeType=S \
  --global-secondary-index-updates '[
    {
      "Create": {
        "IndexName": "email-index",
        "KeySchema": [{"AttributeName": "email", "KeyType": "HASH"}],
        "Projection": {"ProjectionType": "ALL"},
        "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
      }
    }
  ]'

echo "✓ email-index creation initiated"
echo "  Waiting for email-index to become ACTIVE (this may take 5-10 minutes)..."

# Wait for email-index to become ACTIVE
while true; do
    STATUS=$(aws dynamodb describe-table \
        --profile "$PROFILE" \
        --region "$REGION" \
        --table-name "$TABLE_NAME" \
        --query 'Table.GlobalSecondaryIndexes[?IndexName==`email-index`].IndexStatus' \
        --output text)

    if [ "$STATUS" = "CREATING" ]; then
        echo "    Status: CREATING... (checking again in 30s)"
        sleep 30
    elif [ "$STATUS" = "ACTIVE" ]; then
        echo "✓ email-index is now ACTIVE"
        break
    else
        echo "  Status: $STATUS"
        sleep 30
    fi
done

echo ""

# Create cli_iam_arn-index GSI second
echo "2. Creating cli_iam_arn-index GSI..."
aws dynamodb update-table \
  --profile "$PROFILE" \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --attribute-definitions AttributeName=cli_iam_arn,AttributeType=S \
  --global-secondary-index-updates '[
    {
      "Create": {
        "IndexName": "cli_iam_arn-index",
        "KeySchema": [{"AttributeName": "cli_iam_arn", "KeyType": "HASH"}],
        "Projection": {"ProjectionType": "ALL"},
        "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
      }
    }
  ]'

echo "✓ cli_iam_arn-index creation initiated"
echo "  Waiting for cli_iam_arn-index to become ACTIVE (this may take 5-10 minutes)..."

# Wait for cli_iam_arn-index to become ACTIVE
while true; do
    STATUS=$(aws dynamodb describe-table \
        --profile "$PROFILE" \
        --region "$REGION" \
        --table-name "$TABLE_NAME" \
        --query 'Table.GlobalSecondaryIndexes[?IndexName==`cli_iam_arn-index`].IndexStatus' \
        --output text)

    if [ "$STATUS" = "CREATING" ]; then
        echo "    Status: CREATING... (checking again in 30s)"
        sleep 30
    elif [ "$STATUS" = "ACTIVE" ]; then
        echo "✓ cli_iam_arn-index is now ACTIVE"
        break
    else
        echo "  Status: $STATUS"
        sleep 30
    fi
done

echo ""
echo "✓ Both GSIs are now ACTIVE"
echo ""
echo "Verification:"
aws dynamodb describe-table \
    --profile "$PROFILE" \
    --region "$REGION" \
    --table-name "$TABLE_NAME" \
    --query 'Table.GlobalSecondaryIndexes[].{Name:IndexName,Status:IndexStatus}' \
    --output table
