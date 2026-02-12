package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/scttfrdmn/mycelium/spawn/pkg/autoscaler"
	"github.com/spf13/cobra"
)

var autoscaleCmd = &cobra.Command{
	Use:   "autoscale",
	Short: "Manage auto-scaling job arrays",
	Long:  "Launch and manage auto-scaling job arrays that maintain target capacity",
}

var (
	// Launch flags
	autoscaleName           string
	autoscaleJobArrayID     string
	autoscaleDesired        int
	autoscaleMin            int
	autoscaleMax            int
	autoscaleInstanceType   string
	autoscaleAMI            string
	autoscaleSpot           bool
	autoscaleKeyName        string
	autoscaleSubnetID       string
	autoscaleSecurityGroups []string
	autoscaleIAMProfile     string
	autoscaleUserData       string
	autoscaleTags           map[string]string

	// Update flags
	autoscaleNewDesired int
	autoscaleNewMin     int
	autoscaleNewMax     int

	// Scaling policy flags
	scalingPolicy             string
	queueURL                  string
	targetMessagesPerInstance int
	scaleUpCooldown           int
	scaleDownCooldown         int
	removePolicyFlag          bool

	// Global flags
	autoscaleTableName string
	autoscaleEnv       string
)

var autoscaleLaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch an auto-scaling job array",
	Long:  "Launch a new auto-scaling job array with specified capacity and launch template",
	RunE:  runAutoscaleLaunch,
}

var autoscaleUpdateCmd = &cobra.Command{
	Use:   "update <group-name>",
	Short: "Update auto-scaling group capacity",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoscaleUpdate,
}

var autoscaleStatusCmd = &cobra.Command{
	Use:   "status [group-name]",
	Short: "Show auto-scaling group status",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAutoscaleStatus,
}

var autoscaleHealthCmd = &cobra.Command{
	Use:   "health <group-name>",
	Short: "Show instance health for auto-scaling group",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoscaleHealth,
}

var autoscalePauseCmd = &cobra.Command{
	Use:   "pause <group-name>",
	Short: "Pause auto-scaling (stop reconciliation)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoscalePause,
}

var autoscaleResumeCmd = &cobra.Command{
	Use:   "resume <group-name>",
	Short: "Resume auto-scaling",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoscaleResume,
}

var autoscaleTerminateCmd = &cobra.Command{
	Use:   "terminate <group-name>",
	Short: "Terminate auto-scaling group and all instances",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoscaleTerminate,
}

var autoscaleSetPolicyCmd = &cobra.Command{
	Use:   "set-policy <group-name>",
	Short: "Set or update scaling policy for an autoscale group",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoscaleSetPolicy,
}

var autoscaleScalingActivityCmd = &cobra.Command{
	Use:   "scaling-activity <group-name>",
	Short: "Show recent scaling activity for an autoscale group",
	Args:  cobra.ExactArgs(1),
	RunE:  runAutoscaleScalingActivity,
}

