package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display version, build date, and git commit information for truffle.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🍄 Truffle - AWS EC2 Instance Type Finder\n\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Printf("\nProject:    https://github.com/scttfrdmn/spore-host\n")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
