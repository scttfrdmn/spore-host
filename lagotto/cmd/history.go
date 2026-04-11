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

var historyWatchID string

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show match history",
	RunE:  runHistory,
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().StringVar(&historyWatchID, "watch-id", "", "Filter by watch ID")
}

func runHistory(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	store := watcher.NewStore(cfg, watchesTable, historyTable)

	var matches []watcher.MatchResult

	if historyWatchID != "" {
		matches, err = store.ListMatchHistory(ctx, historyWatchID)
	} else {
		// Get user's history
		stsClient := sts.NewFromConfig(cfg)
		identity, err2 := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err2 != nil {
			return fmt.Errorf("get caller identity: %w", err2)
		}
		matches, err = store.ListMatchHistoryByUser(ctx, *identity.Arn)
	}
	if err != nil {
		return fmt.Errorf("list history: %w", err)
	}

	if getOutputFormat() == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(matches)
	}

	if len(matches) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No match history found.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%-12s %-18s %-15s %-15s %-10s %-10s %s\n",
		"WATCH ID", "INSTANCE TYPE", "REGION", "AZ", "PRICE", "ACTION", "MATCHED AT")
	for _, m := range matches {
		fmt.Fprintf(cmd.OutOrStdout(), "%-12s %-18s %-15s %-15s $%-9.4f %-10s %s\n",
			m.WatchID,
			m.InstanceType,
			m.Region,
			m.AvailabilityZone,
			m.Price,
			m.ActionTaken,
			m.MatchedAt.Format(time.RFC3339),
		)
	}
	return nil
}
