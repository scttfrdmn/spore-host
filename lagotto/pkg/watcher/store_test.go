package watcher_test

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/spore-host/lagotto/pkg/testutil"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
)

const (
	testWatchesTable = "test-watches"
	testHistoryTable = "test-history"
)

func setupStore(t *testing.T) *watcher.Store {
	t.Helper()
	env := testutil.SubstrateServer(t)
	env.CreateWatchesTable(t, testWatchesTable)
	env.CreateHistoryTable(t, testHistoryTable)
	return watcher.NewStore(env.AWSConfig, testWatchesTable, testHistoryTable)
}

func newTestWatch(id, userID string) *watcher.Watch {
	now := time.Now().UTC()
	return &watcher.Watch{
		WatchID:             id,
		UserID:              userID,
		Status:              watcher.StatusActive,
		InstanceTypePattern: "g5.xlarge",
		Regions:             []string{"us-east-1"},
		Spot:                true,
		MaxPrice:            1.50,
		Action:              watcher.ActionNotify,
		CreatedAt:           now,
		UpdatedAt:           now,
		ExpiresAt:           now.Add(24 * time.Hour),
		TTLTimestamp:        now.Add(24 * time.Hour).Unix(),
	}
}

func TestPutAndGetWatch(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	w := newTestWatch("w-test1", "arn:aws:iam::123456789012:user/alice")
	if err := store.PutWatch(ctx, w); err != nil {
		t.Fatalf("PutWatch: %v", err)
	}

	got, err := store.GetWatch(ctx, "w-test1")
	if err != nil {
		t.Fatalf("GetWatch: %v", err)
	}
	if got == nil {
		t.Fatal("GetWatch returned nil")
	}
	if got.WatchID != "w-test1" {
		t.Errorf("WatchID = %q, want w-test1", got.WatchID)
	}
	if got.InstanceTypePattern != "g5.xlarge" {
		t.Errorf("Pattern = %q, want g5.xlarge", got.InstanceTypePattern)
	}
	if got.Status != watcher.StatusActive {
		t.Errorf("Status = %q, want active", got.Status)
	}
}

func TestGetWatch_NotFound(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	got, err := store.GetWatch(ctx, "w-nonexistent")
	if err != nil {
		t.Fatalf("GetWatch: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent watch, got %+v", got)
	}
}

func TestListWatchesByUser(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	alice := "arn:aws:iam::123456789012:user/alice"
	bob := "arn:aws:iam::123456789012:user/bob"

	_ = store.PutWatch(ctx, newTestWatch("w-a1", alice))
	_ = store.PutWatch(ctx, newTestWatch("w-a2", alice))
	_ = store.PutWatch(ctx, newTestWatch("w-b1", bob))

	watches, err := store.ListWatchesByUser(ctx, alice, "")
	if err != nil {
		t.Fatalf("ListWatchesByUser: %v", err)
	}
	if len(watches) != 2 {
		t.Errorf("got %d watches for alice, want 2", len(watches))
	}
}

func TestListActiveWatches(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	user := "arn:aws:iam::123456789012:user/test"
	w1 := newTestWatch("w-active", user)
	w2 := newTestWatch("w-cancelled", user)
	w2.Status = watcher.StatusCancelled

	_ = store.PutWatch(ctx, w1)
	_ = store.PutWatch(ctx, w2)

	active, err := store.ListActiveWatches(ctx)
	if err != nil {
		t.Fatalf("ListActiveWatches: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("got %d active watches, want 1", len(active))
	}
	if len(active) > 0 && active[0].WatchID != "w-active" {
		t.Errorf("WatchID = %q, want w-active", active[0].WatchID)
	}
}

func TestUpdateWatchStatus(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	w := newTestWatch("w-status", "arn:aws:iam::123456789012:user/test")
	_ = store.PutWatch(ctx, w)

	if err := store.UpdateWatchStatus(ctx, "w-status", watcher.StatusCancelled); err != nil {
		t.Fatalf("UpdateWatchStatus: %v", err)
	}

	got, _ := store.GetWatch(ctx, "w-status")
	if got.Status != watcher.StatusCancelled {
		t.Errorf("Status = %q, want cancelled", got.Status)
	}
}

func TestRecordMatch(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	user := "arn:aws:iam::123456789012:user/test"
	w := newTestWatch("w-match", user)
	_ = store.PutWatch(ctx, w)

	m := &watcher.MatchResult{
		WatchID:          "w-match",
		UserID:           user,
		Region:           "us-east-1",
		AvailabilityZone: "us-east-1a",
		InstanceType:     "g5.xlarge",
		Price:            0.75,
		IsSpot:           true,
		MatchedAt:        time.Now().UTC(),
		ActionTaken:      "notified",
	}

	if err := store.RecordMatch(ctx, w, m); err != nil {
		t.Fatalf("RecordMatch: %v", err)
	}

	// Verify watch was updated
	got, _ := store.GetWatch(ctx, "w-match")
	if got.MatchCount != 1 {
		t.Errorf("MatchCount = %d, want 1", got.MatchCount)
	}
	if got.LastMatch == nil {
		t.Fatal("LastMatch is nil")
	}
	if got.LastMatch.InstanceType != "g5.xlarge" {
		t.Errorf("LastMatch.InstanceType = %q, want g5.xlarge", got.LastMatch.InstanceType)
	}

	// Verify history was written
	history, err := store.ListMatchHistory(ctx, "w-match")
	if err != nil {
		t.Fatalf("ListMatchHistory: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("got %d history records, want 1", len(history))
	}
}

func TestUpdateLastPolled(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	w := newTestWatch("w-poll", "arn:aws:iam::123456789012:user/test")
	_ = store.PutWatch(ctx, w)

	if err := store.UpdateLastPolled(ctx, "w-poll"); err != nil {
		t.Fatalf("UpdateLastPolled: %v", err)
	}

	// Just verify no error — the timestamp is set server-side
}

func TestExtendWatch(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	w := newTestWatch("w-extend", "arn:aws:iam::123456789012:user/test")
	_ = store.PutWatch(ctx, w)

	newExpiry := time.Now().UTC().Add(48 * time.Hour)
	if err := store.ExtendWatch(ctx, "w-extend", newExpiry, false); err != nil {
		t.Fatalf("ExtendWatch: %v", err)
	}

	got, _ := store.GetWatch(ctx, "w-extend")
	if got.TTLTimestamp != newExpiry.Unix() {
		t.Errorf("TTLTimestamp = %d, want %d", got.TTLTimestamp, newExpiry.Unix())
	}
}

func TestExtendWatch_Reactivate(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	w := newTestWatch("w-reactivate", "arn:aws:iam::123456789012:user/test")
	w.Status = watcher.StatusExpired
	_ = store.PutWatch(ctx, w)

	newExpiry := time.Now().UTC().Add(24 * time.Hour)
	if err := store.ExtendWatch(ctx, "w-reactivate", newExpiry, true); err != nil {
		t.Fatalf("ExtendWatch reactivate: %v", err)
	}

	got, _ := store.GetWatch(ctx, "w-reactivate")
	if got.Status != watcher.StatusActive {
		t.Errorf("Status = %q, want active", got.Status)
	}
}
