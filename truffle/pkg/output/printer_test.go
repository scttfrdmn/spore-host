package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

// captureStdout redirects os.Stdout to a buffer for the duration of fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func sampleInstances() []aws.InstanceTypeResult {
	return []aws.InstanceTypeResult{
		{InstanceType: "m7g.large", Region: "us-east-1", VCPUs: 2, MemoryMiB: 8192, Architecture: "arm64"},
		{InstanceType: "c6a.xlarge", Region: "us-east-1", VCPUs: 4, MemoryMiB: 8192, Architecture: "x86_64"},
		{InstanceType: "m7g.large", Region: "eu-west-1", VCPUs: 2, MemoryMiB: 8192, Architecture: "arm64"},
	}
}

func sampleSpotPrices() []aws.SpotPriceResult {
	return []aws.SpotPriceResult{
		{InstanceType: "c6a.xlarge", Region: "us-east-1", AvailabilityZone: "us-east-1a", SpotPrice: 0.0512},
		{InstanceType: "m7g.large", Region: "eu-west-1", AvailabilityZone: "eu-west-1b", SpotPrice: 0.0314},
	}
}

func sampleCapacity() []aws.CapacityReservationResult {
	return []aws.CapacityReservationResult{
		{ReservationID: "cr-0abc123def456", Region: "us-east-1", InstanceType: "m7g.large", AvailableCapacity: 10, TotalCapacity: 10},
	}
}

// --- JSON ---

func TestPrintJSON_ValidOutput(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintJSON(sampleInstances()); err != nil {
			t.Errorf("PrintJSON returned error: %v", err)
		}
	})
	var results []aws.InstanceTypeResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if results[0].InstanceType != "m7g.large" {
		t.Errorf("unexpected first instance type: %s", results[0].InstanceType)
	}
}

func TestPrintSpotJSON_ValidOutput(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintSpotJSON(sampleSpotPrices()); err != nil {
			t.Errorf("PrintSpotJSON returned error: %v", err)
		}
	})
	var results []aws.SpotPriceResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestPrintCapacityJSON_ValidOutput(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintCapacityJSON(sampleCapacity()); err != nil {
			t.Errorf("PrintCapacityJSON returned error: %v", err)
		}
	})
	if !strings.Contains(out, "cr-0abc123") {
		t.Errorf("output missing reservation ID: %s", out)
	}
}

// --- YAML ---

func TestPrintYAML_ContainsExpectedFields(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintYAML(sampleInstances()); err != nil {
			t.Errorf("PrintYAML returned error: %v", err)
		}
	})
	for _, want := range []string{"m7g.large", "c6a.xlarge", "us-east-1", "arm64"} {
		if !strings.Contains(out, want) {
			t.Errorf("YAML output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintSpotYAML_ContainsExpectedFields(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintSpotYAML(sampleSpotPrices()); err != nil {
			t.Errorf("PrintSpotYAML returned error: %v", err)
		}
	})
	if !strings.Contains(out, "c6a.xlarge") {
		t.Errorf("YAML output missing instance type: %s", out)
	}
}

func TestPrintCapacityYAML_ContainsExpectedFields(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintCapacityYAML(sampleCapacity()); err != nil {
			t.Errorf("PrintCapacityYAML returned error: %v", err)
		}
	})
	if !strings.Contains(out, "m7g.large") {
		t.Errorf("YAML output missing instance type: %s", out)
	}
}

// --- CSV ---

func TestPrintCSV_HasHeaderAndRows(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintCSV(sampleInstances()); err != nil {
			t.Errorf("PrintCSV returned error: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + rows, got %d lines", len(lines))
	}
	if !strings.Contains(lines[0], "instance_type") && !strings.Contains(lines[0], "INSTANCE") {
		t.Errorf("first line doesn't look like a header: %s", lines[0])
	}
}

func TestPrintSpotCSV_HasRows(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintSpotCSV(sampleSpotPrices()); err != nil {
			t.Errorf("PrintSpotCSV returned error: %v", err)
		}
	})
	if !strings.Contains(out, "c6a.xlarge") {
		t.Errorf("CSV output missing instance type: %s", out)
	}
}

func TestPrintCapacityCSV_HasRows(t *testing.T) {
	p := NewPrinter(false)
	out := captureStdout(t, func() {
		if err := p.PrintCapacityCSV(sampleCapacity()); err != nil {
			t.Errorf("PrintCapacityCSV returned error: %v", err)
		}
	})
	if !strings.Contains(out, "m7g.large") {
		t.Errorf("CSV output missing instance type: %s", out)
	}
}

// --- helpers ---

func TestGroupByInstanceType(t *testing.T) {
	grouped := groupByInstanceType(sampleInstances())
	if len(grouped["m7g.large"]) != 2 {
		t.Errorf("expected 2 m7g.large entries, got %d", len(grouped["m7g.large"]))
	}
	if len(grouped["c6a.xlarge"]) != 1 {
		t.Errorf("expected 1 c6a.xlarge entry, got %d", len(grouped["c6a.xlarge"]))
	}
	if grouped := groupByInstanceType(nil); len(grouped) != 0 {
		t.Errorf("expected empty map for nil input")
	}
}

func TestCountUniqueRegions(t *testing.T) {
	if n := countUniqueRegions(sampleInstances()); n != 2 {
		t.Errorf("expected 2 unique regions, got %d", n)
	}
	if n := countUniqueRegions(nil); n != 0 {
		t.Errorf("expected 0 for nil input, got %d", n)
	}
}

func TestShortenReservationID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"cr-0abc123def456789", "cr-0abc123..."},
		{"short", "short"},
		{"exactly10c", "exactly10c"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shortenReservationID(tt.input)
		if got != tt.want {
			t.Errorf("shortenReservationID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
