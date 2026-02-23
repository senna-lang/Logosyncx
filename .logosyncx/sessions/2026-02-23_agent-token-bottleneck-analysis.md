---
id: 22fd13
date: 2026-02-23T11:05:27.953664+09:00
topic: agent-token-bottleneck-analysis
tags:
    - logos
    - agent
    - token-optimization
    - design
    - ux
agent: claude-sonnet-4-6
related:
    - 2026-02-22_logos-task-create-redesign-beads-comparison.md
    - 2026-02-21_logosyncx-ux-feedback.md
---

## Summary

logosの現行設計でエージェントのトークン消費ボトルネックになっている箇所を分析した。最大の問題はセッション・タスク作成時にエージェントが毎回フルMarkdownを構築しなければならない点。flag-baseな入力インターフェースへの改善が最優先。

## Key Decisions

- `logos save` へのフラグベース対応（`--topic`, `--summary`, `--key-decision`, `--tag`）が最優先改善
- `logos task create` のフラグベース対応（`t-86cf53`）に `logos save` 側の設計も含めるべき
- `task-template.md` が存在しないことによる無駄なツール呼び出しを修正すべき
- `logos refer <session> --tasks` で逆方向参照（セッション→関連タスク一覧）が有用
- `knowledge/` 自動更新（`t-bdszo8`）はエージェントの長期的トークン削減に効く

## Context Used

- `.logosyncx/USAGE.md` — エージェント向けワークフロー確認
- `cmd/task.go` — task create の現行インターフェース分析
- `cmd/ls.go` — ls --json の実装確認
- `.logosyncx/task-index.jsonl` — 既存タスク一覧と重複確認
- `2026-02-22_logos-task-create-redesign-beads-comparison.md` — task create 再設計の経緯

## Notes

### ボトルネック一覧（優先度順）

| 優先 | 内容 | 既存タスク |
|------|------|-----------|
| 🔴 高 | `logos save` フラグベース対応（--topic, --summary, --key-decision, --tag） | t-86cf53 に追加 or 新規 |
| 🔴 高 | `logos task create` フラグベース対応 | t-86cf53 |
| 🔴 高 | `task-template.md` が存在しない（USAGE.md の誤記 or ファイル作成） | なし |
| 🟡 中 | `logos refer <session> --tasks` で関連タスク逆引き | なし |
| 🟡 中 | `logos search` の強化（複数キーワード、--related-to） | なし |
| 🟢 低 | `knowledge/` 自動更新 | t-bdszo8 |

### logos save フラグベース案

```bash
logos save \
  --topic "認証リファクタ" \
  --summary "JWTからセッションベースに変更した" \
  --key-decision "cookieはhttpOnly必須" \
  --tag auth --tag refactor
```

現状の「テンプレート読む → 全セクション書く → pipe」という3ステップを1コマンドに圧縮できる。
セッション内容は長文散文なのでフラグで完全代替はできないが、
最低限のメタデータ（topic, tags, summary, key decisions）だけはフラグで渡せると大幅にコスト削減できる。

### task-template.md 問題

USAGE.md の「Workflow for creating a task」ステップ2に「Read `.logosyncx/task-template.md`」とあるが、
ファイルが存在しない。エージェントが毎回空振りのツール呼び出しをする。
修正方法は2つ：
1. `task-template.md` を作成して実際のタスクテンプレートを置く
2. USAGE.md からその行を削除し、flag-based 作成に誘導する（t-86cf53 実装後）
