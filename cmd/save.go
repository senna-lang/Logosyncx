// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/senna-lang/logosyncx/pkg/plan"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Args:  cobra.NoArgs,
	Short: "Scaffold a plan file in .logosyncx/plans/",
	Long: `Create a plan frontmatter scaffold in .logosyncx/plans/.

  logos save --topic "..." [--tag <tag>] [--agent <agent>] \
             [--related <plan>] [--depends-on <partial-plan-name>]

The CLI writes frontmatter only. Open the file and fill in the body sections
guided by .logosyncx/templates/plan.md.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		topic, _ := cmd.Flags().GetString("topic")
		tags, _ := cmd.Flags().GetStringArray("tag")
		agent, _ := cmd.Flags().GetString("agent")
		related, _ := cmd.Flags().GetStringArray("related")
		dependsOn, _ := cmd.Flags().GetStringArray("depends-on")
		return runSave(topic, tags, agent, related, dependsOn)
	},
}

func init() {
	saveCmd.Flags().StringP("topic", "t", "", "Plan topic (required)")
	saveCmd.Flags().StringArray("tag", []string{}, "Tag to attach (repeatable: --tag go --tag cli)")
	saveCmd.Flags().StringP("agent", "a", "", "Agent name (e.g. claude-code)")
	saveCmd.Flags().StringArray("related", []string{}, "Related plan filename (repeatable)")
	saveCmd.Flags().StringArray("depends-on", []string{}, "Plan this depends on (partial name, repeatable)")
	rootCmd.AddCommand(saveCmd)
}

func runSave(topic string, tags []string, agent string, related []string, dependsOnPartials []string) error {
	if strings.TrimSpace(topic) == "" {
		return errors.New("provide --topic <topic>")
	}

	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Load existing plans to resolve --depends-on partial matches.
	allPlans, err := plan.LoadAll(root)
	if err != nil {
		return fmt.Errorf("load plans: %w", err)
	}

	resolvedDeps, err := resolveDependsOn(dependsOnPartials, allPlans)
	if err != nil {
		return err
	}

	// Check for circular plan dependencies.
	candidateFilename := plan.FileName(plan.Plan{Topic: topic})
	if err := detectCircular(candidateFilename, resolvedDeps, allPlans); err != nil {
		return err
	}

	id, err := plan.GenerateID()
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}

	p := plan.Plan{
		ID:        id,
		Topic:     topic,
		Tags:      tags,
		Agent:     agent,
		Related:   related,
		DependsOn: resolvedDeps,
	}

	// DefaultTasksDir is set after FileName is known.
	filename := plan.FileName(p)
	p.TasksDir = plan.DefaultTasksDir(filename)

	savedPath, err := plan.Write(root, p)
	if err != nil {
		return fmt.Errorf("write plan: %w", err)
	}

	rel, _ := relPath(root, savedPath)
	fmt.Printf("✓ Created plan: %s\n", rel)

	// Rebuild the full plan index so logos ls reflects the new plan immediately.
	if _, indexErr := index.Rebuild(root, cfg.Plans.ExcerptSection); indexErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not rebuild index (%v) — run `logos sync` to rebuild\n", indexErr)
	}

	// Stage with git (best-effort).
	_ = gitutil.Add(root, savedPath)
	_ = gitutil.Add(root, index.FilePath(root))

	fmt.Println()
	fmt.Printf("Next: fill in the plan body in %s\n", rel)
	fmt.Printf("      (read .logosyncx/templates/plan.md for section structure)\n")
	return nil
}

// detectCircular returns an error if candidateFilename is a transitive
// dependency of itself via the resolved deps slice.
func detectCircular(candidateFilename string, deps []string, allPlans []plan.Plan) error {
	planByFile := make(map[string]plan.Plan, len(allPlans))
	for _, p := range allPlans {
		planByFile[p.Filename] = p
	}

	visited := make(map[string]bool)
	queue := append([]string{}, deps...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == candidateFilename {
			return fmt.Errorf("circular dependency: %q depends on itself transitively", candidateFilename)
		}
		if visited[cur] {
			continue
		}
		visited[cur] = true
		if p, ok := planByFile[cur]; ok {
			queue = append(queue, p.DependsOn...)
		}
	}
	return nil
}

// resolveDependsOn resolves partial plan name matches for --depends-on flags.
// Returns an error if any partial matches 0 or 2+ plans.
func resolveDependsOn(partials []string, allPlans []plan.Plan) ([]string, error) {
	if len(partials) == 0 {
		return nil, nil
	}

	resolved := make([]string, 0, len(partials))
	for _, partial := range partials {
		var matches []string
		for _, p := range allPlans {
			if strings.Contains(p.Filename, partial) {
				matches = append(matches, p.Filename)
			}
		}
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("plan %q not found", partial)
		case 1:
			resolved = append(resolved, matches[0])
		default:
			return nil, fmt.Errorf("ambiguous plan name %q: matches [%s]", partial, strings.Join(matches, ", "))
		}
	}
	return resolved, nil
}

// relPath returns the path of target relative to base, falling back to target.
func relPath(base, target string) (string, error) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target, err
	}
	return rel, nil
}
