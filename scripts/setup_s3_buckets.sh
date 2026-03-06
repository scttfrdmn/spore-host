#!/bin/bash
set -e

# Setup S3 buckets for spawnd binaries in a central account
# Usage: ./setup_s3_buckets.sh [aws-profile] [regions...]
#
# Examples:
#   ./setup_s3_buckets.sh my-spawn-account us-east-1 us-west-2
#   ./setup_s3_buckets.sh my-spawn-account all  # All regions

PROFILE=${1:-spore-host-infra}
shift

# Default regions if not specified
if [ $# -eq 0 ]; then
    # Default: US regions only
    REGIONS=(
        us-east-1
        us-east-2
        us-west-1
        us-west-2
    )
elif [ "$1" = "all" ]; then
    # All major regions
    REGIONS=(
        us-east-1
        us-east-2
        us-west-1
        us-west-2
        eu-west-1
        eu-west-2
        eu-central-1
        ap-southeast-1
        ap-southeast-2
        ap-northeast-1
    )
else
    REGIONS=("$@")
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up spawn S3 buckets"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile: $PROFILE"
echo "Regions: ${REGIONS[*]}"
echo ""

# Get account ID
ACCOUNT_ID=$(aws sts get-caller-identity --profile "$PROFILE" --query Account --output text)
echo "Account ID: $ACCOUNT_ID"
echo ""

# Create bucket policy (public read for spawnd binaries)
create_bucket_policy() {
    local bucket_name=$1
    cat > /tmp/spawn-bucket-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadSpawndBinaries",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::${bucket_name}/spawnd-*"
    }
  ]
}
EOF
}

# Create buckets in each region
for region in "${REGIONS[@]}"; do
    bucket_name="spawn-binaries-${region}"

    echo "→ Creating bucket: ${bucket_name}"

    # Create bucket
    if [ "$region" = "us-east-1" ]; then
        # us-east-1 doesn't need LocationConstraint
        aws s3api create-bucket \
            --profile "$PROFILE" \
            --bucket "$bucket_name" \
            --region "$region" 2>/dev/null || echo "  Bucket already exists"
    else
        aws s3api create-bucket \
            --profile "$PROFILE" \
            --bucket "$bucket_name" \
            --region "$region" \
            --create-bucket-configuration LocationConstraint="$region" 2>/dev/null || echo "  Bucket already exists"
    fi

    # Block public access settings (but allow bucket policy)
    echo "  Configuring public access block..."
    aws s3api put-public-access-block \
        --profile "$PROFILE" \
        --bucket "$bucket_name" \
        --public-access-block-configuration \
            "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=false,RestrictPublicBuckets=false" \
        --region "$region"

    # Apply bucket policy
    echo "  Applying bucket policy..."
    create_bucket_policy "$bucket_name"
    aws s3api put-bucket-policy \
        --profile "$PROFILE" \
        --bucket "$bucket_name" \
        --policy file:///tmp/spawn-bucket-policy.json \
        --region "$region"

    # Enable versioning
    echo "  Enabling versioning..."
    aws s3api put-bucket-versioning \
        --profile "$PROFILE" \
        --bucket "$bucket_name" \
        --versioning-configuration Status=Enabled \
        --region "$region"

    # Add lifecycle policy (optional: clean up old versions after 90 days)
    echo "  Configuring lifecycle policy..."
    cat > /tmp/spawn-lifecycle.json <<EOF
{
  "Rules": [
    {
      "ID": "DeleteOldVersions",
      "Status": "Enabled",
      "Filter": {},
      "NoncurrentVersionExpiration": {
        "NoncurrentDays": 90
      }
    }
  ]
}
EOF
    aws s3api put-bucket-lifecycle-configuration \
        --profile "$PROFILE" \
        --bucket "$bucket_name" \
        --lifecycle-configuration file:///tmp/spawn-lifecycle.json \
        --region "$region" 2>/dev/null || echo "  (Lifecycle policy skipped)"

    echo "  ✅ ${bucket_name} ready"
    echo ""
done

# Cleanup
rm -f /tmp/spawn-bucket-policy.json /tmp/spawn-lifecycle.json

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ Setup complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo "  1. Upload binaries: ./upload_spawnd.sh $PROFILE"
echo "  2. Test download:"
echo "     curl -O https://spawn-binaries-us-east-1.s3.amazonaws.com/spawnd-linux-amd64"
echo ""
