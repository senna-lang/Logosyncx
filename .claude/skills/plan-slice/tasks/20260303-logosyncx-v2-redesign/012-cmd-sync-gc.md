---
title: "cmd/sync + gc: simplify sync, GC uses plans/"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 12
status: done
priority: low
depends_on: [2, 5]
---

# cmd/sync + gc: simplify sync, GC uses plans/

## What

`cmd/sync.go` からマイグレーションロジックを削除しシンプルにする。`cmd/gc.go` を `plans/` ベースに更新し、GC 基準を v2 スキーマに合わせる。

## Why

マイグレーション不要のため sync はインデックス再構築のみでよい。GC は `sessions/` ではなく `plans/` を対象にする必要がある。

## Scope

変更対象のファイル：
- `cmd/sync.go` (simplify)
- `cmd/sync_test.go` (update)
- `cmd/gc.go` (update)

## Checklist

### cmd/sync.go
- [ ] `config.Migrate()` の呼び出しを削除（Task 001 で関数ごと・呼び出し元ともに削除済み）
- [ ] `index.Rebuild` が `plans/` を走査することを確認（Task 003 済み）
- [ ] `store.RebuildTaskIndex` がフラットレイアウトを走査することを確認（Task 005 済み）
- [ ] 出力メッセージを更新: `"Rebuilt plan index"`, `"Rebuilt task index"`
- [ ] テスト更新

### cmd/gc.go
- [ ] `session.LoadAll` → `plan.LoadAll`（`pkg/plan` を使用）
- [ ] `session.Archive` → `plan.Archive`
- [ ] `session.ArchiveDir` → `plan.ArchiveDir`
- [ ] GC 候補の判定を更新:
  - **Protected**: `plan.TasksDir` 配下に open/in_progress のタスクが存在する
  - **Strong candidate**: `plan.Distilled == true` + 全タスク done + 経過日数 ≥ `linked_task_done_days`
  - **Weak candidate**: `plan.TasksDir` が空（タスクなし）+ 経過日数 ≥ `orphan_plan_days`
- [ ] `cfg.GC.OrphanSessionDays` → `cfg.GC.OrphanPlanDays`（Task 001 で変更済み）
- [ ] GC 候補の表示を更新（"session" → "plan"）
- [ ] `logos gc purge` は変更なし（archive から永続削除）
- [ ] テスト更新（フィクスチャを `plans/` 向けに変更）

## Notes

- `cmd/gc.go` の task ローディングは `store.List(Filter{Plan: planSlug})` を使って行う
- `distilled: true` の plan のみが strong candidate になるため、GC 前に `logos distill` が必要というワークフローになる
