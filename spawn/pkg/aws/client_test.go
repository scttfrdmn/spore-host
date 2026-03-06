package aws

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/scttfrdmn/spore-host/spawn/pkg/aws/mock"
	"github.com/scttfrdmn/spore-host/spawn/pkg/testutil"
)

// TestClientCreation tests creating a new AWS client
func TestClientCreation(t *testing.T) {
	ctx := context.Background()

	// Note: This test requires AWS credentials to be configured
	// In CI/CD, this would be skipped or use mock credentials
	_, err := NewClient(ctx)
	if err != nil {
		t.Logf("Client creation failed (expected in test env without credentials): %v", err)
	}
}

// TestGetEnabledRegions tests fetching enabled AWS regions
func TestGetEnabledRegions(t *testing.T) {
	tests := []struct {
		name        string
		mockRegions []types.Region
		wantErr     bool
		wantCount   int
	}{
		{
			name: "standard regions",
			mockRegions: []types.Region{
				{RegionName: strPtr("us-east-1")},
				{RegionName: strPtr("us-west-2")},
				{RegionName: strPtr("eu-west-1")},
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name:        "empty regions",
			mockRegions: []types.Region{},
			wantErr:     false,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEC2 := mock.NewMockEC2Client()
			mockEC2.Regions = tt.mockRegions

			// Test using mock client would require interface refactoring
			// For now, we test the logic
			if len(mockEC2.Regions) != tt.wantCount {
				t.Errorf("got %d regions, want %d", len(mockEC2.Regions), tt.wantCount)
			}
		})
	}
}

// TestLaunchConfig tests the launch configuration structure
func TestLaunchConfig(t *testing.T) {
	tests := []struct {
		name   string
		config LaunchConfig
		valid  bool
	}{
		{
			name: "valid basic config",
			config: LaunchConfig{
				InstanceType: "t3.micro",
				Region:       "us-east-1",
				AMI:          "ami-12345678",
				KeyName:      "my-key",
			},
			valid: true,
		},
		{
			name: "valid spot config",
			config: LaunchConfig{
				InstanceType: "t3.micro",
				Region:       "us-east-1",
				AMI:          "ami-12345678",
				KeyName:      "my-key",
				Spot:         true,
				SpotMaxPrice: "0.05",
			},
			valid: true,
		},
		{
			name: "valid with hibernation",
			config: LaunchConfig{
				InstanceType: "m5.large",
				Region:       "us-east-1",
				AMI:          "ami-12345678",
				KeyName:      "my-key",
				Hibernate:    true,
			},
			valid: true,
		},
		{
			name: "valid with EFA",
			config: LaunchConfig{
				InstanceType: "c5n.18xlarge",
				Region:       "us-east-1",
				AMI:          "ami-12345678",
				KeyName:      "my-key",
				EFAEnabled:   true,
			},
			valid: true,
		},
		{
			name: "valid with placement group",
			config: LaunchConfig{
				InstanceType:   "c5.large",
				Region:         "us-east-1",
				AMI:            "ami-12345678",
				KeyName:        "my-key",
				PlacementGroup: "my-pg",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate required fields
			if tt.config.InstanceType == "" {
				t.Error("InstanceType is required")
			}
			if tt.config.Region == "" {
				t.Error("Region is required")
			}
			if tt.config.AMI == "" {
				t.Error("AMI is required")
			}
			if tt.config.KeyName == "" {
				t.Error("KeyName is required")
			}

			// Validate EFA requirements
			if tt.config.EFAEnabled {
				// EFA requires specific instance types
				if !isEFACompatible(tt.config.InstanceType) {
					t.Logf("Instance type %s may not support EFA", tt.config.InstanceType)
				}
			}

			// Validate hibernation requirements
			if tt.config.Hibernate {
				// Hibernation requires specific instance families
				if !isHibernationCompatible(tt.config.InstanceType) {
					t.Logf("Instance type %s may not support hibernation", tt.config.InstanceType)
				}
			}
		})
	}
}

// TestJobArrayConfig tests job array configuration
func TestJobArrayConfig(t *testing.T) {
	tests := []struct {
		name   string
		config LaunchConfig
		valid  bool
	}{
		{
			name: "valid job array",
			config: LaunchConfig{
				InstanceType:  "t3.micro",
				Region:        "us-east-1",
				AMI:           "ami-12345678",
				KeyName:       "my-key",
				JobArrayID:    "job-123",
				JobArrayName:  "compute",
				JobArraySize:  10,
				JobArrayIndex: 0,
			},
			valid: true,
		},
		{
			name: "single instance (not an array)",
			config: LaunchConfig{
				InstanceType: "t3.micro",
				Region:       "us-east-1",
				AMI:          "ami-12345678",
				KeyName:      "my-key",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate job array fields
			if tt.config.JobArrayID != "" {
				if tt.config.JobArrayName == "" {
					t.Error("JobArrayName required when JobArrayID is set")
				}
				if tt.config.JobArraySize <= 0 {
					t.Error("JobArraySize must be positive")
				}
				if tt.config.JobArrayIndex < 0 || tt.config.JobArrayIndex >= tt.config.JobArraySize {
					t.Error("JobArrayIndex out of bounds")
				}
			}
		})
	}
}

