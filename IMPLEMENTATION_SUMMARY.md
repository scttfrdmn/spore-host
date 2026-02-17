# Issue #137 Implementation Summary

**Status:** ✅ Complete
**Date:** 2026-02-17

---

## Changes Implemented

### 1. Database Schema Extensions

#### spawn-user-accounts Table
**New Fields:**
- `cli_iam_arn` (string) - Canonical CLI IAM ARN for resource filtering
- `identity_type` (string) - "cli" or "web" to distinguish authentication method
- `linked_at` (string) - RFC3339 timestamp when identity mapping was created

**New Global Secondary Indexes:**
- `email-index` - Query users by email (HASH: email)
- `cli_iam_arn-index` - Reverse lookup from CLI ARN to user record (HASH: cli_iam_arn)

#### spawn-autoscale-groups-production Table
**New Fields:**
- `user_id` (string) - CLI IAM ARN of group owner

**New Global Secondary Indexes:**
- `user_id-index` - Query autoscale groups by user (HASH: user_id)

### 2. Backend Code Changes

#### auth.go (spawn/lambda/dashboard-api/auth.go)
**Modified Functions:**
- `getUserFromRequest()` - Now returns `cliIamArn` instead of `accountID`
  - Detects identity type (CLI vs web)
  - For CLI users: uses IAM user ARN directly
  - For web users: extracts email from Cognito claims and looks up CLI IAM ARN

**New Functions:**
- `extractEmailFromRequest()` - Extracts email from API Gateway request (Cognito claims)
- `lookupCliIamArnByEmail()` - Queries `spawn-user-accounts` by email to find linked CLI IAM ARN
- `createUserAccountWithIdentity()` - Creates/updates user account with identity mapping fields

**Updated Signature:**
```go
// OLD:
func getUserFromRequest(ctx, cfg, request) (userID, accountID, accountBase36, error)

// NEW:
func getUserFromRequest(ctx, cfg, request) (userID, cliIamArn, accountBase36, error)
```

#### sweeps.go (spawn/lambda/dashboard-api/sweeps.go)
**Changed:**
- Removed hardcoded dual-account filter (lines 36-40)
- Replaced with per-user filter: `user_id = :user_id`
- Updated all handler signatures to accept `cliIamArn` instead of `accountID`
- Updated access control checks to verify `sweep.UserID == cliIamArn`

**Before:**
```go
FilterExpression: aws.String("aws_account_id IN (:infra_account, :dev_account)")
```

**After:**
```go
FilterExpression: aws.String("user_id = :user_id")
ExpressionAttributeValues: map[string]types.AttributeValue{
    ":user_id": &types.AttributeValueMemberS{Value: cliIamArn},
}
```

#### instances.go (spawn/lambda/dashboard-api/instances.go)
**Changed:**
- Updated `listInstances()` to filter by `tag:spawn:iam-user` instead of `tag:spawn:account-base36`
- Updated `getInstance()` to verify instance ownership using `spawn:iam-user` tag
- Updated function signatures to accept `cliIamArn` instead of `accountBase36` + `userID`

**Before:**
```go
Filter: tag:spawn:account-base36 = accountBase36
```

**After:**
```go
Filter: tag:spawn:iam-user = cliIamArn
```

#### autoscale.go (spawn/lambda/dashboard-api/autoscale.go)
**Major Refactor:**
- Replaced EC2 tag-based group discovery with DynamoDB `user_id-index` query
- Added user ownership verification in `handleGetAutoscaleGroup()`
- Updated all capacity calculation functions to filter by `spawn:iam-user` tag
- Updated cost calculation to filter by `spawn:iam-user` instead of `spawn:account-base36`

**New Type:**
```go
type AutoScaleGroupWithUserID struct {
    AutoScaleGroup
    UserID string `dynamodbav:"user_id"`
}
```

**Before (EC2 tag discovery):**
```go
func handleListAutoscaleGroups(ctx, cfg, accountBase36) {
    groupIDs := getUserAutoscaleGroupIDsViaEC2Tags(accountBase36)
    groups := getGroupsFromDynamoDB(groupIDs)
}
```

