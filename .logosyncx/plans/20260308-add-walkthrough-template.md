---
id: 5e307c
topic: Add walkthrough template
tags:
    - feature
    - template
agent: claude-sonnet-4-6
related: []
tasks_dir: .logosyncx/tasks/20260308-add-walkthrough-template
distilled: true
---

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
