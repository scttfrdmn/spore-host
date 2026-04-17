package watcher

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"context"
)

// TestSendWebhook_Payload tests the HTTP dispatch directly (bypasses URL validation
// since httptest servers are always http://).
func TestSendWebhook_Payload(t *testing.T) {
	var received map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	n := &Notifier{httpClient: ts.Client()}
	w := &Watch{WatchID: "w-hook1", InstanceTypePattern: "p5.*"}
	m := &MatchResult{
		WatchID:          "w-hook1",
		Region:           "us-west-2",
		AvailabilityZone: "us-west-2a",
		InstanceType:     "p5.48xlarge",
		Price:            32.77,
		IsSpot:           true,
		MatchedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		ActionTaken:      "notified",
	}

	// Call sendWebhook directly to test payload without URL validation
	if err := n.sendWebhook(context.Background(), ts.URL, w, m); err != nil {
		t.Fatalf("sendWebhook: %v", err)
	}
	if received == nil {
		t.Fatal("webhook was not called")
	}
	if got := received["instance_type"]; got != "p5.48xlarge" {
		t.Errorf("instance_type = %v, want p5.48xlarge", got)
	}
	if got := received["price"].(float64); got != 32.77 {
		t.Errorf("price = %v, want 32.77", got)
	}
}

// TestNotifyWebhook_RejectsHTTP verifies that Notify blocks http:// webhook targets.
func TestNotifyWebhook_RejectsHTTP(t *testing.T) {
	n := &Notifier{httpClient: http.DefaultClient}
	w := &Watch{
		WatchID: "w-http-reject",
		NotifyChannels: []NotifyChannel{
			{Type: "webhook", Target: "http://evil.example.com/steal"},
		},
	}
	m := &MatchResult{MatchedAt: time.Now()}
	err := n.Notify(context.Background(), w, m)
	if err == nil {
		t.Fatal("expected Notify to reject http:// webhook URL")
	}
}

func TestNotifyWebhook_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	n := &Notifier{httpClient: ts.Client()}
	w := &Watch{
		WatchID:        "w-err",
		NotifyChannels: []NotifyChannel{{Type: "webhook", Target: ts.URL}},
	}
	m := &MatchResult{InstanceType: "t3.micro", MatchedAt: time.Now()}

	err := n.Notify(context.Background(), w, m)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestNotifyNoChannels(t *testing.T) {
	n := &Notifier{}
	w := &Watch{WatchID: "w-empty"}
	m := &MatchResult{}

	err := n.Notify(context.Background(), w, m)
	if err != nil {
		t.Errorf("expected nil error for no channels, got %v", err)
	}
}

// TODO: SNS tests require Substrate emulator or mock SNS client
