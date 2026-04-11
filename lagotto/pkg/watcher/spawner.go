package watcher

import (
	"context"
	"encoding/json"
	"fmt"

	spawnaws "github.com/scttfrdmn/spore-host/spawn/pkg/aws"
)

// Spawner launches instances via spawn's library when a watch matches.
type Spawner struct {
	client *spawnaws.Client
}

// NewSpawner creates a Spawner that uses spawn's EC2 launch library.
func NewSpawner(ctx context.Context) (*Spawner, error) {
	client, err := spawnaws.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create spawn client: %w", err)
	}
	return &Spawner{client: client}, nil
}

// Spawn deserializes the stored LaunchConfig and launches an instance
// in the region/AZ where capacity was found. Updates the MatchResult
// with the instance ID on success.
func (s *Spawner) Spawn(ctx context.Context, w *Watch, m *MatchResult) error {
	if len(w.LaunchConfigJSON) == 0 {
		return fmt.Errorf("watch %s has no launch config", w.WatchID)
	}

	var cfg spawnaws.LaunchConfig
	if err := json.Unmarshal(w.LaunchConfigJSON, &cfg); err != nil {
		return fmt.Errorf("unmarshal launch config: %w", err)
	}

	// Override region and AZ with where capacity was actually found
	cfg.Region = m.Region
	if m.AvailabilityZone != "" {
		cfg.AvailabilityZone = m.AvailabilityZone
	}
	// Use the matched instance type (pattern may have resolved to specific type)
	cfg.InstanceType = m.InstanceType
	cfg.Spot = m.IsSpot

	result, err := s.client.Launch(ctx, cfg)
	if err != nil {
		m.ActionTaken = "spawn_failed"
		return fmt.Errorf("launch instance: %w", err)
	}

	m.InstanceID = result.InstanceID
	m.ActionTaken = "spawned"
	return nil
}
