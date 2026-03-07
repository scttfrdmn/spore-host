# Security Overview for spore-host (truffle + spawn)

**Audience:** CISOs, Cloud Administrators, Security Engineers

**Purpose:** Comprehensive security assessment and operational guidance for deploying spore-host tools in enterprise AWS environments.

---

## Executive Summary

**spore-host** is a command-line toolset for AWS EC2 instance discovery and ephemeral compute provisioning. It consists of two components:

- **truffle**: EC2 instance type search tool (read-only)
- **spawn**: EC2 instance launcher with auto-termination (read/write)

**Security Model:** Least-privilege IAM permissions, explicit resource tagging, and automatic cleanup of ephemeral resources.

**Risk Profile:** Medium - requires EC2 launch permissions and limited IAM role creation

**Compliance:** Supports AWS best practices, CloudTrail auditing, and resource tagging for cost allocation

---

## 1. Architecture Overview

### truffle (Read-Only Tool)

**Purpose:** Search and compare EC2 instance types across regions

**AWS Services Used:**
- EC2 DescribeInstanceTypes (read-only)
- EC2 DescribeRegions (read-only)
- EC2 DescribeAvailabilityZones (read-only)

**Security Posture:** Zero write permissions, no resource creation

**Data Access:** Only AWS service metadata (pricing, specifications)

**Risk Assessment:** **LOW** - Cannot modify infrastructure

---

### spawn (Instance Launcher)

**Purpose:** Launch ephemeral EC2 instances with automatic termination

**AWS Services Used:**
- **EC2:** Launch, terminate, and query instances
- **IAM:** Create service role for spored agent (once per account)
- **SSM Parameter Store:** Query latest Amazon Linux AMI IDs
- **S3:** Download spored agent binary (public read, SHA256-verified)

**Security Posture:** Requires EC2 launch permissions and limited IAM role creation

**Resource Lifecycle:** Automatic cleanup via TTL and idle detection

**Risk Assessment:** **MEDIUM** - Can launch instances (cost implications)

---

## 2. IAM Permissions Required

