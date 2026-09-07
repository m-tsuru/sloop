---
title: "agent-run / run-record"
description: "Coding Agent の実行結果を現在の Revision に記録する。"
sidebar:
  order: 12
---

```sh
sloop agent-run <specification> <implemented|failed> [--reason <text>] [著者フラグ]
sloop run-record <specification> <implemented|failed> [--reason <text>] [著者フラグ]
```

`run-record` は同じコマンドの alias です。Agent を起動するコマンドではなく、別途実行した結果を記録します。結果は大文字・小文字を区別しません。

```sh
sloop agent-run 1 implemented --author.name coding-agent \
  --reason '仕様に対応するコードとテストを作成した。'
sloop agent-run 1 failed --author.name coding-agent \
  --reason '必要なコンパイラーがなく、ビルドを開始できなかった。'
```

現在の Recorded Revision が必要で、未記録変更があれば失敗します。`--reason` は任意ですが、結果を理解できる具体的な説明を残してください。著者は常に `agent: true` で保存されます。

`IMPLEMENTED` は実装を生成した結果、`FAILED` は技術的失敗です。仕様不足は `reject` に記録し、実行結果として `rejected` を指定しません。FORCEREADY でも技術的な FAILED は記録できます。

仕様状態は変わらず、新しい Revision も生成しません。状態変更には別途 `implemented` / `verified` を使います。入力 Revision へ結果を残したい場合は、状態を変更する**前に** `agent-run` を実行してください。Agent Run の一覧表示専用コマンドは現在ありません。
