#!/bin/bash
set -euo pipefail

# Create regional S3 buckets for data staging
# Usage: ./setup_data_staging_buckets.sh

REGIONS=(
    "us-east-1"
    "us-west-2"
    "eu-west-1"
    "ap-northeast-1"
)

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text --profile spore-host-infra)

echo "Setting up spawn data staging buckets..."
echo "Account ID: ${ACCOUNT_ID}"
echo ""

for REGION in "${REGIONS[@]}"; do
    BUCKET="spawn-data-${REGION}"

    echo "Creating bucket: ${BUCKET} in ${REGION}"

    # Create bucket with region-specific configuration
    if [ "$REGION" == "us-east-1" ]; then
        aws s3api create-bucket \
            --bucket "$BUCKET" \
            --region "$REGION" \
            --profile spore-host-infra 2>/dev/null || echo "  (bucket already exists)"
    else
        aws s3api create-bucket \
            --bucket "$BUCKET" \
            --region "$REGION" \
            --create-bucket-configuration LocationConstraint="$REGION" \
            --profile spore-host-infra 2>/dev/null || echo "  (bucket already exists)"
    fi

    # Enable versioning
    aws s3api put-bucket-versioning \
        --bucket "$BUCKET" \
        --versioning-configuration Status=Enabled \
        --region "$REGION" \
        --profile spore-host-infra

    # Add lifecycle policy (delete after 7 days)
    aws s3api put-bucket-lifecycle-configuration \
        --bucket "$BUCKET" \
        --region "$REGION" \
        --profile spore-host-infra \
        --lifecycle-configuration '{
            "Rules": [{
                "Id": "DeleteOldStaging",
                "Status": "Enabled",
                "Filter": {
                    "Prefix": "staging/"
                },
                "Expiration": {
                    "Days": 7
                },
                "NoncurrentVersionExpiration": {
                    "NoncurrentDays": 1
                }
            }]
        }'

    # Add tags
    aws s3api put-bucket-tagging \
        --bucket "$BUCKET" \
        --region "$REGION" \
        --profile spore-host-infra \
        --tagging 'TagSet=[
            {Key=Project,Value=spawn},
            {Key=Component,Value=data-staging},
            {Key=ManagedBy,Value=spawn}
        ]'

    echo "  ✓ Created ${BUCKET} with 7-day lifecycle"
    echo ""
done

echo "✓ Data staging buckets setup complete"
echo ""
echo "Buckets created:"
for REGION in "${REGIONS[@]}"; do
    echo "  - spawn-data-${REGION}"
done
