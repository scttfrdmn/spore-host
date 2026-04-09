package pricing

import (
	"strings"
	"testing"
)

// floatEqual checks if two floats are approximately equal (within epsilon)
func floatEqual(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func TestGetEC2HourlyRate(t *testing.T) {
	tests := []struct {
		name         string
		region       string
		instanceType string
		expected     float64
	}{
		{
			name:         "us-east-1 t3.micro (exact match)",
			region:       "us-east-1",
			instanceType: "t3.micro",
			expected:     0.0104,
		},
		{
			name:         "us-east-1 c5.xlarge (exact match)",
			region:       "us-east-1",
			instanceType: "c5.xlarge",
			expected:     0.17,
		},
		{
			name:         "eu-west-1 m5.large (exact match)",
			region:       "eu-west-1",
			instanceType: "m5.large",
			expected:     0.107,
		},
		{
			name:         "Case insensitive region",
			region:       "US-EAST-1",
			instanceType: "t3.micro",
			expected:     0.0104,
		},
		{
			name:         "Case insensitive instance type",
			region:       "us-east-1",
			instanceType: "T3.MICRO",
			expected:     0.0104,
		},
		{
			name:         "Whitespace trimming",
			region:       "  us-east-1  ",
			instanceType: "  t3.micro  ",
			expected:     0.0104,
		},
		{
			name:         "Unknown region (fallback to us-east-1)",
			region:       "unknown-region",
			instanceType: "t3.micro",
			expected:     0.0104, // Uses us-east-1 pricing
		},
		{
			name:         "Unknown instance type (uses estimation)",
			region:       "us-east-1",
			instanceType: "unknown.xlarge",
			expected:     1.6, // Estimated: 0.10 (default base) * 16.0 (xlarge multiplier)
		},
		{
			name:         "Region exists but instance type missing (uses estimation)",
			region:       "us-east-1",
			instanceType: "m7i.12xlarge",
			expected:     19.353600000000002, // Estimated: 0.1008 * 192.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEC2HourlyRate(tt.region, tt.instanceType)
			if !floatEqual(result, tt.expected, 0.00001) {
				t.Errorf("expected %.4f, got %.4f", tt.expected, result)
			}
		})
	}
}

