# Deployment Guide for Organizations

This guide is for Cloud Administrators deploying spore-host (truffle + spawn) across their organization.

---

## Deployment Strategy Overview

There are **three deployment models** depending on your organization's security requirements:

| Model | User Permissions | Friction | Security | Best For |
|-------|------------------|----------|----------|----------|
| **1. Pre-created IAM Role** | PowerUserAccess | Low | Good | Most organizations |
| **2. Custom IAM Policy** | Custom spawn policy | Medium | Best | High-security environments |
| **3. Admin Access** | AdministratorAccess | None | Poor | Small teams, dev environments |

**Recommendation:** Use Model 1 (Pre-created IAM Role) for most organizations.

---

## Model 1: Pre-created IAM Role (Recommended)

### Overview

- **Cloud Admin** runs a one-time setup script per AWS account
- **Developers** only need `PowerUserAccess` (AWS-managed policy)
- **spawn** automatically detects and uses the pre-created role

### Setup Steps

#### Step 1: Cloud Admin Creates IAM Role (Once per Account)

```bash
# As Cloud Admin with IAM permissions
./scripts/setup-spawnd-iam-role.sh [aws-profile]
```

This creates:
- IAM Role: `spawnd-instance-role`
- Instance Profile: `spawnd-instance-profile`
- Permissions Policy: Allows spawnd to read tags and self-terminate

**Time:** 5 seconds
**Idempotent:** Can be run multiple times safely

#### Step 2: Grant Developers PowerUserAccess

Attach the AWS-managed `PowerUserAccess` policy to developer IAM users/groups:

```bash
# Option A: Attach to user
aws iam attach-user-policy \
  --user-name developer-name \
  --policy-arn arn:aws:iam::aws:policy/PowerUserAccess

# Option B: Attach to group
aws iam attach-group-policy \
  --group-name developers \
  --policy-arn arn:aws:iam::aws:policy/PowerUserAccess
```

#### Step 3: Developers Use spawn

Developers can now use spawn with no additional setup:

```bash
# No IAM permissions needed - role already exists
spawn launch --instance-type t3.micro --region us-east-1 --ttl 8h
```

### Multi-Account Setup

For organizations with multiple AWS accounts:

```bash
# Create role in all accounts
for profile in prod staging dev; do
  ./scripts/setup-spawnd-iam-role.sh $profile
done
```

**Or use CloudFormation StackSets** (see Model 1a below)

### Benefits

✅ **Zero friction for developers** - No custom policies needed
✅ **Uses AWS-managed policy** - PowerUserAccess is standard
✅ **Centrally controlled** - Cloud Admin manages the role
✅ **One-time setup** - Never needs updating

### Drawbacks

⚠️ Requires Cloud Admin with IAM permissions
⚠️ One-time coordination per account

---

## Model 1a: CloudFormation StackSets (Multi-Account)

For AWS Organizations with multiple accounts, deploy the IAM role automatically:

### CloudFormation Template

Save as `spawnd-iam-role.yaml`:

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: 'spawnd IAM Role for EC2 instance self-management'

Resources:
  SpawndRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: spawnd-instance-role
      Description: IAM role for spawnd daemon on EC2 instances
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              Service: ec2.amazonaws.com
            Action: 'sts:AssumeRole'
      Policies:
        - PolicyName: spawnd-policy
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Action:
                  - 'ec2:DescribeTags'
                  - 'ec2:DescribeInstances'
                Resource: '*'
              - Effect: Allow
                Action:
                  - 'ec2:TerminateInstances'
                  - 'ec2:StopInstances'
                Resource: '*'
                Condition:
                  StringEquals:
                    'ec2:ResourceTag/spawn:managed': 'true'
      Tags:
        - Key: spawn:managed
          Value: 'true'

  SpawndInstanceProfile:
    Type: AWS::IAM::InstanceProfile
    Properties:
      InstanceProfileName: spawnd-instance-profile
      Roles:
        - !Ref SpawndRole

