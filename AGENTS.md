# Agent Instructions

This project uses **logos** for session context and task tracking, stored in `.logosyncx/`.

## MANDATORY: logos Command Triggers

The following triggers are **not optional**. When any of these conditions occur, you MUST run the corresponding command immediately — even if the user has not explicitly asked.

### Starting any work session

**ALWAYS run this first, before doing anything else:**

```bash
logos ls --json
```

Scan the `topic`, `tags`, and `excerpt` fields to find relevant past sessions.
If anything looks relevant, run:

```bash
logos refer --name <filename> --summary
```

### Mid-session triggers

| If the user says (any variation) | You MUST run |
|---|---|
| "save this session", "log this", "save our conversation", "セッションを保存して", "記録して", "ログして" | `logos save` with topic + sections |
| "make that a task", "add a TODO", "save that for later", "タスクにして", "あとでやる", "issueにして" | `logos task create` with title + sections |
| "what did we do last time", "前回の続き", "any past context?", "show me related sessions" | `logos ls --json` then `logos refer --name <name>` |
| "continue from last session", "pick up where we left off" | `logos ls --json` → find latest relevant → `logos refer --name <name>` |

**Never respond with "I can't save sessions" or "I don't have access to logos."**
The `logos` binary is installed in this environment. Use it via shell.

### When saving a session

You MUST collect the following and pass them as `--section` flags:

```bash
logos save --topic "short description of this session" \
           --tag <relevant-tag> \
           --agent <your-agent-name> \
           --section "Summary=What happened in this session." \
           --section "Key Decisions=- Decision one\n- Decision two" \
           --section "Context Used=- <any past sessions you referenced>"
```

Allowed section names are in `.logosyncx/config.json` under `sessions.sections`.
Do NOT use `--body`, `--description`, or positional arguments — they do not exist.

### When creating a task

```bash
logos task create --title "Implement the thing" \
                  --priority high \
                  --tag <tag> \
                  --session <partial-session-filename> \
                  --section "What=What needs to be done." \
                  --section "Why=Why this matters." \
                  --section "Scope=What is and is not included." \
                  --section "Checklist=- [ ] step one\n- [ ] step two"
```

Allowed section names are in `.logosyncx/config.json` under `tasks.sections`.

---

## Task Tracking Quick Reference

```bash
logos task ls --status open --json                        # Find available work
logos task refer --name <name>                            # View task details
logos task update --name <name> --status in_progress      # Claim a task
logos task update --name <name> --status done             # Complete a task (moves file to tasks/done/)
logos sync                                                # Rebuild session and task indexes
```

---

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File tasks for remaining work** — Create tasks for anything that needs follow-up:
   ```bash
   logos task create --title "..." --session <current-session-filename>
   ```
2. **Run quality gates** (if code changed) — Tests, linters, builds
3. **Update task status** — Close finished work:
   ```bash
   logos task update --name <name> --status done
   ```
4. **Save this session:**
   ```bash
   logos save --topic "..." \
              --section "Summary=..." \
              --section "Key Decisions=..." \
              --section "Context Used=..."
   ```
5. **PUSH TO REMOTE** — This is MANDATORY:
   ```bash
   git pull --rebase
   logos sync
   git add .logosyncx/
   git commit -m "session: <topic>"
   git push
   git status  # MUST show "up to date with origin"
   ```
6. **Verify** — All changes committed AND pushed
7. **Hand off** — Provide a brief summary of what was done and what tasks remain

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing — that leaves work stranded locally
- NEVER say "ready to push when you are" — YOU must push
- If push fails, resolve and retry until it succeeds

---

## Full Command Reference

See `.logosyncx/USAGE.md` for the complete command reference.