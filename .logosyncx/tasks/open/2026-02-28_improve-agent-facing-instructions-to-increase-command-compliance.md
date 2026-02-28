---
id: t-39fd98
date: 2026-02-28T10:46:58.548556+09:00
title: Improve agent-facing instructions to increase command compliance
status: open
priority: high
session: ""
tags:
    - documentation
    - agents
    - ux
assignee: ""
---

## What

Rework the agent-facing instructions in USAGE.md (and the usageMD template in cmd/init.go) so that agents reliably use logos commands with the correct syntax. Current evidence: agents in external environments fail to follow command conventions (wrong flags, positional arguments, missing --section, etc.). The goal is to make the instructions clearer, more actionable, and harder to misinterpret.

## Why

The tool's value depends entirely on agents using it correctly. If agents skip logos commands or call them with wrong syntax, context is never saved or retrieved, defeating the purpose of the tool.

## Scope

- Audit USAGE.md for ambiguous or under-specified instructions\n- Add concrete anti-patterns / DON'T examples alongside the existing DO examples\n- Add a structured 'Quick Reference' cheat-sheet at the top of USAGE.md for fast scanning\n- Consider adding a 'Common Mistakes' section covering the most frequent failures observed\n- Investigate whether a mandatory preamble (injected into AGENTS.md by logos init) improves compliance\n- Update usageMD constant in cmd/init.go to match

## Checklist

- [ ] Identify the specific command failures observed in external environments\n- [ ] Draft improved USAGE.md with anti-patterns and Quick Reference\n- [ ] Add a 'Common Mistakes' section\n- [ ] Review whether AGENTS.md preamble text should be strengthened\n- [ ] Update usageMD template in cmd/init.go to match new USAGE.md\n- [ ] Verify changes with logos sync