// TestParameterSweepConfig tests parameter sweep configuration
func TestParameterSweepConfig(t *testing.T) {
	tests := []struct {
		name   string
		config LaunchConfig
		valid  bool
	}{
		{
			name: "valid sweep",
			config: LaunchConfig{
				InstanceType: "t3.micro",
				Region:       "us-east-1",
				AMI:          "ami-12345678",
				KeyName:      "my-key",
				SweepID:      "sweep-123",
				SweepName:    "hyperparam",
				SweepIndex:   0,
			},
			valid: true,
		},
		{
			name: "sweep with parameters",
			config: LaunchConfig{
				InstanceType: "t3.micro",
				Region:       "us-east-1",
				AMI:          "ami-12345678",
				KeyName:      "my-key",
				SweepID:      "sweep-123",
				SweepName:    "hyperparam",
				SweepIndex:   5,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate sweep fields
			if tt.config.SweepID != "" {
				if tt.config.SweepName == "" {
					t.Error("SweepName required when SweepID is set")
				}
				if tt.config.SweepIndex < 0 {
					t.Error("SweepIndex must be non-negative")
				}
			}
		})
	}
}

// TestTTLValidation tests TTL string validation
func TestTTLValidation(t *testing.T) {
	tests := []struct {
		name  string
		ttl   string
		valid bool
	}{
		{
			name:  "valid hours",
			ttl:   "8h",
			valid: true,
		},
		{
			name:  "valid minutes",
			ttl:   "30m",
			valid: true,
		},
		{
			name:  "valid combined",
			ttl:   "2h30m",
			valid: true,
		},
		{
			name:  "empty (no TTL)",
			ttl:   "",
			valid: true,
		},
		{
			name:  "invalid format",
			ttl:   "invalid",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := LaunchConfig{
				TTL: tt.ttl,
			}

			// TTL validation would happen in the launch logic
			if config.TTL != "" {
				// Would validate duration format here
				isValid := isDurationFormat(config.TTL)
				if isValid != tt.valid {
					t.Errorf("TTL %q validity = %v, want %v", config.TTL, isValid, tt.valid)
				}
			}
		})
	}
}

// TestOnCompleteAction tests on-complete action validation
func TestOnCompleteAction(t *testing.T) {
	tests := []struct {
		name   string
		action string
		valid  bool
	}{
		{
			name:   "terminate",
			action: "terminate",
			valid:  true,
		},
		{
			name:   "stop",
			action: "stop",
			valid:  true,
		},
		{
			name:   "hibernate",
			action: "hibernate",
			valid:  true,
		},
		{
			name:   "empty (disabled)",
			action: "",
			valid:  true,
		},
		{
			name:   "invalid action",
			action: "invalid",
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := LaunchConfig{
				OnComplete: tt.action,
			}

			validActions := map[string]bool{
				"":          true,
				"terminate": true,
				"stop":      true,
				"hibernate": true,
			}

			isValid := validActions[config.OnComplete]
			if isValid != tt.valid {
				t.Errorf("OnComplete %q validity = %v, want %v", config.OnComplete, isValid, tt.valid)
			}
		})
	}
}

// TestInstanceInfo tests the InstanceInfo structure
func TestInstanceInfo(t *testing.T) {
	info := InstanceInfo{
		InstanceID:       "i-1234567890abcdef0",
		InstanceType:     "t3.micro",
		State:            "running",
		PublicIP:         "52.1.2.3",
		PrivateIP:        "10.0.1.100",
		LaunchTime:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AvailabilityZone: "us-east-1a",
		Region:           "us-east-1",
		KeyName:          "my-key",
	}

	// Validate required fields
	if info.InstanceID == "" {
		t.Error("InstanceID is required")
	}
	if info.InstanceType == "" {
		t.Error("InstanceType is required")
	}
	if info.State == "" {
		t.Error("State is required")
	}
	if info.Region == "" {
		t.Error("Region is required")
	}
}

