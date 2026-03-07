---
id: t-266875
date: 2026-03-07T21:31:11.252033+09:00
title: Fix --blocked filter no-op in matchesFilter
seq: 2
status: done
priority: high
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
completed_at: 2026-03-07T21:47:25.880393+09:00
---

## What

Fix `internal/task/filter.go:matchesFilter` so that `f.Blocked = true` actually filters tasks. Currently the branch is `_ = f.Blocked` — an explicit no-op — meaning `logos task ls --blocked` returns all tasks instead of only blocked ones when not using `--json`.

## Why

`logos task ls --blocked` is documented behaviour. It works for JSON output (via `matchesJSONFilter`) but silently returns wrong results for table output. Any user relying on the non-JSON path gets an incorrect list.

## Scope

- `internal/task/filter.go` — `matchesFilter` blocked branch
- `internal/task/filter_test.go` — add test for Blocked filter on in-memory Task slice

Out of scope: `matchesJSONFilter` (already correct).

## Checklist

- [ ] Implement blocked check in `matchesFilter` using `IsBlocked` — requires passing sibling tasks or pre-computing the flag
- [ ] Decide approach: pre-set a `Blocked bool` field on Task, or pass planTasks into Apply
- [ ] Add test: task with unfinished dependency → `Apply(tasks, Filter{Blocked: true})` includes it
- [ ] `go test ./internal/task/...` passes
- [ ] `go test ./...` passes

## Notes

`matchesJSONFilter` uses `e.Blocked` which is set by `RebuildTaskIndex`. The cleanest fix for the in-memory path is probably to add a `Blocked bool` field to `Task` and set it during `loadAll`, mirroring what the JSON path does.
