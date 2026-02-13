package autoscaler

import (
	"context"
	"testing"
	"time"
)

func TestGetDefaultDrainConfig(t *testing.T) {
	config := GetDefaultDrainConfig()

	if config.Enabled {
		t.Error("default drain should be disabled")
	}

	if config.TimeoutSeconds != 300 {
		t.Errorf("default timeout: got %d, want 300", config.TimeoutSeconds)
	}

	if config.CheckInterval != 30*time.Second {
		t.Errorf("default check interval: got %v, want 30s", config.CheckInterval)
	}
}

func TestDrainManager_hasActiveWork(t *testing.T) {
	dm := &DrainManager{}

	// Current implementation always returns false (placeholder)
	hasWork, err := dm.hasActiveWork(context.Background(), "i-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasWork {
		t.Error("expected no active work (placeholder implementation)")
	}
}
