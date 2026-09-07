---
title: "init"
description: "プロジェクトの共有設定とローカルストアを初期化する。"
sidebar:
  order: 1
---

```sh
sloop init <project-slug> <repository-path>
```

既存ディレクトリに Sloop Project を作ります。追加フラグはありません。

| 引数 | 意味 |
| --- | --- |
| `project-slug` | プロジェクト名。空、`/`、`\` を含む値は不可 |
| `repository-path` | 初期化する既存ディレクトリ。相対・絶対パスを指定可能 |

```sh
sloop init myapp .
```

```text
Project 'myapp' is created.
Project ID: myapp-<生成された UUID v4>
```

`.sloop/config.yaml`、`.sloop/templates/default.md` と、OS のローカル保存先に `sloop.db`・`objects/`・`cache/` を作ります。ID と設定形式は[設定リファレンス](/configuration/)を参照してください。

既存の config や default template があれば上書きせず失敗します。対象が存在しない、またはディレクトリではない場合も失敗します。Git repository の作成は行わないため、Git 連携が必要なら別途 `git init` してください。
