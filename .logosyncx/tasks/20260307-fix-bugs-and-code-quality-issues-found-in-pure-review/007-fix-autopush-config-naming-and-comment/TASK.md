---
id: t-532ff1
date: 2026-03-07T21:31:11.282971+09:00
title: Fix AutoPush config naming and comment
seq: 7
status: open
priority: low
plan: 20260307-fix-bugs-and-code-quality-issues-found-in-pure-review
tags: []
assignee: ""
---

## What

Rename `GitConfig.AutoPush` to `AutoStage` (or `AutoAdd`) and fix the misleading comment that claims it runs `git commit` and `git push`. The field only controls `git add`; commit and push are always the user's responsibility.

## Why

The field name `auto_push` and its comment promise behaviour the code doesn't deliver. Any agent or user reading the config will expect auto-commit/push and be confused when it doesn't happen.

## Scope

- `pkg/config/config.go` — rename field, fix comment
- `pkg/config/config_test.go` — update references
- All `cfg.Git.AutoPush` call sites: `cmd/save.go`, `cmd/distill.go`, `internal/task/store.go`, `pkg/knowledge/knowledge.go`
- `.logosyncx/config.json` — rename key `auto_push` → `auto_stage`
- `cmd/init.go` (`usageMD` constant) and `.logosyncx/USAGE.md` if the field is documented there

Out of scope: actually implementing auto-commit/push.

## Checklist

- [ ] Rename `AutoPush` → `AutoStage` in `pkg/config/config.go` and fix comment
- [ ] Update all call sites (`cfg.Git.AutoPush` → `cfg.Git.AutoStage`)
- [ ] Update `.logosyncx/config.json` key
- [ ] Update USAGE.md and usageMD constant if referenced
- [ ] `go test ./...` passes

## Notes

The JSON key in config.json will change from `auto_push` to `auto_stage`. Existing configs with `auto_push` will silently ignore the old key after the rename — `applyDefaults` will set the new field to its zero value (false). This is acceptable since the default is false anyway.
