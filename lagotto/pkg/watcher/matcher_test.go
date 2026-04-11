package watcher

import (
	"testing"

	truffleaws "github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

func TestEvaluate_OnDemand_NoMaxPrice(t *testing.T) {
	w := &Watch{
		WatchID:             "w-test1",
		UserID:              "arn:aws:iam::123456789012:user/test",
		InstanceTypePattern: "g5.xlarge",
		Spot:                false,
	}
	candidate := MatchCandidate{
		InstanceType: truffleaws.InstanceTypeResult{
			InstanceType:  "g5.xlarge",
			Region:        "us-east-1",
			AvailableAZs:  []string{"us-east-1a", "us-east-1b"},
			OnDemandPrice: 1.006,
		},
	}

	m := Evaluate(w, candidate)
	if m == nil {
		t.Fatal("expected match, got nil")
	}
	if m.InstanceType != "g5.xlarge" {
		t.Errorf("instance type = %q, want g5.xlarge", m.InstanceType)
	}
	if m.Region != "us-east-1" {
		t.Errorf("region = %q, want us-east-1", m.Region)
	}
	if m.AvailabilityZone != "us-east-1a" {
		t.Errorf("az = %q, want us-east-1a", m.AvailabilityZone)
	}
	if m.Price != 1.006 {
		t.Errorf("price = %f, want 1.006", m.Price)
	}
	if m.IsSpot {
		t.Error("expected IsSpot = false")
	}
}

func TestEvaluate_OnDemand_MaxPriceExceeded(t *testing.T) {
	w := &Watch{
		WatchID:  "w-test2",
		Spot:     false,
		MaxPrice: 0.50,
	}
	candidate := MatchCandidate{
		InstanceType: truffleaws.InstanceTypeResult{
			InstanceType:  "g5.xlarge",
			Region:        "us-east-1",
			OnDemandPrice: 1.006,
		},
	}

	m := Evaluate(w, candidate)
	if m != nil {
		t.Errorf("expected no match when price exceeds max, got %+v", m)
	}
}

func TestEvaluate_OnDemand_MaxPriceMet(t *testing.T) {
	w := &Watch{
		WatchID:  "w-test3",
		Spot:     false,
		MaxPrice: 2.00,
	}
	candidate := MatchCandidate{
		InstanceType: truffleaws.InstanceTypeResult{
			InstanceType:  "g5.xlarge",
			Region:        "us-west-2",
			OnDemandPrice: 1.006,
		},
	}

	m := Evaluate(w, candidate)
	if m == nil {
		t.Fatal("expected match, got nil")
	}
	if m.Region != "us-west-2" {
		t.Errorf("region = %q, want us-west-2", m.Region)
	}
}

func TestEvaluate_Spot_Match(t *testing.T) {
	w := &Watch{
		WatchID:  "w-spot1",
		UserID:   "arn:aws:iam::123456789012:user/test",
		Spot:     true,
		MaxPrice: 0.50,
	}
	spotPrice := &truffleaws.SpotPriceResult{
		InstanceType:     "g5.xlarge",
		Region:           "us-east-1",
		AvailabilityZone: "us-east-1c",
		SpotPrice:        0.30,
	}
	candidate := MatchCandidate{
		SpotPrice: spotPrice,
	}

	m := Evaluate(w, candidate)
	if m == nil {
		t.Fatal("expected match, got nil")
	}
	if !m.IsSpot {
		t.Error("expected IsSpot = true")
	}
	if m.Price != 0.30 {
		t.Errorf("price = %f, want 0.30", m.Price)
	}
	if m.AvailabilityZone != "us-east-1c" {
		t.Errorf("az = %q, want us-east-1c", m.AvailabilityZone)
	}
}

func TestEvaluate_Spot_PriceExceeded(t *testing.T) {
	w := &Watch{
		WatchID:  "w-spot2",
		Spot:     true,
		MaxPrice: 0.20,
	}
	spotPrice := &truffleaws.SpotPriceResult{
		InstanceType: "g5.xlarge",
		SpotPrice:    0.30,
	}
	candidate := MatchCandidate{
		SpotPrice: spotPrice,
	}

	m := Evaluate(w, candidate)
	if m != nil {
		t.Errorf("expected no match when spot price exceeds max, got %+v", m)
	}
}

func TestEvaluate_Spot_NoSpotData(t *testing.T) {
	w := &Watch{
		WatchID: "w-spot3",
		Spot:    true,
	}
	candidate := MatchCandidate{
		InstanceType: truffleaws.InstanceTypeResult{
			InstanceType: "g5.xlarge",
			Region:       "us-east-1",
		},
		// No SpotPrice
	}

	m := Evaluate(w, candidate)
	if m != nil {
		t.Errorf("expected no match when spot watch has no spot data, got %+v", m)
	}
}

func TestEvaluate_Spot_NoMaxPrice(t *testing.T) {
	w := &Watch{
		WatchID:  "w-spot-any",
		UserID:   "arn:aws:iam::123456789012:user/test",
		Spot:     true,
		MaxPrice: 0, // any price
	}
	spotPrice := &truffleaws.SpotPriceResult{
		InstanceType:     "p5.48xlarge",
		Region:           "us-east-1",
		AvailabilityZone: "us-east-1b",
		SpotPrice:        98.50,
	}
	candidate := MatchCandidate{SpotPrice: spotPrice}

	m := Evaluate(w, candidate)
	if m == nil {
		t.Fatal("expected match with MaxPrice=0 (any), got nil")
	}
	if m.Price != 98.50 {
		t.Errorf("price = %f, want 98.50", m.Price)
	}
}

func TestEvaluate_OnDemand_NoAZs(t *testing.T) {
	w := &Watch{
		WatchID: "w-noaz",
		Spot:    false,
	}
	candidate := MatchCandidate{
		InstanceType: truffleaws.InstanceTypeResult{
			InstanceType:  "g5.xlarge",
			Region:        "us-east-1",
			AvailableAZs:  nil, // empty
			OnDemandPrice: 1.006,
		},
	}

	m := Evaluate(w, candidate)
	if m == nil {
		t.Fatal("expected match even with no AZs, got nil")
	}
	if m.AvailabilityZone != "" {
		t.Errorf("az = %q, want empty", m.AvailabilityZone)
	}
}

func TestWildcardToRegex(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{"p5.*", `^p5\..*$`},
		{"g5.xlarge", `^g5\.xlarge$`},
		{"t3.micro", `^t3\.micro$`},
		{"^p5\\..*$", `^p5\..*$`}, // already regex, pass through
	}
	for _, tt := range tests {
		got := wildcardToRegex(tt.pattern)
		if got != tt.want {
			t.Errorf("wildcardToRegex(%q) = %q, want %q", tt.pattern, got, tt.want)
		}
	}
}
