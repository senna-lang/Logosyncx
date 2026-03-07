---
title: "cmd/distill: new command, pre-flight, knowledge scaffold"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 11
status: done
priority: medium
depends_on: [2, 5, 6]
---

# cmd/distill: new command, pre-flight, knowledge scaffold

## What

`cmd/distill.go` を新規作成する。`logos distill --plan <name>` がすべてのタスクの完了とウォークスルーの存在を確認し、source material（plan body + 全 walkthrough）を組み立てて `knowledge/YYYYMMDD-<slug>.md` を書き込む。成功後にのみ plan の `distilled: true` をセットする。

## Why

四段階ライフサイクルの最終ステップ。`logos distill` を通じてエージェントが知識を蒸留できるようになる。

## Scope

変更対象のファイル：
- `cmd/distill.go` (新規)
- `cmd/distill_test.go` (新規)

## Checklist

### フラグ
- [ ] `--plan <partial>` required
- [ ] `--force` — 既に distilled な plan を再蒸留
- [ ] `--dry-run` — 書き込まずにプレビューのみ

### Pre-flight checks（すべて hard error）
1. [ ] Plan が見つからない → `error: plan "X" not found`
2. [ ] `tasks_dir` 配下にタスクが存在しない → `error: no tasks found for plan "X"`
3. [ ] open または in_progress のタスクが存在する → `error: incomplete tasks: [list]. Complete or delete them first.`
4. [ ] WALKTHROUGH.md が 1 つも存在しない → `error: no walkthroughs found for plan "X" — mark tasks done first`
5. [ ] `plan.Distilled == true` かつ `--force` なし → `error: plan "X" already distilled. Re-run with --force to overwrite.`

`--force` は check 5 のみをスキップする。

### Source material の組み立て
- [ ] plan body を読み込む（plan.LoadFile から Body フィールド）
- [ ] `tasks_dir` 配下の全タスクをロード
- [ ] 各タスクの `WALKTHROUGH.md` を読み込む
- [ ] HTML コメントブロックを構築:
  ```
  <!-- SOURCE MATERIAL — read this, fill in the sections below, then remove this block. -->
  <!--
  ## Plan: <topic>

  <plan body>

  ---
  ## Walkthrough: NNN <task title>

  <walkthrough content>
  -->
  ```

### Knowledge ファイル書き込み
- [ ] `templates/knowledge.md` を読み込む
- [ ] `knowledge.Write(projectRoot, k, sourceBlock, templateContent)` を呼ぶ
- [ ] `--dry-run` 時は書き込まず preview のみ出力

### Plan 更新（書き込み成功後のみ）
- [ ] `plan.Distilled = true` をセット
- [ ] plan ファイルのフロントマターを書き直す
- [ ] `index.Rebuild` でインデックスを更新
- [ ] gitutil.Add（best-effort）

### 出力
```
Plan:      auth-refactor
Tasks:     001 Setup RS256 keys [filled]
           002 Add JWT middleware [filled]
           003 Write auth tests [filled]

✓ Knowledge file written: .logosyncx/knowledge/20260610-auth-refactor.md
✓ Plan marked as distilled: .logosyncx/plans/20260601-auth-refactor.md

Next: Open .logosyncx/knowledge/20260610-auth-refactor.md and fill in the sections.
```

### テスト
- [ ] `TestDistill_CreatesKnowledgeFile`
- [ ] `TestDistill_SetsDistilledTrue`
- [ ] `TestDistill_AlreadyDistilled_Error`
- [ ] `TestDistill_Force_OverridesAlreadyDistilled`
- [ ] `TestDistill_DryRun_NoWrite`
- [ ] `TestDistill_IncompleteTasks_Error`
- [ ] `TestDistill_NoWalkthroughs_Error`
- [ ] `go test ./cmd/ -run TestDistill` パス確認

## Notes

- `distilled: true` のセットは knowledge ファイル書き込み成功後のみ（SPEC §9.4 の設計原則）
- `--dry-run` はすべての pre-flight を実行するが、ファイル書き込みとフロントマター更新をスキップする
