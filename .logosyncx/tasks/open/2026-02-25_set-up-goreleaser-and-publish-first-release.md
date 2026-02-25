---
id: t-09ee69
date: 2026-02-25T21:17:55.197405+09:00
title: Set up GoReleaser and publish first release
status: open
priority: high
session: 2026-02-25_distribution-and-installation-design.md
tags:
    - distribution
    - goreleaser
    - homebrew
    - github-actions
assignee: ""
---

## What

GoReleaser をローカルにインストールして make snapshot でビルドを確認し、homebrew-tap リポジトリを作成し、GitHub Actions シークレットを設定して、最初の正式リリースを publish する。

## Checklist

- [ ] brew install goreleaser でローカルに GoReleaser をインストール
- [ ] make snapshot を実行して dist/ にバイナリが生成されることを確認
- [ ] GitHub に senna-lang/homebrew-tap リポジトリを作成（Formula/ ディレクトリのみの最小構成）
- [ ] senna-lang/Logosyncx の Settings > Secrets に HOMEBREW_TAP_GITHUB_TOKEN を追加（homebrew-tap への write 権限を持つ PAT）
- [ ] make release でタグを入力し GitHub Actions release.yml が正常完了することを確認
- [ ] brew install senna-lang/tap/logos で実際にインストールできることを手元で検証
- [ ] curl | bash インストーラーでも動作することを検証

