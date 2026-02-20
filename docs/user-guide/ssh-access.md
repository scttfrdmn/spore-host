# SSH Access

## Connecting to Instances

```bash
# Using spawn ssh (recommended)
spawn ssh my-instance

# Direct SSH with DNS name
ssh ec2-user@my-instance.spore.host

# Direct SSH with IP
ssh -i ~/.spawn/keys/spawn-default.pem ec2-user@54.123.45.67
```

## SSH Keys

spawn auto-creates an SSH key pair named `spawn-default` on first launch. The private key is stored at `~/.spawn/keys/spawn-default.pem`.

To use an existing key:
```bash
spawn launch --key-name my-existing-key
```

To use a custom key path:
```bash
ssh -i ~/.ssh/my-key.pem ec2-user@my-instance.spore.host
```

## SSH Config

Add to `~/.ssh/config` for easier access:

```ssh-config
Host *.spore.host
    User ec2-user
    IdentityFile ~/.spawn/keys/spawn-default.pem
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
```

Then connect with:
```bash
ssh my-instance.spore.host
```

## Default Users by AMI

| AMI | Default User |
|-----|-------------|
| Amazon Linux 2023 | `ec2-user` |
| Ubuntu | `ubuntu` |
| Debian | `admin` |
| CentOS | `centos` |
| RHEL | `ec2-user` |

## Port Forwarding

```bash
# Forward local port 8080 to remote port 80
spawn ssh my-instance -- -L 8080:localhost:80

# Forward local port 5432 to remote PostgreSQL
spawn ssh my-instance -- -L 5432:localhost:5432

# SOCKS proxy on local port 1080
spawn ssh my-instance -- -D 1080
```

## File Transfer

```bash
# Upload file
scp myfile.txt ec2-user@my-instance.spore.host:/tmp/

# Download file
scp ec2-user@my-instance.spore.host:/results/output.csv ./

# Sync directory
rsync -avz ./data/ ec2-user@my-instance.spore.host:/data/
```

## Persistent Sessions (tmux)

SSH connections terminate when your laptop closes. Use tmux for persistent sessions:

```bash
spawn ssh my-instance

# Inside the instance:
tmux new -s work
# Run your long job
# Detach: Ctrl-b d

# Reconnect later
spawn ssh my-instance
tmux attach -t work
```

## SSH Agent Forwarding

Forward your local SSH agent to access other hosts from your instance:

```bash
ssh -A ec2-user@my-instance.spore.host
# or in ~/.ssh/config:
# ForwardAgent yes
```

## Troubleshooting

**Connection refused**: Instance may still be starting. Wait 30-60 seconds after launch.

**Permission denied (publickey)**: Wrong key or user. Check `--key-name` and AMI default user.

**Host key verification failed**: Clear known hosts: `ssh-keygen -R my-instance.spore.host`

**Timeout**: Check security group allows port 22 from your IP. Run:
```bash
truffle get my-instance  # shows security group and network config
```
