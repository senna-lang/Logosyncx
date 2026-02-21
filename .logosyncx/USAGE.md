# Logosyncx Usage for AI Agents

You have access to the `logos` CLI for managing session context.
Use it proactively to find relevant past decisions and designs.

## When to use

- When the user mentions a past discussion or asks about context: run `logos ls --json` and judge which sessions are relevant
- When starting work on a feature: check `logos ls --json` for related sessions
- When the user says "continue from last time": `logos refer` on the latest session
- When the user says "save this session" or similar: generate a session file from the template and run `logos save`

## Workflow for finding relevant context

1. Run `logos ls --json` to get all sessions with excerpts
2. Read the `topic`, `tags`, and `excerpt` fields to judge relevance yourself
3. Run `logos refer <filename> --summary` on relevant sessions to get details
4. If you want to narrow down by keyword first, use `logos search <keyword>`

## Workflow for saving context

1. Read `.logosyncx/template.md` to understand the structure
2. Fill in each section based on the conversation history
3. Run `logos save --stdin` or `logos save --file <path>` to persist it

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
logos save --file <path>    # save from a generated md file
logos save --stdin          # save from stdin (pipe)
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

1. Confirm the current session is already saved (`logos ls --json` to get the latest filename)
2. Read `.logosyncx/task-template.md` to understand the structure
3. Fill in `What`, `Why`, `Scope`, and `Checklist` from the conversation
4. Run `logos task create --session <session-name> --stdin` to save

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
```
