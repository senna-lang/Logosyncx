---
id: k-8d19d4
date: 2026-03-10T10:47:33.309965Z
topic: Fix bugs and code quality issues found in pure review
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review.md
tasks:
    - 009-Require WALKTHROUGH content before marking task done
    - 008-Add can_start field to task ls output
    - 007-Fix AutoPush config naming and comment
    - 006-Replace sortByDateDesc with slices.SortFunc
    - 005-Fix relPath to use filepath.Rel
    - 004-Extract shared markdown helpers to internal package
    - 003-Add status/priority validation in UpdateFields
    - 002-Fix --blocked filter no-op in matchesFilter
    - 001-Fix plan.Marshal to preserve body content
tags:
    - bugfix
    - refactor
---

<!-- SOURCE MATERIAL — read this, fill in the sections below, then remove this block. -->
<!--
## Plan: Fix bugs and code quality issues found in pure review

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

---
## Walkthrough: 009 Require WALKTHROUGH content before marking task done

# Walkthrough: Require WALKTHROUGH content before marking task done

<!-- Auto-generated when this task was marked done. -->

## Key Specification

The old flow (mark done → scaffold generated → agent writes) let agents skip writing the walkthrough with no immediate consequence. The fix reverses the order: agents must write WALKTHROUGH.md content first, then mark done. The CLI enforces this at the `UpdateFields` status→done transition.

A line counts as "real content" if it is non-empty after trimming and does not start with `<!--`. Scaffold-only files (all HTML comment blocks) are rejected.

## What Was Done

- Added `walkthroughHasContent(path string) bool` helper in `store.go` — reads the file and returns true if at least one non-empty, non-comment line exists.
- In `UpdateFields` status→done path: call `walkthroughHasContent` before setting `CompletedAt` and transitioning. Return a descriptive error if the check fails.
- Added three tests in `store_test.go`:
  - `TestStore_UpdateFields_Done_RequiresWalkthroughContent` — no file → error.
  - `TestStore_UpdateFields_Done_ScaffoldOnly_Rejected` — all `<!--` lines → error.
  - `TestStore_UpdateFields_Done_WithContent_Succeeds` — real content → success.
- Updated all existing tests that called `UpdateFields("done")` to pre-write WALKTHROUGH.md content first.
- Updated `cmd/distill_test.go` and `cmd/task_test.go` for the same reason.
- Updated `USAGE.md`, `CLAUDE.md`, and `cmd/init.go` (`usageMD`) to document the new required order.

## How It Was Done

1. Added the helper and the guard clause in `store.go`.
2. Ran `go test ./internal/task/...` — 1 pre-existing test failed (`TestStore_UpdateFields_InProgress_NotBlocked_WhenDepDone`) because it marked a dep task done without a walkthrough. Fixed by pre-writing content.
3. Ran `go test ./...` — distill and task cmd tests also failed for the same reason. Fixed all call sites.
4. Updated documentation in three files (`USAGE.md`, `CLAUDE.md`, `cmd/init.go`).

## Gotchas & Lessons Learned

The impact was wider than expected — many tests in `cmd/` used `UpdateFields("done")` as a test setup step and all needed updating. Any enforcement added to a core transition function will cascade to all test helpers that use it.

`CreateWalkthroughScaffold` remains idempotent — it still creates the scaffold if the file doesn't exist yet. The new check runs before that, so the flow is: check content → (reject if empty) → write TASK.md → create scaffold if missing.

## Reusable Patterns

```go
// walkthroughHasContent: simple heuristic for "has the agent actually written anything?"
func walkthroughHasContent(path string) bool {
    data, err := os.ReadFile(path)
    if err != nil {
        return false
    }
    for _, line := range strings.Split(string(data), "\n") {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" && !strings.HasPrefix(trimmed, "<!--") {
            return true
        }
    }
    return false
}
```

Enforce order at the CLI layer rather than relying on agent discipline — it's the only reliable way to close workflow gaps.

---
## Walkthrough: 008 Add can_start field to task ls output

# Walkthrough: Add can_start field to task ls output

## Key Specification

Add `can_start bool` to `TaskJSON`. True when `status == open && !blocked`.
Expose in `logos task ls --json` and the task table.

## What Was Done

- `internal/task/task.go`: Added `CanStart bool \`json:"can_start"\`` to `TaskJSON`.
  Set `CanStart: false` in `ToJSON()` (store sets it, same pattern as `Blocked`).
