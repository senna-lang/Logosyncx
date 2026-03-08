---
id: t-dace55
date: 2026-03-08T20:11:50.014897+09:00
title: Add walkthrough template support
seq: 1
status: done
priority: high
plan: 20260308-add-walkthrough-template
tags:
    - feature
    - template
assignee: ""
completed_at: 2026-03-08T20:15:29.944287+09:00
---

## What

Add `walkthrough.md` to the logos template system so teams can customize walkthrough section structure. `CreateWalkthroughScaffold()` reads from `.logosyncx/templates/walkthrough.md` (with hardcoded fallback), and `logos init` creates the file on new project setup.

## Why

Every other document type (plan, task, knowledge) has a customizable template. Walkthroughs are the only exception — their scaffold is hardcoded, so teams cannot adapt the structure without forking the codebase.

## Scope

- `cmd/init.go` — add `defaultWalkthroughTemplate` constant; register in templates map; update `usageMD`
- `internal/task/store.go` — refactor `CreateWalkthroughScaffold()` to read template; add `readWalkthroughTemplate()` helper
- `internal/task/store_test.go` — add 2 tests (template-driven, fallback)
- `.logosyncx/templates/walkthrough.md` — create in this repo
- `.logosyncx/USAGE.md` — update walkthrough workflow section

Out of scope: changing existing WALKTHROUGH.md files, modifying distill logic.

## Checklist

- [ ] `defaultWalkthroughTemplate` added to `cmd/init.go`
- [ ] `"walkthrough.md"` added to templates map in `logos init`
- [ ] `usageMD` updated to mention reading walkthrough template
- [ ] `CreateWalkthroughScaffold()` reads from template file with fallback
- [ ] `readWalkthroughTemplate()` helper added to `store.go`
- [ ] Tests added: `TestCreateWalkthroughScaffold_UsesTemplate` and `TestCreateWalkthroughScaffold_FallsBackWithoutTemplate`
- [ ] `go test ./...` passes
- [ ] `.logosyncx/templates/walkthrough.md` created in this repo
- [ ] `.logosyncx/USAGE.md` updated

## Notes

Fallback content must match the current hardcoded sections exactly to avoid breaking existing repos without the template file.