**After (DynamoDB query):**
```go
func handleListAutoscaleGroups(ctx, cfg, cliIamArn) {
    queryInput := &dynamodb.QueryInput{
        IndexName: "user_id-index",
        KeyConditionExpression: "user_id = :user_id",
    }
    // Direct query, no EC2 discovery needed
}
```

#### cross_account.go (spawn/lambda/dashboard-api/cross_account.go)
**Fixed:**
- Updated deprecated cross-account role ARN
- OLD: `arn:aws:iam::942542972736:role/SpawnDashboardCrossAccountRole`
- NEW: `arn:aws:iam::435415984226:role/SpawnDashboardCrossAccountReadRole`

#### main.go (spawn/lambda/dashboard-api/main.go)
**Updated:**
- Changed all handler calls to pass `cliIamArn` instead of `accountID`
- Updated handler signatures throughout the routing switch

#### models.go (spawn/lambda/dashboard-api/models.go)
**Extended:**
- Added new fields to `UserAccountRecord` struct:
  - `CliIamArn`
  - `IdentityType`
  - `LinkedAt`

### 3. Infrastructure Scripts

**Created:**
- `scripts/add-user-accounts-gsi.sh` - Adds GSIs to spawn-user-accounts table
- `scripts/add-autoscale-user-id-gsi.sh` - Adds user_id GSI to spawn-autoscale-groups-production
- `scripts/backfill-autoscale-user-ids.py` - Backfills user_id field in autoscale groups
- `scripts/create-user-mapping.sh` - Creates initial user mapping for scott-admin
- `scripts/setup-cross-account-role.sh` - Sets up cross-account role in dev account
- `scripts/deploy.sh` - Builds and deploys Lambda function

---

## Architecture Changes

### Identity Flow

**Before (Broken):**
```
Web User → Cognito → STS (infra account ARN)
                      ↓
                      ❌ No match with CLI resources (dev account)
```

**After (Fixed):**
```
Web User → Cognito → STS (infra account ARN)
                      ↓
                   Extract email from claims
                      ↓
            Query spawn-user-accounts by email
                      ↓
              Find CLI IAM ARN mapping
                      ↓
       Use CLI IAM ARN for all resource filtering
```

### Resource Filtering

**All endpoints now use CLI IAM ARN for per-user isolation:**
- Sweeps: `user_id = cliIamArn` (DynamoDB filter)
- Instances: `tag:spawn:iam-user = cliIamArn` (EC2 filter)
- Autoscale Groups: `user_id = cliIamArn` (DynamoDB GSI query)
- Cost Summary: `tag:spawn:iam-user = cliIamArn` (EC2 filter)

### Security Improvements

1. **Per-user isolation** - Users can only see their own resources
2. **No hardcoded account IDs** - Removed dual-account hack
3. **Explicit access control** - All handlers verify ownership before returning data
4. **Helpful error messages** - "Identity not linked (email: ...)" for unmapped web users

---

## Deployment Instructions

### Phase 1: Infrastructure Setup (Run First)

```bash
# 1. Add GSIs to spawn-user-accounts
cd /Users/scttfrdmn/src/mycelium/spawn/lambda/dashboard-api
./scripts/add-user-accounts-gsi.sh mycelium-infra

# 2. Add user_id GSI to spawn-autoscale-groups-production
./scripts/add-autoscale-user-id-gsi.sh mycelium-infra

# 3. Setup cross-account role (if not exists)
./scripts/setup-cross-account-role.sh mycelium-dev

# 4. Wait for GSIs to become ACTIVE (5-10 minutes)
watch 'AWS_PROFILE=mycelium-infra aws dynamodb describe-table \
  --table-name spawn-user-accounts \
  --query "Table.GlobalSecondaryIndexes[*].[IndexName,IndexStatus]" \
  --output table'
```

### Phase 2: Data Backfill (Run Second)

```bash
# 1. Backfill autoscale groups with user_id
python3 scripts/backfill-autoscale-user-ids.py mycelium-infra

# 2. Create initial user mapping for scott-admin
./scripts/create-user-mapping.sh scttfrdmn@gmail.com mycelium-infra

# 3. Verify backfill
AWS_PROFILE=mycelium-infra aws dynamodb scan \
  --table-name spawn-autoscale-groups-production \
  --filter-expression "attribute_not_exists(user_id)" \
  --select COUNT
# Should return: Count: 0
```

