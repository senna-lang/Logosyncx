// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/session"
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

	sessions, err := session.LoadAll(root)
	if err != nil {
		// Non-fatal parse errors: warn but continue with what we have.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	// Apply --tag pre-filter.
	if tag != "" {
		sessions = filterTag(sessions, tag)
	}

	// Apply keyword filter.
	sessions = filterKeyword(sessions, keyword)

	// Sort newest first.
	sortByDateDesc(sessions)

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	return printTable(sessions)
}

// filterKeyword returns sessions whose topic, any tag, or excerpt contains
// keyword (case-insensitive substring match).
func filterKeyword(sessions []session.Session, keyword string) []session.Session {
	lower := strings.ToLower(keyword)
	var out []session.Session
	for _, s := range sessions {
		if sessionMatchesKeyword(s, lower) {
			out = append(out, s)
		}
	}
	return out
}

// sessionMatchesKeyword reports whether s contains lower (already lowercased)
// in its topic, any of its tags, or its excerpt.
func sessionMatchesKeyword(s session.Session, lower string) bool {
	if strings.Contains(strings.ToLower(s.Topic), lower) {
		return true
	}
	for _, t := range s.Tags {
		if strings.Contains(strings.ToLower(t), lower) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(s.Excerpt), lower) {
		return true
	}
	return false
}
