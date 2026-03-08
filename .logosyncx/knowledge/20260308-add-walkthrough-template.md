---
id: k-ae6324
date: 2026-03-08T14:50:42.490093Z
topic: Add walkthrough template
plan: 20260308-add-walkthrough-template.md
tasks:
    - 001-Add walkthrough template support
tags:
    - feature
    - template
---


## Summary

Added `walkthrough.md` to the logos template system. `CreateWalkthroughScaffold()` now reads from `.logosyncx/templates/walkthrough.md` and falls back to hardcoded defaults when the file is missing. `logos init` creates the file automatically on new project setup.

## Implemented Features

- `readWalkthroughTemplate()` helper in `internal/task/store.go` — reads `.logosyncx/templates/walkthrough.md`, falls back to `defaultWalkthroughBody` constant if missing.
- `CreateWalkthroughScaffold()` refactored to use `header + s.readWalkthroughTemplate()`.
- `logos init` now creates `templates/walkthrough.md` via the `defaultWalkthroughTemplate` constant in `cmd/init.go`.
- `.logosyncx/templates/walkthrough.md` added to this repo.

## Key Specification

All document types (plan, task, knowledge) had customizable templates in `.logosyncx/templates/`. Walkthroughs were the only exception — their scaffold was hardcoded in `store.go`. This prevented teams from customizing walkthrough structure without modifying source code. The fix adds `walkthrough.md` to the template system while maintaining backwards compatibility via a fallback constant.

## Key Learnings

- Template and fallback constants must be kept in sync across packages (`cmd/init.go` and `internal/task/store.go`). They are separate constants with no compile-time link — drift is a real risk.
- Existing tests that create a scaffold without a template file exercise the fallback path automatically. No test changes needed for the fallback coverage.

## Reusable Patterns

Template-with-fallback pattern for any scaffold type:

```go
// In internal/<pkg>/store.go
func (s *Store) readXTemplate() string {
    p := filepath.Join(s.projectRoot, ".logosyncx", "templates", "x.md")
    data, err := os.ReadFile(p)
    if err != nil {
        return defaultXBody // fallback constant in same package
    }
    return string(data)
}
```

Register the template in `cmd/init.go`:
```go
const defaultXTemplate = `## Section\n\n<!-- ... -->\n`

// in initTemplates map:
"x.md": defaultXTemplate,
```

The fallback constant lives in the `internal/` package so tests remain independent of the project filesystem.

## Gotchas

- `defaultWalkthroughBody` (in `store.go`) and `defaultWalkthroughTemplate` (in `cmd/init.go`) are separate constants in separate packages. Any section change must be applied to both.
- Template file contains sections only — no title header. The title (`# Walkthrough: <task title>`) is prepended by `CreateWalkthroughScaffold()` in code, consistent with how `plan.md` and `task.md` templates work.

## Source Walkthroughs

- .logosyncx/tasks/20260308-add-walkthrough-template/001-add-walkthrough-template-support/WALKTHROUGH.md
