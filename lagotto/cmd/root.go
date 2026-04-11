package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set via ldflags at build time
	Version = "dev"

	// Global flags
	outputFormat string
	jsonOutput   bool
	verbose      bool
	watchesTable string
	historyTable string
)

var rootCmd = &cobra.Command{
	Use:   "lagotto",
	Short: "Watch for EC2 instance capacity across regions",
	Long: `Lagotto watches for EC2 instance type availability that doesn't exist right now
(e.g., scarce GPU instances like p5.48xlarge, or Spot capacity in a preferred region)
and notifies you when capacity appears.

Named after the Lagotto Romagnolo, the truffle-hunting dog.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON (shorthand for -o json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&watchesTable, "watches-table", "lagotto-watches", "DynamoDB table name for watches")
	rootCmd.PersistentFlags().StringVar(&historyTable, "history-table", "lagotto-match-history", "DynamoDB table name for match history")

	rootCmd.CompletionOptions.DisableDefaultCmd = false
}

func getOutputFormat() string {
	if jsonOutput {
		return "json"
	}
	return outputFormat
}
