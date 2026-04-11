package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Store handles DynamoDB persistence for watches and match history.
type Store struct {
	client       *dynamodb.Client
	watchesTable string
	historyTable string
}

// NewStore creates a Store backed by DynamoDB.
func NewStore(cfg aws.Config, watchesTable, historyTable string) *Store {
	return &Store{
		client:       dynamodb.NewFromConfig(cfg),
		watchesTable: watchesTable,
		historyTable: historyTable,
	}
}

// PutWatch creates or updates a watch record.
func (s *Store) PutWatch(ctx context.Context, w *Watch) error {
	w.UpdatedAt = time.Now().UTC()
	item, err := attributevalue.MarshalMap(w)
	if err != nil {
		return fmt.Errorf("marshal watch: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.watchesTable,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put watch: %w", err)
	}
	return nil
}

// GetWatch retrieves a single watch by ID.
func (s *Store) GetWatch(ctx context.Context, watchID string) (*Watch, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.watchesTable,
		Key: map[string]types.AttributeValue{
			"watch_id": &types.AttributeValueMemberS{Value: watchID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get watch: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}
	var w Watch
	if err := attributevalue.UnmarshalMap(result.Item, &w); err != nil {
		return nil, fmt.Errorf("unmarshal watch: %w", err)
	}
	return &w, nil
}

// ListWatchesByUser returns watches belonging to a user, optionally filtered by status.
func (s *Store) ListWatchesByUser(ctx context.Context, userID string, statusFilter WatchStatus) ([]Watch, error) {
	expr := "user_id = :uid"
	exprValues := map[string]types.AttributeValue{
		":uid": &types.AttributeValueMemberS{Value: userID},
	}
	if statusFilter != "" {
		expr += " AND #st = :status"
		exprValues[":status"] = &types.AttributeValueMemberS{Value: string(statusFilter)}
	}

	var exprNames map[string]string
	if statusFilter != "" {
		exprNames = map[string]string{"#st": "status"}
	}

	input := &dynamodb.QueryInput{
		TableName:                 &s.watchesTable,
		IndexName:                 aws.String("user_id-index"),
		KeyConditionExpression:    aws.String("user_id = :uid"),
		ExpressionAttributeValues: exprValues,
	}
	if statusFilter != "" {
		input.FilterExpression = aws.String("#st = :status")
		input.ExpressionAttributeNames = exprNames
	}

	result, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("list watches: %w", err)
	}

	watches := make([]Watch, 0, len(result.Items))
	for _, item := range result.Items {
		var w Watch
		if err := attributevalue.UnmarshalMap(item, &w); err != nil {
			return nil, fmt.Errorf("unmarshal watch: %w", err)
		}
		watches = append(watches, w)
	}
	return watches, nil
}

// ListActiveWatches returns all watches with status "active". Used by the poller.
func (s *Store) ListActiveWatches(ctx context.Context) ([]Watch, error) {
	input := &dynamodb.QueryInput{
		TableName:              &s.watchesTable,
		IndexName:              aws.String("status-index"),
		KeyConditionExpression: aws.String("#st = :status"),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(StatusActive)},
		},
	}

	var watches []Watch
	for {
		result, err := s.client.Query(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list active watches: %w", err)
		}
		for _, item := range result.Items {
			var w Watch
			if err := attributevalue.UnmarshalMap(item, &w); err != nil {
				return nil, fmt.Errorf("unmarshal watch: %w", err)
			}
			watches = append(watches, w)
		}
		if result.LastEvaluatedKey == nil {
			break
		}
		input.ExclusiveStartKey = result.LastEvaluatedKey
	}
	return watches, nil
}

// UpdateWatchStatus atomically updates a watch's status.
func (s *Store) UpdateWatchStatus(ctx context.Context, watchID string, status WatchStatus) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.watchesTable,
		Key: map[string]types.AttributeValue{
			"watch_id": &types.AttributeValueMemberS{Value: watchID},
		},
		UpdateExpression: aws.String("SET #st = :status, updated_at = :now"),
		ExpressionAttributeNames: map[string]string{
			"#st": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(status)},
			":now":    &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("update watch status: %w", err)
	}
	return nil
}

// RecordMatch updates the watch with match info and writes a history record.
func (s *Store) RecordMatch(ctx context.Context, w *Watch, m *MatchResult) error {
	// Update the watch record
	now := time.Now().UTC()
	w.MatchCount++
	w.LastMatch = m
	w.UpdatedAt = now
	if err := s.PutWatch(ctx, w); err != nil {
		return fmt.Errorf("update watch with match: %w", err)
	}

	// Write to match history
	m.TTLTimestamp = now.Add(90 * 24 * time.Hour).Unix() // 90-day retention
	item, err := attributevalue.MarshalMap(m)
	if err != nil {
		return fmt.Errorf("marshal match: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.historyTable,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put match history: %w", err)
	}
	return nil
}

// UpdateLastPolled updates the last_polled_at timestamp on a watch.
func (s *Store) UpdateLastPolled(ctx context.Context, watchID string) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.watchesTable,
		Key: map[string]types.AttributeValue{
			"watch_id": &types.AttributeValueMemberS{Value: watchID},
		},
		UpdateExpression: aws.String("SET last_polled_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("update last polled: %w", err)
	}
	return nil
}

// ListMatchHistory returns match history, optionally filtered by watch ID.
func (s *Store) ListMatchHistory(ctx context.Context, watchID string) ([]MatchResult, error) {
	if watchID != "" {
		return s.listMatchHistoryByWatch(ctx, watchID)
	}
	// Scan all history (for the user's history command — should be scoped by user in practice)
	return s.scanMatchHistory(ctx)
}

func (s *Store) listMatchHistoryByWatch(ctx context.Context, watchID string) ([]MatchResult, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.historyTable,
		KeyConditionExpression: aws.String("watch_id = :wid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":wid": &types.AttributeValueMemberS{Value: watchID},
		},
		ScanIndexForward: aws.Bool(false), // newest first
	})
	if err != nil {
		return nil, fmt.Errorf("query match history: %w", err)
	}
	return unmarshalMatches(result.Items)
}

func (s *Store) scanMatchHistory(ctx context.Context) ([]MatchResult, error) {
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: &s.historyTable,
		Limit:     aws.Int32(100),
	})
	if err != nil {
		return nil, fmt.Errorf("scan match history: %w", err)
	}
	return unmarshalMatches(result.Items)
}

// ListMatchHistoryByUser returns match history for a specific user.
func (s *Store) ListMatchHistoryByUser(ctx context.Context, userID string) ([]MatchResult, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.historyTable,
		IndexName:              aws.String("user_id-index"),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("query match history by user: %w", err)
	}
	return unmarshalMatches(result.Items)
}

func unmarshalMatches(items []map[string]types.AttributeValue) ([]MatchResult, error) {
	matches := make([]MatchResult, 0, len(items))
	for _, item := range items {
		var m MatchResult
		if err := attributevalue.UnmarshalMap(item, &m); err != nil {
			return nil, fmt.Errorf("unmarshal match: %w", err)
		}
		matches = append(matches, m)
	}
	return matches, nil
}
