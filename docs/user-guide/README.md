# User Guide

Welcome to the spore.host user guide. These docs cover everything you need to use spawn and truffle to manage ephemeral AWS EC2 instances.

## Contents

| Document | Description |
|----------|-------------|
| [Getting Started](getting-started.md) | Prerequisites, account setup, first instance |
| [Installation](installation.md) | Install spawn and truffle on macOS/Linux |
| [CLI Reference](cli-reference.md) | Command overview and flag reference |
| [Dashboard](dashboard.md) | Web dashboard guide (instances, cost, alerts) |
| [Configuration](configuration.md) | Config file, environment variables, profiles |
| [Authentication](authentication.md) | AWS credentials, IAM roles, multi-account |
| [SSH Access](ssh-access.md) | Connecting to instances, key management |
| [Troubleshooting](troubleshooting.md) | Common errors and fixes |
| [FAQ](faq.md) | Frequently asked questions |
| [Glossary](glossary.md) | Key terms explained |
| [Upgrading](upgrading.md) | Version migration notes |

## Quick Reference

```bash
# Launch an instance with 2-hour TTL
spawn launch --ttl 2h

# List running instances
truffle ls

# Connect via SSH
spawn ssh <instance-name>

# Terminate an instance
spawn terminate <instance-name>
```

## Feature Docs

For deep dives into specific features, see the [Feature Documentation](../features/README.md).

## Walkthroughs

For step-by-step task guides, see the [Interactive Guide](../guide/README.md).