func init() {
	rootCmd.AddCommand(autoscaleCmd)
	autoscaleCmd.AddCommand(autoscaleLaunchCmd)
	autoscaleCmd.AddCommand(autoscaleUpdateCmd)
	autoscaleCmd.AddCommand(autoscaleStatusCmd)
	autoscaleCmd.AddCommand(autoscaleHealthCmd)
	autoscaleCmd.AddCommand(autoscalePauseCmd)
	autoscaleCmd.AddCommand(autoscaleResumeCmd)
	autoscaleCmd.AddCommand(autoscaleTerminateCmd)
	autoscaleCmd.AddCommand(autoscaleSetPolicyCmd)
	autoscaleCmd.AddCommand(autoscaleScalingActivityCmd)

	// Global flags
	autoscaleCmd.PersistentFlags().StringVar(&autoscaleTableName, "table", "spawn-autoscale-groups", "DynamoDB table name")
	autoscaleCmd.PersistentFlags().StringVar(&autoscaleEnv, "env", "production", "Environment (production or staging)")

	// Launch flags
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleName, "name", "", "Group name (required)")
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleJobArrayID, "job-array-id", "", "Job array ID (auto-generated if not specified)")
	autoscaleLaunchCmd.Flags().IntVar(&autoscaleDesired, "desired-capacity", 0, "Desired instance count (required)")
	autoscaleLaunchCmd.Flags().IntVar(&autoscaleMin, "min-capacity", 0, "Minimum instance count (default: 0)")
	autoscaleLaunchCmd.Flags().IntVar(&autoscaleMax, "max-capacity", 0, "Maximum instance count (default: desired * 2)")
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleInstanceType, "instance-type", "", "EC2 instance type (required)")
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleAMI, "ami", "", "AMI ID (required)")
	autoscaleLaunchCmd.Flags().BoolVar(&autoscaleSpot, "spot", false, "Use spot instances")
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleKeyName, "key-name", "", "SSH key name")
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleSubnetID, "subnet-id", "", "Subnet ID")
	autoscaleLaunchCmd.Flags().StringSliceVar(&autoscaleSecurityGroups, "security-groups", []string{}, "Security group IDs")
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleIAMProfile, "iam-profile", "", "IAM instance profile")
	autoscaleLaunchCmd.Flags().StringVar(&autoscaleUserData, "user-data", "", "User data script (base64 encoded)")
	autoscaleLaunchCmd.Flags().StringToStringVar(&autoscaleTags, "tags", map[string]string{}, "Additional tags (key=value)")

	autoscaleLaunchCmd.Flags().StringVar(&scalingPolicy, "scaling-policy", "",
		"Scaling policy type: 'queue-depth' (empty = manual mode)")
	autoscaleLaunchCmd.Flags().StringVar(&queueURL, "queue-url", "",
		"SQS queue URL for queue-depth policy (required if --scaling-policy=queue-depth)")
	autoscaleLaunchCmd.Flags().IntVar(&targetMessagesPerInstance, "target-messages-per-instance", 10,
		"Target messages per instance for queue-depth scaling")
	autoscaleLaunchCmd.Flags().IntVar(&scaleUpCooldown, "scale-up-cooldown", 60,
		"Scale-up cooldown in seconds")
	autoscaleLaunchCmd.Flags().IntVar(&scaleDownCooldown, "scale-down-cooldown", 300,
		"Scale-down cooldown in seconds")

	autoscaleLaunchCmd.MarkFlagRequired("name")
	autoscaleLaunchCmd.MarkFlagRequired("desired-capacity")
	autoscaleLaunchCmd.MarkFlagRequired("instance-type")
	autoscaleLaunchCmd.MarkFlagRequired("ami")

	// Update flags
	autoscaleUpdateCmd.Flags().IntVar(&autoscaleNewDesired, "desired-capacity", -1, "New desired capacity")
	autoscaleUpdateCmd.Flags().IntVar(&autoscaleNewMin, "min-capacity", -1, "New minimum capacity")
	autoscaleUpdateCmd.Flags().IntVar(&autoscaleNewMax, "max-capacity", -1, "New maximum capacity")

	// Set-policy flags
	autoscaleSetPolicyCmd.Flags().StringVar(&scalingPolicy, "scaling-policy", "",
		"Scaling policy type: 'queue-depth'")
	autoscaleSetPolicyCmd.Flags().StringVar(&queueURL, "queue-url", "",
		"SQS queue URL for queue-depth policy")
	autoscaleSetPolicyCmd.Flags().IntVar(&targetMessagesPerInstance, "target-messages-per-instance", 10,
		"Target messages per instance for queue-depth scaling")
	autoscaleSetPolicyCmd.Flags().IntVar(&scaleUpCooldown, "scale-up-cooldown", 60,
		"Scale-up cooldown in seconds")
	autoscaleSetPolicyCmd.Flags().IntVar(&scaleDownCooldown, "scale-down-cooldown", 300,
		"Scale-down cooldown in seconds")
	autoscaleSetPolicyCmd.Flags().BoolVar(&removePolicyFlag, "none", false,
		"Remove scaling policy (revert to manual mode)")
}

