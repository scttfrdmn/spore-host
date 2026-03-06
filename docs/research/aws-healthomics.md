# AWS HealthOmics Research & Analysis

**Research Date:** 2026-01-27
**Issue:** #75
**Prepared For:** spawn/spore-host project evaluation

---

## Executive Summary

AWS HealthOmics is a **fully managed, HIPAA-eligible service** for genomics and bioinformatics workflows, announced by AWS and replacing the deprecated Amazon Genomics CLI. It provides three integrated components: **Workflows** (execution), **Storage** (data management), and **Analytics** (querying).

**Key Finding:** HealthOmics and spawn serve **complementary use cases** with minimal overlap. HealthOmics is optimized for genomics-specific workflows with managed infrastructure, while spawn provides flexible, general-purpose compute orchestration with user control.

**Recommendation:** **Monitor and document** HealthOmics as a specialized alternative for genomics workloads, but **do not integrate**. spawn's value proposition remains strong for users needing flexibility, cost control, and non-genomics workflows.

---

## Service Overview

### What is AWS HealthOmics?

AWS HealthOmics accelerates **clinical diagnostic testing, drug discovery, and agricultural research** by fully managing bioinformatics workflow infrastructure. It abstracts away compute provisioning, workflow engine management, and data storage optimization.

**Service Launch:** November 2022 (as "Amazon Omics"), rebranded to "AWS HealthOmics"
**Status:** Generally Available (GA), actively maintained
**Replaces:** Amazon Genomics CLI (EOL May 2024)

### Three Core Components

| Component | Purpose | Key Features |
|-----------|---------|--------------|
| **Workflows** | Execute bioinformatics pipelines | WDL, Nextflow, CWL support; 100K+ concurrent vCPUs; auto-scaling |
| **Storage** | Store genomic data efficiently | Sequence stores (FASTQ, BAM, CRAM); variant stores; reference genomes |
| **Analytics** | Query and analyze at scale | Zero-ETL transformation; Athena integration; population-scale queries |

---

## Workflow Orchestration Deep Dive

### Supported Workflow Languages

| Language | Versions Supported | Notes |
|----------|-------------------|-------|
| **Nextflow** | v24.10.8, v23.10.0, v22.04.0 | Stable releases only; no "edge" versions |
| **WDL** | 1.0, 1.1, development | Workflow Description Language (Broad Institute) |
| **CWL** | 1.0, 1.1, 1.2 | Common Workflow Language |

**NOT Supported:**
- ❌ **Snakemake** - Not supported by HealthOmics (was supported by deprecated AGC)
- ❌ **Cromwell** - Engine, not language (WDL execution engine)

### Execution Model

**Private Workflows:**
- User-defined custom workflows
- Upload workflow definition (WDL/Nextflow/CWL file)
- Specify Docker images via Amazon ECR
- Pay per compute instance used (per-second billing, 60s minimum)

**Ready2Run Workflows:**
- Pre-built AWS bioinformatics pipelines
- Fixed cost per successful run (e.g., $10/run for GATK-BP Germline fq2vcf)
- Zero configuration required
- Examples: GATK Best Practices, DRAGEN pipelines, Isaac Aligner

### Compute Architecture

**Key Architectural Details:**
- Uses **"omics instances"** (AWS-managed compute layer)
- Abstracts underlying infrastructure (likely EC2-based, but not exposed)
- No direct access to AWS Batch, Fargate, or raw EC2
- Fully managed orchestration (no user control over scheduling)

**Instance Types:**
```
Standard:    omics.c.large, omics.c.xlarge, omics.c.2xlarge, ...
Compute:     omics.c.* (balanced CPU/memory)
Memory:      omics.r.* (memory-optimized)
Accelerated: omics.g.* (GPU instances)
```

**Scaling Capabilities:**
- **100,000+ concurrent vCPUs** per account
- Automatic task-level parallelization
- Dynamic instance selection (smallest instance fitting task requirements)
- No manual capacity planning required

**Run Groups (Optional):**
- Group related workflow runs
- Set max vCPUs, max duration, concurrent run limits
- Priority scheduling within group

---

## Storage Capabilities

### Sequence Store (Genomic Data)

