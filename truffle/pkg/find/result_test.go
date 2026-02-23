package find

import (
	"testing"

	"github.com/scttfrdmn/mycelium/truffle/pkg/aws"
)

func TestExplainMatch(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		result      aws.InstanceTypeResult
		wantReasons int
	}{
		{
			name:  "graviton vendor match",
			query: "graviton",
			result: aws.InstanceTypeResult{
				InstanceType: "m6g.2xlarge",
				VCPUs:        8,
				MemoryMiB:    32768,
				Architecture: "arm64",
			},
			wantReasons: 1, // vendor match
		},
		{
			name:  "ice lake processor match",
			query: "ice lake",
			result: aws.InstanceTypeResult{
				InstanceType: "m6i.4xlarge",
				VCPUs:        16,
				MemoryMiB:    65536,
				Architecture: "x86_64",
			},
			wantReasons: 1, // processor match
		},
		{
			name:  "amd with vcpu constraint",
			query: "amd 16 cores",
			result: aws.InstanceTypeResult{
				InstanceType: "m6a.4xlarge",
				VCPUs:        16,
				MemoryMiB:    65536,
				Architecture: "x86_64",
			},
			wantReasons: 2, // vendor + vcpu match
		},
		{
			name:  "graviton with memory constraint",
			query: "graviton 32gb",
			result: aws.InstanceTypeResult{
				InstanceType: "m6g.2xlarge",
				VCPUs:        8,
				MemoryMiB:    32768,
				Architecture: "arm64",
			},
			wantReasons: 2, // vendor + memory match
		},
		{
			name:  "a100 gpu",
			query: "a100",
			result: aws.InstanceTypeResult{
				InstanceType: "p4d.24xlarge",
				VCPUs:        96,
				MemoryMiB:    1179648,
				Architecture: "x86_64",
			},
			wantReasons: 1, // GPU match
		},
		{
			name:  "large size",
			query: "graviton large",
			result: aws.InstanceTypeResult{
				InstanceType: "m6g.2xlarge",
				VCPUs:        8,
				MemoryMiB:    32768,
				Architecture: "arm64",
			},
			wantReasons: 2, // vendor + size match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq, err := ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			reasons := ExplainMatch(tt.result, pq)

			if len(reasons) < tt.wantReasons {
				t.Errorf("ExplainMatch() returned %d reasons, want >= %d: %v",
					len(reasons), tt.wantReasons, reasons)
			}
		})
	}
}

func TestExtractFamily(t *testing.T) {
	tests := []struct {
		instanceType string
		want         string
	}{
		{"m6i.2xlarge", "m6i"},
		{"c6g.4xlarge", "c6g"},
		{"p4d.24xlarge", "p4d"},
		{"t4g.nano", "t4g"},
		{"m7i-flex.large", "m7i-flex"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			got := extractFamily(tt.instanceType)
			if got != tt.want {
				t.Errorf("extractFamily(%q) = %v, want %v", tt.instanceType, got, tt.want)
			}
		})
	}
}
