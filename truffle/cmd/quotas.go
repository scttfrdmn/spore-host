package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/scttfrdmn/spore-host/pkg/i18n"
	"github.com/spf13/cobra"
	"github.com/scttfrdmn/spore-host/truffle/pkg/quotas"
)

var (
	quotasRegions []string
	quotasFamily  string
	quotasRequest bool
)

var quotasCmd = &cobra.Command{
	Use:   "quotas",
	Short: "Show AWS Service Quotas for EC2 instances",
	Long: `Display current quotas, usage, and available capacity for EC2 instances.

Requires AWS credentials to be configured.

Examples:
  # Show quotas for default region
  truffle quotas

  # Show quotas for specific regions
  truffle quotas --regions us-east-1,us-west-2

  # Show only GPU quotas
  truffle quotas --family P

  # Generate quota increase request
  truffle quotas --family P --request`,
	RunE: runQuotas,
}

func init() {
	rootCmd.AddCommand(quotasCmd)

	quotasCmd.Flags().StringSliceVar(&quotasRegions, "regions", []string{"us-east-1"},
		"Regions to check (comma-separated)")
	quotasCmd.Flags().StringVar(&quotasFamily, "family", "",
		"Filter by instance family (Standard, G, P, Inf, Trn)")
	quotasCmd.Flags().BoolVar(&quotasRequest, "request", false,
		"Generate quota increase request commands")
}

func runQuotas(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create quota client
	quotaClient, err := quotas.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("%s AWS credentials required for quota checking\n\nTo configure credentials:\n  1. Export environment variables:\n     export AWS_ACCESS_KEY_ID=...\n     export AWS_SECRET_ACCESS_KEY=...\n     export AWS_DEFAULT_REGION=us-east-1\n\n  OR\n\n  2. Run: aws configure\n\nError: %v", i18n.Symbol("error"), err)
	}

	// Get quotas for each region
	quotaInfos := make(map[string]*quotas.QuotaInfo)

	for _, region := range quotasRegions {
		fmt.Fprintf(os.Stderr, "Fetching quotas for %s...\n", region)
		
		info, err := quotaClient.GetQuotas(ctx, region)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not get quotas for %s: %v\n", region, err)
			continue
		}
		
		quotaInfos[region] = info
	}

	if len(quotaInfos) == 0 {
		return fmt.Errorf("could not retrieve quotas for any region")
	}

	// Display quotas
	for _, region := range quotasRegions {
		info, ok := quotaInfos[region]
		if !ok {
			continue
		}

		displayRegionQuotas(region, info, quotasFamily)
	}

	// Generate increase requests if requested
	if quotasRequest {
		fmt.Println()
		fmt.Println("╔════════════════════════════════════════════════════════╗")
		fmt.Println("║  📝 Quota Increase Request Commands                   ║")
		fmt.Println("╚════════════════════════════════════════════════════════╝")
		fmt.Println()

		generateIncreaseRequests(quotaInfos, quotasFamily)
	}

	return nil
}

