# AWS Account Structure - Mycelium Project

**Date:** 2025-12-30
**Status:** Organization configured, pending resource migration

---

## Organization Structure

```
AWS Organization (o-v33p2ygcff)
│
├── Management Account (752123829273)
│   ├── Email: scttfrdmn@gmail.com
│   ├── Purpose: Organization administration only
│   ├── IAM User: scott-admin (for CLI access)
│   └── Profile: management
│
├── Mycelium Infrastructure (966362334030)
│   ├── Email: scttfrdmn+spore-host-infra@gmail.com
│   ├── Purpose: Production infrastructure
│   ├── Resources: DNS (Route53), Website (S3/CloudFront), Lambda, Cognito
│   └── Profile: spore-host-infra
│
├── Mycelium Development (435415984226)
│   ├── Email: scttfrdmn+spore-host-dev@gmail.com
│   ├── Purpose: Development and testing
│   ├── Resources: Test EC2 instances, development workloads
│   └── Profile: spore-host-dev
│
└── SnoozeBot Integration Testing (570220934149)
    ├── Email: scttfrdmn+snoozebot@gmail.com
    └── Purpose: SnoozeBot project
```

---

## AWS CLI Profiles

### New Profiles (Use These)

**`management`** - Management Account (752123829273)
- IAM user: `scott-admin`
- Use for: Organization-wide administration
- Test: `AWS_PROFILE=management aws sts get-caller-identity`

**`spore-host-infra`** - Infrastructure Account (966362334030)
- Role: Assumes `OrganizationAccountAccessRole`
- Use for: DNS, website, Lambda, Cognito operations
- Test: `AWS_PROFILE=spore-host-infra aws sts get-caller-identity`

**`spore-host-dev`** - Development Account (435415984226)
- Role: Assumes `OrganizationAccountAccessRole`
- Use for: Testing EC2 instances, development
- Test: `AWS_PROFILE=spore-host-dev aws sts get-caller-identity`

### Legacy Profiles (Deprecated)

**`default`** - ⚠️ DEPRECATED
- Uses root credentials (insecure)
- Replace with: `management` profile

**`aws`** - ⚠️ DEPRECATED
- Account 942542972736 (no longer used)
- Replace with: `spore-host-dev` profile

---

## Account Usage Guidelines

### Management Account (752123829273)
**DO:**
- Manage AWS Organizations
- Create/delete sub-accounts
- Set up billing and cost allocation
- Configure organization-wide policies

**DON'T:**
- Deploy application workloads
- Store data
- Run EC2 instances
- Host websites

**Current Resources to Migrate:**
- Route53 hosted zone: `spore.host`
- S3 bucket: `spore-host-website`
- CloudFront distribution: E50GL663TTL0I
- Lambda function: `spawn-dns-updater`
- S3 buckets: `spawn-binaries-*` (multiple regions)
- Cognito Identity Pool: (to be created)
- IAM roles: (various)

### Infrastructure Account (966362334030)
**Purpose:** Production infrastructure for Mycelium project

**Resources:**
- Route53: DNS management for `spore.host`
- S3: Website hosting, binary distribution
- CloudFront: CDN for website
- Lambda: DNS updater, future API functions
- Cognito: User authentication for dashboard
- IAM roles: Service roles for Lambda, etc.

**DO:**
- Deploy production infrastructure
- Manage DNS records
- Host public website
- Store release binaries
- Run serverless functions

**DON'T:**
- Run test EC2 instances
- Store temporary data
- Deploy experimental features

### Development Account (435415984226)
**Purpose:** Testing and development workloads

**Resources:**
- EC2 instances: Test spawn-managed instances
- VPCs: Testing network configurations
- Test data: Temporary development data

**DO:**
- Launch test EC2 instances with spawn CLI
- Experiment with new features
- Run integration tests
- Store temporary development data

**DON'T:**
- Deploy production workloads
- Store important data long-term
- Host public services

---

## Security Improvements

### ✅ Completed

1. **No More Root Credentials**
   - Created IAM user `scott-admin` in management account
   - Root credentials should only be used for account recovery

2. **Cross-Account Access via IAM Roles**
   - `OrganizationAccountAccessRole` automatically created in sub-accounts
   - IAM user assumes roles (not sharing credentials)

3. **Account Isolation**
   - Separate billing and resource boundaries
   - Blast radius limited to single account

4. **Principle of Least Privilege**
   - Management account: Organization admin only
   - Infrastructure account: Production workloads
   - Development account: Testing and experiments

### ⏳ Pending

