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
