package autoscaler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AutoScaler is the main reconciliation engine
type AutoScaler struct {
	config             *Config
	dbClient           *Client
	healthChecker      *HealthChecker
	capacityReconciler *CapacityReconciler
	policyEvaluator    *PolicyEvaluator
	metricEvaluator    *MetricEvaluator
	drainManager       *DrainManager
}

// NewAutoScaler creates a new autoscaler
func NewAutoScaler(config *Config) *AutoScaler {
	dbClient := NewClient(config.DynamoClient, config.TableName)
	healthChecker := NewHealthChecker(config.EC2Client, config.DynamoClient, config.RegistryTable)
	capacityReconciler := NewCapacityReconciler(config.EC2Client)
	policyEvaluator := NewPolicyEvaluator(config.SQSClient)
	metricEvaluator := NewMetricEvaluator(config.CloudWatchClient)
	drainManager := NewDrainManager(config.EC2Client, config.RegistryTable)

	return &AutoScaler{
		config:             config,
		dbClient:           dbClient,
		healthChecker:      healthChecker,
		capacityReconciler: capacityReconciler,
		policyEvaluator:    policyEvaluator,
		metricEvaluator:    metricEvaluator,
		drainManager:       drainManager,
	}
}

// Reconcile reconciles a single autoscale group
func (a *AutoScaler) Reconcile(ctx context.Context, groupID string) error {
	// Load group
	group, err := a.dbClient.GetGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("load group: %w", err)
	}

	// Skip non-active groups
	if group.Status != "active" {
		log.Printf("skipping group %s (status: %s)", group.GroupName, group.Status)
		return nil
	}

	// Evaluate scaling policy (if present)
	if group.ScalingPolicy != nil {
		newDesired, queueDepth, changed, err := a.policyEvaluator.EvaluatePolicy(ctx, group)
		if err != nil {
			log.Printf("policy evaluation failed for %s: %v", group.GroupName, err)
		} else if changed {
			log.Printf("[%s] policy triggered: %d → %d (queue: %d msgs)",
				group.GroupName, group.DesiredCapacity, newDesired, queueDepth)

			oldDesired := group.DesiredCapacity
			group.DesiredCapacity = newDesired

			// Update scaling state
			if group.ScalingState == nil {
				group.ScalingState = &ScalingState{}
			}
			group.ScalingState.LastQueueDepth = queueDepth
			group.ScalingState.LastCalculatedCapacity = newDesired
			if newDesired > oldDesired {
				group.ScalingState.LastScaleUp = time.Now()
			} else {
				group.ScalingState.LastScaleDown = time.Now()
			}
		}
	}

	log.Printf("reconciling group %s (desired: %d)", group.GroupName, group.DesiredCapacity)

	// Discover current instances
	instances, err := a.discoverInstances(ctx, group)
	if err != nil {
		return fmt.Errorf("discover instances: %w", err)
	}

	log.Printf("found %d instances for group %s", len(instances), group.GroupName)

	// Evaluate metric policy (if present and no queue policy)
	// Metric policy requires existing instances to query metrics
	if group.MetricPolicy != nil && group.ScalingPolicy == nil {
		newDesired, metricValue, changed, err := a.metricEvaluator.EvaluateMetricPolicy(ctx, group, instances)
		if err != nil {
			log.Printf("metric policy evaluation failed for %s: %v", group.GroupName, err)
		} else if changed {
			log.Printf("[%s] metric policy triggered: %d → %d (metric: %.2f)",
				group.GroupName, group.DesiredCapacity, newDesired, metricValue)

			oldDesired := group.DesiredCapacity
			group.DesiredCapacity = newDesired

			// Update scaling state
			if group.ScalingState == nil {
				group.ScalingState = &ScalingState{}
			}
			group.ScalingState.LastCalculatedCapacity = newDesired
			if newDesired > oldDesired {
				group.ScalingState.LastScaleUp = time.Now()
			} else {
				group.ScalingState.LastScaleDown = time.Now()
			}
		}
	}

	// Handle draining instances (if drain enabled)
	if group.DrainConfig != nil && group.DrainConfig.Enabled {
		drainingInstances, err := a.drainManager.GetDrainingInstances(ctx, group.AutoScaleGroupID)
		if err != nil {
			log.Printf("error getting draining instances for %s: %v", group.GroupName, err)
		} else if len(drainingInstances) > 0 {
			log.Printf("found %d draining instances for %s", len(drainingInstances), group.GroupName)

			// Check which draining instances are ready to terminate
			readyToTerminate, err := a.drainManager.CheckDrainStatus(ctx, drainingInstances, group.DrainConfig)
			if err != nil {
				log.Printf("error checking drain status for %s: %v", group.GroupName, err)
			} else if len(readyToTerminate) > 0 {
				log.Printf("terminating %d drained instances for %s", len(readyToTerminate), group.GroupName)

				// Terminate drained instances
				if err := a.capacityReconciler.TerminateInstances(ctx, readyToTerminate); err != nil {
					log.Printf("error terminating drained instances: %v", err)
				}

				// Clear drain state
				if err := a.drainManager.ClearDrainState(ctx, readyToTerminate); err != nil {
					log.Printf("error clearing drain state: %v", err)
				}
			}
		}
	}

	// Check health
	health, err := a.healthChecker.CheckInstances(ctx, group.JobArrayID, instances)
	if err != nil {
		return fmt.Errorf("check health: %w", err)
	}

	// Plan capacity changes
	plan, err := a.capacityReconciler.PlanCapacity(ctx, group, health)
	if err != nil {
		return fmt.Errorf("plan capacity: %w", err)
	}

	log.Printf("capacity plan for %s: current=%d, desired=%d, healthy=%d, unhealthy=%d, launch=%d, terminate=%d",
		group.GroupName, plan.CurrentCapacity, plan.DesiredCapacity,
		plan.HealthyCount, plan.UnhealthyCount, plan.ToLaunch, len(plan.ToTerminate))

	// Execute plan if changes needed
	if plan.ToLaunch > 0 || len(plan.ToTerminate) > 0 {
		if err := a.capacityReconciler.ExecutePlanWithDrain(ctx, group, plan, a.drainManager, group.DrainConfig); err != nil {
			return fmt.Errorf("execute plan: %w", err)
		}

		// Update last scale event
		group.LastScaleEvent = time.Now()
	}

	// Save updated group
	if err := a.dbClient.UpdateGroup(ctx, group); err != nil {
		return fmt.Errorf("update group: %w", err)
	}

	return nil
}

