# Glossary

## autoscale group

A named set of EC2 instances managed as a unit with automatic scaling rules. The autoscale controller adds or removes instances based on queue depth, metrics, or schedules. See [Autoscaling](../features/autoscaling.md).

## ephemeral instance

An EC2 instance designed for temporary use with a defined TTL. Spawned instances are ephemeral — they terminate automatically when their TTL expires or when they are idle for too long.

## hibernation

A stop mode in which EC2 writes the instance's RAM contents to EBS before stopping. On wake, the instance resumes exactly where it left off, preserving in-memory state, open file handles, and running processes. Not all instance types or AMIs support hibernation.

## idle detection

spored monitors CPU, memory, and network utilization at regular intervals. If all metrics stay below configured thresholds for the idle timeout duration, the instance is considered idle and terminates. Prevents abandoned instances from accumulating costs.

## instance profile

An IAM role attached to an EC2 instance at launch time, granting the instance permission to call AWS APIs. spawn instances use an instance profile to register DNS, write metrics, and clean up resources.

## job array

A set of EC2 instances launched together to run the same script with different inputs. Each instance in the array receives a unique job index. See [Parameter Sweeps](../features/parameter-sweeps.md).

## parameter sweep

A job that runs many instances in parallel, each with a different combination of input parameters. Used for hyperparameter optimization, sensitivity analysis, and simulation grids.

## spawn

The CLI tool used to launch ephemeral EC2 instances. Handles AMI selection, networking, SSH key management, and spored agent installation.

## spored

The monitoring daemon installed on each spawned EC2 instance. Responsibilities: TTL enforcement, idle detection, spot interruption handling, DNS registration, metrics reporting. Runs as a systemd service.

## spot instance

An EC2 instance that runs on unused capacity at up to 70-90% savings compared to on-demand pricing. Can be interrupted by AWS with 2 minutes notice. spored handles interruptions gracefully.

## TTL (Time To Live)

The duration after which an instance automatically terminates. Configurable at launch (`--ttl 4h`). Can be extended at any time (`spawn extend my-instance 2h`). spored warns via logs 10 minutes before TTL expiry.

## truffle

The companion CLI tool for discovering and querying running instances across regions and AWS accounts. Provides `truffle ls`, `truffle get`, and `truffle cost` commands.

## user-data

An EC2 feature that runs a script during instance initialization. spawn uses user-data to install and configure the spored agent.
