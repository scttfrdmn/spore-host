# Dashboard

The spore.host web dashboard provides a real-time view of your running instances, cost metrics, alerts, and autoscale groups.

Access: [https://spore.host/dashboard.html](https://spore.host/dashboard.html)

## Authentication

Sign in with Globus Auth or AWS credentials. After login, the dashboard reads your instance data from AWS DynamoDB.

## Tabs

### Instances Tab

Lists all running instances across all regions. Columns:

| Column | Description |
|--------|-------------|
| Name | Instance name tag |
| Type | EC2 instance type |
| State | running / stopped / hibernated |
| Region | AWS region |
| Public IP | Current public IP address |
| DNS Name | spore.host DNS record |
| TTL | Time remaining before auto-termination |

**Actions** (per-row buttons):
- **SSH** — open SSH connection command
- **Extend** — add time to TTL
- **Hibernate** — stop instance, preserve EBS state
- **Terminate** — immediately terminate

**Filters and Search:**
- Search by name, type, or state using the filter bar
- Sort any column by clicking the header
- Filter by region using the region dropdown

### Sweeps Tab

Shows running and completed parameter sweep jobs.

Columns: Name, Status, Progress, Instances, Start Time, Cost

### Autoscale Tab

Shows autoscale group status with queue depth gauges.

Each group displays:
- Current / desired / min / max capacity
- Queue depth (SQS messages waiting)
- Scale-up and scale-down thresholds
- Recent scaling events

### Settings Tab (Alerts)

Configure alert preferences:

- **Cost alerts**: threshold in USD/month, notification email
- **Instance alerts**: notify on termination, spot interruption
- **Idle alerts**: notify when instance idle timeout approaches
- **Slack integration**: webhook URL for Slack notifications

## Cost Charts

The dashboard header shows monthly cost charts:

- **Cost by Instance** — breakdown per instance
- **Cost by Type** — breakdown by EC2 instance type
- **Savings vs Sticker** — spot savings compared to on-demand pricing
- **Total Monthly** — running total for current month

Hover over chart segments for detailed breakdowns.

## Queue Depth Gauges

On the Autoscale tab, each autoscale group has a circular gauge showing current queue depth relative to configured thresholds. The gauge changes color:
- Green: below scale-up threshold
- Yellow: approaching threshold
- Red: at or above scale-up threshold

## Connection Status

A colored dot in the top-right corner indicates dashboard connection status:
- Green: connected, data fresh
- Orange: polling (websocket fallback)
- Red: disconnected, showing stale data

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `g i` | Switch to Instances tab |
| `g s` | Switch to Sweeps tab |
| `g a` | Switch to Autoscale tab |
| `r` | Refresh current tab |
| `/` | Focus search/filter input |
| `?` | Show keyboard shortcuts help |

## Mobile

On mobile devices (≤768px):
- Navigation collapses to a hamburger menu
- Instance table switches to card view (one card per instance)
- Swipe left/right to switch between tabs