- `internal/task/store.go` `RebuildTaskIndex`: Added `entry.CanStart = t.Status == StatusOpen && !entry.Blocked`
  after setting `entry.Blocked`.
- `cmd/task.go` `printTaskTable`: Added `START` column showing `✓` when `CanStart` is true.
- Added tests: `TestStore_RebuildTaskIndex_SetsCanStart` and `TestStore_RebuildTaskIndex_DoneTask_CanStartFalse`.

## How It Was Done

Followed the exact same pattern as `Blocked` — set to false in `ToJSON`, computed in
`RebuildTaskIndex` after dependency resolution. `CanStart` is purely derived from
`Status` and `Blocked` so no separate computation logic is needed.

## Gotchas & Lessons Learned

`ToJSON()` hardcodes `Blocked: false` and `CanStart: false` because the store sets them
after calling `FromTask`. The in-memory path (`List` → `ToJSON`) returns incorrect
`Blocked`/`CanStart` for search results — pre-existing issue, out of scope for this task.

## Reusable Patterns

Derived boolean fields in JSON output: compute them after dependency resolution in
`RebuildTaskIndex`, not in `ToJSON`, to ensure the task index is the source of truth.

---
## Walkthrough: 007 Fix AutoPush config naming and comment

# Walkthrough: Fix AutoPush config naming and comment

## Key Specification

Rename `GitConfig.AutoPush` to `AutoStage` (or `AutoAdd`) because the field was
originally only controlling `git add`. The comment was misleading.

## What Was Done

No code change made. Task is obsolete.

Task 001 (Auto git commit and push when marking task done) was implemented first,
which made `AutoPush` actually perform `git commit` and `git push` on done transitions.
The field name and the config comment now accurately reflect the behavior — the flag
controls full git automation including add, commit, and push.

## How It Was Done

Reviewed the current implementation in `internal/task/store.go` and confirmed that
`AutoPush=true` now triggers `git add`, `git commit`, and `git push`. The original
concern (misleading name) is no longer valid.

## Gotchas & Lessons Learned

Task ordering matters. If Task 007 had been implemented before Task 001, the rename
would have required updating all references. Doing Task 001 first made Task 007 moot.

## Reusable Patterns

N/A — no implementation required.

---
## Walkthrough: 006 Replace sortByDateDesc with slices.SortFunc

# Walkthrough: Replace sortByDateDesc with slices.SortFunc

## Key Specification

Replace hand-rolled insertion-sort `sortByDateDesc` in `cmd/ls.go` and
`internal/task/store.go` with `slices.SortFunc` from the standard library.

## What Was Done

- `cmd/ls.go`: `sortByDateDesc` replaced with `slices.SortFunc` using `b.Date.Compare(a.Date)`
  (already had `"slices"` imported)
- `internal/task/store.go`: same replacement; added `"slices"` import; removed `"sort"` is kept
  for `sort.Slice` used in `loadPlanTasks`

## How It Was Done

Used `time.Time.Compare` (Go 1.20+) which returns -1/0/1, making it directly usable as the
comparator for `slices.SortFunc`.

## Gotchas & Lessons Learned

`store.go` still needs `"sort"` for `sort.Slice` in `loadPlanTasks`. Only `sortByDateDesc`
was replaced, not all sorting.

## Reusable Patterns

```go
slices.SortFunc(items, func(a, b T) int {
    return b.Date.Compare(a.Date) // newest-first
})
```

---
## Walkthrough: 005 Fix relPath to use filepath.Rel

# Walkthrough: Fix relPath to use filepath.Rel

## Key Specification

Replace `strings.TrimPrefix(target, base+"/")` in `cmd/save.go` `relPath` helper
with `filepath.Rel(base, target)` to be path-separator-correct and consistent with
`distill.go` which already uses `filepath.Rel`.

## What Was Done

