// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/plan"
	"github.com/spf13/cobra"
)

// --- root task command -------------------------------------------------------

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks in .logosyncx/tasks/",
	Long: `Create, list, update, and delete task files stored under
.logosyncx/tasks/<plan-slug>/. Tasks are linked to plans and tracked
in git alongside plan context.`,
}

func init() {
	taskCmd.AddCommand(
		taskCreateCmd,
		taskLsCmd,
		taskReferCmd,
		taskUpdateCmd,
		taskDeleteCmd,
		taskSearchCmd,
		taskWalkthroughCmd,
	)
	rootCmd.AddCommand(taskCmd)
}

// --- logos task create -------------------------------------------------------

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Args:  cobra.NoArgs,
	Short: "Create a new task file",
	Long: `Create a task in .logosyncx/tasks/<plan>/ using flag-based input.

  logos task create --plan <plan-partial> --title "..." \
                    [--priority high|medium|low] [--tag <tag>] \
                    [--depends-on <seq>]

Resolves --plan against plan files in .logosyncx/plans/. Writes a
frontmatter scaffold only; the body is written by the agent using the
Write tool after reading .logosyncx/templates/task.md.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planPartial, _ := cmd.Flags().GetString("plan")
		title, _ := cmd.Flags().GetString("title")
		priority, _ := cmd.Flags().GetString("priority")
		tags, _ := cmd.Flags().GetStringArray("tag")
		dependsOn, _ := cmd.Flags().GetIntSlice("depends-on")

		root, err := project.FindRoot()
		if err != nil {
			return err
		}

		// Resolve --plan partial against plan files.
		allPlans, err := plan.LoadAll(root)
		if err != nil {
			return fmt.Errorf("load plans: %w", err)
		}
		resolvedPlan, err := findPlan(planPartial, allPlans)
		if err != nil {
			return err
		}

		// Reject task creation on a blocked plan (§8.2).
		if blocker := blockedByDep(resolvedPlan, allPlans); blocker != "" {
			return fmt.Errorf("plan %q is blocked: dependency %q is not yet distilled — distill it first", strings.TrimSuffix(resolvedPlan.Filename, ".md"), blocker)
		}

		planSlug := strings.TrimSuffix(resolvedPlan.Filename, ".md")

		return runTaskCreate(root, planSlug, title, priority, tags, dependsOn)
	},
}

func init() {
	taskCreateCmd.Flags().StringP("plan", "P", "", "Plan to attach this task to (partial name match, required)")
	_ = taskCreateCmd.MarkFlagRequired("plan")
	taskCreateCmd.Flags().StringP("title", "T", "", "Task title (required)")
	_ = taskCreateCmd.MarkFlagRequired("title")
	taskCreateCmd.Flags().StringP("priority", "p", "medium", "Task priority (high|medium|low)")
	taskCreateCmd.Flags().StringArray("tag", []string{}, "Tag to attach (repeatable: --tag go --tag cli)")
	taskCreateCmd.Flags().IntSlice("depends-on", []int{}, "Seq number of a task this depends on (repeatable)")
}

// runTaskCreate creates a task under the given planSlug (resolved by caller).
func runTaskCreate(root, planSlug, title, priority string, tags []string, dependsOn []int) error {
	p := task.Priority(priority)
	if priority != "" && !task.IsValidPriority(p) {
		return fmt.Errorf("invalid priority %q: must be one of high, medium, low", priority)
	}

	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	t := task.Task{
		Title:     title,
		Priority:  p,
		Plan:      planSlug,
		Tags:      tags,
		DependsOn: dependsOn,
	}

	store := task.NewStore(root, &cfg)

	createdPath, err := store.Create(&t)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	rel, _ := relPath(root, createdPath)
	fmt.Printf("✓ Created task: %s  (seq: %d)\n", rel, t.Seq)
	fmt.Println()
	fmt.Printf("Next: read .logosyncx/templates/task.md, then fill in %s\n", rel)
	return nil
}

// blockedByDep returns the filename of the first unfinished dependency of p,
// or "" if p is not blocked. A plan is blocked when any plan listed in
// DependsOn has Distilled == false.
func blockedByDep(p plan.Plan, allPlans []plan.Plan) string {
	planByFile := make(map[string]plan.Plan, len(allPlans))
	for _, pl := range allPlans {
		planByFile[pl.Filename] = pl
	}
	for _, dep := range p.DependsOn {
		if d, ok := planByFile[dep]; ok && !d.Distilled {
			return dep
		}
	}
	return ""
}

// findPlan resolves a partial plan name to a single plan.
// Returns an error if 0 or 2+ plans match.
func findPlan(partial string, allPlans []plan.Plan) (plan.Plan, error) {
	if partial == "" {
		return plan.Plan{}, fmt.Errorf("--plan is required")
	}
	var matches []plan.Plan
	for _, p := range allPlans {
		if strings.Contains(p.Filename, partial) {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return plan.Plan{}, fmt.Errorf("plan %q not found", partial)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Filename
		}
		return plan.Plan{}, fmt.Errorf("ambiguous plan name %q: matches [%s]", partial, strings.Join(names, ", "))
	}
}

// --- logos task ls -----------------------------------------------------------

var taskLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List tasks",
	Long: `Display a table of tasks in .logosyncx/tasks/, sorted newest first.
