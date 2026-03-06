# Claude Development Notes

## AWS Account Configuration

**CRITICAL**: Resources are organized in an AWS Organization with separate accounts:

- **`management` profile** (Account: 752123829273)
  - Organization administration ONLY
  - IAM user: scott-admin
  - **NO application workloads**

- **`spore-host-infra` profile** (Account: 966362334030)
  - S3 buckets (spawn-binaries-*, spore-host-website)
  - Lambda functions (spawn-dns-updater)
  - Route53 DNS (spore.host hosted zone)
  - CloudFront distribution
  - Cognito Identity Pool
  - **Production infrastructure**
  - **NO EC2 instances**

- **`spore-host-dev` profile** (Account: 435415984226)
  - **ALL EC2 instance provisioning**
  - Test instances
  - Development/testing instances
  - **NO infrastructure resources**

- **DEPRECATED: `default` profile** - DO NOT USE (being phased out)
- **DEPRECATED: `aws` profile** - DO NOT USE (Account 942542972736 no longer used)

**Cross-Account Requirements**:
- EC2 instances in spore-host-dev account need access to:
  - S3 bucket in spore-host-infra account (for spored binary downloads)
  - Lambda DNS API in spore-host-infra account (for DNS registration)

**Usage Examples**:
```bash
# Upload spored binary to S3 (use spore-host-infra profile)
AWS_PROFILE=spore-host-infra aws s3 cp bin/spored s3://spawn-binaries-us-east-1/spored-linux-amd64

# Launch instances (use spore-host-dev profile ONLY)
AWS_PROFILE=spore-host-dev ./bin/spawn launch --instance-type t3.micro ...

# Deploy Lambda function (use spore-host-infra profile)
AWS_PROFILE=spore-host-infra aws lambda update-function-code --function-name spawn-dns-updater ...

# Deploy website (use spore-host-infra profile)
cd web && AWS_PROFILE=spore-host-infra ./deploy.sh
```

## DNS Implementation

DNS uses base36-encoded account IDs for subdomain isolation:
- Format: `<name>.<account-base36>.spore.host`
- Management Account 752123829273 → Base36: `c0zxr0ao` (DEPRECATED)
- Infrastructure Account 966362334030 → Base36: TBD (calculate when needed)
- Development Account 435415984226 → Base36: TBD (calculate when needed)
- Example: `test-base36.c0zxr0ao.spore.host` (uses old account base36)

## Spored Agent

The spored agent runs on EC2 instances and handles:
- Automatic DNS registration on startup
- DNS cleanup on termination (SIGTERM, TTL, idle, Spot interruption)
- Idle detection and auto-termination
- Hibernation support
