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
	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/senna-lang/logosyncx/pkg/knowledge"
	"github.com/senna-lang/logosyncx/pkg/plan"
	"github.com/spf13/cobra"
)

var distillCmd = &cobra.Command{
	Use:   "distill",
	Short: "Distill a completed plan into a knowledge file",
	Long: `Verify that all tasks are done and walkthroughs exist, then build a
knowledge file from the plan body and walkthrough content.

Pre-flight checks (all hard errors):
  1. Plan not found
  2. No tasks found for the plan
  3. Incomplete tasks exist (open or in_progress)
  4. No WALKTHROUGH.md files found
  5. Plan already distilled (skip with --force)

Use --dry-run to preview without writing any files.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		planPartial, _ := cmd.Flags().GetString("plan")
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return runDistill(planPartial, force, dryRun)
	},
}

func init() {
	distillCmd.Flags().StringP("plan", "P", "", "Plan to distill (partial name match, required)")
	_ = distillCmd.MarkFlagRequired("plan")
	distillCmd.Flags().Bool("force", false, "Re-distill even if the plan is already marked distilled")
	distillCmd.Flags().Bool("dry-run", false, "Preview without writing any files")
	rootCmd.AddCommand(distillCmd)
}

// runDistill is the testable core of the distill command.
func runDistill(planPartial string, force, dryRun bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Resolve plan ----------------------------------------------------------

	allPlans, err := plan.LoadAll(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	p, err := findPlan(planPartial, allPlans)
	if err != nil {
		return err
	}

	planSlug := strings.TrimSuffix(p.Filename, ".md")

	// --- Load tasks for this plan ----------------------------------------------

	store := task.NewStore(root, &cfg)
	tasks, err := store.List(task.Filter{Plan: planSlug})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning loading tasks: %v\n", err)
	}

	// Pre-flight check 2: no tasks.
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found for plan %q", planSlug)
	}

	// Pre-flight check 3: incomplete tasks.
	var incomplete []string
	for _, t := range tasks {
		if t.Status != task.StatusDone {
			incomplete = append(incomplete, fmt.Sprintf("%s (%s)", t.Title, t.Status))
		}
	}
	if len(incomplete) > 0 {
		return fmt.Errorf("incomplete tasks: [%s]. Complete or delete them first",
			strings.Join(incomplete, ", "))
	}

	// Pre-flight check 4: no walkthroughs.
	var walkthroughPaths []string
	for _, t := range tasks {
		wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
		if _, statErr := os.Stat(wtPath); statErr == nil {
			walkthroughPaths = append(walkthroughPaths, wtPath)
		}
	}
	if len(walkthroughPaths) == 0 {
		return fmt.Errorf("no walkthroughs found for plan %q — mark tasks done first", planSlug)
	}

	// Pre-flight check 5: already distilled.
	if p.Distilled && !force {
		return fmt.Errorf("plan %q already distilled. Re-run with --force to overwrite", planSlug)
	}

	// --- Build source material block ------------------------------------------

	sourceBlock := buildSourceBlock(p, tasks)

	// --- Read knowledge template -----------------------------------------------

	templatePath := filepath.Join(root, ".logosyncx", "templates", "knowledge.md")
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			templateContent = []byte{}
		} else {
			return fmt.Errorf("read knowledge template: %w", err)
		}
	}

	// --- Dry-run preview -------------------------------------------------------

	if dryRun {
		fmt.Println("--- DRY RUN PREVIEW ---")
		fmt.Printf("Plan:  %s\n", planSlug)
		fmt.Println("Tasks:")
		for _, t := range tasks {
			wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
			status := walkthroughFillStatus(wtPath)
			fmt.Printf("       %03d %s [%s]\n", t.Seq, t.Title, status)
		}
		fmt.Println()
		fmt.Println("--- SOURCE MATERIAL BLOCK ---")
		fmt.Println(sourceBlock)
		fmt.Println("--- END DRY RUN ---")
		return nil
	}

	// --- Write knowledge file --------------------------------------------------

	var taskFilenames []string
	for _, t := range tasks {
		taskFilenames = append(taskFilenames, fmt.Sprintf("%03d-%s", t.Seq, t.Title))
	}

	k := knowledge.Knowledge{
		Topic: p.Topic,
		Plan:  p.Filename,
		Tasks: taskFilenames,
		Tags:  p.Tags,
	}

	relKnowledgePath, err := knowledge.Write(root, k, sourceBlock, string(templateContent))
	if err != nil {
		return fmt.Errorf("write knowledge file: %w", err)
	}

	// Append walkthrough paths under the Source Walkthroughs section (§10.5).
	if appendErr := appendWalkthroughPaths(filepath.Join(root, relKnowledgePath), root, tasks); appendErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not append walkthrough paths: %v\n", appendErr)
	}

	// --- Mark plan as distilled -----------------------------------------------

	p.Distilled = true
	planPath, err := plan.Write(root, p)
	if err != nil {
		return fmt.Errorf("mark plan distilled: %w", err)
	}

	// Rebuild plan index and git add (best-effort).
	if _, err := index.Rebuild(root, cfg.Plans.ExcerptSection); err != nil {
		fmt.Fprintf(os.Stderr, "warning: rebuild index: %v\n", err)
	}
	_ = gitutil.Add(root, filepath.Join(root, relKnowledgePath))
	_ = gitutil.Add(root, planPath)

	// --- Output ---------------------------------------------------------------

	fmt.Printf("Plan:      %s\n", planSlug)
	fmt.Printf("Tasks:\n")
	for _, t := range tasks {
		wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
		status := walkthroughFillStatus(wtPath)
		fmt.Printf("           %03d %s [%s]\n", t.Seq, t.Title, status)
	}
	fmt.Println()
	fmt.Printf("✓ Knowledge file written: %s\n", relKnowledgePath)
	rel, _ := relPath(root, planPath)
	fmt.Printf("✓ Plan marked as distilled: %s\n", rel)
	fmt.Println()
	fmt.Printf("Next: Open %s and fill in the sections.\n", relKnowledgePath)

	return nil
}

// appendWalkthroughPaths appends a bullet list of WALKTHROUGH.md relative
// paths to the knowledge file (under the existing ## Source Walkthroughs heading).
func appendWalkthroughPaths(knowledgePath, projectRoot string, tasks []*task.Task) error {
	f, err := os.OpenFile(knowledgePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, t := range tasks {
		wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
		rel, relErr := filepath.Rel(projectRoot, wtPath)
		if relErr != nil {
			rel = wtPath
		}
		fmt.Fprintf(f, "\n- %s", rel)
	}
	_, err = fmt.Fprintln(f)
	return err
}

// buildSourceBlock assembles the HTML comment content from the plan body
// and all task walkthroughs.
func buildSourceBlock(p plan.Plan, tasks []*task.Task) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## Plan: %s\n\n", p.Topic)
	if p.Body != "" {
		sb.WriteString(strings.TrimSpace(p.Body))
		sb.WriteString("\n")
	}

	for _, t := range tasks {
		sb.WriteString("\n---\n")
		fmt.Fprintf(&sb, "## Walkthrough: %03d %s\n\n", t.Seq, t.Title)
		wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
		data, err := os.ReadFile(wtPath)
		if err == nil {
			sb.WriteString(strings.TrimSpace(string(data)))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
