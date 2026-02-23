---
id: t-bds34x
date: 2026-02-21T22:03:24.837052+09:00
title: Update USAGE.md template — add tasks section for agent reference
status: in_progress
priority: medium
session: ""
tags:
    - phase2
    - task
    - docs
assignee: ""
---

## What

Add the tasks section to the USAGE.md template that `logos init` generates.

The section to append under `## Commands`:

```
## Tasks

Action items, implementation proposals, and TODO items that arise during a session can be saved as tasks.
Tasks are always linked to a session — the session serves as the rationale for why the task exists.

### When to create a task
- When the user says "make that a task", "do that later", or "add a TODO"
- When you propose an implementation plan, improvement, or refactoring idea
- After saving a session, when you want to preserve a specific proposal for later

### Workflow for creating a task
1. Confirm the current session is already saved (logos ls --json to get the latest filename)
2. Read .logosyncx/task-template.md to understand the structure
3. Fill in What / Why / Scope / Checklist from the conversation
4. Run logos task create --session <session-filename> --stdin to save

### Workflow for checking tasks
1. Run logos task ls --status open --json to get a list of outstanding tasks
2. Read title and excerpt to judge which tasks are relevant
3. Run logos task refer <task> --with-session to get full task details plus the linked session summary

### Commands
logos task ls --json
logos task ls --session <name>
logos task ls --status open
logos task refer <name>
logos task refer <name> --with-session
logos task create --session <name> --stdin
logos task create --session <name> --file <path>
logos task update <name> --status done
logos task update <name> --status in_progress
logos task update <name> --status done --force
logos task delete <name>
logos task search <keyword>
```

## Why

The USAGE.md template embedded in `cmd/init.go` was written before the task subsystem existed.
Without a tasks section, agents initialized in new projects have no guidance on how to create or
manage tasks. Adding the section ensures every `logos init` run produces an agent-ready USAGE.md
that covers the full feature set.

## Scope

- The USAGE.md template string embedded in `cmd/init.go` (the `usageTemplate` const or equivalent)
- Verify the generated content matches the `.logosyncx/USAGE.md` that already exists in this repo

## Checklist

- [ ] Locate the USAGE.md template in `cmd/init.go`
- [ ] Append the tasks section to the template (after the existing `## Commands` block)
- [ ] Run `logos init` in a temp directory and confirm the generated USAGE.md includes the tasks section
- [ ] Run `go test ./cmd/...` and confirm all tests pass

## Notes

Migrated from beads issue `logosyncx-34x`.
The current `.logosyncx/USAGE.md` in this repo already contains the correct tasks section — use it
as the reference for the exact wording to embed in `cmd/init.go`.