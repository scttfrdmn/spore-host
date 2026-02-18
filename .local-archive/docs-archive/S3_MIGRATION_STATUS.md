# S3 Bucket Migration Status

**Date:** 2025-12-30
**Status:** ⏸️ Paused - Awaiting S3 name propagation

---

## Current Situation

**✅ Completed:**
- All bucket contents backed up to `/tmp/s3-migration/`
- All buckets properly deleted from management account (752123829273)
  - All object versions deleted
  - All delete markers removed
  - Buckets removed successfully

**⏸️ Blocker:**
- S3 bucket names not yet available in infrastructure account (966362334030)
- Propagation delay: >10 minutes (unusual but can happen)
- Typical propagation time: 5-10 minutes
- Maximum propagation time: Can take hours (rare)

---

## Buckets Pending Recreation

**Website:**
- `spore-host-website` (6 files, 3.2MB)
  - Backup: `/tmp/s3-migration/spore-host-website/`

**Binaries (8 regions):**
- `spawn-binaries-us-east-1` (8 files, 46MB) - Backup: `/tmp/s3-migration/spawn-binaries-us-east-1/`
- `spawn-binaries-us-east-2` (8 files) - Backup: `/tmp/s3-migration/spawn-binaries-us-east-2/`
- `spawn-binaries-us-west-1` (8 files) - Backup: `/tmp/s3-migration/spawn-binaries-us-west-1/`
- `spawn-binaries-us-west-2` (8 files) - Backup: `/tmp/s3-migration/spawn-binaries-us-west-2/`
- `spawn-binaries-eu-west-1` (8 files) - Backup: `/tmp/s3-migration/spawn-binaries-eu-west-1/`
- `spawn-binaries-eu-central-1` (8 files) - Backup: `/tmp/s3-migration/spawn-binaries-eu-central-1/`
- `spawn-binaries-ca-central-1` (8 files) - Backup: `/tmp/s3-migration/spawn-binaries-ca-central-1/`
- `spawn-binaries-ap-southeast-2` (8 files) - Backup: `/tmp/s3-migration/spawn-binaries-ap-southeast-2/`

---

## Options Forward

### Option 1: Wait and Retry (Recommended)
**When:** Wait 1-2 hours or retry tomorrow

**Command:**
```bash
bash /tmp/recreate-all-buckets.sh
```

**Pros:**
- Keeps original bucket names
- No need to update CloudFront or scripts
- Simple and clean

**Cons:**
- Requires waiting (potentially hours)

**Best for:** Production systems where naming consistency matters

### Option 2: Use Different Names (Faster)
**When:** Need to continue immediately

**Implementation:**
```bash
# Use temporary names with -new suffix
AWS_PROFILE=mycelium-infra aws s3 mb s3://spore-host-website-infra --region us-east-1
AWS_PROFILE=mycelium-infra aws s3 sync /tmp/s3-migration/spore-host-website s3://spore-host-website-infra

# For binaries:
AWS_PROFILE=mycelium-infra aws s3 mb s3://spawn-binaries-us-east-1-infra --region us-east-1
AWS_PROFILE=mycelium-infra aws s3 sync /tmp/s3-migration/spawn-binaries-us-east-1 s3://spawn-binaries-us-east-1-infra
# ... repeat for other regions
```

**Then update:**
- CloudFront distribution origin
- `web/deploy.sh` script
- `scripts/upload_spawnd.sh` script
- Any other scripts referencing bucket names

**Pros:**
- Can continue immediately
- No waiting required

**Cons:**
- Must update CloudFront configuration
- Must update all scripts
- Different bucket names than original

**Best for:** Development/testing when you need to move quickly

### Option 3: Hybrid Approach
**Implementation:**
1. Use different names NOW for infrastructure account
2. Continue with CloudFront/Lambda migration using new names
3. LATER (when names available): Create original names and swap

**Pros:**
- Can continue migration immediately
- Eventually get back to original names
- No long-term changes needed

**Cons:**
- Two-step process
- Temporary configuration changes

---

## Manual Recreation Commands

