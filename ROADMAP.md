# spore-host Roadmap

Feature ideas and planned enhancements for the spore-host project (truffle, spawn, spawnd).

## In Progress / Next Up

### Phase 1: Essential Instance Management (Top Priority)

1. **spawn list** - List all active spawn instances across regions
   - Show instance ID, type, region, age, TTL remaining, cost
   - Filter by region, tag, state

2. **spawn connect/ssh** (aliases) - Quick SSH with automatic key resolution
   - Auto-detect private key from instance metadata
   - Support for session manager if no SSH key available
   - Handle bastion hosts automatically

3. **spawn extend** - Extend TTL for running instances
   - `spawn extend <instance-id> <time>` - e.g., `spawn extend i-123 2h`
   - Update EC2 tags and notify spawnd

4. **spawn stop/hibernate/start** - Instance state management
   - `spawn stop <instance-id>` - Stop instance (EBS-backed only)
   - `spawn hibernate <instance-id>` - Hibernate instance (preserve RAM)
   - `spawn start <instance-id>` - Start stopped/hibernated instance
   - Update TTL countdown to pause during stop/hibernate

5. **spawnd idle detection** - Auto-terminate if idle
   - Monitor disk I/O activity
   - Monitor GPU utilization (nvidia-smi)
   - Monitor user activity (logged in users, keyboard/mouse via w, last)
   - Configurable idle threshold and grace period
   - Send warning before termination

### Phase 2: Developer Experience

6. **Configuration files** - `.spawn.yaml` for project defaults
   - Project-specific instance types, AMIs, user data
   - Shared team configurations
   - Environment variable injection

7. **Automatic DNS** - Human-readable hostnames for instances
   - Domain: **spore.host** (spore needs a host to grow!)
   - Auto-create DNS records on launch: `my-instance.spore.host`
   - Auto-update on IP changes (stop/start)
   - Auto-delete on termination
   - **Security Model**: Cross-account IAM roles
     - Hosted zone in centralized account (with spawnd S3)
     - Scoped trust policy: Only spawn-managed instances
     - Minimal permissions: Only Route53 updates for spawn.dev zone
     - Temporary credentials via AssumeRole (no long-lived secrets)
     - Resource name validation: Instance can only update its own record
     - Full CloudTrail audit trail
   - **Configuration options**:
     - Public hosted zone (easy option): Records visible publicly
     - Private hosted zone (secure option): Only accessible from VPC/VPN
     - Opt-in via `--dns` flag or config file
     - Configurable TTL (default: 60s for quick updates)
   - `spawn dns list/update/delete` commands for management

8. **truffle spot interruption history** - Historical interruption rates
   - Show interruption frequency per instance type/region
   - Recommend least-interrupted alternatives
   - Integration with AWS Data Exchange or historical data

9. **VS Code extension** - Launch and connect from VS Code
   - Launch instances from command palette
   - Auto-connect via Remote-SSH
   - Show running instances in sidebar
   - One-click extend/stop/terminate

## Backlog

### Instance Management & Lifecycle

- **spawn logs** - View spawnd daemon logs from the instance
- **Instance profiles** - Save common configurations (`spawn --profile ml-dev`)
- **Scheduled termination** - Terminate at specific time (EOD, weekend)
- **spawn clone** - Clone instance configuration to new instance

### Cost Optimization

- **Cost alerts** (requires budgets feature) - Warn when spending exceeds threshold
- **Budget management** - Set per-user, per-project, per-team budgets
- **Spot interruption history** - Historical interruption rates per instance type
- **Cost comparison** - On-Demand vs Spot vs Savings Plans pricing
- **Regional arbitrage** - Find cheapest region for your instance type
- **Capacity insights** - Show which instance types have best availability
- **forecast tool** - Predict monthly costs based on usage patterns
- **Cost trends** - Show cost trends over time
- **Cost anomaly detection** - Alert on unusual spending

### Multi-Region & Multi-Account

- **spawn --region all** - Try multiple regions until successful launch
- **spawn --fallback** - Automatic fallback to alternative instance types
- **Multi-account support** - Manage instances across AWS accounts
- **Cross-region cloning** - Copy instance to different region

### Team & Collaboration

