---
id: t-bdsa1j
date: 2026-02-21T10:42:59.064774+09:00
title: "[Phase2] Implement logos status command"
status: open
priority: medium
session: ""
tags:
    - phase2
    - cli
    - git
assignee: ""
---

## What

Implement `logos status` to display unsaved or uncommitted session/task
information. Integrate with git status to list newly added or modified files
under `.logosyncx/sessions/` and `.logosyncx/tasks/`.

## Why

After running `logos save` or `logos task create`, the user is responsible for
committing and pushing the resulting files. Without a status command, there is
no easy way to see which sessions or tasks have been saved locally but not yet
committed — especially relevant for agents that need to confirm their save
operations actually persisted before ending a session.

## Scope

- `cmd/status.go` — new file implementing the `logos status` cobra command
- `cmd/status_test.go` — integration tests using temp git repos
- `internal/gitutil/` — may need a new helper to query git status for a
  specific path prefix

## Checklist

- [ ] Create `cmd/status.go` with the `logos status` command
- [ ] Query git status for files under `.logosyncx/sessions/` (new, modified, staged)
- [ ] Query git status for files under `.logosyncx/tasks/` (new, modified, staged)
- [ ] Display a human-readable summary grouped by state (staged / unstaged / untracked)
- [ ] Exit with code 0 even when there are uncommitted files (informational only)
- [ ] Write integration tests for the command
- [ ] Run `go test ./cmd/...` and confirm all tests pass

## Notes

Migrated from beads issue `logosyncx-a1j`.

Example output shape:

```
Uncommitted sessions:
  (new)      2026-02-21_my-session.md
  (modified) 2026-02-20_older-session.md

Uncommitted tasks:
  (new)      2026-02-21_my-task.md

Nothing to commit in .logosyncx/ — all files are staged or clean.
```

The command is informational and should never block or modify state.