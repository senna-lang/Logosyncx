# Walkthrough: Add walkthrough template support

<!-- Auto-generated when this task was marked done. -->
<!-- Fill in each section before running logos distill. -->

## What Was Done

Added `walkthrough.md` to the logos template system. `CreateWalkthroughScaffold()` now reads from `.logosyncx/templates/walkthrough.md` and falls back to hardcoded defaults when the file is missing. `logos init` creates the file on new project setup.

## How It Was Done

1. Added `defaultWalkthroughTemplate` constant to `cmd/init.go` and registered it in the templates map so `logos init` creates `templates/walkthrough.md`.
2. Added `defaultWalkthroughBody` fallback constant and `readWalkthroughTemplate()` helper to `internal/task/store.go`.
3. Refactored `CreateWalkthroughScaffold()` to build content as `header + s.readWalkthroughTemplate()` instead of one large hardcoded format string.
4. Added two tests: `TestCreateWalkthroughScaffold_UsesTemplate` and `TestCreateWalkthroughScaffold_FallsBackWithoutTemplate`.
5. Created `.logosyncx/templates/walkthrough.md` in this repo.
6. Updated `usageMD` in `cmd/init.go` and `.logosyncx/USAGE.md` to include "read walkthrough template" as step 2 of the task completion workflow.

## Gotchas & Lessons Learned

The `defaultWalkthroughBody` fallback in `store.go` must exactly match the sections in `defaultWalkthroughTemplate` in `cmd/init.go` — they're separate constants in separate packages. Keep them in sync when changing sections.

The existing `TestCreateWalkthroughScaffold_CreatesFile` test (no template file in temp dir) exercises the fallback path automatically, so all existing tests remain valid without modification.

## Reusable Patterns

Template-with-fallback pattern for scaffolds:
```go
func (s *Store) readXTemplate() string {
    p := filepath.Join(s.projectRoot, ".logosyncx", "templates", "x.md")
    data, err := os.ReadFile(p)
    if err != nil {
        return defaultXBody // package-level fallback constant
    }
    return string(data)
}
```
Register the template in `cmd/init.go` `defaultXTemplate` + templates map. The fallback constant lives in the `internal/` package for test independence.
