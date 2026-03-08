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

You have access to the ` + "`logos`" + ` CLI for managing plan and task context.
` + "`logos`" + ` is a shell binary — call it via terminal/shell commands.

---

## MANDATORY Triggers

| Condition | You MUST run |
|---|---|
| Starting any work session | ` + "`logos ls --json`" + ` (check past plans before doing anything) |
| User says "save this plan", "log this", "記録して" | ` + "`logos save --topic \"...\"`" + ` then write body with Write tool |
| User says "make that a task", "add a TODO", "タスクにして" | ` + "`logos task create --plan <name> --title \"...\"`" + ` then write body with Write tool |
| User says "continue from last time", "前回の続き" | ` + "`logos ls --json`" + ` then ` + "`logos refer --name <name> --summary`" + ` |
| User mentions a past plan or decision | ` + "`logos ls --json`" + ` and check excerpts for relevance |

> **Never say "I can't save plans" or "I don't have access to logos."**
> The ` + "`logos`" + ` binary is installed and available via shell. Always use it.

### Agent contract for writing document bodies

Before writing any document body (plan, task, knowledge), you MUST:

1. Read the corresponding template from ` + "`.logosyncx/templates/`" + `.
2. Use the template's section structure as the basis for the document.
3. Write the body **directly into the file** using the Write tool — not via a CLI flag.

The CLI produces a frontmatter scaffold only. You fill the body.

---

## Workflow for finding relevant context

1. Run ` + "`logos ls --json`" + ` to get all plans with excerpts
2. Read ` + "`topic`" + `, ` + "`tags`" + `, and ` + "`excerpt`" + ` fields to judge relevance yourself
3. Run ` + "`logos refer --name <filename> --summary`" + ` on relevant plans to get details
4. If you want to narrow down by keyword first, use ` + "`logos search --keyword <keyword>`" + `

## Workflow for saving a plan

` + "```" + `
# 1. Scaffold the plan file
logos save --topic "short description" --tag go --tag cli --agent claude-code

# 2. Read the plan template
cat .logosyncx/templates/plan.md

# 3. Write the plan body directly into the file
# (use the Write tool to fill in sections)
` + "```" + `

## Commands

### List plans
` + "```" + `
logos ls                       # human-readable table
logos ls --tag auth            # filter by tag
logos ls --since 2026-01-01    # filter by date
logos ls --blocked             # show only blocked plans
logos ls --json                # structured output with excerpts (preferred for agents)
` + "```" + `

### Read a plan
` + "```" + `
logos refer --name <filename>            # full content
logos refer --name <partial-name>        # partial match
logos refer --name <filename> --summary  # key sections only (saves tokens, prefer this)
` + "```" + `

### Save a plan
` + "```" + `
logos save --topic "short description"
logos save --topic "..." --tag go --tag cli --agent claude-code --depends-on 20260304-auth.md
` + "```" + `

### Search (keyword narrowing)
` + "```" + `
logos search --keyword "keyword"
logos search --keyword "auth" --tag security
` + "```" + `

### Sync index
` + "```" + `
logos sync
` + "```" + `

Rebuilds the plan and task indexes from the filesystem.

### Garbage collect stale plans
` + "```" + `
logos gc --dry-run
logos gc
logos gc purge --force
` + "```" + `

---

## Tasks

Tasks are work items linked to a plan. Each task has a ` + "`TASK.md`" + ` (what to do)
and optionally a ` + "`WALKTHROUGH.md`" + ` (what actually happened — filled after completion).

### Workflow for creating a task

` + "```" + `
# 1. Scaffold the task
logos task create --plan <plan-filename> --title "Implement the thing" --priority high --tag go

# 2. Read the task template
cat .logosyncx/templates/task.md

# 3. Write TASK.md body directly using the Write tool
` + "```" + `

### Workflow for completing a task

` + "```" + `
# 1. Mark task done
logos task update --plan <plan-filename> --name <task-name> --status done

# 2. Read the walkthrough template
cat .logosyncx/templates/walkthrough.md

# 3. Write WALKTHROUGH.md body directly using the Write tool
logos task walkthrough --plan <plan-filename> --name <task-name>
` + "```" + `

### Task commands

` + "```" + `
# List tasks
logos task ls                                     # all tasks
logos task ls --plan <plan-filename>              # tasks for a specific plan
logos task ls --status open                       # filter by status
logos task ls --blocked                           # show only blocked tasks
logos task ls --json                              # structured output (preferred for agents)

# Read a task
logos task refer --name <name>                    # full TASK.md content
logos task refer --name <name> --summary          # key sections only (saves tokens)

# Create a task
logos task create --plan <plan-filename> --title "..."
logos task create --plan <plan-filename> --title "..." --priority high --tag go --depends-on 1

# Update a task
logos task update --plan <plan-filename> --name <name> --status in_progress
logos task update --plan <plan-filename> --name <name> --status done
logos task update --plan <plan-filename> --name <name> --priority high

# Open walkthrough scaffold
logos task walkthrough --plan <plan-filename> --name <name>
` + "```" + `

---

## Distill

After all tasks in a plan are done, distil the work into reusable knowledge:

` + "```" + `
# Preview — no writes
logos distill --plan <plan-filename> --dry-run

# Write knowledge scaffold
logos distill --plan <plan-filename>

# Read the knowledge template, then fill in the knowledge file using the Write tool
cat .logosyncx/templates/knowledge.md
` + "```" + `

---

## Token strategy
- Use ` + "`logos ls --json`" + ` first to scan all plans cheaply via excerpts
- Use ` + "`--summary`" + ` on ` + "`refer`" + ` unless you need the full plan body
- Only use full ` + "`refer`" + ` when the summary is insufficient
`

