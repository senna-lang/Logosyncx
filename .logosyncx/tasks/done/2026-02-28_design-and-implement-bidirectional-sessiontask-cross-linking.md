---
id: t-a77699
date: 2026-02-28T10:56:16.282022+09:00
title: Design and implement bidirectional session↔task cross-linking
status: done
priority: high
session: ""
tags:
    - design
    - sessions
    - tasks
    - frontmatter
assignee: ""
---

## What

Add `tasks: []` to session frontmatter and `sessions: []` to task frontmatter, enabling bidirectional references between sessions and tasks. These fields are optional and empty by default.

## Frontmatter changes

### Session (pkg/session/session.go)
```yaml
---
id: abc123
date: 2026-02-28T...
topic: Something
tags: [go, cli]
agent: claude-code
related: []      # existing: session→session links (kept as-is)
tasks: []        # NEW: session→task links (list of task filenames)
---
```

### Task (internal/task/task.go)
```yaml
---
id: t-abc123
date: 2026-02-28T...
title: Do the thing
status: open
priority: medium
session: ""      # existing (kept for backward compat, single primary session)
sessions: []     # NEW: task→session links (list of session filenames, generalises session:)
related: []      # NEW: task→task links (optional dependency / follow-up chains)
tags: []
assignee: ""
---
```

## Backward compatibility rules
- `Task.Session string` is kept and never removed. When `Sessions` is empty and `Session` is non-empty, callers treat Session as the single element of Sessions.
- Existing session files without a `tasks:` field are read as `tasks: []` (yaml zero-value, no migration needed).
- Existing task files without a `sessions:` / `related:` field are read as empty slices (same reason).

## Why

Two planned future features depend on knowing which sessions belong to a task and vice-versa:
1. Auto-cleanup: when a task is marked done, offer to archive or delete its linked sessions (completed work, no longer needed for live context).
2. Knowledge distillation: synthesise durable knowledge entries from the cluster of sessions that produced a completed task.
Neither feature can be built without a queryable session↔task graph.

## Scope

### In scope (this task)
- Add `Tasks []string yaml:"tasks"` to `session.Session` struct
- Add `Sessions []string yaml:"sessions"` and `Related []string yaml:"related"` to `task.Task` struct
- Update `task.TaskJSON` and `index.Entry` accordingly
- Update `logos save`: add `--task <partial>` flag to append to session's `tasks` list
- Update `logos task create`: existing `--session` writes both `session:` (compat) and `sessions:` list
- Add `logos task update --add-session <partial>`: append a session to `sessions` list
- Update `logos refer --name <session>`: show "Linked tasks" section at the bottom when `tasks` is non-empty
- Update `logos task refer --name <task>`: show all sessions from `sessions` list (not just the single `session:` field)
- Update `logos sync` / index rebuild to populate new fields
- Update USAGE.md and cmd/init.go usageMD to document new flags

### Out of scope (future tasks)
- Auto-deletion of sessions on task completion
- Knowledge distillation (`logos distill` command)
- `logos graph` visualisation
- Referential integrity checks (links are informational only — no error if linked file does not exist)

### Design decisions
- **No referential integrity**: links are just strings; a missing linked file is a warning at most, never a hard error.
- **Backward compat over schema migration**: keep `session: ""` alongside `sessions: []`. A migration tool is not needed because yaml zero-values handle absent fields gracefully.
- **`related` in tasks is task→task only**: session→session links already use `related`. Mixing types in one field adds ambiguity; separate fields keep the model simple.
- **`tasks` in sessions is session→task only**: symmetric reasoning.

## Checklist

- [ ] Add `Tasks []string yaml:"tasks"` to session.Session struct
- [ ] Add `Sessions []string yaml:"sessions"` to task.Task struct
- [ ] Add `Related []string yaml:"related"` to task.Task struct
- [ ] Update task.TaskJSON with Sessions and Related fields
- [ ] Update index.Entry with Tasks field
- [ ] Update index.FromSession to populate Tasks
- [ ] Update task.Task.ToJSON() to populate Sessions and Related
- [ ] Update logos save: add --task flag, resolve partial → filename, append to session.Tasks
- [ ] Update logos task create: write to sessions list (keep session: for compat)
- [ ] Add logos task update --add-session flag
- [ ] Update logos refer output: show linked tasks when tasks list is non-empty
- [ ] Update logos task refer output: show all sessions from sessions list
- [ ] Update logos sync to rebuild both indexes with new fields
- [ ] Update USAGE.md and cmd/init.go usageMD
- [ ] Add / update tests for new struct fields and CLI flags

## Notes

### Open questions
1. Should `logos task update --session <partial>` continue to write only to `session:` (compat), or should it also append to `sessions:` list? Recommendation: write to both for forward compat.
2. When `logos task refer --with-session` is used and there are multiple sessions, show all of them or only the primary `session:` field? Recommendation: show all, separated by a header.
3. Should `logos ls --json` output include the `tasks` field? Yes — agents need it for graph traversal without opening each session file.

### Related sessions
Will be linked after the implementation session is saved.

