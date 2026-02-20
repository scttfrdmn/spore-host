# Spot Instances

## What

EC2 spot instances run on AWS's spare capacity at up to 70-90% discount compared to on-demand pricing. In exchange, AWS can reclaim them with 2 minutes notice.

## Why

For batch workloads, parameter sweeps, and training jobs that can checkpoint progress, spot instances dramatically reduce costs:

| Instance | On-Demand | Spot (typical) | Savings |
|----------|-----------|-----------------|---------|
| c7i.large | $0.0850/hr | $0.0255/hr | 70% |
| m7i.xlarge | $0.1920/hr | $0.0576/hr | 70% |
| g5.xlarge | $1.006/hr | $0.302/hr | 70% |
| p3.2xlarge | $3.060/hr | $0.918/hr | 70% |

## How

```bash
# Enable spot
spawn launch --spot --instance-type c7i.large --ttl 4h

# Spot with fallback to on-demand
spawn launch --spot --spot-fallback --instance-type c7i.large
```

When spot capacity is unavailable, spawn can fall back to on-demand (`--spot-fallback`).

## Interruption Handling

spored polls the EC2 instance metadata service every 5 seconds for interruption notices. When a 2-minute warning is received:

1. spored logs the interruption event
2. Runs configured pre-termination cleanup commands
3. Saves outputs to S3 (if configured)
4. Gracefully terminates the instance

### Configure Cleanup Commands

```yaml
# ~/.spawn/config.yaml
spot:
  cleanup_commands:
    - "tar -czf /tmp/checkpoint.tar.gz /data/checkpoint"
    - "aws s3 cp /tmp/checkpoint.tar.gz s3://my-bucket/checkpoints/"
  save_outputs: true
  output_bucket: my-bucket
  output_prefix: spot-outputs/
```

## Spot Fleet Diversification

Use multiple instance types to increase availability:

```bash
spawn launch \
  --spot \
  --instance-type c7i.large,c6i.large,c5.large \
  --allocation-strategy lowest-price
```

## Best Practices

- Use spot for stateless or checkpointed workloads
- Diversify across multiple instance types and AZs
- Implement checkpointing for long-running jobs
- Monitor spot prices: `spawn spot-price --instance-type c7i.large`
- Parameter sweeps are ideal for spot: many short jobs, easily retried

## When NOT to Use Spot

- Interactive development sessions (interruption is disruptive)
- Jobs that cannot be checkpointed and take >4 hours
- Workloads with hard deadlines

## Monitoring Spot Interruptions

```bash
# View interruption history for an instance
spawn logs my-instance | grep "spot interruption"

# Dashboard shows spot events on the Instances tab
```
