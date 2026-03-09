# Walkthrough: Auto git commit and push when marking task done with auto_push enabled

## Key Specification

Task required: when `logos task update --status done` succeeds and `cfg.Git.AutoPush` is true,
automatically run `git commit` and `git push` so the agent does not need separate git commands.
Commit+push failures must be non-fatal (warn to stderr, task still marked done).

## What Was Done

- Added `transitionedToDone bool` flag in `UpdateFields` to track the open→done status transition.
- Set `transitionedToDone = true` inside the `newStatus == StatusDone && t.Status != StatusDone` branch.
- After all git adds and index rebuild, added a block: if `transitionedToDone && s.cfg.Git.AutoPush`,
  call `gitutil.Commit` with message `"logos: mark task done: <title>"`, then `gitutil.Push`.
  Both failures are non-fatal (print warning to stderr).
- `Commit` and `Push` helpers already existed in `internal/gitutil/gitutil.go` — no new helpers needed.
- Added two tests: one confirming UpdateFields succeeds even when git fails (non-real-repo temp dir),
  one confirming no git ops occur when auto_push=false.

## How It Was Done

Read `store.go` and `gitutil.go` to confirm `Commit`/`Push` were already implemented.
Added `transitionedToDone` variable before the fields loop, set it in the done-transition branch,
then used it after the existing git add + index rebuild block.

## Gotchas & Lessons Learned

- `transitionedToDone` is needed because after the loop, `t.Status == StatusDone` could be true
  even if the task was ALREADY done before the update (no transition occurred). The flag ensures
  commit+push only triggers on actual transitions.
- Tests run in `t.TempDir()` which is not a git repo, so commit fails — but that's the expected
  non-fatal path, confirmed by checking the test still passes.

## Reusable Patterns

Non-fatal git operation pattern:
```go
if err := gitutil.Commit(s.projectRoot, msg); err != nil {
    fmt.Fprintf(os.Stderr, "warning: git commit failed: %v\n", err)
} else {
    if err := gitutil.Push(s.projectRoot); err != nil {
        fmt.Fprintf(os.Stderr, "warning: git push failed: %v\n", err)
    }
}
```
