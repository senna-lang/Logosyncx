---
title: "cmd/task: all subcommands + new walkthrough, remove purge"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 10
status: done
priority: high
depends_on: [4, 5]
---

# cmd/task: all subcommands + new walkthrough, remove purge

## What

`cmd/task.go` を全面書き換えする。`task create`・`update`・`ls`・`refer`・`delete`・`search` を新設計に対応させ、新サブコマンド `task walkthrough` を追加。`task purge` を削除。全コマンドに optional `--plan` フラグを追加。

## Why

最大の cmd 変更。task store の新 API（Task 005）を使って、フラットレイアウト・フロントマターのみ更新・WALKTHROUGH 自動生成・ブロック検出を実現する。

## Scope

変更対象のファイル：
- `cmd/task.go` (rewrite — 616 行 → 完全書き換え)
- `cmd/task_create_test.go` (削除 → `cmd/task_test.go` に統合)
- `cmd/task_test.go` (新規または全面更新)

## Checklist

### task create
- [ ] `--plan` required（部分一致で plan を resolve し、plan filename を frontmatter に設定）
- [ ] `--depends-on <int>` repeatable（seq 番号）
- [ ] `--section` フラグを削除
- [ ] フロントマター scaffold のみ書き込み（body なし）
- [ ] 出力: `✓ Created task: .logosyncx/tasks/20260601-auth-refactor/002-add-jwt-middleware/TASK.md  (seq: 2)`
- [ ] テスト: `TestTaskCreate_RequiresPlan`・`TestTaskCreate_AutoAssignsSeq`・`TestTaskCreate_PrintsRelativePath`

### task update
- [ ] `--plan` optional（曖昧なタスク名を解消するため）
- [ ] `--status`: `open | in_progress | done`（`cancelled` を削除）
- [ ] `--add-session` フラグを削除
- [ ] フロントマターのみ更新（ファイル移動なし）
- [ ] `--status in_progress` 時: 依存 seq が done でなければ hard error
  - エラーメッセージ: `error: task "002-add-jwt" is blocked by unfinished tasks: [001-setup-rs256-keys]`
- [ ] `--status done` 時: WALKTHROUGH.md scaffold を自動作成
  - 出力: `✓ WALKTHROUGH.md created: .logosyncx/tasks/20260601-auth-refactor/002-add-jwt-middleware/WALKTHROUGH.md`
- [ ] テスト: `TestTaskUpdate_InProgress_BlockedByDep`・`TestTaskUpdate_Done_CreatesWalkthrough`・`TestTaskUpdate_NoFileMove`

### task ls
- [ ] `--plan` optional（plan slug で絞り込み）
- [ ] `--blocked` フラグ
- [ ] JSON 出力に `seq`・`plan`・`depends_on`・`blocked` を含める
- [ ] テーブル列: `SEQ | DATE | TITLE | STATUS | PRIORITY | PLAN`
- [ ] テスト: `TestTaskLS_PlanFilter`・`TestTaskLS_Blocked`・`TestTaskLS_JSON_IncludesBlockedField`

### task refer
- [ ] `--plan` optional
- [ ] `--with-session` フラグを削除
- [ ] `--summary` 時: `cfg.Tasks.SummarySections` を使用
- [ ] テスト: `TestTaskRefer_Disambiguate_WithPlan`

### task delete
- [ ] `--plan` optional
- [ ] `--force` なし: インタラクティブ確認（`Y/n?` プロンプト）
- [ ] `--force` あり: 確認スキップ
- [ ] タスクディレクトリごと削除（`os.RemoveAll`）
- [ ] テスト: `TestTaskDelete_RemovesDir`・`TestTaskDelete_Force_SkipsPrompt`

### task search
- [ ] `--plan` optional（plan に絞って検索）
- [ ] テスト: `TestTaskSearch_PlanFilter`

### task walkthrough (NEW)
- [ ] `--plan <partial>` のみ指定 → plan 配下の全タスクのウォークスルー一覧
  - 出力形式:
    ```
    SEQ  TITLE                 WALKTHROUGH
    001  Setup RS256 keys      [filled]
    002  Add JWT middleware     [scaffold only]
    003  Write auth tests      -
    ```
  - `[filled]`: WALKTHROUGH.md が存在し内容あり
  - `[scaffold only]`: WALKTHROUGH.md が存在するが空（comment のみ）
  - `-`: WALKTHROUGH.md が存在しない
- [ ] `--plan <partial>` + `--name <partial>` → 該当タスクの WALKTHROUGH.md 内容を出力
- [ ] fill status 検出: 非見出し・非 HTML コメント行が存在すれば "filled"
- [ ] テスト: `TestTaskWalkthrough_ListMode`・`TestTaskWalkthrough_PrintContent`・`TestTaskWalkthrough_FillStatusDetection`

### task purge (DELETE)
- [ ] `logos task purge` サブコマンドを削除
- [ ] `cancelled` ステータス関連のすべての参照を削除

### 共通
- [ ] `go test ./cmd/ -run TestTask` パス確認
- [ ] `go test ./...` 全体パス確認

## Notes

- `task delete` はエージェント向けに `--force` を提供し、人間向けにはインタラクティブ確認を提供する設計（SPEC §6.5）
- `cmd/task_create_test.go` を `cmd/task_test.go` に統合するか、ファイル名を維持するかはどちらでも OK
