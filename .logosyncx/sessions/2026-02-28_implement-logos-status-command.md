---
id: f5315f
date: 2026-02-28T11:57:01.035346+09:00
topic: Implement logos status command
tags:
    - go
    - cli
    - git
agent: claude-sonnet-4-6
related: []
tasks:
    - 2026-02-21_implement-logos-status-command.md
---

## Summary

logos status コマンドを実装した。gitutil.StatusUnderDir 関数を追加し git status --porcelain で .logosyncx/ 配下のファイルを取得。staged / unstaged / untracked の3グループに分けて表示する。全てコミット済みの場合は ✓ メッセージを出力。エージェントが logos save 後に保存が反映されたか確認するのに使う。

## Key Decisions

- go-git ではなく exec git status --porcelain を使用（既存の Commit/Push と同じ方針。ローカルのgit設定を自動で尊重できる）
- logos status は完全にread-onlyで exit code 0 固定（uncommitted があってもエラーにしない）
- パス表示は .logosyncx/ プレフィックスを除去してシンプルに表示

## Context Used

なし（タスク記述から直接実装）

