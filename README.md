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
mv logos /usr/local/bin/   # or anywhere on your $PATH
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

# 2. Save a session
logos save --topic "auth refactor" --tag auth --tag jwt --body-stdin <<'EOF'
## Summary

Decided to migrate from session cookies to JWT. The new flow uses RS256 signing.

## Key Decisions

- RS256 over HS256 for multi-service support
- Refresh tokens stored in httpOnly cookies
EOF

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

- Creates `.logosyncx/` with `config.json`, `USAGE.md`, `template.md`, and `task-template.md`
- Creates `.logosyncx/sessions/` and `.logosyncx/tasks/{open,in_progress,done,cancelled}/`
- Appends a reference line to `AGENTS.md` (or `CLAUDE.md` if present)
- Exits with an error if already initialized

---

### `logos save`

Saves a session to `.logosyncx/sessions/` using flag-based input.

```sh
# Topic only (empty body)
logos save --topic "quick sync"

# Inline body
logos save --topic "auth refactor" --body "## Summary\n\nSwitched to JWT."

# Body from stdin (recommended for multi-line content)
logos save --topic "auth refactor" \
           --tag auth --tag jwt \
           --agent claude-code \
           --related 2025-02-18_db-schema.md \
           --body-stdin < notes.md
```

| Flag | Short | Description |
|------|-------|-------------|
| `--topic` | `-t` | Session topic — required, used as the filename slug |
| `--tag` | | Tag to attach — repeatable (`--tag go --tag cli`) |
| `--agent` | `-a` | Agent name (e.g. `claude-code`) |
| `--related` | | Related session filename — repeatable |
| `--body` | `-b` | Session body text (inline) |
| `--body-stdin` | | Read body prose from stdin (no frontmatter needed) |

- `id` and `date` are auto-filled automatically
- Saved as `<date>_<topic>.md` under `.logosyncx/sessions/`
- `git add` is run automatically; `git commit` and `git push` remain your responsibility

---

### `logos ls`

Lists all saved sessions.

```sh
logos ls                        # human-readable table
logos ls --tag auth             # filter by tag
logos ls --since 2025-02-01     # filter by date
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

### `logos sync`

Rebuilds the session and task indexes from disk. Run this after manually adding, editing, or deleting `.md` files.

```sh
logos sync
```

---

### `logos task`

Manages tasks stored in `.logosyncx/tasks/`.

```sh
# Create a task
logos task create --title "Implement rate limiting" \
                  --description "Add per-IP rate limiting to the auth endpoint." \
                  --priority high \
                  --tag go --tag auth \
                  --session auth-refactor

# List tasks
logos task ls                              # human-readable table
logos task ls --status open               # filter by status
logos task ls --priority high             # filter by priority
logos task ls --tag auth                  # filter by tag
logos task ls --json                      # structured JSON output

# Read a task
logos task refer rate-limiting            # full content
logos task refer rate-limiting --summary  # key sections only
logos task refer rate-limiting --with-session  # append linked session summary

# Update a task
logos task update rate-limiting --status in_progress
logos task update rate-limiting --status done
logos task update rate-limiting --priority medium
logos task update rate-limiting --assignee alice

# Search tasks
logos task search "rate limit"
logos task search "auth" --status open

# Delete tasks
logos task delete rate-limiting           # prompts for confirmation
logos task delete rate-limiting --force   # skip confirmation
logos task purge --status done            # bulk-delete by status
logos task purge --status done --force
```

`logos task create` flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-T` | Task title — required |
| `--description` | `-d` | Task description (becomes the `## What` section) |
| `--priority` | `-p` | `high` / `medium` / `low` (default: `medium`) |
| `--tag` | | Tag to attach — repeatable |
| `--session` | `-s` | Partial name of the session to link |

Tasks are stored as markdown files in `.logosyncx/tasks/<status>/` and tracked in git alongside sessions.

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
| `privacy.filter_patterns` | Regex patterns — matching content triggers a warning on `logos save` |

---

## Repository layout

```
your-project/
├── AGENTS.md                        ← logos init appends here
├── .logosyncx/
│   ├── config.json
│   ├── USAGE.md                     ← agent-facing command reference
│   ├── template.md
│   ├── task-template.md
│   ├── index.jsonl                  ← session index (auto-managed)
│   ├── task-index.jsonl             ← task index (auto-managed)
│   ├── sessions/
│   │   ├── 2025-02-20_auth-refactor.md
│   │   └── 2025-02-18_db-schema.md
│   └── tasks/
│       ├── open/
│       ├── in_progress/
│       ├── done/
│       └── cancelled/
└── ... (your existing files)
```

`.logosyncx/` is committed to git. Context is shared across the team with a simple `git pull`.
`sessions/` uses `<date>_<topic>.md` filenames so concurrent commits from multiple contributors never conflict.

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

--- later ---

User: "Save this session."

Agent:
  6. Runs: logos save --topic "auth middleware implementation" \
                      --tag auth --tag go \
                      --agent claude-code \
                      --related 2025-02-20_auth-refactor.md \
                      --body-stdin <<'EOF'
     ## Summary
     Implemented JWT middleware for the auth service...
     EOF
```

Semantic understanding is the agent's responsibility.
Logosyncx focuses on storing and retrieving data — the LLM decides what is relevant.

---

## Current status

| Command | Status |
|---------|--------|
| `logos version` | ✅ Available |
| `logos init` | ✅ Available |
| `logos save` | ✅ Available |
| `logos ls` | ✅ Available |
| `logos refer` | ✅ Available |
| `logos search` | ✅ Available |
| `logos sync` | ✅ Available |
| `logos task` | ✅ Available |
| `logos status` | 📅 Planned |

---

## License

MIT