package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/session"
	"github.com/spf13/cobra"
)

// --- root task command -------------------------------------------------------

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks in .logosyncx/tasks/",
	Long: `Create, list, update, and delete task files stored under
.logosyncx/tasks/. Tasks can be linked to saved sessions and are tracked
in git alongside session context.`,
}

func init() {
	taskCmd.AddCommand(
		taskCreateCmd,
		taskLsCmd,
		taskReferCmd,
		taskUpdateCmd,
		taskDeleteCmd,
		taskPurgeCmd,
		taskSearchCmd,
	)
	rootCmd.AddCommand(taskCmd)
}

// --- logos task create -------------------------------------------------------

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task file",
	Long: `Create a task in .logosyncx/tasks/ using flag-based input.

  logos task create --title "..." [--description "..."] [--priority high|medium|low] \
                    [--tag <tag>] [--session <partial>]

auto-fills id/date. Optionally link to an existing session with --session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionPartial, _ := cmd.Flags().GetString("session")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		priority, _ := cmd.Flags().GetString("priority")
		tags, _ := cmd.Flags().GetStringArray("tag")
		return runTaskCreate(sessionPartial, title, description, priority, tags)
	},
}

func init() {
	taskCreateCmd.Flags().StringP("session", "s", "", "Partial name of the session to link (resolved by partial match)")
	taskCreateCmd.Flags().StringP("title", "T", "", "Task title (required)")
	taskCreateCmd.Flags().StringP("description", "d", "", "Task description (sets the ## What section body)")
	taskCreateCmd.Flags().StringP("priority", "p", "medium", "Task priority (high|medium|low)")
	taskCreateCmd.Flags().StringArray("tag", []string{}, "Tag to attach (repeatable: --tag go --tag cli)")
}

func runTaskCreate(sessionPartial, title, description, priority string, tags []string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("provide --title <title>")
	}

	// Validate priority.
	p := task.Priority(priority)
	if priority != "" && !task.IsValidPriority(p) {
		return fmt.Errorf("invalid priority %q: must be one of high, medium, low", priority)
	}

	// Build body markdown from description.
	var body string
	if description != "" {
		body = "## What\n\n" + description + "\n"
	} else {
		body = "## What\n\n"
	}

	t := task.Task{
		Title:    title,
		Priority: p,
		Tags:     tags,
		Body:     body,
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

	// Resolve and overwrite session field if --session was supplied.
	if sessionPartial != "" {
		resolved, err := store.ResolveSession(sessionPartial)
		if err != nil {
			return fmt.Errorf("resolve session %q: %w", sessionPartial, err)
		}
		t.Session = resolved
	}

	// Warn but don't block on empty title.
	if strings.TrimSpace(t.Title) == "" {
		fmt.Fprintln(os.Stderr, "warning: frontmatter 'title' is empty — filename will use 'untitled'")
	}

	savedPath, err := store.Save(&t, t.Body)
	if err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	fmt.Printf("✓ Created task: %s\n", savedPath)
	fmt.Println()
	fmt.Println("Next: commit and push to share with your team.")
	return nil
}

// --- logos task ls -----------------------------------------------------------

var taskLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List tasks",
	Long: `Display a table of tasks in .logosyncx/tasks/, sorted newest first.
Use --json for structured output suitable for agent consumption.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionPartial, _ := cmd.Flags().GetString("session")
		statusStr, _ := cmd.Flags().GetString("status")
		priorityStr, _ := cmd.Flags().GetString("priority")
		tagStr, _ := cmd.Flags().GetString("tag")
		asJSON, _ := cmd.Flags().GetBool("json")
		return runTaskLS(sessionPartial, statusStr, priorityStr, tagStr, asJSON)
	},
}

func init() {
	taskLsCmd.Flags().StringP("session", "s", "", "Filter by session (substring match)")
	taskLsCmd.Flags().String("status", "", "Filter by status (open, in_progress, done, cancelled)")
	taskLsCmd.Flags().String("priority", "", "Filter by priority (high, medium, low)")
	taskLsCmd.Flags().StringP("tag", "t", "", "Filter by tag (exact match)")
	taskLsCmd.Flags().Bool("json", false, "Output structured JSON (for agent consumption)")
}

