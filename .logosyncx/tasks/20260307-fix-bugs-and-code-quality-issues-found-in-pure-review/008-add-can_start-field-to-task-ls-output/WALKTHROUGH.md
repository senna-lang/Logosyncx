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
