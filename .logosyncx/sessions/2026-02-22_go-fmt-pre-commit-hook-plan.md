---
id: 5a6e39
date: 2026-02-22T20:29:45.98987+09:00
topic: go-fmt-pre-commit-hook-plan
tags:
    - go
    - tooling
    - git-hooks
    - pre-commit
    - makefile
agent: claude-sonnet-4-6
related: []
---

## Summary

`go fmt` をコミット前に自動実行する仕組みについて検討した。現状はgit hooksもMakefileも設定されていないことを確認。Goプロジェクトで最も一般的な「Makefile + `scripts/hooks/` + `git config core.hooksPath`」方式を採用することに決定。実装は後日行う。

## Key Decisions

- 採用方式: Makefile + `scripts/hooks/pre-commit` + `git config core.hooksPath`
- `make setup` を1回実行するだけでhookが有効になる構成にする
- pre-commit hook は `gofmt -l .` で未フォーマットファイルを検出し、あれば `exit 1` で弾く
- `Makefile` には `setup`, `fmt`, `lint`, `test` ターゲットを追加する

## Context Used

- `.git/hooks/` を確認 → サンプルファイルのみ、有効なhookなし
- `Makefile` 不在を確認
- `.golangci.yml` 等のlinter設定不在を確認

## Notes

実装予定の構成:

Makefile に `setup` / `fmt` / `lint` / `test` ターゲットを追加。
`scripts/hooks/pre-commit` に `gofmt -l .` チェックスクリプトを配置。
開発者は `make setup` を1回実行するだけでhookが有効になる。

実装タスク: 2026-02-22_add-gofmt-pre-commit-hook.md