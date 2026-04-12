# Lagotto Deployment Guide

This document describes how to deploy lagotto's infrastructure (DynamoDB, SNS, Lambda, EventBridge) to the `spore-host-infra` AWS account.

## Prerequisites

- AWS CLI configured with `spore-host-infra` profile (account 966362334030)
- Go 1.26+ for building the Lambda binary
- CloudFormation deploy permissions

## One-time Setup

### 1. Build the Lambda zip

From the repository root:

```bash
cd lagotto/lambda/capacity-poller
make build
# Produces: function.zip (containing bootstrap binary for linux/arm64)
```

### 2. Upload the Lambda zip to S3

```bash
VERSION=v0.32.0  # match the release tag
AWS_PROFILE=spore-host-infra aws s3 cp function.zip \
  s3://spawn-binaries-us-east-1/lagotto/capacity-poller-${VERSION}.zip
```

### 3. Deploy the CloudFormation stack

```bash
AWS_PROFILE=spore-host-infra aws cloudformation deploy \
  --stack-name lagotto \
  --template-file deployment/cloudformation/lagotto-stack.yaml \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides \
      Environment=production \
      LambdaCodeBucket=spawn-binaries-us-east-1 \
      LambdaCodeKey=lagotto/capacity-poller-${VERSION}.zip \
  --region us-east-1
```

Creates:
- DynamoDB tables: `lagotto-watches`, `lagotto-match-history` (PAY_PER_REQUEST, PITR, TTL)
- SNS topic: `lagotto-capacity-alerts` (KMS encrypted with `alias/aws/sns`)
- Lambda: `lagotto-capacity-poller` (arm64, provided.al2023, 512MB, 900s timeout)
- EventBridge Schedule: `lagotto-capacity-poller` at `rate(5 minutes)`, starts **DISABLED**
- IAM roles for Lambda execution + EventBridge invocation

The schedule starts disabled to avoid billing when no watches are active. The `lagotto watch` CLI enables it on first watch creation; the Lambda disables it when the last active watch is removed.

### 4. Verify deployment

```bash
AWS_PROFILE=spore-host-infra aws cloudformation describe-stacks \
  --stack-name lagotto \
  --query 'Stacks[0].Outputs'

# Smoke-test the Lambda
AWS_PROFILE=spore-host-infra aws lambda invoke \
  --function-name lagotto-capacity-poller \
  /tmp/lagotto-out.json
cat /tmp/lagotto-out.json
```

## Updating the Lambda (code-only)

When you only change Go code (no CloudFormation changes):

```bash
cd lagotto/lambda/capacity-poller
make build

AWS_PROFILE=spore-host-infra aws lambda update-function-code \
  --function-name lagotto-capacity-poller \
  --zip-file fileb://function.zip
```

## Running Integration Tests

Integration tests hit the real DynamoDB tables in `spore-host-infra`:

```bash
cd lagotto
AWS_PROFILE=spore-host-infra LAGOTTO_INTEGRATION_TEST=1 \
  go test -tags integration -v ./... -run TestIntegration
```

Tests clean up after themselves (cancel their test watches). The `g7e.xlarge` watch may or may not find capacity — both outcomes are valid.

## Teardown

```bash
AWS_PROFILE=spore-host-infra aws cloudformation delete-stack --stack-name lagotto --region us-east-1
```

DynamoDB tables will be deleted with the stack. Match history is lost (90-day TTL anyway).

## AWS Account Reference

From `spawn/CLAUDE.md`:

- **spore-host-infra** (966362334030): All lagotto infrastructure lives here — DynamoDB, SNS, Lambda, EventBridge.
- **spore-host-dev** (435415984226): EC2 only; not used by lagotto directly.

The Lambda does not assume a cross-account role — it only reads EC2 metadata (instance types, spot pricing) via the `ec2:Describe*` APIs in the infra account's context.
