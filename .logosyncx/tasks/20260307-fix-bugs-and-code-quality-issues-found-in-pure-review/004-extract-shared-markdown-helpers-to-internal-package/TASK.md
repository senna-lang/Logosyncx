---
id: t-184aca
date: 2026-03-07T21:31:11.263931+09:00
title: Extract shared markdown helpers to internal package
seq: 4
status: open
priority: medium
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
---

## What

Create `internal/markdown/markdown.go` with the shared helpers that are currently duplicated between `pkg/plan/plan.go` and `internal/task/task.go`, then update both packages to import from it. Functions to extract: `splitFrontmatter`, `extractExcerpt`, `parseHeading`, `truncateRunes`, `slugify`.

## Why

Five functions are duplicated verbatim (with only minor differences like default excerpt section names). Any bug fix or behaviour change must be applied in two places, and the two copies can drift. Centralising eliminates the duplication.

## Scope

- `internal/markdown/markdown.go` — new file with extracted helpers
- `internal/markdown/markdown_test.go` — new file with shared tests
- `pkg/plan/plan.go` — replace local helpers with imports
- `pkg/plan/plan_test.go` — remove duplicate helper tests if now covered by markdown package
- `internal/task/task.go` — replace local helpers with imports
- `internal/task/task_test.go` — same

Out of scope: behaviour changes to the helpers; this is pure extraction.

## Checklist

- [ ] Create `internal/markdown/markdown.go` with all 5 helpers (exported)
- [ ] Write tests in `internal/markdown/markdown_test.go`
- [ ] Update `pkg/plan/plan.go` to use `markdown.*`
- [ ] Update `internal/task/task.go` to use `markdown.*`
- [ ] Delete duplicated private functions from both files
- [ ] `go test ./...` passes

## Notes

`extractExcerpt` differs only in default section name ("Background" vs "What") — make the default an argument or have callers always pass the section name explicitly.
