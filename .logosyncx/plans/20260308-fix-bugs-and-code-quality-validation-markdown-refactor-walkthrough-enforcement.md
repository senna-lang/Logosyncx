---
id: 6daa39
topic: 'Fix bugs and code quality: validation, markdown refactor, walkthrough enforcement'
tags:
    - bugfix
    - refactor
agent: claude-sonnet-4-6
related: []
tasks_dir: .logosyncx/tasks/20260308-fix-bugs-and-code-quality-validation-markdown-refactor-walkthrough-enforcement
distilled: false
---

## Background

Three medium-priority tasks from the "fix bugs and code quality" review plan were addressed in this session. The tasks covered input validation, code deduplication, and workflow enforcement for the logos CLI tool.

## Spec

**Task 003 — Status/priority validation in UpdateFields:**
Added `IsValidStatus` and `IsValidPriority` checks in `internal/task/store.go:UpdateFields`. Invalid values now return descriptive errors instead of silently corrupting TASK.md.

**Task 004 — Extract shared markdown helpers:**
Created `internal/markdown/markdown.go` with exported helpers (`Slugify`, `SplitFrontmatter`, `ExtractExcerpt`, `ParseHeading`, `TruncateRunes`). Removed duplicated private implementations from `pkg/plan/plan.go` and `internal/task/task.go`. Both packages and their tests now import from the shared package. The `extractExcerpt` default section difference ("Background" vs "What") is resolved by having each caller pass its own default.

**Task 009 — Require WALKTHROUGH content before marking task done:**
Added `walkthroughHasContent(path string) bool` helper in `internal/task/store.go`. `UpdateFields` now checks this before accepting status→done transitions. A line counts as content if it is non-empty and does not start with `<!--`. Updated all affected tests in `internal/task/` and `cmd/` to write WALKTHROUGH.md content before calling `UpdateFields("done")`.

## Key Decisions

Decision: Extract to `internal/markdown` (not `pkg/markdown`). Rationale: both consumers are internal packages; no external code should import it directly.

Decision: Default excerpt section handled by caller, not by shared function. Rationale: keeps `ExtractExcerpt` free of package-specific defaults; each caller explicitly sets "Background" or "What" when `opts.ExcerptSection` is empty.

Decision: Scaffold detection is "any non-empty, non-comment line". Rationale: simple and covers auto-generated scaffold which consists entirely of HTML comment blocks.

## Notes

All tests pass: `go test ./...` green across all packages.
