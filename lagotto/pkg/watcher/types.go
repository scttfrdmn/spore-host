// Package watcher provides EC2 capacity watching, polling, and matching.
package watcher

import (
	"time"
)

// ActionMode defines what happens when capacity is found.
type ActionMode string

const (
	// ActionNotify sends a notification only.
	ActionNotify ActionMode = "notify"
	// ActionSpawn sends a notification and auto-launches an instance.
	ActionSpawn ActionMode = "spawn"
)

// WatchStatus represents the lifecycle state of a watch.
type WatchStatus string

const (
	// StatusActive means the watch is being polled.
	StatusActive WatchStatus = "active"
	// StatusMatched means capacity was found and action was taken.
	StatusMatched WatchStatus = "matched"
	// StatusExpired means the watch TTL elapsed without a match.
	StatusExpired WatchStatus = "expired"
	// StatusCancelled means the user cancelled the watch.
	StatusCancelled WatchStatus = "cancelled"
)

// Watch represents a user's request to monitor for instance capacity.
type Watch struct {
	WatchID             string          `json:"watch_id" dynamodbav:"watch_id"`
	UserID              string          `json:"user_id" dynamodbav:"user_id"`
	Status              WatchStatus     `json:"status" dynamodbav:"status"`
	InstanceTypePattern string          `json:"instance_type_pattern" dynamodbav:"instance_type_pattern"`
	Regions             []string        `json:"regions" dynamodbav:"regions"`
	Spot                bool            `json:"spot" dynamodbav:"spot"`
	MaxPrice            float64         `json:"max_price,omitempty" dynamodbav:"max_price,omitempty"`
	Action              ActionMode      `json:"action" dynamodbav:"action"`
	NotifyChannels      []NotifyChannel `json:"notify_channels,omitempty" dynamodbav:"notify_channels,omitempty"`
	LaunchConfigJSON    []byte          `json:"launch_config_json,omitempty" dynamodbav:"launch_config_json,omitempty"`
	CreatedAt           time.Time       `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" dynamodbav:"updated_at"`
	ExpiresAt           time.Time       `json:"expires_at" dynamodbav:"expires_at"`
	TTLTimestamp        int64           `json:"ttl_timestamp" dynamodbav:"ttl_timestamp"`
	LastPolledAt        time.Time       `json:"last_polled_at,omitempty" dynamodbav:"last_polled_at,omitempty"`
	MatchCount          int             `json:"match_count" dynamodbav:"match_count"`
	LastMatch           *MatchResult    `json:"last_match,omitempty" dynamodbav:"last_match,omitempty"`
}

// MatchResult records a capacity match event.
type MatchResult struct {
	WatchID          string    `json:"watch_id" dynamodbav:"watch_id"`
	UserID           string    `json:"user_id" dynamodbav:"user_id"`
	Region           string    `json:"region" dynamodbav:"region"`
	AvailabilityZone string    `json:"availability_zone" dynamodbav:"availability_zone"`
	InstanceType     string    `json:"instance_type" dynamodbav:"instance_type"`
	Price            float64   `json:"price" dynamodbav:"price"`
	IsSpot           bool      `json:"is_spot" dynamodbav:"is_spot"`
	MatchedAt        time.Time `json:"matched_at" dynamodbav:"matched_at"`
	ActionTaken      string    `json:"action_taken" dynamodbav:"action_taken"`
	InstanceID       string    `json:"instance_id,omitempty" dynamodbav:"instance_id,omitempty"`
	TTLTimestamp     int64     `json:"ttl_timestamp" dynamodbav:"ttl_timestamp"`
}

// NotifyChannel specifies how to reach the user on a match.
type NotifyChannel struct {
	Type   string `json:"type" dynamodbav:"type"`     // "email", "webhook", "sns"
	Target string `json:"target" dynamodbav:"target"` // address, URL, or ARN
}
