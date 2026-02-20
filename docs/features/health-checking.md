# Health Checking

## What

spawn monitors instance health and automatically replaces unhealthy instances in autoscale groups.

## How

Health checks run at two levels:

1. **EC2 status checks**: AWS checks for hardware and system issues
2. **spored health checks**: Application-level checks (configurable)

If an instance fails health checks for 3 consecutive intervals, the autoscale controller terminates and replaces it.

## Default Health Checks

spored runs these by default:
- EC2 instance status: `running`
- System reachability: EC2 system status check
- spored heartbeat: spored writes a heartbeat to DynamoDB every 30s

## Custom Health Checks

Define health check commands in your config:

```yaml
health_check:
  interval: 30s
  timeout: 10s
  retries: 3
  commands:
    - "systemctl is-active my-app"
    - "curl -sf http://localhost:8080/health"
```

A non-zero exit code from any command marks the instance unhealthy.

## Health Check in Autoscale Groups

```yaml
# autoscale-group.yaml
health_check:
  grace_period: 120s   # don't check during startup
  interval: 30s
  retries: 3
  commands:
    - "test -f /data/worker.pid"
```

## Monitoring Health

```bash
# View health status
truffle ls --health

# View health check logs
spawn logs my-instance | grep health
```

Dashboard Instances tab shows health status per instance.

## Best Practices

- Set grace_period to allow instance startup (default: 120s)
- Use lightweight health checks (avoid heavy queries)
- Health check commands should be idempotent and fast