1. **Enable MFA on IAM User**
   ```bash
   # Add MFA to scott-admin user
   AWS_PROFILE=management aws iam enable-mfa-device ...
   ```

2. **Rotate Root Credentials**
   - Change root password for all accounts
   - Store in 1Password/LastPass

3. **Set Up CloudTrail**
   - Enable organization-wide CloudTrail
   - Centralized logging in management account

4. **Configure SCPs (Service Control Policies)**
   - Prevent accidental resource deletion
   - Enforce region restrictions
   - Require MFA for sensitive operations

---

## Migration Plan

### Phase 1: Prepare Infrastructure Account

**1.1 Verify Access**
```bash
# Test profile
AWS_PROFILE=spore-host-infra aws sts get-caller-identity

# List S3 buckets (should be empty)
AWS_PROFILE=spore-host-infra aws s3 ls
```

**1.2 Enable Required Services**
```bash
# Enable Route53
AWS_PROFILE=spore-host-infra aws route53 list-hosted-zones

# Enable Lambda
AWS_PROFILE=spore-host-infra aws lambda list-functions
```

### Phase 2: Migrate DNS (Route53)

**Current State:** Hosted zone in management account (752123829273)

**2.1 Export DNS Records**
```bash
# Get hosted zone ID
AWS_PROFILE=management aws route53 list-hosted-zones \
  --query 'HostedZones[?Name==`spore.host.`].Id' \
  --output text

# Export all records
AWS_PROFILE=management aws route53 list-resource-record-sets \
  --hosted-zone-id <ZONE_ID> > /tmp/spore-host-records.json
```

**2.2 Create New Hosted Zone in Infrastructure Account**
```bash
# Create hosted zone
AWS_PROFILE=spore-host-infra aws route53 create-hosted-zone \
  --name spore.host \
  --caller-reference "migration-$(date +%s)"

# Import records (manually or via script)
# Note: Update NS records with registrar
```

**2.3 Update Registrar**
- Get new nameservers from infrastructure account
- Update nameservers at domain registrar
- Wait for DNS propagation (24-48 hours)

**2.4 Delete Old Hosted Zone**
```bash
# After DNS propagation complete
AWS_PROFILE=management aws route53 delete-hosted-zone \
  --id <OLD_ZONE_ID>
```

### Phase 3: Migrate S3 Buckets

**Current State:** Buckets in management account

**Buckets to Migrate:**
- `spore-host-website` - Website files
- `spawn-binaries-us-east-1` - Release binaries (US East 1)
- `spawn-binaries-us-west-2` - Release binaries (US West 2)
- `spawn-binaries-eu-west-1` - Release binaries (EU West 1)
- (others)

**3.1 Create Buckets in Infrastructure Account**
```bash
# Create website bucket
AWS_PROFILE=spore-host-infra aws s3 mb s3://spore-host-website

# Create binary buckets (one per region)
AWS_PROFILE=spore-host-infra aws s3 mb s3://spawn-binaries-us-east-1 --region us-east-1
AWS_PROFILE=spore-host-infra aws s3 mb s3://spawn-binaries-us-west-2 --region us-west-2
# ... etc
```

**3.2 Copy Data**
```bash
# Copy website files
aws s3 sync \
  s3://spore-host-website \
  s3://spore-host-website \
  --source-profile management \
  --profile spore-host-infra

# Copy binaries
for region in us-east-1 us-west-2 eu-west-1; do
  aws s3 sync \
    s3://spawn-binaries-$region \
    s3://spawn-binaries-$region \
    --source-profile management \
    --profile spore-host-infra \
    --region $region
done
```

**3.3 Update S3 Bucket Policies**
```bash
# Make website bucket public (static website)
AWS_PROFILE=spore-host-infra aws s3api put-bucket-website \
  --bucket spore-host-website \
  --website-configuration file:///tmp/website-config.json

# Make binary buckets public (for downloads)
AWS_PROFILE=spore-host-infra aws s3api put-bucket-policy \
  --bucket spawn-binaries-us-east-1 \
  --policy file:///tmp/public-read-policy.json
```

**3.4 Delete Old Buckets**
```bash
# After verifying data copied
AWS_PROFILE=management aws s3 rb s3://spore-host-website --force
AWS_PROFILE=management aws s3 rb s3://spawn-binaries-us-east-1 --force
# ... etc
```

### Phase 4: Migrate CloudFront Distribution

**Current State:** Distribution E50GL663TTL0I in management account

