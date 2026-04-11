//go:build integration

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/google/uuid"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	truffleaws "github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("LAGOTTO_INTEGRATION_TEST") != "1" {
		t.Skip("set LAGOTTO_INTEGRATION_TEST=1 to run integration tests")
	}
}

func integrationStore(t *testing.T) (*watcher.Store, context.Context) {
	t.Helper()
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		t.Fatalf("load AWS config: %v", err)
	}
	watchesTable := envOrDefault("LAGOTTO_WATCHES_TABLE", "lagotto-watches")
	historyTable := envOrDefault("LAGOTTO_HISTORY_TABLE", "lagotto-match-history")
	return watcher.NewStore(cfg, watchesTable, historyTable), ctx
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestIntegration_WatchLifecycle(t *testing.T) {
	skipUnlessIntegration(t)
	store, ctx := integrationStore(t)

	watchID := "w-integ-" + uuid.New().String()[:6]
	now := time.Now().UTC()

	// Create a watch for t3.micro (always available)
	w := &watcher.Watch{
		WatchID:             watchID,
		UserID:              "integration-test",
		Status:              watcher.StatusActive,
		InstanceTypePattern: "t3.micro",
		Regions:             []string{"us-east-1"},
		Spot:                false,
		Action:              watcher.ActionNotify,
		CreatedAt:           now,
		UpdatedAt:           now,
		ExpiresAt:           now.Add(1 * time.Hour),
		TTLTimestamp:        now.Add(1 * time.Hour).Unix(),
	}

	if err := store.PutWatch(ctx, w); err != nil {
		t.Fatalf("PutWatch: %v", err)
	}
	t.Cleanup(func() {
		_ = store.UpdateWatchStatus(context.Background(), watchID, watcher.StatusCancelled)
	})

	// Verify GetWatch
	got, err := store.GetWatch(ctx, watchID)
	if err != nil {
		t.Fatalf("GetWatch: %v", err)
	}
	if got == nil {
		t.Fatal("GetWatch returned nil")
	}
	if got.Status != watcher.StatusActive {
		t.Errorf("Status = %q, want active", got.Status)
	}

	// Poll — t3.micro should be available in us-east-1
	cfg, _ := config.LoadDefaultConfig(ctx)
	truffleClient, err := truffleaws.NewClient(ctx)
	if err != nil {
		t.Fatalf("create truffle client: %v", err)
	}
	_ = cfg // used by store already

	poller := watcher.NewPoller(truffleClient, store, false)
	matches, err := poller.PollWatch(ctx, got)
	if err != nil {
		t.Fatalf("PollWatch: %v", err)
	}
	if len(matches) == 0 {
		t.Log("Warning: no matches for t3.micro in us-east-1 (expected to match)")
	} else {
		t.Logf("Match found: %s in %s at $%.4f/hr", matches[0].InstanceType, matches[0].Region, matches[0].Price)
	}

	// Cancel
	if err := store.UpdateWatchStatus(ctx, watchID, watcher.StatusCancelled); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	got, _ = store.GetWatch(ctx, watchID)
	if got.Status != watcher.StatusCancelled {
		t.Errorf("Status after cancel = %q, want cancelled", got.Status)
	}
}

func TestIntegration_ExtendWatch(t *testing.T) {
	skipUnlessIntegration(t)
	store, ctx := integrationStore(t)

	watchID := "w-integ-ext-" + uuid.New().String()[:6]
	now := time.Now().UTC()

	w := &watcher.Watch{
		WatchID:             watchID,
		UserID:              "integration-test",
		Status:              watcher.StatusActive,
		InstanceTypePattern: "t3.micro",
		Regions:             []string{"us-east-1"},
		Action:              watcher.ActionNotify,
		CreatedAt:           now,
		UpdatedAt:           now,
		ExpiresAt:           now.Add(1 * time.Hour),
		TTLTimestamp:        now.Add(1 * time.Hour).Unix(),
	}
	if err := store.PutWatch(ctx, w); err != nil {
		t.Fatalf("PutWatch: %v", err)
	}
	t.Cleanup(func() {
		_ = store.UpdateWatchStatus(context.Background(), watchID, watcher.StatusCancelled)
	})

	newExpiry := now.Add(48 * time.Hour)
	if err := store.ExtendWatch(ctx, watchID, newExpiry, false); err != nil {
		t.Fatalf("ExtendWatch: %v", err)
	}

	got, _ := store.GetWatch(ctx, watchID)
	if got.TTLTimestamp != newExpiry.Unix() {
		t.Errorf("TTLTimestamp = %d, want %d", got.TTLTimestamp, newExpiry.Unix())
	}
}