### Phase 3: Code Deploy (Run Last)

```bash
# Build and deploy Lambda
./scripts/deploy.sh mycelium-infra

# Monitor logs for errors
AWS_PROFILE=mycelium-infra aws logs tail /aws/lambda/spawn-dashboard-api --follow
```

---

## Testing

### 1. CLI User Flow (Existing)
```bash
# Launch resources from CLI
AWS_PROFILE=mycelium-dev spawn instance launch test-instance
AWS_PROFILE=mycelium-dev spawn sweep launch test-sweep

# Access via API (CLI credentials)
curl -X GET https://api.spore.host/api/sweeps \
  -H "X-AWS-Credentials: <base64-cli-credentials>"

# ✅ Expected: See only scott-admin's resources
```

### 2. Web User Flow (New)
```bash
# 1. Login to https://spore.host with Google (scttfrdmn@gmail.com)
# 2. Dashboard should load successfully
# 3. Verify all tabs show CLI-created resources:
#    - Sweeps tab: Shows CLI-created sweeps
#    - Instances tab: Shows CLI-created instances
#    - Autoscale tab: Shows CLI-created groups
#    - Cost summary: Shows only user's costs

# ✅ Expected: Web user sees same resources as CLI user
```

### 3. Access Control (Security)
```bash
# Try accessing sweep that doesn't belong to user
curl -X GET https://api.spore.host/api/sweeps/wrong-sweep-id \
  -H "X-AWS-Credentials: <base64-cli-credentials>"

# ✅ Expected: 403 Access denied
```

### 4. Unmapped Web User
```bash
# Login with email NOT in spawn-user-accounts
# Try accessing any endpoint

# ✅ Expected: 401 Authentication failed: identity not linked (email: new@example.com)
```

---

## Rollback Plan

If issues occur:

```bash
# Revert Lambda to previous version
AWS_PROFILE=mycelium-infra aws lambda update-function-code \
  --function-name spawn-dashboard-api \
  --s3-bucket spawn-lambda-artifacts \
  --s3-key dashboard-api-backup.zip
```

**Note:** GSIs and data backfills are non-destructive and can remain.

---

## Success Criteria

- [x] Code compiles without errors
- [x] All handlers updated to use `cliIamArn`
- [x] Hardcoded account filtering removed
- [x] Per-user isolation enforced for all resource types
- [x] Email-based identity mapping implemented
- [x] Cross-account role ARN fixed
- [x] Infrastructure scripts created
- [x] Deployment scripts created
- [x] Backfill scripts created

**Next Steps:**
1. Run Phase 1 (Infrastructure) - wait for GSIs
2. Run Phase 2 (Data Backfill) - verify no missing user_ids
3. Run Phase 3 (Deploy) - monitor logs
4. Test web login and verify per-user isolation

---

## Files Modified

**Backend Code:**
- `spawn/lambda/dashboard-api/auth.go` - Identity mapping logic
- `spawn/lambda/dashboard-api/sweeps.go` - Per-user filtering
- `spawn/lambda/dashboard-api/instances.go` - Per-user filtering
- `spawn/lambda/dashboard-api/autoscale.go` - DynamoDB query + per-user filtering
- `spawn/lambda/dashboard-api/cross_account.go` - Fixed role ARN
- `spawn/lambda/dashboard-api/main.go` - Updated routing
- `spawn/lambda/dashboard-api/models.go` - Extended UserAccountRecord

**Scripts Created:**
- `spawn/lambda/dashboard-api/scripts/add-user-accounts-gsi.sh`
- `spawn/lambda/dashboard-api/scripts/add-autoscale-user-id-gsi.sh`
- `spawn/lambda/dashboard-api/scripts/backfill-autoscale-user-ids.py`
- `spawn/lambda/dashboard-api/scripts/create-user-mapping.sh`
- `spawn/lambda/dashboard-api/scripts/setup-cross-account-role.sh`
- `spawn/lambda/dashboard-api/scripts/deploy.sh`
