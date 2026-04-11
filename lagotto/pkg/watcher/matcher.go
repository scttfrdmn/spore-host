package watcher

import (
	truffleaws "github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

// MatchCandidate represents a potential capacity match to evaluate against a watch.
type MatchCandidate struct {
	InstanceType truffleaws.InstanceTypeResult
	SpotPrice    *truffleaws.SpotPriceResult // nil if not a Spot check
}

// Evaluate checks whether a candidate satisfies the watch criteria.
// Returns a MatchResult if the candidate matches, nil otherwise.
func Evaluate(w *Watch, c MatchCandidate) *MatchResult {
	// If the watch requires Spot and we have Spot pricing, check price
	if w.Spot && c.SpotPrice != nil {
		if w.MaxPrice > 0 && c.SpotPrice.SpotPrice > w.MaxPrice {
			return nil
		}
		return &MatchResult{
			WatchID:          w.WatchID,
			UserID:           w.UserID,
			Region:           c.SpotPrice.Region,
			AvailabilityZone: c.SpotPrice.AvailabilityZone,
			InstanceType:     c.SpotPrice.InstanceType,
			Price:            c.SpotPrice.SpotPrice,
			IsSpot:           true,
			ActionTaken:      "pending",
		}
	}

	// On-demand availability check: instance type exists in the region
	if !w.Spot {
		price := c.InstanceType.OnDemandPrice
		if w.MaxPrice > 0 && price > w.MaxPrice {
			return nil
		}
		az := ""
		if len(c.InstanceType.AvailableAZs) > 0 {
			az = c.InstanceType.AvailableAZs[0]
		}
		return &MatchResult{
			WatchID:          w.WatchID,
			UserID:           w.UserID,
			Region:           c.InstanceType.Region,
			AvailabilityZone: az,
			InstanceType:     c.InstanceType.InstanceType,
			Price:            price,
			IsSpot:           false,
			ActionTaken:      "pending",
		}
	}

	// Spot watch but no pricing data available — no match
	return nil
}
