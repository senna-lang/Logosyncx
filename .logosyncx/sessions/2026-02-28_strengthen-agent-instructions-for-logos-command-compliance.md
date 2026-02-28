---
id: 9bb61b
date: 2026-02-28T10:51:30.601346+09:00
topic: Strengthen agent instructions for logos command compliance
tags:
    - documentation
    - agents
agent: claude-code
related: []
---

## Summary

Added a new task to improve agent-facing instructions, then implemented the fix immediately. Rewrote AGENTS.md and USAGE.md to replace passive 'When to use' suggestions with a MANDATORY triggers table mapping user phrases (including Japanese) to exact logos commands. Added a session-start rule requiring logos ls --json before any work. Updated cmd/init.go to keep usageMD and agentsLine in sync.

## Key Decisions

- Root cause was instructions written as suggestions (not mandates) so agents skipped them\n- AGENTS.md lacked any mid-session or session-start triggers entirely\n- Fix: explicit phrase→command table with MUST language and Japanese phrase variants\n- agentsLine (injected into AGENTS.md/CLAUDE.md by logos init) now includes a compact mandatory trigger summary

## Context Used

- Previous session: Implementation and USAGE Consistency Review

