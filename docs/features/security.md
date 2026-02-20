# Security

## Overview

spawn's security model relies on AWS IAM for authorization and follows the principle of least privilege. No credentials are stored in spawn's infrastructure.

## Authentication & Authorization

### User Authentication

spawn uses your AWS credentials (via `~/.aws/credentials`, environment variables, or IAM instance profiles) to call AWS APIs. There is no separate spawn authentication layer.

### Instance Authorization

Spawned instances use an IAM instance profile (`spawn-instance-role`) that grants only the permissions needed by spored:
- Write to spawn DynamoDB tables
- Call Route53 (DNS registration)
- Read S3 (spored binary)
- Write CloudWatch metrics

Instances do NOT have permissions to launch or terminate other instances.

## IAM Best Practices

### Least Privilege

Use separate IAM policies for different functions:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["ec2:RunInstances"],
    "Resource": "*",
    "Condition": {
      "StringEquals": {
        "ec2:InstanceType": ["t3.micro", "t3.small", "c7i.large"]
      }
    }
  }]
}
```

### Restrict by Tag

Control which instances a user can terminate:
```json
{
  "Effect": "Allow",
  "Action": ["ec2:TerminateInstances"],
  "Resource": "*",
  "Condition": {
    "StringEquals": {
      "ec2:ResourceTag/Owner": "${aws:username}"
    }
  }
}
```

## Network Security

Default security group allows:
- Port 22 (SSH) from `0.0.0.0/0`

For production use, restrict to specific IPs:
```bash
spawn launch \
  --allowed-cidrs "203.0.113.0/24,198.51.100.0/24"
```

Or use a pre-existing security group:
```bash
spawn launch --security-group sg-XXXXXXXXX
```

## Secrets Management

**Never** store secrets in:
- User-data scripts (visible in EC2 console)
- Environment variables passed at launch
- SSH session variables

**Do** use:
- AWS Secrets Manager
- AWS SSM Parameter Store
- IAM instance profiles

```bash
# Access secrets via IAM instance profile
aws secretsmanager get-secret-value --secret-id my-secret
aws ssm get-parameter --name /myapp/db-password --with-decryption
```

## Data Protection

- All EBS volumes use AES-256 encryption by default
- Data in transit protected by TLS (AWS APIs) and SSH
- spored never logs credentials or tokens

## Audit Logging

All AWS API calls are logged to CloudTrail. Enable CloudTrail in your account:

```bash
aws cloudtrail create-trail \
  --name spawn-audit \
  --s3-bucket-name my-cloudtrail-bucket \
  --is-multi-region-trail
```

## Compliance

For regulated workloads (HIPAA, PCI DSS):
- Enable EBS encryption at rest
- Enable CloudTrail logging
- Use VPC with private subnets
- Enable AWS Config rules for compliance monitoring
