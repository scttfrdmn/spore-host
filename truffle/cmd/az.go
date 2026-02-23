package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/truffle/pkg/aws"
	"github.com/scttfrdmn/mycelium/truffle/pkg/output"
)

var (
	azFilter       []string
	minAZCount     int
	showRegionOnly bool
)

var azCmd = &cobra.Command{
	Use:   "az [instance-type-pattern]",
	Short: "Search by availability zone (AZ-first view)",
	Long: `Search for instance types with an availability zone-first perspective.
This command prioritizes showing which specific AZs support each instance type,
making it ideal for multi-AZ deployments and capacity planning.

Examples:
  # Find which AZs have m7i.large
  truffle az m7i.large

  # Search in specific AZs only
  truffle az m7i.large --az us-east-1a,us-east-1b

  # Find instances available in at least 3 AZs per region
  truffle az "m8g.*" --min-az-count 3

  # Show AZ availability summary
  truffle az "c7i.xlarge" --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runAZSearch,
}

func init() {
	rootCmd.AddCommand(azCmd)

	azCmd.Flags().StringSliceVar(&azFilter, "az", []string{}, "Filter by specific availability zones (e.g., us-east-1a,us-west-2b)")
	azCmd.Flags().IntVar(&minAZCount, "min-az-count", 0, "Minimum number of AZs required per region")
	azCmd.Flags().BoolVar(&showRegionOnly, "regions-only", false, "Show only regions that meet AZ count requirement")
	azCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for AWS API calls")

	// Register completion for instance type argument
	azCmd.ValidArgsFunction = completeInstanceType
}

func runAZSearch(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	// Convert wildcard pattern to regex
	regexPattern := wildcardToRegex(pattern)
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "🔍 Searching for instance types matching: %s (AZ-focused)\n", pattern)
		if len(azFilter) > 0 {
			fmt.Fprintf(os.Stderr, "📍 Filtering AZs: %s\n", strings.Join(azFilter, ", "))
		}
		if minAZCount > 0 {
			fmt.Fprintf(os.Stderr, "📊 Minimum AZs per region: %d\n", minAZCount)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Get regions to search (extract from AZ filters if provided)
	searchRegions := extractRegionsFromAZs(azFilter)
	if len(searchRegions) == 0 {
		searchRegions = regions
	}
	
	// If no regions specified, auto-detect enabled regions (respects SCPs)
	if len(searchRegions) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, "🌍 Fetching enabled AWS regions (respects SCPs)...")
		}
		searchRegions, err = awsClient.GetEnabledRegions(ctx)
		if err != nil {
			return fmt.Errorf("failed to get regions: %w", err)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "🔎 Searching across %d regions...\n", len(searchRegions))
	}

	// Search for instance types (always include AZs for this command)
	results, err := awsClient.SearchInstanceTypes(ctx, searchRegions, matcher, aws.FilterOptions{
		IncludeAZs:     true, // Always get AZs for az command
		Architecture:   architecture,
		MinVCPUs:       minVCPUs,
		MinMemory:      minMemory,
		InstanceFamily: instanceFamily,
		Verbose:        verbose,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Filter by specific AZs if requested
	if len(azFilter) > 0 {
		results = filterByAZs(results, azFilter)
	}

	// Filter by minimum AZ count
	if minAZCount > 0 {
		results = filterByMinAZCount(results, minAZCount)
	}

	// Sort by AZ count (most AZs first), then by instance type
	sort.Slice(results, func(i, j int) bool {
		countI := len(results[i].AvailableAZs)
		countJ := len(results[j].AvailableAZs)
		if countI != countJ {
			return countI > countJ
		}
		if results[i].InstanceType != results[j].InstanceType {
			return results[i].InstanceType < results[j].InstanceType
		}
		return results[i].Region < results[j].Region
	})

	if len(results) == 0 {
		fmt.Println("No matching instance types found with specified AZ criteria.")
		return nil
	}

	// Print summary
	if verbose || outputFormat == "table" {
		printAZSummary(results)
	}

	// Output results
	printer := output.NewPrinter(!noColor)
	switch outputFormat {
	case "json":
		return printer.PrintJSON(results)
	case "yaml":
		return printer.PrintYAML(results)
	case "csv":
		return printer.PrintCSV(results)
	case "table":
		return printer.PrintTable(results, true) // Always show AZs
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func extractRegionsFromAZs(azs []string) []string {
	regionMap := make(map[string]bool)
	for _, az := range azs {
		// Extract region from AZ (e.g., us-east-1a -> us-east-1)
		if len(az) > 1 {
			region := az[:len(az)-1] // Remove last character
			regionMap[region] = true
		}
	}

	regions := make([]string, 0, len(regionMap))
	for region := range regionMap {
		regions = append(regions, region)
	}
	return regions
}

func filterByAZs(results []aws.InstanceTypeResult, azFilter []string) []aws.InstanceTypeResult {
	azSet := make(map[string]bool)
	for _, az := range azFilter {
		azSet[az] = true
	}

	filtered := make([]aws.InstanceTypeResult, 0)
	for _, result := range results {
		// Check if any of the result's AZs match the filter
		hasMatch := false
		matchedAZs := make([]string, 0)
		for _, az := range result.AvailableAZs {
			if azSet[az] {
				hasMatch = true
				matchedAZs = append(matchedAZs, az)
			}
		}
		if hasMatch {
			result.AvailableAZs = matchedAZs // Only show matching AZs
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func filterByMinAZCount(results []aws.InstanceTypeResult, minCount int) []aws.InstanceTypeResult {
	filtered := make([]aws.InstanceTypeResult, 0)
	for _, result := range results {
		if len(result.AvailableAZs) >= minCount {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func printAZSummary(results []aws.InstanceTypeResult) {
	// Count unique instance types
	instanceTypes := make(map[string]bool)
	totalAZs := 0
	maxAZs := 0
	minAZs := 999

	for _, result := range results {
		instanceTypes[result.InstanceType] = true
		azCount := len(result.AvailableAZs)
		totalAZs += azCount
		if azCount > maxAZs {
			maxAZs = azCount
		}
		if azCount < minAZs && azCount > 0 {
			minAZs = azCount
		}
	}

	if minAZs == 999 {
		minAZs = 0
	}

	avgAZs := 0.0
	if len(results) > 0 {
		avgAZs = float64(totalAZs) / float64(len(results))
	}

	fmt.Printf("\n📊 AZ Availability Summary:\n")
	fmt.Printf("   Instance Types: %d\n", len(instanceTypes))
	fmt.Printf("   Region Results: %d\n", len(results))
	fmt.Printf("   AZ Range: %d-%d AZs per region\n", minAZs, maxAZs)
	fmt.Printf("   Average AZs: %.1f per region\n\n", avgAZs)
}
