package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/scttfrdmn/spore-host/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/spore-host/truffle/pkg/aws"
	"github.com/scttfrdmn/spore-host/truffle/pkg/output"
)

var (
	spotMaxPrice     float64
	spotShowSavings  bool
	spotSortByPrice  bool
	spotOnlyActive   bool
	spotLookbackHours int
)

var spotCmd = &cobra.Command{
	Use:  "spot [instance-type-pattern]",
	Args: cobra.ExactArgs(1),
	RunE: runSpot,
	// Short and Long will be set after i18n initialization
}

func init() {
	rootCmd.AddCommand(spotCmd)

	spotCmd.Flags().Float64Var(&spotMaxPrice, "max-price", 0, "Maximum Spot price per hour (USD)")
	spotCmd.Flags().BoolVar(&spotShowSavings, "show-savings", false, "Show savings vs On-Demand pricing")
	spotCmd.Flags().BoolVar(&spotSortByPrice, "sort-by-price", false, "Sort by price (cheapest first)")
	spotCmd.Flags().BoolVar(&spotOnlyActive, "active-only", false, "Only show AZs with active Spot capacity")
	spotCmd.Flags().IntVar(&spotLookbackHours, "lookback-hours", 1, "Hours to look back for price history (1-720)")
	spotCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for AWS API calls")

	// Register completion for instance type argument
	spotCmd.ValidArgsFunction = completeInstanceType
}

func runSpot(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	// Convert wildcard pattern to regex
	regexPattern := wildcardToRegex(pattern)
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		return i18n.Te("truffle.spot.error.invalid_pattern", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("money_bag"), i18n.Tf("truffle.spot.searching", map[string]interface{}{
			"Pattern": pattern,
		}))
		if spotMaxPrice > 0 {
			fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("dollar"), i18n.Tf("truffle.spot.max_price_filter", map[string]interface{}{
				"MaxPrice": spotMaxPrice,
			}))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return i18n.Te("error.aws_client_init", err)
	}

	// Get regions to search
	// If no regions specified, auto-detect enabled regions (respects SCPs)
	searchRegions := regions
	if len(searchRegions) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("globe"), i18n.T("truffle.spot.fetching_regions"))
		}
		searchRegions, err = awsClient.GetEnabledRegions(ctx)
		if err != nil {
			return i18n.Te("truffle.spot.error.get_regions_failed", err)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("magnifying_glass_tilted"), i18n.Tf("truffle.spot.searching_across", map[string]interface{}{
			"Count": len(searchRegions),
		}))
	}

	// First find instance types (need to match pattern)
	results, err := awsClient.SearchInstanceTypes(ctx, searchRegions, matcher, aws.FilterOptions{
		IncludeAZs:     true, // Always get AZs for Spot
		Architecture:   architecture,
		MinVCPUs:       minVCPUs,
		MinMemory:      minMemory,
		InstanceFamily: instanceFamily,
		Verbose:        verbose,
	})
	if err != nil {
		return i18n.Te("truffle.spot.error.search_failed", err)
	}

	if len(results) == 0 {
		fmt.Println(i18n.T("truffle.spot.no_matching_types"))
		return nil
	}

	// Get Spot pricing for found instances
	if verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("money_bag"), i18n.T("truffle.spot.fetching_pricing"))
	}

	spotResults, err := awsClient.GetSpotPricing(ctx, results, aws.SpotOptions{
		MaxPrice:      spotMaxPrice,
		ShowSavings:   spotShowSavings,
		LookbackHours: spotLookbackHours,
		OnlyActive:    spotOnlyActive,
		Verbose:       verbose,
	})
	if err != nil {
		return i18n.Te("truffle.spot.error.get_pricing_failed", err)
	}

	if len(spotResults) == 0 {
		fmt.Println(i18n.T("truffle.spot.no_pricing_data"))
		return nil
	}

	// Sort results
	if spotSortByPrice {
		sort.Slice(spotResults, func(i, j int) bool {
			return spotResults[i].SpotPrice < spotResults[j].SpotPrice
		})
	} else {
		// Default: sort by instance type, then region, then AZ
		sort.Slice(spotResults, func(i, j int) bool {
			if spotResults[i].InstanceType != spotResults[j].InstanceType {
				return spotResults[i].InstanceType < spotResults[j].InstanceType
			}
			if spotResults[i].Region != spotResults[j].Region {
				return spotResults[i].Region < spotResults[j].Region
			}
			return spotResults[i].AvailabilityZone < spotResults[j].AvailabilityZone
		})
	}

	// Print summary
	printSpotSummary(spotResults)

	// Output results
	printer := output.NewPrinter(!noColor)
	switch outputFormat {
	case "json":
		return printer.PrintSpotJSON(spotResults)
	case "yaml":
		return printer.PrintSpotYAML(spotResults)
	case "csv":
		return printer.PrintSpotCSV(spotResults)
	case "table":
		return printer.PrintSpotTable(spotResults, spotShowSavings)
	default:
		return i18n.Te("truffle.spot.error.unsupported_format", nil, map[string]interface{}{
			"Format": outputFormat,
		})
	}
}

func printSpotSummary(results []aws.SpotPriceResult) {
	if len(results) == 0 {
		return
	}

	// Calculate stats
	instanceTypes := make(map[string]bool)
	regions := make(map[string]bool)
	azs := make(map[string]bool)
	
	var totalPrice, minPrice, maxPrice float64
	minPrice = 999999.0
	maxPrice = 0.0
	totalSavings := 0.0
	savingsCount := 0

	for _, r := range results {
		instanceTypes[r.InstanceType] = true
		regions[r.Region] = true
		azs[r.AvailabilityZone] = true
		
		totalPrice += r.SpotPrice
		if r.SpotPrice < minPrice {
			minPrice = r.SpotPrice
		}
		if r.SpotPrice > maxPrice {
			maxPrice = r.SpotPrice
		}
		
		if r.SavingsPercent > 0 {
			totalSavings += r.SavingsPercent
			savingsCount++
		}
	}

	avgPrice := totalPrice / float64(len(results))
	avgSavings := 0.0
	if savingsCount > 0 {
		avgSavings = totalSavings / float64(savingsCount)
	}

	fmt.Printf("\n%s %s\n", i18n.Emoji("money_bag"), i18n.T("truffle.spot.summary.title"))
	fmt.Printf("   %s: %d\n", i18n.T("truffle.spot.summary.instance_types"), len(instanceTypes))
	fmt.Printf("   %s: %d\n", i18n.T("truffle.spot.summary.regions"), len(regions))
	fmt.Printf("   %s: %d\n", i18n.T("truffle.spot.summary.availability_zones"), len(azs))
	fmt.Printf("   %s: $%.4f - $%.4f %s\n", i18n.T("truffle.spot.summary.price_range"), minPrice, maxPrice, i18n.T("truffle.spot.summary.per_hour"))
	fmt.Printf("   %s: $%.4f %s\n", i18n.T("truffle.spot.summary.average_price"), avgPrice, i18n.T("truffle.spot.summary.per_hour"))
	if avgSavings > 0 {
		fmt.Printf("   %s: %.1f%% %s\n", i18n.T("truffle.spot.summary.average_savings"), avgSavings, i18n.T("truffle.spot.summary.vs_on_demand"))
	}
	fmt.Println()
}
