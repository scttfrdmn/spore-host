# Truffle Find - Quick Reference

## Command Syntax
```bash
truffle find "<natural language query>"
```

## Workshop Examples

### GPU Queries (Easiest for Researchers!)
```bash
# Find H100 GPUs (latest NVIDIA for training)
truffle find h100

# Find A100 GPUs
truffle find a100

# Find V100 GPUs (older training GPUs)
truffle find v100

# Find T4 GPUs (inference)
truffle find t4

# Find AWS Inferentia (ML inference chips)
truffle find inferentia

# Find AWS Trainium (ML training chips)
truffle find trainium
```

### CPU Vendor Queries
```bash
# All Intel instances
truffle find intel

# All AMD instances
truffle find amd

# All Graviton (ARM) instances
truffle find graviton
```

### Processor Code Name Queries
```bash
# Intel Ice Lake (3rd gen)
truffle find "ice lake"

# AMD Milan (3rd gen EPYC)
truffle find milan

# Intel Sapphire Rapids (4th gen, latest)
truffle find "sapphire rapids"

# AMD Genoa (4th gen EPYC, latest)
truffle find genoa
```

### Size + Vendor Combinations
```bash
# Large AMD instances
truffle find "large amd"

# Huge Intel instances (8xlarge+)
truffle find "huge intel"

# Small Graviton instances
truffle find "small graviton"
```

### Spec-Based Queries
```bash
# AMD with at least 16 cores
truffle find "amd 16 cores"

# Graviton with at least 32GB RAM
truffle find "graviton 32gb"

# Intel with 8 cores and 16GB
truffle find "intel 8 cores 16gb"

# AMD Milan with 64+ cores (HPC)
truffle find "milan 64 cores"
```

### Combined Queries
```bash
# Large Graviton instances
truffle find "graviton large"

# Ice Lake with 32GB+ RAM
truffle find "ice lake 32gb"

# AMD with GPU
truffle find "amd gpu"
```

## Useful Flags

```bash
# Show what the parser understood (debugging)
truffle find "h100" --show-query

# Search specific regions only
truffle find "a100" -r us-east-1,us-west-2

# Skip AZ lookup for faster results
truffle find "graviton" --skip-azs

# JSON output for scripts/automation
truffle find "large amd" -o json

# CSV output for spreadsheets
truffle find "intel gpu" -o csv

# Verbose mode (shows why instances matched)
truffle find "ice lake 16 cores" -v
```

## What It Understands

### CPU Vendors
- `intel` - All Intel Xeon instances
- `amd` - All AMD EPYC instances  
- `graviton`, `arm`, `aws` - AWS Graviton (ARM64)

### Processor Code Names

**Intel:**
- `ice lake` - 3rd gen Xeon (m6i, c6i, r6i families)
- `sapphire rapids` - 4th gen Xeon (m7i, c7i, r7i families)
- `cascade lake` - 2nd gen Xeon (m5, c5, r5 families)

**AMD:**
- `milan` - 3rd gen EPYC (m6a, c6a, r6a families)
- `genoa` - 4th gen EPYC (m7a, c7a, r7a families)
- `rome` - 2nd gen EPYC (m5a, c5a, r5a families)

**AWS Graviton:**
- `graviton`, `graviton2`, `graviton3`, `graviton4`

### GPU Types

**NVIDIA Training:**
- `h100` - Latest, best for large models
- `a100` - Previous gen training
- `v100` - Older training workloads
- `k80` - Legacy

**NVIDIA Inference:**
- `a10g` - Latest inference GPU
- `t4` - Popular inference GPU
- `l4` - Newer inference option

**AWS Accelerators:**
- `inferentia` - AWS ML inference chip
- `inferentia2` - Newer inference chip
- `trainium` - AWS ML training chip

**AMD:**
- `radeon pro v520` - AMD graphics workloads

### Sizes
- `tiny` - nano, micro
- `small` - small, medium
- `medium` - large, xlarge
- `large` - 2xlarge, 4xlarge
- `huge` - 8xlarge and above, metal

### Specs
- **vCPUs:** `8 cores`, `16 vcpus`, `32 cpus`
- **Memory:** `32gb`, `64gib`, `128g`
- **GPUs:** `4 gpus`
- **Architecture:** `x86_64`, `arm64`

## Comparison: Old vs New

### Finding H100 GPUs

**Old way (regex - complicated):**
```bash
truffle search "p5\\..*"
# Requires knowing:
# - p5 is the H100 family
# - Regex syntax for escaping dots
# - Instance naming conventions
```

**New way (natural language - easy!):**
```bash
truffle find h100
# Just say what you want!
```

### Finding Large AMD Instances

**Old way:**
```bash
truffle search "^(m6a|c6a|r6a|m7a|c7a|r7a)\\.(2xlarge|4xlarge)$" --architecture x86_64
# Requires knowing:
# - All AMD family names
# - Size naming conventions
# - Complex regex syntax
```

**New way:**
```bash
truffle find "large amd"
# Natural and intuitive!
```

## Tips for Workshop

1. **Start with simple queries** - `truffle find h100`
2. **Show the comparison** - regex vs natural language
3. **Demo --show-query** - helps researchers understand what it found
4. **Use real examples** - their actual GPU/CPU needs
5. **Combine with regions** - `truffle find a100 -r us-west-2`

## Output Formats

**Table (default):**
```bash
truffle find h100
# Human-readable table with instance details
```

**JSON (for scripts):**
```bash
truffle find h100 -o json | jq '.[] | {instance: .instance_type, vcpus: .vcpus}'
```

**CSV (for Excel):**
```bash
truffle find "large amd" -o csv > amd_instances.csv
```

## Common Use Cases

### ML/AI Researchers
```bash
# Find best training GPUs
truffle find h100

# Find inference GPUs
truffle find t4

# Find AWS ML chips
truffle find inferentia
```

### HPC Users
```bash
# High core count AMD
truffle find "amd 64 cores"

# Latest Intel processors
truffle find "sapphire rapids"

# ARM for efficiency
truffle find "graviton 32 cores"
```

### Cost-Conscious Users
```bash
# Small instances for testing
truffle find "small graviton"

# Find specific sizes
truffle find "amd medium"
```

## Binary Location

The newly built binary is at:
```
/Users/scttfrdmn/src/spore-host/truffle/bin/truffle
```

## Test Without AWS Credentials

```bash
# See parsing without AWS API calls
truffle find "your query" --show-query 2>&1 | head -5
```

This shows what the parser understood from your query!
