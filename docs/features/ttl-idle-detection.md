# TTL & Idle Detection

## What

Two complementary mechanisms prevent instances from running indefinitely:

1. **TTL (Time-To-Live)**: Hard deadline. Instance terminates at a fixed time.
2. **Idle Detection**: Soft deadline. Instance terminates if utilization stays low.

## TTL

### How TTL Works

spored checks elapsed time every 30 seconds. At 10 minutes before expiry, it logs a warning. At TTL expiry, spored runs cleanup commands and terminates the instance.

```
launch → running → TTL-10min warning → TTL expiry → cleanup → terminate
```

### Setting TTL

```bash
# At launch
spawn launch --ttl 2h
spawn launch --ttl 30m
spawn launch --ttl 1h30m

# Absolute time
spawn launch --ttl 2026-01-01T18:00:00Z

# No TTL (not recommended)
spawn launch --ttl 0
```

### Extending TTL

```bash
# Add 2 hours
spawn extend my-instance 2h

# Set absolute expiry
spawn extend my-instance 2026-01-01T20:00:00Z
```

### Default TTL

Default: `4h`. Configure in `~/.spawn/config.yaml`:

```yaml
defaults:
  ttl: 8h
```

## Idle Detection

### How Idle Detection Works

spored samples CPU, memory, and network at 30-second intervals, averaging over a sliding window (default: 5 minutes). If all metrics stay below thresholds for the full `idle-timeout` duration, the instance terminates.

Metrics monitored:
- CPU utilization (%)
- Memory utilization (%)
- Network I/O (bytes/sec)
- Disk I/O (bytes/sec, optional)

### Default Thresholds

| Metric | Threshold |
|--------|-----------|
| CPU | < 5% |
| Memory | < 20% |
| Network | < 10 KB/s |

### Configuration

```bash
# At launch
spawn launch --idle-timeout 30m

# Disable idle detection
spawn launch --idle-timeout 0
```

Custom thresholds in config:
```yaml
defaults:
  idle_timeout: 30m
  idle_cpu_threshold: 5
  idle_memory_threshold: 20
  idle_network_threshold: 10240  # bytes/sec
```

### Interaction Between TTL and Idle Detection

- Both run independently
- Whichever triggers first terminates the instance
- TTL is the hard outer bound; idle detection is the earlier inner bound
- Setting `idle-timeout 0` disables idle detection; TTL still applies

## Best Practices

- Set TTL to maximum expected duration as a safety net
- Set idle timeout to catch truly abandoned instances (30m is a good default)
- Increase idle thresholds for workloads with bursty but real activity
- Disable idle detection for interactive dev sessions where you may be inactive

## Limitations

- spored warns via logs only (no email/Slack for TTL warnings by default; configure alerts in the dashboard Settings tab)
- TTL cannot be set to more than 30 days (720h)
- Idle detection does not monitor GPU utilization in the default configuration