### For truffle (Read-Only)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstanceTypes",
        "ec2:DescribeRegions",
        "ec2:DescribeAvailabilityZones"
      ],
      "Resource": "*"
    }
  ]
}
```

**Compatible with:** ReadOnlyAccess, ViewOnlyAccess, PowerUserAccess

---

### For spawn (Minimum Required)

See `spawn/IAM_PERMISSIONS.md` for complete policy.

**Summary:**
- **EC2:** Launch, terminate, describe instances (standard compute access)
- **IAM:** Create `spored-instance-role` and `spored-instance-profile` (one-time setup)
- **SSM:** Read `/aws/service/ami-amazon-linux-latest/*` (AMI auto-detection)

**Compatible with:** PowerUserAccess + spawn-specific IAM permissions (see below)

**NOT Compatible with:** PowerUserAccess alone (missing IAM permissions)

---

### PowerUser + spawn IAM Policy

For organizations using AWS `PowerUser` managed policy, add this supplementary policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SpawnIAMRoleManagement",
      "Effect": "Allow",
      "Action": [
        "iam:CreateRole",
        "iam:GetRole",
        "iam:PutRolePolicy",
        "iam:CreateInstanceProfile",
        "iam:GetInstanceProfile",
        "iam:AddRoleToInstanceProfile",
        "iam:PassRole"
      ],
      "Resource": [
        "arn:aws:iam::*:role/spored-instance-role",
        "arn:aws:iam::*:instance-profile/spored-instance-profile"
      ]
    },
    {
      "Sid": "SpawnIAMTagging",
      "Effect": "Allow",
      "Action": [
        "iam:TagRole",
        "iam:TagInstanceProfile"
      ],
      "Resource": [
        "arn:aws:iam::*:role/spored-instance-role",
        "arn:aws:iam::*:instance-profile/spored-instance-profile"
      ]
    }
  ]
}
```

**Why:** PowerUser excludes all IAM permissions. This grants minimal IAM access scoped to spawn-specific resources only.

---

## 3. IAM Role Created by spawn

### spored-instance-role

**Purpose:** Allows spored agent (running on EC2 instances) to self-manage based on tags

**Trust Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

**Permissions Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeTags",
        "ec2:DescribeInstances"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:TerminateInstances",
        "ec2:StopInstances"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ec2:ResourceTag/spawn:managed": "true"
        }
      }
    }
  ]
}
```

**Security Controls:**
- ✅ Can only terminate/stop instances tagged `spawn:managed=true`
- ✅ Cannot terminate instances launched by other tools
- ✅ Cannot modify IAM policies
- ✅ Cannot access S3, databases, or other AWS services
- ✅ Scoped to EC2 instance lifecycle only

---

## 4. Security Features

### Automatic Resource Cleanup

**TTL (Time-To-Live):**
- User specifies maximum instance lifetime (e.g., `--ttl 8h`)
- spored agent monitors uptime
- Terminates instance when TTL expires
- **Benefit:** Prevents forgotten instances running indefinitely

**Idle Detection:**
- User specifies idle timeout (e.g., `--idle-timeout 30m`)
- spored monitors SSH sessions and CPU utilization
- Terminates or hibernates when idle threshold reached
- **Benefit:** Reduces cost for intermittent workloads

**Laptop Independence:**
- spored runs as systemd service on the instance (not laptop)
- Auto-termination works even if laptop is off or disconnected
- **Benefit:** Prevents orphaned resources from VPN disconnections

---

### Resource Tagging

All instances launched by spawn are tagged:

```
spawn:managed = true
spawn:root = true
spawn:created-by = spawn
spawn:version = 0.1.0
spawn:ttl = <duration>           (if specified)
spawn:idle-timeout = <duration>  (if specified)
```

**Benefits:**
- Cost allocation and chargeback
- Automated cleanup policies (AWS Config, Lambda)
- Compliance reporting (which instances have TTL?)
- Security auditing (CloudTrail queries by tag)

---

### Binary Distribution Security

**Problem:** How to securely distribute spored binary to instances?

**Solution:** Public S3 with SHA256 verification (industry standard)

**Implementation:**
1. spored binaries stored in public S3 buckets (one per region)
2. Each binary has corresponding `.sha256` checksum file
3. Instance user-data downloads both binary and checksum
4. SHA256 verified before execution: `sha256sum --check spored-linux-amd64.sha256`
5. Installation fails if checksum mismatch

**Why Public S3?**
- Same model as apt/yum/pip repositories
- No AWS credentials in user-data (reduces attack surface)
- Fast downloads (regional buckets, <20ms latency)
- Tamper detection via cryptographic checksums

**Threat Model:**
- ❌ **Compromised S3 Bucket:** Attacker cannot upload malicious binary without SHA256 key
- ❌ **Man-in-the-Middle:** HTTPS + SHA256 verification prevents tampering
- ✅ **Authorized Updates:** Only spawn maintainers can update binaries (S3 bucket policy)

**S3 Bucket Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadSpawndBinaries",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::spawn-binaries-*/spored-*"
    }
  ]
}
```

---

### SSH Key Management

**Fingerprint-Based Key Reuse:**
- spawn calculates MD5 fingerprint of local SSH public key
- Searches AWS EC2 for existing key pair with matching fingerprint
- Reuses existing key if found (no duplicate uploads)
- Only uploads if no matching key exists

**Benefits:**
- No duplicate keys cluttering AWS account
- Consistent key usage across multiple launches
- User can pre-import keys with custom names

**Security:**
- Only public keys handled (private keys never leave laptop)
- Keys stored in AWS EC2 KeyPairs (standard AWS service)
- No custom key storage or management

---

## 5. Audit and Compliance

### CloudTrail Events

All spawn operations generate CloudTrail events:

**IAM Role Creation:**
```
CreateRole (spored-instance-role)
PutRolePolicy (spored-policy)
CreateInstanceProfile (spored-instance-profile)
AddRoleToInstanceProfile
```

**Instance Launch:**
```
RunInstances
  - Tags: spawn:managed=true, spawn:ttl=8h
  - IamInstanceProfile: spored-instance-profile
  - UserData: <base64-encoded script>
```

**Instance Termination (by spored):**
```
TerminateInstances
  - InitiatedBy: spored (via instance profile role)
  - Reason: TTL expired / idle timeout
```

**Query Examples:**

Find all spawn-launched instances:
```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance \
  --query 'Events[?contains(CloudTrailEvent, `spawn:managed`)]'
```

Find instances terminated by spored:
```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=TerminateInstances \
  --query 'Events[?contains(CloudTrailEvent, `spored-instance-role`)]'
