---
id: t-bdszo8
date: 2026-02-21T10:43:20.307227+09:00
title: "[Phase3] Auto-update knowledge/ directory"
status: open
priority: medium
session: ""
tags:
    - phase3
    - knowledge
    - automation
assignee: ""
---

## What

On `logos save` or `logos sync`, automatically extract and update distilled
knowledge files under `.logosyncx/knowledge/`:

- `architecture.md` — high-level system design decisions
- `decisions.md` — key decisions log (append-only)
- `conventions.md` — coding conventions and team agreements

Extraction must be rule-based (no LLM calls): scan the relevant sections of
each saved session (e.g. `## Key Decisions`, `## Architecture`) and append
new entries to the corresponding knowledge file, deduplicating by content hash.

## Why

Over time, sessions accumulate a large amount of context. Agents reading
individual sessions to reconstruct project-wide decisions or conventions face
a growing token cost. Maintaining distilled knowledge files gives agents a
cheap, stable reference that doesn't grow proportionally with the number of
sessions.

The rule-based approach (extract named sections, append, dedup) keeps the
implementation simple and deterministic — no embedding API or LLM call is
needed.

## Scope

- `.logosyncx/knowledge/` — directory created by `logos init` (extend init)
- `internal/knowledge/extractor.go` — rule-based section extraction from a
  session markdown file; produces `[]Entry{Section, Content, SourceSession}`
- `internal/knowledge/store.go` — append `Entry` values to the target
  knowledge file, skip duplicates (SHA-256 of trimmed content)
- `cmd/save.go` — call knowledge extraction + update after a successful save
- `cmd/sync.go` — call knowledge rebuild (re-extract from all sessions) when
  `--rebuild-knowledge` flag is passed
- `cmd/init.go` — create `.logosyncx/knowledge/` and seed empty
  `architecture.md`, `decisions.md`, `conventions.md`
- `cmd/init_test.go` / `cmd/save_test.go` — extend existing tests

## Checklist

- [ ] Extend `logos init` to create `.logosyncx/knowledge/` with three seed files
- [ ] Implement `internal/knowledge/extractor.go` — extract named sections from a session
- [ ] Implement `internal/knowledge/store.go` — append + dedup logic
- [ ] Integrate extraction into `cmd/save.go` (run after successful session write)
- [ ] Add `--rebuild-knowledge` flag to `logos sync` for full rebuild from all sessions
- [ ] Write unit tests for extractor (section found, section missing, multiple sections)
- [ ] Write unit tests for store (append, dedup by hash)
- [ ] Run `go test ./...` and confirm all tests pass

## Notes

Migrated from beads issue `logosyncx-zo8`.

The section names to extract from are driven by `config.json`'s
`save.summary_sections` (already configurable). The mapping from section name
to knowledge file (e.g. `Key Decisions` → `decisions.md`) should be defined
in a new `knowledge.mappings` field in `config.json` with sensible defaults.

Deduplication should be based on a SHA-256 of the trimmed section content so
that minor whitespace changes don't create duplicate entries.