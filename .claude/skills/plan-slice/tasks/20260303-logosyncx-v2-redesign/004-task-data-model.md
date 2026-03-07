---
title: "Task data model: seq, plan, depends_on ints, remove cancelled"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 4
status: done
priority: high
depends_on: [1]
---

# Task data model: seq, plan, depends_on ints, remove cancelled

## What

`internal/task/task.go`・`filter.go`・`index.go` の構造体を v2 スキーマに更新する。`Task` に `Seq int`・`Plan string`・`DependsOn []int` を追加、`Session`・`Sessions`・`Related`・`StatusCancelled` を削除。`TaskJSON` も同様に更新。`Filter` に `Plan string`・`Blocked bool` を追加。

## Why

Task store の書き換え（Task 005）の前に、データ構造を確定させる必要がある。

## Scope

変更対象のファイル：
- `internal/task/task.go` (rewrite struct)
- `internal/task/filter.go` (update Filter)
- `internal/task/index.go` (update TaskJSON fields)
- テストファイルの対応するアサーション

## Checklist

### task.go
- [ ] `Task` 構造体を更新:
  ```go
  type Task struct {
      ID          string     `yaml:"id"`
      Date        time.Time  `yaml:"date"`
      Title       string     `yaml:"title"`
      Seq         int        `yaml:"seq"`
      Status      Status     `yaml:"status"`
      Priority    Priority   `yaml:"priority"`
      Plan        string     `yaml:"plan"`
      DependsOn   []int      `yaml:"depends_on,omitempty"`
      Tags        []string   `yaml:"tags"`
      Assignee    string     `yaml:"assignee"`
      CompletedAt *time.Time `yaml:"completed_at,omitempty"`
      DirPath     string     `yaml:"-"`
      Excerpt     string     `yaml:"-"`
      Body        string     `yaml:"-"`
  }
  ```
- [ ] `Session`, `Sessions`, `Related` フィールドを削除
- [ ] `StatusCancelled` を削除、`ValidStatuses` から除外
- [ ] `TaskDirName(seq int, title string) string` を追加: `fmt.Sprintf("%03d-%s", seq, slugify(title))`
- [ ] `Marshal(t Task) ([]byte, error)` を更新
- [ ] `Parse(filename string, data []byte) (Task, error)` を更新

### TaskJSON (task.go または index.go)
- [ ] `TaskJSON` を更新:
  ```go
  type TaskJSON struct {
      ID          string     `json:"id"`
      DirPath     string     `json:"dir_path"`
      Date        time.Time  `json:"date"`
      Title       string     `json:"title"`
      Seq         int        `json:"seq"`
      Status      Status     `json:"status"`
      Priority    Priority   `json:"priority"`
      Plan        string     `json:"plan"`
      DependsOn   []int      `json:"depends_on"`
      Tags        []string   `json:"tags"`
      Assignee    string     `json:"assignee"`
      CompletedAt *time.Time `json:"completed_at,omitempty"`
      Blocked     bool       `json:"blocked"`
      Excerpt     string     `json:"excerpt"`
  }
  ```
- [ ] `Session`, `Sessions`, `Related` フィールドを削除

### index.go (`task-index.jsonl` 管理)
- [ ] `AppendTaskIndex(projectRoot string, t TaskJSON) error` — TaskJSON を `task-index.jsonl` に JSONL 形式で追記
- [ ] `ReadAllTaskIndex(projectRoot string) ([]TaskJSON, error)` — `task-index.jsonl` を読み込み（存在しなければ `os.ErrNotExist` を返す）
- [ ] `RebuildTaskIndex` のシグネチャ確認: `(s *Store) RebuildTaskIndex() (int, error)` — Task 005 で実装するが、ここでインデックス書き込み関数の型を確定させる
- [ ] `FromTask(t *Task) TaskJSON` — `Task` → `TaskJSON` 変換（`nil` スライスを空配列に正規化）:
  ```go
  func FromTask(t *Task) TaskJSON {
      return TaskJSON{
          ID:          t.ID,
          DirPath:     t.DirPath,
          Date:        t.Date,
          Title:       t.Title,
          Seq:         t.Seq,
          Status:      t.Status,
          Priority:    t.Priority,
          Plan:        t.Plan,
          DependsOn:   normalizeInts(t.DependsOn),   // nil → []int{}
          Tags:        normalizeStrings(t.Tags),       // nil → []string{}
          Assignee:    t.Assignee,
          CompletedAt: t.CompletedAt,
          Blocked:     false, // store.loadAll 時にセット
          Excerpt:     t.Excerpt,
      }
  }
  ```
- [ ] `SortJSONByDateDesc(entries []TaskJSON)` — 日付降順ソート（既存関数の更新）
- [ ] テスト: `TestFromTask_NilSlicesNormalized`・`TestReadAllTaskIndex_Empty`・`TestAppendTaskIndex_WritesJSONL`

### filter.go
- [ ] `Filter` 構造体を更新:
  ```go
  type Filter struct {
      Plan     string
      Status   Status
      Priority Priority
      Tags     []string
      Keyword  string
      Blocked  bool
  }
  ```
- [ ] `Session` フィールドを削除
- [ ] `matchesFilter` / `matchesJSONFilter` を新フィールドに対応

### テスト
- [ ] `TestTask_NoStatusCancelled`
- [ ] `TestTask_TaskDirName_Format` (例: `001-add-jwt-middleware`)
- [ ] `TestFilter_PlanPartial`
- [ ] `TestFilter_Blocked`
- [ ] `go test ./internal/task/...` パス確認

## Notes

- `IsValidStatus` から `cancelled` を除外するのを忘れずに
- `DirPath` は derived フィールド（`yaml:"-"`）— store が `loadAll` 時にセットする