// agentsLine is appended to AGENTS.md (or CLAUDE.md) by logos init.
const agentsLine = "\n## Logosyncx\n\n" +
	"Use `logos` CLI for plan and task management.\n" +
	"Full reference: `.logosyncx/USAGE.md`\n\n" +
	"**MANDATORY triggers:**\n\n" +
	"- **Start of every session** → `logos ls --json` (check past plans before doing anything)\n" +
	"- User says \"save this plan\" / \"記録して\" → `logos save --topic \"...\"` then write body with Write tool\n" +
	"- User says \"make that a task\" / \"タスクにして\" → `logos task create --plan <name> --title \"...\"`\n" +
	"- User says \"continue from last time\" / \"前回の続き\" → `logos ls --json` then `logos refer --name <name> --summary`\n\n" +
	"Always read the template before writing any document body. Write bodies directly into the file using the Write tool.\n"

// defaultPlanTemplate is written to templates/plan.md on logos init.
const defaultPlanTemplate = `## Background

<!-- Why does this work exist? What problem are we solving?
     Dump everything — context, constraints, user needs.
     This section is used as the plan excerpt in ` + "`logos ls`" + `. -->

## Spec

<!-- Crystallise the specification through dialogue.
     What exactly will be built or changed? What are the boundaries? -->

## Key Decisions

<!-- Significant design decisions and their rationale.
     Format: "Decision: <what>. Rationale: <why>." -->

## Notes

<!-- Miscellaneous notes, links, risks, or open questions. -->
`

// defaultTaskTemplate is written to templates/task.md on logos init.
const defaultTaskTemplate = `## What

<!-- One paragraph describing what this task delivers.
     Be concrete. This is used as the task excerpt in ` + "`logos task ls`" + `. -->

## Why

<!-- Why does this task need to exist?
     Link back to the plan's Spec if helpful. -->

## Scope

<!-- Files and directories this task touches:
- ` + "`path/to/file.go`" + `
- ` + "`path/to/file_test.go`" + `

What is explicitly OUT of scope: -->

## Acceptance Criteria

<!-- This task is done when:
- [ ] <observable behaviour or output>
- [ ] <observable behaviour or output> -->

## Checklist

<!-- Step-by-step implementation checklist.
- [ ] Step one
- [ ] Step two
- [ ] Tests added (Red → Green → Refactor)
- [ ] ` + "`go test ./...`" + ` passes -->

## Notes

<!-- Supplementary notes, gotchas known upfront, references. -->
`

