---
title: "Task store: flat layout, NextSeq, CreateWalkthroughScaffold"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 5
status: done
priority: high
depends_on: [4]
---

# Task store: flat layout, NextSeq, CreateWalkthroughScaffold

## What

`internal/task/store.go` を完全書き換えする。status サブディレクトリ構造を廃止し、`tasks/<plan-slug>/NNN-<title>/TASK.md` のフラットレイアウトに移行。`NextSeq`・`CreateWalkthroughScaffold`・`IsBlocked` を新設。`UpdateFields` はフロントマターのみ書き換え（ファイル移動なし）。

## Why

Task 004 のデータモデル変更を実際の I/O に反映させる最大の変更。これが通れば cmd 層のテストが書けるようになる。

## Scope

変更対象のファイル：
- `internal/task/store.go` (rewrite)
- `internal/task/store_test.go` (新規または全面更新)

## Checklist

### Store 構造体
- [ ] `Store` を更新:
  ```go
  type Store struct {
      projectRoot string
      dir         string  // .logosyncx/tasks/
      plansDir    string  // .logosyncx/plans/
      cfg         *config.Config
  }
  ```
- [ ] `NewStore(projectRoot string, cfg *config.Config) *Store`
- [ ] `plansDir` を sessionDir から変更

### loadAll
- [ ] glob: `tasks/<plan-slug>/NNN-<title>/TASK.md` パターンで走査
  - `filepath.Glob` または `filepath.Walk` で `**/TASK.md` を収集
- [ ] `DirPath` を task ディレクトリの絶対パスにセット
- [ ] 各 plan group 内で `IsBlocked` を計算してセット

### Create
- [ ] `Create(t *Task) (string, error)`:
  - `planGroupDir` を `t.Plan` のステムから導出
  - `NextSeq(planGroupDir)` で seq 番号を自動採番
  - `t.Seq` をセット
  - `tasks/<plan-slug>/NNN-<title>/` ディレクトリを作成
  - `TASK.md` にフロントマター scaffold のみ書き込み
  - `gitutil.Add` でステージング
  - 相対パスと seq を返す

### NextSeq
- [ ] `NextSeq(planGroupDir string) (int, error)`:
  - `planGroupDir` 内の `NNN-*` ディレクトリを読み取り
  - 最大の NNN + 1 を返す（存在しなければ 1）

### Get
- [ ] `Get(planPartial, nameOrPartial string) (*Task, error)`:
  - `planPartial` が空 → 全 plan group を検索
  - `planPartial` あり → 部分一致する plan slug のみ検索
  - `nameOrPartial` でタスクディレクトリ名に部分一致
  - 0件: `ErrNotFound`、2件以上: `ErrAmbiguous`

### UpdateFields
- [ ] `UpdateFields(planPartial, nameOrPartial string, fields map[string]interface{}) error`:
  - `Get` でタスクを特定
  - TASK.md を読み、フロントマターのみ更新して書き直す（ファイル移動なし）
  - `status → done` の場合: `CreateWalkthroughScaffold(t)` を呼ぶ
  - `status → in_progress` の場合: `IsBlocked(t, allTasks)` が true なら hard error
  - `completed_at` を `done` 遷移時にセット

### IsBlocked
- [ ] `IsBlocked(t *Task, planTasks []*Task) bool`:
  - `t.DependsOn` の各 seq について同一 plan 内のタスクを検索
  - 1つでも `status != done` なら `true`

### CreateWalkthroughScaffold
- [ ] WALKTHROUGH.md のパス: `<task.DirPath>/WALKTHROUGH.md`
- [ ] 既に存在する場合は何もしない（冪等）
- [ ] scaffold 内容（SPEC §8.3 の固定フォーマット）:
  ```markdown
  # Walkthrough: <task title>

  <!-- Auto-generated when this task was marked done. -->
  <!-- Fill in each section before running logos distill. -->

  ## What Was Done

  <!-- Describe what was actually implemented or resolved. -->

  ## How It Was Done

  <!-- Key steps, approach taken, alternatives considered. -->

  ## Gotchas & Lessons Learned

  <!-- Anything that tripped you up, surprising behaviour, edge cases. -->

  ## Reusable Patterns

  <!-- Code snippets, patterns, or conventions worth reusing. -->
  ```

### Delete
- [ ] `Delete(planPartial, nameOrPartial string) (*Task, error)`:
  - `Get` でタスクを特定
  - タスクディレクトリごと削除（`os.RemoveAll(t.DirPath)`）
  - インデックス再構築

### RebuildTaskIndex
- [ ] 新しい `loadAll` を使ってインデックスを再構築

### テスト
- [ ] `TestStore_Create_WritesTaskMDInPlanGroupDir`
- [ ] `TestStore_Create_AutoAssignsSeq`
- [ ] `TestStore_Create_PrintsRelativePath`
- [ ] `TestStore_UpdateFields_NoFileMoves`
- [ ] `TestStore_UpdateFields_InProgress_BlockedByDep_HardError`
- [ ] `TestStore_UpdateFields_Done_CreatesWalkthrough`
- [ ] `TestStore_UpdateFields_Done_WalkthroughNotOverwritten`
- [ ] `TestStore_Delete_RemovesTaskDir`
- [ ] `TestStore_Get_AmbiguousAcrossPlans`
- [ ] `TestStore_NextSeq_EmptyDir_Returns1`
- [ ] `TestStore_NextSeq_ExistingTasks`
- [ ] `go test ./internal/task/...` パス確認

## Notes

- `Purge` は削除する（cancelled status なし）
- `AppendSession`・`ResolveSession` は削除する
- インデックスへの append は best-effort（失敗してもエラーにしない）
