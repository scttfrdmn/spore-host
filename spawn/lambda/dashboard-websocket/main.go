package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	connectionsTable = "spawn-websocket-connections"
	connectionTTL    = 2 * time.Hour
)

// Connection represents a WebSocket connection in DynamoDB
type Connection struct {
	ConnectionID string    `dynamodbav:"connection_id"`
	UserID       string    `dynamodbav:"user_id"`
	ConnectedAt  time.Time `dynamodbav:"connected_at"`
	TTL          int64     `dynamodbav:"ttl"`
}

// AWSCredentials represents the credentials passed via query string
type AWSCredentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
}

// handler is the main Lambda handler for WebSocket events
func handler(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return errorResponse(500, "Failed to load AWS config"), nil
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Route based on connection event
	switch request.RequestContext.RouteKey {
	case "$connect":
		return handleConnect(ctx, cfg, dynamoClient, request)
	case "$disconnect":
		return handleDisconnect(ctx, dynamoClient, request.RequestContext.ConnectionID)
	case "$default":
		return handleMessage(ctx, dynamoClient, request)
	default:
		return errorResponse(400, "Unknown route"), nil
	}
}

// handleConnect handles WebSocket connection establishment
func handleConnect(ctx context.Context, cfg aws.Config, dynamoClient *dynamodb.Client, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract token from query string
	tokenParam, ok := request.QueryStringParameters["token"]
	if !ok || tokenParam == "" {
		return errorResponse(401, "Missing authentication token"), nil
	}

	// Decode base64 token
	tokenJSON, err := base64.StdEncoding.DecodeString(tokenParam)
	if err != nil {
		return errorResponse(401, "Invalid token format"), nil
	}

	// Parse credentials
	var creds AWSCredentials
	if err := json.Unmarshal(tokenJSON, &creds); err != nil {
		return errorResponse(401, "Invalid token structure"), nil
	}

	// Validate credentials using STS
	userID, err := validateCredentials(ctx, creds)
	if err != nil {
		return errorResponse(401, fmt.Sprintf("Authentication failed: %v", err)), nil
	}

	// Save connection to DynamoDB
	connection := Connection{
		ConnectionID: request.RequestContext.ConnectionID,
		UserID:       userID,
		ConnectedAt:  time.Now(),
		TTL:          time.Now().Add(connectionTTL).Unix(),
	}

	item, err := attributevalue.MarshalMap(connection)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to marshal connection: %v", err)), nil
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(connectionsTable),
		Item:      item,
	})
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to save connection: %v", err)), nil
	}

	fmt.Printf("Connection established: %s (user: %s)\n", connection.ConnectionID, userID)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Connected",
	}, nil
}

// handleDisconnect handles WebSocket disconnection
func handleDisconnect(ctx context.Context, dynamoClient *dynamodb.Client, connectionID string) (events.APIGatewayProxyResponse, error) {
	// Delete connection from DynamoDB
	_, err := dynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(connectionsTable),
		Key: map[string]types.AttributeValue{
			"connection_id": &types.AttributeValueMemberS{
				Value: connectionID,
			},
		},
	})
	if err != nil {
		fmt.Printf("Warning: failed to delete connection %s: %v\n", connectionID, err)
		// Don't fail the disconnect - connection is already closed
	} else {
		fmt.Printf("Connection disconnected: %s\n", connectionID)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Disconnected",
	}, nil
}

// handleMessage handles messages from client
func handleMessage(ctx context.Context, dynamoClient *dynamodb.Client, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Future: handle client→server messages (subscribe to specific resources)
	// v0.22.0: just acknowledge receipt
	fmt.Printf("Received message from connection %s: %s\n", request.RequestContext.ConnectionID, request.Body)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Message received",
	}, nil
}

// validateCredentials validates AWS credentials using STS and returns the IAM user ARN
func validateCredentials(ctx context.Context, creds AWSCredentials) (string, error) {
	// Create a custom config with the provided credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKey,
				SessionToken:    creds.SessionToken,
			}, nil
		})),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create config: %w", err)
	}

	// Call STS GetCallerIdentity to validate credentials
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("invalid credentials: %w", err)
	}

	if identity.Arn == nil {
		return "", fmt.Errorf("ARN not returned by STS")
	}

	// Extract user ARN (format: arn:aws:sts::123456789012:assumed-role/role-name/user-name)
	userARN := *identity.Arn
	return userARN, nil
}

// errorResponse creates an error response
func errorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	body := fmt.Sprintf(`{"error": "%s"}`, message)
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Body:       body,
	}
}

// main is the entry point for the Lambda function
func main() {
	lambda.Start(handler)
}
