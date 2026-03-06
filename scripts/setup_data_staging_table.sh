#!/bin/bash
set -euo pipefail

# Create DynamoDB table for data staging metadata
# Usage: ./setup_data_staging_table.sh

REGION="us-east-1"

echo "Creating spawn-staged-data DynamoDB table..."

aws dynamodb create-table \
    --table-name spawn-staged-data \
    --attribute-definitions \
        AttributeName=staging_id,AttributeType=S \
    --key-schema \
        AttributeName=staging_id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --region "$REGION" \
    --profile spore-host-infra \
    --tags \
        Key=Project,Value=spawn \
        Key=Component,Value=data-staging \
        Key=ManagedBy,Value=spawn \
    2>/dev/null || echo "(table already exists)"

# Add TTL attribute for automatic cleanup
aws dynamodb update-time-to-live \
    --table-name spawn-staged-data \
    --time-to-live-specification "Enabled=true,AttributeName=ttl" \
    --region "$REGION" \
    --profile spore-host-infra \
    2>/dev/null || echo "(TTL already configured)"

echo "✓ Created spawn-staged-data table in ${REGION}"
echo ""
echo "Table: spawn-staged-data"
echo "  Key: staging_id (String)"
echo "  Billing: Pay-per-request"
echo "  TTL: Enabled (ttl attribute)"
