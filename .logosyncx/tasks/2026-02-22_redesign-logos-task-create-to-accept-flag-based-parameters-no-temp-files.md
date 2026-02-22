---
id: t-86cf53
date: 2026-02-22T20:34:56+09:00
title: Redesign logos task create to accept flag-based parameters (no temp files)
status: open
priority: medium
session: 2026-02-22_logos-task-create-redesign-beads-comparison.md
tags:
    - cli
    - dx
    - task
    - design
assignee: ""
---

## What

Add flag-based creation to `logos task create` so that agents (and humans) can
create a task with a single command — no temporary markdown file required.

Proposed interface:

```
logos task create --title "..." [--description "..."] [--priority high|medium|low] \
                  [--tag <tag>] [--session <partial>]
```

The existing `--file` / `--stdin` interface must be kept for backward
compatibility but the flag-based path becomes the preferred route.

## Why

Currently `logos task create` requires a full markdown document with YAML
frontmatter to be passed via `--file <path>` or `--stdin`. This forces agents
to:

1. Build a complete frontmatter + body string.
2. Either write it to a temporary file or pass it through a heredoc/process
   substitution.
3. Clean up the temporary file afterward.

This is unnecessary friction. `internal/task/store.Store.Save()` already
auto-fills `id`, `date`, `status`, and `priority`, so the backend is ready —
only the CLI surface is missing the flag-based entry point.

beads (`bd create "title" --description "..." --priority 2 --labels "..."`)
shows the right pattern: structured fields as flags, no intermediate file.

## Scope

- `cmd/task.go` — extend `taskCreateCmd` (and `runTaskCreate`) to accept
  `--title`, `--description`, `--priority`, `--tag` (repeatable), `--session`
  flags; when `--title` is present and neither `--file` nor `--stdin` is given,
  construct a `task.Task` directly from flags and call `store.Save()`
- `internal/task/task.go` — no changes expected (Task struct is already
  sufficient)
- `internal/task/store.go` — no changes expected
- `.logosyncx/USAGE.md` — update the "Create a task" section to document the
  new flag-based interface as the preferred method
- `cmd/task_test.go` (or new file) — add unit/integration tests for the
  flag-based path

## Checklist

- [ ] Add `--title` flag to `taskCreateCmd`
- [ ] Add `--description` flag (sets the `## What` section body)
- [ ] Add `--priority` flag (high | medium | low)
- [ ] Add `--tag` flag (repeatable: `--tag go --tag tooling`)
- [ ] Validate that `--title` is mutually exclusive with `--file`/`--stdin`
- [ ] Construct `task.Task` from flags and call `store.Save()` directly
- [ ] Resolve `--session` partial name via `store.ResolveSession()` (already exists)
- [ ] Update `USAGE.md` — flag-based example as primary, file/stdin as secondary
- [ ] Add tests: flag-based create with title only, with all flags, conflict errors
- [ ] Run `go test ./...` and confirm all tests pass

## Notes

The `--description` flag maps to the body of the `## What` section in the
generated markdown file. If omitted, the section is left empty (same behaviour
as an empty body in the current `--stdin` flow).

`--tag` should be repeatable (cobra supports `StringArrayVar`).

Longer-term, the same friction exists for `logos save` (sessions), but
session content is inherently long-form prose so a flag-based interface is
less practical there. That is a separate problem out of scope for this task.