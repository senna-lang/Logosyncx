---
id: "58e003"
date: 2026-02-24T20:00:10.199102+09:00
topic: update-usage-md-tasks-section
tags:
    - docs
    - init
    - task
agent: claude-sonnet-4-5
related: []
---

## Summary
Completed task t-bds34x: USAGE.md template tasks section was already present in cmd/init.go (added in a prior session). Verified content, added missing test, and fixed a related bug.

## Key Decisions
- Tasks section already existed in usageMD const in cmd/init.go — task was effectively done
- Added TestInit_USAGEMDIncludesTasksSection test to enforce the section is generated
- Also fixed templateMD const in cmd/init.go: removed id/date placeholder fields to match the .logosyncx/template.md fix from t-10bead

## Context Used
- Task: 2026-02-21_update-usage-md-template-add-tasks-section-for-agent-reference.md (t-bds34x)
- Session: 2026-02-24_fix-logos-save-ux-issues.md (t-10bead, template.md fix)