---
title: "Logosyncx v2 — Spec-Driven Redesign"
date: 2026-03-03
status: open
tasks_dir: .claude/skills/plan-slice/tasks/20260303-logosyncx-v2-redesign/
---

# Plan: Logosyncx v2 — Spec-Driven Redesign

## 概要

logosyncx を session-centric なツールから spec-driven な開発ワークフローツールへ全面改定する。sessions/ → plans/、status-in-path → status-in-frontmatter の flat task layout、ウォークスルーと蒸留（distillation）機能を新設する。後方互換なし・マイグレーションなし。

## 背景・理由

現行の「セッション」モデルは会話記録の保存には便利だが、実際の開発フロー（計画→タスク→実装→知識蒸留）を表現できない。SPEC.md で定義した4ステージライフサイクル（Plan → Task → Walkthrough → Knowledge）を CLI のデータモデルとして採用することで、エージェントが spec-driven 開発を自然に実践できるようにする。

## アーキテクチャ / 設計

### ディレクトリ構造（目標）
```
.logosyncx/
├── config.json         (version "2")
├── templates/          NEW: plan.md / task.md / knowledge.md
├── plans/              (旧 sessions/)
├── tasks/
│   └── 20260601-auth-refactor/   (plan slug ごとのグループ、status サブディレクトリなし)
│       ├── 001-setup-rs256-keys/TASK.md
│       └── 002-add-jwt-middleware/TASK.md
└── knowledge/          NEW: 蒸留済みファイル
```

### 主要な設計決定
- **Status in frontmatter only** — ファイルは移動しない
- **Sequential task numbering** — `001`, `002`, ... で依存を seq 番号で表現
- **Scaffold pattern** — CLI はフロントマターだけ作成、body はエージェントが Write ツールで書く
- **Dependency model** — Plans: plan filename のリスト。Tasks: 同一プラン内の seq 番号のリスト
- **Distillation** — `logos distill` が source material を組み立て、エージェントが knowledge ファイルを埋める

### 影響ファイル（変更対象）
```
pkg/config/config.go          rewrite (Sessions→Plans, remove SectionConfig, add Knowledge)
pkg/plan/plan.go               NEW (replaces pkg/session/session.go)
pkg/index/index.go             update (Entry struct, Rebuild scans plans/)
pkg/knowledge/knowledge.go     NEW
internal/task/task.go          rewrite (seq, plan, depends_on as ints, no cancelled)
internal/task/store.go         rewrite (flat layout, no status dirs, CreateWalkthrough, NextSeq)
internal/task/filter.go        update (Plan filter, Blocked filter)
internal/task/index.go         update (new TaskJSON fields)
cmd/init.go                    update (plans/, knowledge/, templates/, v2 config, usageMD)
cmd/save.go                    rewrite (scaffold only, --depends-on, plans/)
cmd/ls.go                      update (--blocked, blocked in JSON)
cmd/refer.go                   update (use pkg/plan)
cmd/search.go                  update (use pkg/plan)
cmd/task.go                    rewrite (create/update/ls/refer/delete/search + new walkthrough)
cmd/sync.go                    simplify (no migration)
cmd/gc.go                      update (plans/ instead of sessions/)
cmd/distill.go                 NEW
cmd/sections.go                DELETE
pkg/session/session.go         DELETE
```

## タスク一覧

| # | ファイル | タイトル | 優先度 | 依存 |
|---|---------|---------|--------|------|
| 001 | 001-config-v2.md | Config v2: Sessions→Plans, remove SectionConfig, add Knowledge | high | - |
| 002 | 002-plan-package.md | New pkg/plan package (replaces pkg/session) | high | 001 |
| 003 | 003-index-update.md | Update pkg/index Entry for plans, add Blocked field | high | 002 |
| 004 | 004-task-data-model.md | Task data model: seq, plan, depends_on ints, remove cancelled | high | 001 |
| 005 | 005-task-store-rewrite.md | Task store: flat layout, NextSeq, CreateWalkthroughScaffold | high | 004 |
| 006 | 006-knowledge-package.md | New pkg/knowledge package | medium | 001 |
| 007 | 007-cmd-init.md | cmd/init: plans/, knowledge/, templates/, v2 config, usageMD | medium | 001,002,006 |
| 008 | 008-cmd-save.md | cmd/save: scaffold pattern, --depends-on, delete sections.go | medium | 002,003,005 |
| 009 | 009-cmd-ls-refer-search.md | cmd/ls + refer + search: plans/, --blocked flag | medium | 002,003 |
| 010 | 010-cmd-task-rewrite.md | cmd/task: all subcommands + new walkthrough, remove purge | high | 004,005 |
| 011 | 011-cmd-distill.md | cmd/distill: new command, pre-flight, knowledge scaffold | medium | 002,005,006 |
| 012 | 012-cmd-sync-gc.md | cmd/sync + gc: simplify sync, GC uses plans/ | low | 002,005 |
| 013 | 013-cleanup-usage.md | Cleanup: delete pkg/session, update USAGE.md + usageMD | low | 007,008,009,010,011,012 |

## リスク・考慮事項

- `go test ./...` を各 PR 後に必ず通す（TDD）
- `cmd/init.go` の `usageMD` 定数と `.logosyncx/USAGE.md` は常に同期すること
- `logos distill` は knowledge ファイル書き込み成功後にのみ `distilled: true` を書く（冪等性）
- `logos task update --status in_progress` は依存 seq が done でなければ hard error
- `pkg/session` の削除はすべての import が `pkg/plan` に移行済みであることを確認してから
