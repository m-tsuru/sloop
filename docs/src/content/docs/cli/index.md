---
title: "CLI の使い方とコマンド一覧"
description: "指定方法、共通フラグ、出力形式と全サブコマンドの索引。"
sidebar:
  order: 0
---

## 基本形

```sh
sloop <command> [arguments] [flags]
sloop --help
sloop help <command>
sloop <command> --help
```

`init` とヘルプ以外は Sloop Project 内で実行します。成功時は終了コード `0`、エラー時は `1` となり、診断は標準エラー出力へ出ます。ヘルプは `-h` / `--help` でも表示できます。

## Specification と Revision の指定

| 指定 | 例 | 利用できるコマンド |
| --- | --- | --- |
| 番号 | `1` | 仕様を受け取るコマンド全般 |
| 完全な表示 ID | `myapp-1` | 仕様を受け取るコマンド全般 |
| カンマ区切りの仕様 | `1,2,myapp-3` | `status` と状態 alias、`commit-record` |
| Revision 番号 | `'myapp-1#2'` または `'1#2'` | `edit`、`view` |
| Full Hash / 一意な prefix | `a1b2c3d4` | `edit`、`view` |

番号は現在の `project.spec-prefix` と組み合わせて解決します。ハッシュは通常の仕様 ID と使える場所が異なります。短縮ハッシュが曖昧なら長くしてください。数字だけの指定では既存の仕様番号が優先されます。

## コマンド一覧

| コマンド | 目的 |
| --- | --- |
| [init](/cli/init/) | プロジェクトを初期化 |
| [new](/cli/new/) | テンプレートから仕様を作成 |
| [edit](/cli/edit/) | 編集・記録・過去 Revision の復元 |
| [view](/cli/view/) | Markdown や Section を表示 |
| [list](/cli/list/) | 一覧・フィルター・JSON / CSV 出力 |
| [log](/cli/log/) | Recorded Revision の履歴を表示 |
| [status](/cli/status/) | 仕様の状態を変更 |
| [draft / ready / force-ready / implemented / verified / cancel / complete](/cli/status/#状態-alias) | 状態変更の省略形 |
| [ref add / list / remove](/cli/ref/) | コード・テスト・補足ファイルを関連付け |
| [bind add / remove / list](/cli/bind/) | Feature と実装・テスト Symbol を関連付け |
| [query](/cli/query/) | 複数仕様から Section を取得 |
| [context](/cli/context/) | Coding Agent 用 Context を出力 |
| [accept / reject](/cli/review/) | Revision に対するレビューを記録 |
| [agent-run / run-record](/cli/agent-run/) | Agent の実行結果を記録 |
| [commit-record / cr](/cli/commit-record/) | Git Commit と Revision を関連付け |
| [index rebuild](/cli/index-rebuild/) | Object Store から Revision 索引を再構築 |
| [gc](/cli/gc/) | 再生成可能な cache を削除 |
| `help [command]` | コマンドのヘルプを表示 |

## フラグの適用範囲

著者フラグは対応する書き込みコマンドに指定します。[著者情報の解決規則](/configuration/#著者情報)を参照してください。全コマンド共通の `--json` はありません。`--json` は `list`、`bind list`、`context`、`--csv` は `list` のみで利用できます。`--reason` は状態変更・レビュー・Agent Run で利用できます。

コマンドの出力は現在英語です。このドキュメントではその意味を日本語で説明しています。
