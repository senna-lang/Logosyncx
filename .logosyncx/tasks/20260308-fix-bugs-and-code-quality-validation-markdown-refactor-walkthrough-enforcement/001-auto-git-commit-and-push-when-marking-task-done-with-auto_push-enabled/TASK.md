---
id: t-286120
date: 2026-03-08T23:14:09.384284+09:00
title: Auto git commit and push when marking task done with auto_push enabled
seq: 1
status: done
priority: medium
plan: 20260308-fix-bugs-and-code-quality-validation-markdown-refactor-walkthrough-enforcement
tags: []
assignee: ""
completed_at: 2026-03-09T21:33:05.677678+09:00
---

## What

When `logos task update --status done` succeeds and `config.git.auto_push` is true, automatically run `git add`, `git commit`, and `git push` on the task directory so the agent does not need to issue separate git commands. If `auto_push` is false (default), behaviour is unchanged.

## Why

Git operations (add/commit/push) are the logos CLI's responsibility when `auto_push` is enabled. Currently `auto_push` only triggers `git add`; commit and push are left to the user, creating manual steps that break the agent workflow. Completing a task should be a single atomic operation that is immediately reflected in the remote.

## Scope

- `internal/task/store.go` — `UpdateFields` status→done path: after writing TASK.md and WALKTHROUGH.md, run `git commit` and `git push` when `cfg.Git.AutoPush` is true
- `internal/gitutil/` — add `Commit(root, message string) error` and `Push(root string) error` helpers
- `internal/task/store_test.go` — tests for the new behaviour

Out of scope: auto-commit on other status transitions; changing the `auto_push` field name (task 007).

## Acceptance Criteria

- [ ] `logos task update --name <task> --status done` with `auto_push: true` runs `git add <task-dir>`, `git commit -m "logos: mark task done: <title>"`, `git push`
- [ ] With `auto_push: false`, no git commit or push is performed (existing behaviour)
- [ ] Commit/push failures are non-fatal: print warning to stderr, task is still marked done
- [ ] `go test ./...` passes

## Checklist

- [ ] Add `Commit` and `Push` helpers to `internal/gitutil/`
- [ ] Call them in `UpdateFields` status→done path when `cfg.Git.AutoPush` is true
- [ ] Add unit tests (use temp git repo or stub)
- [ ] `go test ./...` passes

## Notes

Commit message format: `"logos: mark task done: <task-title>"`.
Push failures should warn, not fail — network is not always available.
