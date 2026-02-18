# CloudFront Migration Status

**Date:** 2025-12-30
**Status:** ⏳ Waiting for dependencies

---

## Current Status

**✅ Completed:**
- ACM certificate requested in infrastructure account
  - Certificate ARN: `arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9`
  - Domains: `spore.host`, `*.spore.host`
  - DNS validation record added to Route53
  - Status: Pending validation (5-30 minutes)

**⏳ Pending:**
- ACM certificate validation
- S3 bucket creation (`spore-host-website`)
- CloudFront distribution creation
- DNS update to point to new distribution

---

## Dependencies

1. **S3 Bucket:** `spore-host-website` must exist in infrastructure account
   - Currently: Waiting for name propagation
   - See: `S3_MIGRATION_STATUS.md`

2. **ACM Certificate:** Must be validated
   - Currently: Pending DNS validation
   - Check status:
     ```bash
     AWS_PROFILE=mycelium-infra aws acm describe-certificate \
       --certificate-arn arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9 \
       --region us-east-1 \
       --query 'Certificate.Status'
     ```

---

## Step-by-Step Completion

### Step 1: Wait for ACM Certificate Validation

**Check status:**
```bash
AWS_PROFILE=mycelium-infra aws acm describe-certificate \
  --certificate-arn arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9 \
  --region us-east-1 \
  --query 'Certificate.Status' \
  --output text
```

**Expected:** `ISSUED` (takes 5-30 minutes)

### Step 2: Wait for S3 Bucket Creation

**Ensure bucket exists:**
```bash
AWS_PROFILE=mycelium-infra aws s3 ls s3://spore-host-website
```

**If not exists:** Run S3 migration completion script (see `S3_MIGRATION_STATUS.md`)

### Step 3: Create CloudFront Distribution

**Save this configuration to `/tmp/cloudfront-new-config.json`:**
```json
{
  "CallerReference": "spore-host-migration-2025-12-30",
  "Comment": "spore.host website - infrastructure account",
  "Enabled": true,
  "Origins": {
    "Quantity": 1,
    "Items": [{
      "Id": "S3-spore-host-website",
      "DomainName": "spore-host-website.s3-website-us-east-1.amazonaws.com",
      "OriginPath": "",
      "CustomHeaders": {"Quantity": 0},
      "CustomOriginConfig": {
        "HTTPPort": 80,
        "HTTPSPort": 443,
        "OriginProtocolPolicy": "http-only",
        "OriginSslProtocols": {
          "Quantity": 3,
          "Items": ["TLSv1", "TLSv1.1", "TLSv1.2"]
        },
        "OriginReadTimeout": 30,
        "OriginKeepaliveTimeout": 5
      },
      "ConnectionAttempts": 3,
      "ConnectionTimeout": 10,
      "OriginShield": {"Enabled": false}
    }]
  },
  "DefaultRootObject": "index.html",
  "DefaultCacheBehavior": {
    "TargetOriginId": "S3-spore-host-website",
    "ViewerProtocolPolicy": "redirect-to-https",
    "AllowedMethods": {
      "Quantity": 2,
      "Items": ["GET", "HEAD"],
      "CachedMethods": {
        "Quantity": 2,
        "Items": ["GET", "HEAD"]
      }
    },
    "Compress": true,
    "ForwardedValues": {
      "QueryString": false,
      "Cookies": {"Forward": "none"},
      "Headers": {"Quantity": 0}
    },
    "MinTTL": 0,
    "DefaultTTL": 86400,
    "MaxTTL": 31536000,
    "TrustedSigners": {
      "Enabled": false,
      "Quantity": 0
    },
    "TrustedKeyGroups": {
      "Enabled": false,
      "Quantity": 0
    },
    "SmoothStreaming": false,
    "FieldLevelEncryptionId": ""
  },
  "CacheBehaviors": {"Quantity": 0},
  "CustomErrorResponses": {"Quantity": 0},
  "Aliases": {
    "Quantity": 1,
    "Items": ["spore.host"]
  },
  "ViewerCertificate": {
    "ACMCertificateArn": "arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9",
    "SSLSupportMethod": "sni-only",
    "MinimumProtocolVersion": "TLSv1.2_2021",
    "Certificate": "arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9",
    "CertificateSource": "acm"
  },
  "PriceClass": "PriceClass_100",
  "HttpVersion": "http2and3",
  "IsIPV6Enabled": true,
  "Logging": {
    "Enabled": false,
    "IncludeCookies": false,
    "Bucket": "",
    "Prefix": ""
  },
  "WebACLId": "",
  "Restrictions": {
    "GeoRestriction": {
      "RestrictionType": "none",
      "Quantity": 0
    }
  }
}
```