**4.1 Create New Distribution in Infrastructure Account**
```bash
# Get current distribution config
AWS_PROFILE=management aws cloudfront get-distribution-config \
  --id E50GL663TTL0I > /tmp/cf-config.json

# Create new distribution in infrastructure account
AWS_PROFILE=spore-host-infra aws cloudfront create-distribution \
  --distribution-config file:///tmp/cf-config-new.json
```

**4.2 Update DNS Records**
```bash
# Update Route53 to point to new CloudFront distribution
AWS_PROFILE=spore-host-infra aws route53 change-resource-record-sets \
  --hosted-zone-id <NEW_ZONE_ID> \
  --change-batch file:///tmp/cf-dns-update.json
```

**4.3 Delete Old Distribution**
```bash
# Disable distribution
AWS_PROFILE=management aws cloudfront update-distribution \
  --id E50GL663TTL0I --if-match <ETAG> \
  --distribution-config file:///tmp/cf-disable.json

# Wait for deployment
# Delete distribution
AWS_PROFILE=management aws cloudfront delete-distribution \
  --id E50GL663TTL0I --if-match <ETAG>
```

### Phase 5: Migrate Lambda Functions

**Current State:** `spawn-dns-updater` in management account

**5.1 Export Lambda Function**
```bash
# Get function code
AWS_PROFILE=management aws lambda get-function \
  --function-name spawn-dns-updater \
  --query 'Code.Location' \
  --output text | xargs curl -o /tmp/lambda.zip

# Get function configuration
AWS_PROFILE=management aws lambda get-function-configuration \
  --function-name spawn-dns-updater > /tmp/lambda-config.json
```

**5.2 Create IAM Role in Infrastructure Account**
```bash
# Create Lambda execution role
AWS_PROFILE=spore-host-infra aws iam create-role \
  --role-name SpawnDNSUpdaterRole \
  --assume-role-policy-document file:///tmp/lambda-trust-policy.json

# Attach policies
AWS_PROFILE=spore-host-infra aws iam attach-role-policy \
  --role-name SpawnDNSUpdaterRole \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

# Add Route53 permissions
AWS_PROFILE=spore-host-infra aws iam put-role-policy \
  --role-name SpawnDNSUpdaterRole \
  --policy-name Route53Access \
  --policy-document file:///tmp/route53-policy.json
```

**5.3 Deploy Lambda in Infrastructure Account**
```bash
# Create function
AWS_PROFILE=spore-host-infra aws lambda create-function \
  --function-name spawn-dns-updater \
  --runtime provided.al2023 \
  --role arn:aws:iam::966362334030:role/SpawnDNSUpdaterRole \
  --handler bootstrap \
  --zip-file fileb:///tmp/lambda.zip \
  --timeout 30 \
  --memory-size 512 \
  --environment Variables={HOSTED_ZONE_ID=<NEW_ZONE_ID>}
```

**5.4 Update API Gateway**
```bash
# Update Lambda integration in API Gateway
# (or recreate API Gateway in infrastructure account)
```

**5.5 Delete Old Lambda**
```bash
AWS_PROFILE=management aws lambda delete-function \
  --function-name spawn-dns-updater
```

### Phase 6: Migrate Cognito (If Exists)

**Note:** Cognito is not yet created, but should be created directly in infrastructure account.

```bash
# When setting up Cognito, use spore-host-infra profile
AWS_PROFILE=spore-host-infra ./scripts/setup-dashboard-cognito.sh
```

### Phase 7: Update Spawn CLI Configuration

**7.1 Update CLAUDE.md**
```bash
# Edit: /Users/scttfrdmn/src/spore-host/spawn/CLAUDE.md
# Replace references to 'default' account with 'spore-host-infra'
# Replace references to 'aws' account with 'spore-host-dev'
```

**7.2 Update Scripts**
```bash
# Find all scripts using AWS_PROFILE=default
grep -r "AWS_PROFILE=default" /Users/scttfrdmn/src/spore-host/scripts/

# Replace with AWS_PROFILE=spore-host-infra
sed -i '' 's/AWS_PROFILE=default/AWS_PROFILE=spore-host-infra/g' \
  /Users/scttfrdmn/src/spore-host/scripts/*.sh

# Find all scripts using AWS_PROFILE=aws
grep -r "AWS_PROFILE=aws" /Users/scttfrdmn/src/spore-host/

# Replace with AWS_PROFILE=spore-host-dev
sed -i '' 's/AWS_PROFILE=aws/AWS_PROFILE=spore-host-dev/g' \
  /Users/scttfrdmn/src/spore-host/scripts/*.sh
```

**7.3 Update README and Documentation**
```bash
# Update all documentation to reference new profiles
# Test all examples with new profiles
```

