# Walkthrough: Extract shared markdown helpers to internal package

<!-- Auto-generated when this task was marked done. -->

## Key Specification

Five helper functions (`slugify`, `splitFrontmatter`, `extractExcerpt`, `parseHeading`, `truncateRunes`) were duplicated verbatim between `pkg/plan/plan.go` and `internal/task/task.go`, with only one behavioural difference: the default excerpt section name ("Background" vs "What"). The task was pure extraction — no behaviour changes.

## What Was Done

- Created `internal/markdown/markdown.go` with exported versions: `Slugify`, `SplitFrontmatter`, `ExtractExcerpt`, `ParseHeading`, `TruncateRunes`, and the `ExcerptMaxRunes` constant.
- Created `internal/markdown/markdown_test.go` covering all five functions.
- Removed local private implementations from both `pkg/plan/plan.go` and `internal/task/task.go`.
- Updated both packages to import `internal/markdown` and use the exported names.
- Fixed test files (`plan_test.go`, `task_test.go`, `index_test.go`) to use `markdown.*` calls.
- Removed now-unused `excerptMaxRunes` constants from both packages.
- The "Background" vs "What" default difference is resolved by having each caller set its own default before passing to `ExtractExcerpt`.

## How It Was Done

1. Read both source files side-by-side to confirm the functions were identical (modulo default section name).
2. Created the new package with exported names and no hard-coded defaults in `ExtractExcerpt`.
3. Updated callers one package at a time (`pkg/plan` first, then `internal/task`), running `go build` after each to surface errors.
4. The `replace_all` on `slugify(` accidentally replaced the function definition line — caught by the build error and fixed manually.
5. Ran `make fmt` to satisfy the gofmt pre-commit hook before committing.

## Gotchas & Lessons Learned

`replace_all` on short strings like `slugify(` is risky — it also hit the function definition `func slugify(`. Use longer unique context strings or replace the definition separately.

`errors` and `unicode/utf8` imports became unused after removing the local helpers — Go's compiler catches these immediately.

## Reusable Patterns

```go
// Caller sets own default before delegating to shared helper
section := opts.ExcerptSection
if section == "" {
    section = "Background" // or "What" in task package
}
p.Excerpt = markdown.ExtractExcerpt(body, section)
```

When extracting shared code between packages with minor behavioural differences, push the difference to the caller rather than adding parameters or flags to the shared function.
