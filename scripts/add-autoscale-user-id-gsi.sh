#!/bin/bash
# Add user_id field and user_id-index GSI to spawn-autoscale-groups-production table

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <aws-profile>"
    echo "Example: $0 spore-host-infra"
    exit 1
fi

PROFILE="$1"
TABLE_NAME="spawn-autoscale-groups-production"
REGION="us-east-1"

echo "Adding user_id GSI to $TABLE_NAME..."

# Add GSI
aws dynamodb update-table \
  --profile "$PROFILE" \
  --region "$REGION" \
  --table-name "$TABLE_NAME" \
  --attribute-definitions \
    AttributeName=user_id,AttributeType=S \
  --global-secondary-index-updates '[
    {
      "Create": {
        "IndexName": "user_id-index",
        "KeySchema": [{"AttributeName": "user_id", "KeyType": "HASH"}],
        "Projection": {"ProjectionType": "ALL"},
        "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
      }
    }
  ]'

echo "✓ GSI creation initiated"
echo ""
echo "Waiting for GSI to become ACTIVE (this may take 5-10 minutes)..."

# Wait for GSI to become ACTIVE
while true; do
    STATUS=$(aws dynamodb describe-table \
        --profile "$PROFILE" \
        --region "$REGION" \
        --table-name "$TABLE_NAME" \
        --query 'Table.GlobalSecondaryIndexes[?IndexName==`user_id-index`].IndexStatus' \
        --output text)

    if [ "$STATUS" = "CREATING" ]; then
        echo "  Status: CREATING... (checking again in 30s)"
        sleep 30
    else
        break
    fi
done

echo "✓ GSI is now ACTIVE"
echo ""
echo "Verification:"
aws dynamodb describe-table \
    --profile "$PROFILE" \
    --region "$REGION" \
    --table-name "$TABLE_NAME" \
    --query 'Table.GlobalSecondaryIndexes[].{Name:IndexName,Status:IndexStatus}' \
    --output table
