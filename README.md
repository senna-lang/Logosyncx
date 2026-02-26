# Logosyncx

> Git-native session memory and task management infrastructure for AI agents.

**Logosyncx** (`logos`) is a CLI tool that gives AI agents two things: **session memory** — a searchable record of past decisions and discussions — and **task management** — a structured backlog of what to do next. Both live as plain markdown files in your git repository, so the whole team shares context with a simple `git pull`. No embedding servers, no vector databases, no extra services.

---

## How it works

`logos init` creates a `.logosyncx/` directory in your project and appends one line to `AGENTS.md`:

```
Use `logos` CLI for session context management.
Full reference: .logosyncx/USAGE.md
```

From that point on, any agent that reads `AGENTS.md` (Claude Code, Cursor, aider, etc.) automatically knows how to use `logos` to recall past decisions and manage tasks.

**Sessions answer "why"** — they record what was discussed, what was decided, and the reasoning behind it.
**Tasks answer "what"** — they track what needs to be done, and can link back to the session that motivated them.

**Agents handle their own semantic search** — `logos ls --json` returns all sessions with excerpts, and the LLM itself judges which ones are relevant. No embedding API needed.

---

## Installation

### Homebrew (macOS / Linux)

```sh
brew install senna-lang/tap/logos
```

### curl | bash (Linux / macOS / CI)

```sh
curl -sSfL https://raw.githubusercontent.com/senna-lang/Logosyncx/main/scripts/install.sh | bash
```

