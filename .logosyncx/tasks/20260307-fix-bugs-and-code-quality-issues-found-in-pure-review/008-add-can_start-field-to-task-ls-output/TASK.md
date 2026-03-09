---
id: t-19afa0
date: 2026-03-07T21:42:40.649994+09:00
title: Add can_start field to task ls output
seq: 8
status: done
priority: low
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
completed_at: 2026-03-09T21:38:35.64161+09:00
---

## What

Add a `can_start` boolean field to `TaskJSON` and expose it in `logos task ls --json` output and the task table. `can_start` is true when the task is `open` and not blocked (all `depends_on` deps are done). This lets agents determine which tasks are actionable without reasoning about the dependency graph themselves.

## Why

Currently parallelism is implicit: agents must infer "can I start this?" by cross-referencing `blocked` and `status`. Making it explicit reduces cognitive load and removes the need for agents to re-implement `IsBlocked` logic. `seq` stays as-is; this is a display/output improvement only.

## Scope

- `internal/task/task.go` — add `CanStart bool` to `TaskJSON`
- `internal/task/store.go` — set `CanStart` in `RebuildTaskIndex` and `loadAll` (mirror how `Blocked` is set)
- `internal/task/filter.go` — no change needed
- `cmd/task.go` — add `CAN START` column to table output; already in JSON via TaskJSON
- `internal/task/index.go` / `internal/task/index_test.go` — update TaskJSON serialisation tests

Out of scope: changing `seq` semantics, changing `depends_on` key type.

## Checklist

- [ ] Add `CanStart bool \`json:"can_start"\`` to `TaskJSON`
- [ ] Set `CanStart = !blocked && status == open` in `RebuildTaskIndex`
- [ ] Set `CanStart` in `loadAll` alongside `Blocked` computation
- [ ] Add `CAN START` column to `logos task ls` table (non-JSON)
- [ ] Add test: task with all deps done → `can_start: true`; task with open dep → `can_start: false`
- [ ] `go test ./...` passes

## Notes

`can_start` depends on task 002 (Blocked field working correctly in matchesFilter) being done first, but can be implemented independently since it uses the JSON/index path where Blocked is already correct.
