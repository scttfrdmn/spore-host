package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/scttfrdmn/spore-host/pkg/i18n"
	"github.com/scttfrdmn/spore-host/truffle/pkg/aws"
	"gopkg.in/yaml.v3"
)

// Printer handles output formatting
type Printer struct {
	useColor bool
}

// NewPrinter creates a new output printer
func NewPrinter(useColor bool) *Printer {
	return &Printer{useColor: useColor}
}

// newTable creates a standard table with consistent options.
func newTable(headers []string, useColor bool) *tablewriter.Table {
	if useColor {
		hdr := color.New(color.FgHiCyan, color.Bold).SprintFunc()
		for i, h := range headers {
			headers[i] = hdr(h)
		}
	}
	return tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader(headers),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
	)
}

// appendRow appends a []string row to a v1.x table.
func appendRow(table *tablewriter.Table, row []string) error {
	return table.Append(row)
}

// PrintTable outputs results as a formatted table
func (p *Printer) PrintTable(results []aws.InstanceTypeResult, includeAZs bool, showPrice bool) error {
	// Check if any results have GPU info
	hasGPU := false
	for _, r := range results {
		if r.GPUs > 0 {
			hasGPU = true
			break
		}
	}

	headers := []string{
		i18n.T("truffle.output.header.instance_type"),
		i18n.T("truffle.output.header.region"),
		i18n.T("truffle.output.header.vcpus"),
		i18n.T("truffle.output.header.memory"),
		i18n.T("truffle.output.header.architecture"),
	}
	if hasGPU {
		headers = append(headers, "GPUs", "GPU Model", "VRAM (GiB)")
	}
	if showPrice {
		headers = append(headers, "$/hr")
	}
	if includeAZs {
		headers = append(headers, i18n.T("truffle.output.header.availability_zones"))
	}

	table := newTable(headers, p.useColor)

	// Deduplicate: group by instanceType+region to collapse duplicate rows
	type resultKey struct{ instanceType, region string }
	seen := make(map[resultKey]bool)
	var deduped []aws.InstanceTypeResult
	for _, r := range results {
		k := resultKey{r.InstanceType, r.Region}
		if !seen[k] {
			seen[k] = true
			deduped = append(deduped, r)
		}
	}

	grouped := groupByInstanceType(deduped)

	for instanceType, regions := range grouped {
		for i, result := range regions {
			memGiB := fmt.Sprintf("%.1f", float64(result.MemoryMiB)/1024.0)
			row := []string{
				instanceType,
				result.Region,
				strconv.Itoa(int(result.VCPUs)),
				memGiB,
				result.Architecture,
			}
			if hasGPU {
				if result.GPUs > 0 {
					gpuModel := result.GPUModel
					if result.GPUManufacturer != "" {
						gpuModel = result.GPUManufacturer + " " + gpuModel
					}
					vramGiB := fmt.Sprintf("%.0f", float64(result.GPUMemoryMiB)/1024.0)
					row = append(row, strconv.Itoa(int(result.GPUs)), gpuModel, vramGiB)
				} else {
					row = append(row, "-", "-", "-")
				}
			}
			if showPrice {
				if result.OnDemandPrice > 0 {
					row = append(row, fmt.Sprintf("$%.4f", result.OnDemandPrice))
				} else {
					row = append(row, "N/A")
				}
			}
			if includeAZs {
				azs := strings.Join(result.AvailableAZs, ", ")
				if azs == "" {
					azs = "N/A"
				}
				row = append(row, azs)
			}
			if i > 0 {
				row[0] = ""
			}
			if err := appendRow(table, row); err != nil {
				return err
			}
		}
	}

	summaryMsg := i18n.Tf("truffle.output.summary.found", map[string]interface{}{
		"Count":   len(grouped),
		"Regions": countUniqueRegions(deduped),
	})
	if p.useColor {
		cyan := color.New(color.FgCyan, color.Bold)
		_, _ = cyan.Printf("\n%s %s\n\n", i18n.Emoji("mushroom"), summaryMsg)
	} else {
		fmt.Printf("\n%s\n\n", summaryMsg)
	}

	return table.Render()
}

// PrintJSON outputs results as JSON
func (p *Printer) PrintJSON(results []aws.InstanceTypeResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// PrintYAML outputs results as YAML
func (p *Printer) PrintYAML(results []aws.InstanceTypeResult) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(results)
}

