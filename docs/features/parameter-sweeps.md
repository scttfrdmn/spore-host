# Parameter Sweeps

## What

A parameter sweep launches multiple EC2 instances in parallel, each running the same script with different input parameters. Results are collected and aggregated.

Use cases:
- Machine learning hyperparameter optimization
- Sensitivity analysis
- Monte Carlo simulations
- Batch processing with partitioned data

## Why

- Parallelize embarrassingly parallel workloads
- Leverage spot instances for 70%+ cost savings
- Auto-terminate instances when jobs complete
- Collect structured results to S3

## How

Define a sweep config file:

```yaml
# sweep.yaml
name: hyperparameter-search
instances:
  type: c7i.large
  spot: true
  ttl: 2h

parameters:
  learning_rate: [0.001, 0.01, 0.1]
  batch_size: [32, 64, 128]
  epochs: [10, 20]

script: train.sh
output:
  bucket: my-results-bucket
  prefix: sweep/hyperparameter-search/
```

Launch:
```bash
spawn sweep --config sweep.yaml
```

This launches `3 × 3 × 2 = 18` instances, each receiving a unique combination of parameters.

## Parameter Combinatorics

By default, spawn creates all combinations (Cartesian product). To specify explicit combinations:

```yaml
parameters:
  explicit:
    - {learning_rate: 0.001, batch_size: 32, epochs: 10}
    - {learning_rate: 0.01, batch_size: 64, epochs: 20}
    - {learning_rate: 0.1, batch_size: 128, epochs: 10}
```

Or use a CSV file:
```yaml
parameters:
  file: params.csv
```

## Script Interface

The sweep script receives parameters as environment variables:

```bash
#!/bin/bash
# train.sh — called with $SWEEP_LEARNING_RATE, $SWEEP_BATCH_SIZE, $SWEEP_EPOCHS

python train.py \
  --lr $SWEEP_LEARNING_RATE \
  --batch $SWEEP_BATCH_SIZE \
  --epochs $SWEEP_EPOCHS \
  --output /results/

# Upload results
aws s3 cp /results/ s3://$SPAWN_OUTPUT_BUCKET/$SPAWN_OUTPUT_PREFIX/$SPAWN_JOB_ID/
```

Built-in environment variables per instance:
| Variable | Value |
|----------|-------|
| `SPAWN_JOB_ID` | Unique job ID |
| `SPAWN_SWEEP_INDEX` | 0-based index in sweep |
| `SPAWN_OUTPUT_BUCKET` | Output S3 bucket |
| `SPAWN_OUTPUT_PREFIX` | Output S3 prefix |

## Monitoring Progress

```bash
# View sweep status
truffle ls --sweep my-sweep-name

# Dashboard Sweeps tab shows progress bar
# open https://spore.host/dashboard.html
```

## Collecting Results

```bash
# Download all results after sweep completes
aws s3 sync s3://my-results-bucket/sweep/hyperparameter-search/ ./results/

# Or use spawn collect
spawn sweep collect --name my-sweep-name --output ./results/
```

## Best Practices

- Use spot instances for cost savings (sweeps are ideal spot workloads)
- Set TTL to 2× expected job duration
- Implement S3 result saving in your script
- Test with 1-2 instances before launching full sweep: `--count 2`
- Use explicit parameter lists to avoid combinatorial explosion

## Limitations

- Maximum 200 instances per sweep
- Parameters must be serializable as environment variables (strings, numbers)
- No built-in result aggregation; bring your own (pandas, DuckDB, etc.)
