---
id: aca47b
date: 2026-02-21T12:05:22.626412+09:00
topic: logosyncx-ux-feedback
tags:
    - logosyncx
    - feedback
    - improvement
    - ux
agent: claude-sonnet-4-6
related: []
---

## Summary

`logos save` コマンドをエージェントが実際に使用した際に発生した問題を分析し、改善提案をまとめた。テンプレートのYAML不整合とDate型の厳格すぎるパースが主因。

## Key Decisions

- テンプレートから `id`/`date` フィールドを削除するのが最優先の改善
- `Session.Date` を `*time.Time` にして空値を許容すべき
- `make install` ターゲット追加でインストールUXを改善
- エラーメッセージに具体的なヒントを追加すべき

## Context Used

- logosyncx を study-log プロジェクトで実際に使用してみた体験
- `pkg/session/session.go` および `cmd/save.go` のソースコード分析

## Notes

### 問題1: `logos` がPATHに未登録

ビルド済みバイナリが `/Users/senna/Documents/Repos/logosyncx/logos` にあったが PATH に入っておらず、毎回 `export PATH=...` が必要だった。

**改善案**: `make install` ターゲットを追加
```makefile
install:
    go build -o logos .
    install -m 755 logos /usr/local/bin/logos
```

---

### 問題2: テンプレートの `{{id}}` / `{{date}}` がYAML的に不正

`{` はYAMLのflow mapping開始文字のため、`{{id}}` は `{id: null}` のmapとして解釈され、string型にunmarshalできずエラー。

```
yaml: unmarshal errors:
  line 1: cannot unmarshal !!map into string
```

**改善案A（最小コスト・最大効果）**: テンプレートから `id`/`date` を完全削除

```markdown
---
topic: {{topic}}
tags: []
agent:
related: []
---
```

`id` と `date` は `logos save` 時に自動補完されるので不要。USAGE.md にも「`id`/`date` は省略してください（自動補完）」と明記する。

---

### 問題3: `date: ""` でもパース失敗

`Session.Date` が `time.Time` 型のため、空文字列はYAMLレベルで unmarshal できずエラー。自動補完ロジック (`s.Date.IsZero()`) に到達する前に落ちる。

```go
// session.go:29 現状
Date    time.Time `yaml:"date"`  // "" を受け付けない
```

**改善案B**: `*time.Time` にして nil を許容

```go
Date  *time.Time `yaml:"date,omitempty"`
```

省略・空文字でもパースが通り、nil なら `time.Now()` を使う形にする。

---

### 問題4: エラーメッセージが不親切

現在のエラーはYAMLの内部エラーをそのまま出力しており、エージェントが原因を特定しにくい。

**改善案C**: 具体的なヒントを付与

例: 「`{{...}}` はYAMLとして無効です。テンプレートの `id`/`date` を削除して再試行してください」

---

### 優先度まとめ

| 優先度 | 改善内容 | コスト |
|--------|----------|--------|
| 高 | テンプレートから `id`/`date` を削除 | 低 |
| 高 | `Date` を `*time.Time` に変更 | 低 |
| 中 | エラーメッセージ改善 | 低 |
| 低 | `make install` 追加 | 低 |

## Raw Conversation

study-log プロジェクトで `logos save` を使おうとした際に3つのエラーが発生。ソースコード (`pkg/session/session.go`, `cmd/save.go`) を分析して根本原因を特定し、改善提案をまとめた。
