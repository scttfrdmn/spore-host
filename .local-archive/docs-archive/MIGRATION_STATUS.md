# AWS Account Migration Status - Mycelium Project

**Last Updated:** 2025-12-30 18:05 PST
**Overall Status:** 🟡 In Progress - Awaiting user actions

---

## ✅ Completed Phases

### Phase 1: AWS Organization Setup ✅
- **Management Account** (752123829273)
  - IAM user `scott-admin` created
  - AdministratorAccess granted
  - Can assume roles in sub-accounts

- **Infrastructure Account** (966362334030)
  - Email: scttfrdmn+mycelium-infra@gmail.com
  - Purpose: DNS, website, Lambda, Cognito
  - CLI Profile: `mycelium-infra`

- **Development Account** (435415984226)
  - Email: scttfrdmn+mycelium-dev@gmail.com
  - Purpose: Test EC2 instances
  - CLI Profile: `mycelium-dev`

### Phase 2: DNS Migration ✅
- **New Hosted Zone** created in infrastructure account
  - Zone ID: `Z0341053304H0DQXF6U4X`
  - All DNS records migrated (9 records)

- **⚠️ ACTION REQUIRED:** Update nameservers at registrar
  - See: `DNS_MIGRATION_STATUS.md` for details
  - New nameservers:
    ```
    ns-827.awsdns-39.net
    ns-1175.awsdns-18.org
    ns-425.awsdns-53.com
    ns-1874.awsdns-42.co.uk
    ```

---

## 🟡 In Progress

### Phase 3: S3 Bucket Migration 🟡
**Status:** Partially complete

**Website Bucket:**
- ❌ `spore-host-website` - Waiting for deletion propagation (5-10 minutes)
- Files backed up to: `/tmp/s3-migration/spore-host-website`
- **Action:** Retry creation in ~5 minutes:
  ```bash
  # Wait for propagation, then:
  AWS_PROFILE=mycelium-infra aws s3 mb s3://spore-host-website --region us-east-1
  AWS_PROFILE=mycelium-infra aws s3 sync /tmp/s3-migration/spore-host-website s3://spore-host-website
  AWS_PROFILE=mycelium-infra aws s3 website s3://spore-host-website --index-document index.html
  ```

**Binary Buckets:**
- ⏳ 8 buckets pending migration: `spawn-binaries-*` (us-east-1, us-west-2, eu-west-1, etc.)
- **Action:** Run migration script:
  ```bash
  bash /tmp/migrate-binary-buckets.sh
  ```
- Script will:
  - Download from management account
  - Delete from management account
  - Create in infrastructure account
  - Upload to infrastructure account
  - Set public-read policy

---

## ⏳ Pending Phases

### Phase 4: CloudFront Migration
**Current State:** Distribution E50GL663TTL0I in management account

**Tasks:**
1. Export current distribution config
2. Create new distribution in infrastructure account
3. Point to new S3 origin
4. Update DNS A record (already points to CloudFront)
5. Wait for deployment (~15 minutes)
6. Delete old distribution

**Estimated Time:** 1 hour

### Phase 5: Lambda Migration
**Current State:** `spawn-dns-updater` in management account

**Tasks:**
1. Export Lambda function code and config
2. Create IAM role in infrastructure account
3. Deploy Lambda in infrastructure account
4. Update environment variables (new hosted zone ID)
5. Test API Gateway integration
6. Delete old Lambda

**Estimated Time:** 1 hour

### Phase 6: Cognito Setup
**Current State:** Not yet created

**Tasks:**
1. Run `scripts/setup-dashboard-cognito.sh` using `mycelium-infra` profile
2. Update `web/js/auth.js` with Identity Pool ID
3. Test authentication flow

**Estimated Time:** 30 minutes (after OAuth apps registered)

### Phase 7: Update CLI and Scripts
**Tasks:**
1. Update `spawn/CLAUDE.md` - replace profile references
2. Update all shell scripts in `scripts/` directory
3. Update deployment scripts
4. Update README and documentation

