# Walkthrough: Fix plan.Marshal to preserve body content

## What Was Done

Fixed a data-loss bug in `pkg/plan/plan.go:Marshal` where the plan body was
silently discarded on every write. The function previously serialised only the
YAML frontmatter, so any call that rewrote an existing plan file — most
critically `logos distill` setting `Distilled: true` — would overwrite the
file with frontmatter-only, destroying the body the agent had written.

The fix appends `p.Body` after the closing `---` when it is non-empty,
inserting a leading newline if the body does not already start with one. The
scaffold path (`logos save`, where `p.Body == ""`) is unaffected.

Three regression tests were added to `pkg/plan/plan_test.go`:

- `TestMarshal_PreservesBodyContent` — full Marshal → Parse round-trip with a
  non-empty body
- `TestMarshal_EmptyBody_NoTrailingContent` — confirms the scaffold path still
  ends with `---`
- `TestMarshal_BodyPreservedAfterDistilledUpdate` — directly reproduces the
  `logos distill` scenario: set `Distilled = true`, call `Marshal`, re-parse,
  assert body is intact

## How It Was Done

1. Read `pkg/plan/plan.go` to understand the existing `Marshal` and `Write`
   functions.
2. Compared with `internal/task/task.go:Marshal`, which already handled body
   correctly — used it as the reference implementation as called out in the
   task spec.
3. Added the body-append block to `plan.Marshal`:

```go
if p.Body != "" {
    if !strings.HasPrefix(p.Body, "\n") {
        buf.WriteByte('\n')
    }
    buf.WriteString(p.Body)
}
```

4. Updated the doc comment to clarify that `Marshal` serves both the scaffold
   path (body empty) and the rewrite path (body present).
5. Wrote the three tests, ran `go test ./pkg/plan/... -v` to confirm they all
   passed, then ran `go test ./...` to verify no regressions elsewhere.

## Gotchas & Lessons Learned

**The bug was invisible in normal usage.** `logos save` never sets `p.Body`
before calling `Write`, so the scaffold path always wrote correctly. The bug
only triggered on rewrite paths (`logos distill`) where an already-populated
`Plan` was re-serialised. Without a round-trip test that includes a body, this
class of bug is easy to miss.

**The task package had the correct implementation all along.** `task.Marshal`
in `internal/task/task.go` already appended the body. The two `Marshal`
functions were written independently and diverged on this detail. This is also
the motivation for task 004 (extract shared markdown helpers), which would
prevent the two implementations from drifting again.

**Newline normalisation matters.** Bodies parsed by `splitFrontmatter` always
start with a newline stripped (the parser consumes the `\n` after the closing
`---`). Storing the body as-is and then checking for a leading newline on
write ensures the round-trip is lossless regardless of how the body was
originally stored.

## Reusable Patterns

When a struct has both frontmatter and a free-form body, the canonical
serialisation pattern in this codebase is:

```go
var buf bytes.Buffer
buf.WriteString("---\n")
buf.Write(fm) // yaml.Marshal output
buf.WriteString("---\n")
if t.Body != "" {
    if !strings.HasPrefix(t.Body, "\n") {
        buf.WriteByte('\n')
    }
    buf.WriteString(t.Body)
}
return buf.Bytes(), nil
```

The guard `!strings.HasPrefix(t.Body, "\n")` ensures exactly one blank line
between the closing `---` and the body content, making the output consistent
whether or not the stored body already carries a leading newline.