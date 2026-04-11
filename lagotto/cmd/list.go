package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	"github.com/spf13/cobra"
)

var listAll bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your watches",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all statuses (default: active only)")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("get caller identity: %w", err)
	}

	store := watcher.NewStore(cfg, watchesTable, historyTable)

	var statusFilter watcher.WatchStatus
	if !listAll {
		statusFilter = watcher.StatusActive
	}

	watches, err := store.ListWatchesByUser(ctx, *identity.Arn, statusFilter)
	if err != nil {
		return fmt.Errorf("list watches: %w", err)
	}

	if getOutputFormat() == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(watches)
	}

	if len(watches) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No watches found.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%-12s %-10s %-20s %-25s %-6s %-10s %s\n",
		"WATCH ID", "STATUS", "PATTERN", "REGIONS", "SPOT", "ACTION", "EXPIRES")
	for _, w := range watches {
		regions := displayRegions(w.Regions)
		if len(regions) > 25 {
			regions = regions[:22] + "..."
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-12s %-10s %-20s %-25s %-6v %-10s %s\n",
			w.WatchID,
			w.Status,
			truncate(w.InstanceTypePattern, 20),
			regions,
			w.Spot,
			w.Action,
			w.ExpiresAt.Format(time.RFC3339),
		)
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