Outputs:
  RoleArn:
    Description: ARN of the spawnd IAM role
    Value: !GetAtt SpawndRole.Arn
  InstanceProfileArn:
    Description: ARN of the spawnd instance profile
    Value: !GetAtt SpawndInstanceProfile.Arn
```

### Deploy to All Accounts

```bash
# Deploy to all accounts in organization
aws cloudformation create-stack-set \
  --stack-set-name spawnd-iam-role \
  --template-body file://spawnd-iam-role.yaml \
  --capabilities CAPABILITY_NAMED_IAM \
  --permission-model SERVICE_MANAGED \
  --auto-deployment Enabled=true,RetainStacksOnAccountRemoval=false

# Deploy to all accounts in all OUs
aws cloudformation create-stack-instances \
  --stack-set-name spawnd-iam-role \
  --deployment-targets OrganizationalUnitIds=ou-xxxx-yyyyyyyy \
  --regions us-east-1
```

### Benefits

✅ **Fully automated** across all accounts
✅ **New accounts get role automatically**
✅ **Consistent policy** everywhere
✅ **Version controlled** in CloudFormation

---

## Model 2: Custom IAM Policy (High Security)

For organizations that require users to explicitly have IAM permissions.

### Setup Steps

#### Step 1: Create Custom IAM Policy

See `spawn/IAM_PERMISSIONS.md` for the complete policy.

Create a company-managed policy: `YourCompany-SpawnAccess`

#### Step 2: Attach to Users/Groups

```bash
aws iam attach-user-policy \
  --user-name developer \
  --policy-arn arn:aws:iam::123456789012:policy/YourCompany-SpawnAccess
```

#### Step 3: Developers Use spawn

First launch creates the IAM role (takes ~10 seconds for propagation):

```bash
spawn launch --instance-type t3.micro --region us-east-1 --ttl 8h
# First launch: Creates spawnd-instance-role (10s wait)
# Subsequent launches: Uses existing role (instant)
```

### Benefits

✅ **Least privilege** - Users only get spawn-specific permissions
✅ **No pre-setup needed** - Role created on first use
✅ **Audit trail** - CloudTrail shows who created the role

### Drawbacks

⚠️ Custom policy requires creation and maintenance
⚠️ Users need IAM permissions (some orgs don't allow this)
⚠️ First launch has 10-second wait for IAM propagation

---

## Model 3: AdministratorAccess (Not Recommended)

Only for small teams or development environments.

### Setup

Attach AWS-managed `AdministratorAccess` policy to users.

### Benefits

✅ Zero friction
✅ No custom policies or setup

### Drawbacks

❌ **Excessive permissions** - Users can delete production resources
❌ **Security risk** - Full access to everything
❌ **Compliance issues** - Violates least privilege principle

**Only use in sandbox/dev accounts.**

---

## Comparison Matrix

| Feature | Model 1: Pre-created | Model 2: Custom Policy | Model 3: Admin |
|---------|---------------------|------------------------|----------------|
| **User Permissions** | PowerUserAccess | Custom spawn policy | AdministratorAccess |
| **Setup Effort** | Low (one-time script) | Medium (create policy) | None |
| **First Launch Wait** | None | 10 seconds (IAM propagation) | None |
| **Security** | Good | Best | Poor |
| **Friction** | Very low | Medium | None |
| **Multi-Account** | StackSets or script | Policy in each account | Not recommended |
| **Audit Complexity** | Low | Medium | High |
| **IAM Permissions Required** | Only Cloud Admin | All users | All users |

---

## Permission Validation

Before deploying, validate user permissions:

```bash
# Test user's current permissions
./scripts/validate-permissions.sh [aws-profile]
```

This checks:
- EC2 permissions (launch, describe, etc.)
- IAM permissions (if using Model 2)
- SSM Parameter Store access

---

## Cost Controls

Regardless of deployment model, implement these controls:

### 1. AWS Budgets

Alert on spawn-tagged resource spend:

```bash
aws budgets create-budget \
  --account-id 123456789012 \
  --budget file://spawn-budget.json