- **Shared instance pool** - Team members can see/manage shared instances
- **Resource tagging** - Tag instances by project/team/environment
- **Usage quotas** - Per-user or per-team spending limits
- **Audit trail** - Who launched what, when, and for how long
- **Instance sharing** - Share access to running instances with team members

### truffle Enhancements

- **truffle recommend** - ML model to recommend instance type based on workload
- **truffle compare** - Side-by-side comparison of instance specs
- **GPU filtering** - Better GPU instance search (CUDA cores, VRAM, etc.)
- **truffle interactive** - Interactive exploratory mode
- **Saved searches** - Save common truffle queries
- **Instance bookmarks** - Quick access to frequently used configurations

### spawn Enhancements

- **Environment templates** - Pre-configured environments (Python ML, Node.js, Rust)
- **Volume attachment** - Automatically attach/create EBS volumes
- **Snapshot support** - Save instance state as AMI for later reuse
- **Container support** - Launch with Docker/containerd pre-configured
- **Cluster mode** - Launch multiple related instances (e.g., k8s cluster)
- **Data persistence** - Automatic EFS/S3 mounting
- **Snapshot scheduling** - Auto-snapshot at intervals
- **Warm pools** - Keep pre-configured instances ready for instant use

### Integration & Automation

- **Terraform provider** - Use spawn in IaC workflows
- **GitHub Actions** - CI/CD integration for ephemeral test environments
- **Slack/Discord bot** - Launch instances from chat
- **API/gRPC server** - Programmatic access to spawn functionality
- **Webhooks** - Notify external systems on instance events

### Monitoring & Observability

**New tool: observe**
- Real-time instance metrics (CPU, memory, network, disk, GPU)
- Cost tracking dashboard
- Spot interruption notifications
- Instance health checks
- Cost calendar - Visual calendar showing costs per day

### Security & Compliance

- **IMDSv2 enforcement** - Require IMDSv2 for all instances
- **Security group templates** - Pre-approved security configurations
- **Compliance tagging** - Automatic compliance-related tags
- **Key rotation** - Automatic SSH key rotation
- **Audit exports** - Export usage logs for compliance
- **Secrets management** - Integration with AWS Secrets Manager/Parameter Store

### Quality of Life

- **Shell completions** - Bash/Zsh/Fish completions for all commands
- **Better error messages** - Actionable error messages with suggestions
- **Dry-run mode** - Preview what would happen without executing
- **JSON output** - Machine-readable output for scripting
- **Logging levels** - Configurable verbosity
- **Progress indicators** - Show progress for long-running operations

## Recently Completed

- ✅ **truffle** - EC2 instance type search with Spot pricing
- ✅ **spawn** - Ephemeral instance launcher with TTL auto-termination
- ✅ **spawnd** - Automatic instance monitoring and cleanup daemon
- ✅ **IAM role auto-creation** - Automatic spawnd IAM role setup
- ✅ **SSH key fingerprint matching** - Reuse existing AWS keys
- ✅ **S3 binary distribution** - spawnd binaries with SHA256 verification
- ✅ **Homebrew tap** - Easy installation on macOS/Linux
- ✅ **Scoop bucket** - Easy installation on Windows
- ✅ **Pre-setup workflow** - Support for pre-created IAM roles
- ✅ **Comprehensive documentation** - Security, deployment, IAM guides

## User Feedback

- [ ] `spawn connect` should be an alias for `spawn ssh`
- [ ] Idle detection should include disk and GPU activity, not just CPU/network
- [ ] Idle detection should check for logged in users, keyboard activity
- [ ] Cost alerts require budget feature (defer to later phase)

## Decision Log

### Instance State Management
- Support stop (preserve EBS), hibernate (preserve RAM), and start
- TTL countdown should pause during stop/hibernate
- spawnd should handle state transitions gracefully

### Idle Detection Metrics
- CPU utilization (existing)
- Network I/O (existing)
- Disk I/O (new)
- GPU utilization via nvidia-smi (new)
- User activity via `w`, `last`, `/var/log/wtmp` (new)
- Configurable thresholds and grace periods

### Command Aliases
- `spawn ssh` = `spawn connect` (both should work)
- Consider other common aliases based on user feedback

---

**Last Updated:** 2025-12-21
