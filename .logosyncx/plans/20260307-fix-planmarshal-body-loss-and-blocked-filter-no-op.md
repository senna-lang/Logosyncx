---
id: abfac4
topic: Fix plan.Marshal body loss and --blocked filter no-op
tags:
    - bugfix
agent: claude-sonnet-4-5
related: []
tasks_dir: .logosyncx/tasks/20260307-fix-planmarshal-body-loss-and-blocked-filter-no-op
distilled: false
---

## Background

Two bugs were identified in a pure code review and fixed in this session.

**Bug 1 — plan.Marshal destroys body content (data loss):**
`pkg/plan/plan.go:Marshal` only serialised the YAML frontmatter and discarded `p.Body`. Any code path that rewrites a plan file via `Marshal` (e.g. `logos distill` setting `Distilled: true`) silently overwrote the file with frontmatter-only, destroying the written body. The fix mirrors the already-correct `internal/task/task.Marshal` implementation.

**Bug 2 — `--blocked` filter is a no-op for table output:**
`internal/task/filter.go:matchesFilter` had an explicit `_ = f.Blocked` no-op, so `logos task ls --blocked` returned all tasks instead of only blocked ones when not using `--json`. The JSON path (`matchesJSONFilter` / `ApplyToJSON`) worked correctly because it read `e.Blocked` from the index. The in-memory path had no equivalent field to check.

## Spec

### Task 001 — Fix plan.Marshal to preserve body content

- Modified `pkg/plan/plan.go:Marshal` to append `p.Body` after the closing `---` when non-empty, inserting a leading newline if the body doesn't already start with one.
- Updated the doc comment to clarify that `Write` (scaffold path, body empty) and callers like `logos distill` (rewrite path, body present) both use the same `Marshal`.
- Added three tests to `pkg/plan/plan_test.go`:
  - `TestMarshal_PreservesBodyContent` — full round-trip with body
  - `TestMarshal_EmptyBody_NoTrailingContent` — scaffold path still ends with `---`
  - `TestMarshal_BodyPreservedAfterDistilledUpdate` — regression for the distill data-loss scenario

### Task 002 — Fix --blocked filter no-op in matchesFilter

- Added `Blocked bool \`yaml:"-"\`` derived field to `internal/task/task.go:Task` (not written to frontmatter).
- Updated `internal/task/store.go:loadAll` to set `t.Blocked = IsBlocked(t, planTasks)` for each task after loading a plan group, replacing the previous no-op loop.
- Updated `internal/task/filter.go:matchesFilter` to check `!t.Blocked` when `f.Blocked` is true, replacing the `_ = f.Blocked` no-op.
- Added three tests to `internal/task/filter_test.go` for the in-memory path:
  - `TestApply_BlockedFilter_InMemory` — blocked task is returned
  - `TestApply_BlockedFilter_False_MatchesAll` — `Blocked: false` means no constraint
  - `TestApply_BlockedFilter_NoBlockedTasks_ReturnsEmpty` — filter returns empty when nothing is blocked

## Key Decisions

Decision: Store `Blocked` on the `Task` struct as a derived `yaml:"-"` field rather than computing it on every `matchesFilter` call by passing `planTasks` into `Apply`. Rationale: avoids changing the `Apply` signature and matches the pattern already used by the JSON path (`TaskJSON.Blocked` set by the store). The store (`loadAll`) is the natural place to compute per-plan group metadata.

Decision: Keep `Write` (scaffold creation) using the same `Marshal` function. Rationale: when `Write` is called for a new plan, `p.Body` is always `""`, so the body-append branch is never taken — no behaviour change for the scaffold path.

## Notes

- All tests pass: `go test ./...` green across all packages.
- The `TestWrite_ScaffoldOnly_NoBody` existing test continued to pass, confirming the scaffold path was not broken.
- Tasks 003–008 from the same plan remain open.
