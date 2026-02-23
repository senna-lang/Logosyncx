---
id: t-bdsld8
date: 2026-02-21T10:43:09.459964+09:00
title: "[Phase2] Interactive UI for logos ls (fzf-style)"
status: open
priority: medium
session: ""
tags:
    - phase2
    - ui
    - bubbletea
    - ux
assignee: ""
---

## What

Implement an interactive fzf-style session browser that launches when
`logos ls` is run without arguments in an interactive terminal.

Behaviour:
- Display a scrollable, filterable list of sessions (date, topic, tags, excerpt)
- Typing narrows the list via fuzzy/substring match on topic, tags, and excerpt
- Arrow keys / `j` / `k` navigate the list
- `Enter` prints the selected session filename to stdout (enabling shell
  composition: `logos refer $(logos ls)`)
- `Esc` / `q` exits without output
- Falls back to the existing non-interactive table output when:
  - `--json`, `--tag`, or `--since` flags are provided
  - stdout is not a TTY (e.g. piped to another command)

## Why

The current `logos ls` output is a static table — useful for agents but not
optimised for humans who want to quickly browse and pick a session to read.
An interactive UI lowers the friction of finding the right session without
requiring the user to remember exact names or dates.

`bubbletea` is already listed as a Phase 2 dependency in the design spec, so
no new dependencies are needed.

## Scope

- `cmd/ls.go` — detect TTY + absence of filter flags; when both conditions are
  met, hand off to the interactive model instead of printing the table
- `internal/ui/ls_model.go` — new bubbletea `Model` for the interactive session
  list (Init / Update / View)
- `internal/ui/ls_model_test.go` — unit tests for filter logic and key handling
- `go.mod` / `go.sum` — add `github.com/charmbracelet/bubbletea` and
  `github.com/charmbracelet/lipgloss` (for styling) if not already present

## Checklist

- [ ] Add `bubbletea` and `lipgloss` to `go.mod`
- [ ] Implement `internal/ui/ls_model.go` with Init / Update / View
- [ ] Implement substring filter on topic, tags, and excerpt
- [ ] Wire the model into `cmd/ls.go` (TTY detection + flag check)
- [ ] Ensure `--json` / `--tag` / `--since` bypass the interactive mode
- [ ] Ensure piped stdout bypasses the interactive mode
- [ ] Write unit tests for filter logic
- [ ] Manual smoke-test: run `logos ls` in terminal, verify interactive UI appears
- [ ] Run `go test ./...` and confirm all tests pass

## Notes

Migrated from beads issue `logosyncx-ld8`.

TTY detection can be done with `golang.org/x/term` (`term.IsTerminal(int(os.Stdout.Fd()))`).

The interactive model should read from the in-memory session list already loaded
by the existing `ls` logic — no additional I/O inside the UI layer.

Long-term, the same pattern could be applied to `logos task ls`, but that is
out of scope for this task.