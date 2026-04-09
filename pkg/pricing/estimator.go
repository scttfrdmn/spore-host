package pricing

import (
	"fmt"
	"time"
)

// CostEstimate represents the estimated cost breakdown for a parameter sweep
type CostEstimate struct {
	ComputeCost float64 // Total EC2 compute cost
	LambdaCost  float64 // Lambda orchestration cost
	StorageCost float64 // S3 parameter storage cost
	TotalCost   float64 // Sum of all costs
}

// ParamFileFormat matches the sweep parameter file structure
// Duplicated here to avoid circular dependency
type ParamFileFormat struct {
	Defaults map[string]interface{}   `json:"defaults"`
	Params   []map[string]interface{} `json:"params"`
}

// EstimateSweepCost calculates the estimated cost for a parameter sweep
func EstimateSweepCost(params *ParamFileFormat) (*CostEstimate, error) {
	estimate := &CostEstimate{}

	// Calculate compute cost for each parameter set
	for i, paramSet := range params.Params {
		// Get instance type (from param set or defaults)
		instanceType := getStringValue(paramSet, "instance_type", "")
		if instanceType == "" {
			if defaults, ok := params.Defaults["instance_type"].(string); ok {
				instanceType = defaults
			} else {
				return nil, fmt.Errorf("param set %d: no instance_type specified", i)
			}
		}

		// Get region (from param set or defaults)
		region := getStringValue(paramSet, "region", "")
		if region == "" {
			if defaults, ok := params.Defaults["region"].(string); ok {
				region = defaults
			} else {
				region = "us-east-1" // Default region
			}
		}

		// Get TTL (from param set or defaults)
		ttlStr := getStringValue(paramSet, "ttl", "")
		if ttlStr == "" {
			if defaults, ok := params.Defaults["ttl"].(string); ok {
				ttlStr = defaults
			} else {
				ttlStr = "1h" // Default 1 hour
			}
		}

		// Parse TTL duration
		ttl, err := time.ParseDuration(ttlStr)
		if err != nil {
			return nil, fmt.Errorf("param set %d: invalid ttl format %s: %w", i, ttlStr, err)
		}
		hours := ttl.Hours()

		// Get hourly rate
		hourlyRate := GetEC2HourlyRate(region, instanceType)

		// Add to total compute cost
		estimate.ComputeCost += hourlyRate * hours
	}

	// Lambda cost estimation
	// Lambda: $0.0000166667 per GB-second
	// Assume 512MB memory, estimate 5 minutes runtime per 10 instances
	numParams := len(params.Params)
	estimatedLambdaSeconds := float64(numParams) * 30.0 // 30 seconds per instance estimate
	if estimatedLambdaSeconds < 300 {
		estimatedLambdaSeconds = 300 // Minimum 5 minutes
	}
	memorySizeGB := 512.0 / 1024.0
	estimate.LambdaCost = 0.0000166667 * memorySizeGB * estimatedLambdaSeconds

	// S3 storage cost (very small, usually negligible)
	// $0.023 per GB per month, assume params file < 1MB, prorated for 1 day
	estimate.StorageCost = 0.023 * 0.001 * (1.0 / 30.0)

	// Total cost
	estimate.TotalCost = estimate.ComputeCost + estimate.LambdaCost + estimate.StorageCost

	return estimate, nil
}

// DisplayCostEstimate formats and displays the cost estimate
func (e *CostEstimate) Display() string {
	return fmt.Sprintf(`Estimated cost for this sweep:
  Compute (EC2):  %s
  Lambda:         %s (orchestration)
  Storage (S3):   %s (parameters)
  ────────────────────────────────
  Total:          %s`,
		FormatCost(e.ComputeCost),
		FormatCostDetailed(e.LambdaCost),
		FormatCostDetailed(e.StorageCost),
		FormatCost(e.TotalCost))
}

// DisplayCompact formats the cost estimate in a compact format
func (e *CostEstimate) DisplayCompact() string {
	return fmt.Sprintf("Total estimated cost: %s (EC2: %s, Lambda: %s, S3: %s)",
		FormatCost(e.TotalCost),
		FormatCost(e.ComputeCost),
		FormatCostDetailed(e.LambdaCost),
		FormatCostDetailed(e.StorageCost))
}

// Helper function to get string values from param map
func getStringValue(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}