// PrintCSV outputs results as CSV
func (p *Printer) PrintCSV(results []aws.InstanceTypeResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	header := []string{"instance_type", "region", "vcpus", "memory_gib", "architecture", "availability_zones"}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, result := range results {
		memGiB := fmt.Sprintf("%.1f", float64(result.MemoryMiB)/1024.0)
		azs := strings.Join(result.AvailableAZs, ";")
		row := []string{
			result.InstanceType,
			result.Region,
			strconv.Itoa(int(result.VCPUs)),
			memGiB,
			result.Architecture,
			azs,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// PrintSpotTable outputs Spot pricing results as a formatted table
func (p *Printer) PrintSpotTable(results []aws.SpotPriceResult) error {
	headers := []string{
		i18n.T("truffle.output.header.instance_type"),
		i18n.T("truffle.output.header.region"),
		i18n.T("truffle.output.header.availability_zone"),
		i18n.T("truffle.output.header.spot_price"),
		"On-Demand",
		"Savings",
	}

	table := newTable(headers, p.useColor)

	grouped := make(map[string][]aws.SpotPriceResult)
	for _, result := range results {
		grouped[result.InstanceType] = append(grouped[result.InstanceType], result)
	}

	for instanceType, prices := range grouped {
		for i, result := range prices {
			odStr, savStr := "N/A", "N/A"
			if result.OnDemandPrice > 0 {
				odStr = fmt.Sprintf("$%.4f", result.OnDemandPrice)
				savStr = fmt.Sprintf("%.0f%%", result.SavingsPercent)
			}
			row := []string{
				instanceType,
				result.Region,
				result.AvailabilityZone,
				fmt.Sprintf("$%.4f", result.SpotPrice),
				odStr,
				savStr,
			}
			if i > 0 {
				row[0] = ""
			}
			if err := appendRow(table, row); err != nil {
				return err
			}
		}
	}

	return table.Render()
}

// PrintSpotJSON outputs Spot pricing results as JSON
func (p *Printer) PrintSpotJSON(results []aws.SpotPriceResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// PrintSpotYAML outputs Spot pricing results as YAML
func (p *Printer) PrintSpotYAML(results []aws.SpotPriceResult) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(results)
}

// PrintSpotCSV outputs Spot pricing results as CSV
func (p *Printer) PrintSpotCSV(results []aws.SpotPriceResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	header := []string{"instance_type", "region", "availability_zone", "spot_price", "on_demand_price", "savings_percent", "timestamp"}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, result := range results {
		row := []string{
			result.InstanceType,
			result.Region,
			result.AvailabilityZone,
			fmt.Sprintf("%.4f", result.SpotPrice),
			fmt.Sprintf("%.4f", result.OnDemandPrice),
			fmt.Sprintf("%.2f", result.SavingsPercent),
			result.Timestamp,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// PrintCapacityTable outputs capacity reservation results as a formatted table
func (p *Printer) PrintCapacityTable(results []aws.CapacityReservationResult) error {
	headers := []string{
		i18n.T("truffle.output.header.instance_type"),
		i18n.T("truffle.output.header.region"),
		i18n.T("truffle.output.header.az"),
		i18n.T("truffle.output.header.total"),
		i18n.T("truffle.output.header.available"),
		i18n.T("truffle.output.header.used"),
		i18n.T("truffle.output.header.state"),
		i18n.T("truffle.output.header.reservation_id"),
	}

	table := newTable(headers, p.useColor)

	for _, r := range results {
		utilizationPct := 0.0
		if r.TotalCapacity > 0 {
			utilizationPct = float64(r.UsedCapacity) / float64(r.TotalCapacity) * 100
		}
		row := []string{
			r.InstanceType,
			r.Region,
			r.AvailabilityZone,
			fmt.Sprintf("%d", r.TotalCapacity),
			fmt.Sprintf("%d", r.AvailableCapacity),
			fmt.Sprintf("%d (%.0f%%)", r.UsedCapacity, utilizationPct),
			r.State,
			shortenReservationID(r.ReservationID),
		}
		if err := appendRow(table, row); err != nil {
			return err
		}
	}

	return table.Render()
}

// PrintCapacityJSON outputs capacity reservation results as JSON
func (p *Printer) PrintCapacityJSON(results []aws.CapacityReservationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// PrintCapacityYAML outputs capacity reservation results as YAML
func (p *Printer) PrintCapacityYAML(results []aws.CapacityReservationResult) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(results)
}

// PrintCapacityCSV outputs capacity reservation results as CSV
func (p *Printer) PrintCapacityCSV(results []aws.CapacityReservationResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	header := []string{"instance_type", "region", "availability_zone", "total_capacity", "available_capacity", "used_capacity", "state", "reservation_id", "end_date", "platform"}
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, r := range results {
		row := []string{
			r.InstanceType,
			r.Region,
			r.AvailabilityZone,
			fmt.Sprintf("%d", r.TotalCapacity),
			fmt.Sprintf("%d", r.AvailableCapacity),
			fmt.Sprintf("%d", r.UsedCapacity),
			r.State,
			r.ReservationID,
			r.EndDate,
			r.Platform,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func groupByInstanceType(results []aws.InstanceTypeResult) map[string][]aws.InstanceTypeResult {
	grouped := make(map[string][]aws.InstanceTypeResult)
	for _, result := range results {
		grouped[result.InstanceType] = append(grouped[result.InstanceType], result)
	}
	return grouped
}

func countUniqueRegions(results []aws.InstanceTypeResult) int {
	regions := make(map[string]bool)
	for _, result := range results {
		regions[result.Region] = true
	}
	return len(regions)
}

func shortenReservationID(id string) string {
	if len(id) > 10 {
		return id[:10] + "..."
	}
	return id
}
