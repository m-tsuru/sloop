---
title: "new"
description: "エディターで新しい仕様を作成する。"
sidebar:
  order: 2
---

```sh
sloop new [--template <name>] [著者フラグ]
```

次の仕様番号と UUID を割り当て、テンプレート本文に Front Matter を付けてエディターを起動します。既定の状態は DRAFT です。

| フラグ | 既定値 | 用途 |
| --- | --- | --- |
| `--template` | `default` | `.sloop/templates/<name>.md` を選ぶ |
| `--author.name` / `--author.email` / `--author.agent` | [著者設定](/configuration/#著者情報) | 操作の著者 |

```sh
sloop new
sloop new --template bugfix
sloop new --author.name coding-agent --author.agent true
```

正常終了時は `Specification myapp-1 is created.` を表示します。DRAFT の新規仕様は未記録の Working State です。人間が Front Matter に DRAFT 以外を設定した場合は、その状態の Revision も記録します。

エディター未設定・起動失敗・異常終了、テンプレート未検出、Front Matter 付きテンプレート、不正な Markdown、ID の変更は失敗します。Agent が初期状態の DRAFT を直接変更することも拒否します。既定テンプレートを変更したい場合は[テンプレート](/guides/markdown/#テンプレート)を参照してください。