### Phase 8: Update Dashboard Configuration

**8.1 Update DASHBOARD_STATUS.md**
```bash
# Replace account ID references:
# - 752123829273 → 966362334030 (for infrastructure)
# - Update profile references: default → spore-host-infra
```

**8.2 Update Deployment Scripts**
```bash
# Update web/deploy.sh to use spore-host-infra profile
sed -i '' 's/AWS_PROFILE=default/AWS_PROFILE=spore-host-infra/g' \
  /Users/scttfrdmn/src/spore-host/web/deploy.sh
```

---

## Quick Reference

### Account IDs

| Account | ID | Email | Profile |
|---------|-----|-------|---------|
| Management | 752123829273 | scttfrdmn@gmail.com | `management` |
| Infrastructure | 966362334030 | scttfrdmn+spore-host-infra@gmail.com | `spore-host-infra` |
| Development | 435415984226 | scttfrdmn+spore-host-dev@gmail.com | `spore-host-dev` |
| ~~AWS~~ (deprecated) | 942542972736 | - | ~~`aws`~~ |

### Common Commands

**Check Current Identity:**
```bash
# Management account
AWS_PROFILE=management aws sts get-caller-identity

# Infrastructure account
AWS_PROFILE=spore-host-infra aws sts get-caller-identity

# Development account
AWS_PROFILE=spore-host-dev aws sts get-caller-identity
```

**Launch Test Instance:**
```bash
# Use development account
AWS_PROFILE=spore-host-dev ./bin/spawn launch \
  --instance-type t3.micro \
  --name test-instance \
  --ttl 2h
```

**Deploy Website:**
```bash
# Use infrastructure account
cd /Users/scttfrdmn/src/spore-host/web
AWS_PROFILE=spore-host-infra ./deploy.sh
```

**Manage DNS:**
```bash
# Use infrastructure account
AWS_PROFILE=spore-host-infra aws route53 list-hosted-zones
```

**Update Lambda:**
```bash
# Use infrastructure account
AWS_PROFILE=spore-host-infra aws lambda update-function-code \
  --function-name spawn-dns-updater \
  --zip-file fileb://function.zip
```

---

## Migration Timeline (Estimated)

| Phase | Task | Time | Complexity |
|-------|------|------|------------|
| 1 | Prepare Infrastructure Account | 30 min | Low |
| 2 | Migrate DNS (Route53) | 2-3 days* | Medium |
| 3 | Migrate S3 Buckets | 2 hours | Low |
| 4 | Migrate CloudFront | 1 hour | Medium |
| 5 | Migrate Lambda | 1 hour | Low |
| 6 | Migrate Cognito | N/A | N/A |
| 7 | Update Spawn CLI | 1 hour | Low |
| 8 | Update Documentation | 1 hour | Low |

*DNS migration includes 24-48 hours for DNS propagation

**Total Active Work:** ~8 hours
**Total Calendar Time:** 2-3 days (waiting for DNS propagation)

---

## Testing Checklist

After migration, verify:

- [ ] DNS resolves correctly: `dig spore.host`
- [ ] Website loads: https://spore.host
- [ ] CloudFront serves content
- [ ] Lambda function responds to API calls
- [ ] Binary downloads work from S3
- [ ] spawn CLI can launch instances in dev account
- [ ] Dashboard authentication works (after Cognito setup)
- [ ] All scripts use correct profiles
- [ ] Documentation is updated

---

## Rollback Plan

If migration fails:

1. **DNS:** Keep old hosted zone active until new one verified
2. **S3:** Don't delete old buckets until data verified in new account
3. **CloudFront:** Keep old distribution active during transition
4. **Lambda:** Deploy to new account without deleting old one
5. **Profiles:** Keep legacy profiles active during migration

**Critical:** Don't delete resources in management account until verified in infrastructure account.

---

## Next Steps

1. ✅ Create sub-accounts (completed)
2. ✅ Set up IAM roles (completed)
3. ✅ Configure CLI profiles (completed)
4. ⏳ **Execute migration plan** (start with Phase 1)
5. ⏳ Test all functionality
6. ⏳ Update documentation
7. ⏳ Delete legacy resources
8. ⏳ Retire `default` and `aws` profiles

---

## Questions?

- AWS Organizations: https://docs.aws.amazon.com/organizations/
- Cross-Account Access: https://docs.aws.amazon.com/IAM/latest/UserGuide/tutorial_cross-account-with-roles.html
- Resource Migration: https://aws.amazon.com/premiumsupport/knowledge-center/

