---
title: "Update pkg/index Entry for plans, add Blocked field"
plan: .claude/skills/plan-slice/plans/20260303-logosyncx-v2-redesign.md
seq: 3
status: done
priority: high
depends_on: [2]
---

# Update pkg/index Entry for plans, add Blocked field

## What

`pkg/index/index.go` の `Entry` 構造体を Plan 用に更新する。`Tasks []string` を削除し、`DependsOn`, `TasksDir`, `Distilled`, `Blocked` を追加。`FromSession` → `FromPlan`、`Rebuild` が `plans/` を走査するよう変更する。

## Why

Plan インデックス（`index.jsonl`）が新フィールドを持つことで、`logos ls --json` が `blocked` フィールドを返せるようになり、エージェントがどのプランに着手できるかを判断できる。

## Scope

変更対象のファイル：
- `pkg/index/index.go` (update)
- `pkg/index/index_test.go` (更新)

## Checklist

- [ ] `Entry` 構造体を更新:
  ```go
  type Entry struct {
      ID        string    `json:"id"`
      Filename  string    `json:"filename"`
      Date      time.Time `json:"date"`
      Topic     string    `json:"topic"`
      Tags      []string  `json:"tags"`
      Agent     string    `json:"agent"`
      Related   []string  `json:"related"`
      DependsOn []string  `json:"depends_on"`
      TasksDir  string    `json:"tasks_dir"`
      Distilled bool      `json:"distilled"`
      Blocked   bool      `json:"blocked"`
      Excerpt   string    `json:"excerpt"`
  }
  ```
- [ ] `Tasks []string` フィールドを削除
- [ ] `FromSession` を `FromPlan(p plan.Plan, allPlans []plan.Plan) Entry` に変更
  - `Blocked` の計算: `DependsOn` 内のすべての plan filename が `Distilled: true` であれば `false`、1つでも `false` なら `Blocked: true`
- [ ] `Rebuild(projectRoot, excerptSection string) (int, error)` を `plans/` 走査に変更
  - `session.LoadAll` → `plan.LoadAll`
  - `FromSession` → `FromPlan`
- [ ] テスト更新:
  - `TestFromPlan_Blocked_WhenDepsNotDistilled`
  - `TestFromPlan_NotBlocked_WhenNoDeps`
  - `TestFromPlan_NotBlocked_WhenDepsDistilled`
  - `TestRebuild_ScansPlansDir`
- [ ] `go test ./pkg/index/...` パス確認

## Notes

- `Blocked` の計算は `Rebuild` 時に全 plan をロードして `allPlans` を渡す形にする
- `cmd/ls.go` はこのタスクでは変更しない（コンパイルエラーが出る場合は一時的に `FromSession` を alias として残してもよい）
