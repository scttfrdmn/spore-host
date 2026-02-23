package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/mycelium/truffle/pkg/aws"
)

var (
	showFamilies bool
	showSizes    bool
	regionFilter string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List instance types and families",
	Long: `List available EC2 instance types, families, or sizes.

Examples:
  truffle list --family
  truffle list --sizes
  truffle list --region us-east-1`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVar(&showFamilies, "family", false, "List instance families (e.g., m5, c5, r5)")
	listCmd.Flags().BoolVar(&showSizes, "sizes", false, "List available sizes (e.g., large, xlarge, 2xlarge)")
	listCmd.Flags().StringVar(&regionFilter, "region", "us-east-1", "Region to query for listing (default: us-east-1)")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if verbose {
		fmt.Fprintf(os.Stderr, "🔍 Querying region: %s\n", regionFilter)
	}

	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Get instance types from one region
	types, err := awsClient.GetInstanceTypes(ctx, regionFilter)
	if err != nil {
		return fmt.Errorf("failed to get instance types: %w", err)
	}

	if showFamilies {
		families := extractFamilies(types)
		printList("Instance Families", families)
	} else if showSizes {
		sizes := extractSizes(types)
		printList("Instance Sizes", sizes)
	} else {
		printList("Instance Types", types)
	}

	return nil
}

func extractFamilies(types []string) []string {
	familyMap := make(map[string]bool)
	for _, t := range types {
		// Extract family (e.g., m5 from m5.large)
		parts := strings.Split(t, ".")
		if len(parts) > 0 {
			familyMap[parts[0]] = true
		}
	}

	families := make([]string, 0, len(familyMap))
	for f := range familyMap {
		families = append(families, f)
	}
	sort.Strings(families)
	return families
}

func extractSizes(types []string) []string {
	sizeMap := make(map[string]bool)
	for _, t := range types {
		// Extract size (e.g., large from m5.large)
		parts := strings.Split(t, ".")
		if len(parts) > 1 {
			sizeMap[parts[1]] = true
		}
	}

	sizes := make([]string, 0, len(sizeMap))
	for s := range sizeMap {
		sizes = append(sizes, s)
	}
	sort.Strings(sizes)
	return sizes
}

func printList(title string, items []string) {
	fmt.Printf("📋 %s (%d total):\n\n", title, len(items))
	
	// Print in columns
	const itemsPerRow = 5
	for i := 0; i < len(items); i += itemsPerRow {
		end := i + itemsPerRow
		if end > len(items) {
			end = len(items)
		}
		row := items[i:end]
		
		// Format each item with padding
		for j, item := range row {
			fmt.Printf("  %-20s", item)
			if j < len(row)-1 {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}
	fmt.Println()
}