- Added `"path/filepath"` import to `cmd/save.go`
- Replaced `strings.TrimPrefix(target, base+"/")` with `filepath.Rel(base, target)`
- Error is now returned properly (old impl silently returned wrong paths on Windows or
  when paths didn't share a common prefix)

## How It Was Done

Simple one-function change. `relPath` is shared across `cmd/save.go`, `cmd/task.go`,
and `cmd/distill.go` via the same package, so one fix covers all callers.

## Gotchas & Lessons Learned

`strings.TrimPrefix` is fragile on Windows (backslash separators) and incorrect when
`target` doesn't start with `base+"/"` exactly. `filepath.Rel` handles all OS cases.

## Reusable Patterns

Always use `filepath.Rel` for computing relative paths — never string manipulation.

---
## Walkthrough: 004 Extract shared markdown helpers to internal package

# Walkthrough: Extract shared markdown helpers to internal package

<!-- Auto-generated when this task was marked done. -->

## Key Specification

Five helper functions (`slugify`, `splitFrontmatter`, `extractExcerpt`, `parseHeading`, `truncateRunes`) were duplicated verbatim between `pkg/plan/plan.go` and `internal/task/task.go`, with only one behavioural difference: the default excerpt section name ("Background" vs "What"). The task was pure extraction — no behaviour changes.

## What Was Done

- Created `internal/markdown/markdown.go` with exported versions: `Slugify`, `SplitFrontmatter`, `ExtractExcerpt`, `ParseHeading`, `TruncateRunes`, and the `ExcerptMaxRunes` constant.
- Created `internal/markdown/markdown_test.go` covering all five functions.
- Removed local private implementations from both `pkg/plan/plan.go` and `internal/task/task.go`.
- Updated both packages to import `internal/markdown` and use the exported names.
- Fixed test files (`plan_test.go`, `task_test.go`, `index_test.go`) to use `markdown.*` calls.
- Removed now-unused `excerptMaxRunes` constants from both packages.
- The "Background" vs "What" default difference is resolved by having each caller set its own default before passing to `ExtractExcerpt`.

## How It Was Done

1. Read both source files side-by-side to confirm the functions were identical (modulo default section name).
2. Created the new package with exported names and no hard-coded defaults in `ExtractExcerpt`.
3. Updated callers one package at a time (`pkg/plan` first, then `internal/task`), running `go build` after each to surface errors.
4. The `replace_all` on `slugify(` accidentally replaced the function definition line — caught by the build error and fixed manually.
5. Ran `make fmt` to satisfy the gofmt pre-commit hook before committing.

## Gotchas & Lessons Learned

`replace_all` on short strings like `slugify(` is risky — it also hit the function definition `func slugify(`. Use longer unique context strings or replace the definition separately.

`errors` and `unicode/utf8` imports became unused after removing the local helpers — Go's compiler catches these immediately.

## Reusable Patterns

```go
// Caller sets own default before delegating to shared helper
section := opts.ExcerptSection
if section == "" {
    section = "Background" // or "What" in task package
}
p.Excerpt = markdown.ExtractExcerpt(body, section)
```

When extracting shared code between packages with minor behavioural differences, push the difference to the caller rather than adding parameters or flags to the shared function.

---
## Walkthrough: 003 Add status/priority validation in UpdateFields

# Walkthrough: Add status/priority validation in UpdateFields

<!-- Auto-generated when this task was marked done. -->

## Key Specification

`UpdateFields` in `internal/task/store.go` accepted any string for `status` and `priority` and wrote it directly to TASK.md without checking validity. `IsValidStatus` and `IsValidPriority` helpers already existed in `task.go` but were never called from `UpdateFields`, so `logos task update --status typo` would silently corrupt the task file.

## What Was Done

Added guard clauses in `UpdateFields` for both fields:

- `"status"` case: `IsValidStatus` check returns `"invalid status %q: must be one of open, in_progress, done"` on failure.
- `"priority"` case: `IsValidPriority` check returns `"invalid priority %q: must be one of low, medium, high"` on failure.

Added two tests in `store_test.go`:
- `TestStore_UpdateFields_InvalidStatus_ReturnsError` — verifies error and that status is unchanged on disk.
- `TestStore_UpdateFields_InvalidPriority_ReturnsError` — same for priority.

## How It Was Done

1. Located the `switch k` block in `UpdateFields`.
2. Added validation at the top of each case, before any side-effect logic (blocked check, CompletedAt, etc.).
3. Tests reload the task from disk after the failed update to confirm the field was not mutated.

## Gotchas & Lessons Learned

Validation must happen before any state transition logic — otherwise an invalid status could partially mutate in-memory state before returning an error. Guard clauses at the top of each case prevent this.

## Reusable Patterns

```go
// Validate before acting — guard clause at top of each case
case "status":
    newStatus := Status(v)
    if !IsValidStatus(newStatus) {
        return fmt.Errorf("invalid status %q: must be one of open, in_progress, done", v)
    }
    // transition logic follows
```

---
## Walkthrough: 002 Fix --blocked filter no-op in matchesFilter

# Walkthrough: Fix --blocked filter no-op in matchesFilter

## What Was Done

Fixed a silent correctness bug in `internal/task/filter.go:matchesFilter` where
`logos task ls --blocked` returned all tasks instead of only blocked ones when
not using `--json`.

The root cause was an explicit no-op in the `Blocked` branch:

```go
_ = f.Blocked  // previous code — intentional no-op
```

The fix required changes across three files:

1. **`internal/task/task.go`** — Added a `Blocked bool \`yaml:"-"\`` derived
   field to the `Task` struct so the in-memory path has something to check.
2. **`internal/task/store.go`** — Updated `loadAll` to set `t.Blocked =
   IsBlocked(t, planTasks)` for each task after loading a plan group.
3. **`internal/task/filter.go`** — Replaced the `_ = f.Blocked` no-op with an
   actual check against `t.Blocked`.

Three tests were added to `internal/task/filter_test.go` to cover the
in-memory path:

- `TestApply_BlockedFilter_InMemory` — a task with `Blocked: true` is returned
- `TestApply_BlockedFilter_False_MatchesAll` — `Blocked: false` means no constraint
- `TestApply_BlockedFilter_NoBlockedTasks_ReturnsEmpty` — filter returns empty
  when nothing is blocked

## How It Was Done

1. Read `filter.go` to understand the two code paths: `matchesFilter`
   (in-memory, used by `Apply`) and `matchesJSONFilter` (index-based, used by
   `ApplyToJSON`). Confirmed the JSON path already worked correctly via
   `e.Blocked`, which the store sets during `RebuildTaskIndex`.
2. Read `store.go:loadAll` and `IsBlocked` to understand how blocked status is
   computed. `IsBlocked` walks `t.DependsOn` and checks each seq's status
   against the sibling task list — it is already correct and reusable.
3. Read `task.go:Task` to confirm `Blocked` was not present as a struct field.
   The `ToJSON()` method hardcoded `Blocked: false` with a comment saying "store
   sets this during loadAll" — confirming the intent was always to pre-compute
   it, just never wired up for the in-memory path.
4. Added `Blocked bool \`yaml:"-"\`` to `Task` (derived field, not persisted).
5. Replaced the no-op loop in `loadAll` with `t.Blocked = IsBlocked(t,
   planTasks)`.
6. Replaced `_ = f.Blocked` in `matchesFilter` with:

```go
if f.Blocked {
    if !t.Blocked {
        return false
    }
}
```

7. Wrote three tests, ran `go test ./internal/task/... -v` to confirm they all
   passed, then ran `go test ./...` to verify no regressions.

## Gotchas & Lessons Learned

**The no-op was intentional at the time of writing.** The original comment
explained the reasoning: `matchesFilter` works on in-memory `*Task` values
that "don't carry the full task set", so computing `IsBlocked` inline would
require passing `planTasks` into every `Apply` call. The comment concluded that
the blocked filter was "for now" only surfaced through the JSON path. The fix
avoids the API change by instead pre-computing the flag in `loadAll` — the
same approach the JSON path uses.

**`Apply` and `ApplyToJSON` are now symmetric.** Before the fix, the two
filter paths had different behaviour for `--blocked`. After the fix, both read
a pre-computed `Blocked` field (on `Task` and `TaskJSON` respectively), so the
logic is consistent and testable independently.

**`ToJSON()` already set `Blocked: false` unconditionally.** After adding the
`Blocked` field to `Task`, `ToJSON()` should propagate `t.Blocked` rather than
hardcoding `false`. This was not changed in this task because the `RebuildTaskIndex`
path overwrites `entry.Blocked` via `IsBlocked` anyway — but it is a latent
inconsistency worth cleaning up if `ToJSON` is ever called outside the index
rebuild.

## Reusable Patterns

When a filter operates on in-memory structs that don't carry all the data
needed to evaluate a predicate, the cleanest fix is to **pre-compute the
derived flag at load time** rather than plumbing extra context through the
filter API. This keeps `Apply`'s signature stable and mirrors how the index
path already works.

Pattern: computed-at-load derived fields on domain structs.

```go
// In the struct definition — yaml:"-" ensures it is never written to disk.
Blocked bool `yaml:"-"`

// In loadAll, after grouping tasks by plan:
for _, t := range planTasks {
    t.Blocked = IsBlocked(t, planTasks)
}

// In matchesFilter — simple boolean check, no extra arguments needed.
if f.Blocked {
    if !t.Blocked {
        return false
    }
}
```

---
## Walkthrough: 001 Fix plan.Marshal to preserve body content

# Walkthrough: Fix plan.Marshal to preserve body content

## What Was Done

Fixed a data-loss bug in `pkg/plan/plan.go:Marshal` where the plan body was
silently discarded on every write. The function previously serialised only the
YAML frontmatter, so any call that rewrote an existing plan file — most
critically `logos distill` setting `Distilled: true` — would overwrite the
file with frontmatter-only, destroying the body the agent had written.

The fix appends `p.Body` after the closing `---` when it is non-empty,
inserting a leading newline if the body does not already start with one. The
scaffold path (`logos save`, where `p.Body == ""`) is unaffected.

Three regression tests were added to `pkg/plan/plan_test.go`:

- `TestMarshal_PreservesBodyContent` — full Marshal → Parse round-trip with a
  non-empty body
- `TestMarshal_EmptyBody_NoTrailingContent` — confirms the scaffold path still
  ends with `---`
- `TestMarshal_BodyPreservedAfterDistilledUpdate` — directly reproduces the
  `logos distill` scenario: set `Distilled = true`, call `Marshal`, re-parse,
  assert body is intact

## How It Was Done

1. Read `pkg/plan/plan.go` to understand the existing `Marshal` and `Write`
   functions.
2. Compared with `internal/task/task.go:Marshal`, which already handled body
   correctly — used it as the reference implementation as called out in the
   task spec.
3. Added the body-append block to `plan.Marshal`:

```go
if p.Body != "" {
    if !strings.HasPrefix(p.Body, "\n") {
        buf.WriteByte('\n')
    }
    buf.WriteString(p.Body)
}
```

4. Updated the doc comment to clarify that `Marshal` serves both the scaffold
   path (body empty) and the rewrite path (body present).
5. Wrote the three tests, ran `go test ./pkg/plan/... -v` to confirm they all
   passed, then ran `go test ./...` to verify no regressions elsewhere.

## Gotchas & Lessons Learned

**The bug was invisible in normal usage.** `logos save` never sets `p.Body`
before calling `Write`, so the scaffold path always wrote correctly. The bug
only triggered on rewrite paths (`logos distill`) where an already-populated
`Plan` was re-serialised. Without a round-trip test that includes a body, this
class of bug is easy to miss.

**The task package had the correct implementation all along.** `task.Marshal`
in `internal/task/task.go` already appended the body. The two `Marshal`
functions were written independently and diverged on this detail. This is also
the motivation for task 004 (extract shared markdown helpers), which would
prevent the two implementations from drifting again.

**Newline normalisation matters.** Bodies parsed by `splitFrontmatter` always
start with a newline stripped (the parser consumes the `\n` after the closing
`---`). Storing the body as-is and then checking for a leading newline on
write ensures the round-trip is lossless regardless of how the body was
originally stored.

## Reusable Patterns

When a struct has both frontmatter and a free-form body, the canonical
serialisation pattern in this codebase is:

```go
var buf bytes.Buffer
buf.WriteString("---\n")
buf.Write(fm) // yaml.Marshal output
buf.WriteString("---\n")
if t.Body != "" {
    if !strings.HasPrefix(t.Body, "\n") {
        buf.WriteByte('\n')
    }
    buf.WriteString(t.Body)
}
return buf.Bytes(), nil
```

The guard `!strings.HasPrefix(t.Body, "\n")` ensures exactly one blank line
between the closing `---` and the body content, making the output consistent
whether or not the stored body already carries a leading newline.
-->

## Summary

Pure code review of the logosyncx codebase surfaced 3 runtime bugs and 6 code quality issues.
All 9 tasks were implemented across two sessions. The bugs affected core workflows; quality issues
were refactors and additions to improve correctness and agent ergonomics.

## Implemented Features

**Bug fixes (runtime impact):**
- `plan.Marshal` now preserves body content — `logos distill` no longer silently destroys plan bodies
- `--blocked` filter in `logos task ls` (non-JSON path) now works — was a silent no-op
- `UpdateFields` validates status/priority before writing — invalid values no longer corrupt TASK.md

**Code quality:**
- Shared markdown helpers extracted to `internal/markdown` — eliminated duplication between `pkg/plan` and `internal/task`
- `relPath` now uses `filepath.Rel` instead of fragile `strings.TrimPrefix`
- `sortByDateDesc` replaced with `slices.SortFunc` + `time.Time.Compare`
- `TaskJSON.CanStart` field added — `true` when `status == open && !blocked`; in table (`✓`) and `--json`
- `AutoPush` naming left as-is after Task 001 made it actually commit+push (obsolete rename)

## Key Specification

- `plan.Marshal` must append `p.Body` after `---` when non-empty; scaffold path (`Body == ""`) is unchanged
- `Task.Blocked` is a derived `yaml:"-"` field set in `loadAll` via `IsBlocked`; `matchesFilter` checks it
- Validation guard clauses in `UpdateFields` must come before any state-transition logic
- `CanStart` and `Blocked` in `TaskJSON` are always `false` in `ToJSON()`; `RebuildTaskIndex` sets them after dependency resolution

## Key Learnings

- **Pre-compute derived flags at load time** rather than plumbing context through filter APIs. Both `Blocked` and `CanStart` follow this pattern: computed once in `loadAll`/`RebuildTaskIndex`, consumed cheaply in filter/table.
- **Bugs invisible in normal usage**: `plan.Marshal` body loss only triggered on rewrite paths (`logos distill`), not `logos save`. Without round-trip tests that include body content, this class of bug is easy to miss.
- **Task ordering matters**: implementing auto commit/push (Task 001) before the AutoPush rename (Task 007) made the rename unnecessary.
- **Enforcement cascades**: adding a guard to a core transition function broke many test helpers that used it as setup. Any enforcement at a central chokepoint requires updating all callers.

## Reusable Patterns

**Derived field pattern (computed at load, not persisted):**
```go
Blocked bool `yaml:"-"`

for _, t := range planTasks {
    t.Blocked = IsBlocked(t, planTasks)
}

if f.Blocked && !t.Blocked {
    return false
}
```

**Marshal with optional body:**
```go
buf.WriteString("---\n")
if p.Body != "" {
    if !strings.HasPrefix(p.Body, "\n") {
        buf.WriteByte('\n')
    }
    buf.WriteString(p.Body)
}
```

**Newest-first sort:**
```go
slices.SortFunc(items, func(a, b T) int {
    return b.Date.Compare(a.Date)
})
```

**Walkthrough content check (scaffold vs. real):**
```go
func walkthroughHasContent(path string) bool {
    data, _ := os.ReadFile(path)
    for _, line := range strings.Split(string(data), "\n") {
        t := strings.TrimSpace(line)
        if t != "" && !strings.HasPrefix(t, "<!--") {
            return true
        }
    }
    return false
}
```

**Shared helper extraction — push caller-specific defaults to the caller:**
```go
section := opts.ExcerptSection
if section == "" {
    section = "Background" // "What" in task package
}
excerpt = markdown.ExtractExcerpt(body, section)
```

## Gotchas

- `replace_all` on short strings (e.g. `slugify(`) also hits the function definition — use longer unique context
- `ToJSON()` hardcodes `Blocked: false` — `RebuildTaskIndex` overwrites it, but in-memory `List` → `ToJSON` path does not set `Blocked`/`CanStart` correctly (pre-existing latent inconsistency)
- `store.go` still needs `"sort"` for `sort.Slice` in `loadPlanTasks` even after replacing `sortByDateDesc`

## Source Walkthroughs

- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/009-require-walkthrough-content-before-marking-task-done/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/008-add-can_start-field-to-task-ls-output/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/007-fix-autopush-config-naming-and-comment/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/006-replace-sortbydatedesc-with-slicessortfunc/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/005-fix-relpath-to-use-filepathrel/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/004-extract-shared-markdown-helpers-to-internal-package/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/003-add-statuspriority-validation-in-updatefields/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/002-fix-blocked-filter-no-op-in-matchesfilter/WALKTHROUGH.md
- .logosyncx/tasks/20260307-fix-bugs-and-code-quality-issues-found-in-pure-review/001-fix-planmarshal-to-preserve-body-content/WALKTHROUGH.md
