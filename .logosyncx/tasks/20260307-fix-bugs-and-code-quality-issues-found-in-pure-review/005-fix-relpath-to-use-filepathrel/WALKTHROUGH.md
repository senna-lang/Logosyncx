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
