# Walkthrough: Replace sortByDateDesc with slices.SortFunc

## Key Specification

Replace hand-rolled insertion-sort `sortByDateDesc` in `cmd/ls.go` and
`internal/task/store.go` with `slices.SortFunc` from the standard library.

## What Was Done

- `cmd/ls.go`: `sortByDateDesc` replaced with `slices.SortFunc` using `b.Date.Compare(a.Date)`
  (already had `"slices"` imported)
- `internal/task/store.go`: same replacement; added `"slices"` import; removed `"sort"` is kept
  for `sort.Slice` used in `loadPlanTasks`

## How It Was Done

Used `time.Time.Compare` (Go 1.20+) which returns -1/0/1, making it directly usable as the
comparator for `slices.SortFunc`.

## Gotchas & Lessons Learned

`store.go` still needs `"sort"` for `sort.Slice` in `loadPlanTasks`. Only `sortByDateDesc`
was replaced, not all sorting.

## Reusable Patterns

```go
slices.SortFunc(items, func(a, b T) int {
    return b.Date.Compare(a.Date) // newest-first
})
```
