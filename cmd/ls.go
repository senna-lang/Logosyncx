package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List saved plans",
	Long: `Display a list of saved plans in .logosyncx/plans/.

Without flags, prints a human-readable table sorted by date (newest first).
Use --json to get structured output with excerpts, suitable for agent consumption.
Use --blocked to show only plans blocked by an undistilled dependency.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		since, _ := cmd.Flags().GetString("since")
		asJSON, _ := cmd.Flags().GetBool("json")
		blocked, _ := cmd.Flags().GetBool("blocked")
		if asJSON {
			suppressUpdateCheck = true
		}
		return runLS(tag, since, asJSON, blocked)
	},
}

func init() {
	lsCmd.Flags().StringP("tag", "t", "", "Filter plans by tag")
	lsCmd.Flags().StringP("since", "s", "", "Filter plans on or after this date (YYYY-MM-DD)")
	lsCmd.Flags().Bool("json", false, "Output structured JSON (for agent consumption)")
	lsCmd.Flags().Bool("blocked", false, "Show only plans blocked by an undistilled dependency")
	rootCmd.AddCommand(lsCmd)
}

func runLS(tag, since string, asJSON, blocked bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	entries, err := index.ReadAll(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Auto-rebuild: inform the user and build the index on the fly.
			fmt.Fprintln(os.Stderr, "index.jsonl not found. Building index from plans/...")
			cfg, cfgErr := config.Load(root)
			if cfgErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not load config (%v) — using defaults\n", cfgErr)
				cfg = config.Default("")
			}
			n, buildErr := index.Rebuild(root, cfg.Plans.ExcerptSection)
			if buildErr != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", buildErr)
			}
			fmt.Fprintf(os.Stderr, "Done. %d plans indexed.\n\n", n)
			entries, err = index.ReadAll(root)
			if err != nil {
				return fmt.Errorf("read index after rebuild: %w", err)
			}
		} else {
			return fmt.Errorf("read index: %w", err)
		}
	}

	// Apply --since filter.
	if since != "" {
		sinceTime, err := time.Parse("2006-01-02", since)
		if err != nil {
			return fmt.Errorf("invalid --since date %q: expected YYYY-MM-DD", since)
		}
		entries = filterSince(entries, sinceTime)
	}

	// Apply --tag filter.
	if tag != "" {
		entries = filterTag(entries, tag)
	}

	// Apply --blocked filter.
	if blocked {
		entries = filterBlocked(entries)
	}

	// Sort newest first.
	sortByDateDesc(entries)

	if len(entries) == 0 {
		fmt.Println("No plans found.")
		return nil
	}

	if asJSON {
		return printJSON(entries)
	}
	return printTable(entries)
}

// printTable writes a human-readable tab-aligned table to stdout.
func printTable(entries []index.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tTOPIC\tTAGS\tDISTILLED")
	fmt.Fprintln(w, "----\t-----\t----\t---------")
	for _, e := range entries {
		date := e.Date.Format("2006-01-02 15:04")
		tags := joinTags(e.Tags)
		distilled := "no"
		if e.Distilled {
			distilled = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", date, e.Topic, tags, distilled)
	}
	return w.Flush()
}

// printJSON writes the entries as a JSON array to stdout.
func printJSON(entries []index.Entry) error {
	// Normalise nil slices so JSON output always uses [] rather than null.
	out := make([]index.Entry, len(entries))
	for i, e := range entries {
		if e.Tags == nil {
			e.Tags = []string{}
		}
		if e.Related == nil {
			e.Related = []string{}
		}
		out[i] = e
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// --- filters -----------------------------------------------------------------

func filterSince(entries []index.Entry, since time.Time) []index.Entry {
	// Truncate to date only for comparison.
	sinceDate := since.Truncate(24 * time.Hour)
	var out []index.Entry
	for _, e := range entries {
		sessionDate := e.Date.UTC().Truncate(24 * time.Hour)
		if !sessionDate.Before(sinceDate) {
			out = append(out, e)
		}
	}
	return out
}

func filterBlocked(entries []index.Entry) []index.Entry {
	var out []index.Entry
	for _, e := range entries {
		if e.Blocked {
			out = append(out, e)
		}
	}
	return out
}

func filterTag(entries []index.Entry, tag string) []index.Entry {
	var out []index.Entry
	for _, e := range entries {
		if slices.Contains(e.Tags, tag) {
			out = append(out, e)
		}
	}
	return out
}

// --- sort --------------------------------------------------------------------

// sortByDateDesc sorts entries newest-first (in-place).
func sortByDateDesc(entries []index.Entry) {
	slices.SortFunc(entries, func(a, b index.Entry) int {
		return b.Date.Compare(a.Date)
	})
}

// --- helpers -----------------------------------------------------------------

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += ", "
		}
		result += t
	}
	return result
}
