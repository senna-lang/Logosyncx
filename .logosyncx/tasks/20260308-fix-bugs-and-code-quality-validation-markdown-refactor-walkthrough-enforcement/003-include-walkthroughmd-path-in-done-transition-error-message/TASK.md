---
id: t-c5f88f
date: 2026-03-08T23:21:59.631741+09:00
title: Include WALKTHROUGH.md path in done-transition error message
seq: 3
status: done
priority: low
plan: 20260308-fix-bugs-and-code-quality-validation-markdown-refactor-walkthrough-enforcement
tags: []
assignee: ""
completed_at: 2026-03-09T21:33:05.735192+09:00
---

## What

Improve the error message returned when `logos task update --status done` is rejected due to missing or scaffold-only WALKTHROUGH.md content. The message should include the exact file path the agent needs to write to, eliminating the need to infer the path.

## Why

Current error:
```
WALKTHROUGH.md has no content: write WALKTHROUGH.md content first, then re-run
```

The agent still has to infer the path from the task directory structure. A more actionable message:
```
WALKTHROUGH.md has no content: write content to
  .logosyncx/tasks/<plan-slug>/<NNN-task-name>/WALKTHROUGH.md
then re-run
```

This eliminates a reasoning step and prevents mistakes on deeply nested paths.

## Scope

- `internal/task/store.go` — `UpdateFields` status→done path: include `wPath` (absolute or project-relative) in the error string
- `internal/task/store_test.go` — update existing error message assertions to match new format

Out of scope: changing the validation logic itself (task 009).

## Acceptance Criteria

- [ ] Error message contains the relative path to WALKTHROUGH.md
- [ ] Path is relative to project root (not absolute) for readability
- [ ] Existing tests updated to match new message format
- [ ] `go test ./...` passes

## Checklist

- [ ] Compute relative path from `t.DirPath` + `walkthroughFileName` relative to `s.projectRoot`
- [ ] Embed path in error string
- [ ] Update `store_test.go` assertions
- [ ] `go test ./...` passes

## Notes

Use `filepath.Rel(s.projectRoot, wPath)` to get the relative path. Fall back to absolute if `Rel` fails.
