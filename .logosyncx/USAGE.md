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
