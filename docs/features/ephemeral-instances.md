# Ephemeral Instances

## What

Ephemeral instances are EC2 instances designed for temporary, single-purpose workloads. Unlike traditional long-running servers, they launch on demand, run a task, and terminate automatically.

## Why

Traditional cloud compute creates persistent infrastructure that accumulates costs even when idle. Ephemeral instances:

- Terminate automatically via TTL — no manual cleanup
- Never accumulate "forgotten" costs
- Start fresh each run — no configuration drift
- Scale to zero when not in use

## How

When you run `spawn launch`, spawn:

1. **Resolves defaults** — selects region, AMI, instance type, networking
2. **Creates resources** — security group, SSH key (first time only)
3. **Calls EC2 RunInstances** — launches the instance with a user-data script
4. **Installs spored** — the user-data installs and starts the spored agent
5. **Registers metadata** — writes instance record to DynamoDB
6. **Registers DNS** — creates Route53 record (async, ~60s)
7. **Reports ready** — returns instance name, IP, DNS, TTL

The instance then runs independently. spored enforces the lifecycle without requiring a persistent connection from your machine.

## Configuration

```bash
# Launch with defaults (t3.micro, 4h TTL, idle detection 30m)
spawn launch

# Custom configuration
spawn launch \
  --instance-type m7i.large \
  --ttl 8h \
  --idle-timeout 1h \
  --name my-compute-job \
  --tags project=ml-research,team=ai
```

## Instance Lifecycle

```
launch → pending → running → [working/idle] → terminated
                                     ↓
                              (TTL or idle)
```

States:
- **pending**: EC2 starting, spored installing
- **running**: Instance active, spored monitoring
- **stopping**: TTL expired or idle, graceful shutdown
- **stopped**: Hibernated (not terminated)
- **terminated**: Permanently gone

## Best Practices

- Set TTL to slightly longer than expected job duration + buffer
- Use idle detection as a safety net for TTL overruns
- Tag instances with project and team for cost allocation
- Use `--dry-run` to validate config before launching

## Limitations

- Instance state (RAM) is lost on termination; persist outputs to S3 or EBS
- Launch time is 1-3 minutes (AMI boot + spored install)
- Maximum 100 concurrent operations per account
