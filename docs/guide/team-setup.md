# Team Setup

Set up spawn for a team with multiple users, shared infrastructure, and cost visibility.

## Architecture

For team use, a typical setup uses:
- One **infrastructure AWS account** for shared resources (Route53, Lambda, S3)
- One or more **workload accounts** for EC2 instances (per team or project)
- IAM roles for cross-account access
- Tagging conventions for cost allocation

---

## Step 1: Create AWS Accounts

### Option A: AWS Organizations (recommended for teams > 5)

1. Create a management account (or use existing)
2. Create member accounts via AWS Organizations:
   - `infrastructure` — shared Route53, Lambda, S3
   - `team-research` — EC2 for research team
   - `team-platform` — EC2 for platform team

### Option B: Single Account with IAM Boundaries

For smaller teams, use a single account with IAM permission boundaries per user:

```json
{
  "Effect": "Allow",
  "Action": ["ec2:RunInstances", "ec2:TerminateInstances"],
  "Resource": "*",
  "Condition": {
    "StringEquals": {
      "ec2:ResourceTag/Owner": "${aws:username}"
    }
  }
}
```

---

## Step 2: Create Team IAM Policy

Create a shared IAM policy for all spawn users:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SpawnLaunch",
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:DescribeInstances",
        "ec2:CreateTags",
        "ec2:DescribeImages",
        "ec2:DescribeSubnets",
        "ec2:DescribeVpcs"
      ],
      "Resource": "*"
    },
    {
      "Sid": "SpawnTerminate",
      "Effect": "Allow",
      "Action": ["ec2:TerminateInstances", "ec2:StopInstances"],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ec2:ResourceTag/SpawnTeam": "your-team-name"
        }
      }
    }
  ]
}
```

---

## Step 3: Configure Shared Tagging

Use consistent tags so costs are attributable:

```yaml
# ~/.spawn/config.yaml (per user)
defaults:
  tags:
    - team=research
    - owner=${USER}        # auto-populated from shell env
    - project=             # fill in per project
    - cost-center=eng-123
```

Or per-project `.spawn.yaml`:
```yaml
defaults:
  tags:
    - project=llm-training
    - team=research
    - cost-center=ml-ops
```

---

## Step 4: Set Up AWS SSO (Optional, Recommended)

For teams, AWS SSO provides centralized access management:

1. Enable AWS IAM Identity Center (SSO) in your management account
2. Create permission sets for spawn users
3. Assign to team members and accounts

Users authenticate with:
```bash
aws sso login --profile research-account
spawn launch --profile research-account
```

---

## Step 5: Configure spawn for Each Developer

Share a base config template:

```yaml
# ~/.spawn/config.yaml
region: us-east-1
profile: research-account

defaults:
  instance_type: t3.medium
  ttl: 8h
  idle_timeout: 30m
  tags:
    - team=research
    - owner=${USER}

cost:
  alert_threshold: 100.00
  alert_email: ${USER}@yourcompany.com
```

---

## Step 6: Monitor Team Costs

Set up a weekly cost review:

```bash
# Cost by team member (uses Owner tag)
truffle cost --group tag:owner --days 7

# Cost by project
truffle cost --group tag:project --days 30
```

Or use AWS Cost Explorer with cost allocation tags (enable tags as cost allocation tags in Billing console).

---

## Step 7: Create Shared Autoscale Groups (Optional)

For shared batch processing queues, deploy autoscale groups to shared accounts:

```bash
spawn autoscale deploy --config team-batch.yaml --profile research-account
```

All team members submit work to the shared SQS queue; the autoscale controller manages instances.
