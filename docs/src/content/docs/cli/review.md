---
title: "accept / reject"
description: "状態を変えずに特定 Revision へのレビューを記録する。"
sidebar:
  order: 11
---

```sh
sloop accept <specification> [--reason <text>] [著者フラグ]
sloop reject <specification> --reason <text> [著者フラグ]
```

現在の Recorded Revision に ACCEPTED または REJECTED を記録します。Working State に未記録の変更がある場合と、Revision がまだない場合は失敗します。

```sh
sloop accept 1 --author.name coding-agent --author.agent true \
  --reason '正常系と異常系の振る舞いが定義されており実装可能と判断した。'
sloop reject 1 --author.name coding-agent --author.agent true \
  --reason '設定ファイルが既に存在するときの挙動が未定義。上書きか拒否かの指定が必要。'
```

| 操作 | Reason | 制約 |
| --- | --- | --- |
| `accept` | 任意 | ACCEPTED を記録するだけで、READY にしない |
| `reject` | 必須・空不可 | FORCEREADY に対しては拒否される |

成功すると結果・仕様 ID・短縮ハッシュを表示します。レビューは Specification Status を変更せず、新しい Revision も作りません。ACCEPTED は実装完了を意味せず、REJECTED は実装の技術的失敗を意味しません。

新しい Revision ができても古いレビューは残りますが、新しい Revision に対する判断として引き継がれません。現在の Revision のレビューは `sloop context 1 --json` で確認できます。
