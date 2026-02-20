# DNS Integration

## What

spawn automatically registers a DNS record for each launched instance in the `spore.host` hosted zone. Every instance gets a subdomain like `my-instance.spore.host` that resolves to its current public IP.

## Why

- No need to track IP addresses (they change on stop/start)
- Human-readable names for SSH connections
- Works with SSH config and known_hosts
- DNS updates automatically when IP changes (e.g., after hibernate/wake)

## How

The DNS registration flow:

1. Instance launches and boots
2. spored starts and gets the instance's public IP from EC2 metadata
3. spored calls a Lambda function with its name and IP
4. Lambda updates Route53 A record
5. DNS propagates (typically 30-60 seconds)

```
launch → boot → spored start → Lambda call → Route53 update → DNS active
```

## DNS Name Format

```
<instance-name>.spore.host
```

Examples:
- `my-instance-abc123.spore.host`
- `training-job-001.spore.host`
- `sweep-worker-042.spore.host`

## Configuration

Custom DNS prefix at launch:
```bash
spawn launch --name my-compute-job
# DNS: my-compute-job.spore.host
```

## Using DNS Names

```bash
# SSH using DNS
ssh ec2-user@my-instance.spore.host

# spawn ssh also uses DNS
spawn ssh my-instance

# SCP
scp ec2-user@my-instance.spore.host:/results/output.csv ./
```

## DNS Updates on IP Change

When an instance stops and restarts (or hibernates and wakes), it gets a new public IP. spored detects the IP change and updates the DNS record automatically.

This means you can always use the same DNS name even after a restart.

## DNS Propagation

DNS TTL is set to 60 seconds. After a launch or IP change, allow up to 60-120 seconds for DNS to propagate globally.

Check DNS resolution:
```bash
dig my-instance.spore.host
nslookup my-instance.spore.host 8.8.8.8
```

## Troubleshooting

**DNS not resolving after launch**:
- Allow 60-120 seconds for propagation
- Check spored started: `spawn logs my-instance | grep dns`
- Verify Lambda has Route53 permissions

**DNS resolves to old IP**:
- TTL may be cached locally; flush: `sudo dscacheutil -flushcache` (macOS)
- Check spored updated DNS after IP change
