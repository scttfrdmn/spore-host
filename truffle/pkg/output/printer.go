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

// PrintTable outputs results as a formatted table
func (p *Printer) PrintTable(results []aws.InstanceTypeResult, includeAZs bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	
	// Set table style
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	
	// Set headers
	headers := []string{
		i18n.T("truffle.output.header.instance_type"),
		i18n.T("truffle.output.header.region"),
		i18n.T("truffle.output.header.vcpus"),
		i18n.T("truffle.output.header.memory"),
		i18n.T("truffle.output.header.architecture"),
	}
	if includeAZs {
		headers = append(headers, i18n.T("truffle.output.header.availability_zones"))
	}
	table.SetHeader(headers)

	// Configure colors - must match number of headers
	if p.useColor {
		colors := make([]tablewriter.Colors, len(headers))
		for i := range colors {
			colors[i] = tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor}
		}
		table.SetHeaderColor(colors...)
	}

	// Group results by instance type
	grouped := groupByInstanceType(results)

	// Add rows
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

			if includeAZs {
				azs := strings.Join(result.AvailableAZs, ", ")
				if azs == "" {
					azs = "N/A"
				}
				row = append(row, azs)
			}

			// Only show instance type for first occurrence
			if i > 0 {
				row[0] = ""
			}

			table.Append(row)
		}
	}

	// Print summary
	summaryMsg := i18n.Tf("truffle.output.summary.found", map[string]interface{}{
		"Count":   len(grouped),
		"Regions": countUniqueRegions(results),
	})

	if p.useColor {
		cyan := color.New(color.FgCyan, color.Bold)
		cyan.Printf("\n%s %s\n\n", i18n.Emoji("mushroom"), summaryMsg)
	} else {
		fmt.Printf("\n%s\n\n", summaryMsg)
	}

	table.Render()
	return nil
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
	defer encoder.Close()
	return encoder.Encode(results)
}

// PrintCSV outputs results as CSV
func (p *Printer) PrintCSV(results []aws.InstanceTypeResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"instance_type", "region", "vcpus", "memory_gib", "architecture", "availability_zones"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
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

// PrintSpotTable outputs Spot pricing results as a formatted table
func (p *Printer) PrintSpotTable(results []aws.SpotPriceResult, showSavings bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	
	// Configure colors
	if p.useColor {
		headerColors := []tablewriter.Colors{
			{tablewriter.Bold, tablewriter.FgHiCyanColor},
			{tablewriter.Bold, tablewriter.FgHiCyanColor},
			{tablewriter.Bold, tablewriter.FgHiCyanColor},
			{tablewriter.Bold, tablewriter.FgHiCyanColor},
		}
		if showSavings {
			headerColors = append(headerColors, 
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
				tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor})
		}
		table.SetHeaderColor(headerColors...)
	}

	// Set headers
	headers := []string{
		i18n.T("truffle.output.header.instance_type"),
		i18n.T("truffle.output.header.region"),
		i18n.T("truffle.output.header.availability_zone"),
		i18n.T("truffle.output.header.spot_price"),
	}
	if showSavings {
		headers = append(headers,
			i18n.T("truffle.output.header.on_demand_price"),
			i18n.T("truffle.output.header.savings"))
	}
	table.SetHeader(headers)

	// Group by instance type
	grouped := make(map[string][]aws.SpotPriceResult)
	for _, result := range results {
		grouped[result.InstanceType] = append(grouped[result.InstanceType], result)
	}

	// Add rows
	for instanceType, prices := range grouped {
		for i, result := range prices {
			row := []string{
				instanceType,
				result.Region,
				result.AvailabilityZone,
				fmt.Sprintf("$%.4f", result.SpotPrice),
			}

			if showSavings {
				if result.OnDemandPrice > 0 {
					row = append(row, 
						fmt.Sprintf("$%.4f", result.OnDemandPrice),
						fmt.Sprintf("%.1f%%", result.SavingsPercent))
				} else {
					row = append(row, "N/A", "N/A")
				}
			}

			// Only show instance type for first occurrence
			if i > 0 {
				row[0] = ""
			}

			table.Append(row)
		}
	}

	table.Render()
	return nil
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
	defer encoder.Close()
	return encoder.Encode(results)
}

// PrintSpotCSV outputs Spot pricing results as CSV
func (p *Printer) PrintSpotCSV(results []aws.SpotPriceResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"instance_type", "region", "availability_zone", "spot_price", "on_demand_price", "savings_percent", "timestamp"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
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
	table := tablewriter.NewWriter(os.Stdout)
	
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	
	// Configure colors
	if p.useColor {
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		)
	}

	// Set headers
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
	table.SetHeader(headers)

	// Add rows
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

		table.Append(row)
	}

	table.Render()
	return nil
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
	defer encoder.Close()
	return encoder.Encode(results)
}

// PrintCapacityCSV outputs capacity reservation results as CSV
func (p *Printer) PrintCapacityCSV(results []aws.CapacityReservationResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"instance_type", "region", "availability_zone", "total_capacity", "available_capacity", "used_capacity", "state", "reservation_id", "end_date", "platform"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
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

func shortenReservationID(id string) string {
	// Shorten reservation ID for table display (cr-xxxxxxxxx -> cr-xxx...)
	if len(id) > 10 {
		return id[:10] + "..."
	}
	return id
}
