# Logosyncx Usage for AI Agents

You have access to the `logos` CLI for managing session context.
Use it proactively to find relevant past decisions and designs.

## When to use

- When the user mentions a past discussion or asks about context: run `logos ls --json` and judge which sessions are relevant
- When starting work on a feature: check `logos ls --json` for related sessions
- When the user says "continue from last time": `logos refer` on the latest session
- When the user says "save this session" or similar: run `logos save --topic "..." --body-stdin`

## Workflow for finding relevant context

1. Run `logos ls --json` to get all sessions with excerpts
2. Read the `topic`, `tags`, and `excerpt` fields to judge relevance yourself
3. Run `logos refer <filename> --summary` on relevant sessions to get details
4. If you want to narrow down by keyword first, use `logos search <keyword>`

## Workflow for saving context

```
logos save --topic "short description of the session" \
           --tag go --tag cli \
           --agent claude-code \
           --body-stdin <<'EOF'
## Summary

What happened in this session.

## Key Decisions

- Decision one
EOF
```

Or inline for short sessions:
```
logos save --topic "quick fix" --body "## Summary\n\nFixed the bug."
```

## Commands

### List sessions
```
logos ls                    # human-readable table
logos ls --tag auth         # filter by tag
logos ls --since 2025-02-01 # filter by date
logos ls --json             # structured output with excerpts (preferred for agents)
```

### Read a session
```
logos refer <filename>            # full content
logos refer <partial-name>        # partial match
logos refer <filename> --summary  # key sections only (saves tokens, prefer this)
```

### Save a session
```
logos save --topic "..."                         # topic only, empty body
logos save --topic "..." --body "..."            # inline body
logos save --topic "..." --body-stdin            # body prose from stdin
logos save --topic "..." --tag go --tag cli \
           --agent claude-code \
           --related 2026-01-01_previous.md \
           --body-stdin
```

### Search (keyword narrowing)
```
logos search "keyword"         # search on topic, tags, and excerpt
logos search "auth" --tag security
```

## Token strategy
- Use `logos ls --json` first to scan all sessions cheaply via excerpts
- Use `--summary` on `refer` unless you need the full conversation log
- Only use full `refer` when the summary is insufficient

## Tasks

Action items, implementation proposals, and TODO items that arise during a session can be saved as tasks.
Tasks are always linked to a session — the session serves as the rationale for why the task exists.

### When to create a task

- When the user says "make that a task", "do that later", or "add a TODO"
- When you propose an implementation plan, improvement, or refactoring idea
- After saving a session, when you want to preserve a specific proposal for later

### Workflow for creating a task

```
logos task create --title "Implement the thing" \
                  --description "Add X so that Y." \
                  --priority high \
                  --tag go --tag cli \
                  --session <partial-session-name>
```

### Workflow for checking tasks

1. Run `logos task ls --status open --json` to get a list of outstanding tasks
2. Read `title` and `excerpt` to judge which tasks are relevant
3. Run `logos task refer <name> --with-session` to get full task details plus the linked session summary

### Commands

```
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
logos task create --title "..."                    # title only, medium priority
logos task create --title "..." --description "..." --priority high --tag <tag>
logos task create --title "..." --session <name>   # link to a session

# Update a task
logos task update <name> --status in_progress  # moves file to tasks/in_progress/
logos task update <name> --status done         # moves file to tasks/done/
logos task update <name> --priority high
logos task update <name> --assignee <name>

# Delete a single task
logos task delete <name>                  # prompts for confirmation
logos task delete <name> --force          # skip confirmation

# Bulk-delete all tasks with a given status
logos task purge --status done            # shows list + confirmation prompt
logos task purge --status done --force    # skip confirmation
logos task purge --status cancelled --force

# Search tasks
logos task search "keyword"               # search title, tags, and excerpt
logos task search "keyword" --status open
logos task search "keyword" --tag <tag>