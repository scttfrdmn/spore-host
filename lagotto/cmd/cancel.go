package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel <watch-id>",
	Short: "Cancel an active watch",
	Args:  cobra.ExactArgs(1),
	RunE:  runCancel,
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}

func runCancel(cmd *cobra.Command, args []string) error {
	watchID := args[0]
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	store := watcher.NewStore(cfg, watchesTable, historyTable)

	// Verify watch exists
	w, err := store.GetWatch(ctx, watchID)
	if err != nil {
		return fmt.Errorf("get watch: %w", err)
	}
	if w == nil {
		return fmt.Errorf("watch %s not found", watchID)
	}
	if w.Status != watcher.StatusActive {
		return fmt.Errorf("watch %s is not active (status: %s)", watchID, w.Status)
	}

	if err := store.UpdateWatchStatus(ctx, watchID, watcher.StatusCancelled); err != nil {
		return fmt.Errorf("cancel watch: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Cancelled watch %s\n", watchID)
	return nil
}
