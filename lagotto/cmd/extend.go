package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	"github.com/scttfrdmn/spore-host/pkg/i18n"
	"github.com/spf13/cobra"
)

var extendTTL string

var extendCmd = &cobra.Command{
	Use:   "extend <watch-id>",
	Short: "Extend a watch's TTL",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtend,
}

func init() {
	rootCmd.AddCommand(extendCmd)
	extendCmd.Flags().StringVar(&extendTTL, "ttl", "24h", "New TTL from now (e.g., 24h, 7d)")
}

func runExtend(cmd *cobra.Command, args []string) error {
	watchID := args[0]
	ctx := context.Background()

	ttl, err := time.ParseDuration(extendTTL)
	if err != nil {
		ttl, err = parseDuration(extendTTL)
		if err != nil {
			return fmt.Errorf("%s", i18n.T("lagotto.error.invalid_ttl", map[string]interface{}{
				"TTL": extendTTL, "Error": err,
			}))
		}
	}

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
		return fmt.Errorf("%s", i18n.T("lagotto.error.watch_not_found", map[string]interface{}{
			"WatchID": watchID,
		}))
	}

	// Only active or expired watches can be extended
	if w.Status != watcher.StatusActive && w.Status != watcher.StatusExpired {
		return fmt.Errorf("%s", i18n.T("lagotto.cancel.not_active", map[string]interface{}{
			"WatchID": watchID, "Status": w.Status,
		}))
	}

	newExpiry := time.Now().UTC().Add(ttl)
	reactivate := w.Status == watcher.StatusExpired

	if err := store.ExtendWatch(ctx, watchID, newExpiry, reactivate); err != nil {
		return fmt.Errorf("extend watch: %w", err)
	}

	if reactivate {
		fmt.Fprintln(cmd.OutOrStdout(), i18n.T("lagotto.extend.reactivated", map[string]interface{}{
			"WatchID": watchID,
		}))
		// Re-enable polling schedule since we reactivated a watch
		if err := enablePollingSchedule(ctx, cfg); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Note: could not enable polling schedule: %v\n", err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), i18n.T("lagotto.extend.extended", map[string]interface{}{
		"WatchID":   watchID,
		"ExpiresAt": newExpiry.Format(time.RFC3339),
	}))
	return nil
}