// ReconcileAll reconciles all active autoscale groups
func (a *AutoScaler) ReconcileAll(ctx context.Context) error {
	groups, err := a.dbClient.ListActiveGroups(ctx)
	if err != nil {
		return fmt.Errorf("list groups: %w", err)
	}

	log.Printf("found %d active autoscale groups", len(groups))

	for _, group := range groups {
		if err := a.Reconcile(ctx, group.AutoScaleGroupID); err != nil {
			log.Printf("failed to reconcile group %s: %v", group.GroupName, err)
		}
	}

	return nil
}

// discoverInstances finds all instances for an autoscale group
func (a *AutoScaler) discoverInstances(ctx context.Context, group *AutoScaleGroup) ([]string, error) {
	// Query EC2 for instances with autoscale-group tag
	result, err := a.config.EC2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:spawn:autoscale-group"),
				Values: []string{group.AutoScaleGroupID},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "stopping", "stopped"},
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

// GetGroup retrieves a group by ID
func (a *AutoScaler) GetGroup(ctx context.Context, groupID string) (*AutoScaleGroup, error) {
	return a.dbClient.GetGroup(ctx, groupID)
}

// GetGroupByName retrieves a group by name
func (a *AutoScaler) GetGroupByName(ctx context.Context, name string) (*AutoScaleGroup, error) {
	return a.dbClient.GetGroupByName(ctx, name)
}

// CreateGroup creates a new autoscale group
func (a *AutoScaler) CreateGroup(ctx context.Context, group *AutoScaleGroup) error {
	return a.dbClient.CreateGroup(ctx, group)
}

// UpdateGroup updates an existing autoscale group
func (a *AutoScaler) UpdateGroup(ctx context.Context, group *AutoScaleGroup) error {
	return a.dbClient.UpdateGroup(ctx, group)
}

// DeleteGroup deletes an autoscale group
func (a *AutoScaler) DeleteGroup(ctx context.Context, groupID string) error {
	return a.dbClient.DeleteGroup(ctx, groupID)
}

// ListActiveGroups lists all active groups
func (a *AutoScaler) ListActiveGroups(ctx context.Context) ([]*AutoScaleGroup, error) {
	return a.dbClient.ListActiveGroups(ctx)
}