```

---

### Cost Control

**Built-In Mechanisms:**
1. **TTL Enforcement:** Hard limit on instance runtime
2. **Idle Detection:** Automatic termination of unused instances
3. **Resource Tagging:** Cost allocation via `spawn:*` tags
4. **Spot Instance Support:** `--spot` flag for 70% savings

**Recommended Controls:**
1. **AWS Budgets:** Alert on `spawn:managed=true` tag spend
2. **Service Control Policies:** Limit instance types per OU
3. **IAM Conditions:** Require `--ttl` flag via policy conditions
4. **Cost Anomaly Detection:** Alert on unusual spawn usage patterns

**Example Budget:**
```bash
aws budgets create-budget \
  --account-id 123456789012 \
  --budget file://spawn-budget.json
```

```json
{
  "BudgetName": "spawn-instances-monthly",
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

---

### Compliance Considerations

**GDPR / Data Residency:**
- spawn respects regional boundaries (no cross-region data transfer)
- Users control region via `--region` flag
- AMI selection automatic per region

**PCI-DSS / HIPAA:**
- spawn does not handle sensitive data
- Instances launched in user's VPC (user controls network security)
- Encryption at rest supported (EBS encryption via instance type requirements)

**SOC 2 / ISO 27001:**
- Audit trail via CloudTrail
- Resource tagging for inventory management
- Automatic cleanup reduces security surface area

---

## 6. Risk Assessment

### Identified Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Excessive instance launches | Medium | AWS Budgets, SCPs, `--ttl` requirement |
| IAM role privilege escalation | Low | Role scoped to `spawn:managed=true` only |
| Binary tampering | Low | SHA256 verification, HTTPS |
| Forgotten instances | Low | TTL enforcement, idle detection |
| Insider threat (malicious user) | Medium | CloudTrail auditing, IAM least privilege |
| Credential exposure | Medium | No credentials in user-data, IMDSv2 supported |

### Mitigations Summary

✅ **Least Privilege IAM:** Users only get permissions they need
✅ **Resource Tagging:** All resources identifiable and auditable
✅ **Automatic Cleanup:** TTL/idle detection prevents resource leaks
✅ **Audit Trail:** CloudTrail captures all operations
✅ **Cost Controls:** Budgets, SCPs, Spot instances
✅ **Binary Integrity:** SHA256 verification
✅ **Network Security:** User controls VPC, security groups, subnets

---

## 7. Deployment Recommendations

### For Small Teams (< 50 users)

1. **IAM Users with spawn policy** (see `IAM_PERMISSIONS.md`)
2. **AWS Budget alert** at $500/month
3. **CloudTrail enabled** (standard AWS best practice)
4. **Require `--ttl` via documentation** (not enforced)

### For Enterprises (> 50 users)

1. **IAM Groups:**
   - `spawn-power-users`: Full spawn access
   - `spawn-basic-users`: spawn + instance type restrictions via SCP

2. **Service Control Policies:**
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
       },
       "StringEquals": {
         "aws:PrincipalTag/spawn-user": "true"
       }
     }
   }
   ```

3. **Mandatory TTL via IAM Policy:**
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
   *(Note: Requires spawn to always set TTL tag)*

4. **Cost Allocation Tags:**
   - Enable `spawn:*` tags as cost allocation tags
   - Monthly reports by team/project

5. **Security Hub Integration:**
   - Custom AWS Config rule: "Instances must have `spawn:ttl` tag"
   - Alert on non-compliant spawn instances

---

## 8. Incident Response

### Scenario: Unauthorized Instance Launch

**Detection:**
```bash
# Find all spawn instances
aws ec2 describe-instances \
  --filters "Name=tag:spawn:managed,Values=true" \
  --query 'Reservations[].Instances[].[InstanceId,LaunchTime,State.Name]'

