package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/scttfrdmn/mycelium/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/truffle/pkg/aws"
	"github.com/scttfrdmn/mycelium/truffle/pkg/output"
	"github.com/scttfrdmn/mycelium/truffle/pkg/progress"
)

var (
	skipAZs        bool
	architecture   string
	minVCPUs       int
	minMemory      float64
	instanceFamily string
	timeout        time.Duration
)

var searchCmd = &cobra.Command{
	Use:  "search [instance-type-pattern]",
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
	// Short and Long will be set after i18n initialization
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().BoolVar(&skipAZs, "skip-azs", false, "Skip availability zone lookup (faster but less detailed)")
	searchCmd.Flags().StringVar(&architecture, "architecture", "", "Filter by architecture (x86_64, arm64, i386)")
	searchCmd.Flags().IntVar(&minVCPUs, "min-vcpu", 0, "Minimum number of vCPUs")
	searchCmd.Flags().Float64Var(&minMemory, "min-memory", 0, "Minimum memory in GiB")
	searchCmd.Flags().StringVar(&instanceFamily, "family", "", "Filter by instance family (e.g., m5, c5)")
	searchCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for AWS API calls")

	// Register completion for instance type argument
	searchCmd.ValidArgsFunction = completeInstanceType

	// Register completion for flags
	searchCmd.RegisterFlagCompletionFunc("architecture", completeArchitecture)
	searchCmd.RegisterFlagCompletionFunc("family", completeInstanceFamily)
}

func runSearch(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	// Convert wildcard pattern to regex
	regexPattern := wildcardToRegex(pattern)
	matcher, err := regexp.Compile(regexPattern)
	if err != nil {
		return i18n.Te("truffle.search.error.invalid_pattern", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("magnifying_glass"), i18n.Tf("truffle.search.searching", map[string]interface{}{
			"Pattern": pattern,
		}))
		if len(regions) > 0 {
			fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("pushpin"), i18n.Tf("truffle.search.filtering_regions", map[string]interface{}{
				"Regions": strings.Join(regions, ", "),
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
			fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("globe"), i18n.T("truffle.search.fetching_regions"))
		}
		searchRegions, err = awsClient.GetEnabledRegions(ctx)
		if err != nil {
			return i18n.Te("truffle.search.error.get_regions_failed", err)
		}
	}

	// Show spinner for non-verbose mode
	var spinner *progress.Spinner
	if !verbose && outputFormat == "table" {
		msg := i18n.Tf("truffle.search.searching_regions", map[string]interface{}{
			"Count": len(searchRegions),
		})
		spinner = progress.NewSpinner(os.Stderr, msg)
		spinner.Start()
	} else if verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", i18n.Emoji("magnifying_glass_tilted"), i18n.Tf("truffle.search.searching_across", map[string]interface{}{
			"Count": len(searchRegions),
		}))
	}

	// Search for instance types
	results, err := awsClient.SearchInstanceTypes(ctx, searchRegions, matcher, aws.FilterOptions{
		IncludeAZs:     !skipAZs, // AZs included by default
		Architecture:   architecture,
		MinVCPUs:       minVCPUs,
		MinMemory:      minMemory,
		InstanceFamily: instanceFamily,
		Verbose:        verbose,
	})

	if spinner != nil {
		spinner.Stop()
	}

	if err != nil {
		return i18n.Te("truffle.search.error.search_failed", err)
	}

	// Sort results for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].InstanceType != results[j].InstanceType {
			return results[i].InstanceType < results[j].InstanceType
		}
		return results[i].Region < results[j].Region
	})

	if len(results) == 0 {
		fmt.Println(i18n.T("truffle.search.no_results"))
		return nil
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
		return printer.PrintTable(results, !skipAZs) // Show AZs by default
	default:
		return i18n.Te("truffle.search.error.unsupported_format", nil, map[string]interface{}{
			"Format": outputFormat,
		})
	}
}

func wildcardToRegex(pattern string) string {
	// Escape special regex characters except * and ?
	pattern = regexp.QuoteMeta(pattern)
	// Replace wildcards
	pattern = strings.ReplaceAll(pattern, `\*`, ".*")
	pattern = strings.ReplaceAll(pattern, `\?`, ".")
	// Anchor the pattern
	return "^" + pattern + "$"
}
