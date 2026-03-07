// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/plan"
	"github.com/spf13/cobra"
)

var referCmd = &cobra.Command{
	Use:   "refer",
	Short: "Print the content of a saved plan",
	Long: `Find a plan by name (exact or partial match against filename and topic)
and print its full content to stdout.

Use --summary to return only the sections listed in config's summary_sections,
saving tokens when the command is used by agents.

If multiple plans match the given name, a candidate list is printed and
the command exits with an error so the caller knows to narrow the search.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		summaryOnly, _ := cmd.Flags().GetBool("summary")
		return runRefer(name, summaryOnly)
	},
}

func init() {
	referCmd.Flags().StringP("name", "n", "", "Plan name to look up (exact or partial match against filename, topic, or ID)")
	_ = referCmd.MarkFlagRequired("name")
	referCmd.Flags().Bool("summary", false, "Return only summary_sections from config (saves tokens)")
	rootCmd.AddCommand(referCmd)
}

// runRefer is the testable core of the refer command.
func runRefer(name string, summaryOnly bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	plans, err := plan.LoadAll(root)
	if err != nil {
		// Non-fatal parse errors: warn but continue with what we have.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	matches := matchPlans(plans, name)

	switch len(matches) {
	case 0:
		return fmt.Errorf("no plan found matching %q", name)
	case 1:
		return printRefer(matches[0], summaryOnly, root)
	default:
		return printPlanCandidates(matches, name)
	}
}

// matchPlans returns all plans whose filename stem, topic, or ID contains name
// (case-insensitive). A single exact match on any of those three fields is
// returned alone, bypassing any partial matches.
func matchPlans(plans []plan.Plan, name string) []plan.Plan {
	lower := strings.ToLower(name)

	var exact, partial []plan.Plan

	for _, p := range plans {
		stem := strings.TrimSuffix(p.Filename, ".md")

		// Exact match (case-insensitive) on stem, topic, or ID.
		if strings.EqualFold(stem, name) ||
			strings.EqualFold(p.Topic, name) ||
			strings.EqualFold(p.ID, name) {
			exact = append(exact, p)
			continue
		}

		// Partial / substring match.
		if strings.Contains(strings.ToLower(stem), lower) ||
			strings.Contains(strings.ToLower(p.Topic), lower) ||
			strings.Contains(strings.ToLower(p.ID), lower) {
			partial = append(partial, p)
		}
	}

	// If there is exactly one exact match, prefer it over everything else.
	if len(exact) == 1 {
		return exact
	}

	return append(exact, partial...)
}

// printRefer writes the plan content to stdout.
// With summaryOnly=true, only the sections listed in config's summary_sections
// are printed; otherwise the full plan (frontmatter + body) is printed.
func printRefer(p plan.Plan, summaryOnly bool, root string) error {
	if summaryOnly {
		cfg, err := config.Load(root)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		out := plan.ExtractSections(p.Body, cfg.Plans.SummarySections)
		if out == "" {
			fmt.Fprintln(os.Stderr, "warning: no matching summary sections found in this plan")
		}
		fmt.Println(out)
		return nil
	}

	data, err := plan.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	if p.Body != "" {
		data = append(data, []byte(p.Body)...)
	}
	_, err = fmt.Print(string(data))
	return err
}

// printPlanCandidates writes a numbered list of matching plans to stderr and
// returns an error telling the caller to narrow the search.
func printPlanCandidates(plans []plan.Plan, name string) error {
	fmt.Fprintf(os.Stderr, "Multiple plans match %q:\n\n", name)
	for i, p := range plans {
		fmt.Fprintf(os.Stderr, "  %d. %s  (topic: %s)\n", i+1, p.Filename, p.Topic)
	}
	fmt.Fprintln(os.Stderr)
	return fmt.Errorf("use a more specific name to select one plan")
}
