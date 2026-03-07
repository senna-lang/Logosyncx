---
title: "cmd/init: plans/, knowledge/, templates/, v2 config, usageMD"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 7
status: done
priority: medium
depends_on: [1, 2, 6]
---

# cmd/init: plans/, knowledge/, templates/, v2 config, usageMD

## What

`cmd/init.go` を v2 ディレクトリ構造に対応させる。作成ディレクトリを `plans/`・`plans/archive/`・`knowledge/`・`templates/` に変更。デフォルトテンプレートファイル 3 点を書き込む。config.json を v2 スキーマで生成。`usageMD` 定数を新コマンド面に合わせて全面書き換え。

## Why

`logos init` は新規プロジェクトのエントリポイント。v1 の `sessions/`・status サブディレクトリを作成しないよう修正する必要がある。

## Scope

変更対象のファイル：
- `cmd/init.go` (update)
- `cmd/init_test.go` (rewrite)

## Checklist

### ディレクトリ構造
- [ ] 作成する dirs: `plans/`, `plans/archive/`, `knowledge/`, `templates/`
- [ ] 作成しない dirs: `sessions/`, `tasks/open/`, `tasks/in_progress/`, `tasks/done/`, `tasks/cancelled/`
- [ ] `.logosyncx/` が既に存在する場合は hard error（マイグレーションなし）

### テンプレートファイル
- [ ] `templates/plan.md` をデフォルト内容で書き込み（SPEC §4.4 の内容）
- [ ] `templates/task.md` をデフォルト内容で書き込み（SPEC §4.4 の内容）
- [ ] `templates/knowledge.md` をデフォルト内容で書き込み（SPEC §4.4 の内容）

### config.json
- [ ] `config.Default()` が v2 スキーマを返すことを確認（Task 001 済み）
- [ ] `"sessions"` キーが出力されないことを確認
- [ ] `"plans"`, `"knowledge"` キーが出力されることを確認

### usageMD 定数
- [ ] `usageMD` 定数を完全書き換え。以下の新コマンドを記述:
  - `logos save --topic ... --depends-on ...` (scaffold only)
  - `logos ls --blocked --json`
  - `logos task create --plan ... --depends-on <seq>`
  - `logos task update --plan ... --name ... --status done`
  - `logos task ls --plan ... --blocked --json`
  - `logos task walkthrough --plan ...`
  - `logos distill --plan ...`
  - エージェント向け注意事項（テンプレートを先に読む、body は Write ツールで書く）
- [ ] `--section` フラグへの言及を削除
- [ ] `logos task purge` への言及を削除

### テスト更新
- [ ] `TestInit_CreatesPlansDir`
- [ ] `TestInit_CreatesKnowledgeDir`
- [ ] `TestInit_CreatesTemplatesDir`
- [ ] `TestInit_CreatesTemplateFiles` (plan.md, task.md, knowledge.md の存在確認)
- [ ] `TestInit_ConfigVersion2`
- [ ] `TestInit_NoStatusSubdirs` (tasks/open/ が存在しないことを確認)
- [ ] `TestInit_AlreadyInitialized_Error`
- [ ] `go test ./cmd/ -run TestInit` パス確認

## Notes

- `detectAgentsFile`・`appendAgentsLine` は変更不要
- USAGE.md の内容は Task 013 で `.logosyncx/USAGE.md`（live ファイル）とも同期させる
