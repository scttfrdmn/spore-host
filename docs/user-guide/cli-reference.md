# CLI Reference

spawn and truffle are the two CLI tools in the spore-host suite.

## spawn — Launch and Manage Instances

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--region` | `$AWS_REGION` | AWS region |
| `--profile` | `$AWS_PROFILE` | AWS credentials profile |
| `--config` | `~/.spawn/config.yaml` | Config file path |
| `--json` | false | Output as JSON |
| `--debug` | false | Enable debug logging |

### Commands

#### `spawn launch`

Launch a new EC2 instance.

```bash
spawn launch [flags]

Flags:
  --instance-type   string    EC2 instance type (default: t3.micro)
  --ami             string    AMI ID (default: auto-detected Amazon Linux 2023)
  --ttl             duration  Time-to-live before auto-termination (default: 4h)
  --idle-timeout    duration  Terminate if idle for this duration (default: 30m)
  --name            string    Instance name tag
  --spot            bool      Use spot instances (default: false)
  --key-name        string    SSH key pair name (default: auto-created)
  --security-group  string    Security group ID (default: auto-created)
  --subnet-id       string    VPC subnet ID
  --tags            strings   Additional tags (key=value)
  --user-data       string    User data script path
  --no-spored       bool      Skip spored agent installation
  --dry-run         bool      Validate without launching
```

#### `spawn ssh`

Connect to an instance via SSH.

```bash
spawn ssh <instance-name|id> [-- ssh-args]

Flags:
  --user    string    SSH user (default: ec2-user)
  --port    int       SSH port (default: 22)
  --forward strings   Port forwarding (local:remote)
```

#### `spawn terminate`

Terminate one or more instances.

```bash
spawn terminate <instance-name|id> [...]

Flags:
  --all     bool    Terminate all instances
  --force   bool    Skip confirmation prompt
```

#### `spawn extend`

Extend an instance's TTL.

```bash
spawn extend <instance-name|id> <duration>

# Examples:
spawn extend my-instance 1h
spawn extend my-instance +30m
spawn extend my-instance 2026-01-01T00:00:00Z  # absolute time
```

#### `spawn hibernate`

Hibernate an instance (stop with EBS state preserved).

```bash
spawn hibernate <instance-name|id>
spawn wake <instance-name|id>
```

#### `spawn sweep`

Launch a parameter sweep job across multiple instances.

```bash
spawn sweep --config sweep.yaml

Flags:
  --config   string    Sweep config file
  --dry-run  bool      Show instances that would be launched
```

#### `spawn logs`

View spored agent logs for an instance.

```bash
spawn logs <instance-name|id> [flags]

Flags:
  --tail   int     Number of recent lines (default: 50)
  --follow bool    Stream logs
```

---

## truffle — Discover and Query Instances

### Commands

#### `truffle ls`

List running instances.

```bash
truffle ls [flags]

Flags:
  --region    string    Filter by region (default: all regions)
  --state     string    Filter by state (running|stopped|all)
  --tag       strings   Filter by tag (key=value)
  --json      bool      Output as JSON
  --format    string    Output format template
```

#### `truffle get`

Get details for a specific instance.

```bash
truffle get <instance-name|id>
```

#### `truffle cost`

Show cost information for instances.

```bash
truffle cost [flags]

Flags:
  --days     int      Number of days of history (default: 30)
  --group    string   Group by: instance|type|region|tag
  --format   string   Output format: table|json|csv
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (see stderr) |
| 2 | Usage error (bad flags or arguments) |

## Environment Variables

See [Configuration](configuration.md#environment-variables) for the full list.

## Configuration File

See [Configuration](configuration.md) for format and options.
