---
id: t-2354fa
date: 2026-03-08T23:21:35.552147+09:00
title: Auto-run logos sync after logos save and logos task update
seq: 2
status: done
priority: low
plan: 20260308-fix-bugs-and-code-quality-validation-markdown-refactor-walkthrough-enforcement
tags: []
assignee: ""
completed_at: 2026-03-09T21:33:05.711552+09:00
---

## What

Automatically rebuild the plan and task indexes after `logos save` and `logos task update` without requiring a manual `logos sync` call. The index should always reflect the latest state immediately after any write operation.

## Why

Currently agents must remember to run `logos sync` after saving plans or updating tasks, otherwise `logos ls --json` and `logos task ls --json` return stale data. This is a hidden dependency that causes confusion and is easy to forget. Making sync implicit removes a manual step from every workflow.

## Scope

- `cmd/save.go` — call index rebuild after writing the plan file
- `cmd/task.go` — call index rebuild after `logos task update` and `logos task create`
- Both already call partial index helpers (`AppendTaskIndex`, `RebuildTaskIndex`); this task is about ensuring the plan index side is also kept in sync automatically

Out of scope: `logos sync` command itself (keep it for manual recovery/repair); session-level sync triggered by other commands.

## Acceptance Criteria

- [ ] After `logos save`, `logos ls --json` returns the new plan without running `logos sync`
- [ ] After `logos task update`, `logos task ls --json` reflects the change without running `logos sync`
- [ ] `logos sync` still works as a manual repair command
- [ ] `go test ./...` passes

## Checklist

- [ ] Identify all write paths in `cmd/save.go` and `cmd/task.go`
- [ ] Add index rebuild calls where missing
- [ ] Verify with integration tests that index is up to date after each command
- [ ] `go test ./...` passes

## Notes

`RebuildTaskIndex` already exists in `internal/task/store.go` and is called in some paths. Check whether plan index (`pkg/index`) has an equivalent and wire it up similarly.
