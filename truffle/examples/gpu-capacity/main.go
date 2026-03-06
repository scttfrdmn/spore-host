package main

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/scttfrdmn/spore-host/truffle/pkg/aws"
)

// Example: Check capacity reservations for GPU instances
// This is critical for getting in-demand ML instances like p5, g6, etc.
func main() {
	ctx := context.Background()

	// Create AWS client
	client, err := aws.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create AWS client: %v", err)
	}

	// Get all regions
	regions, err := client.GetAllRegions(ctx)
	if err != nil {
		log.Fatalf("Failed to get regions: %v", err)
	}

	fmt.Println("Checking capacity reservations for GPU instances...")
	fmt.Printf("Searching across %d regions...\n\n", len(regions))

	// Check for GPU instance capacity reservations
	gpuInstances := []string{
		"p5.48xlarge",   // NVIDIA H100 - latest
		"g6.xlarge",     // NVIDIA L4 - cost-effective
		"inf2.xlarge",   // AWS Inferentia2
		"trn1.2xlarge",  // AWS Trainium
	}

	reservations, err := client.GetCapacityReservations(ctx, regions, aws.CapacityReservationOptions{
		InstanceTypes: gpuInstances,
		OnlyAvailable: true,  // Only show with available capacity
		OnlyActive:    true,  // Only active reservations
		MinCapacity:   1,     // At least 1 instance available
		Verbose:       true,
	})
	if err != nil {
		log.Fatalf("Failed to get capacity reservations: %v", err)
	}

	if len(reservations) == 0 {
		fmt.Println("No available GPU capacity reservations found.")
		fmt.Println("\nTip: You may need to create your own ODCRs for guaranteed access")
		return
	}

	// Sort by available capacity (most available first)
	sort.Slice(reservations, func(i, j int) bool {
		return reservations[i].AvailableCapacity > reservations[j].AvailableCapacity
	})

	// Print results
	fmt.Printf("Found %d reservations with available capacity:\n\n", len(reservations))
	
	for _, r := range reservations {
		fmt.Printf("Instance: %s\n", r.InstanceType)
		fmt.Printf("  Region/AZ: %s / %s\n", r.Region, r.AvailabilityZone)
		fmt.Printf("  Available: %d instances\n", r.AvailableCapacity)
		fmt.Printf("  Total Reserved: %d instances\n", r.TotalCapacity)
		fmt.Printf("  Utilization: %d%% (%.d/%.d used)\n",
			int(float64(r.UsedCapacity)/float64(r.TotalCapacity)*100),
			r.UsedCapacity,
			r.TotalCapacity)
		fmt.Printf("  State: %s\n", r.State)
		fmt.Printf("  Reservation ID: %s\n\n", r.ReservationID)
	}

	// Summary by instance type
	summary := make(map[string]int32)
	for _, r := range reservations {
		summary[r.InstanceType] += r.AvailableCapacity
	}

	fmt.Println("Summary by Instance Type:")
	for instanceType, available := range summary {
		fmt.Printf("  %s: %d available instances\n", instanceType, available)
	}
}
