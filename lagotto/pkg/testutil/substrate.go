// Package testutil provides shared test helpers for lagotto packages.
package testutil

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	substrate "github.com/scttfrdmn/substrate"
)

// TestEnv holds a running Substrate server and a pre-configured AWS config.
type TestEnv struct {
	URL       string
	AWSConfig aws.Config
}

// SubstrateServer starts a Substrate emulator and returns a TestEnv.
func SubstrateServer(t *testing.T) *TestEnv {
	t.Helper()
	ts := substrate.StartTestServer(t)

	cfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithBaseEndpoint(ts.URL),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", "test"),
		),
	)
	if err != nil {
		t.Fatalf("SubstrateServer: build AWS config: %v", err)
	}

	return &TestEnv{URL: ts.URL, AWSConfig: cfg}
}

// DynamoDBClient returns a DynamoDB client pointed at the Substrate server.
func (e *TestEnv) DynamoDBClient() *dynamodb.Client {
	return dynamodb.NewFromConfig(e.AWSConfig)
}

// CreateWatchesTable creates the lagotto-watches table in the emulator.
func (e *TestEnv) CreateWatchesTable(t *testing.T, tableName string) {
	t.Helper()
	client := e.DynamoDBClient()
	_, err := client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("watch_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("status"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("watch_id"), KeyType: types.KeyTypeHash},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("user_id-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("user_id"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String("status-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("status"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateWatchesTable: %v", err)
	}
}

// CreateHistoryTable creates the lagotto-match-history table in the emulator.
func (e *TestEnv) CreateHistoryTable(t *testing.T, tableName string) {
	t.Helper()
	client := e.DynamoDBClient()
	_, err := client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("watch_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("matched_at"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("user_id"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("watch_id"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("matched_at"), KeyType: types.KeyTypeRange},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("user_id-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("user_id"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("matched_at"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateHistoryTable: %v", err)
	}
}