func displayRegionQuotas(region string, info *quotas.QuotaInfo, filterFamily string) {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Printf("║  📊 AWS Service Quotas - %-28s ║\n", region)
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Prepare table data
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader([]string{"Family", "Type", "Quota", "Usage", "Available", "Status"}),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	families := []quotas.QuotaFamily{
		quotas.FamilyStandard,
		quotas.FamilyG,
		quotas.FamilyP,
		quotas.FamilyInf,
		quotas.FamilyTrn,
		quotas.FamilyF,
		quotas.FamilyX,
	}

	var applyGreen, applyYellow, applyRed func(a ...interface{}) string
	if noColor {
		id := func(a ...interface{}) string {
			if len(a) > 0 {
				return fmt.Sprint(a[0])
			}
			return ""
		}
		applyGreen, applyYellow, applyRed = id, id, id
	} else {
		applyGreen = color.New(color.FgGreen).SprintFunc()
		applyYellow = color.New(color.FgYellow).SprintFunc()
		applyRed = color.New(color.FgRed).SprintFunc()
	}

	for _, family := range families {
		// Skip if filtering and doesn't match
		if filterFamily != "" && string(family) != filterFamily {
			continue
		}

		// On-Demand
		onDemandQuota := info.OnDemand[family]
		onDemandUsage := info.Usage[family]
		onDemandAvailable := onDemandQuota - onDemandUsage

		if onDemandQuota > 0 || onDemandUsage > 0 {
			status := getQuotaStatus(onDemandQuota, onDemandUsage)
			statusStr := ""
			switch status {
			case "healthy":
				statusStr = applyGreen(i18n.Symbol("success") + " OK")
			case "warning":
				statusStr = applyYellow(i18n.Symbol("warning") + " Low")
			case "critical":
				statusStr = applyRed(i18n.Symbol("error") + " Full")
			case "zero":
				statusStr = applyRed(i18n.Symbol("error") + " Zero")
			}

			_ = table.Append([]string{
				string(family),
				"On-Demand",
				fmt.Sprintf("%d vCPUs", onDemandQuota),
				fmt.Sprintf("%d vCPUs", onDemandUsage),
				fmt.Sprintf("%d vCPUs", onDemandAvailable),
				statusStr,
			})
		}

		// Spot
		spotQuota := info.Spot[family]
		if spotQuota > 0 {
			_ = getQuotaStatus(spotQuota, 0) // Can't track Spot usage easily
			statusStr := ""
			if spotQuota == 0 {
				statusStr = applyRed(i18n.Symbol("error") + " Zero")
			} else {
				statusStr = applyGreen(i18n.Symbol("success") + " OK")
			}

			_ = table.Append([]string{
				string(family),
				"Spot",
				fmt.Sprintf("%d vCPUs", spotQuota),
				"-",
				fmt.Sprintf("%d vCPUs", spotQuota),
				statusStr,
			})
		}
	}

	if err := table.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "table render error: %v\n", err)
	}

	// Show instance count
	fmt.Println()
	fmt.Printf("%s Running Instances: %d / %d\n",
		i18n.Emoji("computer"), info.RunningInstances, info.RunningInstancesMax)

	// Show family descriptions
	fmt.Println()
	fmt.Printf("%s Instance Family Reference:\n", i18n.Emoji("books"))
	fmt.Println("   Standard: A, C, D, H, I, M, R, T, Z (general purpose)")
	fmt.Println("   G: Graphics/GPU instances (g4dn, g5, g6)")
	fmt.Println("   P: GPU training instances (p3, p4, p5)")
	fmt.Println("   Inf: Inferentia instances (inf1, inf2)")
	fmt.Println("   Trn: Trainium instances (trn1)")
	fmt.Println("   F: FPGA instances (f1)")
	fmt.Println("   X: Memory optimized (x1, x2)")
	fmt.Println()
}

func getQuotaStatus(quota, usage int32) string {
	if quota == 0 {
		return "zero"
	}

	available := quota - usage
	percentAvailable := float64(available) / float64(quota) * 100

	if percentAvailable >= 50 {
		return "healthy"
	} else if percentAvailable >= 25 {
		return "warning"
	} else if percentAvailable > 0 {
		return "critical"
	}

	return "zero"
}

func generateIncreaseRequests(quotaInfos map[string]*quotas.QuotaInfo, filterFamily string) {
	// Sort regions for consistent output
	var regions []string
	for region := range quotaInfos {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	for _, region := range regions {
		info := quotaInfos[region]

		families := []quotas.QuotaFamily{
			quotas.FamilyStandard,
			quotas.FamilyG,
			quotas.FamilyP,
			quotas.FamilyInf,
			quotas.FamilyTrn,
		}

		for _, family := range families {
			// Skip if filtering and doesn't match
			if filterFamily != "" && string(family) != filterFamily {
				continue
			}

			quota := info.OnDemand[family]
			usage := info.Usage[family]

			// Only generate requests for quotas that are zero or nearly full
			available := quota - usage
			if quota == 0 || (available > 0 && float64(available)/float64(quota) > 0.25) {
				continue
			}

			// Suggest doubling the quota (or 32 minimum)
			desiredValue := quota * 2
			if desiredValue < 32 {
				desiredValue = 32
			}
			if quota == 0 {
				// Common starting values
				switch family {
				case quotas.FamilyP:
					desiredValue = 192 // Enough for one p5.48xlarge
				case quotas.FamilyG:
					desiredValue = 128
				default:
					desiredValue = 32
				}
			}

			fmt.Printf("# %s - %s Family\n", region, family)
			fmt.Println(quotas.QuotaIncreaseCommand(region, family, desiredValue, false))
			fmt.Println()
		}
	}

	// Helpful notes
	fmt.Println("💡 Notes:")
	fmt.Println("   • Quota increases are typically approved within 24-48 hours")
	fmt.Println("   • GPU quotas (P, G, Inf, Trn) often require business justification")
	fmt.Println("   • Include your use case in the request for faster approval")
	fmt.Println("   • You can track request status in AWS Console → Service Quotas")
	fmt.Println()
}
