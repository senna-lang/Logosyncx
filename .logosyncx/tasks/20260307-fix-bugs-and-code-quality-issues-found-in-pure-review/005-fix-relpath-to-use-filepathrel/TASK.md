---
id: t-91a36f
date: 2026-03-07T21:31:11.270209+09:00
title: Fix relPath to use filepath.Rel
seq: 5
status: open
priority: low
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
---

## What

Replace the `relPath` helper in `cmd/save.go` with a proper `filepath.Rel` call. The current implementation uses `strings.TrimPrefix(target, base+"/")` which is path-separator-fragile and inconsistent with `distill.go` which already uses `filepath.Rel` correctly.

## Why

The string-based implementation breaks on Windows (backslash separator) and on any path where the prefix doesn't match exactly. `filepath.Rel` is the correct standard-library solution and is already used in the same codebase.

## Scope

- `cmd/save.go` — `relPath` function and its callers
- `cmd/save_test.go` — add test if not already covered

Out of scope: other cmd files (they already use filepath.Rel directly).

## Checklist

- [ ] Replace `relPath` body with `filepath.Rel(base, target)`
- [ ] Handle the error return properly
- [ ] `go test ./cmd/...` passes
- [ ] `go test ./...` passes
