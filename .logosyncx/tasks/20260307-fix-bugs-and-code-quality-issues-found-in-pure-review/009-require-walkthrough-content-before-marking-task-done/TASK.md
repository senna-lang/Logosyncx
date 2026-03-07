---
id: t-271590
date: 2026-03-07T21:57:44.069206+09:00
title: Require WALKTHROUGH content before marking task done
seq: 9
status: open
priority: medium
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
---

## What

Change the workflow so that `logos task update --status done` requires WALKTHROUGH.md to already exist with real content. If WALKTHROUGH.md is missing or contains only the scaffold (HTML comments, no substantive text), the CLI returns an error. This forces agents to write the walkthrough with the Write tool before marking done, rather than after.

## Why

The current flow (mark done → scaffold generated → agent writes) allows agents to skip the write step with no immediate consequence. Reversing the order and enforcing content at the CLI layer closes the gap without adding shell escaping risk or extra tool calls. The agent's tooling (Write tool) stays unchanged; only the enforced order changes.

## Scope

- `internal/task/store.go` — `UpdateFields` status→done path: check WALKTHROUGH.md before accepting the transition
- `internal/task/store.go` — `CreateWalkthroughScaffold` is still called when WALKTHROUGH.md doesn't exist yet (scaffold creation stays, but done-transition now requires content)
- New helper: `walkthroughHasContent(path string) bool` — returns true if file exists and has at least one non-empty, non-comment line
- `internal/task/store_test.go` — test rejection when WALKTHROUGH missing, test rejection when scaffold-only, test acceptance when content present
- `cmd/task.go` or `CLAUDE.md` — update agent instructions: "Write WALKTHROUGH.md first, then mark done"
- `.logosyncx/USAGE.md` — update done workflow description

Out of scope: changing how scaffold is generated; changing distill pre-flight checks (those remain as a second safety net).

## Checklist

- [ ] Implement `walkthroughHasContent(path string) bool`: file must exist, have ≥1 line that is non-empty and not starting with `<!--`
- [ ] In `UpdateFields` status→done: call `walkthroughHasContent`; if false, return descriptive error
- [ ] Error message should tell the agent exactly what to do: `"write WALKTHROUGH.md content first, then re-run"`
- [ ] Add test: mark done with no WALKTHROUGH.md → error
- [ ] Add test: mark done with scaffold-only WALKTHROUGH.md → error
- [ ] Add test: mark done with content-filled WALKTHROUGH.md → success
- [ ] Update CLAUDE.md agent instructions for the new order
- [ ] Update `.logosyncx/USAGE.md`
- [ ] `go test ./...` passes

## Notes

Scaffold detection heuristic: a line is "content" if it is non-empty after trimming and does not start with `<!--`. This is simple and covers the auto-generated scaffold which consists entirely of HTML comment blocks.