Use --json for structured output suitable for agent consumption.
Use --blocked to show only tasks blocked by unfinished dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planPartial, _ := cmd.Flags().GetString("plan")
		statusStr, _ := cmd.Flags().GetString("status")
		priorityStr, _ := cmd.Flags().GetString("priority")
		tagStr, _ := cmd.Flags().GetString("tag")
		asJSON, _ := cmd.Flags().GetBool("json")
		blocked, _ := cmd.Flags().GetBool("blocked")
		if asJSON {
			suppressUpdateCheck = true
		}
		return runTaskLS(planPartial, statusStr, priorityStr, tagStr, asJSON, blocked)
	},
}

func init() {
	taskLsCmd.Flags().StringP("plan", "P", "", "Filter by plan slug (substring match)")
	taskLsCmd.Flags().String("status", "", "Filter by status (open, in_progress, done)")
	taskLsCmd.Flags().String("priority", "", "Filter by priority (high, medium, low)")
	taskLsCmd.Flags().StringP("tag", "t", "", "Filter by tag (exact match)")
	taskLsCmd.Flags().Bool("json", false, "Output structured JSON (for agent consumption)")
	taskLsCmd.Flags().Bool("blocked", false, "Show only tasks blocked by unfinished dependencies")
}

func runTaskLS(planPartial, statusStr, priorityStr, tagStr string, asJSON, blocked bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	entries, err := task.ReadAllTaskIndex(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "task-index.jsonl not found. Building index from tasks/...")
			n, buildErr := store.RebuildTaskIndex()
			if buildErr != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", buildErr)
			}
			fmt.Fprintf(os.Stderr, "Done. %d tasks indexed.\n\n", n)
			entries, err = task.ReadAllTaskIndex(root)
			if err != nil {
				return fmt.Errorf("read task index after rebuild: %w", err)
			}
		} else {
			return fmt.Errorf("read task index: %w", err)
		}
	}

	f := task.Filter{
		Plan:     planPartial,
		Status:   task.Status(statusStr),
		Priority: task.Priority(priorityStr),
		Blocked:  blocked,
	}
	if tagStr != "" {
		f.Tags = []string{tagStr}
	}

	filtered := task.ApplyToJSON(entries, f)
	task.SortJSONByDateDesc(filtered)

	if len(filtered) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	if asJSON {
		return printTaskJSON(filtered)
	}
	return printTaskTable(filtered)
}

// --- logos task refer --------------------------------------------------------

var taskReferCmd = &cobra.Command{
	Use:   "refer",
	Short: "Print the content of a task file",
	Long: `Print a task file to stdout. Use --summary to print only the sections
listed in config.tasks.summary_sections (saves tokens). Use --plan to
narrow the search when task names are ambiguous across plans.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		planPartial, _ := cmd.Flags().GetString("plan")
		summary, _ := cmd.Flags().GetBool("summary")
		return runTaskRefer(name, planPartial, summary)
	},
}

func init() {
	taskReferCmd.Flags().StringP("name", "n", "", "Task name to look up (exact or partial match against task dir name)")
	_ = taskReferCmd.MarkFlagRequired("name")
	taskReferCmd.Flags().StringP("plan", "P", "", "Plan slug to narrow the search (substring match)")
	taskReferCmd.Flags().Bool("summary", false, "Print only summary sections (saves tokens)")
}

func runTaskRefer(nameOrPartial, planPartial string, summary bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	t, err := store.Get(planPartial, nameOrPartial)
	if err != nil {
		return err
	}

	if summary {
		sections := task.ExtractSections(t.Body, cfg.Tasks.SummarySections)
		if sections == "" {
			fmt.Fprintln(os.Stderr, "warning: no matching summary sections found in this task")
		}
		fmt.Println(sections)
	} else {
		data, err := task.Marshal(*t)
		if err != nil {
			return fmt.Errorf("marshal task: %w", err)
		}
		fmt.Print(string(data))
	}
	return nil
}

// --- logos task update -------------------------------------------------------

var taskUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update task fields",
	Long: `Update frontmatter fields of a task. Supported flags: --name, --status,
--priority, --assignee. Use --plan to narrow the search when task names are
ambiguous across plans.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		planPartial, _ := cmd.Flags().GetString("plan")
		statusStr, _ := cmd.Flags().GetString("status")
		priorityStr, _ := cmd.Flags().GetString("priority")
		assignee, _ := cmd.Flags().GetString("assignee")
		return runTaskUpdate(planPartial, name, statusStr, priorityStr, assignee)
	},
}

