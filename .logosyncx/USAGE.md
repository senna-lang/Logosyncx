
# Logosyncx Usage for AI Agents

You have access to the `logos` CLI for managing session context.
`logos` is a shell binary — call it via terminal/shell commands.

---

## MANDATORY Triggers

The following are **not optional**. When any condition below occurs, you MUST run the corresponding command.

| Condition | You MUST run |
|---|---|
| Starting any work session | `logos ls --json` (always, before doing anything else) |
| User says "save this session", "log this", "記録して", "セッションを保存して" | `logos save --topic "..." --section "Summary=..." ...` |
| User says "make that a task", "add a TODO", "タスクにして", "issueにして" | `logos task create --title "..." --section "What=..." ...` |
| User says "continue from last time", "前回の続き", "show past context" | `logos ls --json` then `logos refer --name <name> --summary` |
| User mentions a past feature, decision, or discussion | `logos ls --json` and check excerpts for relevance |

> **Never say "I can't save sessions" or "I don't have access to logos."**
> The `logos` binary is installed and available via shell. Always use it.

### Syntax rules (apply to every command)

- **Always use flags** — never pass positional arguments.
- **Section content via `--section` only** — `--body`, `--body-stdin`, and `--description` do not exist.
- All section names must be defined in `.logosyncx/config.json`.

---

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
           --section "Summary=What happened in this session." \
           --section "Key Decisions=- Decision one"
```

For longer content, use a variable:
```
logos save --topic "short description" \
           --section "Summary=Implemented the auth flow. Chose JWT over sessions because of stateless requirements." \
           --section "Key Decisions=- JWT over sessions\n- RS256 algorithm"
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
logos refer --name <filename>            # full content
logos refer --name <partial-name>        # partial match
logos refer --name <filename> --summary  # key sections only (saves tokens, prefer this)
```

### Save a session
```
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
```

Allowed section names are defined in `.logosyncx/config.json` under `sessions.sections`.
Unknown section names are rejected with an error.

### Search (keyword narrowing)
```
logos search --keyword "keyword"              # search on topic, tags, and excerpt
logos search --keyword "auth" --tag security
```

## Sync index

If you manually edit, add, or delete session or task files, run:

```
logos sync
```

This rebuilds both `index.jsonl` and `task-index.jsonl` from the filesystem so that
`logos ls` and `logos task ls` return accurate results.

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
                  --priority high \
                  --tag go --tag cli \
                  --session <partial-session-name> \
                  --section "What=Add X so that Y." \
                  --section "Why=Required for the new auth flow."
```

Allowed section names are defined in `.logosyncx/config.json` under `tasks.sections`.
Unknown section names are rejected with an error.

> All fields are passed as flags — never use positional arguments.
> Section content must be provided via `--section "Name=content"`. There is no `--description` flag.

### Workflow for checking tasks

1. Run `logos task ls --status open --json` to get a list of outstanding tasks
2. Read `title` and `excerpt` to judge which tasks are relevant
3. Run `logos task refer --name <name> --with-session` to get full task details plus the linked session summary

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