// defaultWalkthroughTemplate is written to templates/walkthrough.md on logos init.
const defaultWalkthroughTemplate = `## Key Specification

<!-- What spec, task description, or requirements drove this implementation?
     Link to TASK.md sections, design docs, or paste the key constraints. -->

## What Was Done

<!-- Describe what was actually implemented or resolved. -->

## How It Was Done

<!-- Key steps, approach taken, alternatives considered. -->

## Gotchas & Lessons Learned

<!-- Anything that tripped you up, surprising behaviour, edge cases. -->

## Reusable Patterns

<!-- Code snippets, patterns, or conventions worth reusing. -->
`

// defaultKnowledgeTemplate is written to templates/knowledge.md on logos init.
const defaultKnowledgeTemplate = `## Summary

<!-- Concise summary of what was learned across all tasks.
     This is used as the knowledge excerpt in future lookups. -->

## Implemented Features

<!-- What was actually built or changed in this session?
     For each feature/change, describe:
     - Feature: <name>
     - Spec: <what it does, key constraints, acceptance criteria>
     - Files changed: <relevant files> -->

## Key Specification

<!-- What spec, task description, or requirements drove this implementation?
     Link to TASK.md sections, design docs, or paste the key constraints. -->

## Key Learnings

<!-- Most important insights from the walkthroughs.
     Focus on what would change how you approach similar work next time. -->

## Reusable Patterns

<!-- Code snippets, architectural patterns, or conventions worth reusing.
     Make each pattern self-contained with enough context. -->

## Gotchas

<!-- Surprising behaviour, edge cases, or mistakes to avoid.
     Format: "Gotcha: <what>. Fix: <how to handle it>." -->

## Source Walkthroughs

<!-- Auto-populated by logos distill. Do not edit. -->
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Logosyncx in the current directory",
	Long: `Create .logosyncx/ with plans/, knowledge/, templates/, config.json and USAGE.md.
Append a reference line to AGENTS.md (or CLAUDE.md if present).
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

	// 1. Create v2 directory structure.
	for _, dir := range []string{
		filepath.Join(logosyncxDir, "plans", "archive"),
		filepath.Join(logosyncxDir, "knowledge"),
		filepath.Join(logosyncxDir, "templates"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// 2. Write default template files.
	templates := map[string]string{
		"plan.md":        defaultPlanTemplate,
		"task.md":        defaultTaskTemplate,
		"knowledge.md":   defaultKnowledgeTemplate,
		"walkthrough.md": defaultWalkthroughTemplate,
	}
	for name, content := range templates {
		path := filepath.Join(logosyncxDir, "templates", name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write templates/%s: %w", name, err)
		}
	}

	// 3. Write config.json with defaults.
	projectName := filepath.Base(cwd)
	cfg := config.Default(projectName)

	agentsFile := detectAgentsFile(cwd)
	cfg.AgentsFile = agentsFile

	if err := config.Save(cwd, cfg); err != nil {
		return fmt.Errorf("write config.json: %w", err)
	}

	// 4. Write USAGE.md.
	usagePath := filepath.Join(logosyncxDir, "USAGE.md")
	if err := os.WriteFile(usagePath, []byte(usageMD), 0o644); err != nil {
		return fmt.Errorf("write USAGE.md: %w", err)
	}

	// 5. Append reference line to agents file.
	agentsPath := filepath.Join(cwd, agentsFile)
	if err := appendAgentsLine(agentsPath); err != nil {
		return fmt.Errorf("update %s: %w", agentsFile, err)
	}

	fmt.Printf("✓ Initialized Logosyncx in %s\n", cwd)
	fmt.Printf("  Created  .logosyncx/\n")
	fmt.Printf("  Created  .logosyncx/plans/\n")
	fmt.Printf("  Created  .logosyncx/knowledge/\n")
	fmt.Printf("  Created  .logosyncx/templates/\n")
	fmt.Printf("  Created  .logosyncx/config.json\n")
	fmt.Printf("  Created  .logosyncx/USAGE.md\n")
	fmt.Printf("  Updated  %s\n", agentsFile)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  1. Commit .logosyncx/ to git\n")
	fmt.Printf("  2. Run `logos save --topic <topic>` to save your first plan\n")

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