**Create distribution:**
```bash
AWS_PROFILE=mycelium-infra aws cloudfront create-distribution \
  --distribution-config file:///tmp/cloudfront-new-config.json \
  --output json > /tmp/cloudfront-new-distribution.json

# Get the new distribution ID and domain
NEW_DIST_ID=$(cat /tmp/cloudfront-new-distribution.json | jq -r '.Distribution.Id')
NEW_DIST_DOMAIN=$(cat /tmp/cloudfront-new-distribution.json | jq -r '.Distribution.DomainName')

echo "New Distribution ID: $NEW_DIST_ID"
echo "New Distribution Domain: $NEW_DIST_DOMAIN"
```

### Step 4: Wait for Distribution Deployment

**Check status:**
```bash
AWS_PROFILE=mycelium-infra aws cloudfront get-distribution \
  --id $NEW_DIST_ID \
  --query 'Distribution.Status' \
  --output text
```

**Expected:** `InProgress` → `Deployed` (takes 15-30 minutes)

### Step 5: Update DNS to Point to New Distribution

**Create DNS update:**
```bash
cat > /tmp/dns-update-cloudfront.json <<EOF
{
  "Changes": [{
    "Action": "UPSERT",
    "ResourceRecordSet": {
      "Name": "spore.host.",
      "Type": "A",
      "AliasTarget": {
        "HostedZoneId": "Z2FDTNDATAQYW2",
        "DNSName": "$NEW_DIST_DOMAIN.",
        "EvaluateTargetHealth": false
      }
    }
  }]
}
EOF

AWS_PROFILE=mycelium-infra aws route53 change-resource-record-sets \
  --hosted-zone-id Z0341053304H0DQXF6U4X \
  --change-batch file:///tmp/dns-update-cloudfront.json
```

**Note:** Z2FDTNDATAQYW2 is the hosted zone ID for CloudFront (constant for all CloudFront distributions)

### Step 6: Test New Distribution

```bash
# Test CloudFront directly
curl -I https://$NEW_DIST_DOMAIN

# Test via domain (after DNS propagates)
curl -I https://spore.host

# Check DNS
dig spore.host
```

### Step 7: Delete Old Distribution (After Verification)

**Disable old distribution:**
```bash
# Get old distribution config
AWS_PROFILE=management aws cloudfront get-distribution-config \
  --id E50GL663TTL0I \
  --output json > /tmp/old-dist-config.json

# Get ETag
OLD_ETAG=$(cat /tmp/old-dist-config.json | jq -r '.ETag')

# Update config to disable
cat /tmp/old-dist-config.json | jq '.DistributionConfig.Enabled = false' > /tmp/old-dist-disabled.json

# Update distribution
AWS_PROFILE=management aws cloudfront update-distribution \
  --id E50GL663TTL0I \
  --distribution-config file:///tmp/old-dist-disabled.json \
  --if-match $OLD_ETAG
```

**Wait for deployment, then delete:**
```bash
# Wait for status = Deployed
AWS_PROFILE=management aws cloudfront wait distribution-deployed --id E50GL663TTL0I

# Get new ETag
OLD_ETAG=$(AWS_PROFILE=management aws cloudfront get-distribution --id E50GL663TTL0I --query 'ETag' --output text)

# Delete distribution
AWS_PROFILE=management aws cloudfront delete-distribution \
  --id E50GL663TTL0I \
  --if-match $OLD_ETAG
```

