//go:build integration
// +build integration

package registry

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/scttfrdmn/spore-host/spawn/pkg/provider"
)

// MockDynamoDBClient would go here for proper unit tests
// For now, these are integration test examples that require real DynamoDB

func TestPeerRegistry_RegisterAndDiscover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create test identity
	identity := &provider.Identity{
		InstanceID: "test-instance-01",
		Region:     "us-east-1",
		AccountID:  "123456789012",
		Provider:   "local",
		PublicIP:   "192.168.1.100",
		PrivateIP:  "192.168.1.100",
	}

	// Create registry
	registry, err := NewPeerRegistry(ctx, identity)
	if err != nil {
		t.Skipf("Skipping test: %v (AWS credentials not available?)", err)
	}

	jobArrayID := "test-array-" + time.Now().Format("20060102-150405")

	// Register instance
	err = registry.Register(ctx, jobArrayID, 0)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Discover peers (should find ourselves)
	peers, err := registry.DiscoverPeers(ctx, jobArrayID)
	if err != nil {
		t.Fatalf("DiscoverPeers() error = %v", err)
	}

	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	if len(peers) > 0 {
		if peers[0].InstanceID != identity.InstanceID {
			t.Errorf("InstanceID = %v, want %v", peers[0].InstanceID, identity.InstanceID)
		}
		if peers[0].Provider != identity.Provider {
			t.Errorf("Provider = %v, want %v", peers[0].Provider, identity.Provider)
		}
		if peers[0].IP != identity.PublicIP {
			t.Errorf("IP = %v, want %v", peers[0].IP, identity.PublicIP)
		}
	}

	// Cleanup
	err = registry.Deregister(ctx, jobArrayID)
	if err != nil {
		t.Errorf("Deregister() error = %v", err)
	}
}

func TestPeerRegistry_Heartbeat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	identity := &provider.Identity{
		InstanceID: "test-heartbeat-01",
		Region:     "us-east-1",
		Provider:   "local",
		PublicIP:   "192.168.1.101",
	}

	registry, err := NewPeerRegistry(ctx, identity)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	jobArrayID := "test-heartbeat-" + time.Now().Format("20060102-150405")

	// Register
	err = registry.Register(ctx, jobArrayID, 0)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	defer registry.Deregister(ctx, jobArrayID)

	// Send heartbeat
	err = registry.Heartbeat(ctx, jobArrayID)
	if err != nil {
		t.Errorf("Heartbeat() error = %v", err)
	}

	// Verify still discoverable
	peers, err := registry.DiscoverPeers(ctx, jobArrayID)
	if err != nil {
		t.Fatalf("DiscoverPeers() error = %v", err)
	}

	if len(peers) != 1 {
		t.Errorf("Expected 1 peer after heartbeat, got %d", len(peers))
	}
}

func TestPeerRegistry_TTLExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	identity := &provider.Identity{
		InstanceID: "test-ttl-01",
		Region:     "us-east-1",
		Provider:   "local",
		PublicIP:   "192.168.1.102",
	}

	registry, err := NewPeerRegistry(ctx, identity)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Set very short TTL for testing
	registry.ttl = 2 // 2 seconds

	jobArrayID := "test-ttl-" + time.Now().Format("20060102-150405")

	// Register
	err = registry.Register(ctx, jobArrayID, 0)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	defer registry.Deregister(ctx, jobArrayID)

	// Wait for TTL to expire
	time.Sleep(3 * time.Second)

	// Should not find expired peer
	peers, err := registry.DiscoverPeers(ctx, jobArrayID)
	if err != nil {
		t.Fatalf("DiscoverPeers() error = %v", err)
	}

	if len(peers) != 0 {
		t.Errorf("Expected 0 peers after TTL expiration, got %d", len(peers))
	}
}

func TestPeerRegistry_MultipleInstances(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	jobArrayID := "test-multi-" + time.Now().Format("20060102-150405")

	// Create two registries with different identities
	identity1 := &provider.Identity{
		InstanceID: "test-multi-01",
		Region:     "us-east-1",
		Provider:   "local",
		PublicIP:   "192.168.1.103",
	}

	identity2 := &provider.Identity{
		InstanceID: "test-multi-02",
		Region:     "us-east-1",
		Provider:   "ec2",
		PublicIP:   "54.1.2.3",
	}

	registry1, err := NewPeerRegistry(ctx, identity1)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	registry2, err := NewPeerRegistry(ctx, identity2)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Register both
	err = registry1.Register(ctx, jobArrayID, 0)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	defer registry1.Deregister(ctx, jobArrayID)

	err = registry2.Register(ctx, jobArrayID, 1)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	defer registry2.Deregister(ctx, jobArrayID)

	// Discover from first registry
	peers, err := registry1.DiscoverPeers(ctx, jobArrayID)
	if err != nil {
		t.Fatalf("DiscoverPeers() error = %v", err)
	}

	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(peers))
	}

	// Verify both instances present
	foundLocal := false
	foundEC2 := false
	for _, peer := range peers {
		if peer.Provider == "local" {
			foundLocal = true
		}
		if peer.Provider == "ec2" {
			foundEC2 = true
		}
	}

	if !foundLocal {
		t.Errorf("Did not find local provider instance")
	}
	if !foundEC2 {
		t.Errorf("Did not find ec2 provider instance")
	}
}

func TestDiscoverPeersForJobArray(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	jobArrayID := "test-discover-" + time.Now().Format("20060102-150405")

	// Register an instance first
	identity := &provider.Identity{
		InstanceID: "test-discover-01",
		Region:     "us-east-1",
		Provider:   "local",
		PublicIP:   "192.168.1.104",
	}

	registry, err := NewPeerRegistry(ctx, identity)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	err = registry.Register(ctx, jobArrayID, 0)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	defer registry.Deregister(ctx, jobArrayID)

	// Use DiscoverPeersForJobArray (orchestrator use case)
	cfg, err := createTestAWSConfig(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	peers, err := DiscoverPeersForJobArray(ctx, cfg, jobArrayID)
	if err != nil {
		t.Fatalf("DiscoverPeersForJobArray() error = %v", err)
	}

	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}
}

// Helper function to create AWS config for testing
func createTestAWSConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx)
}
