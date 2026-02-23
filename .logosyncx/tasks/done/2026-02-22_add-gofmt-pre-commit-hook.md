---
id: t-gofmt01
date: 2026-02-22T20:29:24.68231+09:00
title: Add go fmt pre-commit hook via Makefile and scripts/hooks
status: done
priority: medium
session: 2026-02-22_go-fmt-pre-commit-hook-plan.md
tags:
    - go
    - tooling
    - git-hooks
    - pre-commit
    - makefile
assignee: ""
---

## What

`go fmt` をコミット前に自動チェックする仕組みを導入する。
具体的には以下の2ファイルを新規作成し、`git config core.hooksPath` で有効化する。

1. `Makefile` — `setup` / `fmt` / `lint` / `test` ターゲットを定義
2. `scripts/hooks/pre-commit` — `gofmt -l .` でフォーマット違反を検出するシェルスクリプト

## Why

現状、`go fmt` の実行を強制する仕組みが存在しない。
`.git/hooks/` にも `Makefile` にも何も設定されていないため、フォーマット違反のコードがコミットされるリスクがある。
Goプロジェクトで最も一般的な「Makefile + `scripts/hooks/` + `git config core.hooksPath`」方式を採用し、チームメンバーが `make setup` を1回実行するだけでhookが有効になるようにする。

## Scope

- `Makefile` — 新規作成（`setup`, `fmt`, `lint`, `test` ターゲット）
- `scripts/hooks/pre-commit` — 新規作成（`gofmt -l .` チェック、実行権限付与が必要）
- `README.md` — セットアップ手順に `make setup` の説明を追記

## Checklist

- [ ] `Makefile` を作成し `setup` / `fmt` / `lint` / `test` ターゲットを定義
- [ ] `scripts/hooks/` ディレクトリを作成
- [ ] `scripts/hooks/pre-commit` を作成（`gofmt -l .` チェック）
- [ ] `scripts/hooks/pre-commit` に実行権限を付与 (`chmod +x`)
- [ ] `make setup` を実行して動作確認
- [ ] フォーマット違反ファイルがある状態でコミットを試み、hookが弾くことを確認
- [ ] `README.md` にセットアップ手順を追記

## Notes

### Makefile の実装内容

```makefile
.PHONY: setup fmt lint test

setup:
	git config core.hooksPath scripts/hooks

fmt:
	go fmt ./...

lint:
	go vet ./...

test:
	go test ./...
```

### scripts/hooks/pre-commit の実装内容

```sh
#!/bin/sh
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
  echo "The following files are not formatted (run 'make fmt'):"
  echo "$UNFORMATTED"
  exit 1
fi
```

関連セッション: 2026-02-22_go-fmt-pre-commit-hook-plan.md