The script detects your OS and architecture, downloads the correct pre-built binary from
[GitHub Releases](https://github.com/senna-lang/Logosyncx/releases/latest), verifies the
SHA256 checksum, and installs `logos` to `~/.local/bin`.

Pin a specific version:

```sh
curl -sSfL https://raw.githubusercontent.com/senna-lang/Logosyncx/main/scripts/install.sh | \
  LOGOS_VERSION=v0.2.0 bash
```

### Direct download

Download the pre-built binary for your platform from the
[latest GitHub Release](https://github.com/senna-lang/Logosyncx/releases/latest),
extract the archive, and place the `logos` binary somewhere on your `$PATH`.

### go install (Go developers)

```sh
go install github.com/senna-lang/logosyncx@latest
```

Requires Go 1.21+.

### Verify

```sh
logos version
# logos v0.1.0 (darwin/arm64)
```

### Updating

```sh
logos update          # self-update to the latest release
brew upgrade logos    # if installed via Homebrew
```

---

## Quick Start

```sh
# 1. Initialize Logosyncx in your project
cd your-project
logos init

# 2. Save a session
logos save --topic "auth refactor" \
           --tag auth --tag jwt \
           --section "Summary=Decided to migrate from session cookies to JWT. The new flow uses RS256 signing." \
           --section "Key Decisions=- RS256 over HS256 for multi-service support\n- Refresh tokens stored in httpOnly cookies"

# 3. List all saved sessions
logos ls

# 4. Read a session
logos refer --name auth-refactor --summary

# 5. Search by keyword
logos search --keyword "JWT"
```

---

## Commands

### `logos init`

Initializes Logosyncx in the current directory.

```sh
logos init
```

- Creates `.logosyncx/` with `config.json` and `USAGE.md`
- Creates `.logosyncx/sessions/` and `.logosyncx/tasks/{open,in_progress,done,cancelled}/`
- Appends a reference line to `AGENTS.md` (or `CLAUDE.md` if present)
- Exits with an error if already initialized

---

### `logos save`

Saves a session to `.logosyncx/sessions/` using flag-based input.

```sh
# Topic only (no body)
logos save --topic "quick sync"

# With sections
logos save --topic "auth refactor" \
           --tag auth --tag jwt \
           --agent claude-code \
           --related 2025-02-18_db-schema.md \
           --section "Summary=Switched to JWT." \
           --section "Key Decisions=- RS256 chosen for multi-service support"
```

| Flag | Short | Description |
|------|-------|-------------|
| `--topic` | `-t` | Session topic — required, used as the filename slug |
| `--tag` | | Tag to attach — repeatable (`--tag go --tag cli`) |
| `--agent` | `-a` | Agent name (e.g. `claude-code`) |
| `--related` | | Related session filename — repeatable |
| `--section` | | Section content as `"Name=content"` — repeatable; name must be defined in `config.json` |

- `id` and `date` are auto-filled automatically
- Saved as `<date>_<topic>.md` under `.logosyncx/sessions/`
- By default (`git.auto_push: false`) no git operations are performed — commit and push are your responsibility
- When `git.auto_push: true` in `config.json`, `git add`, `git commit`, and `git push` run automatically

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
logos refer --name 2025-02-20_auth-refactor.md   # full content
logos refer --name auth                           # partial name match
logos refer --name auth --summary                 # key sections only (saves tokens)
```

`--summary` returns only the sections listed in `sessions.summary_sections` in `config.json`
(default: `Summary` and `Key Decisions`). Recommended for agents to keep token usage low.

---

### `logos search`

Keyword search across topic, tags, and excerpt.

```sh
logos search --keyword "JWT"
logos search --keyword "auth" --tag security
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

### `logos update`

Updates the `logos` binary to the latest release from GitHub.

```sh
logos update           # download and install the latest release
logos update --check   # check only; print status without installing
```

Not available for development (`dev`) builds.

---

### `logos task`

Manages tasks stored in `.logosyncx/tasks/`.

```sh
# Create a task
logos task create --title "Implement rate limiting" \
                  --priority high \
                  --tag go --tag auth \
                  --session auth-refactor \
                  --section "What=Add per-IP rate limiting to the auth endpoint." \
                  --section "Checklist=- [ ] Implement middleware\n- [ ] Add tests"

# List tasks
logos task ls                              # human-readable table
logos task ls --status open               # filter by status
logos task ls --priority high             # filter by priority
logos task ls --tag auth                  # filter by tag
logos task ls --json                      # structured JSON output

# Read a task
logos task refer --name rate-limiting                       # full content
logos task refer --name rate-limiting --summary             # key sections only
logos task refer --name rate-limiting --with-session        # append linked session summary

# Update a task
logos task update --name rate-limiting --status in_progress
logos task update --name rate-limiting --status done
logos task update --name rate-limiting --priority medium
logos task update --name rate-limiting --assignee alice

# Search tasks
logos task search --keyword "rate limit"
logos task search --keyword "auth" --status open

# Delete tasks
logos task delete --name rate-limiting           # prompts for confirmation
logos task delete --name rate-limiting --force   # skip confirmation
logos task purge --status done                   # bulk-delete by status
logos task purge --status done --force
```

`logos task create` flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--title` | `-T` | Task title — required |
| `--priority` | `-p` | `high` / `medium` / `low` (default: `medium`) |
| `--tag` | | Tag to attach — repeatable |
| `--session` | `-s` | Partial name of the session to link |
| `--section` | | Section content as `"Name=content"` — repeatable; name must be defined in `config.json` |

Tasks are stored as markdown files in `.logosyncx/tasks/<status>/` and tracked in git alongside sessions.

---

## Configuration

`.logosyncx/config.json` is created by `logos init` and can be edited by hand:

```json
{
  "version": "1",
  "project": "my-project",
  "agents_file": "AGENTS.md",
  "sessions": {
    "summary_sections": ["Summary", "Key Decisions"],
    "excerpt_section": "Summary",
    "sections": [
      { "name": "Summary",         "level": 2, "required": true  },
      { "name": "Key Decisions",   "level": 2, "required": false },
      { "name": "Context Used",    "level": 2, "required": false },
      { "name": "Notes",           "level": 2, "required": false },
      { "name": "Raw Conversation","level": 2, "required": false }
    ]
  },
  "tasks": {
    "default_status": "open",
    "default_priority": "medium",
    "summary_sections": ["What", "Checklist"],
    "excerpt_section": "What",
    "sections": [
      { "name": "What",      "level": 2, "required": true  },
      { "name": "Why",       "level": 2, "required": false },
      { "name": "Scope",     "level": 2, "required": false },
      { "name": "Checklist", "level": 2, "required": false },
      { "name": "Notes",     "level": 2, "required": false }
    ]
  },
  "privacy": {
    "filter_patterns": []
  },
  "git": {
    "auto_push": false
  }
}
```

| Field | Description |
|-------|-------------|
| `agents_file` | The file `logos init` appends its reference line to |
| `sessions.summary_sections` | Sections returned by `logos refer --summary` |
| `sessions.excerpt_section` | Section used as the session excerpt in the index (default: `Summary`) |
| `sessions.sections` | Ordered list of body sections for session files |
| `tasks.summary_sections` | Sections returned by `logos task refer --summary` |
| `tasks.excerpt_section` | Section used as the task excerpt in the index (default: `What`) |
| `tasks.sections` | Ordered list of body sections for task files |
| `privacy.filter_patterns` | Regex patterns — matching content triggers a warning on `logos save` |
| `git.auto_push` | When `true`, `logos save` runs `git add`, `git commit`, and `git push` automatically |

---

## Repository layout

```
your-project/
├── AGENTS.md                        ← logos init appends here
├── .logosyncx/
│   ├── config.json
│   ├── USAGE.md                     ← agent-facing command reference
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
  4. Runs: logos refer --name auth-refactor --summary
  5. Answers with awareness of past decisions

--- later ---

User: "Save this session."

Agent:
  6. Runs: logos save --topic "auth middleware implementation" \
                      --tag auth --tag go \
                      --agent claude-code \
                      --related 2025-02-20_auth-refactor.md \
                      --section "Summary=Implemented JWT middleware for the auth service..." \
                      --section "Key Decisions=- Used RS256 signing\n- Middleware applied at router level"
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
| `logos update` | ✅ Available |
| `logos task` | ✅ Available |
| `logos status` | 📅 Planned |

---

## Development

### Setup

After cloning the repository, run `make setup` once to activate the git pre-commit hook:

```sh
git clone https://github.com/senna-lang/Logosyncx.git
cd Logosyncx
make setup
```

This configures `git config core.hooksPath scripts/hooks`, which activates a pre-commit hook that rejects commits containing unformatted Go files.

### Available make targets

| Target | Description |
|--------|-------------|
| `make setup` | Configure git hooks (run once after cloning) |
| `make fmt` | Format all Go source files (`go fmt ./...`) |
| `make lint` | Run static analysis (`go vet ./...`) |
| `make test` | Run all tests (`go test ./...`) |
| `make build` | Build the `logos` binary (version shows as `dev`) |
| `make install` | Build and install to `~/bin/logos` |
| `make clean` | Remove the built binary and `dist/` |
| `make snapshot` | Build a local snapshot for all platforms via GoReleaser (no publish) |
| `make release-dry-run` | Full release dry run — builds all platforms but does not publish |
| `make release` | Tag HEAD and push to trigger the GitHub Actions release pipeline |

### Before committing

```sh
make fmt   # fix formatting
make lint  # check for issues
make test  # run tests
```

The pre-commit hook automatically blocks commits that include unformatted files.
If the hook fires, run `make fmt` and re-stage your changes.

---

## License

MIT