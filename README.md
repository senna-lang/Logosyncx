# Logosyncx

> Git-native plan and task management infrastructure for AI agents.

**Logosyncx** (`logos`) is a CLI tool that gives AI agents structured, persistent memory across sessions. Plans record **why** work was done and **what** was decided. Tasks track **what** to do next. Knowledge files distill completed plans into reusable learnings. Everything lives as plain markdown in your git repository — no external services required.

---

## How it works

`logos init` creates a `.logosyncx/` directory in your project and adds a reference to your agents file (`AGENTS.md` or `CLAUDE.md`):

```
Use `logos` CLI for session context management.
Full reference: .logosyncx/USAGE.md
```

Any agent that reads your agents file automatically knows how to use `logos` to recall past decisions and manage work.

**Plans answer "why"** — they record what was discussed, what was decided, and the context behind it. They can depend on each other and are distilled into knowledge files when complete.

**Tasks answer "what"** — they track implementation steps inside a plan, with seq numbers, priority, and dependency tracking.

**Knowledge files answer "what did we learn"** — distilled from completed plan walkthroughs, they become the long-term memory of the project.

**Agents do their own semantic search** — `logos ls --json` returns all plans with excerpts, and the LLM judges relevance. No embedding API needed.

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

Pin a specific version:

```sh
curl -sSfL https://raw.githubusercontent.com/senna-lang/Logosyncx/main/scripts/install.sh | \
  LOGOS_VERSION=v0.2.0 bash
```

### Go install

```sh
go install github.com/senna-lang/logosyncx@latest
```

---

## Quick start

```sh
# 1. Initialize in your project
cd my-project
logos init

# 2. Save a plan (scaffold only — write the body yourself)
logos save --topic "Migrate auth to JWT" --tag auth --tag backend

# 3. Create tasks for the plan
logos task create --plan "20260301-migrate-auth-to-jwt" --title "Update token signing" --priority high
logos task create --plan "20260301-migrate-auth-to-jwt" --title "Add refresh token endpoint"

# 4. Work through tasks
logos task update --name "update-token-signing" --status in_progress
logos task update --name "update-token-signing" --status done   # creates WALKTHROUGH.md

# 5. When all tasks are done, distill into knowledge
logos distill --plan "20260301-migrate-auth-to-jwt"

# 6. Commit everything
git add .logosyncx/
git commit -m "logos: distill migrate-auth-to-jwt"
git push
```

---

## Commands

### `logos init`

Initialize Logosyncx in the current directory. Creates:

```
.logosyncx/
├── config.json          # project config
├── USAGE.md             # agent-facing command reference
├── plans/               # plan markdown files
│   └── archive/         # plans moved here by logos gc
├── tasks/               # flat layout: <plan-slug>/NNN-<title>/TASK.md
├── knowledge/           # distilled knowledge files
└── templates/           # plan.md, task.md, knowledge.md templates
```

---

### `logos save`

Create a new plan file (scaffold + frontmatter only — write the body yourself).

```sh
logos save --topic "Topic of this plan" [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--topic` | `-t` | Plan topic — required |
| `--tag` | | Tag — repeatable |
| `--agent` | `-a` | Agent name (e.g. `claude-code`) |
| `--related` | | Related plan filename — repeatable |
| `--depends-on` | | Plan this one depends on (partial name match) — repeatable |

Plans with unresolved `--depends-on` dependencies (not yet distilled) cannot have tasks created against them.

After running `logos save`, open the file and fill in the body using `.logosyncx/templates/plan.md` as a guide.

---

### `logos ls`

List all plans.

```sh
logos ls [flags]
```

| Flag | Description |
|------|-------------|
| `--tag <tag>` | Filter by tag |
| `--since <date>` | Filter to plans after date (YYYY-MM-DD) |
| `--blocked` | Show only blocked plans |
| `--json` | Output JSON with excerpts for agent consumption |

```json
[
  {
    "id": "p-abc123",
    "filename": "20260301-migrate-auth-to-jwt.md",
    "topic": "Migrate auth to JWT",
    "date": "2026-03-01",
    "tags": ["auth", "backend"],
    "excerpt": "The current session-cookie auth cannot scale...",
    "distilled": false,
    "blocked": ""
  }
]
```

---

### `logos refer`

Print a plan's content.

```sh
logos refer --name <partial-name> [--summary]
```

`--summary` returns only the sections listed in `plans.summary_sections` in `config.json` (default: `Background`, `Spec`). Use this to save tokens.

---

### `logos search`

Keyword search across plan topic, tags, and excerpt.

```sh
logos search --keyword <word> [--json]
```

---

### `logos task`

Manage tasks within a plan.

```sh
# Create
logos task create --plan <plan-slug> --title "Title" [--priority high|medium|low] [--depends-on <seq>]

# List
logos task ls [--plan <plan-slug>] [--status open|in_progress|done] [--blocked] [--json]

# View
logos task refer --name <partial-name> [--plan <plan-slug>] [--summary]

# Update
logos task update --name <partial-name> --status <status> [--priority <p>] [--title <t>]

# Search
logos task search --keyword <word> [--plan <plan-slug>] [--json]

# Walkthrough
logos task walkthrough [--name <partial-name>] [--list]

# Delete
logos task delete --name <partial-name> [--force]
```

