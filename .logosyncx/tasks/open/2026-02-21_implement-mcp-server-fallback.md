---
id: t-bds79j
date: 2026-02-21T10:43:20.308140+09:00
title: "[Phase3] Implement MCP server (fallback)"
status: open
priority: medium
session: ""
tags:
    - phase3
    - mcp
    - server
assignee: ""
---

## What

Implement an MCP (Model Context Protocol) server that exposes `logos` functionality
as tools for environments without shell access (e.g. Claude Desktop, Cursor in
restricted modes).

The MCP server should provide tools equivalent to the core CLI commands:

- `logos_ls` — list sessions with excerpts (equivalent to `logos ls --json`)
- `logos_refer` — read a session by name (equivalent to `logos refer <name>`)
- `logos_search` — keyword search over sessions (equivalent to `logos search`)
- `logos_task_ls` — list tasks (equivalent to `logos task ls --json`)
- `logos_task_refer` — read a task by name (equivalent to `logos task refer <name>`)

## Why

Agents that have shell access should use the CLI directly — it is simpler, more
composable, and requires no additional infrastructure. However, some environments
(Claude Desktop, hosted agent runtimes) do not expose a shell. An MCP server
provides these environments with equivalent read access to the shared session and
task context without requiring a shell.

The MCP server is explicitly a supplementary fallback, not the primary interface.

## Scope

- `cmd/mcp.go` — new cobra subcommand `logos mcp` that starts the MCP server
  (listens on stdio or a TCP port, configurable via flag)
- `internal/mcp/` — MCP protocol implementation (JSON-RPC 2.0 over stdio)
- `internal/mcp/tools.go` — tool definitions and handlers for the five tools listed above
- `go.mod` / `go.sum` — add MCP SDK dependency if a suitable Go library exists;
  otherwise implement the minimal JSON-RPC layer manually
- `README.md` — document how to configure Claude Desktop / Cursor to use the MCP server

## Checklist

- [ ] Research available Go MCP SDK libraries; choose one or implement minimal JSON-RPC layer
- [ ] Implement `logos mcp` subcommand (stdio transport)
- [ ] Implement `logos_ls` tool
- [ ] Implement `logos_refer` tool
- [ ] Implement `logos_search` tool
- [ ] Implement `logos_task_ls` tool
- [ ] Implement `logos_task_refer` tool
- [ ] Write integration tests (mock stdio transport)
- [ ] Document Claude Desktop config (`claude_desktop_config.json` snippet)
- [ ] Run `go test ./...` and confirm all tests pass

## Notes

Migrated from beads issue `logosyncx-79j`.

Design note from the original spec: agents with shell access should use the CLI;
MCP is a supplementary fallback only. Write-operations (`logos save`,
`logos task create`) are intentionally excluded from the MCP surface — those
require the CLI to ensure git staging works correctly.