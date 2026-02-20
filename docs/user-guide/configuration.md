# Configuration

spawn and truffle are configured via a YAML file, environment variables, and CLI flags. Later sources override earlier ones:

```
CLI flags > Environment variables > Config file > AWS config > Defaults
```

## Config File Location

Default: `~/.spawn/config.yaml`

Override: `--config /path/to/config.yaml` or `$SPAWN_CONFIG_DIR`

## Config File Format

```yaml
# ~/.spawn/config.yaml

# Default AWS settings
region: us-east-1
profile: default

# Instance defaults
defaults:
  instance_type: t3.micro
  ttl: 4h
  idle_timeout: 30m
  key_name: spawn-default
  spot: false

# SSH settings
ssh:
  user: ec2-user
  port: 22
  extra_args: "-o StrictHostKeyChecking=no"

# Network settings
network:
  vpc_id: ""            # auto-detect if empty
  subnet_id: ""         # auto-detect if empty
  security_groups: []   # auto-create if empty

# Cost management
cost:
  alert_threshold: 10.00  # USD, alert when monthly cost exceeds
  track_savings: true     # compare spot vs on-demand

# Alerts
alerts:
  email: ""             # notification email
  slack_webhook: ""     # Slack webhook URL

# Logging
log:
  level: info           # debug|info|warn|error
  file: ""              # log to file if set

# Cache
cache:
  ttl: 5m               # how long to cache AMI/pricing data
  dir: ~/.spawn/cache
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AWS_PROFILE` | AWS credentials profile |
| `AWS_REGION` | AWS region |
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `AWS_SESSION_TOKEN` | STS session token |
| `SPAWN_CONFIG_DIR` | Config directory (default: `~/.spawn`) |
| `SPAWN_DEBUG` | Enable debug logging (`1` or `true`) |
| `SPAWN_NO_EMOJI` | Disable emoji in output (`1` or `true`) |
| `SPAWN_ACCESSIBILITY` | Enable screen reader mode (`1` or `true`) |
| `SPAWN_LOG_LEVEL` | Log level: `debug|info|warn|error` |
| `SPAWN_LOG_FILE` | Log file path |
| `SPAWN_CACHE_TTL` | Cache TTL (e.g., `5m`, `1h`) |

## Per-Project Configuration

Create a `.spawn.yaml` in your project directory. spawn looks for this file in the current directory and all parent directories.

```yaml
# .spawn.yaml (project-level overrides)
defaults:
  instance_type: c7i.large
  ttl: 8h
  tags:
    - project=my-ml-project
    - team=research
```

## AWS Profiles

Use different AWS accounts by specifying profiles:

```bash
# Use a named profile
spawn launch --profile production --instance-type m7i.large

# Or set in config
region: us-west-2
profile: production
```

## Multiple Profiles (Config)

```yaml
# ~/.spawn/config.yaml
profile: dev

# Override per-command with --profile
```

To manage multiple accounts, see [Authentication](authentication.md#multi-account).
