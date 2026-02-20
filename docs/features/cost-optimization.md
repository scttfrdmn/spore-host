# Cost Optimization

## What

spawn provides multiple mechanisms to reduce EC2 costs: automatic termination (TTL), idle detection, spot instances, hibernation, and detailed cost tracking.

## Cost Components

For each instance, you pay:
- **Compute**: Instance-hours × hourly rate
- **Storage**: EBS volume GB-months × $0.08-0.10/GB
- **Data transfer**: Outbound data to internet

spawn's infrastructure overhead: ~$1-6/month (Lambda, DynamoDB, S3).

## Cost Formula

```
Cost = Hours × Hourly Rate
     + Storage GB × $0.08
     - Spot Discount (0-70%)
```

## Optimization Strategies

### 1. TTL Enforcement

The most important control. Every spawned instance has a TTL that triggers auto-termination.

```bash
# Match TTL to expected job duration
spawn launch --ttl 2h          # for 2-hour jobs
spawn launch --ttl 30m         # for quick tasks
```

Even with a 2-hour TTL, if the job takes 45 minutes and idle detection triggers, the instance terminates at 45 minutes.

### 2. Idle Detection

Automatically terminates instances that finish early:

```bash
spawn launch --ttl 8h --idle-timeout 30m
```

An 8-hour job that completes in 3 hours terminates 30 minutes after completion — not 5 hours later.

### 3. Spot Instances

Up to 70-90% cheaper than on-demand:

```bash
spawn launch --spot --instance-type c7i.large
```

Best for: batch workloads, parameter sweeps, ML training.

### 4. Right-Sizing

Don't over-provision. Use the smallest instance type that meets your needs.

```bash
# Use truffle to see actual utilization
truffle metrics my-instance --days 7
```

Rule of thumb:
- Dev/testing: t3.micro–t3.medium
- General compute: m7i.large–m7i.2xlarge
- CPU-intensive: c7i.large–c7i.4xlarge
- Memory-intensive: r7i.large–r7i.2xlarge

### 5. Hibernation for Dev Instances

Stop dev instances overnight instead of terminating:

```yaml
defaults:
  idle_action: hibernate
  idle_timeout: 1h
```

You pay for EBS storage (~$0.08/GB/month) instead of instance hours during off-hours.

### 6. Regional Arbitrage

Spot prices vary by region and AZ. Use the cheapest:

```bash
spawn spot-price --instance-type c7i.large --region all
```

## Cost Tracking

```bash
# Monthly cost summary
truffle cost --days 30

# Cost by instance type
truffle cost --group type --days 30

# Cost with spot savings breakdown
truffle cost --show-savings --days 30
```

## Dashboard Cost Charts

The dashboard header shows:
- Cost by instance (30-day)
- Cost by instance type
- Spot savings vs on-demand
- Monthly trend

## Cost Alerts

Configure in `~/.spawn/config.yaml`:
```yaml
cost:
  alert_threshold: 50.00   # USD/month
  alert_email: me@example.com
```

Or in the dashboard Settings tab.

## Anti-Patterns (What Not To Do)

- Launch without TTL (`--ttl 0`) and forget to terminate
- Use large instance types for lightweight tasks
- Leave hibernated instances indefinitely (EBS costs accumulate)
- Use on-demand for batch workloads that tolerate spot interruption

## Best Practices

1. Always set a TTL — even a long one is better than none
2. Enable idle detection as a safety net
3. Use spot for any workload that can checkpoint
4. Tag instances with project/team for cost allocation
5. Set a cost alert threshold in the dashboard
6. Review `truffle cost --days 30` weekly
