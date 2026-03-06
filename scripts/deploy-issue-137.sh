#!/bin/bash
# Deploy Issue #137: Per-User Resource Isolation
# This script orchestrates the full deployment in the correct order

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================="
echo "Issue #137: Per-User Resource Isolation"
echo "========================================="
echo ""

# ============================================================
# PHASE 1: Infrastructure (can run in parallel)
# ============================================================
echo "PHASE 1: Infrastructure Setup"
echo "---------------------------------------------"
echo ""

echo "1.1. Adding GSIs to spawn-user-accounts table..."
"$SCRIPT_DIR/add-user-accounts-gsi.sh" spore-host-infra
echo ""

echo "1.2. Adding user_id field and GSI to autoscale groups table..."
"$SCRIPT_DIR/add-autoscale-user-id-gsi.sh" spore-host-infra
echo ""

echo "✓ Phase 1 complete - Infrastructure updated"
echo ""

# ============================================================
# PHASE 2: Data Backfill
# ============================================================
echo "PHASE 2: Data Backfill"
echo "---------------------------------------------"
echo ""

echo "2.1. Backfilling autoscale groups with user_id..."
"$SCRIPT_DIR/backfill-autoscale-user-ids.py" spore-host-infra spore-host-dev
echo ""

echo "2.2. Creating initial user mapping for scott-admin..."
"$SCRIPT_DIR/create-user-mapping.sh" scttfrdmn@gmail.com
echo ""

echo "2.3. Verifying backfill..."
COUNT=$(AWS_PROFILE=spore-host-dev aws dynamodb scan \
  --table-name spawn-autoscale-groups-production \
  --region us-east-1 \
  --filter-expression "attribute_not_exists(user_id)" \
  --select COUNT \
  --query "Count" \
  --output text)

if [ "$COUNT" = "0" ]; then
    echo "✓ All autoscale groups have user_id"
else
    echo "⚠ Warning: $COUNT autoscale groups missing user_id"
fi
echo ""

echo "✓ Phase 2 complete - Data backfilled"
echo ""

# ============================================================
# PHASE 3: Lambda Code Deploy
# ============================================================
echo "PHASE 3: Lambda Code Deployment"
echo "---------------------------------------------"
echo ""

LAMBDA_DIR="$SCRIPT_DIR/../spawn/lambda/dashboard-api"

echo "3.1. Building Lambda function..."
cd "$LAMBDA_DIR"
GOOS=linux GOARCH=amd64 go build -o bootstrap .
zip -q dashboard-api.zip bootstrap
echo "✓ Built dashboard-api.zip"
echo ""

echo "3.2. Deploying to Lambda..."
AWS_PROFILE=spore-host-infra aws lambda update-function-code \
  --function-name spawn-dashboard-api \
  --zip-file fileb://dashboard-api.zip \
  --region us-east-1 \
  --query '{FunctionName:FunctionName,LastModified:LastModified,Version:Version}' \
  --output table

echo ""
echo "3.3. Cleaning up build artifacts..."
rm -f bootstrap dashboard-api.zip
echo "✓ Cleanup complete"
echo ""

echo "✓ Phase 3 complete - Lambda deployed"
echo ""

# ============================================================
# Summary
# ============================================================
echo "========================================="
echo "Deployment Complete!"
echo "========================================="
echo ""
echo "Next steps:"
echo "  1. Monitor Lambda logs for errors:"
echo "     AWS_PROFILE=spore-host-infra aws logs tail /aws/lambda/spawn-dashboard-api --follow"
echo ""
echo "  2. Test the dashboard at:"
echo "     https://spore.host"
echo ""
echo "  3. Verify per-user isolation:"
echo "     - Login with Cognito (web)"
echo "     - Check that dashboard shows CLI-created resources"
echo "     - Verify sweeps, instances, and autoscale groups are visible"
echo ""
echo "Rollback plan (if needed):"
echo "  AWS_PROFILE=spore-host-infra aws lambda update-function-code \\"
echo "    --function-name spawn-dashboard-api \\"
echo "    --s3-bucket spawn-lambda-artifacts \\"
echo "    --s3-key dashboard-api-v<previous-version>.zip"
echo ""
