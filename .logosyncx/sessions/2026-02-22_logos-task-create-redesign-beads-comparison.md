---
id: a3f10d
date: 2026-02-22T20:34:56+09:00
topic: logos-task-create-redesign-beads-comparison
tags:
    - task
    - design
    - dx
    - beads
    - cli
agent: claude-sonnet-4-6
related:
    - 2026-02-22_go-fmt-pre-commit-hook-plan.md
---

## Summary

Compared the current `logos task create` design against how beads (`bd create`) handles task creation. Identified that the current logosyncx design forces agents to construct a full markdown file with YAML frontmatter every time they save a task or session, which creates unnecessary friction. Agreed to redesign `logos task create` to accept structured parameters directly via flags (Option A), eliminating temporary files entirely.

## Key Decisions

- **Problem confirmed**: the current `logos task create --file / --stdin` interface requires agents to build a full frontmatter + markdown document before calling the command. This means a temporary file (or heredoc) is always needed as an intermediate step.
- **Beads approach studied**: `bd create "title"` accepts the title as a positional argument and additional fields as flags (`--description`, `--priority`, `--labels`, etc.). SQLite is the source of truth; JSONL is the git-friendly export. No temporary files.
- **Chosen direction: Option A** — add flag-based parameters (`--title`, `--description`, `--priority`, `--tag`, `--session`) to `logos task create`. The existing `--file` / `--stdin` interface is kept for backward compatibility but flag-based creation becomes the preferred path for agents.
- Option B (plain text via stdin) and Option C (delegate all task management to beads) were considered but not chosen.

## Context Used

- `.beads/issues.jsonl` — reviewed existing beads issues to understand JSONL storage model
- `.beads/README.md` — confirmed beads stores data in SQLite + JSONL, no per-issue files
- `bd create --help` — confirmed `bd create [title]` with rich flag set; no temporary files required
- `cmd/task.go` — current `logos task create` implementation; requires `--file <path>` or `--stdin` with full markdown frontmatter
- `internal/task/store.go` — `Store.Save()` implementation; auto-fills id/date/status/priority already
- `.logosyncx/tasks/2026-02-21_fix-logos-save-ux-issues-from-agent-feedback.md` — example of the verbose markdown format agents must currently produce

## Notes

The core insight is that `Store.Save()` already handles all auto-fill logic (id, date, status, priority defaults). The only reason agents need to build a full markdown document today is that the CLI surface has no other entry point. Adding flag-based creation is a thin layer on top of the existing store — the backend does not need to change.

The same friction exists for `logos save` (sessions), but that was not in scope for this session. Session content is inherently long-form prose, so a flag-based interface is less practical there; that problem may need a different solution.