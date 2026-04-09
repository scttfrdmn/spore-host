package pricing

import (
	"fmt"
	"strings"
)

// EC2Pricing holds hourly rates for EC2 instance types by region
// Prices are approximate On-Demand rates as of 2026-01 (subject to change)
var EC2Pricing = map[string]map[string]float64{
	"us-east-1": {
		// General Purpose
		"t3.micro":    0.0104,
		"t3.small":    0.0208,
		"t3.medium":   0.0416,
		"t3.large":    0.0832,
		"t3.xlarge":   0.1664,
		"t3.2xlarge":  0.3328,
		"t4g.micro":   0.0084,
		"t4g.small":   0.0168,
		"t4g.medium":  0.0336,
		"t4g.large":   0.0672,
		"t4g.xlarge":  0.1344,
		"t4g.2xlarge": 0.2688,
		"m5.large":    0.096,
		"m5.xlarge":   0.192,
		"m5.2xlarge":  0.384,
		"m5.4xlarge":  0.768,
		"m5.8xlarge":  1.536,
		"m6i.large":   0.096,
		"m6i.xlarge":  0.192,
		"m6i.2xlarge": 0.384,
		"m6i.4xlarge": 0.768,
		"m7i.large":   0.1008,
		"m7i.xlarge":  0.2016,
		"m7i.2xlarge": 0.4032,
		// Compute Optimized
		"c5.large":    0.085,
		"c5.xlarge":   0.17,
		"c5.2xlarge":  0.34,
		"c5.4xlarge":  0.68,
		"c5.9xlarge":  1.53,
		"c6i.large":   0.085,
		"c6i.xlarge":  0.17,
		"c6i.2xlarge": 0.34,
		"c6i.4xlarge": 0.68,
		"c7i.large":   0.0893,
		"c7i.xlarge":  0.1785,
		"c7i.2xlarge": 0.357,
		"c7i.4xlarge": 0.714,
		// Memory Optimized
		"r5.large":    0.126,
		"r5.xlarge":   0.252,
		"r5.2xlarge":  0.504,
		"r5.4xlarge":  1.008,
		"r6i.large":   0.126,
		"r6i.xlarge":  0.252,
		"r6i.2xlarge": 0.504,
		// GPU
		"g4dn.xlarge":  0.526,
		"g4dn.2xlarge": 0.752,
		"g5.xlarge":    1.006,
		"g5.2xlarge":   1.212,
		"p3.2xlarge":   3.06,
		"p4d.24xlarge": 32.77,
	},
	"us-east-2": {
		"t3.micro":    0.0104,
		"t3.small":    0.0208,
		"t3.medium":   0.0416,
		"t3.large":    0.0832,
		"m5.large":    0.096,
		"m5.xlarge":   0.192,
		"c5.large":    0.085,
		"c5.xlarge":   0.17,
		"c7i.xlarge":  0.1785,
		"c7i.4xlarge": 0.714,
		"r5.large":    0.126,
		"r5.xlarge":   0.252,
	},
	"us-west-1": {
		"t3.micro":    0.0116,
		"t3.small":    0.0232,
		"t3.medium":   0.0464,
		"t3.large":    0.0928,
		"m5.large":    0.107,
		"m5.xlarge":   0.214,
		"c5.large":    0.094,
		"c5.xlarge":   0.188,
		"c7i.xlarge":  0.199,
		"c7i.4xlarge": 0.796,
	},
	"us-west-2": {
		"t3.micro":    0.0104,
		"t3.small":    0.0208,
		"t3.medium":   0.0416,
		"t3.large":    0.0832,
		"m5.large":    0.096,
		"m5.xlarge":   0.192,
		"m6i.xlarge":  0.192,
		"c5.large":    0.085,
		"c5.xlarge":   0.17,
		"c7i.xlarge":  0.1785,
		"c7i.4xlarge": 0.714,
		"r5.large":    0.126,
		"r5.xlarge":   0.252,
		"g4dn.xlarge": 0.526,
		"g5.xlarge":   1.006,
	},
	"eu-west-1": {
		"t3.micro":   0.0114,
		"t3.small":   0.0228,
		"t3.medium":  0.0456,
		"t3.large":   0.0912,
		"m5.large":   0.107,
		"m5.xlarge":  0.214,
		"c5.large":   0.094,
		"c5.xlarge":  0.188,
		"c7i.xlarge": 0.199,
		"r5.large":   0.14,
		"r5.xlarge":  0.28,
	},
	"eu-central-1": {
		"t3.micro":   0.012,
		"t3.small":   0.024,
		"t3.medium":  0.048,
		"t3.large":   0.096,
		"m5.large":   0.113,
		"m5.xlarge":  0.226,
		"c5.large":   0.099,
		"c5.xlarge":  0.198,
		"c7i.xlarge": 0.21,
		"r5.large":   0.148,
	},
	"ap-southeast-1": {
		"t3.micro":  0.0116,
		"t3.small":  0.0232,
		"t3.medium": 0.0464,
		"t3.large":  0.0928,
		"m5.large":  0.107,
		"m5.xlarge": 0.214,
		"c5.large":  0.094,
		"c5.xlarge": 0.188,
	},
	"ap-northeast-1": {
		"t3.micro":  0.0128,
		"t3.small":  0.0256,
		"t3.medium": 0.0512,
		"t3.large":  0.1024,
		"m5.large":  0.118,
		"m5.xlarge": 0.236,
		"c5.large":  0.103,
		"c5.xlarge": 0.206,
	},
}

