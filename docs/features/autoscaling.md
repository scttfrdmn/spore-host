# Autoscaling

## What

Autoscaling automatically adjusts the number of instances in a group based on workload demand. Instead of manually launching and terminating instances, you define a group with scaling rules, and spawn manages capacity for you.

## Why

- No manual intervention for variable workloads
- Scale to zero when queue is empty (zero idle cost)
- Scale up fast when work arrives
- Graceful draining prevents job loss

## Scaling Strategies

### Queue-Depth Scaling

Scale based on SQS queue depth:

```yaml
scaling:
  type: queue
  queue_url: https://sqs.us-east-1.amazonaws.com/123/my-queue
  messages_per_instance: 10  # 1 instance per 10 messages
```

- 0 messages → 0 instances (scale to zero)
- 10 messages → 1 instance
- 100 messages → 10 instances (capped at max)

### Metric-Based Scaling

Scale based on CPU, memory, or custom CloudWatch metrics:

```yaml
scaling:
  type: metric
  metric:
    namespace: AWS/EC2
    name: CPUUtilization
  scale_up_threshold: 70
  scale_down_threshold: 30
```

### Scheduled Scaling

Scale on a time-based schedule:

```yaml
scaling:
  type: scheduled
  schedules:
    - cron: "0 9 * * MON-FRI"    # Monday-Friday 9am
      desired: 5
    - cron: "0 18 * * MON-FRI"   # Monday-Friday 6pm
      desired: 0
    - cron: "0 0 * * SAT,SUN"    # Weekends
      desired: 0
  timezone: America/New_York
```

### Hybrid Scaling

Combine queue-depth and scheduled scaling:

```yaml
scaling:
  type: hybrid
  queue_url: https://sqs.us-east-1.amazonaws.com/123/my-queue
  messages_per_instance: 5
  schedules:
    - cron: "0 22 * * *"
      desired: 0            # Force scale to zero at night
```

## Configuration

```yaml
# autoscale-group.yaml
name: ml-training
instance:
  type: g5.xlarge
  spot: true
  ttl: 2h

capacity:
  min: 0
  max: 20
  desired: 0

cooldown:
  scale_up: 60s
  scale_down: 300s

drain:
  enabled: true
  timeout: 30m

scaling:
  type: queue
  queue_url: https://sqs.us-east-1.amazonaws.com/123/training-queue
  messages_per_instance: 1
```

Deploy:
```bash
spawn autoscale deploy --config autoscale-group.yaml
```

## Graceful Drain

When scaling down, spawn doesn't immediately terminate instances. The drain mechanism:

1. Marks instance as draining (no new work dispatched)
2. Waits for in-flight jobs to complete (up to `drain.timeout`)
3. Terminates instance after drain completes

Disable drain for stateless workloads:
```yaml
drain:
  enabled: false
```

## Dashboard Monitoring

The Autoscale tab in the dashboard shows:
- Current / desired / min / max capacity for each group
- Queue depth gauge with scale-up threshold indicator
- Recent scaling events timeline

## Cost

Autoscaling infrastructure costs:
- Lambda (scaling controller): ~$0.30/month
- DynamoDB (job registry): ~$1-5/month
- CloudWatch alarms: ~$0.10/alarm/month

## Best Practices

- Use queue-depth scaling for batch workloads
- Set `min: 0` to scale to zero and eliminate idle costs
- Set cooldown to prevent thrashing (default: 60s up, 300s down)
- Use spot instances in autoscale groups (large savings on batch workloads)
- Monitor queue depth in the dashboard
