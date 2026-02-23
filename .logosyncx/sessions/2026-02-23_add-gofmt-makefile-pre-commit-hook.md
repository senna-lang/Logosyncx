---
id: "331e77"
date: 2026-02-23T11:28:08.756087+09:00
topic: add-gofmt-makefile-pre-commit-hook
tags:
    - go
    - tooling
    - makefile
    - git-hooks
agent: claude-sonnet-4-5
related: []
---

## Summary\n\nImplemented task t-gofmt01: added Makefile and scripts/hooks/pre-commit to enforce go fmt before commits.\n\n## What was done\n\n- Created Makefile with targets: setup, fmt, lint, test, build, clean, help\n- Created scripts/hooks/pre-commit (gofmt -l . check, blocks commit on failure)\n- Added execute permission to scripts/hooks/pre-commit\n- Updated README.md with a new Development section\n\n## Key Decisions\n\n- git config core.hooksPath scripts/hooks via make setup so hook is version-controlled\n- make lint runs go vet ./... (no external linter dependency)\n\n## Verification\n\n- make setup, make fmt, make lint, make test all pass\n- Unformatted file triggers exit 1 with clear message