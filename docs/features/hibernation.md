# Hibernation

## What

Hibernation stops an EC2 instance while preserving its RAM contents to EBS. On wake, the instance resumes exactly where it left off — processes running, files open, network connections restored.

## Why

- Resume a long computation without restarting from scratch
- Pause expensive instances (GPU, memory-optimized) during off-hours
- Save ~70% vs leaving instances running during idle periods
- Preserve in-memory state between sessions

## How

```bash
# Hibernate an instance
spawn hibernate my-instance

# Wake it up
spawn wake my-instance

# Check hibernation status
truffle get my-instance
```

### What Happens During Hibernate

1. spored initiates EC2 StopInstances with `hibernate=true`
2. EC2 writes RAM contents to the root EBS volume (encrypted)
3. Instance enters `stopped` state
4. DNS record updated to reflect stopped status
5. EBS volumes remain attached (you pay for storage, not compute)

### What Happens During Wake

1. EC2 StartInstances called
2. RAM contents restored from EBS
3. Instance returns to `running` state
4. DNS updated with new IP address
5. spored resumes TTL and monitoring

## Requirements

- Instance type must support hibernation (most modern types do)
- Root EBS volume must be encrypted
- Root EBS volume must be large enough to hold RAM contents
- Instance must not have been running longer than 60 days

Supported families: C3, C4, C5, M3, M4, M5, R3, R4, R5, and newer.

## Configuration

Enable hibernation support at launch:

```bash
spawn launch \
  --instance-type m7i.large \
  --enable-hibernation \
  --ttl 8h
```

Or in config:
```yaml
defaults:
  enable_hibernation: true
  ebs_encrypted: true
```

## Automatic Hibernation (Idle)

Configure spored to hibernate instead of terminate on idle:

```yaml
defaults:
  idle_action: hibernate  # default: terminate
  idle_timeout: 1h
```

With this setting, an idle instance hibernates (preserving state) rather than terminating.

## Cost During Hibernation

During hibernation, you pay only for:
- EBS storage (root volume): ~$0.08/GB/month
- EIP (if allocated): ~$0.005/hour

You do NOT pay for instance hours while stopped.

## Best Practices

- Use hibernation for interactive dev sessions you return to daily
- Encrypt EBS volumes (required for hibernation, also good security practice)
- Set `idle_action: hibernate` for GPU instances (expensive to re-launch)
- Wake instances before they exceed the 60-day hibernate limit

## Limitations

- Not all instance types support hibernation
- Maximum hibernation duration: 60 days
- Bare metal instances do not support hibernation
- Instance store volumes are not preserved (EBS only)
