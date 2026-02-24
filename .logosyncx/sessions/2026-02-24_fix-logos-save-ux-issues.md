---
id: efdf8f
date: 2026-02-24T19:51:13.679081+09:00
topic: fix-logos-save-ux-issues
tags:
    - bugfix
    - ux
    - session
agent: claude-sonnet-4-5
related: []
---

## Summary
Fixed 4 UX issues in logos save reported from agent feedback (task t-10bead).

## Key Decisions
- Removed id/date fields from .logosyncx/template.md (YAML parse error root cause)
- Changed Session.Date from time.Time to *time.Time with omitempty to tolerate empty/omitted values
- Added FileName() nil guard: falls back to time.Now() when Date is nil
- Added {{ detection hint to Parse() error message for better DX
- Added install target to Makefile (build/install/test all now present)
- Updated all test helpers across 5 test files to use &date pointer pattern

## Context Used
- Task: 2026-02-21_fix-logos-save-ux-issues-from-agent-feedback.md (t-10bead)

## Notes
All go test ./... pass. index.Entry.Date remains time.Time; only session.Session.Date changed to pointer.