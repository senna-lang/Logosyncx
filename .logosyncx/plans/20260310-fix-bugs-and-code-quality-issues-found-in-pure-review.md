---
id: f0b4af
topic: Fix bugs and code quality issues found in pure review
tags:
    - bugfix
    - refactor
agent: claude-code
related: []
tasks_dir: .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
distilled: true
---

## Background

A pure code review of the logosyncx codebase surfaced 3 bugs with runtime impact and 4 code quality issues. The bugs affect core workflows: `logos distill` silently destroys plan body content, `--blocked` filtering is a no-op for non-JSON task output, and invalid status/priority values are accepted without error. The code quality issues include duplicated markdown parsing helpers across plan and task packages, a fragile relPath implementation, an O(n²) sort, and a misleading config field name.

## Spec

Fix all 7 issues identified in the review. Bugs are prioritised first (001–003), then code quality (004–007). Task 004 (extract shared helpers) should be done after 001–003 so the refactored files are stable before deduplication.

## Key Decisions

- Decision: Fix plan.Marshal to write body on update. Rationale: Write is called both for initial scaffold (no body) and for updates like marking distilled. Body must be preserved on update.
- Decision: Extract shared markdown helpers to `internal/markdown` package. Rationale: `splitFrontmatter`, `extractExcerpt`, `parseHeading`, `truncateRunes`, `slugify` are duplicated verbatim between `pkg/plan` and `internal/task`.
- Decision: 001–003 are independent. 004 depends on 001–003 being stable first.

## Notes

- Review conducted 2026-03-07 by claude-code.
- All issues are in existing code; no new features involved.
