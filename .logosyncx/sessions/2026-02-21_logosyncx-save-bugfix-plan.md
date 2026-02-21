---
id: ff64fd
date: 2026-02-21T12:17:15.937491+09:00
topic: logosyncx-save-bugfix-plan
tags:
    - logosyncx
    - bugfix
    - template
    - session
agent: claude-sonnet-4-6
related:
    - 2026-02-21_logosyncx-ux-feedback.md
---

## Summary

UX feedback session (logosyncx-ux-feedback) を元に、logos save コマンドの3つのバグと1つのUX改善に対する修正案を策定した。beads issueも4件作成済み。

## Key Decisions

- template.md から id/date フィールドを削除する（YAML parse エラーの根本原因）
- Session.Date を *time.Time に変更して空値・省略を許容する
- logos save のエラーメッセージに {{ 検出時のヒントを追加する
- Makefile を新規作成して build/install/test ターゲットを提供する

## Context Used

- .logosyncx/sessions/2026-02-21_logosyncx-ux-feedback.md
- pkg/session/session.go（Session.Date の型定義、Parse/FileName 関数）
- cmd/save.go（runSave、エラーハンドリング）
- .logosyncx/template.md（問題の再現確認）

## Notes

### Issue 一覧

| beads ID | 内容 | 優先度 |
|----------|------|--------|
| logosyncx-44g | Remove id/date from template.md | 高 |
| logosyncx-lbn | Change Session.Date to *time.Time | 高（44g に依存） |
| logosyncx-r1l | Improve error messages with hints | 中 |
| logosyncx-98p | Add Makefile | 低 |

### 修正箇所の詳細

**template.md**
- `id: {{id}}` と `date: {{date}}` の2行を削除

**pkg/session/session.go**
- `Date time.Time` → `Date *time.Time \`yaml:"date,omitempty"\``
- `FileName()`: nil の場合は `time.Now()` を使う

**cmd/save.go**
- `s.Date.IsZero()` → `s.Date == nil`
- parse エラー時に `{{` を含む場合はヒントを付与

**Makefile（新規）**
- `build`, `install`, `test` ターゲット

## Raw Conversation

logos search → logos refer ux-feedback --summary → ソース分析 → 修正案策定 → bd create で4件のissue作成
