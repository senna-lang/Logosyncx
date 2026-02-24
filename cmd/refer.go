// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/session"
	"github.com/spf13/cobra"
)

var referCmd = &cobra.Command{
	Use:   "refer <name>",
	Short: "Print the content of a saved session",
	Long: `Find a session by name (exact or partial match against filename and topic)
and print its full content to stdout.

Use --summary to return only the sections listed in config's summary_sections,
saving tokens when the command is used by agents.

If multiple sessions match the given name, a candidate list is printed and
the command exits with an error so the caller knows to narrow the search.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		summaryOnly, _ := cmd.Flags().GetBool("summary")
		return runRefer(args[0], summaryOnly)
	},
}

func init() {
	referCmd.Flags().Bool("summary", false, "Return only summary_sections from config (saves tokens)")
	rootCmd.AddCommand(referCmd)
}

// runRefer is the testable core of the refer command.
func runRefer(name string, summaryOnly bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	sessions, err := session.LoadAll(root)
	if err != nil {
		// Non-fatal parse errors: warn but continue with what we have.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	matches := matchSessions(sessions, name)

	switch len(matches) {
	case 0:
		return fmt.Errorf("no session found matching %q", name)
	case 1:
		return printRefer(matches[0], summaryOnly, root)
	default:
		return printCandidates(matches, name)
	}
}

// matchSessions returns all sessions whose filename stem, topic, or ID
// contains name (case-insensitive). A single exact match — on any of those
// three fields — is returned alone, bypassing any partial matches.
func matchSessions(sessions []session.Session, name string) []session.Session {
	lower := strings.ToLower(name)

	var exact, partial []session.Session

	for _, s := range sessions {
		stem := strings.TrimSuffix(s.Filename, ".md")

		// Exact match (case-insensitive) on stem, topic, or ID.
		if strings.EqualFold(stem, name) ||
			strings.EqualFold(s.Topic, name) ||
			strings.EqualFold(s.ID, name) {
			exact = append(exact, s)
			continue
		}

		// Partial / substring match.
		if strings.Contains(strings.ToLower(stem), lower) ||
			strings.Contains(strings.ToLower(s.Topic), lower) ||
			strings.Contains(strings.ToLower(s.ID), lower) {
			partial = append(partial, s)
		}
	}

	// If there is exactly one exact match, prefer it over everything else.
	if len(exact) == 1 {
		return exact
	}

	return append(exact, partial...)
}

// printRefer writes the session content to stdout.
// With summaryOnly=true, only the sections listed in config's summary_sections
// are printed; otherwise the full session (frontmatter + body) is printed.
func printRefer(s session.Session, summaryOnly bool, root string) error {
	if summaryOnly {
		cfg, err := config.Load(root)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		out := session.ExtractSections(s.Body, cfg.Sessions.SummarySections)
		if out == "" {
			fmt.Fprintln(os.Stderr, "warning: no matching summary sections found in this session")
		}
		fmt.Println(out)
		return nil
	}

	data, err := session.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	_, err = fmt.Print(string(data))
	return err
}

// printCandidates writes a numbered list of matching sessions to stderr and
// returns an error telling the caller to narrow the search.
func printCandidates(sessions []session.Session, name string) error {
	fmt.Fprintf(os.Stderr, "Multiple sessions match %q:\n\n", name)
	for i, s := range sessions {
		fmt.Fprintf(os.Stderr, "  %d. %s  (topic: %s)\n", i+1, s.Filename, s.Topic)
	}
	fmt.Fprintln(os.Stderr)
	return fmt.Errorf("use a more specific name to select one session")
}
