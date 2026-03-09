# Walkthrough: Include WALKTHROUGH.md path in done-transition error message

## Key Specification

Error message when `logos task update --status done` is rejected due to missing/scaffold-only
WALKTHROUGH.md must include the exact relative path the agent needs to write to.

## What Was Done

- `internal/task/store.go` `UpdateFields`: In the `newStatus == StatusDone && t.Status != StatusDone` branch,
  compute `relWPath` via `filepath.Rel(s.projectRoot, wPath)`, falling back to absolute `wPath` if Rel fails.
  Changed error format from:
    `"WALKTHROUGH.md has no content: write WALKTHROUGH.md content first, then re-run"`
  to:
    `"WALKTHROUGH.md has no content: write content to\n  %s\nthen re-run"` (with relative path)
- Added `TestStore_UpdateFields_Done_ErrorIncludesWalkthroughPath` test verifying:
  - Error contains `.logosyncx` (relative path indicator)
  - Error contains `WALKTHROUGH.md`
  - Error does NOT contain the absolute temp dir path

## How It Was Done

Existing `wPath` variable (`filepath.Join(t.DirPath, walkthroughFileName)`) was already computed
before the `walkthroughHasContent` check. Added `filepath.Rel(s.projectRoot, wPath)` to get
the relative path, then embedded it in the error string.

## Gotchas & Lessons Learned

- `filepath.Rel` can fail if the paths are on different volumes (Windows) — fallback to absolute
  ensures robustness.
- Existing tests only checked for `"WALKTHROUGH.md"` presence in error, so no test updates were
  needed for backward compatibility. Added a new test for the path inclusion requirement.

## Reusable Patterns

```go
relPath, relErr := filepath.Rel(s.projectRoot, absPath)
if relErr != nil {
    relPath = absPath // fallback to absolute
}
return fmt.Errorf("...: write content to\n  %s\nthen re-run", relPath)
```
