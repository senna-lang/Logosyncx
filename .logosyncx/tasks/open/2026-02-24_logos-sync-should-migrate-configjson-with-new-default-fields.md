---
id: t-de9b87
date: 2026-02-24T20:33:24.96172+09:00
title: logos sync should migrate config.json with new default fields
status: open
priority: medium
session: ""
tags:
    - cli
    - sync
    - config
assignee: ""
---

## What

applyDefaults fills missing fields in memory at Load time but never writes them back to disk. Existing projects that were initialized before new config fields were added (e.g. save.excerpt_section, save.sections, tasks.sections) will not see those fields in their config.json unless they manually edit the file.

## What
Add a migration step to `logos sync` that detects missing config fields and writes them back to disk with default values.

## Why
When new config fields are introduced (e.g. save.sections, tasks.sections added in feat: replace template.md with config.json sections), existing projects silently use in-memory defaults but their config.json stays stale. Users must manually update config.json, which is error-prone and not obvious.

## Scope
- `pkg/config/config.go`: add `Migrate(projectRoot string) (changed bool, err error)` that loads config, applies defaults, and writes back only if fields were missing
- `cmd/sync.go`: call `config.Migrate(root)` at the start of `runSync()`, print a message if config was updated
- Add tests for the migration logic

## Checklist
- [ ] Implement `config.Migrate()`
- [ ] Call from `logos sync`
- [ ] Print a message when config.json is updated (e.g. `✓ config.json migrated with new default fields`)
- [ ] go test ./... passes
