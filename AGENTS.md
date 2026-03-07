# Agent Instructions

This project uses **logos** for plan and task tracking, stored in `.logosyncx/`.

## Task Tracking

```bash
logos task ls --status open --json                        # Find available work
logos task refer --name <name>                            # View task details
logos task update --name <name> --status in_progress      # Claim a task
logos task update --name <name> --status done             # Complete a task
logos sync                                                # Rebuild plan and task indexes
```

**Session completion is mandatory** — see the workflow below.

## MANDATORY: logos Command Triggers

The following triggers are **not optional**. When any of these conditions occur, you MUST run the corresponding command immediately.

### Starting any work session

**ALWAYS run this first, before doing anything else:**

```bash
logos ls --json
```

Scan the `topic`, `tags`, and `excerpt` fields to find relevant past plans.
If anything looks relevant, run:

```bash
logos refer --name <filename> --summary
```

### Mid-session triggers

| If the user says (any variation) | You MUST run |
|---|---|
| "save this plan", "log this", "記録して" | `logos save --topic "..."` then write body with Write tool |
| "make that a task", "タスクにして" | `logos task create --plan <name> --title "..."` |
| "what did we do last time", "前回の続き" | `logos ls --json` then `logos refer --name <name>` |
| "continue from last session", "前回の続きから" | `logos ls --json` -> find latest relevant -> `logos refer --name <name>` |

### When saving a plan

```bash
logos save --topic "short description" --tag <tag> --agent <agent-name>
```

Then open the created file and write the body using the template:

```bash
# Read the plan template first
cat .logosyncx/templates/plan.md

# Write the body directly into the plan file using the Write tool
```

Do NOT use `--section` flags — they do not exist in v2.

### When creating a task

```bash
logos task create --plan <plan-slug> --title "Implement the thing" --priority high
```

Then open the created TASK.md and write the body using the template:

```bash
cat .logosyncx/templates/task.md
```

---

## Landing the Plane (Session Completion)

**When ending a work session**, complete ALL steps below. Work is NOT complete until `git push` succeeds.

1. **File tasks for remaining work:**
   ```bash
   logos task create --plan <plan-slug> --title "..."
   ```
2. **Run quality gates** (if code changed): `go test ./...`
3. **Update task status:**
   ```bash
   logos task update --name <name> --status done
   ```
4. **Save this plan:**
   ```bash
   logos save --topic "..."
   # Write the body into the plan file
   ```
5. **PUSH TO REMOTE** — MANDATORY:
   ```bash
   git pull --rebase
   logos sync
   git add .logosyncx/
   git commit -m "logos: save plan \"<topic>\""
   git push
   ```
6. **Verify** — `git status` MUST show "up to date with origin"

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing
- Always read the template before writing any document body

---

## Full Command Reference

See `.logosyncx/USAGE.md` for the complete command reference.
