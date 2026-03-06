# DNSSEC Configuration for spore.host

**Date:** 2025-12-30
**Status:** ✅ Enabled in Route53, ⏳ Pending registrar update

---

## DS Record for Domain Registrar

You need to add this DS (Delegation Signer) record to your domain registrar:

### Full DS Record (Single Line)
```
10831 13 2 F324476F158C8FA41966789CD6E04F793009AE872EA021DC0F387BF217CDAE5A
```

### DS Record Fields (for registrar web forms)

| Field | Value |
|-------|-------|
| **Key Tag** | 10831 |
| **Algorithm** | 13 (ECDSAP256SHA256) |
| **Digest Type** | 2 (SHA-256) |
| **Digest** | F324476F158C8FA41966789CD6E04F793009AE872EA021DC0F387BF217CDAE5A |

---

## How to Add DS Record at Your Registrar

### General Steps:
1. Log in to your domain registrar (e.g., Namecheap, GoDaddy, Route53 Registrar)
2. Navigate to domain management for `spore.host`
3. Find DNSSEC settings (usually under "Advanced DNS" or "Security")
4. Add DS record with the values above
5. Save changes

### Common Registrar Instructions:

**Namecheap:**
1. Domain List → Manage → Advanced DNS
2. Scroll to "DNSSEC"
3. Click "Add DS Record"
4. Enter values from table above
5. Save

**GoDaddy:**
1. My Products → Domains → Manage
2. Additional Settings → Manage DNSSEC
3. Enter DS record values
4. Save

**Route53 Registrar:**
```bash
# If your domain is registered with Route53
AWS_PROFILE=spore-host-infra aws route53domains enable-domain-auto-renew \
  --domain-name spore.host

# Get domain info
AWS_PROFILE=spore-host-infra aws route53domains get-domain-detail \
  --domain-name spore.host
```

---

## Verification

After adding the DS record to your registrar (24-48 hour propagation):

```bash
# Check DNSSEC validation
dig spore.host +dnssec

# Check DS record at registrar
dig DS spore.host

# Verify DNSSEC chain
delv spore.host
```

Expected output should show `ad` (authenticated data) flag.

---

## Route53 DNSSEC Details

**Hosted Zone ID:** Z0341053304H0DQXF6U4X
**Key Signing Key (KSK) Name:** spore-host-ksk
**KMS Key ARN:** arn:aws:kms:us-east-1:966362334030:key/0e41f267-eec6-498e-a24a-094d5d56a228
**DNSSEC Status:** SIGNING

**Check status:**
```bash
AWS_PROFILE=spore-host-infra aws route53 get-dnssec \
  --hosted-zone-id Z0341053304H0DQXF6U4X
```

**DNSKEY Record (for reference):**
```
257 3 13 pJ0Ul25hs3e+SGFO6lNI23jET6gDwcUWq9aD8BrDs1ruSmO9msETxQmDmvwjd/p8ZpIl8qRTWTKo9/jLRhI0YA==
```

---

## Migration Notes

**Old DS Record (management account):**
- Key Tag: 12735
- Digest: 0179EFB5FA92E41D46256E7C1D8628B9DD7C0529E85E400F9B48213685BBA5E4

**New DS Record (infrastructure account):**
- Key Tag: 10831
- Digest: F324476F158C8FA41966789CD6E04F793009AE872EA021DC0F387BF217CDAE5A

⚠️ **Important:** Replace the old DS record with the new one at your registrar. Do not leave both!

---

## Troubleshooting

**If DNSSEC validation fails:**

1. **Check DS record at registrar:**
   ```bash
   dig DS spore.host @8.8.8.8
   ```

2. **Check Route53 is signing:**
   ```bash
   dig DNSKEY spore.host @ns-827.awsdns-39.net
   ```

3. **Verify KMS key permissions:**
   ```bash
   AWS_PROFILE=spore-host-infra aws kms describe-key \
     --key-id 0e41f267-eec6-498e-a24a-094d5d56a228
   ```

4. **Check DNSSEC status:**
   ```bash
   AWS_PROFILE=spore-host-infra aws route53 get-dnssec \
     --hosted-zone-id Z0341053304H0DQXF6U4X \
     --query 'Status.ServeSignature'
   ```

**Common Issues:**
- DS record propagation takes 24-48 hours
- Old DS record still present (remove it!)
- Nameservers not updated yet (check with `dig NS spore.host`)
- DNSSEC chain broken (check parent zone)

---

## Resources

- [Route53 DNSSEC Documentation](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/dns-configuring-dnssec.html)
- [DNSSEC Validation Testing](https://dnssec-analyzer.verisignlabs.com/)
- [Dig DNSSEC Tutorial](https://www.digwebinterface.com/)
