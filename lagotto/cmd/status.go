package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <watch-id>",
	Short: "Show details of a watch",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	watchID := args[0]
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	store := watcher.NewStore(cfg, watchesTable, historyTable)

	w, err := store.GetWatch(ctx, watchID)
	if err != nil {
		return fmt.Errorf("get watch: %w", err)
	}
	if w == nil {
		return fmt.Errorf("watch %s not found", watchID)
	}

	if getOutputFormat() == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(w)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Watch:    %s\n", w.WatchID)
	fmt.Fprintf(out, "Status:   %s\n", w.Status)
	fmt.Fprintf(out, "Pattern:  %s\n", w.InstanceTypePattern)
	fmt.Fprintf(out, "Regions:  %s\n", displayRegions(w.Regions))
	fmt.Fprintf(out, "Spot:     %v\n", w.Spot)
	if w.MaxPrice > 0 {
		fmt.Fprintf(out, "Max price: $%.4f/hr\n", w.MaxPrice)
	}
	fmt.Fprintf(out, "Action:   %s\n", w.Action)
	fmt.Fprintf(out, "Created:  %s\n", w.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(out, "Expires:  %s\n", w.ExpiresAt.Format(time.RFC3339))
	if !w.LastPolledAt.IsZero() {
		fmt.Fprintf(out, "Last polled: %s\n", w.LastPolledAt.Format(time.RFC3339))
	}
	fmt.Fprintf(out, "Matches:  %d\n", w.MatchCount)

	if w.LastMatch != nil {
		m := w.LastMatch
		fmt.Fprintf(out, "\nLast match:\n")
		fmt.Fprintf(out, "  Instance: %s\n", m.InstanceType)
		fmt.Fprintf(out, "  Region:   %s\n", m.Region)
		fmt.Fprintf(out, "  AZ:       %s\n", m.AvailabilityZone)
		fmt.Fprintf(out, "  Price:    $%.4f/hr\n", m.Price)
		fmt.Fprintf(out, "  Spot:     %v\n", m.IsSpot)
		fmt.Fprintf(out, "  Action:   %s\n", m.ActionTaken)
		if m.InstanceID != "" {
			fmt.Fprintf(out, "  Instance ID: %s\n", m.InstanceID)
		}
	}

	return nil
}
