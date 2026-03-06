#!/bin/bash
set -e

# Setup S3 lifecycle policy for MPI SSH key cleanup
# Usage: ./setup-mpi-s3-lifecycle.sh [aws-profile] [region]
#
# This sets up automatic cleanup of temporary MPI SSH keys stored in S3.
# Keys are automatically deleted after 1 day.

PROFILE=${1:-spore-host-infra}
REGION=${2:-us-east-1}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Setting up MPI S3 Lifecycle Policy"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Profile: $PROFILE"
echo "Region:  $REGION"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Bucket name
BUCKET="spawn-binaries-${REGION}"

# Check if bucket exists
echo "→ Checking S3 bucket..."
if ! aws s3 ls "s3://${BUCKET}" --profile "$PROFILE" >/dev/null 2>&1; then
    echo -e "${RED}✗ Bucket ${BUCKET} not found${NC}"
    echo "  Please create the spawn-binaries bucket first"
    exit 1
fi
echo -e "  ${GREEN}✓${NC} Bucket exists: ${BUCKET}"
echo ""

# Create lifecycle policy
echo "→ Creating lifecycle policy..."
LIFECYCLE_POLICY=$(cat <<'EOF'
{
  "Rules": [
    {
      "ID": "DeleteOldMPIKeys",
      "Status": "Enabled",
      "Filter": {
        "Prefix": "mpi-keys/"
      },
      "Expiration": {
        "Days": 1
      }
    }
  ]
}
EOF
)

# Apply lifecycle policy
aws s3api put-bucket-lifecycle-configuration \
  --bucket "$BUCKET" \
  --lifecycle-configuration "$LIFECYCLE_POLICY" \
  --profile "$PROFILE" 2>/dev/null || {
    echo -e "${RED}✗ Failed to apply lifecycle policy${NC}"
    exit 1
}

echo -e "  ${GREEN}✓${NC} Lifecycle policy applied"
echo ""

# Verify policy
echo "→ Verifying lifecycle policy..."
POLICY_JSON=$(aws s3api get-bucket-lifecycle-configuration \
  --bucket "$BUCKET" \
  --profile "$PROFILE" \
  --query 'Rules[?ID==`DeleteOldMPIKeys`]' \
  --output json 2>/dev/null)

if [ "$POLICY_JSON" = "[]" ] || [ -z "$POLICY_JSON" ]; then
    echo -e "${YELLOW}⚠  Warning: Could not verify lifecycle policy${NC}"
else
    echo -e "  ${GREEN}✓${NC} Policy verified"
    echo ""
    echo "Policy details:"
    echo "$POLICY_JSON" | jq -r '.[0] | "  • Prefix: \(.Filter.Prefix)\n  • Expiration: \(.Expiration.Days) days"'
fi

echo ""
echo -e "${GREEN}✅ MPI S3 lifecycle policy setup complete!${NC}"
echo ""
echo "MPI SSH keys will be automatically deleted after 1 day."
echo "Location: s3://${BUCKET}/mpi-keys/"
echo ""
