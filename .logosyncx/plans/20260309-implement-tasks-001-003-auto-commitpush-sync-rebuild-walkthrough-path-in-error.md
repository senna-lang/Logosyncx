---
id: 248ae9
topic: 'implement tasks 001-003: auto commit/push, sync rebuild, walkthrough path in error'
tags:
    - go
    - fix
agent: claude-sonnet-4-6
related: []
tasks_dir: .logosyncx/tasks/20260309-implement-tasks-001-003-auto-commitpush-sync-rebuild-walkthrough-path-in-error
distilled: false
---

## Background

Implemented three open tasks from the 20260308-fix-bugs plan:
- Task 001: Auto git commit+push when marking task done with auto_push enabled
- Task 002: Auto-run logos sync (full index rebuild) after logos save and logos task update/create
- Task 003: Include WALKTHROUGH.md path in done-transition error message

## Spec

Task 001: In `UpdateFields`, track `transitionedToDone` flag; after git add + index rebuild,
call `gitutil.Commit`/`gitutil.Push` when `auto_push=true`. Non-fatal (warn to stderr).

Task 002: Replace `index.Append` in `cmd/save.go` with `index.Rebuild` (full rebuild).
Add `config.Load` to get `ExcerptSection`. Replace `AppendTaskIndex` in `store.Create` with
`RebuildTaskIndex` for full task index consistency.

Task 003: In `UpdateFields` walkthrough check, compute `filepath.Rel(s.projectRoot, wPath)`
and embed it in the error message so agents know exactly where to write.

## Key Decisions

Decision: Use full `index.Rebuild` instead of `index.Append` in `logos save`.
Rationale: Consistency over performance; for a local dev CLI, a linear scan is acceptable
and prevents stale/duplicate entries in the index.

Decision: Commit+push failures are non-fatal (warn to stderr).
Rationale: Network unavailability or missing git config should not prevent task completion.

## Notes

All tests pass: `go test ./...` green. Three WALKTHROUGH.md files written and tasks marked done.