**Purpose:** Store FASTQ, BAM, CRAM files with compression and metadata

**Features:**
- Charged per **gigabase/month** (1 billion DNA bases)
- Two storage tiers:
  - **Active:** $0.005769/gigabase/month (EU-West-Ireland example)
  - **Archive:** $0.001154/gigabase/month (5x cheaper)
- Automatic compression and format optimization
- Metadata cataloging and search
- AWS KMS encryption

**Data Model:**
- **Read sets:** Abstraction of genomic reads (FASTQ, BAM, CRAM)
- Attribute-based access control (ABAC)
- Granular permissions per read set

### Reference Store

**Purpose:** Store reference genomes (FASTA format)

**Features:**
- One reference store per account/region
- Used for read mapping and alignment
- Compression and tiering included
- No per-request charges for reference access

### Variant & Annotation Stores (Analytics)

**Purpose:** Zero-ETL transformation for querying variant data

**Features:**
- Ingest VCF/gVCF files → query-optimized format
- Population-scale variant queries via Amazon Athena
- TSV/CSV annotation database import
- 30-day minimum storage duration

---

## Pricing Analysis

### Workflow Execution Costs

**Private Workflows (Per-Instance Billing):**

| Instance Type | vCPU | Memory | Price/Hour | Use Case |
|---------------|------|--------|-----------|----------|
| omics.c.large | 2 | 4 GB | $0.1134 | Small tasks |
| omics.c.4xlarge | 16 | 32 GB | $0.918 | Standard pipelines |
| omics.r.8xlarge | 32 | 256 GB | $2.7216 | Memory-intensive |
| omics.m.xlarge | 4 | 16 GB | $0.2592 | Balanced workloads |

- Billed per second (60-second minimum per task)
- No data staging charges (S3 imports/exports)
- No cross-AZ transfer fees

**Ready2Run Workflows (Flat-Rate):**
- GATK-BP Germline fq2vcf (30x genome): **$10.00/run**
- Fixed cost regardless of duration
- Incomplete runs: Prorated if cancelled within 1 hour

**Cost Comparison Example:**

| Scenario | HealthOmics Cost | spawn Cost (EC2 Direct) | Notes |
|----------|------------------|-------------------------|-------|
| 4-hour WGS pipeline (16 vCPU, 32GB) | $3.67 | ~$0.50 (c6i.4xlarge spot) | HealthOmics 7x more expensive |
| Ready2Run GATK pipeline | $10.00 | ~$2-3 (custom build) | HealthOmics 3-4x premium |
| Long-running (48h+) | High cost | Minimal (spot + auto-terminate) | spawn advantage increases |

**Key Insight:** HealthOmics trades **cost efficiency for managed simplicity**. spawn remains significantly cheaper for cost-conscious users.

### Storage Costs

**Sequence Store Pricing:**
- Active: $0.005769/gigabase/month (~$17/30x genome/month)
- Archive: $0.001154/gigabase/month (~$3.50/30x genome/month)

**Comparison to S3:**
- S3 Standard: ~$0.023/GB/month (~$69/30x genome/month at 100GB compressed)
- HealthOmics compression: Achieves ~25% size reduction vs raw S3
- **Winner:** HealthOmics for long-term genomic storage

---

## Compliance & Security

### Certifications

| Framework | Status | Details |
|-----------|--------|---------|
| **HIPAA** | ✅ Eligible | BAA available via AWS Artifact |
| **FedRAMP** | ✅ Authorized | Moderate/High baselines supported |
| **GxP** | ✅ Supported | Audit trails, validation support |
| **SOC 2** | ✅ Certified | Part of AWS compliance program |
| **CLIA** | ⚠️ Not Applicable | Lab operations, not cloud infrastructure |
| **HITRUST** | ❓ Unknown | Not explicitly documented |

### Security Features

✅ **Data encryption:** At-rest (AWS KMS), in-transit (TLS)
✅ **Access control:** IAM, ABAC for granular permissions
✅ **Audit logging:** CloudTrail integration for all API calls
✅ **Network isolation:** VPC endpoints available
✅ **Provenance tracking:** Full workflow execution history

### Important Limitations

