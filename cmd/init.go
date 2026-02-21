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
- When the user says "save this session" or similar: generate a session file from the template and run ` + "`logos save`" + `

## Workflow for finding relevant context

1. Run ` + "`logos ls --json`" + ` to get all sessions with excerpts
2. Read the ` + "`topic`" + `, ` + "`tags`" + `, and ` + "`excerpt`" + ` fields to judge relevance yourself
3. Run ` + "`logos refer <filename> --summary`" + ` on relevant sessions to get details
4. If you want to narrow down by keyword first, use ` + "`logos search <keyword>`" + `

## Workflow for saving context

1. Read ` + "`.logosyncx/template.md`" + ` to understand the structure
2. Fill in each section based on the conversation history
3. Run ` + "`logos save --stdin`" + ` or ` + "`logos save --file <path>`" + ` to persist it

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
logos refer <filename>            # full content
logos refer <partial-name>        # partial match
logos refer <filename> --summary  # key sections only (saves tokens, prefer this)
` + "```" + `

### Save a session
` + "```" + `
logos save --file <path>    # save from a generated md file
logos save --stdin          # save from stdin (pipe)
` + "```" + `

### Search (keyword narrowing)
` + "```" + `
logos search "keyword"         # search on topic, tags, and excerpt
logos search "auth" --tag security
` + "```" + `

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

1. Confirm the current session is already saved (` + "`logos ls --json`" + ` to get the latest filename)
2. Read ` + "`.logosyncx/task-template.md`" + ` to understand the structure
3. Fill in ` + "`What`" + `, ` + "`Why`" + `, ` + "`Scope`" + `, and ` + "`Checklist`" + ` from the conversation
4. Run ` + "`logos task create --session <session-name> --stdin`" + ` to save

### Workflow for checking tasks

1. Run ` + "`logos task ls --status open --json`" + ` to get a list of outstanding tasks
2. Read ` + "`title`" + ` and ` + "`excerpt`" + ` to judge which tasks are relevant
3. Run ` + "`logos task refer <name> --with-session`" + ` to get full task details plus the linked session summary

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
logos task refer <name>                   # full content
logos task refer <name> --summary         # key sections only (saves tokens)
logos task refer <name> --with-session    # append linked session summary

# Create a task
logos task create --file <path>           # from a generated md file
logos task create --stdin                 # from stdin (pipe)
logos task create --session <name> --stdin  # link to a session while creating

# Update a task
logos task update <name> --status in_progress
logos task update <name> --status done    # marks done and deletes the file (prompts for confirmation)
logos task update <name> --status done --force  # skip confirmation
logos task update <name> --priority high
logos task update <name> --assignee <name>

# Delete a task
logos task delete <name>                  # prompts for confirmation
logos task delete <name> --force          # skip confirmation

# Search tasks
logos task search "keyword"               # search title, tags, and excerpt
logos task search "keyword" --status open
logos task search "keyword" --tag <tag>
` + "```" + `
`

// templateMD is the default session template written to .logosyncx/template.md.
const templateMD = `---
id: {{id}}
date: {{date}}
topic: {{topic}}
tags: []
agent:
related: []
---

## Summary
<!-- Briefly describe what was discussed and decided in this session -->

## Key Decisions
<!-- List important decisions as bullet points -->
-

## Context Used
<!-- Past sessions or external resources referenced -->

## Notes
<!-- Other notes and supplementary information -->

## Raw Conversation
<!-- Paste the conversation log here (optional) -->
`

// taskTemplateMD is the default task template written to .logosyncx/task-template.md.
const taskTemplateMD = `---
id: {{id}}
date: {{date}}
title: {{title}}
status: open
priority: medium
session: {{session}}
tags: []
assignee:
---

## What

## Why

## Scope

## Checklist

- [ ]

## Notes
`

// agentsLine is appended to AGENTS.md (or CLAUDE.md) by logos init.
const agentsLine = "\n## Logosyncx\n\nUse `logos` CLI for session context management.\nFull reference: .logosyncx/USAGE.md\n"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Logosyncx in the current directory",
	Long: `Create .logosyncx/ with config.json, USAGE.md, and template.md.
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

	// 1. Create directory structure.
	sessionsDir := filepath.Join(logosyncxDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return fmt.Errorf("create sessions directory: %w", err)
	}

	tasksDir := filepath.Join(logosyncxDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		return fmt.Errorf("create tasks directory: %w", err)
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

	// 4. Write template.md.
	templatePath := filepath.Join(logosyncxDir, "template.md")
	if err := os.WriteFile(templatePath, []byte(templateMD), 0o644); err != nil {
		return fmt.Errorf("write template.md: %w", err)
	}

	// 5. Write task-template.md.
	taskTemplatePath := filepath.Join(logosyncxDir, "task-template.md")
	if err := os.WriteFile(taskTemplatePath, []byte(taskTemplateMD), 0o644); err != nil {
		return fmt.Errorf("write task-template.md: %w", err)
	}

	// 6. Append reference line to agents file.
	agentsPath := filepath.Join(cwd, agentsFile)
	if err := appendAgentsLine(agentsPath); err != nil {
		return fmt.Errorf("update %s: %w", agentsFile, err)
	}

	fmt.Printf("✓ Initialized Logosyncx in %s\n", cwd)
	fmt.Printf("  Created  .logosyncx/\n")
	fmt.Printf("  Created  .logosyncx/config.json\n")
	fmt.Printf("  Created  .logosyncx/USAGE.md\n")
	fmt.Printf("  Created  .logosyncx/template.md\n")
	fmt.Printf("  Created  .logosyncx/task-template.md\n")
	fmt.Printf("  Created  .logosyncx/sessions/\n")
	fmt.Printf("  Created  .logosyncx/tasks/\n")
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
