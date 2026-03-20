package aws

import (
	"context"
	"regexp"
	"testing"

	"github.com/scttfrdmn/spore-host/truffle/pkg/testutil"
)

func TestGetEnabledRegions(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	regions, err := c.GetEnabledRegions(ctx)
	if err != nil {
		t.Fatalf("GetEnabledRegions() error = %v", err)
	}
	if len(regions) == 0 {
		t.Error("GetEnabledRegions() returned 0 regions, want >= 1")
	}
	// Substrate seeds us-east-1 by default.
	found := false
	for _, r := range regions {
		if r == "us-east-1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("us-east-1 not in enabled regions: %v", regions)
	}
}

func TestGetInstanceTypes(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	types, err := c.GetInstanceTypes(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("GetInstanceTypes() error = %v", err)
	}
	if len(types) == 0 {
		t.Fatal("GetInstanceTypes() returned 0 instance types, want >= 1")
	}

	// Verify the seeded catalog includes t3.micro.
	found := false
	for _, it := range types {
		if it == "t3.micro" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("t3.micro not in seeded instance types: %v", types)
	}
}

func TestSearchInstanceTypes_FilterByVCPU(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	// Match all types (empty regex matches everything).
	matcher := regexp.MustCompile(".*")
	results, err := c.SearchInstanceTypes(ctx, []string{"us-east-1"}, matcher, FilterOptions{
		MinVCPUs: 4, // Only 4+ vCPU types
	})
	if err != nil {
		t.Fatalf("SearchInstanceTypes() error = %v", err)
	}
	for _, r := range results {
		if r.VCPUs < 4 {
			t.Errorf("result %s has %d vCPUs, want >= 4", r.InstanceType, r.VCPUs)
		}
	}
}

func TestSearchInstanceTypes_IncludeAZs(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	matcher := regexp.MustCompile(`^t3\.micro$`)
	results, err := c.SearchInstanceTypes(ctx, []string{"us-east-1"}, matcher, FilterOptions{
		IncludeAZs: true,
	})
	if err != nil {
		t.Fatalf("SearchInstanceTypes() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("SearchInstanceTypes() returned 0 results for t3.micro, want 1")
	}
	r := results[0]
	if r.InstanceType != "t3.micro" {
		t.Errorf("result.InstanceType = %q, want %q", r.InstanceType, "t3.micro")
	}
	if len(r.AvailableAZs) == 0 {
		t.Error("AvailableAZs is empty with IncludeAZs=true")
	}
}

func TestGetSpotPricing(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	// First fetch instance types to build the input slice.
	matcher := regexp.MustCompile(`^t3\.micro$`)
	instances, err := c.SearchInstanceTypes(ctx, []string{"us-east-1"}, matcher, FilterOptions{})
	if err != nil {
		t.Fatalf("SearchInstanceTypes() error = %v", err)
	}
	if len(instances) == 0 {
		t.Skip("t3.micro not in seeded catalog — skipping Spot pricing test")
	}

	prices, err := c.GetSpotPricing(ctx, instances, SpotOptions{
		LookbackHours: 24,
	})
	if err != nil {
		t.Fatalf("GetSpotPricing() error = %v", err)
	}
	if len(prices) == 0 {
		t.Fatal("GetSpotPricing() returned 0 prices for t3.micro, want >= 1")
	}
	for _, p := range prices {
		if p.SpotPrice <= 0 {
			t.Errorf("SpotPrice for %s = %f, want > 0", p.InstanceType, p.SpotPrice)
		}
	}
}

func TestGetSpotPricing_MaxPriceFilter(t *testing.T) {
	env := testutil.SubstrateServer(t)
	ctx := context.Background()

	c := NewClientFromConfig(env.AWSConfig)

	matcher := regexp.MustCompile(".*")
	instances, err := c.SearchInstanceTypes(ctx, []string{"us-east-1"}, matcher, FilterOptions{})
	if err != nil {
		t.Fatalf("SearchInstanceTypes() error = %v", err)
	}
	if len(instances) == 0 {
		t.Skip("no seeded instance types")
	}

	// Very low max price — should filter out most (or all) results.
	prices, err := c.GetSpotPricing(ctx, instances, SpotOptions{
		LookbackHours: 24,
		MaxPrice:      0.001, // $0.001/hr — below all seeded prices
	})
	if err != nil {
		t.Fatalf("GetSpotPricing() error = %v", err)
	}
	for _, p := range prices {
		if p.SpotPrice > 0.001 {
			t.Errorf("price %f exceeded MaxPrice 0.001 for %s", p.SpotPrice, p.InstanceType)
		}
	}
}
