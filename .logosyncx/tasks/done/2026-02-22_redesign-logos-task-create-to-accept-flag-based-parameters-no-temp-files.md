---
id: t-86cf53
date: 2026-02-22T20:34:56+09:00
title: Redesign logos task create and logos save to accept flag-based parameters (no temp files)
status: done
priority: medium
session: 2026-02-22_logos-task-create-redesign-beads-comparison.md
tags:
    - cli
    - dx
    - task
    - save
    - design
assignee: ""
---

## What

Add flag-based creation to both `logos task create` and `logos save` so that
agents (and humans) can create a task or session with a single command —
no temporary markdown file required.

Proposed interfaces:

```
logos task create --title "..." [--description "..."] [--priority high|medium|low] \
                  [--tag <tag>] [--session <partial>]

logos save --topic "..." [--tag <tag>] [--agent <agent>] \
           [--related <session>] [--body "..."] [--body-stdin]
```

The existing `--file` / `--stdin` interface must be kept for backward
compatibility on both commands, but the flag-based path becomes the preferred
route.

## Why

Currently both `logos task create` and `logos save` require a full markdown
document with YAML frontmatter to be passed via `--file <path>` or `--stdin`.
This forces agents to:

1. Build a complete frontmatter + body string.
2. Either write it to a temporary file or pass it through a heredoc/process
   substitution.
3. Clean up the temporary file afterward.

This is unnecessary friction. `internal/task/store.Store.Save()` and
`session.Write()` already auto-fill `id`, `date`, `status`, and `priority`,
so the backends are ready — only the CLI surfaces are missing the flag-based
entry points.

beads (`bd create "title" --description "..." --priority 2 --labels "..."`)
shows the right pattern: structured fields as flags, no intermediate file.

For `logos save`, the body content (the actual session prose) can be provided
via `--body "..."` for short content, or `--body-stdin` to read only the body
from stdin without needing a full markdown document with frontmatter.

## Scope

### logos task create
- `cmd/task.go` — extend `taskCreateCmd` (and `runTaskCreate`) to accept
  `--title`, `--description`, `--priority`, `--tag` (repeatable), `--session`
  flags; when `--title` is present and neither `--file` nor `--stdin` is given,
  construct a `task.Task` directly from flags and call `store.Save()`
- `internal/task/task.go` — no changes expected (Task struct is already
  sufficient)
- `internal/task/store.go` — no changes expected

### logos save
- `cmd/save.go` — extend `saveCmd` (and `runSave`) to accept `--topic`,
  `--tag` (repeatable), `--agent`, `--related` (repeatable), `--body`,
  `--body-stdin` flags; when `--topic` is present and neither `--file` nor
  `--stdin` is given, construct a `session.Session` directly from flags and
  call `session.Write()`
- `pkg/session/session.go` — no changes expected (Session struct is already
  sufficient)

### Shared
- `.logosyncx/USAGE.md` — update both the "Save a session" and "Create a task"
  sections to document the new flag-based interfaces as the preferred methods
- Tests — add unit/integration tests for the flag-based paths in both commands

## Checklist

### logos task create
- [ ] Add `--title` flag to `taskCreateCmd`
- [ ] Add `--description` flag (sets the `## What` section body)
- [ ] Add `--priority` flag (high | medium | low)
- [ ] Add `--tag` flag (repeatable: `--tag go --tag tooling`)
- [ ] Validate that `--title` is mutually exclusive with `--file`/`--stdin`
- [ ] Construct `task.Task` from flags and call `store.Save()` directly
- [ ] Resolve `--session` partial name via `store.ResolveSession()` (already exists)
- [ ] Add tests: flag-based create with title only, with all flags, conflict errors

### logos save
- [ ] Add `--topic` flag to `saveCmd`
- [ ] Add `--tag` flag (repeatable: `--tag go --tag cli`)
- [ ] Add `--agent` flag (optional, defaults to empty)
- [ ] Add `--related` flag (repeatable: `--related session1 --related session2`)
- [ ] Add `--body` flag (inline body text)
- [ ] Add `--body-stdin` flag (read body prose only from stdin, no frontmatter needed)
- [ ] Validate that `--topic` is mutually exclusive with `--file`/`--stdin`
- [ ] Validate that `--body` and `--body-stdin` are mutually exclusive
- [ ] Construct `session.Session` from flags and call `session.Write()` directly
- [ ] Add tests: flag-based save with topic only, with all flags, with --body-stdin, conflict errors

### Shared
- [ ] Update `USAGE.md` — flag-based examples as primary, file/stdin as secondary
- [ ] Run `go test ./...` and confirm all tests pass

## Notes

For `logos task create`, the `--description` flag maps to the body of the
`## What` section in the generated markdown file. If omitted, the section is
left empty (same behaviour as an empty body in the current `--stdin` flow).

For `logos save`, `--body-stdin` reads raw prose from stdin without any
frontmatter. This is distinct from the existing `--stdin` which expects a
complete markdown document with frontmatter. The `--body` flag is suitable for
short summaries passed inline; `--body-stdin` is better for piping longer
session notes.

`--tag` and `--related` should be repeatable on both commands (cobra supports
`StringArrayVar`).