# Check CloudTrail for who launched
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance
```

**Response:**
1. Identify user via CloudTrail `userIdentity`
2. Terminate instance: `aws ec2 terminate-instances --instance-ids i-xxxxx`
3. Review user IAM permissions
4. Check AWS Budget impact

---

### Scenario: spored Role Privilege Escalation Attempt

**Detection:** CloudWatch Logs Insights query:

```
fields @timestamp, errorCode, errorMessage
| filter eventSource = "ec2.amazonaws.com"
| filter userIdentity.principalId like /spored-instance-role/
| filter errorCode like /Unauthorized/
```

**Response:**
1. Identify instance: Extract instance ID from CloudTrail
2. Terminate instance immediately
3. Review spored role policy (should be read-only except for self-termination)
4. Check for IAM policy modifications

---

## 9. Validation and Testing

### Pre-Deployment Checklist

- [ ] Run `./scripts/validate-permissions.sh <aws-profile>` for each user
- [ ] Create test AWS Budget for spawn resources
- [ ] Enable CloudTrail in all regions
- [ ] Test IAM role creation (first spawn launch)
- [ ] Verify TTL enforcement (launch instance with `--ttl 5m`)
- [ ] Verify idle detection (launch instance with `--idle-timeout 10m`)
- [ ] Test Spot instances (`--spot` flag)
- [ ] Review CloudTrail events for spawn operations

### Security Testing

```bash
# Test 1: Verify spored cannot terminate non-spawn instances
aws ec2 run-instances --image-id ami-xxxxx --instance-type t3.micro
# (manually launched, no spawn:managed tag)
# spored should fail to terminate this

# Test 2: Verify TTL enforcement
echo '[{"instance_type":"t3.micro","region":"us-east-1"}]' | \
  spawn launch --ttl 5m
# Wait 6 minutes, verify instance terminated

# Test 3: Verify SHA256 verification
# Corrupt spored binary on S3, verify instance launch fails

