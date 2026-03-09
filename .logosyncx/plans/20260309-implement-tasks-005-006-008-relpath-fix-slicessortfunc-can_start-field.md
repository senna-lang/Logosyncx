---
id: 3c6e90
topic: 'implement tasks 005-006-008: relpath fix, slices.SortFunc, can_start field'
tags:
    - go
    - fix
agent: claude-sonnet-4-6
related: []
tasks_dir: .logosyncx/tasks/20260309-implement-tasks-005-006-008-relpath-fix-slicessortfunc-can_start-field
distilled: false
---

## Background

Implemented three remaining open tasks from the 20260307 code-quality plan:
- Task 005: Fix relPath to use filepath.Rel (was using fragile strings.TrimPrefix)
- Task 006: Replace hand-rolled sortByDateDesc with slices.SortFunc in cmd/ls.go and store.go
- Task 008: Add can_start boolean field to TaskJSON (open && !blocked)

## Spec

Task 005: `relPath(base, target)` now calls `filepath.Rel(base, target)` with proper error return.
Task 006: Both `sortByDateDesc` functions replaced with `slices.SortFunc` + `time.Time.Compare`.
Task 008: `TaskJSON.CanStart` added; set in `RebuildTaskIndex` after blocked computation;
table shows `✓` in `START` column.

## Key Decisions

Decision: Follow the existing `Blocked` pattern for `CanStart` (false in ToJSON, set by store).
Rationale: Ensures task-index.jsonl is the authoritative source; avoids stale values.

## Notes

Task 007 (rename AutoPush to AutoStage) is now obsolete — AutoPush actually does commit+push
since Task 001. All tests pass: `go test ./...` green.
