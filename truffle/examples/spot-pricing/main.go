package main

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"github.com/scttfrdmn/mycelium/truffle/pkg/aws"
)

// Example: Get Spot pricing for instance types
func main() {
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create AWS client: %v", err)
	}

	// Search for Graviton instances
	regions := []string{"us-east-1", "us-west-2"}
	pattern := regexp.MustCompile(`^m8g\..*`)
	
	results, err := client.SearchInstanceTypes(ctx, regions, pattern, aws.FilterOptions{
		IncludeAZs: true,
		MinVCPUs:   4,
		MinMemory:  16,
	})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	// Get Spot pricing
	fmt.Println("Fetching Spot prices...")
	spotResults, err := client.GetSpotPricing(ctx, results, aws.SpotOptions{
		MaxPrice:      0.10, // Max $0.10/hour
		ShowSavings:   true,
		LookbackHours: 1,
		Verbose:       true,
	})
	if err != nil {
		log.Fatalf("Failed to get Spot pricing: %v", err)
	}

	// Find cheapest options
	fmt.Printf("\nFound %d Spot pricing options:\n", len(spotResults))
	for i, spot := range spotResults {
		if i >= 5 { // Show top 5
			break
		}
		fmt.Printf("  %s in %s: $%.4f/hr",
			spot.InstanceType,
			spot.AvailabilityZone,
			spot.SpotPrice)
		
		if spot.SavingsPercent > 0 {
			fmt.Printf(" (%.1f%% savings)\n", spot.SavingsPercent)
		} else {
			fmt.Println()
		}
	}
}
