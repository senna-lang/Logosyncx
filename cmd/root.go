// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/senna-lang/logosyncx/internal/updater"
	"github.com/senna-lang/logosyncx/internal/version"
	"github.com/spf13/cobra"
)

// suppressUpdateCheck can be set to true by commands that emit --json output
// so that the update hint is not mixed into machine-readable stdout.
var suppressUpdateCheck bool

var rootCmd = &cobra.Command{
	Use:   "logos",
	Short: "AI agent conversation context manager for git repositories",
	Long: `Logosyncx (logos) is a CLI tool for managing AI agent conversation context
in git repositories. It lets agents save, search, and retrieve past session
summaries — enabling team-wide context sharing without external databases
or embedding servers.`,
	// PersistentPostRun fires after every subcommand (including nested ones).
	// It performs a lightweight update check and prints a one-line hint to
	// stderr when a newer version is available.
	//
	// Rules:
	//   - Skipped for dev builds (no meaningful version to compare against).
	//   - Skipped when LOGOS_NO_UPDATE_CHECK=1 (CI / automation opt-out).
	//   - Skipped when the subcommand set suppressUpdateCheck = true (--json output).
	//   - The check is served from a local cache file; a network call is only
	//     made when the cache is older than 24 hours.
	//   - A 2-second context deadline prevents any noticeable latency on the
	//     once-per-day network refresh.
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		printUpdateHintIfAvailable()
	},
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

// printUpdateHintIfAvailable checks for an available update and prints a
// one-line hint to stderr if one is found. It returns immediately without
// printing anything on error or when the check is suppressed.
func printUpdateHintIfAvailable() {
	if suppressUpdateCheck {
		return
	}
	if version.IsDev() {
		return
	}
	if os.Getenv("LOGOS_NO_UPDATE_CHECK") == "1" {
		return
	}

	// 2-second budget: served from cache (instant) on most invocations;
	// only hits the network once per day when the cache is stale.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	latest, err := updater.CheckWithCache(ctx, version.Version)
	if err != nil || latest == "" {
		return
	}

	fmt.Fprintf(os.Stderr, "\nA new version of logos is available: %s\n", latest)
	fmt.Fprintf(os.Stderr, "Run 'logos update' to upgrade.\n")
}
