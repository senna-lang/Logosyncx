---
id: 6c644e
date: 2026-02-28T11:36:46.794954+09:00
topic: 'Session GC design: logos gc command and completed_at field'
tags:
    - design
    - gc
    - sessions
    - tasks
agent: claude-sonnet-4-6
related: []
---

## Summary

セッションが無限にスケールする問題への対応として logos gc コマンドの設計を検討した。トークン消費問題とストレージ問題を分離し、アーカイブベースの非破壊GCアプローチを採用することにした。

## Key Decisions

- logos gc はセッションを sessions/archive/ に移動する（削除ではない）。完全削除は logos session purge --status archived で別途行う
- GC候補の判定基準は3tier: (1)linked tasksが全てdone/cancelled + セッション作成から30日 = 強候補、(2)linked tasksなし + 90日 = 弱候補、(3)linked tasksに1つでもopen/in_progressがある = 保護
- completed_at フィールドを logos task update --status done 時にタスクfrontmatterへ書き込む。これによりGC判定を「セッション作成日」ではなく「タスク完了日」を基準にできる
- しきい値のconfig化（config.jsonへのgcセクション追加）は後回し。まずはデフォルト値ハードコードで実装し、実際に調整ニーズが出てから対応する
- 蒸留（logos distill）はGCの前提条件にしない。アーカイブと知識保存は疎結合にする

## Context Used

なし（新規設計議論）

