package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// handler is the main Lambda handler
func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract path and method
	path := request.Path
	method := request.HTTPMethod

	// Log request for debugging
	fmt.Printf("Request: method=%s path=%s\n", method, path)
	fmt.Printf("Headers: %+v\n", request.Headers)

	// Handle OPTIONS for CORS preflight
	if method == "OPTIONS" {
		fmt.Println("Handling OPTIONS request")
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    corsHeaders,
			Body:       "",
		}, nil
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return errorResponse(500, "Failed to load AWS config"), nil
	}

	// Extract user identity and account info
	userID, cliIamArn, accountBase36, err := getUserFromRequest(ctx, cfg, request)
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		return errorResponse(401, fmt.Sprintf("Authentication failed: %v", err)), nil
	}

	fmt.Printf("✓ Authentication successful: userID=%s cliIamArn=%s accountBase36=%s\n", userID, cliIamArn, accountBase36)
	fmt.Printf("Routing request: path=%s method=%s\n", path, method)

	// Route to appropriate handler
	switch {
	case path == "/api/instances" && method == "GET":
		return handleListInstances(ctx, cfg, cliIamArn)

	case path == "/api/instances/" && method == "GET":
		// Extract instance ID from path
		instanceID := request.PathParameters["id"]
		if instanceID == "" {
			return errorResponse(400, "Instance ID is required"), nil
		}
		return handleGetInstance(ctx, cfg, instanceID, cliIamArn)

	case path == "/api/sweeps" && method == "GET":
		return handleListSweeps(ctx, cfg, cliIamArn)

	case path == "/api/sweeps/" && method == "GET":
		// Extract sweep ID from path
		sweepID := request.PathParameters["id"]
		if sweepID == "" {
			return errorResponse(400, "Sweep ID is required"), nil
		}
		return handleGetSweep(ctx, cfg, sweepID, cliIamArn)

	case path == "/api/sweeps//cancel" && method == "POST":
		// Extract sweep ID from path
		sweepID := request.PathParameters["id"]
		if sweepID == "" {
			return errorResponse(400, "Sweep ID is required"), nil
		}
		return handleCancelSweep(ctx, cfg, sweepID, cliIamArn)

	case path == "/api/sweeps/cleanup" && method == "POST":
		return handleCleanupSweeps(ctx, cfg, request.Body, cliIamArn)

	case path == "/api/autoscale-groups" && method == "GET":
		return handleListAutoscaleGroups(ctx, cfg, cliIamArn)

	case path == "/api/autoscale-groups/" && method == "GET":
		// Extract group ID from path
		groupID := request.PathParameters["id"]
		if groupID == "" {
			return errorResponse(400, "Autoscale group ID is required"), nil
		}
		return handleGetAutoscaleGroup(ctx, cfg, groupID, cliIamArn)

	case path == "/api/autoscale-groups//pause" && method == "POST":
		groupID := request.PathParameters["id"]
		if groupID == "" {
			return errorResponse(400, "Autoscale group ID is required"), nil
		}
		return handlePauseAutoscaleGroup(ctx, cfg, groupID, cliIamArn)

	case path == "/api/autoscale-groups//resume" && method == "POST":
		groupID := request.PathParameters["id"]
		if groupID == "" {
			return errorResponse(400, "Autoscale group ID is required"), nil
		}
		return handleResumeAutoscaleGroup(ctx, cfg, groupID, cliIamArn)

	case path == "/api/autoscale-groups/" && method == "DELETE":
		groupID := request.PathParameters["id"]
		if groupID == "" {
			return errorResponse(400, "Autoscale group ID is required"), nil
		}
		return handleTerminateAutoscaleGroup(ctx, cfg, groupID, cliIamArn)

	case path == "/api/instances/" && method == "DELETE":
		instanceID := request.PathParameters["id"]
		if instanceID == "" {
			return errorResponse(400, "Instance ID is required"), nil
		}
		return handleTerminateInstance(ctx, cfg, instanceID, cliIamArn)

	case path == "/api/cost-summary" && method == "GET":
		return handleGetCostSummary(ctx, cfg, cliIamArn)

	case path == "/api/cost-history" && method == "GET":
		days := 30
		if d := request.QueryStringParameters["days"]; d != "" {
			if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
				days = n
			}
		}
		return handleGetCostHistory(ctx, cfg, days, cliIamArn)

	case path == "/api/alert-preferences" && method == "GET":
		return handleGetAlertPreferences(ctx, cfg, cliIamArn)

	case path == "/api/alert-preferences" && method == "POST":
		return handleSaveAlertPreferences(ctx, cfg, request.Body, cliIamArn)

	case path == "/api/user/profile" && method == "GET":
		return handleGetUserProfile(ctx, cfg, userID, cliIamArn, accountBase36)

	default:
		return errorResponse(404, "Endpoint not found"), nil
	}
}

// handleListInstances handles GET /api/instances
func handleListInstances(ctx context.Context, cfg aws.Config, cliIamArn string) (events.APIGatewayProxyResponse, error) {
	startTime := time.Now()

	// Query all regions in parallel (filtered by IAM user for per-user isolation)
	instances, err := listInstances(ctx, cfg, cliIamArn)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to list instances: %v", err)), nil
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Listed %d instances across %d regions in %v\n", len(instances), len(awsRegions), elapsed)

	// Build response
	response := APIResponse{
		Success:        true,
		RegionsQueried: awsRegions,
		TotalInstances: len(instances),
		Instances:      instances,
	}

	return successResponse(response)
}

// handleGetInstance handles GET /api/instances/{id}
func handleGetInstance(ctx context.Context, cfg aws.Config, instanceID, cliIamArn string) (events.APIGatewayProxyResponse, error) {
	// Get single instance (with per-user isolation check)
	instance, err := getInstance(ctx, cfg, instanceID, cliIamArn)
	if err != nil {
		return errorResponse(404, fmt.Sprintf("Instance not found: %v", err)), nil
	}

	// Build response
	response := APIResponse{
		Success:  true,
		Instance: instance,
	}

	return successResponse(response)
}

// handleGetUserProfile handles GET /api/user/profile
func handleGetUserProfile(ctx context.Context, cfg aws.Config, userID, cliIamArn, accountBase36 string) (events.APIGatewayProxyResponse, error) {
	// Get user profile from DynamoDB
	cached, err := getUserAccount(ctx, cfg, userID)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to get user profile: %v", err)), nil
	}

	var profile UserProfile
	if cached != nil {
		createdAt, _ := time.Parse(time.RFC3339, cached.CreatedAt)
		lastAccess, _ := time.Parse(time.RFC3339, cached.LastAccess)

		profile = UserProfile{
			UserID:        cached.UserID,
			AWSAccountID:  cached.AWSAccountID,
			AccountBase36: cached.AccountBase36,
			Email:         cached.Email,
			CreatedAt:     createdAt,
			LastAccess:    lastAccess,
		}
	} else {
		// No cache entry, return detected info
		profile = UserProfile{
			UserID:        userID,
			AWSAccountID:  cliIamArn,
			AccountBase36: accountBase36,
			CreatedAt:     time.Now(),
			LastAccess:    time.Now(),
		}
	}

	// Build response
	response := APIResponse{
		Success: true,
		User:    &profile,
	}

	return successResponse(response)
}

// main is the entry point for the Lambda function
func main() {
	lambda.Start(handler)
}
