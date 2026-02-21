# Logosyncx

> Shared AI agent conversation context — stored in git, no external database required.

**Logosyncx** (`logos`) is a CLI tool that lets AI agents save, search, and retrieve past session summaries inside your git repository. Team members and agents share context simply by running `git pull` — no embedding servers, no vector databases, no extra services.

---

## How it works

`logos init` creates a `.logosyncx/` directory in your project and appends one line to `AGENTS.md`:

```
Use `logos` CLI for session context management.
Full reference: .logosyncx/USAGE.md
```

From that point on, any agent that reads `AGENTS.md` (Claude Code, Cursor, aider, etc.) automatically knows how to use `logos` to find past decisions and save new context.

**Agents handle their own semantic search** — `logos ls --json` returns all sessions with excerpts, and the LLM itself judges which ones are relevant. No embedding API needed.

---

## Installation

### Build from source

Requires [Go 1.21+](https://go.dev/dl/).

```sh
git clone https://github.com/senna-lang/logosyncx.git
cd logosyncx
go build -o logos .
sudo mv logos /usr/local/bin/   # or anywhere on your $PATH
```

### Verify

```sh
logos version
# logos version 0.0.1
```

> **Homebrew tap and pre-built binaries are planned for a future release.**

---

## Quick Start

```sh
# 1. Initialize Logosyncx in your project
cd your-project
logos init

# 2. Save a session (agents do this automatically)
logos save --file session.md

# 3. List all saved sessions
logos ls

# 4. Read a session
logos refer auth-refactor --summary

# 5. Search by keyword
logos search "JWT"
```

---

## Commands

### `logos init`

Initializes Logosyncx in the current directory.

```sh
logos init
```

- Creates `.logosyncx/` with `config.json`, `USAGE.md`, and `template.md`
- Appends a reference line to `AGENTS.md` (or `CLAUDE.md` if present)
- Exits with an error if already initialized

---

### `logos save`

Saves a session file to `.logosyncx/sessions/`.

```sh
logos save --file path/to/session.md   # from a file
cat session.md | logos save --stdin    # from stdin
```

- `id` and `date` in the frontmatter are auto-filled if missing
- Saved as `<date>_<topic>.md` and `git add` is run automatically
- `git commit` and `git push` remain the user's responsibility

---

### `logos ls`

Lists all saved sessions.

```sh
logos ls                        # human-readable table
logos ls --tag auth             # filter by tag
logos ls --since 2025-02-01    # filter by date
logos ls --json                 # structured JSON output (for agents)
```

`--json` includes `topic`, `tags`, and `excerpt` (first ~300 chars of the `## Summary` section).
Agents read this list and decide which sessions to load — no separate search index needed.

```json
[
  {
    "id": "a1b2c3",
    "filename": "2025-02-20_auth-refactor.md",
    "date": "2025-02-20T10:30:00Z",
    "topic": "auth-refactor",
    "tags": ["auth", "jwt"],
    "agent": "claude-code",
    "related": [],
    "excerpt": "Decided to migrate from session cookies to JWT..."
  }
]
```

---

### `logos refer`

Prints a session's content.

```sh
logos refer 2025-02-20_auth-refactor.md   # full content
logos refer auth                           # partial name match
logos refer auth --summary                 # key sections only (saves tokens)
```

`--summary` returns only the sections listed in `summary_sections` in `config.json`
(default: `Summary` and `Key Decisions`). Recommended for agents to keep token usage low.

---

### `logos search`

Keyword search across topic, tags, and excerpt.

```sh
logos search "JWT"
logos search "auth" --tag security
```

Case-insensitive string match — useful for quickly narrowing candidates by keyword.
For deeper semantic search, use `logos ls --json` and let the agent reason over the excerpts.

---

### `logos status`

Shows uncommitted or unsaved changes in `.logosyncx/sessions/`.

```sh
logos status
```

---

### `logos sync`

Rebuilds the session index from disk. Run this after manually adding or editing `.md` files.

```sh
logos sync
```

---

## Session file format

Sessions are plain Markdown files with YAML frontmatter.
The default template (`.logosyncx/template.md`):

```markdown
---
id: {{id}}
date: {{date}}
topic: {{topic}}
tags: []
agent:
related: []
---

## Summary
<!-- Briefly describe what was discussed and decided -->

## Key Decisions
<!-- Important decisions as bullet points -->
-

## Context Used
<!-- Past sessions or external resources referenced -->

## Notes
<!-- Other notes -->

## Raw Conversation
<!-- Paste the conversation log here (optional) -->
```

`{{id}}` and `{{date}}` are auto-filled by `logos save`.
`{{topic}}` must be provided by the agent — a warning is shown if missing.

---

## Configuration

`.logosyncx/config.json` is created by `logos init` and can be edited by hand:

```json
{
  "version": "1",
  "project": "my-project",
  "agents_file": "AGENTS.md",
  "save": {
    "summary_sections": ["Summary", "Key Decisions"]
  },
  "privacy": {
    "filter_patterns": []
  }
}
```

| Field | Description |
|-------|-------------|
| `agents_file` | The file `logos init` appends its reference line to |
| `save.summary_sections` | Sections returned by `logos refer --summary` |
| `privacy.filter_patterns` | Regex patterns — matching content is warned/masked on `logos save` |

---

## Repository layout

```
your-project/
├── AGENTS.md                    ← logos init appends here
├── .logosyncx/
│   ├── config.json
│   ├── USAGE.md                 ← agent-facing command reference
│   ├── template.md
│   └── sessions/
│       ├── 2025-02-20_auth-refactor.md
│       └── 2025-02-18_db-schema.md
└── ... (your existing files)
```

`.logosyncx/` is committed to git. Context is shared across the team with a simple `git pull`.
`sessions/` uses `<date>_<topic>.md` filenames so concurrent commits from multiple contributors
never conflict.

---

## Agent workflow example

```
User: "I want to work on auth — is there any past context?"

Agent:
  1. Reads AGENTS.md → "Use logos CLI, see .logosyncx/USAGE.md"
  2. Runs: logos ls --json
  3. Reads excerpts, judges "2025-02-20_auth-refactor.md looks relevant"
  4. Runs: logos refer auth-refactor --summary
  5. Answers with awareness of past decisions
```

Semantic understanding is the agent's responsibility.
Logosyncx focuses on storing and retrieving data — the LLM decides what is relevant.

---

## Current status

> ⚠️ This project is under active development. CLI commands are being implemented now.

| Command | Status |
|---------|--------|
| `logos version` | ✅ Available |
| `logos init` | ✅ Available |
| `logos save` | ✅ Available |
| `logos ls` | ✅ Available |
| `logos refer` | ✅ Available |
| `logos search` | 🔧 In progress |
| `logos status` | 📅 Planned |
| `logos sync` | 📅 Planned |

---

## License

MIT