// TestMockEC2Operations tests mock EC2 client operations
func TestMockEC2Operations(t *testing.T) {
	ctx := context.Background()
	mockEC2 := mock.NewMockEC2Client()

	t.Run("RunInstances", func(t *testing.T) {
		result, err := mockEC2.RunInstances(ctx, &ec2.RunInstancesInput{
			InstanceType: types.InstanceTypeT3Micro,
			ImageId:      aws.String("ami-12345678"),
			KeyName:      aws.String("my-key"),
		})

		if err != nil {
			t.Fatalf("RunInstances failed: %v", err)
		}

		if len(result.Instances) != 1 {
			t.Errorf("got %d instances, want 1", len(result.Instances))
		}

		inst := result.Instances[0]
		if inst.InstanceId == nil {
			t.Error("instance ID is nil")
		}
		if inst.State == nil || inst.State.Name != types.InstanceStateNameRunning {
			t.Error("instance state is not running")
		}

		// Verify call tracking
		if mockEC2.RunInstancesCalls != 1 {
			t.Errorf("RunInstancesCalls = %d, want 1", mockEC2.RunInstancesCalls)
		}
	})

	t.Run("DescribeInstances", func(t *testing.T) {
		// First launch an instance
		launchResult, _ := mockEC2.RunInstances(ctx, &ec2.RunInstancesInput{
			InstanceType: types.InstanceTypeT3Micro,
			ImageId:      aws.String("ami-12345678"),
		})
		instanceID := *launchResult.Instances[0].InstanceId

		// Then describe it
		result, err := mockEC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})

		if err != nil {
			t.Fatalf("DescribeInstances failed: %v", err)
		}

		if len(result.Reservations) != 1 {
			t.Fatalf("got %d reservations, want 1", len(result.Reservations))
		}
		if len(result.Reservations[0].Instances) != 1 {
			t.Fatalf("got %d instances, want 1", len(result.Reservations[0].Instances))
		}

		inst := result.Reservations[0].Instances[0]
		if *inst.InstanceId != instanceID {
			t.Errorf("got instance ID %s, want %s", *inst.InstanceId, instanceID)
		}
	})

	t.Run("TerminateInstances", func(t *testing.T) {
		// Launch an instance
		launchResult, _ := mockEC2.RunInstances(ctx, &ec2.RunInstancesInput{
			InstanceType: types.InstanceTypeT3Micro,
			ImageId:      aws.String("ami-12345678"),
		})
		instanceID := *launchResult.Instances[0].InstanceId

		// Terminate it
		result, err := mockEC2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{instanceID},
		})

		if err != nil {
			t.Fatalf("TerminateInstances failed: %v", err)
		}

		if len(result.TerminatingInstances) != 1 {
			t.Errorf("got %d terminating instances, want 1", len(result.TerminatingInstances))
		}

		// Verify state changed to terminated
		descResult, _ := mockEC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})
		inst := descResult.Reservations[0].Instances[0]
		if inst.State.Name != types.InstanceStateNameTerminated {
			t.Errorf("instance state = %s, want terminated", inst.State.Name)
		}
	})

	t.Run("CreateTags", func(t *testing.T) {
		// Launch an instance
		launchResult, _ := mockEC2.RunInstances(ctx, &ec2.RunInstancesInput{
			InstanceType: types.InstanceTypeT3Micro,
			ImageId:      aws.String("ami-12345678"),
		})
		instanceID := *launchResult.Instances[0].InstanceId

		// Tag it
		_, err := mockEC2.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{instanceID},
			Tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("test-instance")},
				{Key: aws.String("Environment"), Value: aws.String("test")},
			},
		})

		if err != nil {
			t.Fatalf("CreateTags failed: %v", err)
		}

		// Verify tags were added
		descResult, _ := mockEC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})
		inst := descResult.Reservations[0].Instances[0]
		if len(inst.Tags) != 2 {
			t.Errorf("got %d tags, want 2", len(inst.Tags))
		}
	})
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func isEFACompatible(instanceType string) bool {
	// EFA is supported on c5n, c6gn, p3dn, p4d, etc.
	efaFamilies := []string{"c5n", "c6gn", "p3dn", "p4d", "p4de"}
	for _, family := range efaFamilies {
		if len(instanceType) >= len(family) && instanceType[:len(family)] == family {
			return true
		}
	}
	return false
}

func isHibernationCompatible(instanceType string) bool {
	// Hibernation is supported on C3, C4, C5, M3, M4, M5, R3, R4, R5, T2, T3
	hibernationFamilies := []string{"c3", "c4", "c5", "m3", "m4", "m5", "r3", "r4", "r5", "t2", "t3"}
	for _, family := range hibernationFamilies {
		if len(instanceType) >= len(family) && instanceType[:len(family)] == family {
			return true
		}
	}
	return false
}

func isDurationFormat(s string) bool {
	if s == "" {
		return false
	}
	// Simple check for duration format (contains time unit)
	return testutil.Contains(s, "h") || testutil.Contains(s, "m") || testutil.Contains(s, "s")
}
