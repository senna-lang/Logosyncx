---
id: 9bb20c
date: 2026-02-25T14:11:19.421504+09:00
topic: distribution-and-installation-design
tags:
    - distribution
    - installation
    - goreleaser
    - homebrew
    - self-update
agent: claude-sonnet-4-5
related: []
---

## Summary

logosyncx の配布・インストール設計を策定した。現状は go build + 手動 mv という開発者向けフローのみだったが、これをゼロ依存・ワンコマンドで導入できるよう設計した。

## Key Decisions

配布チャンネルは Homebrew Tap / curl-bash / GitHub Releases / go install の4本柱とする。リリースパイプラインは GoReleaser + GitHub Actions で構成し、semver タグをプッシュするだけで全プラットフォームバイナリのビルド・アーカイブ・checksums.txt 生成・GitHub Release 作成・homebrew-tap への Formula PR 自動送信が行われる。バージョン文字列は internal/version/version.go に var Version = dev として定義し、go build -ldflags でビルド時に埋め込む。logos update コマンドで自己更新を実装する（GitHub API で最新タグ取得 → アーカイブ DL → SHA256 検証 → アトミックリネームによるバイナリ置換）。バックグラウンド更新通知は 1 日 1 回のみ・コマンド出力の後に表示・--json や LOGOS_NO_UPDATE_CHECK=1 では無効化する設計とする。設計の詳細は DistributionDesign.md に記録した。

