---
name: plan-slice
description: "プランをPR粒度のタスクに分割してファイルとして保存する"
---

# プラン分割・タスク化

コーディングエージェントがプランモードで作成した大きな実装プランを、PRが肥大化しないよう PR粒度のタスクに分割し、プランファイルとタスクファイルを保存する。

ツール依存なし。Write ツールでマークダウンファイルを直接作成する。

---

## トリガー

- ユーザーが「プランを分割して」「タスクに分けて」「split this plan」等を言ったとき
- プランモードで作成したプランの実装がPRとして大きくなりそうなとき

---

## 保存ディレクトリ

```
.claude/skills/plan-slice/
├── SKILL.md
├── plans/
│   └── YYYYMMDD-<slug>.md        ← プラン全体
└── tasks/
    └── YYYYMMDD-<slug>/
        ├── 001-<task-title>.md   ← タスク1
        ├── 002-<task-title>.md   ← タスク2
        └── ...
```

- `YYYYMMDD`: 実行日の日付
- `<slug>`: プランタイトルをケバブケースに変換（例: `user-auth-refactor`）
- ディレクトリが存在しない場合は Bash で `mkdir -p` して作成する

---

## 実行手順

### 1. プランの取得

以下の優先順でプランを取得する：

**A. 会話コンテキストから取得（最優先）**
- 現在の会話にプランが含まれている場合はそのまま使用する

**B. ファイルから取得**
- ユーザーがファイルパスを指定した場合は Read ツールで読み込む

**C. ユーザーに確認**
- 不明な場合はプランの貼り付けをユーザーに依頼する

---

### 2. プランの分析・分割

**PR粒度の判断基準:**
- 1タスク = 1PRとして独立してレビュー可能な変更単位
- 目安: ファイル変更数が10以内、差分が400行以内
- 単一責任の原則: 1タスクには1つの明確な目的
- テストと実装は同一タスクに含める（TDDの原則）

**分割パターン:**
- レイヤー順に分割（型定義 → リポジトリ → ユースケース → API → UI）
- 機能単位で分割（認証 → CRUD → 検索 等）
- リファクタリングは実装とは別タスクに分離

**依存関係の整理:**
- タスク間の依存を明確にする（001完了後に002着手、等）
- 並行実施可能なタスクを識別する

---

### 3. プランファイルの保存

Write ツールで `.claude/skills/plan-slice/plans/YYYYMMDD-<slug>.md` を作成する。

**フォーマット:**

```markdown
---
title: "<プランタイトル>"
date: YYYY-MM-DD
status: open
tasks_dir: .claude/skills/plan-slice/tasks/YYYYMMDD-<slug>/
---

# Plan: <タイトル>

## 概要

<プランの目的と背景を2〜3文で>

## 背景・理由

<なぜこの変更が必要か>

## アーキテクチャ / 設計

<変更の全体像、関連ファイル・コンポーネント、設計上の決定事項>

## タスク一覧

| # | ファイル | タイトル | 優先度 | 依存 |
|---|---------|---------|--------|------|
| 001 | 001-<title>.md | ... | high | - |
| 002 | 002-<title>.md | ... | medium | 001 |
| 003 | 003-<title>.md | ... | medium | 001 |

## リスク・考慮事項

<潜在的なリスク、注意点、非機能要件への影響>
```

---

### 4. タスクファイルの作成

各タスクを Write ツールで `.claude/skills/plan-slice/tasks/YYYYMMDD-<slug>/NNN-<task-title>.md` として作成する。

**フォーマット:**

```markdown
---
title: "<タスクタイトル>"
plan: .claude/skills/plan-slice/plans/YYYYMMDD-<slug>.md
seq: NNN
status: open
priority: high|medium|low
depends_on: []  # 依存するタスクのseq番号リスト。なければ空
---

# <タスクタイトル>

## What

<何を実装するか。具体的に>

## Why

<なぜこのタスクが必要か。プラン全体のどの部分を担当するか>

## Scope

変更対象のファイル・ディレクトリ：
- `path/to/file.ts`
- `path/to/test.ts`

## Checklist

- [ ] <実装ステップ1>
- [ ] <実装ステップ2>
- [ ] テスト追加（Red → Green → Refactor）
- [ ] `npm test` パス確認

## Notes

<補足・注意点があれば>
```

**命名規則:**
- `NNN` は3桁ゼロ埋め（001, 002, ...）
- `<task-title>` はタスクタイトルをケバブケース（例: `add-user-repository`）

---

### 5. 完了報告

全ファイル作成後、以下の形式でユーザーに報告する：

```
## プラン分割完了

**プランファイル**: `.claude/skills/plan-slice/plans/YYYYMMDD-<slug>.md`
**タスクディレクトリ**: `.claude/skills/plan-slice/tasks/YYYYMMDD-<slug>/`

**作成タスク（N件）:**

| # | タイトル | 優先度 | 依存 |
|---|---------|--------|------|
| 001 | タスクA | high | - |
| 002 | タスクB | medium | 001 |
| 003 | タスクC | medium | 001 |

**推奨着手順序**: 001 → 002 & 003（並行可）
```

---

## 注意事項

- ディレクトリ作成は Bash で `mkdir -p .claude/skills/plan-slice/plans .claude/skills/plan-slice/tasks/YYYYMMDD-<slug>` を実行する
- ファイル作成は必ず Write ツールを使う（Bash の echo リダイレクトは使わない）
- タスクファイルの `plan:` フィールドには必ずプランファイルのパスを記載してリンクさせる
- 日付は実行時の実際の日付を使用する