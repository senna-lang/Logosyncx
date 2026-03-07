---
title: "New pkg/plan package (replaces pkg/session)"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 2
status: done
priority: high
depends_on: [1]
---

# New pkg/plan package (replaces pkg/session)

## What

`pkg/plan/plan.go` を新規作成する。`pkg/session/session.go` のパース・マーシャル・I/O ロジックを移植しつつ、Plan 構造体の新フィールド（`DependsOn`, `TasksDir`, `Distilled`）を追加。ファイル名フォーマットを `YYYYMMDD-<slug>.md` に変更する。

## Why

`sessions/` → `plans/` の移行に伴い、Plan を表す独立したパッケージが必要。既存の `pkg/session` はタスク 013 まで残す（並行してコンパイルが通る状態を保つため）。

## Scope

変更対象のファイル：
- `pkg/plan/plan.go` (新規)
- `pkg/plan/plan_test.go` (新規)

## Checklist

- [ ] `pkg/plan/` ディレクトリ作成
- [ ] `Plan` 構造体定義:
  ```go
  type Plan struct {
      ID        string     `yaml:"id"`
      Date      *time.Time `yaml:"date,omitempty"`
      Topic     string     `yaml:"topic"`
      Tags      []string   `yaml:"tags"`
      Agent     string     `yaml:"agent"`
      Related   []string   `yaml:"related"`
      DependsOn []string   `yaml:"depends_on,omitempty"`
      TasksDir  string     `yaml:"tasks_dir"`
      Distilled bool       `yaml:"distilled"`
      Filename  string     `yaml:"-"`
      Excerpt   string     `yaml:"-"`
      Body      string     `yaml:"-"`
  }
  ```
- [ ] `PlansDir(projectRoot string) string` — `.logosyncx/plans/`
- [ ] `ArchiveDir(projectRoot string) string` — `.logosyncx/plans/archive/`
- [ ] `FileName(p Plan) string` — `YYYYMMDD-<slug>.md`（`20060102` フォーマット）
- [ ] `DefaultTasksDir(filename string) string` — `.logosyncx/tasks/<stem>/`
- [ ] `Parse(filename string, data []byte) (Plan, error)`
- [ ] `LoadAll(projectRoot string) ([]Plan, error)`
- [ ] `LoadFile(path string) (Plan, error)`
- [ ] `Write(projectRoot string, p Plan) (string, error)` — frontmatter scaffold のみ書く
- [ ] `Marshal(p Plan) ([]byte, error)`
- [ ] `ExtractSections(body string, sectionNames []string) string`
- [ ] `extractExcerpt(body []byte, excerptSection string) string` (内部)
- [ ] `Archive(projectRoot, filename string) (string, error)`
- [ ] `generateID() string` — 6文字 hex
- [ ] `slugify(s string) string` — kebab-case 変換
- [ ] テスト:
  - `TestFileName_Format` (YYYYMMDD プレフィックスを確認)
  - `TestParse_RoundTrip`
  - `TestParse_DependsOn`
  - `TestParse_Distilled`
  - `TestLoadAll_ScansPlansDir`
  - `TestDefaultTasksDir`
- [ ] `go test ./pkg/plan/...` パス確認

## Notes

- `splitFrontmatter`, `parseHeading`, `truncateRunes` は `pkg/session/session.go` から完全コピーして移植
- `Write` はフロントマター scaffold のみ書き込み（body は空のまま）。エージェントが別途 Write ツールで body を書く
- `pkg/session` のインポートはこのタスクでは変更しない
