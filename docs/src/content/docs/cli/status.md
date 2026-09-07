---
title: "status と状態 alias"
description: "人間と Agent の状態変更ルール、理由、短縮コマンド。"
sidebar:
  order: 7
---

```sh
sloop status <specification[,specification...]> <status> [--reason <text>] [著者フラグ]
```

`status` は表示コマンドではなく、状態を変更するコマンドです。状態の参照は `list` または `view` を使います。

## 人間の変更

```sh
sloop status 1 ready
sloop status 1,2,3 draft
sloop status myapp-1 completed --reason '手動確認が完了した。'
```

DRAFT、READY、FORCEREADY、IMPLEMENTED、VERIFIED、CANCELED、COMPLETED を指定できます。大文字・小文字は区別せず、`force-ready` も受け付けます。人間は任意の状態を設定でき、`--reason` は任意です。

変更すると Working State を記録し、新しい Revision と状態変更記録を作ります。同じ状態の指定は何もしません。複数指定は順に処理するので、途中の失敗でそれ以前の変更が取り消されるとは限りません。

## Agent の変更

```sh
sloop status 1 implemented \
  --author.name coding-agent --author.agent true \
  --reason '仕様で定義された入力検証と異常系のテストを実装した。'
```

Agent は IMPLEMENTED または VERIFIED のみ設定でき、`--reason` が必須です。空文字や `done`・`completed`・`success` だけの理由は拒否します。何を実装・検証したのかを説明してください。

状態を変更するには、未記録の変更がなく現在の Recorded Revision が必要です。その Revision が `based-on-revision` になり、生成される子 Revision が `resulting-revision` になります。基準 Revision を指定する CLI フラグはありません。Context を受け取ってから状態を記録するまで、他の編集を混在させないでください。

## 状態 alias

すべて同じ `--reason` と著者フラグに対応し、Agent の制約も同じです。

| コマンド | 対応する操作 |
| --- | --- |
| `sloop draft 1` | `sloop status 1 draft` |
| `sloop ready 1` | `sloop status 1 ready` |
| `sloop force-ready 1` | `sloop status 1 forceready` |
| `sloop implemented 1` | `sloop status 1 implemented` |
| `sloop verified 1` | `sloop status 1 verified` |
| `sloop cancel 1` | `sloop status 1 canceled` |
| `sloop complete 1` | `sloop status 1 completed` |

例えば `sloop ready 1,2` と複数指定できます。`complete --author.agent true` は許可されません。

`force-ready` は人間が仕様不足を許容すると決めたときに使います。この状態では `reject` による仕様不足の記録が拒否されます。実行環境の技術的失敗には `agent-run failed` を使います。
