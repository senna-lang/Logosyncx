---
id: t-a6dd93
date: 2026-03-07T21:31:11.257949+09:00
title: Add status/priority validation in UpdateFields
seq: 3
status: open
priority: medium
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
---

## What

Add input validation in `internal/task/store.go:UpdateFields` for the `status` and `priority` fields. Currently any string is accepted and written to TASK.md without checking against `ValidStatuses` / `ValidPriorities`, so `logos task update --name foo --status typo` silently corrupts the task.

## Why

Invalid status/priority values break downstream filtering and index queries. The validation helpers `IsValidStatus` and `IsValidPriority` already exist in `internal/task/task.go` but are not called from `UpdateFields`.

## Scope

- `internal/task/store.go` — `UpdateFields` status and priority cases
- `internal/task/store_test.go` — add tests for invalid status and priority rejection

Out of scope: validation at the cmd layer (store is the right place).

## Checklist

- [ ] Call `IsValidStatus` in the `"status"` case; return error if invalid
- [ ] Call `IsValidPriority` in the `"priority"` case; return error if invalid
- [ ] Add test: `UpdateFields` with invalid status returns error, file unchanged
- [ ] Add test: `UpdateFields` with invalid priority returns error, file unchanged
- [ ] `go test ./internal/task/...` passes
- [ ] `go test ./...` passes
