// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "logos",
	Short: "AI agent conversation context manager for git repositories",
	Long: `Logosyncx (logos) is a CLI tool for managing AI agent conversation context
in git repositories. It lets agents save, search, and retrieve past session
summaries — enabling team-wide context sharing without external databases
or embedding servers.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