**Commands:**
```bash
# Replace default → mycelium-infra in scripts
cd /Users/scttfrdmn/src/mycelium
grep -r "AWS_PROFILE=default" scripts/ --files-with-matches | \
  xargs sed -i '' 's/AWS_PROFILE=default/AWS_PROFILE=mycelium-infra/g'

# Replace aws → mycelium-dev in scripts
grep -r "AWS_PROFILE=aws" scripts/ --files-with-matches | \
  xargs sed -i '' 's/AWS_PROFILE=aws/AWS_PROFILE=mycelium-dev/g'
```

**Estimated Time:** 30 minutes

### Phase 8: Testing and Documentation
**Tasks:**
1. Test website loads: https://spore.host
2. Test DNS lookups for subdomains
3. Test spawn CLI in dev account
4. Test Lambda DNS updater
5. Test dashboard authentication
6. Update documentation

**Estimated Time:** 2 hours

---

## Current Status Summary

| Phase | Status | Blocker | ETA |
|-------|--------|---------|-----|
| 1. Organization Setup | ✅ Complete | None | Done |
| 2. DNS Migration | ✅ Complete | Nameserver update needed | User action |
| 3. S3 Buckets | 🟡 In Progress | Deletion propagation | 5-10 min |
| 4. CloudFront | ⏳ Pending | S3 migration | 1 hour |
| 5. Lambda | ⏳ Pending | CloudFront migration | 1 hour |
| 6. Cognito | ⏳ Pending | OAuth apps | 30 min |
| 7. Scripts/CLI | ⏳ Pending | Resources migrated | 30 min |
| 8. Testing | ⏳ Pending | All phases complete | 2 hours |

**Total Remaining:** ~5-6 hours active work + DNS propagation time (24-48 hours)

---

## Immediate Next Steps

1. **Wait 5-10 minutes** for S3 bucket deletion to propagate

2. **Complete S3 migration:**
   ```bash
   # Website bucket
   AWS_PROFILE=mycelium-infra aws s3 mb s3://spore-host-website --region us-east-1
   AWS_PROFILE=mycelium-infra aws s3 sync /tmp/s3-migration/spore-host-website s3://spore-host-website

   # Binary buckets
   bash /tmp/migrate-binary-buckets.sh
   ```

3. **Update nameservers** at domain registrar for spore.host
   - See `DNS_MIGRATION_STATUS.md` for new nameservers
   - DNS propagation takes 24-48 hours

4. **While waiting for DNS:**
   - Continue with CloudFront migration (Phase 4)
   - Continue with Lambda migration (Phase 5)
   - These don't depend on DNS propagation

5. **After resources migrated:**
   - Update scripts to use new profiles (Phase 7)
   - Run comprehensive tests (Phase 8)

---

## Files Created

- `/Users/scttfrdmn/src/mycelium/AWS_ACCOUNT_STRUCTURE.md` - Complete account structure and original migration plan
- `/Users/scttfrdmn/src/mycelium/DNS_MIGRATION_STATUS.md` - DNS-specific migration details and nameserver info
- `/Users/scttfrdmn/src/mycelium/MIGRATION_STATUS.md` - This file - overall migration progress
- `/tmp/migrate-binary-buckets.sh` - Script to migrate binary S3 buckets
- `/tmp/s3-migration/` - Local backup of S3 bucket contents

## AWS CLI Profiles Configured

```bash
# Management account (IAM user: scott-admin)
AWS_PROFILE=management aws sts get-caller-identity

# Infrastructure account (for DNS, website, Lambda)
AWS_PROFILE=mycelium-infra aws sts get-caller-identity

# Development account (for test EC2 instances)
AWS_PROFILE=mycelium-dev aws sts get-caller-identity
```

## Rollback Notes

- **DNS:** Old hosted zone still active until nameservers updated
- **S3:** Files backed up in `/tmp/s3-migration/`
- **CloudFront:** Old distribution still serving traffic
- **Lambda:** Old function still active

**No service disruption** until you update nameservers and CloudFront origin.

---

## Questions or Issues?

- AWS Organizations docs: https://docs.aws.amazon.com/organizations/
- S3 bucket migration: https://docs.aws.amazon.com/AmazonS3/latest/userguide/
- Route53 migration: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/

**Ready to continue?** Run the S3 bucket migration commands above after waiting ~5 more minutes.
