---
id: t-c4b8db
date: 2026-03-07T21:31:11.245426+09:00
title: Fix plan.Marshal to preserve body content
seq: 1
status: done
priority: high
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
completed_at: 2026-03-07T21:47:25.87231+09:00
---

## What

Fix `pkg/plan/plan.go:Marshal` to include the plan body in its output. Currently it only serialises the YAML frontmatter, so any call to `plan.Write` that passes a Plan with an existing body (e.g. marking `distilled: true` in `logos distill`) silently overwrites the file with frontmatter-only, destroying the written content.

## Why

`logos distill` calls `plan.Write(root, p)` after setting `p.Distilled = true`. Because `Marshal` omits `p.Body`, the plan body is lost every time a plan is distilled. This is a data-loss bug.

## Scope

- `pkg/plan/plan.go` — `Marshal` function
- `pkg/plan/plan_test.go` — add round-trip test covering body preservation

Out of scope: task.Marshal (already correct).

## Checklist

- [ ] Update `Marshal` to append `p.Body` after the closing `---` when non-empty
- [ ] Add test: parse a plan with body → Marshal → parse again → body unchanged
- [ ] `go test ./pkg/plan/...` passes
- [ ] `go test ./...` passes

## Notes

`internal/task/store.go:Marshal` already handles body correctly — use it as reference.