# Test 4: Verify CloudTrail logging
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=RunInstances
```

---

## 10. Code-Level Security Hardening (v0.13.0)

### Command Injection Protection

spawn implements comprehensive protection against shell command injection attacks:

**Security Package** (`spawn/pkg/security`):
- `ShellEscape()`: Properly escapes strings for POSIX shell arguments
- `ValidateUsername()`: Validates usernames against safe patterns
- `ValidateCommand()`: Detects potentially dangerous shell characters

**Protected Operations**:
1. **SSH Commands** (`spawn/cmd/config.go`):
   - Config keys and values properly escaped before SSH execution
   - Prevents injection via `spawn config set "key" "value; rm -rf /"`

2. **User Data Scripts** (`spawn/cmd/launch.go`):
   - Usernames validated against regex: `^[a-z][a-z0-9_-]{0,31}$`
   - SSH keys validated as proper base64
   - All dynamic values shell-escaped in bash scripts

3. **MPI Templates** (`spawn/pkg/userdata/mpi.go`):
   - Custom `shellEscape` template function
   - MPI commands properly escaped: `{{.MPICommand | shellEscape}}`

4. **Storage Mounting** (`spawn/pkg/userdata/storage.go`):
   - Mount paths validated and escaped
   - Filesystem DNS names escaped in mount commands

**Test Coverage**: 78.4% with attack pattern fuzzing

---

### Path Traversal Protection

All file path operations are validated to prevent directory traversal attacks:

**Validation Functions**:
- `ValidatePathForReading()`: Blocks `../../etc/passwd` style attacks
- `ValidateMountPath()`: Restricts mounts to `/mnt`, `/data`, `/scratch`
- `SanitizePath()`: Removes traversal sequences for safe logging

**Blocked Paths**:
- `/etc/` - System configuration
- `/sys/` - System kernel interface
- `/proc/` - Process information
- `/root/` - Root home directory
- `/boot/` - Boot loader files
- `/dev/` - Device files
- `/var/lib/` - System libraries

**Protected Operations**:
1. **User Data File Reads** (`spawn/cmd/launch.go`):
   - Both `--user-data-file` and `--user-data @file` validated
   - Prevents reading system files

2. **Job Result Uploads** (`spawn/pkg/agent/queue_runner.go`):
   - Glob patterns validated before expansion
   - Prevents uploading sensitive files

---

### Credential Protection

Sensitive credentials are protected from exposure:

**Security Functions**:
- `MaskSecret()`: Masks secrets showing only first/last 4 chars
- `MaskURL()`: Shows scheme and domain, masks path
- `SanitizeForLog()`: Removes AWS keys from log messages
- `EncryptSecret()`/`DecryptSecret()`: KMS-based encryption (ready for use)

**Use Cases**:
- Webhook URLs in alert configurations
- API tokens in logs
- AWS access keys accidentally logged

---

### Audit Logging Infrastructure

Structured audit logging for security-sensitive operations:

**Audit Package** (`spawn/pkg/audit`):

**AuditLogger** - Structured JSON logging:
```go
logger := audit.NewLogger(os.Stderr, userID, correlationID)
logger.LogOperation("terminate_instances", sweepID, "success", nil)
```

**Context Propagation**:
```go
ctx = audit.NewContextWithAudit(ctx, userID)
logger := audit.NewLoggerFromContext(ctx)
```

**Audit Event Fields**:
- `timestamp`: UTC timestamp
- `level`: info/error
- `operation`: Action performed
- `user_id`: AWS account/user ID
- `instance_id`: Resource identifier
- `region`: AWS region
- `correlation_id`: Trace requests across systems
- `result`: success/failed/initiated
- `error`: Error message if failed
- `additional_data`: Structured metadata

**Ready for CloudWatch Logs Integration**:
- JSON format for CloudWatch Logs Insights
- Correlation IDs for distributed tracing
- User attribution for compliance

**Test Coverage**: 79.2%

---

### Security Testing

**Test Suites**:
1. **Attack Pattern Tests** (`pkg/security/shell_test.go`):
   - Command injection: `; rm -rf /`, `$(whoami)`, backticks
   - Path traversal: `../../etc/passwd`
   - Variable expansion: `${IFS}malicious`

2. **Path Validation Tests** (`pkg/security/path_test.go`):
   - System directory blocking
   - Mount point restrictions
   - Traversal detection

3. **Audit Logger Tests** (`pkg/audit/logger_test.go`):
   - JSON formatting
   - Context propagation
   - Error logging

**Coverage Targets**:
- Security package: 78.4% (target: 80%)
- Audit package: 79.2% (target: 80%)

---

### Vulnerability Reporting

**Security Contact**: scott@spore.host

**Please Report**:
- Command injection bypasses
- Path traversal attacks
- Credential exposure
- Privilege escalation
- Any security vulnerability

**Do NOT report via public GitHub issues**

Response time: 48 hours

---

## 11. Frequently Asked Questions (Security)

### Q: Can spawn access my existing EC2 instances?

**A:** No. spawn can only launch new instances and terminate instances tagged `spawn:managed=true`. It cannot access, modify, or terminate instances launched by other tools.

---

### Q: What data does spawn send to external services?

**A:** None. spawn only communicates with:
- AWS APIs (EC2, IAM, SSM)
- S3 buckets for spored binary download (public, read-only)

No telemetry, analytics, or user data leaves your AWS account.

---

### Q: Can spored access my S3 buckets or databases?

**A:** No. The `spored-instance-role` only has permissions for:
- Reading its own EC2 tags
- Terminating/stopping itself

It cannot access S3, RDS, DynamoDB, or any other AWS services.

---

### Q: What if someone modifies the spored binary on S3?

**A:** The instance will fail to launch. User-data verifies SHA256 checksum before executing spored. If the binary is tampered, the checksum won't match and installation aborts.

---

### Q: Can users extend or bypass TTL limits?

**A:** Users can extend the TTL of a running instance with `spawn extend <name> <duration>`. This updates the EC2 tag and spored picks up the new deadline live. They cannot remove the TTL entirely once set.

They could also:
- Manually terminate the instance and launch a new one (new TTL starts)
- SSH in and stop the spored service (instance won't auto-terminate)

**Mitigation:** CloudWatch Events rule to detect spored service failures.

---

### Q: How do I audit spawn usage?

**A:** Use CloudTrail:

```bash
# All spawn launches (last 7 days)
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance \
  --max-results 1000 \
  --query 'Events[?contains(CloudTrailEvent, `spawn:managed`)].{Time:EventTime,User:Username,Instance:Resources[0].ResourceName}'

# Cost by user (requires Cost Allocation Tags)
aws ce get-cost-and-usage \
  --time-period Start=2025-12-01,End=2025-12-31 \
  --granularity MONTHLY \
  --group-by Type=TAG,Key=spawn:created-by \
  --metrics BlendedCost
```

---

## 11. Contact and Support

**Security Issues:** Report to project maintainers (see CONTRIBUTING.md)

**Documentation:**
- Full IAM policy: `spawn/IAM_PERMISSIONS.md`
- User guide: `spawn/README.md`
- Validation script: `scripts/validate-permissions.sh`

**References:**
- [AWS IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)
- [AWS Well-Architected Security Pillar](https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/welcome.html)
- [CloudTrail Event Reference](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-event-reference.html)

---

**Document Version:** 1.0
**Last Updated:** December 21, 2025
**Next Review:** January 21, 2026
