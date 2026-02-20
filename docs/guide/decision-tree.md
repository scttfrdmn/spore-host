# Decision Tree

Use this guide to choose the right approach for your workload.

## Should I Use Spot Instances?

```
Is your workload...
├── Fault-tolerant (can restart if interrupted)?
│   ├── Yes → Use spot instances (--spot)
│   │         Save 70-90% vs on-demand
│   └── No → Use on-demand
│             Is job runtime < 6 hours?
│             ├── Yes → On-demand is fine
│             └── No → Consider checkpointing + spot
```

## What TTL Should I Set?

```
How long will the job run?
├── Know the exact duration → TTL = duration + 20% buffer
│   Example: 2-hour job → --ttl 2h30m
│
├── Variable duration → TTL = max expected + idle detection
│   Example: --ttl 8h --idle-timeout 30m
│   (instance terminates 30m after job completes)
│
└── Interactive session → Long TTL + hibernate on idle
    Example: --ttl 720h --idle-action hibernate --idle-timeout 1h
```

## Single Instance vs Parameter Sweep

```
How many configurations do I need to run?
├── 1 → spawn launch
├── 2-5 → spawn launch (in a loop or manually)
├── 6-200 → spawn sweep
└── 200+ → autoscaling + SQS queue
```

## Terminate vs Hibernate

```
Will I need the instance again (same session)?
├── Yes, within a few hours → Hibernate (--idle-action hibernate)
│   Pros: Resume exactly where left off
│   Cons: Pays for EBS storage while stopped
│
└── No, or clean start is fine → Terminate (default)
    Pros: No ongoing costs, clean state
    Cons: Full re-launch needed (~2 minutes)
```

## On-Demand vs Autoscaling

```
Is your workload...
├── Ad-hoc / interactive → spawn launch
│   Simple, one-at-a-time
│
├── Regular batch jobs → spawn sweep
│   Known parameters, finite job list
│
└── Continuous queue processing → spawn autoscale
    Variable load, SQS-driven
```

## Instance Type Selection

```
What's your workload?
├── Development / testing → t3.micro or t3.small
│   (free-tier eligible)
│
├── General web/app server → m7i.large or m7i.xlarge
│   Balanced CPU + memory
│
├── CPU-bound (rendering, simulation, ML training) → c7i.large+
│   High CPU:memory ratio
│
├── Memory-bound (large datasets in RAM) → r7i.large+
│   High memory:CPU ratio
│
└── GPU workloads (deep learning, CUDA) → g5.xlarge+ or p3.2xlarge+
    Use --spot for 70% savings
```
