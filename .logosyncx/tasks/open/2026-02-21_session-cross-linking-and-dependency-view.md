---
id: t-bdsuff
date: 2026-02-21T10:43:20.307170+09:00
title: "[Phase3] Session cross-linking and dependency view"
status: open
priority: medium
session: ""
tags:
    - phase3
    - cli
    - graph
    - sessions
assignee: ""
---

## What

Visualize and navigate the session reference graph based on the `related` field
in session frontmatter.

- When running `logos refer <name>`, display a "Related sessions" section at the
  bottom listing all sessions referenced in the `related` frontmatter field
- Add a `logos graph` (or `logos refer <name> --graph`) subcommand that prints
  a text-based dependency tree of the session reference graph
- Optionally warn when a `related` entry points to a session file that no longer
  exists

## Why

Sessions frequently build on prior work — a design session references an earlier
architecture decision, an implementation session references a design session.
The `related` field already captures these links in the frontmatter, but the CLI
currently makes no use of them. Surfacing the graph lets agents and humans
quickly navigate the full context chain without manually looking up each
referenced session.

## Scope

- `cmd/refer.go` — append a "Related sessions" block when printing a session
  that has a non-empty `related` field
- `cmd/graph.go` (new) — implement `logos graph` to print the full reference
  graph as an ASCII tree, starting from a given session or from all sessions
- `pkg/session/session.go` — confirm `Related []string` field exists in the
  `Session` struct (add if missing)
- `pkg/index/` — add a helper to resolve session filenames from partial names
  (reuse existing partial-match logic if available)
- `cmd/graph_test.go` — integration tests using temp session files with
  cross-references

## Checklist

- [ ] Confirm `Related []string` is present in the `Session` struct
- [ ] `logos refer <name>` prints related session titles/filenames at the bottom
- [ ] Warn when a `related` entry references a non-existent session file
- [ ] Implement `logos graph` — ASCII tree from a root session
- [ ] Implement `logos graph --all` — print the full reference graph for all sessions
- [ ] Handle cycles in the reference graph gracefully (detect and stop recursion)
- [ ] Write integration tests for the related-display and graph commands
- [ ] Run `go test ./...` and confirm all tests pass

## Notes

Migrated from beads issue `logosyncx-uff`.

The `related` field in session frontmatter is a list of session filenames
(e.g. `related: ["2026-02-20_architecture.md", "2026-02-19_design.md"]`).
Cross-linking should use exact filename matching first, then fall back to
partial matching (same logic as `logos refer`).

For the ASCII tree, a simple indented format is sufficient:
```
2026-02-21_implementation.md
  └─ 2026-02-20_design.md
       └─ 2026-02-19_requirements.md
```
