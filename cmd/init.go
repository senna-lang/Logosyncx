// Package cmd implements the logos CLI commands using the cobra framework.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/spf13/cobra"
)

// usageMD is the content written to .logosyncx/USAGE.md on logos init.
const usageMD = `# Logosyncx Usage for AI Agents

You have access to the ` + "`logos`" + ` CLI for managing session context.
Use it proactively to find relevant past decisions and designs.

## When to use

- When the user mentions a past discussion or asks about context: run ` + "`logos ls --json`" + ` and judge which sessions are relevant
- When starting work on a feature: check ` + "`logos ls --json`" + ` for related sessions
- When the user says "continue from last time": ` + "`logos refer`" + ` on the latest session
- When the user says "save this session" or similar: run ` + "`logos save --topic \"...\" --section \"Summary=...\" --section \"Key Decisions=...\"`" + `

> **Always use flags** — never pass positional arguments to ` + "`logos`" + ` commands.
> **Section content via --section only** — ` + "`--body`" + ` and ` + "`--body-stdin`" + ` do not exist. All body content must be provided as ` + "`--section \"Name=content\"`" + ` flags where ` + "`Name`" + ` is defined in ` + "`.logosyncx/config.json`" + `.

## Workflow for finding relevant context

1. Run ` + "`logos ls --json`" + ` to get all sessions with excerpts
2. Read the ` + "`topic`" + `, ` + "`tags`" + `, and ` + "`excerpt`" + ` fields to judge relevance yourself
3. Run ` + "`logos refer --name <filename> --summary`" + ` on relevant sessions to get details
4. If you want to narrow down by keyword first, use ` + "`logos search --keyword <keyword>`" + `

## Workflow for saving context

` + "```" + `
logos save --topic "short description of the session" \
           --tag go --tag cli \
           --agent claude-code \
           --section "Summary=What happened in this session." \
           --section "Key Decisions=- Decision one"
` + "```" + `

For longer content, use a variable:
` + "```" + `
logos save --topic "short description" \
           --section "Summary=Implemented the auth flow. Chose JWT over sessions because of stateless requirements." \
           --section "Key Decisions=- JWT over sessions\n- RS256 algorithm"
` + "```" + `

## Commands

### List sessions
` + "```" + `
logos ls                    # human-readable table
logos ls --tag auth         # filter by tag
logos ls --since 2025-02-01 # filter by date
logos ls --json             # structured output with excerpts (preferred for agents)
` + "```" + `

### Read a session
` + "```" + `
logos refer --name <filename>            # full content
logos refer --name <partial-name>        # partial match
logos refer --name <filename> --summary  # key sections only (saves tokens, prefer this)
` + "```" + `

### Save a session
` + "```" + `
# topic only, no body sections
logos save --topic "..."

# with section content (--section is the only way to add body content)
logos save --topic "..." --section "Summary=text"
logos save --topic "..." \
           --tag go --tag cli \
           --agent claude-code \
           --related 2026-01-01_previous.md \
           --section "Summary=What happened." \
           --section "Key Decisions=- Decision A"
` + "```" + `

Allowed section names are defined in ` + "`.logosyncx/config.json`" + ` under ` + "`sessions.sections`" + `.
Unknown section names are rejected with an error.

### Search (keyword narrowing)
` + "```" + `
logos search --keyword "keyword"              # search on topic, tags, and excerpt
logos search --keyword "auth" --tag security
` + "```" + `

## Sync index

If you manually edit, add, or delete session or task files, run:

` + "```" + `
logos sync
` + "```" + `

This rebuilds both ` + "`index.jsonl`" + ` and ` + "`task-index.jsonl`" + ` from the filesystem so that
` + "`logos ls`" + ` and ` + "`logos task ls`" + ` return accurate results.

## Token strategy
- Use ` + "`logos ls --json`" + ` first to scan all sessions cheaply via excerpts
- Use ` + "`--summary`" + ` on ` + "`refer`" + ` unless you need the full conversation log
- Only use full ` + "`refer`" + ` when the summary is insufficient

## Tasks

Action items, implementation proposals, and TODO items that arise during a session can be saved as tasks.
Tasks are always linked to a session — the session serves as the rationale for why the task exists.

### When to create a task

- When the user says "make that a task", "do that later", or "add a TODO"
- When you propose an implementation plan, improvement, or refactoring idea
- After saving a session, when you want to preserve a specific proposal for later

### Workflow for creating a task

` + "```" + `
logos task create --title "Implement the thing" \
                  --priority high \
                  --tag go --tag cli \
                  --session <partial-session-name> \
                  --section "What=Add X so that Y." \
                  --section "Why=Required for the new auth flow."
` + "```" + `

Allowed section names are defined in ` + "`.logosyncx/config.json`" + ` under ` + "`tasks.sections`" + `.
Unknown section names are rejected with an error.

> All fields are passed as flags — never use positional arguments.
> Section content must be provided via ` + "`--section \"Name=content\"`" + `. There is no ` + "`--description`" + ` flag.

### Workflow for checking tasks

1. Run ` + "`logos task ls --status open --json`" + ` to get a list of outstanding tasks
2. Read ` + "`title`" + ` and ` + "`excerpt`" + ` to judge which tasks are relevant
3. Run ` + "`logos task refer --name <name> --with-session`" + ` to get full task details plus the linked session summary

### Commands

` + "```" + `
# List tasks
logos task ls                              # human-readable table
logos task ls --status open               # filter by status (open, in_progress, done, cancelled)
logos task ls --session <name>            # filter by linked session
logos task ls --priority high             # filter by priority (high, medium, low)
logos task ls --tag <tag>                 # filter by tag
logos task ls --json                      # structured output with excerpts (preferred for agents)

# Read a task
logos task refer --name <name>                   # full content
logos task refer --name <name> --summary         # key sections only (saves tokens)
logos task refer --name <name> --with-session    # append linked session summary

# Create a task
logos task create --title "..."                                        # title only, empty body
logos task create --title "..." --section "What=..." --priority high --tag <tag>
logos task create --title "..." --session <name>                       # link to a session
logos task create --title "..." \
                  --section "What=Implement X." \
                  --section "Why=Needed for Y." \
                  --section "Checklist=- [ ] step one\n- [ ] step two"

# Update a task
logos task update --name <name> --status in_progress  # moves file to tasks/in_progress/
logos task update --name <name> --status done         # moves file to tasks/done/
logos task update --name <name> --priority high
logos task update --name <name> --assignee <assignee>

# Delete a single task
logos task delete --name <name>           # prompts for confirmation
logos task delete --name <name> --force   # skip confirmation

# Bulk-delete all tasks with a given status
logos task purge --status done            # shows list + confirmation prompt
logos task purge --status done --force    # skip confirmation
logos task purge --status cancelled --force

# Search tasks
logos task search --keyword "keyword"                    # search title, tags, and excerpt
logos task search --keyword "keyword" --status open
logos task search --keyword "keyword" --tag <tag>
` + "```" + `
`

