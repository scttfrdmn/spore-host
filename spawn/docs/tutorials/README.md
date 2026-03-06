# spawn Tutorials

Step-by-step guides to learn spawn from beginner to advanced. Follow in order for best results.

## Getting Started (Beginners)

### [Tutorial 1: Getting Started](01-getting-started.md)
**Duration:** 15 minutes | **Level:** Beginner

Your first steps with spawn:
- Install spawn
- Configure AWS credentials
- Launch your first instance
- Connect via SSH
- Terminate instances

**Prerequisites:** AWS account, basic command line knowledge

---

### [Tutorial 2: Your First Instance](02-first-instance.md)
**Duration:** 20 minutes | **Level:** Beginner

Deep dive into instance configuration:
- Choose the right instance type
- Understand AMIs
- Configure security groups and SSH keys
- Manage instance lifecycle (stop, start, hibernate)
- Use tags for organization

**Prerequisites:** [Tutorial 1: Getting Started](01-getting-started.md)

---

## Intermediate (Parallel & Batch Processing)

### [Tutorial 3: Parameter Sweeps](03-parameter-sweeps.md)
**Duration:** 30 minutes | **Level:** Intermediate

Launch multiple instances with different configurations:
- Create parameter sweep files (YAML/JSON)
- Launch dozens of instances simultaneously
- Monitor sweep progress
- Collect results
- ML hyperparameter tuning workflows
- Batch data processing

**Prerequisites:** [Tutorial 2: Your First Instance](02-first-instance.md)

---

### [Tutorial 4: Job Arrays](04-job-arrays.md)
**Duration:** 30 minutes | **Level:** Intermediate

Launch hundreds of identical instances in parallel:
- Launch job arrays (100s of instances)
- Use array indices for task distribution
- Process data files in parallel
- Monte Carlo simulations
- Cost optimization strategies

**Prerequisites:** [Tutorial 3: Parameter Sweeps](03-parameter-sweeps.md)

---

### [Tutorial 5: Batch Queues](05-batch-queues.md)
**Duration:** 45 minutes | **Level:** Intermediate

Run sequential jobs with dependencies:
- Create queue configuration files
- Define job dependencies (DAGs)
- Set retry strategies and timeouts
- Monitor queue execution
- Build ML training pipelines
- ETL workflows

**Prerequisites:** [Tutorial 4: Job Arrays](04-job-arrays.md)

---

## Advanced (Operations & Cost Management)

### [Tutorial 6: Cost Management](06-cost-management.md)
**Duration:** 20 minutes | **Level:** Intermediate

Track and optimize AWS costs:
- Track costs for instances and sweeps
- Set budget alerts
- Analyze spending patterns
- Optimize costs with spot instances
- Right-size instance types
- Avoid unexpected bills

**Prerequisites:** [Tutorial 2: Your First Instance](02-first-instance.md)

---

### [Tutorial 7: Monitoring & Alerts](07-monitoring-alerts.md)
**Duration:** 30 minutes | **Level:** Intermediate

Set up monitoring and notifications:
- Monitor instance status and health
- Set up Slack/Discord/Email alerts
- Create cost threshold alerts
- Monitor parameter sweeps
- Debug failed instances

**Prerequisites:** [Tutorial 3: Parameter Sweeps](03-parameter-sweeps.md)

---

## Learning Paths

### Path 1: Quick Start (Get Running Fast)
Perfect for first-time users who want to launch instances quickly.

1. [Tutorial 1: Getting Started](01-getting-started.md) - 15 min
2. [Tutorial 2: Your First Instance](02-first-instance.md) - 20 min

**Total:** 35 minutes

---

### Path 2: ML/Research Workflows
For machine learning and research users running parameter sweeps.

1. [Tutorial 1: Getting Started](01-getting-started.md) - 15 min
2. [Tutorial 2: Your First Instance](02-first-instance.md) - 20 min
3. [Tutorial 3: Parameter Sweeps](03-parameter-sweeps.md) - 30 min
4. [Tutorial 6: Cost Management](06-cost-management.md) - 20 min
5. [Tutorial 7: Monitoring & Alerts](07-monitoring-alerts.md) - 30 min

**Total:** 1 hour 55 minutes

---

### Path 3: Batch Processing
For data engineers and batch processing workloads.

1. [Tutorial 1: Getting Started](01-getting-started.md) - 15 min
2. [Tutorial 2: Your First Instance](02-first-instance.md) - 20 min
3. [Tutorial 4: Job Arrays](04-job-arrays.md) - 30 min
4. [Tutorial 5: Batch Queues](05-batch-queues.md) - 45 min
5. [Tutorial 6: Cost Management](06-cost-management.md) - 20 min

**Total:** 2 hours 10 minutes

---

### Path 4: Complete Mastery
Complete all tutorials for comprehensive understanding.

1. [Tutorial 1: Getting Started](01-getting-started.md) - 15 min
2. [Tutorial 2: Your First Instance](02-first-instance.md) - 20 min
3. [Tutorial 3: Parameter Sweeps](03-parameter-sweeps.md) - 30 min
4. [Tutorial 4: Job Arrays](04-job-arrays.md) - 30 min
5. [Tutorial 5: Batch Queues](05-batch-queues.md) - 45 min
6. [Tutorial 6: Cost Management](06-cost-management.md) - 20 min
7. [Tutorial 7: Monitoring & Alerts](07-monitoring-alerts.md) - 30 min

**Total:** 3 hours 10 minutes

---

## What's Next?

After completing tutorials, explore:

🛠️ **[How-To Guides](../how-to/)** - Task-oriented recipes for specific scenarios

📚 **[Command Reference](../reference/)** - Complete command documentation

💡 **[FAQ](../FAQ.md)** - Common questions and troubleshooting

📖 **[Main Documentation](../README.md)** - Full documentation index

---

## Tutorial Format

Each tutorial follows a consistent structure:

**Header:**
- Duration estimate
- Difficulty level
- Prerequisites

**Content:**
- "What You'll Learn" section
- Step-by-step instructions with code examples
- Expected outputs
- Practical exercises
- Real-world examples
- Best practices
- Troubleshooting tips

**Footer:**
- "What You Learned" summary
- Practice exercises
- Next steps and related resources
- Quick reference

---

## Feedback

Found an issue or have suggestions? [Open an issue](https://github.com/scttfrdmn/spore-host/issues/new?labels=type:docs,component:spawn)
