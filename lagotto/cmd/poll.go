package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/scttfrdmn/spore-host/lagotto/pkg/watcher"
	truffleaws "github.com/scttfrdmn/spore-host/truffle/pkg/aws"
	"github.com/spf13/cobra"
)

var pollCmd = &cobra.Command{
	Use:   "poll",
	Short: "Run one polling cycle (for testing/debugging)",
	Long:  `Manually trigger a single poll of all active watches. This is for local testing; in production, polling runs on a Lambda schedule.`,
	RunE:  runPoll,
}

func init() {
	rootCmd.AddCommand(pollCmd)
}

func runPoll(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}

	truffleClient, err := truffleaws.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("create truffle client: %w", err)
	}

	store := watcher.NewStore(cfg, watchesTable, historyTable)

	// Set up optional notifier
	var notifier *watcher.Notifier
	topicArn := os.Getenv("LAGOTTO_SNS_TOPIC_ARN")
	if topicArn != "" {
		notifier = watcher.NewNotifier(cfg, topicArn)
	}

	// Set up optional spawner
	var spawner *watcher.Spawner
	spawner, err = watcher.NewSpawner(ctx)
	if err != nil && verbose {
		fmt.Fprintf(os.Stderr, "Note: auto-spawn unavailable: %v\n", err)
	}

	holder := watcher.NewHolder(cfg)

	poller := watcher.NewPoller(truffleClient, store, verbose, watcher.PollerOpts{
		Notifier: notifier,
		Spawner:  spawner,
		Holder:   holder,
	})

	matches, err := poller.PollAll(ctx)
	if err != nil {
		return fmt.Errorf("poll: %w", err)
	}

	if getOutputFormat() == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(matches)
	}

	if len(matches) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No matches found in this poll cycle.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Found %d match(es):\n", len(matches))
	for _, m := range matches {
		spotLabel := "on-demand"
		if m.IsSpot {
			spotLabel = "spot"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s %s in %s (%s) $%.4f/hr [watch: %s] action: %s\n",
			m.InstanceType, spotLabel, m.Region, m.AvailabilityZone, m.Price, m.WatchID, m.ActionTaken)
	}
	return nil
}