func init() {
	taskUpdateCmd.Flags().StringP("name", "n", "", "Task name to update (partial match against task dir name)")
	_ = taskUpdateCmd.MarkFlagRequired("name")
	taskUpdateCmd.Flags().StringP("plan", "P", "", "Plan slug to narrow the search (substring match)")
	taskUpdateCmd.Flags().String("status", "", "New status (open, in_progress, done)")
	taskUpdateCmd.Flags().String("priority", "", "New priority (high, medium, low)")
	taskUpdateCmd.Flags().String("assignee", "", "New assignee")
}

func runTaskUpdate(planPartial, nameOrPartial, statusStr, priorityStr, assignee string) error {
	if statusStr == "" && priorityStr == "" && assignee == "" {
		return errors.New("provide at least one of --status, --priority, or --assignee")
	}

	if statusStr != "" && !task.IsValidStatus(task.Status(statusStr)) {
		return fmt.Errorf("invalid status %q: must be one of open, in_progress, done", statusStr)
	}
	if priorityStr != "" && !task.IsValidPriority(task.Priority(priorityStr)) {
		return fmt.Errorf("invalid priority %q: must be one of high, medium, low", priorityStr)
	}

	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	fields := make(map[string]string)
	if statusStr != "" {
		fields["status"] = statusStr
	}
	if priorityStr != "" {
		fields["priority"] = priorityStr
	}
	if assignee != "" {
		fields["assignee"] = assignee
	}

	if err := store.UpdateFields(planPartial, nameOrPartial, fields); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if statusStr != "" {
		fmt.Printf("✓ Updated task %q → status: %s\n", nameOrPartial, statusStr)
	} else {
		fmt.Printf("✓ Updated task %q.\n", nameOrPartial)
	}

	// When marking done, print the WALKTHROUGH.md path.
	if statusStr == string(task.StatusDone) {
		t, err := store.Get(planPartial, nameOrPartial)
		if err == nil {
			wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
			if _, statErr := os.Stat(wtPath); statErr == nil {
				rel, _ := relPath(root, wtPath)
				fmt.Printf("✓ WALKTHROUGH.md created: %s\n", rel)
				fmt.Println()
				fmt.Println("Next: fill in the walkthrough body, then run `logos distill --plan <plan>` when all tasks are done.")
			}
		}
	}

	return nil
}

// --- logos task delete -------------------------------------------------------

var taskDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a task directory",
	Long: `Delete a task directory from .logosyncx/tasks/. A confirmation prompt is
shown unless --force is passed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		planPartial, _ := cmd.Flags().GetString("plan")
		force, _ := cmd.Flags().GetBool("force")
		return runTaskDelete(planPartial, name, force)
	},
}

func init() {
	taskDeleteCmd.Flags().StringP("name", "n", "", "Task name to delete (partial match against task dir name)")
	_ = taskDeleteCmd.MarkFlagRequired("name")
	taskDeleteCmd.Flags().StringP("plan", "P", "", "Plan slug to narrow the search (substring match)")
	taskDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runTaskDelete(planPartial, nameOrPartial string, force bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	t, err := store.Get(planPartial, nameOrPartial)
	if err != nil {
		return err
	}

	if !force {
		fmt.Printf("Delete task %q (status: %s, dir: %s)? [y/N] ", t.Title, t.Status, t.DirPath)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	deleted, err := store.Delete(planPartial, nameOrPartial)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	fmt.Printf("✓ Deleted task %q.\n", deleted.Title)
	return nil
}

// --- logos task search -------------------------------------------------------

var taskSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Keyword search across task title, tags, and excerpt",
	Long: `Case-insensitive keyword search across the title, tags, and excerpt
(## What section) of every task. Optionally pre-filter by --plan, --status, or --tag.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword, _ := cmd.Flags().GetString("keyword")
		planPartial, _ := cmd.Flags().GetString("plan")
		statusStr, _ := cmd.Flags().GetString("status")
		tagStr, _ := cmd.Flags().GetString("tag")
		return runTaskSearch(keyword, planPartial, statusStr, tagStr)
	},
}

