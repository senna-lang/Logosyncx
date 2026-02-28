---
id: t-b3c0b3
date: 2026-02-28T11:36:58.619658+09:00
title: Implement logos gc command and completed_at for tasks
status: done
priority: medium
session: 2026-02-28_session-gc-design-logos-gc-command-and-completed_at-field.md
tags:
    - gc
    - sessions
    - tasks
    - go
assignee: ""
completed_at: 2026-02-28T11:46:52.30439+09:00
---

## What

logos gc コマンドの実装と、logos task update --status done 時に completed_at をタスクfrontmatterへ書き込む改修を行う。

## Why

セッションが無限に蓄積されるとエージェントの logos ls --json のトークン消費が増大し、検索ノイズも増える。定期的なアーカイブによってセッションのライフサイクルを管理できるようにする。

## Scope

1. logos task update --status done 時に completed_at: <timestamp> をfrontmatterに書き込む（小改修）
2. logos gc コマンドを新規実装
   - --dry-run フラグ: 候補一覧を表示するだけ（何もしない）
   - デフォルト動作: 候補セッションを sessions/archive/ に移動
   - --linked-days N: 強候補のしきい値（デフォルト30日）
   - --orphan-days N: 弱候補のしきい値（デフォルト90日）
   - 判定基準: linked tasks全done/cancelled + completed_atから30日 = 強候補 / tasksなし + セッション作成から90日 = 弱候補 / active taskあり = 保護
3. logos session purge --status archived で archive/ を完全削除できるようにする
4. logos sync が sessions/archive/ 内のファイルをindexから除外する対応

## Checklist

- [ ] logos task update: completed_at の書き込み実装
- [ ] pkg/session: Archive ステータス / archive/ パスの対応
- [ ] cmd/gc.go: logos gc コマンド実装（--dry-run, --linked-days, --orphan-days）
- [ ] cmd/session_purge.go: logos session purge --status archived 実装
- [ ] logos sync: archive/ を除外する対応
- [ ] USAGE.md と cmd/init.go の usageMD を更新