func runTaskLS(sessionPartial, statusStr, priorityStr, tagStr string, asJSON bool) error {
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
			// Auto-rebuild: inform the user and build the index on the fly.
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
		Session:  sessionPartial,
		Status:   task.Status(statusStr),
		Priority: task.Priority(priorityStr),
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
	Use:   "refer <name>",
	Short: "Print the content of a task file",
	Long: `Print a task file to stdout. Use --summary to print only the sections
listed in config.tasks.summary_sections (saves tokens). Use --with-session to
also append the summary of the linked session.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		summary, _ := cmd.Flags().GetBool("summary")
		withSession, _ := cmd.Flags().GetBool("with-session")
		return runTaskRefer(args[0], summary, withSession)
	},
}

func init() {
	taskReferCmd.Flags().Bool("summary", false, "Print only summary sections (saves tokens)")
	taskReferCmd.Flags().Bool("with-session", false, "Append the linked session summary")
}

func runTaskRefer(nameOrPartial string, summary, withSession bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	t, err := store.Get(nameOrPartial)
	if err != nil {
		return err
	}

	if summary {
		sections := task.ExtractSections(t.Body, cfg.Tasks.SummarySections)
		fmt.Println(sections)
	} else {
		// Print frontmatter + body.
		data, err := task.Marshal(*t)
		if err != nil {
			return fmt.Errorf("marshal task: %w", err)
		}
		fmt.Print(string(data))
	}

	if withSession && t.Session != "" {
		fmt.Println()
		fmt.Printf("--- linked session: %s ---\n", t.Session)

		sessPath := fmt.Sprintf("%s/.logosyncx/sessions/%s", root, t.Session)
		s, err := session.LoadFile(sessPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load linked session %q: %v\n", t.Session, err)
		} else {
			sections := session.ExtractSections(s.Body, cfg.Save.SummarySections)
			fmt.Println(sections)
		}
	}

	return nil
}

// --- logos task update -------------------------------------------------------

var taskUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update task fields",
	Long: `Update frontmatter fields of a task. Supported flags: --status,
--priority, --assignee. Setting --status done removes the task file after
confirmation (use --force to skip the prompt).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		statusStr, _ := cmd.Flags().GetString("status")
		priorityStr, _ := cmd.Flags().GetString("priority")
		assignee, _ := cmd.Flags().GetString("assignee")
		force, _ := cmd.Flags().GetBool("force")
		return runTaskUpdate(args[0], statusStr, priorityStr, assignee, force)
	},
}

