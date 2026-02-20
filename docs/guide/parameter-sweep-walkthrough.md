# Parameter Sweep Walkthrough

Run a complete parameter sweep from configuration to result collection.

## Scenario

You have a machine learning training script and want to find the best hyperparameters across a grid of learning rates, batch sizes, and model sizes. Running each combination takes ~20 minutes.

## What You'll Build

A sweep that:
- Runs 12 combinations in parallel (instead of sequentially)
- Uses spot instances for 70% cost savings
- Saves results to S3
- Completes in ~25 minutes instead of 4 hours

---

## Step 1: Write Your Training Script

Create `train.sh`:

```bash
#!/bin/bash
set -e

echo "Job $SPAWN_JOB_ID starting"
echo "Learning rate: $SWEEP_LR"
echo "Batch size: $SWEEP_BATCH_SIZE"
echo "Hidden dim: $SWEEP_HIDDEN_DIM"

# Run your actual training
python3 train.py \
    --lr $SWEEP_LR \
    --batch-size $SWEEP_BATCH_SIZE \
    --hidden-dim $SWEEP_HIDDEN_DIM \
    --output /tmp/results.json

# Save results to S3
aws s3 cp /tmp/results.json \
    s3://${SPAWN_OUTPUT_BUCKET}/${SPAWN_OUTPUT_PREFIX}/${SPAWN_JOB_ID}/results.json

echo "Job $SPAWN_JOB_ID complete"
```

Make it executable:
```bash
chmod +x train.sh
```

---

## Step 2: Create the Sweep Config

Create `sweep.yaml`:

```yaml
name: hyperparameter-search-v1
script: train.sh

instances:
  type: c7i.large
  spot: true
  ttl: 1h
  idle_timeout: 10m

parameters:
  SWEEP_LR: [0.001, 0.01, 0.1]
  SWEEP_BATCH_SIZE: [32, 64]
  SWEEP_HIDDEN_DIM: [256, 512]

output:
  bucket: my-results-bucket
  prefix: sweeps/hyperparameter-search-v1/
```

This creates 3 × 2 × 2 = 12 combinations.

---

## Step 3: Test with 2 Instances

Before launching the full sweep, test with 2 instances:

```bash
spawn sweep --config sweep.yaml --count 2 --dry-run
```

Check the dry-run output. Then launch real test:

```bash
spawn sweep --config sweep.yaml --count 2
```

Verify the script runs correctly on 1-2 instances before the full sweep.

---

## Step 4: Launch the Full Sweep

```bash
spawn sweep --config sweep.yaml
```

Output:
```
Launching 12 instances for sweep: hyperparameter-search-v1
  Instance type: c7i.large (spot)
  TTL: 1h
  Parameters: 3×2×2 = 12 combinations

Launch progress: [████████████] 12/12 instances launched

Sweep ID: sweep-abc123
Monitor: truffle ls --sweep sweep-abc123
```

---

## Step 5: Monitor Progress

```bash
# Command line
truffle ls --sweep hyperparameter-search-v1

# Or open the dashboard Sweeps tab
# https://spore.host/dashboard.html
```

Output:
```
SWEEP: hyperparameter-search-v1 (12 instances)
Running:   8
Completed: 3
Failed:    1
Progress:  [████████░░░░] 11/12

sweep-worker-001  completed  SWEEP_LR=0.001,BATCH=32,HIDDEN=256
sweep-worker-002  running    SWEEP_LR=0.001,BATCH=32,HIDDEN=512
...
```

---

## Step 6: Collect Results

After all instances complete (or terminate):

```bash
aws s3 sync \
    s3://my-results-bucket/sweeps/hyperparameter-search-v1/ \
    ./results/
```

Aggregate:
```python
import json
import glob

results = []
for path in glob.glob('results/*/results.json'):
    with open(path) as f:
        results.append(json.load(f))

# Find best configuration
best = max(results, key=lambda r: r['val_accuracy'])
print(f"Best: LR={best['lr']}, batch={best['batch_size']}, accuracy={best['val_accuracy']:.4f}")
```

---

## Cost Example

12 × c7i.large × 25 minutes × $0.0255/hr (spot) = **$0.13**

Vs 12 × sequential × 20 min on-demand = **$0.34** and 4 hours wall-clock time.
