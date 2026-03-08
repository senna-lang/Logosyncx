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

<!-- SOURCE MATERIAL — read this, fill in the sections below, then remove this block. -->
<!--
## Plan: Add walkthrough template

## Background

Plans, tasks, and knowledge files all have customizable templates in `.logosyncx/templates/`. Walkthroughs (`WALKTHROUGH.md`) are the only document type without a template — their scaffold content is hardcoded in `internal/task/store.go`. Teams cannot customize walkthrough structure without modifying source code.

## Spec

Add `walkthrough.md` to the templates system:

1. Add `defaultWalkthroughTemplate` constant to `cmd/init.go` and register it in the `logos init` templates map.
2. Refactor `CreateWalkthroughScaffold()` in `internal/task/store.go` to read from `.logosyncx/templates/walkthrough.md`, falling back to hardcoded defaults if the file is missing.
3. Add tests for template-driven and fallback scaffold creation.
4. Create `.logosyncx/templates/walkthrough.md` in this repo immediately.
5. Update `usageMD` in `cmd/init.go` and `.logosyncx/USAGE.md` to mention reading the walkthrough template.

## Key Decisions

Decision: Fall back to hardcoded content when template file is missing. Rationale: Backwards compatibility — existing repos without `templates/walkthrough.md` should continue to work.

Decision: Template contains sections only (no title header). Rationale: Consistent with `task.md` and `plan.md` which also have no title line; the title is prepended by code as `# Walkthrough: <task title>`.

## Notes

Existing walkthroughs are unaffected. Only newly created scaffolds use the template.

---
## Walkthrough: 001 Add walkthrough template support

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
-->

## Summary

## Implemented Features

## Key Specification

## Key Learnings

## Reusable Patterns

## Gotchas

## Source Walkthroughs

- .logosyncx/tasks/20260308-add-walkthrough-template/001-add-walkthrough-template-support/WALKTHROUGH.md
