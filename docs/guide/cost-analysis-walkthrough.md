# Cost Analysis Walkthrough

Use spawn's cost tracking features to understand and reduce your AWS spending.

## Scenario

You've been running spawn for a few weeks and want to understand your costs, identify waste, and set up alerts before the monthly bill surprises you.

---

## Step 1: Check Your Current Month's Cost

```bash
truffle cost --days 30
```

Output:
```
Cost Summary (last 30 days)
───────────────────────────────────────
Total:          $47.23
Spot savings:   $31.18 vs on-demand

By Instance:
  training-job-001   $12.40   (c7i.4xlarge, 24h, spot)
  dev-workspace      $8.20    (m7i.large, 40h, on-demand)
  sweep-batch-v3     $6.80    (12× c7i.large, 2h, spot)
  old-instance       $15.83   (r7i.2xlarge, running 76h?!)
  ...
```

Notice `old-instance` — running 76 hours when jobs take 2 hours. That's the first thing to fix.

---

## Step 2: Find Long-Running Instances

```bash
truffle ls --state running --sort age
```

Look for instances running much longer than expected.

Terminate forgotten instances:
```bash
spawn terminate old-instance
```

---

## Step 3: View Cost Breakdown by Type

```bash
truffle cost --group type --days 30
```

Output:
```
By Instance Type (last 30 days):
  r7i.2xlarge    $18.30   38%
  c7i.4xlarge    $14.20   30%
  m7i.large      $10.50   22%
  c7i.large      $4.23    9%
```

r7i.2xlarge is 38% of costs. Check if those jobs actually need high memory:

```bash
truffle metrics --instance-type r7i.2xlarge --days 30
# If memory utilization < 30%, switch to m7i.xlarge
```

---

## Step 4: Check Dashboard Cost Charts

Open [https://spore.host/dashboard.html](https://spore.host/dashboard.html) and look at the cost charts:

- **Cost by Instance**: identify high-cost instances
- **Spot Savings**: see how much spot is saving vs on-demand
- **Monthly Trend**: is cost trending up?

---

## Step 5: Set a Cost Alert

In the dashboard Settings tab, configure a monthly alert:

1. Go to Settings tab
2. Set "Monthly cost alert threshold": $50
3. Enter notification email
4. Save

Or via config file:
```yaml
# ~/.spawn/config.yaml
cost:
  alert_threshold: 50.00
  alert_email: me@example.com
```

---

## Step 6: Apply Optimizations

Based on the analysis:

**1. Set shorter TTLs for batch jobs:**
```bash
# Before (no TTL safety net)
spawn launch --instance-type r7i.2xlarge

# After
spawn launch --instance-type r7i.2xlarge --ttl 4h --idle-timeout 30m
```

**2. Switch appropriate jobs to spot:**
```bash
# Training jobs that can checkpoint
spawn launch --spot --instance-type c7i.4xlarge
```

**3. Right-size memory-intensive instances:**
```bash
# Test with smaller instance first
spawn launch --instance-type m7i.xlarge  # vs r7i.2xlarge
```

**4. Hibernate dev workspace instead of leaving it running:**
```bash
spawn launch --instance-type m7i.large \
  --idle-timeout 1h \
  --idle-action hibernate
```

---

## Step 7: Track Savings

After 2 weeks, recheck:
```bash
truffle cost --days 14 --show-savings
```

Compare total cost and spot savings percentage to the baseline.
