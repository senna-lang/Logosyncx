---
id: t-bds6z9
date: 2026-02-21T10:42:59.065657+09:00
title: "[Phase2] Implement logos sync command"
status: open
priority: medium
session: ""
tags:
    - phase2
    - cli
    - sync
assignee: ""
---

## What

Implement `logos sync` to scan `sessions/` and rebuild `index.jsonl` from the
current state of the filesystem.

Behaviour:
- Walk every `.md` file under `.logosyncx/sessions/`
- Parse frontmatter and extract excerpt from each file
- Overwrite `.logosyncx/index.jsonl` with the rebuilt entries
- Warn if a session file is missing sections listed in `config.json`'s
  `summary_sections`
- Warn if `.md` files appear to have been added manually (no matching index
  entry existed before the rebuild)
- Run `git add` on the updated `index.jsonl` via go-git

## Why

`logos save` keeps `index.jsonl` up to date automatically, but the index can
drift from the filesystem when session files are edited, renamed, or added
manually outside of the CLI. `logos sync` acts as a repair tool that agents
and users can run to guarantee consistency between the files and the index that
`logos ls --json` reads from.

## Scope

- `cmd/sync.go` — new file implementing the `sync` cobra command
- `pkg/index/` — add `Rebuild(root string) (int, error)` function (or extend
  existing index helpers) that re-scans `sessions/` and rewrites `index.jsonl`
- `cmd/sync_test.go` — integration tests using a temp directory with pre-seeded
  session files
- `USAGE.md` template in `cmd/init.go` — document `logos sync`

## Checklist

- [ ] Implement `Rebuild` in `pkg/index/`
- [ ] Implement `cmd/sync.go` calling `Rebuild` then `git add index.jsonl`
- [ ] Warn on sessions missing `summary_sections` headers
- [ ] Warn on sessions not previously in the index (manually added files)
- [ ] Write integration tests: empty dir, single file, multiple files, missing section
- [ ] Run `go test ./...` and confirm all tests pass

## Notes

Migrated from beads issue `logosyncx-6z9`.

`logos sync` is complementary to `logos save` — save keeps the index current
during normal use, sync repairs it after out-of-band changes. Both commands
should produce an identical `index.jsonl` when run on the same set of files.