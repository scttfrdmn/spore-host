# Multi-Region

## What

spawn supports launching instances across multiple AWS regions and AWS accounts from a single CLI invocation.

## Why

- Launch instances close to data sources
- Improve redundancy and availability
- Use cheaper regions for batch workloads
- Meet data sovereignty requirements

## Basic Multi-Region Usage

```bash
# Launch in a specific region
spawn launch --region eu-west-1 --instance-type t3.large

# Launch in multiple regions simultaneously
spawn launch \
  --region us-east-1,eu-west-1,ap-southeast-1 \
  --instance-type t3.large

# List all instances across all regions
truffle ls --region all
```

## Multi-Account Setup

### AWS Organizations Approach

1. **Infrastructure account**: S3, Lambda, Route53 for shared infrastructure
2. **Workload accounts**: EC2, DynamoDB per project or team

### Cross-Account Configuration

```yaml
# ~/.spawn/config.yaml
accounts:
  - name: production
    profile: prod-profile
    regions: [us-east-1, us-west-2]
  - name: research
    profile: research-profile
    regions: [us-east-1, eu-west-1]
```

Use:
```bash
spawn launch --account production --region us-east-1
truffle ls --account all
```

### Cross-Account IAM Roles

Create trust relationships between accounts:

```json
{
  "Effect": "Allow",
  "Principal": {
    "AWS": "arn:aws:iam::CENTRAL_ACCOUNT:root"
  },
  "Action": "sts:AssumeRole",
  "Condition": {
    "StringEquals": {
      "sts:ExternalId": "spawn-cross-account"
    }
  }
}
```

Configure in spawn:
```yaml
accounts:
  - name: workload-account
    role_arn: arn:aws:iam::WORKLOAD_ACCOUNT:role/SpawnRole
    external_id: spawn-cross-account
```

## Regional Cost Differences

Instance pricing varies by region. Typical savings by region vs us-east-1:
- us-west-2: similar
- eu-west-1: +5-10%
- ap-southeast-1: +10-20%
- us-east-2: similar or slightly cheaper

Use `spawn pricing --region all --instance-type c7i.large` to compare.

## Limitations

- Route53 DNS (`spore.host`) requires the infrastructure Lambda to have Route53 access
- Multi-account setup requires cross-account IAM configuration
- Some features (autoscaling) require additional setup per account
