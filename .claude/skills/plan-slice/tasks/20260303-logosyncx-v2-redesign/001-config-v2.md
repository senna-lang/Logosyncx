---
title: "Config v2: Sessions→Plans, remove SectionConfig, add Knowledge"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 1
status: done
priority: high
depends_on: []
---

# Config v2: Sessions→Plans, remove SectionConfig, add Knowledge

## What

`pkg/config/config.go` を v2 スキーマに書き換える。`Sessions SessionsConfig` → `Plans PlansConfig` へリネーム、`SectionConfig` 型と `sections` 配列をすべて削除、`KnowledgeConfig` を追加、`OrphanSessionDays` → `OrphanPlanDays`、デフォルト version を `"2"` に変更する。

## Why

すべての他パッケージが `config.Config` に依存するため、最初に修正する必要がある。`sections` 配列は新設計ではテンプレートファイルに置き換わるため不要。

## Scope

変更対象のファイル：
- `pkg/config/config.go` (rewrite)
- `pkg/config/config_test.go` (新規または更新)

## Checklist

- [ ] `SectionConfig` 型を削除
- [ ] `SessionsConfig` → `PlansConfig` にリネーム（フィールドは `ExcerptSection`, `SummarySections` のみ）
- [ ] `TasksConfig` から `Sections []SectionConfig` を削除
- [ ] `KnowledgeConfig` 構造体を追加（`ExcerptSection`, `SummarySections`）
- [ ] `Config.Sessions` → `Config.Plans`（JSON key: `"plans"`）
- [ ] `Config.Knowledge KnowledgeConfig` を追加
- [ ] `GcConfig.OrphanSessionDays` → `OrphanPlanDays`（JSON key: `"orphan_plan_days"`）
- [ ] `Default()` のデフォルト値を更新:
  - `Plans.ExcerptSection = "Background"`
  - `Plans.SummarySections = ["Background", "Spec"]`
  - `Tasks.ExcerptSection = "What"`
  - `Tasks.SummarySections = ["What", "Checklist"]`
  - `Knowledge.ExcerptSection = "Summary"`
  - `Knowledge.SummarySections = ["Summary", "Key Learnings"]`
- [ ] `Migrate()` と `isMigrationNeeded()` を削除（呼び出し元も確認）
- [ ] `applyDefaults()` を v2 フィールドに合わせて更新
- [ ] テスト: `TestDefault_Version2`, `TestDefault_HasPlansConfig`, `TestDefault_NoSections`, `TestLoad_V2Config`, `TestSave_RoundTrip`
- [ ] `go test ./pkg/config/...` パス確認

## Notes

- `Migrate()`・`isMigrationNeeded()` は完全削除。呼び出し元 `cmd/sync.go` も同一 PR 内で削除すること（コンパイルエラーを残さない）
- `pkg/session` はこのタスクでは触らない。削除は最後（Task 013）