⚠️ **NOT a medical device** - No FDA clearance
⚠️ **NOT for clinical decisions** - Human review required
⚠️ **Data management only** - Not diagnostic or therapeutic

**Key Takeaway:** HealthOmics provides **compliance-ready infrastructure** but users still bear responsibility for compliance attestation (same as spawn).

---

## Regional Availability

**Generally Available Regions (Active by Default):**
- us-east-1 (N. Virginia)
- us-west-2 (Oregon)
- eu-west-1 (Ireland)
- eu-west-2 (London)
- eu-central-1 (Frankfurt)
- ap-southeast-1 (Singapore)

**Opt-In Regions:** Available upon account activation

**Limitation:** Not available in all AWS regions (unlike spawn, which works wherever EC2 is available)

---

## Integration with Other AWS Services

| Service | Integration | Use Case |
|---------|------------|----------|
| **Amazon ECR** | Required | Docker image hosting for custom workflows |
| **Amazon S3** | Deep integration | Input/output staging, intermediate storage |
| **AWS Lake Formation** | Access control | Permissions for analytics data stores |
| **Amazon Athena** | Direct query | SQL queries on variant stores |
| **Amazon SageMaker** | Notebook access | Run workflows from Jupyter notebooks |
| **EventBridge** | Event-driven | Trigger workflows on data arrival |
| **Step Functions** | Higher-level orchestration | Chain multiple HealthOmics workflows |
| **AWS KMS** | Encryption | Sequence store encryption keys |

**Example Architecture Pattern:**
```
S3 (raw FASTQ) → HealthOmics Workflow → Sequence Store → Analytics Store → Athena Query
```

---

## spawn vs AWS HealthOmics: Comparison Matrix

### Feature Comparison

| Feature | spawn | AWS HealthOmics | Winner |
|---------|-------|----------------|--------|
| **Workflow Languages** | Any (CLI integration) | WDL, Nextflow, CWL only | spawn (flexibility) |
| **Compute Cost** | EC2 spot (~70% savings) | Managed instances (premium) | **spawn (7x cheaper)** |
| **Setup Complexity** | Medium (one-time config) | Low (fully managed) | HealthOmics (ease) |
| **Infrastructure Control** | Full (instance types, spot, etc.) | None (abstracted) | spawn (control) |
| **Scaling** | Manual (job arrays, sweeps) | Automatic (100K+ vCPUs) | HealthOmics (scale) |
| **Storage Optimization** | Standard S3 | Genomics-optimized | HealthOmics (genomics) |
| **Analytics Integration** | Manual (query S3 or export) | Native (Athena, zero-ETL) | HealthOmics (analytics) |
| **Use Case Breadth** | General-purpose HPC | Genomics/bioinformatics only | **spawn (versatility)** |
| **Compliance Documentation** | DIY | Built-in audit trails | HealthOmics (compliance) |
| **Workflow Engine Mgmt** | User-managed | Fully managed | HealthOmics (managed) |
| **Data Provenance** | Manual tracking | Automatic | HealthOmics (provenance) |
| **Region Availability** | All EC2 regions | Limited regions | spawn (coverage) |
| **Vendor Lock-in** | Low (standard EC2/S3) | High (proprietary format) | **spawn (portability)** |
| **Cost Transparency** | Full (EC2 pricing) | Opaque (managed premium) | spawn (transparency) |

### When to Use Each

**Use AWS HealthOmics When:**
- ✅ Running **genomics-specific workflows** (WDL, Nextflow, CWL)
- ✅ Need **zero infrastructure management** (fully managed)
- ✅ Require **compliance-ready audit trails** out-of-box
- ✅ Working with **petabyte-scale genomic data**
- ✅ Need **population-scale variant analytics**
- ✅ Budget allows **managed service premium** (3-7x cost)
- ✅ Team lacks DevOps expertise

**Use spawn When:**
- ✅ Running **any type of compute workload** (genomics, ML, simulations, etc.)
- ✅ Need **maximum cost efficiency** (spot instances, auto-termination)
- ✅ Want **full infrastructure control** (instance types, networking, storage)
- ✅ Using **Snakemake or other unsupported engines**
- ✅ Require **multi-region flexibility** (spawn works everywhere)
- ✅ Building **custom orchestration** (CI/CD, workflow tools)
- ✅ Have DevOps capacity for infrastructure management
- ✅ Need **transparent, predictable costs**