```

```json
{
  "BudgetName": "spawn-monthly-limit",
  "BudgetLimit": {
    "Amount": "1000",
    "Unit": "USD"
  },
  "TimeUnit": "MONTHLY",
  "BudgetType": "COST",
  "CostFilters": {
    "TagKeyValue": ["user:spawn:managed$true"]
  }
}
```

### 2. Service Control Policies (Optional)

Restrict instance types for spawn users:

```json
{
  "Effect": "Deny",
  "Action": "ec2:RunInstances",
  "Resource": "arn:aws:ec2:*:*:instance/*",
  "Condition": {
    "StringNotLike": {
      "ec2:InstanceType": [
        "t3.*",
        "t4g.*",
        "m7i.large",
        "m7i.xlarge"
      ]
    }
  }
}
```

### 3. Mandatory TTL (Optional)

Require all spawn launches to have TTL:

```json
{
  "Effect": "Deny",
  "Action": "ec2:RunInstances",
  "Resource": "arn:aws:ec2:*:*:instance/*",
  "Condition": {
    "StringNotEquals": {
      "aws:RequestTag/spawn:ttl": "*"
    }
  }
}
```

---

## Monitoring and Audit

### CloudTrail Queries

Find all spawn-launched instances:

```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance \
  --query 'Events[?contains(CloudTrailEvent, `spawn:managed`)]'
```

Find instances terminated by spawnd:

```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=TerminateInstances \
  --query 'Events[?contains(CloudTrailEvent, `spawnd-instance-role`)]'
```

### Cost Allocation Tags

Enable these tags for cost tracking:

```bash
aws ce update-cost-allocation-tags-status \
  --cost-allocation-tags-status \
    TagKey=spawn:managed,Status=Active \
    TagKey=spawn:created-by,Status=Active \
    TagKey=spawn:ttl,Status=Active
```

---

## Troubleshooting

### "User is not authorized to perform: iam:CreateRole"

**Solution:** Use Model 1 (pre-created role). User doesn't need IAM permissions.

### "Invalid IAM Instance Profile name"

**Cause:** IAM propagation delay
**Solution:** This is fixed in latest spawn version (10-second wait after role creation)

### spawnd not terminating instances

**Check:**
1. Does instance have IAM instance profile attached?
   ```bash
   aws ec2 describe-instances --instance-ids i-xxxxx \
     --query 'Reservations[0].Instances[0].IamInstanceProfile'
   ```

2. Are tags readable by spawnd?
   ```bash
   ssh ec2-user@<instance> 'sudo cat /var/log/spawnd.log | grep Config'
   # Should show: Config: TTL=8h0m0s, IdleTimeout=0s, Hibernate=false
   ```

---

## Migration from Existing Setup

### From Manual IAM Policy → Pre-created Role

1. Cloud Admin runs: `./scripts/setup-spawnd-iam-role.sh`
2. Remove spawn IAM permissions from user policies
3. Ensure users have PowerUserAccess
4. No code changes needed

### From AdministratorAccess → Pre-created Role

1. Cloud Admin runs: `./scripts/setup-spawnd-iam-role.sh`
2. Replace AdministratorAccess with PowerUserAccess for spawn users
3. Test with a single user first
4. Roll out to all users

---

## Security Considerations

See `SECURITY.md` for comprehensive security documentation.

**Key Points:**
- spawnd role can only terminate instances tagged `spawn:managed=true`
- Users cannot escalate privileges via the spawnd role
- All operations logged in CloudTrail
- SHA256 verification for spawnd binary distribution

---

## Support and Documentation

- **Full IAM Policy:** `spawn/IAM_PERMISSIONS.md`
- **Security Guide:** `SECURITY.md`
- **User Guide:** `spawn/README.md`
- **Validation Script:** `scripts/validate-permissions.sh`
- **Setup Script:** `scripts/setup-spawnd-iam-role.sh`

---

**Recommended:** Model 1 (Pre-created IAM Role) for most organizations - best balance of security and usability.
