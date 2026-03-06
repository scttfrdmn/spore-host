package main

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

// Example: Basic instance type search using Truffle as a library
func main() {
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create AWS client: %v", err)
	}

	// Get all AWS regions
	regions, err := client.GetAllRegions(ctx)
	if err != nil {
		log.Fatalf("Failed to get regions: %v", err)
	}

	fmt.Printf("Searching across %d regions...\n", len(regions))

	// Search for m7i.large instances
	pattern := regexp.MustCompile(`^m7i\.large$`)
	
	results, err := client.SearchInstanceTypes(ctx, regions, pattern, aws.FilterOptions{
		IncludeAZs: true,
		Verbose:    true,
	})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	// Print results
	fmt.Printf("\nFound %d results:\n", len(results))
	for _, r := range results {
		fmt.Printf("  %s in %s: %d vCPUs, %.1f GiB RAM\n",
			r.InstanceType,
			r.Region,
			r.VCPUs,
			float64(r.MemoryMiB)/1024.0)
		
		if len(r.AvailableAZs) > 0 {
			fmt.Printf("    AZs: %v\n", r.AvailableAZs)
		}
	}
}
