# Walkthrough: Auto-run logos sync after logos save and logos task update

## Key Specification

After `logos save`, `logos ls --json` must return the new plan without requiring `logos sync`.
After `logos task update`/create, `logos task ls --json` must reflect changes without `logos sync`.

## What Was Done

- `cmd/save.go`: Replaced `index.Append(root, entry)` with `index.Rebuild(root, cfg.Plans.ExcerptSection)`.
  Added `config.Load(root)` call to obtain `cfg.Plans.ExcerptSection` for excerpt extraction.
  Removed the `plan.LoadFile(savedPath)` + `index.FromPlan` code that was only needed for Append.
- `internal/task/store.go` `Create`: Replaced `AppendTaskIndex` with `s.RebuildTaskIndex()` for
  full task index consistency after task creation.

## How It Was Done

Examined `cmd/save.go`, `cmd/task.go`, `cmd/sync.go`, and `pkg/index/index.go`.
`logos sync` uses `index.Rebuild` (plan) + `RebuildTaskIndex` (task).
`logos save` was using `index.Append` (partial) — switched to `index.Rebuild`.
`store.Create` was using `AppendTaskIndex` (partial) — switched to `RebuildTaskIndex`.
`store.UpdateFields` already used `RebuildTaskIndex` — no change needed there.

## Gotchas & Lessons Learned

- `index.Rebuild` requires `excerptSection` from config; had to add `config.Load` to `cmd/save.go`.
- The old `Append` path required loading the saved plan from disk and constructing an Entry manually.
  Switching to `Rebuild` simplified the code (fewer lines, no manual Entry construction).
- `RebuildTaskIndex` in `Create` is slightly heavier than `Append` but ensures consistency,
  which is preferable for a dev CLI tool where correctness > raw performance.

## Reusable Patterns

Use full `Rebuild`/`RebuildTaskIndex` after write operations instead of partial `Append` —
ensures consistency at the cost of a linear scan, acceptable for local dev tooling.
