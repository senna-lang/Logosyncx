---
id: t-10bead
date: 2026-02-21T23:38:55.938086+09:00
title: Fix logos save UX issues from agent feedback
status: open
priority: high
session: 2026-02-21_logosyncx-ux-feedback.md
tags:
    - bugfix
    - ux
    - save
    - template
assignee: ""
---

## What

`logos save` で発生した4つのUX問題を修正する。

1. `template.md` から `id`/`date` フィールドを削除（YAML parse エラーの根本原因）
2. `Session.Date` を `*time.Time` に変更して空値・省略を許容
3. `logos save` のエラーメッセージに `{{` 検出時のヒントを追加
4. Makefile に `build` / `install` / `test` ターゲットを追加

## Why

エージェントが実際に `logos save` を使おうとした際、テンプレートの `{{id}}` / `{{date}}` が YAML の flow mapping として誤解釈されてパースエラーが発生した。また `date: ""` も `time.Time` 型が空文字を受け付けないため自動補完ロジックに到達できなかった。エラーメッセージも内部エラーをそのまま出力しており原因特定が困難だった。

## Scope

- `.logosyncx/template.md` — `id`/`date` 行を削除
- `pkg/session/session.go` — `Date time.Time` → `Date *time.Time` (omitempty)、`FileName()` で nil 時は `time.Now()` を使用
- `cmd/save.go` — `s.Date.IsZero()` → `s.Date == nil`、`{{` を含む parse エラー時にヒントを付与
- `Makefile` — 新規作成（`build`, `install`, `test` ターゲット）

## Checklist
- [ ] template.md から id/date 行を削除
- [ ] Session.Date を *time.Time に変更
- [ ] FileName() の nil ガード追加
- [ ] save.go のゼロ値チェックを nil チェックに変更
- [ ] parse エラー時に {{ 検出ヒントを追加
- [ ] Makefile 作成
- [ ] go test ./... がパスすることを確認

## Notes

関連セッション: 2026-02-21_logosyncx-ux-feedback.md
beads issue 参照: logosyncx-44g, logosyncx-lbn, logosyncx-r1l, logosyncx-98p