func getAutoscaler(ctx context.Context) (*autoscaler.AutoScaler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	// Build table name with environment suffix
	tableName := fmt.Sprintf("%s-%s", autoscaleTableName, autoscaleEnv)

	ec2Client := ec2.NewFromConfig(cfg)
	dynamoClient := dynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)

	return autoscaler.NewAutoScaler(&autoscaler.Config{
		EC2Client:     ec2Client,
		DynamoClient:  dynamoClient,
		SQSClient:     sqsClient,
		TableName:     tableName,
		RegistryTable: "spawn-hybrid-registry",
	}), nil
}

func runAutoscaleLaunch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate inputs
	if autoscaleDesired < 1 {
		return fmt.Errorf("desired-capacity must be at least 1")
	}

	// Validate scaling policy flags
	if scalingPolicy != "" {
		if scalingPolicy != "queue-depth" {
			return fmt.Errorf("invalid scaling policy: %s (only 'queue-depth' supported)", scalingPolicy)
		}
		if queueURL == "" {
			return fmt.Errorf("--queue-url required when --scaling-policy is set")
		}
	}

	// Set defaults
	if autoscaleMin < 0 {
		autoscaleMin = 0
	}
	if autoscaleMax <= 0 {
		autoscaleMax = autoscaleDesired * 2
	}
	if autoscaleJobArrayID == "" {
		autoscaleJobArrayID = fmt.Sprintf("%s-%d", autoscaleName, time.Now().Unix())
	}

	// Validate capacity ranges
	if autoscaleMin > autoscaleDesired {
		return fmt.Errorf("min-capacity cannot exceed desired-capacity")
	}
	if autoscaleMax < autoscaleDesired {
		return fmt.Errorf("max-capacity cannot be less than desired-capacity")
	}

	// Decode user data if provided
	userData := autoscaleUserData
	if userData != "" {
		if decoded, err := base64.StdEncoding.DecodeString(userData); err == nil {
			userData = string(decoded)
		}
	}

	// Create autoscaler
	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	// Create group
	groupID := fmt.Sprintf("asg-%s-%d", autoscaleName, time.Now().Unix())
	group := &autoscaler.AutoScaleGroup{
		AutoScaleGroupID: groupID,
		GroupName:        autoscaleName,
		JobArrayID:       autoscaleJobArrayID,
		DesiredCapacity:  autoscaleDesired,
		MinCapacity:      autoscaleMin,
		MaxCapacity:      autoscaleMax,
		Status:           "active",
		LaunchTemplate: autoscaler.LaunchTemplate{
			InstanceType:       autoscaleInstanceType,
			AMI:                autoscaleAMI,
			Spot:               autoscaleSpot,
			KeyName:            autoscaleKeyName,
			SubnetID:           autoscaleSubnetID,
			SecurityGroups:     autoscaleSecurityGroups,
			IAMInstanceProfile: autoscaleIAMProfile,
			UserData:           userData,
			Tags:               autoscaleTags,
		},
		HealthCheckInterval: 60 * time.Second,
		ReplacementStrategy: "immediate",
	}

	// Add scaling policy if specified
	if scalingPolicy == "queue-depth" {
		group.ScalingPolicy = &autoscaler.ScalingPolicy{
			PolicyType:                "queue-depth",
			QueueURL:                  queueURL,
			TargetMessagesPerInstance: targetMessagesPerInstance,
			ScaleUpCooldownSeconds:    scaleUpCooldown,
			ScaleDownCooldownSeconds:  scaleDownCooldown,
		}
	}

	if err := as.CreateGroup(ctx, group); err != nil {
		return fmt.Errorf("create group: %w", err)
	}

	fmt.Printf("Created autoscale group: %s\n", groupID)
	fmt.Printf("Group name: %s\n", autoscaleName)
	fmt.Printf("Job array ID: %s\n", autoscaleJobArrayID)
	fmt.Printf("Desired capacity: %d\n", autoscaleDesired)
	fmt.Printf("Min/Max: %d/%d\n", autoscaleMin, autoscaleMax)

	// Trigger Lambda immediately
	if err := triggerLambda(ctx, groupID); err != nil {
		log.Printf("Warning: failed to trigger Lambda: %v", err)
		fmt.Println("\nGroup created but Lambda not triggered. Instances will launch on next scheduled run (within 1 minute).")
	} else {
		fmt.Println("\nTriggered immediate reconciliation. Instances will launch shortly.")
	}

	return nil
}

func runAutoscaleUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	groupName := args[0]

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	group, err := as.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	changed := false
	if autoscaleNewDesired >= 0 {
		group.DesiredCapacity = autoscaleNewDesired
		changed = true
	}
	if autoscaleNewMin >= 0 {
		group.MinCapacity = autoscaleNewMin
		changed = true
	}
	if autoscaleNewMax >= 0 {
		group.MaxCapacity = autoscaleNewMax
		changed = true
	}

	if !changed {
		return fmt.Errorf("no changes specified")
	}

	// Validate
	if group.MinCapacity > group.DesiredCapacity {
		return fmt.Errorf("min-capacity cannot exceed desired-capacity")
	}
	if group.MaxCapacity < group.DesiredCapacity {
		return fmt.Errorf("max-capacity cannot be less than desired-capacity")
	}

	if err := as.UpdateGroup(ctx, group); err != nil {
		return fmt.Errorf("update group: %w", err)
	}

	fmt.Printf("Updated group %s\n", groupName)
	fmt.Printf("New capacity: desired=%d, min=%d, max=%d\n",
		group.DesiredCapacity, group.MinCapacity, group.MaxCapacity)

	// Trigger Lambda
	if err := triggerLambda(ctx, group.AutoScaleGroupID); err != nil {
		log.Printf("Warning: failed to trigger Lambda: %v", err)
	} else {
		fmt.Println("Triggered immediate reconciliation.")
	}

	return nil
}

func runAutoscaleStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	// If group name specified, show just that group
	if len(args) > 0 {
		group, err := as.GetGroupByName(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get group: %w", err)
		}

		printGroupStatus(group)
		return nil
	}

	// Otherwise list all active groups
	groups, err := as.ListActiveGroups(ctx)
	if err != nil {
		return fmt.Errorf("list groups: %w", err)
	}

	if len(groups) == 0 {
		fmt.Println("No active autoscale groups")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tDESIRED\tMIN\tMAX\tCREATED")
	for _, group := range groups {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%s\n",
			group.GroupName,
			group.Status,
			group.DesiredCapacity,
			group.MinCapacity,
			group.MaxCapacity,
			group.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()

	return nil
}

func runAutoscaleHealth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	groupName := args[0]

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	group, err := as.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	// Get instances
	cfg, _ := config.LoadDefaultConfig(ctx)
	ec2Client := ec2.NewFromConfig(cfg)

	result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:spawn:autoscale-group"),
				Values: []string{group.AutoScaleGroupID},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("describe instances: %w", err)
	}

	instanceIDs := make([]string, 0)
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil {
				instanceIDs = append(instanceIDs, aws.ToString(instance.InstanceId))
			}
		}
	}

	if len(instanceIDs) == 0 {
		fmt.Println("No instances found")
		return nil
	}

	// Check health
	dynamoClient := dynamodb.NewFromConfig(cfg)
	healthChecker := autoscaler.NewHealthChecker(ec2Client, dynamoClient, "spawn-hybrid-registry")

	health, err := healthChecker.CheckInstances(ctx, group.JobArrayID, instanceIDs)
	if err != nil {
		return fmt.Errorf("check health: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INSTANCE\tSTATE\tHEARTBEAT\tSPOT\tHEALTH")
	for _, h := range health {
		spotStr := "no"
		if h.SpotInterruption {
			spotStr = "yes"
		}

		heartbeatStr := "N/A"
		if h.HeartbeatAge > 0 {
			heartbeatStr = fmt.Sprintf("%v ago", h.HeartbeatAge.Round(time.Second))
		}

		healthStr := "✓ healthy"
		if !h.Healthy {
			healthStr = fmt.Sprintf("✗ %s", h.Reason)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			h.InstanceID, h.EC2State, heartbeatStr, spotStr, healthStr)
	}
	w.Flush()

	return nil
}

