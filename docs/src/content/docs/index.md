---
title: "Sloop ドキュメント"
description: "仕様を起点に、コード・テスト・Git・Coding Agent をつなぐ。"
---

Sloop は、実装仕様とコード・テスト・Git Commit・Coding Agent の関係を、手元の端末で管理する CLI です。仕様を Markdown で書き、判断時点の内容を履歴に残して、利用する Agent に渡せます。

## 最初の一歩

- [Getting Started](/getting-started/) — インストールし、最初の仕様を作成する。
- [Tutorial](/tutorial/) — 仕様作成、レビュー、実装、検証、完了までを体験する。
- [CLI リファレンス](/cli/) — 各コマンドの引数・オプション・失敗条件を確認する。
- [設定ファイル](/configuration/) — `.sloop/config.yaml` と環境変数を設定する。

## 仕様から始める開発

```text
仕様を書く → READY にする → Context を渡す → 実装する → 検証する → 人間が完了を確認
                ↓
         仕様不足があればレビューに記録し、編集へ戻る
```

Sloop が保存するのは「何を実装するか」と「どの仕様に基づいて判断したか」です。Agent 自体の実行やテストの実行は、利用者のツールで行います。`implemented` や `verified` はその結果を記録する操作です。

## ローカルで保持する

通常の CLI 操作はオフラインで利用でき、DB サーバーや LLM ベンダーのアカウントは不要です。共有設定はリポジトリの `.sloop/`、仕様本体や履歴は OS ごとのローカルデータディレクトリに保存します。

このドキュメントはリポジトリ内の現行 CLI 実装を対象にしています。端末間の `push`・`pull`・`sync`、履歴の分岐・マージ、MCP Server、ベンダー固有 Connector、Embedding 検索は現在提供していません。Git Notes はコミットと仕様 Revision の関連を保持しますが、仕様そのものの同期は行いません。