func init() {
	taskUpdateCmd.Flags().String("status", "", "New status (open, in_progress, done, cancelled)")
	taskUpdateCmd.Flags().String("priority", "", "New priority (high, medium, low)")
	taskUpdateCmd.Flags().String("assignee", "", "New assignee")
	taskUpdateCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runTaskUpdate(nameOrPartial, statusStr, priorityStr, assignee string, force bool) error {
	if statusStr == "" && priorityStr == "" && assignee == "" {
		return errors.New("provide at least one of --status, --priority, or --assignee")
	}

	if statusStr != "" && !task.IsValidStatus(task.Status(statusStr)) {
		return fmt.Errorf("invalid status %q: must be one of open, in_progress, done, cancelled", statusStr)
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

	if err := store.UpdateFields(nameOrPartial, fields); err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if statusStr != "" {
		fmt.Printf("✓ Updated task %q → status: %s\n", nameOrPartial, statusStr)
		if task.Status(statusStr) == task.StatusDone {
			fmt.Println("  Tip: run `logos task purge --status done --force` to delete all done tasks.")
		}
	} else {
		fmt.Printf("✓ Updated task %q.\n", nameOrPartial)
	}
	return nil
}

// --- logos task purge --------------------------------------------------------

var taskPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Delete all tasks with a given status",
	Long: `List all tasks matching --status, show a confirmation prompt, then
delete them all at once. Use --force to skip the confirmation prompt.

Unlike 'logos task delete' (which targets a single task by name), purge
operates on every task in a status bucket — useful for cleaning up done or
cancelled tasks in bulk.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		statusStr, _ := cmd.Flags().GetString("status")
		force, _ := cmd.Flags().GetBool("force")
		return runTaskPurge(statusStr, force)
	},
}

func init() {
	taskPurgeCmd.Flags().String("status", "", "Status bucket to purge (open, in_progress, done, cancelled)")
	taskPurgeCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	_ = taskPurgeCmd.MarkFlagRequired("status")
}

func runTaskPurge(statusStr string, force bool) error {
	if !task.IsValidStatus(task.Status(statusStr)) {
		return fmt.Errorf("invalid status %q: must be one of open, in_progress, done, cancelled", statusStr)
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

	// List tasks to be deleted so the user can review them.
	tasks, err := store.List(task.Filter{Status: task.Status(statusStr)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	if len(tasks) == 0 {
		fmt.Printf("No %s tasks to purge.\n", statusStr)
		return nil
	}

	fmt.Printf("The following %d task(s) with status %q will be permanently deleted:\n\n", len(tasks), statusStr)
	for _, t := range tasks {
		fmt.Printf("  • %s  %s\n", t.Date.Format("2006-01-02"), t.Title)
	}
	fmt.Println()

	if !force {
		fmt.Printf("Delete all %d task(s)? [y/N] ", len(tasks))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	n, err := store.Purge(task.Status(statusStr))
	if err != nil {
		return fmt.Errorf("purge: %w", err)
	}
	fmt.Printf("✓ Deleted %d task(s) with status %q.\n", n, statusStr)
	return nil
}

// --- logos task delete -------------------------------------------------------

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a task file",
	Long: `Delete a task file from .logosyncx/tasks/. A confirmation prompt is
shown unless --force is passed.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return runTaskDelete(args[0], force)
	},
}

func init() {
	taskDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

func runTaskDelete(nameOrPartial string, force bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	store := task.NewStore(root, &cfg)

	t, err := store.Get(nameOrPartial)
	if err != nil {
		return err
	}

	if !force {
		fmt.Printf("Delete task %q (status: %s)? [y/N] ", t.Title, t.Status)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := store.Delete(t.Filename); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	fmt.Printf("✓ Deleted task %q.\n", t.Title)
	return nil
}

// --- logos task search -------------------------------------------------------

var taskSearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Keyword search across task title, tags, and excerpt",
	Long: `Case-insensitive keyword search across the title, tags, and excerpt
(## What section) of every task. Optionally pre-filter by --status or --tag.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		statusStr, _ := cmd.Flags().GetString("status")
		tagStr, _ := cmd.Flags().GetString("tag")
		return runTaskSearch(args[0], statusStr, tagStr)
	},
}

func init() {
	taskSearchCmd.Flags().String("status", "", "Pre-filter by status before keyword match")
	taskSearchCmd.Flags().StringP("tag", "t", "", "Pre-filter by tag before keyword match")
}

func runTaskSearch(keyword, statusStr, tagStr string) error {
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

	// Convert []*task.Task to []task.TaskJSON for the shared display helpers.
	var jsonEntries []task.TaskJSON
	for _, t := range tasks {
		jsonEntries = append(jsonEntries, t.ToJSON())
	}
	return printTaskTable(jsonEntries)
}

// --- shared output helpers ---------------------------------------------------

// printTaskTable writes a human-readable tab-aligned task table to stdout.
func printTaskTable(entries []task.TaskJSON) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tTITLE\tSTATUS\tPRIORITY\tSESSION")
	fmt.Fprintln(w, "----\t-----\t------\t--------\t-------")
	for _, e := range entries {
		date := e.Date.Format("2006-01-02 15:04")
		sess := e.Session
		if sess == "" {
			sess = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			date, e.Title, string(e.Status), string(e.Priority), sess)
	}
	return w.Flush()
}

// printTaskJSON writes a JSON array of TaskJSON objects to stdout.
func printTaskJSON(entries []task.TaskJSON) error {
	// Normalise nil slices so JSON output always uses [] rather than null.
	out := make([]task.TaskJSON, len(entries))
	for i, e := range entries {
		if e.Tags == nil {
			e.Tags = []string{}
		}
		out[i] = e
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