func runAutoscalePause(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	groupName := args[0]

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	group, err := as.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	group.Status = "paused"
	if err := as.UpdateGroup(ctx, group); err != nil {
		return fmt.Errorf("update group: %w", err)
	}

	fmt.Printf("Paused group %s (instances preserved)\n", groupName)
	return nil
}

func runAutoscaleResume(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	groupName := args[0]

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	group, err := as.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	group.Status = "active"
	if err := as.UpdateGroup(ctx, group); err != nil {
		return fmt.Errorf("update group: %w", err)
	}

	fmt.Printf("Resumed group %s\n", groupName)

	// Trigger Lambda
	if err := triggerLambda(ctx, group.AutoScaleGroupID); err != nil {
		log.Printf("Warning: failed to trigger Lambda: %v", err)
	}

	return nil
}

func runAutoscaleTerminate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	groupName := args[0]

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	group, err := as.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	// Terminate all instances
	cfg, _ := config.LoadDefaultConfig(ctx)
	ec2Client := ec2.NewFromConfig(cfg)

	result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
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
		return fmt.Errorf("describe instances: %w", err)
	}

	instanceIDs := make([]string, 0)
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil {
				instanceIDs = append(instanceIDs, aws.ToString(instance.InstanceId))
			}
		}
	}

	if len(instanceIDs) > 0 {
		_, err = ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: instanceIDs,
		})
		if err != nil {
			return fmt.Errorf("terminate instances: %w", err)
		}
		fmt.Printf("Terminated %d instances\n", len(instanceIDs))
	}

	// Delete group
	if err := as.DeleteGroup(ctx, group.AutoScaleGroupID); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	fmt.Printf("Terminated group %s\n", groupName)
	return nil
}

func runAutoscaleSetPolicy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	groupName := args[0]

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	group, err := as.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	// Handle --none flag (remove policy)
	if removePolicyFlag {
		group.ScalingPolicy = nil
		fmt.Printf("Removed scaling policy from %s (reverted to manual mode)\n", groupName)
	} else {
		// Validate and set policy
		if scalingPolicy == "" {
			return fmt.Errorf("--scaling-policy required (or use --none to remove)")
		}
		if scalingPolicy != "queue-depth" {
			return fmt.Errorf("invalid scaling policy: %s (only 'queue-depth' supported)", scalingPolicy)
		}
		if queueURL == "" {
			return fmt.Errorf("--queue-url required when --scaling-policy is set")
		}

		group.ScalingPolicy = &autoscaler.ScalingPolicy{
			PolicyType:                scalingPolicy,
			QueueURL:                  queueURL,
			TargetMessagesPerInstance: targetMessagesPerInstance,
			ScaleUpCooldownSeconds:    scaleUpCooldown,
			ScaleDownCooldownSeconds:  scaleDownCooldown,
		}

		fmt.Printf("Updated scaling policy for %s\n", groupName)
	}

	// Update group in DynamoDB
	if err := as.UpdateGroup(ctx, group); err != nil {
		return fmt.Errorf("update group: %w", err)
	}

	// Trigger Lambda
	if err := triggerLambda(ctx, group.AutoScaleGroupID); err != nil {
		log.Printf("Warning: failed to trigger Lambda: %v", err)
	} else {
		fmt.Println("Triggered immediate reconciliation.")
	}

	return nil
}