// GetEC2HourlyRate returns the hourly On-Demand rate for an instance type in a region
// Returns a default estimate if exact pricing not found
func GetEC2HourlyRate(region, instanceType string) float64 {
	// Normalize region and instance type
	region = strings.ToLower(strings.TrimSpace(region))
	instanceType = strings.ToLower(strings.TrimSpace(instanceType))

	// Check if we have pricing for this region
	regionPricing, ok := EC2Pricing[region]
	if !ok {
		// Use us-east-1 as fallback for unknown regions
		regionPricing = EC2Pricing["us-east-1"]
	}

	// Check if we have pricing for this instance type
	if price, ok := regionPricing[instanceType]; ok {
		return price
	}

	// Fallback: estimate based on instance family
	return estimatePriceByFamily(instanceType)
}

// estimatePriceByFamily provides rough estimates for instance types not in the table
func estimatePriceByFamily(instanceType string) float64 {
	// Extract family (e.g., "t3" from "t3.xlarge")
	parts := strings.Split(instanceType, ".")
	if len(parts) < 2 {
		return 0.10 // Default estimate: $0.10/hour
	}

	family := parts[0]
	size := parts[1]

	// Base prices by family (approximate)
	basePrice := map[string]float64{
		"t2":   0.0104,
		"t3":   0.0104,
		"t3a":  0.0094,
		"t4g":  0.0084,
		"m5":   0.096,
		"m5a":  0.086,
		"m5n":  0.119,
		"m6i":  0.096,
		"m6a":  0.086,
		"m7i":  0.1008,
		"c5":   0.085,
		"c5a":  0.077,
		"c5n":  0.108,
		"c6i":  0.085,
		"c6a":  0.077,
		"c7i":  0.0893,
		"r5":   0.126,
		"r5a":  0.113,
		"r6i":  0.126,
		"g4dn": 0.526,
		"g5":   1.006,
		"p3":   3.06,
		"p4":   32.77,
	}

	// Size multipliers
	sizeMultiplier := map[string]float64{
		"nano":     0.25,
		"micro":    1.0,
		"small":    2.0,
		"medium":   4.0,
		"large":    8.0,
		"xlarge":   16.0,
		"2xlarge":  32.0,
		"4xlarge":  64.0,
		"8xlarge":  128.0,
		"12xlarge": 192.0,
		"16xlarge": 256.0,
		"24xlarge": 384.0,
	}

	base, ok := basePrice[family]
	if !ok {
		base = 0.10 // Generic default
	}

	multiplier, ok := sizeMultiplier[size]
	if !ok {
		multiplier = 16.0 // Assume xlarge if unknown
	}

	return base * multiplier
}

// FormatCost formats a cost value as a currency string
func FormatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

// FormatCostDetailed formats cost with more precision for small amounts
func FormatCostDetailed(cost float64) string {
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}
