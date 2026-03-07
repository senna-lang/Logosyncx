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