Tasks are stored as:

```
.logosyncx/tasks/<plan-slug>/NNN-<title>/
├── TASK.md
└── WALKTHROUGH.md    # created automatically when status → done
```

Marking a task `done` automatically creates a `WALKTHROUGH.md` scaffold. Write a walkthrough of what you did — this becomes source material for `logos distill`.

---

### `logos distill`

Distill a completed plan into a knowledge file.

```sh
logos distill --plan <partial-name> [--force] [--dry-run]
```

Pre-flight checks (all hard errors):
1. Plan not found
2. No tasks found for the plan
3. Incomplete tasks exist (open or in_progress)
4. No WALKTHROUGH.md files found
5. Plan already distilled (override with `--force`)

`--dry-run` previews what would be written without creating any files.

On success, creates a knowledge file in `.logosyncx/knowledge/` and marks the plan as `distilled: true`.

---

### `logos sync`

Rebuild the plan index and task index from disk. Run after manually editing `.md` files.

```sh
logos sync
```

---

### `logos gc`

Garbage-collect old plans by moving them to `plans/archive/`.

```sh
logos gc [--linked-days <n>] [--orphan-days <n>] [--dry-run] [--force]
```

| Candidate type | Condition |
|----------------|-----------|
| Strong (linked) | `distilled: true` + all tasks done + age >= `linked_task_done_days` (default: 30) |
| Weak (orphan) | No tasks + age >= `orphan_plan_days` (default: 90) |

`--dry-run` lists candidates without moving anything. `--force` skips the confirmation prompt.

---

### `logos status`

Show uncommitted changes in `.logosyncx/`.

```sh
logos status
```

---

### `logos update`

Update `logos` to the latest release.

```sh
logos update           # download and install
logos update --check   # check only
```

---

## Configuration

`.logosyncx/config.json`:

```json
{
  "version": "2",
  "project": "my-project",
  "agents_file": "AGENTS.md",
  "plans": {
    "summary_sections": ["Background", "Spec"],
    "excerpt_section": "Background"
  },
  "tasks": {
    "default_status": "open",
    "default_priority": "medium",
    "summary_sections": ["What", "Checklist"],
    "excerpt_section": "What"
  },
  "knowledge": {
    "summary_sections": ["Summary", "Key Learnings"],
    "excerpt_section": "Summary"
  },
  "privacy": {
    "filter_patterns": []
  },
  "git": {
    "auto_push": false
  },
  "gc": {
    "linked_task_done_days": 30,
    "orphan_plan_days": 90
  }
}
```

| Key | Description |
|-----|-------------|
| `plans.summary_sections` | Sections returned by `logos refer --summary` |
| `plans.excerpt_section` | Section used as the plan excerpt in the index |
| `tasks.excerpt_section` | Section used as the task excerpt in task ls |
| `knowledge.excerpt_section` | Section used as the knowledge excerpt |
| `gc.linked_task_done_days` | Days after task completion before a distilled plan is GC-eligible |
| `gc.orphan_plan_days` | Days after creation before a plan with no tasks is GC-eligible |

---

## Data layout

```
.logosyncx/
├── config.json
├── USAGE.md
├── index.jsonl             # plan index (auto-managed)
├── task-index.jsonl        # task index (auto-managed)
├── plans/
│   ├── 20260301-migrate-auth-to-jwt.md
│   └── archive/
├── tasks/
│   └── 20260301-migrate-auth-to-jwt/
│       ├── 001-update-token-signing/
│       │   ├── TASK.md
│       │   └── WALKTHROUGH.md
│       └── 002-add-refresh-token-endpoint/
│           └── TASK.md
├── knowledge/
│   └── 20260315-migrate-auth-to-jwt.md
└── templates/
    ├── plan.md
    ├── task.md
    └── knowledge.md
```

Plan filenames use `YYYYMMDD-<slug>.md` so concurrent contributions from multiple agents never conflict.

---

## Agent workflow example

```
Agent: logos ls --json
       # Scans excerpts. Finds "JWT auth" plan from last week.

Agent: logos refer --name jwt-auth --summary
       # Gets Background + Spec. Confirms context without loading the full file.

Agent: logos task ls --plan 20260301-migrate-auth-to-jwt --status open --json
       # Finds open tasks. Claims one.

Agent: logos task update --name update-token-signing --status in_progress

       ... implements the task ...

Agent: logos task update --name update-token-signing --status done
       # WALKTHROUGH.md scaffold is auto-created. Agent writes the walkthrough.

Agent: logos distill --plan 20260301-migrate-auth-to-jwt
       # All tasks done + walkthroughs written → knowledge file created.

Agent: git add .logosyncx/ && git commit -m "logos: distill jwt-auth"
Agent: git push
```

---

## Design principles

- **Agents do semantic search themselves** — `logos ls --json` returns excerpts; the LLM judges relevance. No vector DB or embedding API needed.
- **Token budget awareness** — `logos refer --summary` exists so agents don't load full plans unnecessarily.
- **Scaffold-only pattern** — CLI writes frontmatter; agents write the body using the Write tool. No `--section` flags.
- **git add is automatic; git commit/push is the agent's responsibility** after `logos save`.
- **No interactive prompts** — all commands are fully non-interactive and script-safe.
- **Plain markdown** — every file is human-readable. No database, no binary formats.
