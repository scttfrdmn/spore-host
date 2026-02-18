# DNS Migration Status - spore.host

**Migration Date:** 2025-12-30
**Status:** ✅ DNS records migrated, ⏳ awaiting nameserver update at registrar

---

## ✅ Completed

1. **New Hosted Zone Created** in Infrastructure Account (966362334030)
   - Zone ID: `Z0341053304H0DQXF6U4X`
   - Region: `us-east-1`

2. **All DNS Records Migrated:**
   - ✅ spore.host. → CloudFront (Alias A record)
   - ✅ spore.host. → Google verification (TXT record)
   - ✅ _3edc56ea7b394db6f4239ed536c3a44d.spore.host. → ACM validation (CNAME)
   - ✅ test-base36.c0zxr0ao.spore.host. → 3.90.216.63 (A record)
   - ✅ test-fixed.c0zxr0ao.spore.host. → 54.163.216.97 (A record)
   - ✅ dns-api-test.spore.host. → 54.162.169.222 (A record)
   - ✅ test-dns-flag.spore.host. → 54.164.27.106 (A record)

---

## ⚠️ ACTION REQUIRED: Update Nameservers at Registrar

You need to update the nameservers for `spore.host` at your domain registrar.

### Old Nameservers (Management Account)
```
ns-1774.awsdns-29.co.uk
ns-422.awsdns-52.com
ns-816.awsdns-38.net
ns-1091.awsdns-08.org
```

### New Nameservers (Infrastructure Account)
```
ns-827.awsdns-39.net
ns-1175.awsdns-18.org
ns-425.awsdns-53.com
ns-1874.awsdns-42.co.uk
```

### How to Update Nameservers

1. **Log into your domain registrar** (where you registered spore.host)
   - Common registrars: GoDaddy, Namecheap, Route53 Registrar, Google Domains, etc.

2. **Find DNS/Nameserver settings** for spore.host

3. **Replace the nameservers** with the new ones listed above

4. **Save changes** (may take a few minutes to process)

5. **Wait for DNS propagation** (typically 24-48 hours, but can be faster)

### Verify DNS Propagation

After updating nameservers, you can check propagation status:

```bash
# Check current nameservers
dig spore.host NS +short

# Check if website resolves
dig spore.host +short

# Test from multiple locations
# https://www.whatsmydns.net/#NS/spore.host
```

Expected result after propagation:
```
ns-827.awsdns-39.net.
ns-1175.awsdns-18.org.
ns-425.awsdns-53.com.
ns-1874.awsdns-42.co.uk.
```

---

## ⏳ After DNS Propagation

Once DNS has fully propagated (verify with `dig` commands above), you can:

1. **Delete old hosted zone** in management account
   ```bash
   # Wait at least 48 hours after nameserver update
   AWS_PROFILE=management aws route53 delete-hosted-zone \
     --id Z048907324UNXKEK9KX93
   ```

2. **Continue with S3/CloudFront migration** (Phase 3-4)

---

## Important Notes

- **Both hosted zones will work during migration** - old nameservers → old zone, new nameservers → new zone
- **DNS queries will use old zone until registrar updated** - no downtime
- **Don't delete old hosted zone until fully propagated** - critical!
- **TTL is low (60s for most records)** - changes propagate quickly after NS update
- **CloudFront will continue working** - points to same distribution in both zones

---

## Current Status

| Task | Status |
|------|--------|
| Create new hosted zone | ✅ Complete |
| Migrate DNS records | ✅ Complete |
| Update nameservers at registrar | ⏳ **Awaiting action** |
| Verify DNS propagation | ⏳ Pending (after NS update) |
| Delete old hosted zone | ⏳ Pending (after propagation) |

---

## Quick Reference

**Old Hosted Zone (Management Account):**
- Account: 752123829273
- Zone ID: Z048907324UNXKEK9KX93
- Profile: `management`

**New Hosted Zone (Infrastructure Account):**
- Account: 966362334030
- Zone ID: Z0341053304H0DQXF6U4X
- Profile: `mycelium-infra`

**Commands:**
```bash
# View old zone records
AWS_PROFILE=management aws route53 list-resource-record-sets \
  --hosted-zone-id Z048907324UNXKEK9KX93

# View new zone records
AWS_PROFILE=mycelium-infra aws route53 list-resource-record-sets \
  --hosted-zone-id Z0341053304H0DQXF6U4X

# Check nameservers from new zone
AWS_PROFILE=mycelium-infra aws route53 get-hosted-zone \
  --id Z0341053304H0DQXF6U4X \
  --query 'DelegationSet.NameServers'
```
