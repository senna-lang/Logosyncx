---
title: "Cleanup: delete pkg/session, update USAGE.md + usageMD"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 13
status: done
priority: low
depends_on: [7, 8, 9, 10, 11, 12]
---

# Cleanup: delete pkg/session, update USAGE.md + usageMD

## What

全タスク完了後の最終クリーンアップ。`pkg/session/session.go` を削除し、`.logosyncx/USAGE.md`（live ファイル）と `cmd/init.go` の `usageMD` 定数を同期させる。全テストが通ることを最終確認する。

## Why

`pkg/session` への import がすべて `pkg/plan` に移行済みであることを確認してから削除する。USAGE.md の同期はドキュメントポリシー（CLAUDE.md）の要件。

## Scope

変更対象のファイル：
- `pkg/session/session.go` (DELETE)
- `pkg/session/` ディレクトリ (DELETE)
- `.logosyncx/USAGE.md` (update)
- `cmd/init.go` の `usageMD` 定数 (sync)

## Checklist

### pkg/session 削除
- [ ] `grep -r "pkg/session"` で import が残っていないことを確認
- [ ] `pkg/session/session.go` を削除
- [ ] `pkg/session/` ディレクトリを削除
- [ ] `go build ./...` でコンパイルエラーがないことを確認

### USAGE.md 更新（`.logosyncx/USAGE.md`）
- [ ] `cmd/init.go` の `usageMD` 定数と内容を完全同期
- [ ] 以下の変更が反映されていることを確認:
  - `logos save` — scaffold パターン、`--depends-on` フラグ
  - `logos ls` — `--blocked` フラグ、JSON の `blocked`・`distilled` フィールド
  - `logos task create` — `--plan` required、`--depends-on <seq>`
  - `logos task update` — フロントマターのみ、WALKTHROUGH 自動生成
  - `logos task ls` — `--plan`・`--blocked`
  - `logos task walkthrough` — 新コマンド
  - `logos distill` — 新コマンド
  - `--section` フラグへの言及を削除
  - `logos task purge` への言及を削除
  - cancelled ステータスへの言及を削除
  - エージェント向け注意事項（テンプレートを先に読む旨）

### 最終確認
- [ ] `go test ./...` 全体パス確認
- [ ] `go build -o logos .` でビルド成功
- [ ] スモークテスト（以下を順番に実行）:
  ```sh
  ./logos init
  ./logos save --topic "test-plan"
  ./logos task create --title "Task one" --plan "test-plan"
  ./logos task create --title "Task two" --plan "test-plan" --depends-on 1
  ./logos task update --plan "test-plan" --name "001" --status done
  ./logos task update --plan "test-plan" --name "002" --status in_progress
  ./logos task ls --plan "test-plan" --json
  ./logos distill --plan "test-plan" --dry-run
  ./logos ls --json
  ```

## Notes

- `usageMD` 定数と `.logosyncx/USAGE.md` は常に同期（CLAUDE.md のドキュメント更新チェックリスト参照）
- このタスクはすべての依存タスク完了後に着手すること