func runAutoscaleScalingActivity(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	groupName := args[0]

	as, err := getAutoscaler(ctx)
	if err != nil {
		return err
	}

	group, err := as.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}

	// Display scaling state
	if group.ScalingPolicy == nil {
		fmt.Println("No scaling policy configured (manual mode)")
		return nil
	}

	fmt.Printf("Scaling Policy: %s\n", group.ScalingPolicy.PolicyType)
	fmt.Printf("Queue: %s\n", group.ScalingPolicy.QueueURL)
	fmt.Printf("Target: %d messages/instance\n", group.ScalingPolicy.TargetMessagesPerInstance)
	fmt.Println()

	if group.ScalingState == nil {
		fmt.Println("No scaling activity yet")
		return nil
	}

	fmt.Printf("Last Queue Depth: %d messages\n", group.ScalingState.LastQueueDepth)
	fmt.Printf("Last Calculated Capacity: %d instances\n", group.ScalingState.LastCalculatedCapacity)

	if !group.ScalingState.LastScaleUp.IsZero() {
		fmt.Printf("Last Scale Up: %s (%s ago)\n",
			group.ScalingState.LastScaleUp.Format(time.RFC3339),
			time.Since(group.ScalingState.LastScaleUp).Round(time.Second))
	}
	if !group.ScalingState.LastScaleDown.IsZero() {
		fmt.Printf("Last Scale Down: %s (%s ago)\n",
			group.ScalingState.LastScaleDown.Format(time.RFC3339),
			time.Since(group.ScalingState.LastScaleDown).Round(time.Second))
	}

	return nil
}

func triggerLambda(ctx context.Context, groupID string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	lambdaClient := lambda.NewFromConfig(cfg)
	functionName := fmt.Sprintf("spawn-autoscale-orchestrator-%s", autoscaleEnv)

	payload := fmt.Sprintf(`{"group_id":"%s"}`, groupID)

	_, err = lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: lambdaTypes.InvocationTypeEvent,
		Payload:        []byte(payload),
	})

	return err
}

func printGroupStatus(group *autoscaler.AutoScaleGroup) {
	fmt.Printf("Group: %s (%s)\n", group.AutoScaleGroupID, group.GroupName)
	fmt.Printf("Job Array ID: %s\n", group.JobArrayID)
	fmt.Printf("Status: %s\n", group.Status)
	fmt.Printf("Capacity: desired=%d, min=%d, max=%d\n",
		group.DesiredCapacity, group.MinCapacity, group.MaxCapacity)
	fmt.Printf("Created: %s\n", group.CreatedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Updated: %s\n", group.UpdatedAt.Format("2006-01-02 15:04:05 MST"))
	if !group.LastScaleEvent.IsZero() {
		fmt.Printf("Last Scale Event: %s\n", group.LastScaleEvent.Format("2006-01-02 15:04:05 MST"))
	}
	fmt.Printf("Health Check Interval: %v\n", group.HealthCheckInterval)

	// Display scaling policy info
	if group.ScalingPolicy != nil {
		fmt.Printf("\nScaling Policy: %s\n", group.ScalingPolicy.PolicyType)
		fmt.Printf("  Queue: %s\n", group.ScalingPolicy.QueueURL)
		fmt.Printf("  Target: %d messages/instance\n", group.ScalingPolicy.TargetMessagesPerInstance)
		fmt.Printf("  Cooldowns: up=%ds, down=%ds\n",
			group.ScalingPolicy.ScaleUpCooldownSeconds,
			group.ScalingPolicy.ScaleDownCooldownSeconds)

		if group.ScalingState != nil {
			fmt.Printf("\nCurrent State:\n")
			fmt.Printf("  Queue Depth: %d messages\n", group.ScalingState.LastQueueDepth)
			if !group.ScalingState.LastScaleUp.IsZero() {
				fmt.Printf("  Last Scale Up: %s ago\n",
					time.Since(group.ScalingState.LastScaleUp).Round(time.Second))
			}
			if !group.ScalingState.LastScaleDown.IsZero() {
				fmt.Printf("  Last Scale Down: %s ago\n",
					time.Since(group.ScalingState.LastScaleDown).Round(time.Second))
			}
		}
	} else {
		fmt.Println("\nScaling Policy: Manual (no automatic scaling)")
	}
}
