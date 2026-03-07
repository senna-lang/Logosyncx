---
title: "cmd/ls + refer + search: plans/, --blocked flag"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 9
status: done
priority: medium
depends_on: [2, 3]
---

# cmd/ls + refer + search: plans/, --blocked flag

## What

`cmd/ls.go`・`cmd/refer.go`・`cmd/search.go` を `pkg/plan` を使うよう更新する。`ls` に `--blocked` フラグを追加、JSON に `blocked`・`distilled` フィールドを含める。`refer` は `cfg.Plans.SummarySections` を使う。

## Why

これら 3 コマンドは `pkg/session` に依存しているため、`pkg/plan` に移行する必要がある。変更量が小さく、まとめて 1 PR に収まる。

## Scope

変更対象のファイル：
- `cmd/ls.go` (update)
- `cmd/ls_test.go` (update)
- `cmd/refer.go` (update)
- `cmd/refer_test.go` (update)
- `cmd/search.go` (update)
- `cmd/search_test.go` (update)

## Checklist

### cmd/ls.go
- [ ] `--blocked` フラグを追加
- [ ] `runLS(tag, since string, asJSON, blocked bool) error`
- [ ] インデックス読み込み: `index.ReadAll` → 変更なし（Entry 構造体が更新済み）
- [ ] `--blocked` 時: `entry.Blocked == true` でフィルタ
- [ ] JSON 出力に `blocked`・`distilled`・`tasks_dir`・`depends_on` を含める（Entry 構造体に追加済み）
- [ ] テーブル出力列: `DATE | TOPIC | TAGS | DISTILLED`
- [ ] テスト: `TestLS_Blocked_Filter`・`TestLS_JSON_IncludesBlockedField`・`TestLS_JSON_IncludesDistilledField`

### cmd/refer.go
- [ ] `session.LoadAll` → `plan.LoadAll`
- [ ] `matchSessions` → `matchPlans(plans []plan.Plan, name string) []plan.Plan`
- [ ] `--summary` 時: `plan.ExtractSections(body, cfg.Plans.SummarySections)` を使用
- [ ] テスト: セッション関連のヘルパーをプランに変更

### cmd/search.go
- [ ] `session.LoadAll` → plan index から読み込み（`index.ReadAll` でも可）
- [ ] keyword フィルタロジックは変更なし（topic/tags/excerpt）
- [ ] テスト更新

### 共通
- [ ] `import "github.com/senna-lang/logosyncx/pkg/session"` の参照を削除
- [ ] `go test ./cmd/ -run "TestLS|TestRefer|TestSearch"` パス確認

## Notes

- `refer.go` の `--summary` フラグは `cfg.Sessions.SummarySections` → `cfg.Plans.SummarySections` に変更
- `search.go` は index を使うか `plan.LoadAll` を使うかどちらでも OK（パフォーマンス差は小さい）
