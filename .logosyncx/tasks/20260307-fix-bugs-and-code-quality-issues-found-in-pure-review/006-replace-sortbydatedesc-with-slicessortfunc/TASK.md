---
id: t-59d4e3
date: 2026-03-07T21:31:11.276656+09:00
title: Replace sortByDateDesc with slices.SortFunc
seq: 6
status: done
priority: low
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
completed_at: 2026-03-09T21:38:35.6177+09:00
---

## What

Replace the hand-rolled insertion-sort `sortByDateDesc` in `cmd/ls.go` and `internal/task/store.go` with `slices.SortFunc` from the standard library.

## Why

The current implementation is O(n²) insertion sort. While plan/task counts are small today, `slices.SortFunc` is idiomatic Go 1.21+, already used elsewhere in the codebase (`slices.Contains`, `slices.ContainsFunc`), and more readable.

## Scope

- `cmd/ls.go` — `sortByDateDesc`
- `internal/task/store.go` — `sortByDateDesc`
- Tests: verify sort order is unchanged

Out of scope: sort behaviour changes (still newest-first).

## Checklist

- [ ] Replace `cmd/ls.go:sortByDateDesc` with `slices.SortFunc`
- [ ] Replace `internal/task/store.go:sortByDateDesc` with `slices.SortFunc`
- [ ] `go test ./...` passes
