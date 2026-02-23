---
id: t-bds407
date: 2026-02-21T10:43:09.464742+09:00
title: "[Phase2] Implement privacy filter"
status: open
priority: medium
session: ""
tags:
    - phase2
    - security
    - privacy
assignee: ""
---

## What

Implement a privacy filter that runs during `logos save`. The filter detects
content matching patterns defined in `config.json` under `privacy.filter_patterns`
(API keys, tokens, personal info, etc.) and warns or masks the matched values
before the session file is written to disk.

Provide a sensible set of default regex patterns out of the box so the feature
is useful without any configuration.

## Why

Sessions may contain sensitive data (API keys, passwords, email addresses,
internal URLs) that were mentioned during a conversation. Committing such data
to a git repository — even a private one — is a security risk. A filter that
catches these patterns at save-time prevents accidental leakage.

## Scope

- `pkg/privacy/filter.go` — new package with `Filter` struct, `Check(content string) []Match`
  and `Mask(content string) string` functions
- `pkg/privacy/filter_test.go` — unit tests for each default pattern
- `cmd/save.go` — integrate privacy filter: run before writing the session file,
  print warnings for each match (with line number and masked preview), abort or
  mask depending on config severity setting
- `internal/config/` (or `pkg/config/`) — ensure `privacy.filter_patterns` is
  loaded and passed to the filter
- `cmd/init.go` — default `config.json` already includes `"privacy": {"filter_patterns": []}`;
  no change needed unless