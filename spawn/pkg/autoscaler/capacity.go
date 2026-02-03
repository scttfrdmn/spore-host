package autoscaler

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// CapacityReconciler manages instance capacity
type CapacityReconciler struct {
	ec2Client *ec2.Client
}

// NewCapacityReconciler creates a new capacity reconciler
func NewCapacityReconciler(ec2Client *ec2.Client) *CapacityReconciler {
	return &CapacityReconciler{
		ec2Client: ec2Client,
	}
}

// PlanCapacity determines what capacity changes are needed
func (c *CapacityReconciler) PlanCapacity(
	ctx context.Context,
	group *AutoScaleGroup,
	health []HealthStatus,
) (*CapacityPlan, error) {
	plan := &CapacityPlan{
		DesiredCapacity: group.DesiredCapacity,
		ToTerminate:     make([]string, 0),
	}

	// Count healthy, unhealthy, pending
	for _, h := range health {
		if h.Healthy {
			if h.EC2State == "pending" {
				plan.PendingCount++
			} else {
				plan.HealthyCount++
			}
		} else {
			plan.UnhealthyCount++
			plan.ToTerminate = append(plan.ToTerminate, h.InstanceID)
		}
	}

	plan.CurrentCapacity = plan.HealthyCount + plan.PendingCount

	// Calculate how many to launch
	delta := plan.DesiredCapacity - plan.CurrentCapacity
	if delta > 0 {
		plan.ToLaunch = delta
	}

	return plan, nil
}

// ExecutePlan executes the capacity plan
func (c *CapacityReconciler) ExecutePlan(
	ctx context.Context,
	group *AutoScaleGroup,
	plan *CapacityPlan,
) error {
	// Launch new instances first
	if plan.ToLaunch > 0 {
		if err := c.launchInstances(ctx, group, plan.ToLaunch); err != nil {
			return fmt.Errorf("launch instances: %w", err)
		}
		log.Printf("launched %d instances for group %s", plan.ToLaunch, group.GroupName)
	}

	// Terminate unhealthy instances
	for _, instanceID := range plan.ToTerminate {
		if err := c.terminateInstance(ctx, instanceID); err != nil {
			log.Printf("failed to terminate %s: %v", instanceID, err)
		} else {
			log.Printf("terminated unhealthy instance %s", instanceID)
		}
	}

	return nil
}

// launchInstances launches new instances according to the launch template
func (c *CapacityReconciler) launchInstances(
	ctx context.Context,
	group *AutoScaleGroup,
	count int,
) error {
	lt := group.LaunchTemplate

	// Build tag specifications
	tags := []types.Tag{
		{Key: aws.String("Name"), Value: aws.String(group.GroupName)},
		{Key: aws.String("spawn:managed"), Value: aws.String("true")},
		{Key: aws.String("spawn:job-array-id"), Value: aws.String(group.JobArrayID)},
		{Key: aws.String("spawn:autoscale-group"), Value: aws.String(group.AutoScaleGroupID)},
		{Key: aws.String("spawn:managed-by"), Value: aws.String("autoscaler")},
	}

	// Add custom tags from launch template
	for k, v := range lt.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(lt.AMI),
		InstanceType: types.InstanceType(lt.InstanceType),
		MinCount:     aws.Int32(int32(count)),
		MaxCount:     aws.Int32(int32(count)),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         tags,
			},
		},
	}

	// Optional fields
	if lt.KeyName != "" {
		input.KeyName = aws.String(lt.KeyName)
	}
	if lt.SubnetID != "" {
		input.SubnetId = aws.String(lt.SubnetID)
	}
	if len(lt.SecurityGroups) > 0 {
		input.SecurityGroupIds = lt.SecurityGroups
	}
	if lt.IAMInstanceProfile != "" {
		input.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Name: aws.String(lt.IAMInstanceProfile),
		}
	}
	if lt.UserData != "" {
		input.UserData = aws.String(lt.UserData)
	}

	// Spot instances
	if lt.Spot {
		input.InstanceMarketOptions = &types.InstanceMarketOptionsRequest{
			MarketType: types.MarketTypeSpot,
			SpotOptions: &types.SpotMarketOptions{
				SpotInstanceType: types.SpotInstanceTypeOneTime,
			},
		}
	}

	_, err := c.ec2Client.RunInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("run instances: %w", err)
	}

	// Instances will register themselves via spored agent

	return nil
}

// terminateInstance terminates an unhealthy instance
func (c *CapacityReconciler) terminateInstance(ctx context.Context, instanceID string) error {
	_, err := c.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("terminate instance: %w", err)
	}

	return nil
}