**Hybrid Use Case:**
- Use **HealthOmics for production clinical pipelines** (compliance, audit trails)
- Use **spawn for research, development, and exploratory analyses** (cost, flexibility)

---

## Integration Opportunities & Gaps

### Potential Integration Points

**1. spawn → HealthOmics Workflow Launcher**
- Add `spawn healthomics run` command
- Submit WDL/Nextflow workflows to HealthOmics API
- Monitor run status and retrieve results
- **Value:** spawn CLI becomes single interface for both execution models

**2. HealthOmics → spawn Compute Backend**
- NOT FEASIBLE: HealthOmics does not support custom compute backends
- HealthOmics uses proprietary "omics instances" (no BYO compute)

**3. Hybrid Data Pipeline**
- Use spawn for **data preprocessing** (QC, alignment) on spot instances
- Upload to HealthOmics **Sequence Store** for long-term storage
- Run HealthOmics **Analytics** for population-scale queries
- **Value:** Cost-effective preprocessing + managed analytics

**4. spawn as EventBridge Trigger**
- Use spawn to launch ancillary analyses when HealthOmics workflow completes
- EventBridge → Lambda → spawn API call
- **Value:** Extend HealthOmics with custom post-processing

### Gaps HealthOmics Doesn't Cover

❌ **Non-genomics workloads** (ML training, simulations, rendering)
❌ **Snakemake workflows** (not supported)
❌ **Custom orchestration patterns** (no access to underlying scheduler)
❌ **Cost optimization for dev/test** (managed premium unsuitable for iteration)
❌ **Multi-cloud portability** (AWS-only, proprietary storage format)
❌ **Fine-grained resource control** (can't specify exact instance types)

### spawn's Unique Value Propositions

✅ **Universal workflow engine support** (Snakemake, Cromwell, Luigi, Airflow, Prefect, etc.)
✅ **Massive cost savings** (70-90% via spot instances)
✅ **Zero vendor lock-in** (standard EC2 + S3)
✅ **Full transparency** (see exactly what you're paying for)
✅ **Multi-region by default** (works in all EC2 regions)
✅ **General-purpose HPC** (not limited to genomics)

---

## Competitive Positioning

### Market Segmentation

| Segment | Recommended Solution | Rationale |
|---------|---------------------|-----------|
| **Clinical Diagnostics Labs** | HealthOmics | HIPAA/CLIA compliance, audit trails, managed service |
| **Academic Research** | **spawn** | Cost-sensitive, flexible, multi-workflow support |
| **Pharma Drug Discovery** | Hybrid | HealthOmics for GxP pipelines, spawn for exploratory |
| **Agricultural Genomics** | HealthOmics | Purpose-built for crop genomics at scale |
| **Bioinformatics Core Facilities** | **spawn** | Diverse user needs, cost accountability, flexibility |
| **Contract Research Orgs** | Hybrid | HealthOmics for client deliverables, spawn for internal |

### spawn's Competitive Advantages

**vs HealthOmics:**
1. **7x lower compute costs** (spot instances + auto-termination)
2. **Universal workflow support** (not limited to WDL/Nextflow/CWL)
3. **Full control** (instance types, networking, storage)
4. **Transparent pricing** (no hidden managed service fees)
5. **Multi-cloud ready** (standard infrastructure patterns)

**vs Amazon Genomics CLI (deprecated):**
- spawn is **actively maintained** (AGC EOL'd May 2024)
- spawn supports **broader use cases** beyond genomics
- spawn has **better cost optimization** (spot, idle detection, hibernation)

**vs AWS Batch:**
- spawn provides **higher-level abstractions** (sweeps, job arrays)
- spawn has **automatic cost tracking** and budgets
- spawn includes **DNS management** (spore.host)
- spawn offers **integrated monitoring/alerting**

---

## Use Case Analysis

### 1. Population Genomics (100K+ samples)

**Best Choice:** AWS HealthOmics

**Rationale:**
- Sequence Store: Petabyte-scale storage with compression
- Analytics: Population-scale variant queries via Athena
- Managed scaling: 100K+ concurrent vCPUs
- Compliance: Built-in audit trails for research publications

**spawn Role:** Development and pipeline optimization (cheaper for iteration)

---

### 2. Clinical Diagnostic Testing (CAP/CLIA labs)

**Best Choice:** AWS HealthOmics

**Rationale:**
- HIPAA BAA available
- Complete audit trails (required for CAP inspection)
- Data provenance tracking
- Predictable cost-per-sample (billing model)

**spawn Role:** None (compliance overhead too high)

---

### 3. Drug Discovery (Target Identification)

**Best Choice:** Hybrid (spawn + HealthOmics)

**Rationale:**
- Use spawn for **exploratory analyses** (ChIP-seq, RNA-seq, ATAC-seq)
- Use HealthOmics for **GxP-compliant pipelines** (regulatory submissions)
- Cost: spawn for bulk compute (70% cheaper)
- Compliance: HealthOmics for validated workflows

---

### 4. Agricultural Genomics (Plant/Animal Breeding)

**Best Choice:** HealthOmics (with spawn for cost-conscious groups)

**Rationale:**
- HealthOmics purpose-built for agricultural research
- Handles massive reference genomes (wheat, pine, etc.)
- Compliance less critical (no HIPAA/CLIA)
- spawn viable if budget-constrained

---

### 5. Bioinformatics Methods Development

**Best Choice:** **spawn**

**Rationale:**
- Rapid iteration (cheap spot instances)
- Flexible infrastructure (test different instance types)
- No workflow language constraints (prototype in Python/R)
- Cost-effective for failed experiments
- Full control over compute environment

---

### 6. Multi-Omics Integration (Genomics + Proteomics + Metabolomics)

**Best Choice:** **spawn**

**Rationale:**
- HealthOmics optimized for genomics only
- spawn handles heterogeneous workloads (mass spec, imaging, etc.)
- Custom orchestration across domains
- No vendor lock-in to genomics-specific format

---

## Performance Characteristics

### Throughput Comparison

| Metric | spawn | HealthOmics |
|--------|-------|-------------|
| **Max concurrent vCPUs** | Limited by AWS account quota | 100,000+ |
| **Launch time** | ~90 seconds (EC2 launch) | ~60 seconds (omics instance) |
| **Scaling responsiveness** | Manual (sweep concurrency) | Automatic (task-level) |
| **Data staging** | S3 (user-managed) | Optimized (zero-charge) |
| **Job queue depth** | Unlimited (DynamoDB-backed) | Run groups (user-defined limits) |

**Verdict:** HealthOmics wins on **automatic scaling**, spawn wins on **flexibility**.

### Cost-Performance Trade-offs

**Example: 30x Whole Genome Sequencing (WGS)**

| Solution | Duration | Cost | Cost/Genome |
|----------|----------|------|-------------|
| HealthOmics Ready2Run | 4 hours | $10 | $10 |
| HealthOmics Private Workflow | 4 hours | ~$3.67 | $3.67 |
| spawn (c6i.4xlarge spot) | 4 hours | ~$0.50 | $0.50 |
| spawn (optimized, preemptible) | 3.5 hours | ~$0.35 | $0.35 |

**At Scale (1000 genomes/month):**
- HealthOmics Ready2Run: **$10,000/month**
- HealthOmics Private: **$3,670/month**
- spawn: **$350-500/month** (7-10x cheaper)

**Break-even Analysis:**
- HealthOmics worth premium if DevOps cost > $3,000/month
- For most academic labs: spawn is clear winner
- For clinical labs: HealthOmics compliance value justifies cost

---

## Technical Limitations & Drawbacks

### AWS HealthOmics Limitations

❌ **Limited workflow engines:** WDL, Nextflow, CWL only (no Snakemake, Cromwell, custom)
❌ **No custom compute:** Cannot use reserved instances, savings plans, or custom AMIs
❌ **Opaque pricing:** Cannot predict costs for new workflows (trial required)
❌ **Regional restrictions:** Not available in all AWS regions
❌ **Vendor lock-in:** Sequence Store uses proprietary format (migration difficult)
❌ **No SSH access:** Cannot debug failed tasks interactively
❌ **Managed-only:** No option to self-host or run on-prem

### spawn Limitations (for Genomics Use Case)

❌ **No genomics-optimized storage:** Uses standard S3 (less efficient)
❌ **Manual provenance tracking:** No built-in audit trails
❌ **Infrastructure management:** User responsible for EC2, networking, etc.
❌ **No population-scale analytics:** No Athena integration for variant queries
❌ **Compliance documentation:** User must build audit trails manually

---

## Recommendations

### Primary Recommendation: **MONITOR, DO NOT INTEGRATE**

**Rationale:**
1. **Minimal overlap:** HealthOmics targets managed genomics, spawn targets flexible HPC
2. **Complementary positioning:** HealthOmics for production, spawn for research/dev
3. **Integration complexity:** HealthOmics API integration adds minimal value
4. **spawn's unique value intact:** Cost, flexibility, and control remain key differentiators

### Secondary Actions

**1. Document HealthOmics as Alternative**
- Add section to spawn documentation: "When to Use AWS HealthOmics Instead"
- Position spawn as **development/research** tool, HealthOmics as **production/clinical**
- Highlight cost comparison (7x difference)

**2. Target Distinct User Segments**
- **spawn:** Academic labs, bioinformatics core facilities, method developers
- **HealthOmics:** Clinical labs, pharma, agriculture companies, HIPAA/CLIA environments

**3. Maintain Cost Efficiency Advantage**
- Continue optimizing spot instance usage (spawn's killer feature)
- Emphasize workflow engine flexibility (Snakemake, custom Python)
- Highlight transparent pricing (no hidden managed fees)

**4. Future Reassessment Triggers**
- IF HealthOmics adds Snakemake support → Re-evaluate integration
- IF HealthOmics allows BYO compute → Consider hybrid architecture
- IF HealthOmics pricing drops 50%+ → Reassess cost advantage
- IF spawn adds compliance features → Compete more directly

### Do NOT Build

❌ **spawn → HealthOmics API integration** - Low value, high complexity
❌ **Sequence Store compatibility** - Proprietary format, vendor lock-in
❌ **HealthOmics workflow converter** - Limited audience, niche use case

---

## Conclusion

AWS HealthOmics is a **powerful, specialized service** for genomics pipelines that require managed infrastructure, compliance-ready audit trails, and population-scale analytics. However, it comes at a **significant cost premium** (3-7x) and **limited flexibility** (WDL/Nextflow/CWL only, no Snakemake).

**spawn's value proposition remains strong:**
- **7x lower compute costs** (spot instances + ephemeral compute)
- **Universal workflow support** (any engine, any workload)
- **Full infrastructure control** (instance types, networking, regions)
- **Zero vendor lock-in** (standard AWS services)
- **Transparent, predictable pricing**

**Recommended Strategy:** Position spawn as the **cost-effective, flexible alternative** for research and development, while acknowledging HealthOmics as the **managed, compliance-ready solution** for production clinical/diagnostic use cases. The two services are **complementary, not competitive**.

---

## Sources

- [AWS HealthOmics Overview](https://aws.amazon.com/healthomics/)
- [What is AWS HealthOmics? - Documentation](https://docs.aws.amazon.com/omics/latest/dev/what-is-healthomics.html)
- [AWS HealthOmics Features](https://aws.amazon.com/healthomics/features/)
- [AWS HealthOmics Pricing](https://aws.amazon.com/healthomics/pricing/)
- [Orchestrating Multiple AWS HealthOmics Workflows at Scale](https://aws.amazon.com/blogs/industries/orchestrating-multiple-aws-healthomics-workflows-at-scale/)
- [Version support for HealthOmics workflow definition languages](https://docs.aws.amazon.com/omics/latest/dev/workflows-lang-versions.html)
- [Compliance validation for AWS HealthOmics](https://docs.aws.amazon.com/omics/latest/dev/compliance-validation.html)
- [Healthcare Compliance on AWS](https://aws.amazon.com/health/healthcare-compliance/)
- [Amazon Genomics CLI (Deprecated)](https://github.com/aws/amazon-genomics-cli)
- [Genomic Data Analysis - HealthOmics Resources](https://aws.amazon.com/healthomics/resources/)
