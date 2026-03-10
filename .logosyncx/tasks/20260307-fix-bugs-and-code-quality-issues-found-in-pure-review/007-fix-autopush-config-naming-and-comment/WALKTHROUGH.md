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
