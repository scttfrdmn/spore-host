# Cheat Sheet

## Launch

```bash
spawn launch                                    # defaults (t3.micro, 4h TTL)
spawn launch --instance-type c7i.large          # specific type
spawn launch --ttl 2h                           # 2-hour TTL
spawn launch --spot                             # spot instance
spawn launch --name my-job                      # custom name
spawn launch --ami ami-XXXXXXXXX               # custom AMI
spawn launch --region eu-west-1                 # specific region
spawn launch --dry-run                          # validate, don't launch
```

## Connect

```bash
spawn ssh my-instance                           # SSH via spawn
ssh ec2-user@my-instance.spore.host             # direct SSH
spawn ssh my-instance -- -L 8080:localhost:80   # port forward
scp file.txt ec2-user@my-instance.spore.host:/tmp/  # upload
```

## Manage

```bash
spawn extend my-instance 2h                     # extend TTL
spawn hibernate my-instance                     # hibernate
spawn wake my-instance                          # wake from hibernate
spawn terminate my-instance                     # terminate
spawn terminate --all                           # terminate all
spawn logs my-instance                          # view logs
spawn logs my-instance --follow                 # stream logs
```

## Discover

```bash
truffle ls                                      # list all instances
truffle ls --region us-east-1                   # filter by region
truffle ls --state running                      # filter by state
truffle ls --json                               # JSON output
truffle get my-instance                         # instance details
truffle cost --days 30                          # monthly cost
truffle cost --group type                       # cost by type
```

## Parameter Sweeps

```bash
spawn sweep --config sweep.yaml                 # run sweep
spawn sweep --config sweep.yaml --count 2       # test with 2 instances
spawn sweep collect --name my-sweep --output ./  # collect results
truffle ls --sweep my-sweep                     # sweep status
```

## Autoscaling

```bash
spawn autoscale deploy --config group.yaml      # deploy group
spawn autoscale status                          # view all groups
spawn autoscale scale --name my-group --desired 5  # manual scale
spawn autoscale destroy --name my-group         # remove group
```

## Config & Debug

```bash
SPAWN_DEBUG=1 spawn launch                      # debug mode
spawn --version                                 # version
truffle --version                               # version
spawn completion bash >> ~/.bashrc              # shell completion
```

## Common Flags

| Flag | Description |
|------|-------------|
| `--region` | AWS region |
| `--profile` | AWS credentials profile |
| `--json` | JSON output |
| `--debug` | Debug logging |
| `--dry-run` | Validate without executing |
| `--force` | Skip confirmation |

## Environment Variables

```bash
export AWS_PROFILE=my-profile
export AWS_REGION=us-east-1
export SPAWN_DEBUG=1
export SPAWN_NO_EMOJI=1        # disable emoji
export SPAWN_LOG_LEVEL=debug
```
