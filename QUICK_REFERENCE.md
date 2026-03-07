# truffle + spawn Quick Reference

## truffle — Find Capacity (no AWS credentials required for most commands)

```bash
# Search for instance types
truffle search m7i.large
truffle search "m7i.*" --regions us-east-1
truffle search "*.xlarge" --architecture arm64 --min-vcpu 4

# Compare spot prices
truffle spot c6i.xlarge c6a.xlarge c7g.xlarge --sort-by-price --active-only
truffle spot "m8g.*" --max-price 0.10 --show-savings

# Check GPU/ML capacity reservations
truffle capacity --gpu-only --available-only
truffle capacity --instance-types p5.48xlarge,g6.xlarge

# Check your account quotas (requires credentials)
truffle quotas
truffle quotas --family P
```

## spawn — Launch and Manage (requires AWS credentials)

```bash
# Launch
spawn launch --name my-job --instance-type c6a.xlarge --ttl 4h --on-complete terminate
spawn launch --name my-job --instance-type c6a.xlarge --spot --ttl 8h
spawn launch --name my-job --instance-type t4g.medium --idle-timeout 30m --on-complete terminate
spawn launch --name my-job --instance-type t4g.medium --ttl 4h --script job.sh

# Connect, check, manage
spawn connect my-job          # SSH by name
spawn status my-job           # TTL remaining, idle state
spawn extend my-job 2h        # Add 2 hours to TTL (live, no restart)
spawn list                    # All running instances

# Parameter sweeps
spawn sweep --params grid.yaml --job-array-name my-sweep
spawn list --array my-sweep
spawn extend --job-array-name my-sweep 2h
```

---

## Common Workflows

### Quick dev box

```bash
spawn launch --name dev --instance-type t4g.medium --ttl 8h --idle-timeout 1h
spawn connect dev
```

### Find cheapest spot, then launch

```bash
# Compare across Intel, AMD, and Graviton
truffle spot c6i.xlarge c6a.xlarge c7g.xlarge --sort-by-price --active-only

# Launch into cheapest region
spawn launch --name my-job --instance-type c6a.xlarge --region us-east-1 --spot --ttl 4h --on-complete terminate
```

### Job with completion signal

```bash
# Add to your job script:
touch /tmp/SPAWN_COMPLETE

# Launch with TTL as backstop:
spawn launch --name my-analysis --instance-type t4g.medium --ttl 4h --on-complete terminate --script job.sh
```

### Extend a running job

```bash
spawn status my-analysis      # Check remaining TTL
spawn extend my-analysis 2h   # Add 2 hours — spored picks it up live
spawn status my-analysis      # Confirm
```

### GPU instance

```bash
truffle quotas --family P               # Check quota (often 0)
truffle capacity --gpu-only --available-only   # Find available capacity
spawn launch --name gpu-job --instance-type g4dn.xlarge --ttl 24h
```

---

## Termination Triggers

| Trigger | Flag | Fires when |
|---------|------|-----------|
| Completion signal | — | `touch /tmp/SPAWN_COMPLETE` on the instance |
| Idle timeout | `--idle-timeout 20m` | CPU idle for N minutes |
| TTL | `--ttl 4h` | Hard deadline reached |

**Default safety net:** if neither `--ttl` nor `--idle-timeout` is set, spawn applies `--idle-timeout 1h` automatically.

---

## Output Formats

```bash
truffle search m7i.large --output table    # default
truffle search m7i.large --output json     # for scripting
truffle search m7i.large --output yaml     # human-readable
truffle search m7i.large --output csv      # spreadsheet

# Pipeline example
truffle spot c6a.xlarge --sort-by-price --active-only --output json | jq -r '.[0].region'
```

---

## AWS Credentials

| Command | Credentials needed? |
|---------|-------------------|
| `truffle search` | No |
| `truffle spot` | No |
| `truffle capacity` | No |
| `truffle quotas` | Yes |
| `spawn` (all commands) | Yes |

```bash
aws configure
# or
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_DEFAULT_REGION=us-east-1
```

---

## Help

```bash
truffle --help
truffle search --help
truffle spot --help
truffle capacity --help
truffle quotas --help

spawn --help
spawn launch --help
spawn connect --help
spawn extend --help
```
