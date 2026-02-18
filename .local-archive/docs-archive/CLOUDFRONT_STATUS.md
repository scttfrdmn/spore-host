# CloudFront Migration Status - Updated

**Date:** 2025-12-30
**Status:** ✅ Distribution Created, ⏳ Deploying

---

## Current Status

**✅ Completed:**
- S3 website bucket created and configured (`spore-host-website`)
- ACM certificate requested: `arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9`
- DNS validation record added to Route53
- CloudFront distribution created: **EY67INS5HDFLU**
  - Domain: `d1hcjyt3z5xzq4.cloudfront.net`
  - Status: InProgress → Deployed (15-30 minutes)
  - Currently using CloudFront default certificate (*.cloudfront.net)

**⏳ Pending:**
- ACM certificate validation (requires nameserver update at registrar)
- CloudFront distribution deployment (15-30 minutes)
- Update CloudFront to use custom domain (spore.host) after certificate validates
- Update Route53 A record to point to CloudFront
- Binary buckets (spawn-binaries-*) creation (S3 propagation delay)

---

## Distribution Details

| Property | Value |
|----------|-------|
| **Distribution ID** | EY67INS5HDFLU |
| **Domain Name** | d1hcjyt3z5xzq4.cloudfront.net |
| **Origin** | spore-host-website.s3-website-us-east-1.amazonaws.com |
| **Status** | InProgress |
| **SSL Certificate** | CloudFront default (temporary) |
| **Custom Domain** | None (will add spore.host after cert validation) |

---

## Testing the Distribution

**Wait for deployment to complete:**
```bash
AWS_PROFILE=mycelium-infra aws cloudfront wait distribution-deployed --id EY67INS5HDFLU
```

**Test the CloudFront domain:**
```bash
curl -I https://d1hcjyt3z5xzq4.cloudfront.net
```

**Check deployment status:**
```bash
AWS_PROFILE=mycelium-infra aws cloudfront get-distribution \
  --id EY67INS5HDFLU \
  --query 'Distribution.Status' \
  --output text
```

---

## Next Steps

### Step 1: Update Nameservers at Domain Registrar

**Current nameservers** (old hosted zone):
```
ns-1774.awsdns-29.co.uk
ns-422.awsdns-52.com
ns-706.awsdns-24.net
ns-1191.awsdns-20.org
```

**New nameservers** (infrastructure account hosted zone):
```
ns-827.awsdns-39.net
ns-1175.awsdns-18.org
ns-425.awsdns-53.com
ns-1874.awsdns-42.co.uk
```

**Action Required:** Update nameservers at your domain registrar (e.g., Namecheap, GoDaddy, Route53)

### Step 2: Wait for ACM Certificate Validation

Once nameservers are updated (24-48 hour propagation):

```bash
# Check certificate status
AWS_PROFILE=mycelium-infra aws acm describe-certificate \
  --certificate-arn arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9 \
  --query 'Certificate.Status' \
  --output text
```

Expected: `PENDING_VALIDATION` → `ISSUED`

### Step 3: Update CloudFront to Add Custom Domain

Once certificate is validated, update the distribution:

```bash
# Get current config
AWS_PROFILE=mycelium-infra aws cloudfront get-distribution-config \
  --id EY67INS5HDFLU \
  --output json > /tmp/cloudfront-current-config.json

# Extract ETag
ETAG=$(cat /tmp/cloudfront-current-config.json | jq -r '.ETag')

# Update config to add custom domain and certificate
cat /tmp/cloudfront-current-config.json | jq '
  .DistributionConfig.Aliases.Quantity = 1 |
  .DistributionConfig.Aliases.Items = ["spore.host"] |
  .DistributionConfig.ViewerCertificate = {
    "ACMCertificateArn": "arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9",
    "SSLSupportMethod": "sni-only",
    "MinimumProtocolVersion": "TLSv1.2_2021",
    "Certificate": "arn:aws:acm:us-east-1:966362334030:certificate/55203b10-e7cc-46d2-a3fd-d134d8f523d9",
    "CertificateSource": "acm"
  } |
  .DistributionConfig.DefaultCacheBehavior.ViewerProtocolPolicy = "redirect-to-https"
' | jq '.DistributionConfig' > /tmp/cloudfront-updated-config.json

# Apply update
AWS_PROFILE=mycelium-infra aws cloudfront update-distribution \
  --id EY67INS5HDFLU \
  --distribution-config file:///tmp/cloudfront-updated-config.json \
  --if-match "$ETAG"
```

### Step 4: Update Route53 A Record

Point spore.host to the CloudFront distribution:

```bash
cat > /tmp/dns-update-cloudfront.json <<'EOF'
{
  "Changes": [{
    "Action": "UPSERT",
    "ResourceRecordSet": {
      "Name": "spore.host.",
      "Type": "A",
      "AliasTarget": {
        "HostedZoneId": "Z2FDTNDATAQYW2",
        "DNSName": "d1hcjyt3z5xzq4.cloudfront.net.",
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

**Note:** Z2FDTNDATAQYW2 is the CloudFront hosted zone ID (constant for all distributions)

### Step 5: Verify Everything Works

```bash
# Test CloudFront directly
curl -I https://d1hcjyt3z5xzq4.cloudfront.net

# Test custom domain (after DNS update)
curl -I https://spore.host

# Check DNS
dig spore.host
```

---

## Summary

**What's Working Now:**
- ✅ Website files in S3 (spore-host-website)
- ✅ CloudFront distribution created (deploying)
- ✅ Accessible via d1hcjyt3z5xzq4.cloudfront.net (once deployed)

**What's Pending:**
- ⏳ Nameserver update at domain registrar
- ⏳ ACM certificate validation (after nameserver update)
- ⏳ Add custom domain to CloudFront (after cert validation)
- ⏳ Binary buckets (S3 propagation)

**Critical Path:**
1. Update nameservers → 2. Wait for cert validation → 3. Update CloudFront config → 4. Update DNS → Done

---

## Resources

- [CloudFront Distribution](https://console.aws.amazon.com/cloudfront/v3/home?region=us-east-1#/distributions/EY67INS5HDFLU)
- [ACM Certificate](https://console.aws.amazon.com/acm/home?region=us-east-1#/certificates/55203b10-e7cc-46d2-a3fd-d134d8f523d9)
- [Route53 Hosted Zone](https://console.aws.amazon.com/route53/v2/hostedzones#ListRecordSets/Z0341053304H0DQXF6U4X)
