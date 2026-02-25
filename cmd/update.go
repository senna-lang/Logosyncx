package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/senna-lang/logosyncx/internal/updater"
	"github.com/senna-lang/logosyncx/internal/version"
	"github.com/spf13/cobra"
)

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update logos to the latest release",
	Long: `Check for a newer version of logos on GitHub Releases and install it.

By default, logos update downloads and installs the latest release,
atomically replacing the current binary.

Use --check to only report whether an update is available without installing.

Examples:
  logos update           # download and install the latest release
  logos update --check   # check only; print status, do not install`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Check for updates without installing")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	current := version.Version

	if version.IsDev() {
		fmt.Fprintln(os.Stderr, "logos update is not available for development builds.")
		fmt.Fprintln(os.Stderr, "Build a release binary or download one from GitHub Releases.")
		return nil
	}

	fmt.Printf("Current version: %s\n", current)
	fmt.Println("Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	latest, err := updater.FetchLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("could not reach GitHub Releases: %w\nCheck your network connection and try again.", err)
	}

	if latest == current {
		fmt.Printf("Already up to date (%s).\n", current)
		return nil
	}

	// Determine direction: newer or older.
	// We compare via the updater's semverGreater; expose the result through a
	// simple local check so we can warn the user if they are somehow running a
	// version newer than the latest release.
	if !semverGT(latest, current) {
		fmt.Printf("Already up to date (%s).\n", current)
		return nil
	}

	fmt.Printf("New version available: %s → %s\n", current, latest)

	if updateCheckOnly {
		fmt.Printf("Run 'logos update' (without --check) to install %s.\n", latest)
		return nil
	}

	// Resolve path of the running binary before overwriting it.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine path of current binary: %w", err)
	}

	fmt.Printf("Downloading logos %s ...\n", latest)

	// Use a longer timeout for the actual download.
	dlCtx, dlCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer dlCancel()

	if err := updater.Apply(dlCtx, latest, execPath); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("Updated logos to %s\n", latest)
	fmt.Println("Run 'logos version' to confirm.")

	// Invalidate the local update-check cache so that the next invocation
	// does not immediately show an (already resolved) update hint.
	_ = clearUpdateCache()

	return nil
}

// clearUpdateCache removes the cached update-check result so that the next
// invocation re-queries the API.
func clearUpdateCache() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	cacheFile := configDir + "/logosyncx/update-check.json"
	err = os.Remove(cacheFile)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// semverGT is a thin wrapper that mirrors the logic in the updater package,
// kept here to avoid exporting an internal helper.
func semverGT(a, b string) bool {
	// Delegate to the updater's exported check by asking "is a an update over b?"
	// We achieve this by using CheckWithCache with a guaranteed-stale cache path.
	// Instead, implement the comparison directly — it's simple enough.
	av := parseVer(a)
	bv := parseVer(b)
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return true
		}
		if av[i] < bv[i] {
			return false
		}
	}
	return false
}

func parseVer(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var r [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		p = strings.SplitN(p, "-", 2)[0]
		n, _ := strconv.Atoi(p)
		r[i] = n
	}
	return r
}
