---
id: t-1a8c0a
date: 2026-03-08T23:22:48.760996+09:00
title: 'Design: one-step task creation with --body-stdin'
seq: 4
status: open
priority: low
plan: 20260308-fix-bugs-and-code-quality-validation-markdown-refactor-walkthrough-enforcement
tags: []
assignee: ""
---

## What

Design a mechanism for one-step task creation where the agent can supply the task body in a single command, without a separate Write tool call. This is a **design/investigation task** — the output is a decision doc, not an implementation.

## Why

Current 3-step flow:
1. `logos task create ...` → creates scaffold
2. Read template
3. Write body with Write tool

The Write tool call is unavoidable today because the CLI only produces frontmatter. Eliminating it would reduce agent overhead and make task creation atomic. However, `--body-stdin` or similar flags introduce new design questions that need to be answered before implementation.

## Questions to Answer

1. **Input mechanism**: `--body-stdin` (pipe), `--body-file <path>`, or both?
2. **Template application**: Does the agent pre-fill the template, or does the CLI inject section headers automatically?
3. **Validation**: Should the CLI validate that required sections (What, Why, etc.) are present?
4. **Conflict with Write tool approach**: Does this replace or coexist with the current flow? Do we keep both paths?
5. **WALKTHROUGH.md implication**: If task creation is 1-step, does completion also need a parallel `--walkthrough-stdin` flag?
6. **Agent ergonomics**: Can LLMs reliably generate well-formed markdown + frontmatter in one shot to pipe via stdin? Or is the Write tool actually more reliable?

## Acceptance Criteria

- [ ] Written design doc (in this task's body or a linked plan) answering all questions above
- [ ] Decision: proceed with implementation, or reject with rationale
- [ ] If proceeding: scope and interface spec for the implementation task

## Checklist

- [ ] Review how `logos save` currently handles body (Write tool only)
- [ ] Check if `logos task create --stdin` was considered in original design docs
- [ ] Answer the 6 questions above
- [ ] Write decision and proposed interface (e.g. `logos task create --title "..." --body-stdin`)

## Notes

This is a **design task only** — do not implement until this task is done and the decision is "proceed".
May also apply to `logos save` (plan body via stdin) as a parallel improvement.
