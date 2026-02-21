package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/session"
	"github.com/spf13/cobra"
)

// lsJSONOutput is the JSON shape returned by logos ls --json.
type lsJSONOutput struct {
	ID       string    `json:"id"`
	Filename string    `json:"filename"`
	Date     time.Time `json:"date"`
	Topic    string    `json:"topic"`
	Tags     []string  `json:"tags"`
	Agent    string    `json:"agent"`
	Related  []string  `json:"related"`
	Excerpt  string    `json:"excerpt"`
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List saved sessions",
	Long: `Display a list of saved sessions in .logosyncx/sessions/.

Without flags, prints a human-readable table sorted by date (newest first).
Use --json to get structured output with excerpts, suitable for agent consumption.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, _ := cmd.Flags().GetString("tag")
		since, _ := cmd.Flags().GetString("since")
		asJSON, _ := cmd.Flags().GetBool("json")
		return runLS(tag, since, asJSON)
	},
}

func init() {
	lsCmd.Flags().StringP("tag", "t", "", "Filter sessions by tag")
	lsCmd.Flags().StringP("since", "s", "", "Filter sessions on or after this date (YYYY-MM-DD)")
	lsCmd.Flags().Bool("json", false, "Output structured JSON (for agent consumption)")
	rootCmd.AddCommand(lsCmd)
}

func runLS(tag, since string, asJSON bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	sessions, err := session.LoadAll(root)
	if err != nil {
		// Non-fatal parse errors: print a warning but continue with what we have.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	// Apply --since filter.
	if since != "" {
		sinceTime, err := time.Parse("2006-01-02", since)
		if err != nil {
			return fmt.Errorf("invalid --since date %q: expected YYYY-MM-DD", since)
		}
		sessions = filterSince(sessions, sinceTime)
	}

	// Apply --tag filter.
	if tag != "" {
		sessions = filterTag(sessions, tag)
	}

	// Sort newest first.
	sortByDateDesc(sessions)

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	if asJSON {
		return printJSON(sessions)
	}
	return printTable(sessions)
}

// printTable writes a human-readable tab-aligned table to stdout.
func printTable(sessions []session.Session) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tTOPIC\tTAGS")
	fmt.Fprintln(w, "----\t-----\t----")
	for _, s := range sessions {
		date := s.Date.Format("2006-01-02 15:04")
		tags := joinTags(s.Tags)
		fmt.Fprintf(w, "%s\t%s\t%s\n", date, s.Topic, tags)
	}
	return w.Flush()
}

// printJSON writes the sessions as a JSON array to stdout.
func printJSON(sessions []session.Session) error {
	out := make([]lsJSONOutput, len(sessions))
	for i, s := range sessions {
		tags := s.Tags
		if tags == nil {
			tags = []string{}
		}
		related := s.Related
		if related == nil {
			related = []string{}
		}
		out[i] = lsJSONOutput{
			ID:       s.ID,
			Filename: s.Filename,
			Date:     s.Date,
			Topic:    s.Topic,
			Tags:     tags,
			Agent:    s.Agent,
			Related:  related,
			Excerpt:  s.Excerpt,
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// --- filters -----------------------------------------------------------------

func filterSince(sessions []session.Session, since time.Time) []session.Session {
	// Truncate to date only for comparison.
	sinceDate := since.Truncate(24 * time.Hour)
	var out []session.Session
	for _, s := range sessions {
		sessionDate := s.Date.UTC().Truncate(24 * time.Hour)
		if !sessionDate.Before(sinceDate) {
			out = append(out, s)
		}
	}
	return out
}

func filterTag(sessions []session.Session, tag string) []session.Session {
	var out []session.Session
	for _, s := range sessions {
		for _, t := range s.Tags {
			if t == tag {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// --- sort --------------------------------------------------------------------

// sortByDateDesc sorts sessions newest-first (in-place).
func sortByDateDesc(sessions []session.Session) {
	for i := 1; i < len(sessions); i++ {
		for j := i; j > 0 && sessions[j].Date.After(sessions[j-1].Date); j-- {
			sessions[j], sessions[j-1] = sessions[j-1], sessions[j]
		}
	}
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
