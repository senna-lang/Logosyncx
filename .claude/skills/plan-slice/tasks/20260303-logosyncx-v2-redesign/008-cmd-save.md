---
title: "cmd/save: scaffold pattern, --depends-on, delete sections.go"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 8
status: done
priority: medium
depends_on: [2, 3, 5]
---

# cmd/save: scaffold pattern, --depends-on, delete sections.go

## What

`cmd/save.go` を scaffold パターンに書き換える。`--section`・`--task` フラグを削除し、`--depends-on`（繰り返し可、plan 名の部分一致）を追加。`plans/YYYYMMDD-<slug>.md` にフロントマターのみ書き込み、パスを出力する。`cmd/sections.go` を削除する。

## Why

新設計では CLI はフロントマター scaffold のみ作成し、body はエージェントが Write ツールで直接書く。`sections.go` は `--section` フラグのためだけに存在していたため不要になる。

## Scope

変更対象のファイル：
- `cmd/save.go` (rewrite)
- `cmd/save_test.go` (rewrite)
- `cmd/sections.go` (DELETE)

## Checklist

### フラグ変更
- [ ] 削除: `--section`, `--task`
- [ ] 追加: `--depends-on` (repeatable, string — plan 名の部分一致)
- [ ] 維持: `--topic` (required), `--tag`, `--agent`, `--related`

### runSave ロジック
- [ ] `plan.LoadAll` で既存プランを読み込み
- [ ] `--depends-on` の各値を部分一致で resolve
  - 0件 → hard error: `error: plan "X" not found`
  - 2件以上 → hard error: `error: ambiguous plan name "X": matches [A, B]`
- [ ] `plan.FileName()` で `YYYYMMDD-<slug>.md` を生成
- [ ] `plan.DefaultTasksDir(filename)` で `tasks_dir` を自動設定
- [ ] `plan.Write()` でフロントマター scaffold のみ書き込み（body なし）
- [ ] `index.Append()` でインデックスに追加
- [ ] gitutil.Add でステージング（best-effort）
- [ ] 出力: `✓ Created plan: .logosyncx/plans/20260601-auth-refactor.md`

### 削除
- [ ] `cmd/sections.go` を削除
- [ ] `buildBodyFromSections`, `parseSectionFlag`, `allowedSectionSet`, `warnMissingSections` の呼び出しをすべて削除
- [ ] `warnPrivacy` は plan body がないため不要（またはシンプルに削除）

### テスト
- [ ] `TestSave_CreatesInPlansDir`
- [ ] `TestSave_FileNameFormat_YYYYMMDD` (YYYYMMDD プレフィックスを確認)
- [ ] `TestSave_TasksDirSetInFrontmatter`
- [ ] `TestSave_ScaffoldOnly_NoBody`
- [ ] `TestSave_DependsOn_ResolvesPartialMatch`
- [ ] `TestSave_DependsOn_Ambiguous_HardError`
- [ ] `TestSave_DependsOn_NotFound_HardError`
- [ ] `go test ./cmd/ -run TestSave` パス確認
- [ ] `go test ./...` 全体パス確認（sections.go 削除の影響確認）

## Notes

- `sections.go` 削除前に `save.go`・`task.go` 内の参照をすべて除去すること
- `task.go` での `buildBodyFromSections` 参照は Task 010 で削除するため、このタスクと Task 010 を同一 PR にするか、先に `task.go` の該当箇所だけ削除する
- `warnPrivacy` 削除は optional（body がないので実質スキャン対象なし）
