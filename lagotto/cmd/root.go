package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/scttfrdmn/spore-host/pkg/i18n"
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

	// i18n flags
	flagLang          string
	flagNoEmoji       bool
	flagAccessibility bool
)

var rootCmd = &cobra.Command{
	Use: "lagotto",
	// Short and Long set after i18n initialization
}

var i18nInitialized = false

// Execute runs the root command.
func Execute() {
	_ = rootCmd.ParseFlags(os.Args[1:])
	ensureI18nInitialized()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ensureI18nInitialized()
		return nil
	}

	rootCmd.PersistentFlags().StringVar(&flagLang, "lang", "", "Language for output (en, es, fr, de, ja, pt)")
	rootCmd.PersistentFlags().BoolVar(&flagNoEmoji, "no-emoji", false, "Disable emoji in output")
	rootCmd.PersistentFlags().BoolVar(&flagAccessibility, "accessibility", false, "Enable accessibility mode (implies --no-emoji)")

	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON (shorthand for -o json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&watchesTable, "watches-table", "lagotto-watches", "DynamoDB table name for watches")
	rootCmd.PersistentFlags().StringVar(&historyTable, "history-table", "lagotto-match-history", "DynamoDB table name for match history")

	rootCmd.CompletionOptions.DisableDefaultCmd = false
}

func ensureI18nInitialized() {
	if i18nInitialized {
		return
	}
	initI18n()
	i18nInitialized = true
}

func initI18n() {
	cfg := i18n.Config{
		Language:          flagLang,
		Verbose:           false,
		AccessibilityMode: flagAccessibility,
		NoEmoji:           flagNoEmoji,
	}

	if err := i18n.Init(cfg); err != nil {
		log.Printf("Warning: failed to initialize i18n: %v", err)
	}

	updateCommandDescriptions()
}

func updateCommandDescriptions() {
	rootCmd.Short = i18n.T("lagotto.root.short")
	rootCmd.Long = i18n.T("lagotto.root.long")

	if cmd, _, err := rootCmd.Find([]string{"watch"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.watch.short")
		cmd.Long = i18n.T("lagotto.watch.long")
	}
	if cmd, _, err := rootCmd.Find([]string{"list"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.list.short")
	}
	if cmd, _, err := rootCmd.Find([]string{"cancel"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.cancel.short")
	}
	if cmd, _, err := rootCmd.Find([]string{"status"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.status.short")
	}
	if cmd, _, err := rootCmd.Find([]string{"history"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.history.short")
	}
	if cmd, _, err := rootCmd.Find([]string{"poll"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.poll.short")
		cmd.Long = i18n.T("lagotto.poll.long")
	}
	if cmd, _, err := rootCmd.Find([]string{"version"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.version.short")
	}
	if cmd, _, err := rootCmd.Find([]string{"extend"}); err == nil && cmd != nil {
		cmd.Short = i18n.T("lagotto.extend.short")
	}
}

func getOutputFormat() string {
	if jsonOutput {
		return "json"
	}
	return outputFormat
}
