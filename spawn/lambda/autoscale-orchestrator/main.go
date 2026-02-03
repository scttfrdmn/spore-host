package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	lambdaSDK "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/scttfrdmn/mycelium/spawn/pkg/autoscaler"
)

// AutoScaleEvent is the Lambda event payload
type AutoScaleEvent struct {
	GroupID string `json:"group_id,omitempty"`
}

var (
	autoscalerInstance *autoscaler.AutoScaler
	lambdaClient       *lambdaSDK.Client
	functionName       string
)

func init() {
	ctx := context.Background()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Get environment variables
	autoscaleTableName := os.Getenv("AUTOSCALE_GROUPS_TABLE")
	if autoscaleTableName == "" {
		autoscaleTableName = "spawn-autoscale-groups"
	}

	registryTableName := os.Getenv("HYBRID_REGISTRY_TABLE")
	if registryTableName == "" {
		registryTableName = "spawn-hybrid-registry"
	}

	functionName = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	// Create clients
	ec2Client := ec2.NewFromConfig(cfg)
	dynamoClient := dynamodb.NewFromConfig(cfg)
	lambdaClient = lambdaSDK.NewFromConfig(cfg)

	// Create autoscaler
	autoscalerInstance = autoscaler.NewAutoScaler(&autoscaler.Config{
		EC2Client:     ec2Client,
		DynamoClient:  dynamoClient,
		TableName:     autoscaleTableName,
		RegistryTable: registryTableName,
	})

	log.Printf("autoscale orchestrator initialized (table: %s, registry: %s)",
		autoscaleTableName, registryTableName)
}

func handler(ctx context.Context, event AutoScaleEvent) error {
	start := time.Now()
	log.Printf("autoscale orchestrator started (group_id: %s)", event.GroupID)

	// If specific group ID provided, reconcile just that group
	if event.GroupID != "" {
		if err := autoscalerInstance.Reconcile(ctx, event.GroupID); err != nil {
			log.Printf("failed to reconcile group %s: %v", event.GroupID, err)
			return err
		}
		log.Printf("reconciled group %s in %v", event.GroupID, time.Since(start))
		return nil
	}

	// Otherwise, reconcile all active groups
	groups, err := autoscalerInstance.ListActiveGroups(ctx)
	if err != nil {
		return fmt.Errorf("list active groups: %w", err)
	}

	log.Printf("found %d active groups to reconcile", len(groups))

	for i, group := range groups {
		if err := autoscalerInstance.Reconcile(ctx, group.AutoScaleGroupID); err != nil {
			log.Printf("failed to reconcile group %s: %v", group.GroupName, err)
		}

		// Check timeout (13-minute limit for Lambda)
		if time.Since(start) > 13*time.Minute {
			log.Printf("approaching timeout, self-invoking to continue from group %d/%d",
				i+1, len(groups))

			// Self-invoke to continue with remaining groups
			if i+1 < len(groups) {
				payload := fmt.Sprintf(`{"group_id":"%s"}`, groups[i+1].AutoScaleGroupID)
				_, err := lambdaClient.Invoke(ctx, &lambdaSDK.InvokeInput{
					FunctionName:   &functionName,
					InvocationType: types.InvocationTypeEvent,
					Payload:        []byte(payload),
				})
				if err != nil {
					log.Printf("failed to self-invoke: %v", err)
				}
			}
			break
		}
	}

	log.Printf("autoscale orchestrator completed in %v", time.Since(start))
	return nil
}

func main() {
	lambda.Start(handler)
}
