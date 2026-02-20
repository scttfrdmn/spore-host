# Frequently Asked Questions

## General

### What is spawn?

spawn is a free, open-source CLI tool for launching ephemeral AWS EC2 instances. It auto-configures networking, SSH keys, AMI selection, and installs a self-monitoring agent (spored) that enforces auto-termination and detects idle instances.

### What is truffle?

truffle is the companion discovery tool. It queries running instances across regions and AWS accounts, providing listing, filtering, and cost reporting.

### What is spored?

spored is a small daemon that runs on each spawned EC2 instance. It enforces TTL, detects idle conditions, handles spot interruptions, registers DNS, and reports metrics — all without your laptop needing to stay connected.

### Is spawn free?

spawn and truffle are free, open-source software. You pay only for AWS resources: EC2 instance hours, storage, and data transfer. spawn helps minimize these costs through TTL enforcement and idle detection.

## AWS and Costs

### How much does it cost to run spawn?

The overhead cost of spawn's AWS infrastructure is approximately $1-6/month:
- Lambda functions: ~$0.30/month
- DynamoDB tables: ~$1-5/month
- S3 storage: minimal

EC2 instance costs depend on your usage. A t3.micro costs ~$0.01/hour.

### How does spawn prevent surprise bills?

- **TTL**: Every instance has a time-to-live. It terminates automatically when TTL expires.
- **Idle detection**: Terminates instances that have been idle (low CPU/memory/network) for a configurable period.
- **Cost alerts**: Alert you when monthly costs exceed a threshold.
- **Spot instances**: Up to 70% cheaper than on-demand.

### Can I use spawn with multiple AWS accounts?

Yes. Use `--profile` to specify different AWS credential profiles per launch. See [Authentication](authentication.md#multi-account) for multi-account setup.

### Will spawn incur charges in all my regions?

spawn only creates resources (EC2, DynamoDB) in the regions you use. By default it uses your configured default region.

## Instances

### What instance types does spawn support?

All EC2 instance types are supported. Common choices:
- **Dev/test**: t3.micro, t3.small (free-tier eligible)
- **General compute**: m7i.large, m7i.xlarge
- **CPU-intensive**: c7i.large, c7i.2xlarge
- **Memory-intensive**: r7i.large, r7i.2xlarge
- **GPU**: g5.xlarge, p3.2xlarge (ML/AI workloads)

### Can I use a custom AMI?

Yes:
```bash
spawn launch --ami ami-XXXXXXXXX
```

spawn installs the spored agent via user-data on launch.

### Can I keep an instance running longer than the TTL?

Yes. Extend the TTL:
```bash
spawn extend my-instance 4h
```

Or set a very long TTL at launch (e.g., `--ttl 720h` for 30 days).

### What happens when a spot instance gets interrupted?

spored monitors the EC2 instance metadata for spot interruption notices (2-minute warning). When detected, it runs cleanup commands (configurable), saves output to S3 if configured, and gracefully terminates.

### Can spawn launch Windows instances?

Yes, with `--ami <windows-ami-id>`. Note that Windows instances cannot use SSH; use RDP instead.

## SSH and Connectivity

### How do I get the SSH key?

spawn auto-creates a key pair named `spawn-default` on first use. The private key is at `~/.spawn/keys/spawn-default.pem`.

### Can I use my existing SSH key?

Yes: `spawn launch --key-name my-existing-key-pair`

### Why does `spawn ssh` fail immediately after launch?

Instances take 30-90 seconds to boot and start sshd. Wait or poll:
```bash
while ! spawn ssh my-instance -- echo ok; do sleep 5; done
```

## Features

### What is a parameter sweep?

A parameter sweep launches many instances with different configurations (e.g., different model hyperparameters), waits for results, and aggregates output. See [Parameter Sweeps](../features/parameter-sweeps.md).

### What is hibernation?

Hibernation stops an EC2 instance and preserves RAM state to EBS. The instance resumes exactly where it left off on wake. Hibernation requires instance types and AMIs that support it. See [Hibernation](../features/hibernation.md).

### What is autoscaling?

Autoscaling automatically adjusts instance count based on queue depth, CPU, memory, or schedules. See [Autoscaling](../features/autoscaling.md).
