---
id: "430746"
date: 2026-02-28T11:47:05.803892+09:00
topic: Implement logos gc and completed_at
tags:
    - go
    - gc
    - sessions
    - tasks
agent: claude-sonnet-4-6
related: []
tasks:
    - 2026-02-28_implement-logos-gc-command-and-completed_at-for-tasks.md
---

## Summary

logos gc コマンドと completed_at フィールドを実装した。タスクのステータスが done/cancelled に遷移する際に completed_at タイムスタンプをfrontmatterに自動書き込みするよう store.UpdateFields を改修。session パッケージに ArchiveDir/Archive/LoadArchived 関数を追加。logos gc コマンド（--dry-run, --linked-days, --orphan-days）と logos gc purge サブコマンドを新規実装。USAGE.md と usageMD 定数を更新。

## Key Decisions

- LoadAll は entry.IsDir() をスキップするため sessions/archive/ は変更なしで自動除外される
- completed_at は done/cancelled への初回遷移時のみ書き込む（既にdoneのタスクを再度doneにしても上書きしない）
- logos gc は sessions/archive/ への移動（非破壊）。完全削除は logos gc purge で分離
- GC候補判定: linked tasks全done/cancelled + completed_at（なければ session date）からN日 = 強候補、tasks無し + session dateからN日 = 弱候補、active taskあり = 保護
- しきい値のconfig.json化は今回見送り。デフォルト30日/90日はフラグで上書き可能

## Context Used

2026-02-28_session-gc-design-logos-gc-command-and-completed_at-field.md