func init() {
	taskSearchCmd.Flags().StringP("keyword", "k", "", "Keyword to search for (case-insensitive, matches title, tags, and excerpt)")
	_ = taskSearchCmd.MarkFlagRequired("keyword")
	taskSearchCmd.Flags().StringP("plan", "P", "", "Pre-filter by plan slug before keyword match")
	taskSearchCmd.Flags().String("status", "", "Pre-filter by status before keyword match")
	taskSearchCmd.Flags().StringP("tag", "t", "", "Pre-filter by tag before keyword match")
}

func runTaskSearch(keyword, planPartial, statusStr, tagStr string) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	f := task.Filter{
		Plan:    planPartial,
		Status:  task.Status(statusStr),
		Keyword: keyword,
	}
	if tagStr != "" {
		f.Tags = []string{tagStr}
	}

	tasks, err := store.List(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	var jsonEntries []task.TaskJSON
	for _, t := range tasks {
		jsonEntries = append(jsonEntries, t.ToJSON())
	}
	return printTaskTable(jsonEntries)
}

// --- logos task walkthrough --------------------------------------------------

var taskWalkthroughCmd = &cobra.Command{
	Use:   "walkthrough",
	Short: "List or print walkthrough status for tasks in a plan",
	Long: `Without --name: list all tasks in the plan with their WALKTHROUGH.md fill status.
With --name: print the content of that task's WALKTHROUGH.md.

Fill status:
  [filled]        WALKTHROUGH.md has real content beyond headings and comments
  [scaffold only] WALKTHROUGH.md exists but contains only headings/comments
  -               No WALKTHROUGH.md exists yet`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		planPartial, _ := cmd.Flags().GetString("plan")
		namePartial, _ := cmd.Flags().GetString("name")
		return runTaskWalkthrough(planPartial, namePartial)
	},
}

func init() {
	taskWalkthroughCmd.Flags().StringP("plan", "P", "", "Plan slug to filter tasks (partial match, required)")
	_ = taskWalkthroughCmd.MarkFlagRequired("plan")
	taskWalkthroughCmd.Flags().StringP("name", "n", "", "Task name to print WALKTHROUGH.md for (partial match)")
}

func runTaskWalkthrough(planPartial, namePartial string) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	if namePartial != "" {
		// Print mode: output WALKTHROUGH.md content for the specific task.
		t, err := store.Get(planPartial, namePartial)
		if err != nil {
			return err
		}
		wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
		data, err := os.ReadFile(wtPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("no WALKTHROUGH.md found for task %q (run `logos task update --status done` first)", namePartial)
			}
			return fmt.Errorf("read WALKTHROUGH.md: %w", err)
		}
		fmt.Print(string(data))
		return nil
	}

	// List mode: show all tasks in the plan with walkthrough fill status.
	tasks, err := store.List(task.Filter{Plan: planPartial})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SEQ\tTITLE\tWALKTHROUGH")
	fmt.Fprintln(w, "---\t-----\t-----------")
	for _, t := range tasks {
		wtPath := filepath.Join(t.DirPath, "WALKTHROUGH.md")
		status := walkthroughFillStatus(wtPath)
		fmt.Fprintf(w, "%03d\t%s\t%s\n", t.Seq, t.Title, status)
	}
	return w.Flush()
}

// walkthroughFillStatus returns the fill status string for a WALKTHROUGH.md at path.
func walkthroughFillStatus(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "-"
	}

	inComment := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "<!--") {
			if !strings.Contains(trimmed, "-->") {
				inComment = true
			}
			continue
		}
		if inComment {
			if strings.Contains(trimmed, "-->") {
				inComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Real content line found.
		return "[filled]"
	}
	return "[scaffold only]"
}

// --- shared output helpers ---------------------------------------------------

// printTaskTable writes a human-readable tab-aligned task table to stdout.
func printTaskTable(entries []task.TaskJSON) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SEQ\tDATE\tTITLE\tSTATUS\tPRIORITY\tSTART\tPLAN")
	fmt.Fprintln(w, "---\t----\t-----\t------\t--------\t-----\t----")
	for _, e := range entries {
		date := e.Date.Format("2006-01-02")
		planName := e.Plan
		if planName == "" {
			planName = "-"
		}
		canStart := " "
		if e.CanStart {
			canStart = "✓"
		}
		fmt.Fprintf(w, "%03d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			e.Seq, date, e.Title, string(e.Status), string(e.Priority), canStart, planName)
	}
	return w.Flush()
}

// printTaskJSON writes a JSON array of TaskJSON objects to stdout.
func printTaskJSON(entries []task.TaskJSON) error {
	out := make([]task.TaskJSON, len(entries))
	for i, e := range entries {
		if e.Tags == nil {
			e.Tags = []string{}
		}
		if e.DependsOn == nil {
			e.DependsOn = []int{}
		}
		out[i] = e
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