// agentsLine is appended to AGENTS.md (or CLAUDE.md) by logos init.
const agentsLine = "\n## Logosyncx\n\nUse `logos` CLI for session context management.\nFull reference: .logosyncx/USAGE.md\n"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Logosyncx in the current directory",
	Long: `Create .logosyncx/ with config.json and USAGE.md.
Append a reference line to AGENTS.md (or CLAUDE.md if present).
The session and task body structure is configured in config.json under
"save.sections" and "tasks.sections" respectively.
Exits with an error if the project has already been initialized.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	logosyncxDir := filepath.Join(cwd, config.DirName)

	// Guard: already initialized.
	if _, err := os.Stat(logosyncxDir); err == nil {
		return errors.New("already initialized: .logosyncx/ already exists")
	}

	// 1. Create directory structure.
	sessionsDir := filepath.Join(logosyncxDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return fmt.Errorf("create sessions directory: %w", err)
	}

	tasksDir := filepath.Join(logosyncxDir, "tasks")
	for _, statusDir := range []string{"open", "in_progress", "done", "cancelled"} {
		if err := os.MkdirAll(filepath.Join(tasksDir, statusDir), 0o755); err != nil {
			return fmt.Errorf("create tasks/%s directory: %w", statusDir, err)
		}
	}

	// 2. Write config.json with defaults.
	projectName := filepath.Base(cwd)
	cfg := config.Default(projectName)

	// Detect which agents file to use and record it in config.
	agentsFile := detectAgentsFile(cwd)
	cfg.AgentsFile = agentsFile

	if err := config.Save(cwd, cfg); err != nil {
		return fmt.Errorf("write config.json: %w", err)
	}

	// 3. Write USAGE.md.
	usagePath := filepath.Join(logosyncxDir, "USAGE.md")
	if err := os.WriteFile(usagePath, []byte(usageMD), 0o644); err != nil {
		return fmt.Errorf("write USAGE.md: %w", err)
	}

	// 4. Append reference line to agents file.
	agentsPath := filepath.Join(cwd, agentsFile)
	if err := appendAgentsLine(agentsPath); err != nil {
		return fmt.Errorf("update %s: %w", agentsFile, err)
	}

	fmt.Printf("✓ Initialized Logosyncx in %s\n", cwd)
	fmt.Printf("  Created  .logosyncx/\n")
	fmt.Printf("  Created  .logosyncx/config.json\n")
	fmt.Printf("  Created  .logosyncx/USAGE.md\n")
	fmt.Printf("  Created  .logosyncx/sessions/\n")
	fmt.Printf("  Created  .logosyncx/tasks/{open,in_progress,done,cancelled}/\n")
	fmt.Printf("  Updated  %s\n", agentsFile)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Commit .logosyncx/ to git\n")
	fmt.Printf("  2. Run `logos save --file <path>` to save your first session\n")

	return nil
}

// detectAgentsFile returns "CLAUDE.md" if it exists in the project root,
// otherwise falls back to "AGENTS.md" (creating it if needed).
func detectAgentsFile(projectRoot string) string {
	claudePath := filepath.Join(projectRoot, "CLAUDE.md")
	if _, err := os.Stat(claudePath); err == nil {
		return "CLAUDE.md"
	}
	return "AGENTS.md"
}

// appendAgentsLine appends the logosyncx reference block to the agents file.
// The file is created if it does not exist.
// If the file already contains "logosyncx/USAGE.md", the append is skipped.
func appendAgentsLine(path string) error {
	// Read existing content (tolerate file not existing).
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Idempotency guard: skip if already referenced.
	if strings.Contains(string(existing), "logosyncx/USAGE.md") {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(agentsLine)
	return err
}
