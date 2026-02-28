---
id: 15a93a
date: 2026-02-28T11:12:45.228451+09:00
topic: Implement bidirectional session-task cross-linking
tags:
    - go
    - sessions
    - tasks
    - frontmatter
agent: claude-code
related: []
---

## Summary

Designed and implemented bidirectional cross-linking between sessions and tasks. Added Tasks field to Session struct, Sessions and Related fields to Task struct, updated index.Entry, added AppendSession method to Store, added --task flag to logos save, and --add-session flag to logos task update. logos task refer --with-session now shows all linked sessions from the Sessions list.

## Key Decisions

- Session.Tasks (no omitempty, consistent with Related) stores session→task links\n- Task.Sessions (omitempty) and Task.Related (omitempty) added to avoid cluttering existing task files on re-marshal\n- Task.Session string kept for backward compat; AppendSession also writes first entry to Session\n- logos task create --session writes to both Session and Sessions for forward compat\n- No referential integrity checks: missing linked file is silent (informational links only)\n- New fields are yaml zero-value safe: existing files without them parse as empty slices, no migration needed

## Context Used

- Design task: 2026-02-28_design-and-implement-bidirectional-sessiontask-cross-linking.md

