package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Keyword search across session topic, tags, and excerpt",
	Long: `Case-insensitive keyword search across the topic, tags, and excerpt of every
saved session. Results are printed as a human-readable table sorted by date
(newest first).

Combine with --tag to pre-filter by tag before applying the keyword match.

For deeper semantic search, use 'logos ls --json' and let the agent reason
over the full excerpt list — no embedding API required.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		return runSearch(args[0], tag)
	},
}

func init() {
	searchCmd.Flags().StringP("tag", "t", "", "Pre-filter sessions by tag before applying the keyword match")
	rootCmd.AddCommand(searchCmd)
}

// runSearch is the testable core of the search command.
func runSearch(keyword, tag string) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	entries, err := index.ReadAll(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Auto-rebuild: inform the user and build the index on the fly.
			fmt.Fprintln(os.Stderr, "index.jsonl not found. Building index from sessions/...")
			cfg, cfgErr := config.Load(root)
			if cfgErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not load config (%v) — using defaults\n", cfgErr)
				cfg = config.Default("")
			}
			n, buildErr := index.Rebuild(root, cfg.Sessions.ExcerptSection)
			if buildErr != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", buildErr)
			}
			fmt.Fprintf(os.Stderr, "Done. %d sessions indexed.\n\n", n)
			entries, err = index.ReadAll(root)
			if err != nil {
				return fmt.Errorf("read index after rebuild: %w", err)
			}
		} else {
			return fmt.Errorf("read index: %w", err)
		}
	}

	// Apply --tag pre-filter.
	if tag != "" {
		entries = filterTag(entries, tag)
	}

	// Apply keyword filter.
	entries = filterKeyword(entries, keyword)

	// Sort newest first.
	sortByDateDesc(entries)

	if len(entries) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	return printTable(entries)
}

// filterKeyword returns entries whose topic, any tag, or excerpt contains
// keyword (case-insensitive substring match).
func filterKeyword(entries []index.Entry, keyword string) []index.Entry {
	lower := strings.ToLower(keyword)
	var out []index.Entry
	for _, e := range entries {
		if entryMatchesKeyword(e, lower) {
			out = append(out, e)
		}
	}
	return out
}

// entryMatchesKeyword reports whether e contains lower (already lowercased)
// in its topic, any of its tags, or its excerpt.
func entryMatchesKeyword(e index.Entry, lower string) bool {
	if strings.Contains(strings.ToLower(e.Topic), lower) {
		return true
	}
	for _, t := range e.Tags {
		if strings.Contains(strings.ToLower(t), lower) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(e.Excerpt), lower) {
		return true
	}
	return false
}
