---
title: "New pkg/knowledge package"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 6
status: done
priority: medium
depends_on: [1]
---

# New pkg/knowledge package

## What

`pkg/knowledge/knowledge.go` を新規作成する。Knowledge 構造体の定義、ファイル名生成、knowledge/ ディレクトリへの書き込み（フロントマター + source material ブロック + 空のセクション見出し）を担う。

## Why

`logos distill`（Task 011）が knowledge ファイルを書き込む際に使用するパッケージ。Config（Task 001）に依存するが、他のパッケージに依存しないため早めに作成できる。

## Scope

変更対象のファイル：
- `pkg/knowledge/knowledge.go` (新規)
- `pkg/knowledge/knowledge_test.go` (新規)

## Checklist

- [ ] `Knowledge` 構造体定義:
  ```go
  type Knowledge struct {
      ID    string     `yaml:"id"`
      Date  *time.Time `yaml:"date,omitempty"`
      Topic string     `yaml:"topic"`
      Plan  string     `yaml:"plan"`   // source plan filename
      Tags  []string   `yaml:"tags"`
      Body  string     `yaml:"-"`
  }
  ```
- [ ] `KnowledgeDir(projectRoot string) string` — `.logosyncx/knowledge/`
- [ ] `FileName(k Knowledge) string` — `YYYYMMDD-<slug>.md`
- [ ] `Write(projectRoot string, k Knowledge, sourceBlock string, templateSections string) (string, error)`:
  - `knowledge/` ディレクトリを作成（存在しなければ）
  - YAML フロントマターを書き込む
  - `<!-- SOURCE MATERIAL ... -->` HTML コメントブロックを挿入
  - `templateSections`（templates/knowledge.md から読んだ内容）の見出しのみを空セクションとして追加
  - 相対パスを返す
- [ ] `generateID() string` — `"k-"` + 6文字 hex
- [ ] テスト:
  - `TestKnowledge_FileName_Format`
  - `TestWrite_CreatesFile`
  - `TestWrite_ContainsSourceBlock`
  - `TestWrite_ContainsSectionHeadings`
- [ ] `go test ./pkg/knowledge/...` パス確認

## Notes

- `sourceBlock` は `logos distill` が plan body + 各 walkthrough を連結して渡す
- `templateSections` は `templates/knowledge.md` の内容をそのまま渡す（セクション見出しのみ出力）
- knowledge ファイルは SPEC §9.5 のフォーマットに従う