func TestEstimatePriceByFamily(t *testing.T) {
	tests := []struct {
		name         string
		instanceType string
		expected     float64
	}{
		{
			name:         "t3.micro",
			instanceType: "t3.micro",
			expected:     0.0104, // 0.0104 * 1.0
		},
		{
			name:         "t3.small",
			instanceType: "t3.small",
			expected:     0.0208, // 0.0104 * 2.0
		},
		{
			name:         "t3.xlarge",
			instanceType: "t3.xlarge",
			expected:     0.1664, // 0.0104 * 16.0
		},
		{
			name:         "m5.large",
			instanceType: "m5.large",
			expected:     0.768, // 0.096 * 8.0
		},
		{
			name:         "c5.4xlarge",
			instanceType: "c5.4xlarge",
			expected:     5.44, // 0.085 * 64.0
		},
		{
			name:         "r5.2xlarge",
			instanceType: "r5.2xlarge",
			expected:     4.032, // 0.126 * 32.0
		},
		{
			name:         "g5.xlarge",
			instanceType: "g5.xlarge",
			expected:     16.096, // 1.006 * 16.0
		},
		{
			name:         "Unknown family (uses default)",
			instanceType: "unknown.xlarge",
			expected:     1.6, // 0.10 * 16.0
		},
		{
			name:         "Unknown size (uses xlarge default)",
			instanceType: "t3.unknown",
			expected:     0.1664, // 0.0104 * 16.0
		},
		{
			name:         "Invalid format (no dot)",
			instanceType: "invalid",
			expected:     0.10, // Default
		},
		{
			name:         "Metal instance",
			instanceType: "c5.metal",
			expected:     1.36, // 0.085 * 16.0 (metal treated as xlarge)
		},
		{
			name:         "t4g.nano",
			instanceType: "t4g.nano",
			expected:     0.0021, // 0.0084 * 0.25
		},
		{
			name:         "m6i.24xlarge",
			instanceType: "m6i.24xlarge",
			expected:     36.864, // 0.096 * 384.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimatePriceByFamily(tt.instanceType)
			if !floatEqual(result, tt.expected, 0.00001) {
				t.Errorf("expected %.4f, got %.4f", tt.expected, result)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		expected string
	}{
		{
			name:     "Zero cost",
			cost:     0.0,
			expected: "$0.00",
		},
		{
			name:     "Small cost",
			cost:     0.05,
			expected: "$0.05",
		},
		{
			name:     "Typical cost",
			cost:     1.25,
			expected: "$1.25",
		},
		{
			name:     "Large cost",
			cost:     123.45,
			expected: "$123.45",
		},
		{
			name:     "Very precise cost (rounded)",
			cost:     1.23456789,
			expected: "$1.23",
		},
		{
			name:     "Negative cost (edge case)",
			cost:     -5.50,
			expected: "$-5.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCost(tt.cost)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatCostDetailed(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		expected string
	}{
		{
			name:     "Zero cost",
			cost:     0.0,
			expected: "$0.0000",
		},
		{
			name:     "Very small cost (4 decimals)",
			cost:     0.0005,
			expected: "$0.0005",
		},
		{
			name:     "Small cost under $0.01 (4 decimals)",
			cost:     0.0099,
			expected: "$0.0099",
		},
		{
			name:     "Cost at $0.01 boundary (2 decimals)",
			cost:     0.01,
			expected: "$0.01",
		},
		{
			name:     "Typical cost over $0.01 (2 decimals)",
			cost:     1.25,
			expected: "$1.25",
		},
		{
			name:     "Large cost (2 decimals)",
			cost:     123.45,
			expected: "$123.45",
		},
		{
			name:     "Very precise small cost",
			cost:     0.00123456,
			expected: "$0.0012",
		},
		{
			name:     "Very precise large cost",
			cost:     1.23456789,
			expected: "$1.23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCostDetailed(tt.cost)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEstimateSweepCost(t *testing.T) {
	tests := []struct {
		name        string
		params      *ParamFileFormat
		wantErr     bool
		checkResult func(*testing.T, *CostEstimate)
	}{
		{
			name: "Single instance, 1 hour",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"instance_type": "t3.micro",
					"region":        "us-east-1",
					"ttl":           "1h",
				},
				Params: []map[string]interface{}{
					{"name": "job1"},
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, e *CostEstimate) {
				expectedCompute := 0.0104 // t3.micro for 1 hour
				if e.ComputeCost != expectedCompute {
					t.Errorf("expected compute cost %.4f, got %.4f", expectedCompute, e.ComputeCost)
				}
				if e.TotalCost <= e.ComputeCost {
					t.Error("total cost should include Lambda and storage")
				}
			},
		},
		{
			name: "Multiple instances, same config",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"instance_type": "c5.xlarge",
					"region":        "us-east-1",
					"ttl":           "2h",
				},
				Params: []map[string]interface{}{
					{"name": "job1"},
					{"name": "job2"},
					{"name": "job3"},
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, e *CostEstimate) {
				expectedCompute := 0.17 * 2.0 * 3.0 // 3 instances, 2 hours each
				if e.ComputeCost != expectedCompute {
					t.Errorf("expected compute cost %.4f, got %.4f", expectedCompute, e.ComputeCost)
				}
			},
		},
		{
			name: "Mixed regions",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"instance_type": "t3.micro",
					"ttl":           "1h",
				},
				Params: []map[string]interface{}{
					{"name": "job1", "region": "us-east-1"},
					{"name": "job2", "region": "eu-west-1"},
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, e *CostEstimate) {
				expectedCompute := 0.0104 + 0.0114 // Different prices per region
				if e.ComputeCost != expectedCompute {
					t.Errorf("expected compute cost %.4f, got %.4f", expectedCompute, e.ComputeCost)
				}
			},
		},
		{
			name: "Mixed instance types",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"region": "us-east-1",
					"ttl":    "1h",
				},
				Params: []map[string]interface{}{
					{"name": "job1", "instance_type": "t3.micro"},
					{"name": "job2", "instance_type": "c5.xlarge"},
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, e *CostEstimate) {
				expectedCompute := 0.0104 + 0.17
				if e.ComputeCost != expectedCompute {
					t.Errorf("expected compute cost %.4f, got %.4f", expectedCompute, e.ComputeCost)
				}
			},
		},
		{
			name: "No instance type specified",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"region": "us-east-1",
				},
				Params: []map[string]interface{}{
					{"name": "job1"},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid TTL format",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"instance_type": "t3.micro",
					"region":        "us-east-1",
					"ttl":           "invalid",
				},
				Params: []map[string]interface{}{
					{"name": "job1"},
				},
			},
			wantErr: true,
		},
		{
			name: "Default 1 hour TTL when not specified",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"instance_type": "t3.micro",
					"region":        "us-east-1",
				},
				Params: []map[string]interface{}{
					{"name": "job1"},
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, e *CostEstimate) {
				expectedCompute := 0.0104 // 1 hour default
				if e.ComputeCost != expectedCompute {
					t.Errorf("expected compute cost %.4f, got %.4f", expectedCompute, e.ComputeCost)
				}
			},
		},
		{
			name: "Default us-east-1 region when not specified",
			params: &ParamFileFormat{
				Defaults: map[string]interface{}{
					"instance_type": "t3.micro",
					"ttl":           "1h",
				},
				Params: []map[string]interface{}{
					{"name": "job1"},
				},
			},
			wantErr: false,
			checkResult: func(t *testing.T, e *CostEstimate) {
				expectedCompute := 0.0104 // us-east-1 price
				if e.ComputeCost != expectedCompute {
					t.Errorf("expected compute cost %.4f, got %.4f", expectedCompute, e.ComputeCost)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EstimateSweepCost(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
				return
			}

			if err == nil && tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestCostEstimateDisplay(t *testing.T) {
	estimate := &CostEstimate{
		ComputeCost: 10.50,
		LambdaCost:  0.0005,
		StorageCost: 0.00001,
		TotalCost:   10.50051,
	}

	t.Run("Display format", func(t *testing.T) {
		result := estimate.Display()

		// Check that output contains expected components
		expectedStrings := []string{
			"$10.50",  // Compute cost
			"$0.0005", // Lambda cost (detailed)
			"$0.0000", // Storage cost (detailed, very small)
			"$10.50",  // Total cost
			"Compute",
			"Lambda",
			"Storage",
			"Total",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(result, expected) {
				t.Errorf("expected output to contain %q, got: %s", expected, result)
			}
		}
	})

	t.Run("DisplayCompact format", func(t *testing.T) {
		result := estimate.DisplayCompact()

		expectedStrings := []string{
			"Total estimated cost:",
			"$10.50",
			"EC2:",
			"Lambda:",
			"S3:",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(result, expected) {
				t.Errorf("expected output to contain %q, got: %s", expected, result)
			}
		}
	})
}

func TestGetStringValuePricing(t *testing.T) {
	// Same tests as sweep package, but using pricing package's version
	tests := []struct {
		name         string
		m            map[string]interface{}
		key          string
		defaultValue string
		expected     string
	}{
		{
			name: "Key exists with string value",
			m: map[string]interface{}{
				"instance_type": "t3.micro",
			},
			key:          "instance_type",
			defaultValue: "default",
			expected:     "t3.micro",
		},
		{
			name: "Key does not exist",
			m: map[string]interface{}{
				"instance_type": "t3.micro",
			},
			key:          "region",
			defaultValue: "us-east-1",
			expected:     "us-east-1",
		},
		{
			name: "Key exists with non-string value",
			m: map[string]interface{}{
				"count": 42,
			},
			key:          "count",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name: "Empty string value",
			m: map[string]interface{}{
				"region": "",
			},
			key:          "region",
			defaultValue: "us-east-1",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringValue(tt.m, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