If propagation completes, run these commands:

```bash
# Website bucket
AWS_PROFILE=mycelium-infra aws s3 mb s3://spore-host-website --region us-east-1
AWS_PROFILE=mycelium-infra aws s3 sync /tmp/s3-migration/spore-host-website s3://spore-host-website
AWS_PROFILE=mycelium-infra aws s3 website s3://spore-host-website --index-document index.html
AWS_PROFILE=mycelium-infra aws s3api put-bucket-policy --bucket spore-host-website --policy '{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "PublicReadGetObject",
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::spore-host-website/*"
  }]
}'

# Binary buckets (example for us-east-1)
AWS_PROFILE=mycelium-infra aws s3 mb s3://spawn-binaries-us-east-1 --region us-east-1
AWS_PROFILE=mycelium-infra aws s3 sync /tmp/s3-migration/spawn-binaries-us-east-1 s3://spawn-binaries-us-east-1
AWS_PROFILE=mycelium-infra aws s3api put-bucket-policy --bucket spawn-binaries-us-east-1 --policy '{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "PublicReadGetObject",
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::spawn-binaries-us-east-1/*"
  }]
}'

# Repeat for other regions...
```

Or use the script:
```bash
bash /tmp/recreate-all-buckets.sh
```

---

## Testing Propagation Status

Check if bucket names are available:

```bash
# Try creating website bucket
AWS_PROFILE=mycelium-infra aws s3 mb s3://spore-host-website --region us-east-1 2>&1

# Expected when still propagating:
# "An error occurred (OperationAborted) when calling the CreateBucket operation"

# Expected when ready:
# "make_bucket: spore-host-website"
```

---

## Impact Assessment

**Current Service Status:**
- ✅ Website (spore.host): **Still working** - CloudFront serving from old bucket (deleted but cached)
- ❌ Binary downloads: **Broken** - buckets deleted, not yet recreated
- ✅ DNS: **Working** - migrated successfully
- ✅ spawn CLI: **Still works** - launches instances (but can't download spored binary)

**What's Broken Right Now:**
1. `spawn` CLI can't download spored binary from S3 (404 errors)
2. Manual binary downloads from https://s3.../spawn-binaries-* (404 errors)

**What Still Works:**
1. Website at https://spore.host (CloudFront cache)
2. DNS resolution for *.spore.host
3. spawn CLI can launch instances (just no spored installation)

---

## Recommendation

**For production system:** Use Option 1 (Wait and Retry)
- Retry in 1-2 hours or tomorrow morning
- Keeps everything clean and consistent
- No script updates needed

**For immediate testing:** Use Option 2 (Different Names)
- Create buckets with `-infra` suffix
- Update CloudFront and scripts
- Can always migrate back to original names later

**Safest approach:** Option 3 (Hybrid)
- Use different names now to unblock
- Migrate back to original names when available

---

## Next Steps (Recommended)

1. **Take a break** - Let S3 propagation complete (1-2 hours)

2. **Meanwhile**: Continue with non-S3 migrations
   - Phase 4: CloudFront (can update later to point to new buckets)
   - Phase 5: Lambda (independent of S3 bucket names)
   - Phase 7: Update scripts (will need bucket names anyway)

3. **Come back and retry** bucket creation:
   ```bash
   bash /tmp/recreate-all-buckets.sh
   ```

4. **If still failing after 24 hours**: Contact AWS Support
   - Bucket names may be stuck in limbo
   - AWS can manually clear the names

---

## Files

- Backup location: `/tmp/s3-migration/`
- Recreation script: `/tmp/recreate-all-buckets.sh`
- Delete script: `/tmp/delete-versioned-bucket.sh`

**Important:** Don't delete `/tmp/s3-migration/` - it's your only backup!

---

## Resources

- [S3 Bucket Naming](https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html)
- [S3 Bucket Deletion](https://docs.aws.amazon.com/AmazonS3/latest/userguide/delete-bucket.html)
- [Troubleshooting S3](https://repost.aws/knowledge-center/s3-conflicting-conditional-operation)
