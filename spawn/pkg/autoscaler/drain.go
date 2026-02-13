package autoscaler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// DrainConfig defines graceful drain behavior
type DrainConfig struct {
	Enabled        bool          `dynamodbav:"enabled"`                  // Enable graceful drain
	TimeoutSeconds int           `dynamodbav:"timeout_seconds"`          // Max time to wait for drain (default: 300)
	CheckInterval  time.Duration `dynamodbav:"check_interval,omitempty"` // How often to check drain status
}

// DrainState tracks instance drain status
type DrainState struct {
	InstanceID  string
	StartedAt   time.Time
	TimeoutAt   time.Time
	HasActiveWork bool
}

// DrainManager handles graceful instance draining
type DrainManager struct {
	ec2Client     *ec2.Client
	registryTable string
}

// NewDrainManager creates a new drain manager
func NewDrainManager(ec2Client *ec2.Client, registryTable string) *DrainManager {
	return &DrainManager{
		ec2Client:     ec2Client,
		registryTable: registryTable,
	}
}

// MarkForDrain tags instances for graceful drain
func (d *DrainManager) MarkForDrain(ctx context.Context, instanceIDs []string) error {
	if len(instanceIDs) == 0 {
		return nil
	}

	// Tag instances with drain state
	_, err := d.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: instanceIDs,
		Tags: []ec2types.Tag{
			{
				Key:   aws.String("spawn:drain-state"),
				Value: aws.String("draining"),
			},
			{
				Key:   aws.String("spawn:drain-started"),
				Value: aws.String(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("tag instances for drain: %w", err)
	}

	log.Printf("marked %d instances for drain: %v", len(instanceIDs), instanceIDs)
	return nil
}

// GetDrainingInstances returns instances currently in drain state
func (d *DrainManager) GetDrainingInstances(ctx context.Context, groupID string) ([]string, error) {
	result, err := d.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:spawn:autoscale-group"),
				Values: []string{groupID},
			},
			{
				Name:   aws.String("tag:spawn:drain-state"),
				Values: []string{"draining"},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "stopping"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	instanceIDs := make([]string, 0)
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil {
				instanceIDs = append(instanceIDs, aws.ToString(instance.InstanceId))
			}
		}
	}

	return instanceIDs, nil
}

// CheckDrainStatus checks if draining instances are ready for termination
func (d *DrainManager) CheckDrainStatus(ctx context.Context, instanceIDs []string, config *DrainConfig) ([]string, error) {
	if len(instanceIDs) == 0 {
		return nil, nil
	}

	// Get instance details with drain tags
	result, err := d.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("describe instances: %w", err)
	}

	readyToTerminate := make([]string, 0)
	now := time.Now()

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			instanceID := aws.ToString(instance.InstanceId)

			// Get drain start time from tags
			var drainStarted time.Time
			for _, tag := range instance.Tags {
				if aws.ToString(tag.Key) == "spawn:drain-started" {
					if t, err := time.Parse(time.RFC3339, aws.ToString(tag.Value)); err == nil {
						drainStarted = t
					}
					break
				}
			}

			if drainStarted.IsZero() {
				// No drain start time, skip
				continue
			}

			// Calculate timeout
			timeout := time.Duration(config.TimeoutSeconds) * time.Second
			if config.TimeoutSeconds == 0 {
				timeout = 300 * time.Second // Default 5 minutes
			}

			// Check if timeout exceeded
			if now.Sub(drainStarted) > timeout {
				log.Printf("instance %s drain timeout exceeded (%.0fs), ready for termination",
					instanceID, now.Sub(drainStarted).Seconds())
				readyToTerminate = append(readyToTerminate, instanceID)
				continue
			}

			// Check if instance has active work
			// For now, we'll use a simple heuristic: if heartbeat is old, no active work
			// This will be enhanced with proper job tracking in the future
			hasActiveWork, err := d.hasActiveWork(ctx, instanceID)
			if err != nil {
				log.Printf("error checking active work for %s: %v", instanceID, err)
				continue
			}

			if !hasActiveWork {
				log.Printf("instance %s has no active work, ready for termination", instanceID)
				readyToTerminate = append(readyToTerminate, instanceID)
			} else {
				log.Printf("instance %s still draining (active work detected)", instanceID)
			}
		}
	}

	return readyToTerminate, nil
}

// hasActiveWork checks if an instance has active work (simplified version)
// In a full implementation, this would query the job registry for active jobs
func (d *DrainManager) hasActiveWork(ctx context.Context, instanceID string) (bool, error) {
	// TODO: Query DynamoDB registry table for active jobs on this instance
	// For now, assume no active work after a short grace period
	// This is a placeholder that can be enhanced with proper job tracking

	// In practice, you would:
	// 1. Query registry for heartbeats from this instance
	// 2. Check if any jobs are running (job-status = "running")
	// 3. Verify heartbeat is recent (< 2 minutes old)

	return false, nil
}

// ClearDrainState removes drain tags from instances
func (d *DrainManager) ClearDrainState(ctx context.Context, instanceIDs []string) error {
	if len(instanceIDs) == 0 {
		return nil
	}

	_, err := d.ec2Client.DeleteTags(ctx, &ec2.DeleteTagsInput{
		Resources: instanceIDs,
		Tags: []ec2types.Tag{
			{Key: aws.String("spawn:drain-state")},
			{Key: aws.String("spawn:drain-started")},
		},
	})

	if err != nil {
		return fmt.Errorf("clear drain tags: %w", err)
	}

	return nil
}

// GetDefaultDrainConfig returns default drain configuration
func GetDefaultDrainConfig() *DrainConfig {
	return &DrainConfig{
		Enabled:        false, // Disabled by default for backward compatibility
		TimeoutSeconds: 300,   // 5 minutes
		CheckInterval:  30 * time.Second,
	}
}