---

## Quick Completion Script

Once dependencies are met, run this script:

```bash
#!/bin/bash
set -e

echo "=== CloudFront Migration Completion ==="

# 1. Verify ACM certificate
echo "Checking ACM certificate..."
CERT_STATUS=$(AWS_PROFILE=mycelium-infra aws acm describe-certificate \
  --certificate-arn arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9 \
  --region us-east-1 \
  --query 'Certificate.Status' \
  --output text)

if [ "$CERT_STATUS" != "ISSUED" ]; then
  echo "✗ Certificate not yet issued: $CERT_STATUS"
  exit 1
fi
echo "✓ Certificate issued"

# 2. Verify S3 bucket exists
echo "Checking S3 bucket..."
if ! AWS_PROFILE=mycelium-infra aws s3 ls s3://spore-host-website > /dev/null 2>&1; then
  echo "✗ S3 bucket does not exist"
  exit 1
fi
echo "✓ S3 bucket exists"

# 3. Create CloudFront distribution
echo "Creating CloudFront distribution..."
AWS_PROFILE=mycelium-infra aws cloudfront create-distribution \
  --distribution-config file:///tmp/cloudfront-new-config.json \
  --output json > /tmp/cloudfront-new-distribution.json

NEW_DIST_ID=$(cat /tmp/cloudfront-new-distribution.json | jq -r '.Distribution.Id')
NEW_DIST_DOMAIN=$(cat /tmp/cloudfront-new-distribution.json | jq -r '.Distribution.DomainName')

echo "✓ Distribution created: $NEW_DIST_ID"
echo "  Domain: $NEW_DIST_DOMAIN"

# 4. Wait for deployment
echo "Waiting for distribution deployment (15-30 min)..."
AWS_PROFILE=mycelium-infra aws cloudfront wait distribution-deployed --id $NEW_DIST_ID
echo "✓ Distribution deployed"

# 5. Update DNS
echo "Updating DNS..."
cat > /tmp/dns-update-cloudfront.json <<EOF
{
  "Changes": [{
    "Action": "UPSERT",
    "ResourceRecordSet": {
      "Name": "spore.host.",
      "Type": "A",
      "AliasTarget": {
        "HostedZoneId": "Z2FDTNDATAQYW2",
        "DNSName": "$NEW_DIST_DOMAIN.",
        "EvaluateTargetHealth": false
      }
    }
  }]
}
EOF

AWS_PROFILE=mycelium-infra aws route53 change-resource-record-sets \
  --hosted-zone-id Z0341053304H0DQXF6U4X \
  --change-batch file:///tmp/dns-update-cloudfront.json

echo "✓ DNS updated"

echo ""
echo "=== CloudFront Migration Complete! ==="
echo "New distribution: $NEW_DIST_ID"
echo "Domain: $NEW_DIST_DOMAIN"
echo "Website: https://spore.host"
```

Save as `/tmp/complete-cloudfront-migration.sh` and run when ready.

---

## Current State Summary

| Component | Old (Management) | New (Infrastructure) | Status |
|-----------|------------------|----------------------|--------|
| **ACM Certificate** | arn:...752123829273:.../59bec0fb-28e5-4c9a-a313-af5a3249ea58 | arn:...966362334030:.../55203b10-e7cc-46d2-a3fd-d134d8f523d9 | ⏳ Pending validation |
| **S3 Origin** | spore-host-website.s3-website-us-east-1.amazonaws.com | (same, but in new account) | ⏳ Bucket doesn't exist yet |
| **CloudFront Distribution** | E50GL663TTL0I (d2b6b8labdbh8l.cloudfront.net) | Not yet created | ⏳ Waiting for dependencies |
| **DNS A Record** | Points to old CloudFront | (same domain) | ✅ Ready to update |

---

## Resources

- [CloudFront Migration Guide](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/introduction.html)
- [ACM Certificate Validation](https://docs.aws.amazon.com/acm/latest/userguide/dns-validation.html)
- [CloudFront Distributions](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/distribution-working-with.html)
