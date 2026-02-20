# Authentication

spawn uses standard AWS credential resolution. No special setup is required if you already have AWS CLI configured.

## AWS Credential Resolution Order

1. CLI flags: `--profile`, `--region`
2. Environment variables: `AWS_PROFILE`, `AWS_ACCESS_KEY_ID`, etc.
3. AWS config file: `~/.aws/credentials` and `~/.aws/config`
4. EC2 instance profile (when running on EC2)
5. ECS/EKS task role

## Basic Setup

```bash
# Option 1: AWS CLI configure
aws configure

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_DEFAULT_REGION=us-east-1

# Option 3: Named profile
aws configure --profile myprofile
spawn launch --profile myprofile
```

## Minimum IAM Permissions

Create an IAM policy with these minimum permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:TerminateInstances",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceStatus",
        "ec2:CreateTags",
        "ec2:DescribeTags",
        "ec2:DescribeImages",
        "ec2:DescribeSubnets",
        "ec2:DescribeVpcs",
        "ec2:DescribeSecurityGroups",
        "ec2:CreateSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:CreateKeyPair",
        "ec2:DescribeKeyPairs",
        "ec2:ModifyInstanceAttribute",
        "ec2:StopInstances",
        "ec2:StartInstances"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/spawn-*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:PassRole"
      ],
      "Resource": "arn:aws:iam::*:role/spawn-instance-role"
    }
  ]
}
```

For the full IAM policy reference, see `spawn/docs/reference/iam-policies.md`.

## IAM Roles (Recommended)

For production use, use IAM roles instead of long-lived credentials:

**EC2 instance profile** (when running spawn on EC2):
- Attach the IAM role to your EC2 instance
- No credential configuration needed

**Assumed role via AWS SSO**:
```bash
aws sso login --profile my-sso-profile
spawn launch --profile my-sso-profile
```

## Multi-Account Setup

### Named Profiles

```ini
# ~/.aws/credentials
[default]
aws_access_key_id = AKIA...
aws_secret_access_key = ...

[production]
aws_access_key_id = AKIA...
aws_secret_access_key = ...
region = us-west-2

[dev]
source_profile = default
role_arn = arn:aws:iam::123456789012:role/SpawnDevRole
```

Use with:
```bash
spawn launch --profile production
truffle ls --profile production
```

### AWS Organizations

For enterprise setups with AWS Organizations:

1. Create a central infrastructure account for S3, Lambda, Route53
2. Create workload accounts for EC2 instances
3. Configure cross-account IAM roles

```bash
# Assume cross-account role
spawn launch \
  --profile central \
  --assume-role arn:aws:iam::WORKLOAD_ACCOUNT:role/SpawnRole
```

See [Multi-Region](../features/multi-region.md) for full multi-account documentation.

## Temporary Credentials (STS)

```bash
# Get temporary credentials for 1 hour
aws sts assume-role \
  --role-arn arn:aws:iam::123456789012:role/MyRole \
  --role-session-name spawn-session \
  --duration-seconds 3600

# Use with spawn
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...
spawn launch
```

## Troubleshooting Auth Errors

**`NoCredentialProviders`**: No credentials found. Run `aws configure` or set environment variables.

**`AccessDenied`**: Insufficient permissions. Check IAM policy has required actions.

**`ExpiredTokenException`**: Temporary credentials expired. Re-authenticate.

**`InvalidClientTokenId`**: Invalid access key. Verify key ID is correct.

Run `aws sts get-caller-identity` to verify your credentials are working.
