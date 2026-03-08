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
