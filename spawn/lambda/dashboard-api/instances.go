package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AWS regions to query
var awsRegions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"eu-west-1",
	"eu-west-2",
	"eu-central-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-northeast-1",
}

// listInstances queries all regions in parallel and returns spawn-managed instances
// Filters by spawn:iam-user tag to enforce per-user isolation
func listInstances(ctx context.Context, cfg aws.Config, cliIamArn string) ([]InstanceInfo, error) {
	var wg sync.WaitGroup
	instancesChan := make(chan []InstanceInfo, len(awsRegions))
	errorsChan := make(chan error, len(awsRegions))

	// Query all regions in parallel
	for _, region := range awsRegions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()

			// Get EC2 client for region with cross-account credentials
			ec2Client, err := getEC2ClientForRegion(ctx, cfg, r)
			if err != nil {
				errorsChan <- fmt.Errorf("region %s: %w", r, err)
				return
			}

			// Filter by spawn:managed and spawn:iam-user tags (per-user isolation)
			filters := []types.Filter{
				{
					Name:   aws.String("tag:spawn:managed"),
					Values: []string{"true"},
				},
				{
					Name:   aws.String("tag:spawn:iam-user"),
					Values: []string{cliIamArn},
				},
			}

			// Query EC2
			result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				Filters: filters,
			})
			if err != nil {
				errorsChan <- fmt.Errorf("region %s: %w", r, err)
				return
			}

			// Convert reservations to InstanceInfo
			instances := convertReservationsToInstances(result.Reservations, r, "")
			if len(instances) > 0 {
				instancesChan <- instances
			}
		}(region)
	}

	// Wait for all goroutines
	go func() {
		wg.Wait()
		close(instancesChan)
		close(errorsChan)
	}()

	// Collect results
	var allInstances []InstanceInfo
	for instances := range instancesChan {
		allInstances = append(allInstances, instances...)
	}

	// Collect errors (don't fail on region errors, just log)
	for err := range errorsChan {
		fmt.Printf("Warning: %v\n", err)
	}

	// Sort by launch time (newest first)
	sort.Slice(allInstances, func(i, j int) bool {
		return allInstances[i].LaunchTime.After(allInstances[j].LaunchTime)
	})

	return allInstances, nil
}

// getInstance gets a single instance by ID
func getInstance(ctx context.Context, cfg aws.Config, instanceID, cliIamArn string) (*InstanceInfo, error) {
	// Try all regions in parallel until we find the instance
	var wg sync.WaitGroup
	instanceChan := make(chan *InstanceInfo, 1)
	foundMutex := sync.Mutex{}
	found := false

	for _, region := range awsRegions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()

			// Check if already found
			foundMutex.Lock()
			if found {
				foundMutex.Unlock()
				return
			}
			foundMutex.Unlock()

			// Get EC2 client for region
			ec2Client, err := getEC2ClientForRegion(ctx, cfg, r)
			if err != nil {
				return
			}

			// Query EC2 for specific instance
			result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{instanceID},
			})
			if err != nil {
				// Instance not in this region, continue
				return
			}

			if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
				return
			}

			// Found instance, verify it belongs to the calling user (per-user isolation)
			instance := result.Reservations[0].Instances[0]
			instanceUserID := getTagValue(instance.Tags, "spawn:iam-user")

			if instanceUserID != cliIamArn {
				// Instance exists but doesn't belong to this user
				return
			}

			// Convert to InstanceInfo
			instances := convertReservationsToInstances(result.Reservations, r, "")
			if len(instances) > 0 {
				foundMutex.Lock()
				if !found {
					found = true
					instanceChan <- &instances[0]
				}
				foundMutex.Unlock()
			}
		}(region)
	}

	// Wait for all goroutines
	go func() {
		wg.Wait()
		close(instanceChan)
	}()

	// Return first found instance
	if instance := <-instanceChan; instance != nil {
		return instance, nil
	}

	return nil, fmt.Errorf("instance not found or access denied")
}

// convertReservationsToInstances converts EC2 Reservations to InstanceInfo
func convertReservationsToInstances(reservations []types.Reservation, region, accountBase36 string) []InstanceInfo {
	var instances []InstanceInfo

	for _, reservation := range reservations {
		for _, instance := range reservation.Instances {
			info := InstanceInfo{
				InstanceID:       aws.ToString(instance.InstanceId),
				InstanceType:     string(instance.InstanceType),
				State:            string(instance.State.Name),
				Region:           region,
				AvailabilityZone: aws.ToString(instance.Placement.AvailabilityZone),
				PublicIP:         aws.ToString(instance.PublicIpAddress),
				PrivateIP:        aws.ToString(instance.PrivateIpAddress),
				LaunchTime:       aws.ToTime(instance.LaunchTime),
				SpotInstance:     instance.InstanceLifecycle == types.InstanceLifecycleTypeSpot,
				KeyName:          aws.ToString(instance.KeyName),
				Tags:             make(map[string]string),
			}

			// Extract name from tags
			info.Name = getTagValue(instance.Tags, "Name")

			// Extract spawn-specific tags
			info.TTL = getTagValue(instance.Tags, "spawn:ttl")
			info.IdleTimeout = getTagValue(instance.Tags, "spawn:idle-timeout")

			// Calculate TTL remaining (if TTL tag exists)
			if info.TTL != "" {
				ttlDuration, err := time.ParseDuration(info.TTL)
				if err == nil {
					elapsed := time.Since(info.LaunchTime)
					remaining := ttlDuration - elapsed
					if remaining > 0 {
						info.TTLRemainingSeconds = int(remaining.Seconds())
					} else {
						info.TTLRemainingSeconds = 0
					}
				}
			}

			// Construct DNS name
			dnsName := getTagValue(instance.Tags, "spawn:dns-name")
			if dnsName != "" {
				info.DNSName = getFullDNSName(dnsName, accountBase36)
			}

			// Copy all tags
			for _, tag := range instance.Tags {
				if tag.Key != nil && tag.Value != nil {
					info.Tags[*tag.Key] = *tag.Value
				}
			}

			instances = append(instances, info)
		}
	}

	return instances
}

// getTagValue extracts a tag value by key
func getTagValue(tags []types.Tag, key string) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == key && tag.Value != nil {
			return *tag.Value
		}
	}
	return ""
}
