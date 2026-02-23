# Agent Instructions

This project uses **logos tasks** for issue tracking, stored in `.logosyncx/tasks/`.

## Quick Reference

```bash
logos task ls --status open --json          # Find available work
logos task refer <name>                     # View task details
logos task update <name> --status in_progress  # Claim work
logos task update <name> --status done      # Complete work (deletes the file)
logos sync                                  # Rebuild session and task indexes
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File tasks for remaining work** - Create tasks for anything that needs follow-up (`logos task create --title "..." --session <current-session>`)
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update task status** - Close finished work (`logos task update <name> --status done --force`), update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   logos sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds