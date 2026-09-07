---
title: "edit"
description: "Working State の編集、明示記録、過去 Revision の復元。"
sidebar:
  order: 3
---

```sh
sloop edit <specification-or-revision> [--record] [著者フラグ]
```

番号・完全な ID なら現在の Working State を開きます。

```sh
sloop edit 1
sloop edit myapp-1 --record
```

| フラグ | 既定値 | 用途 |
| --- | --- | --- |
| `--record` | `false` | 編集前を記録し、変更があれば編集後も記録 |
| 著者フラグ | [著者設定](/configuration/#著者情報) | 編集者を指定 |

## 編集と記録

同じ著者が DRAFT を編集する通常操作では、細かい編集のたびに Revision を作りません。著者が変わる場合や状態が変わる場合は履歴を記録します。本文・タイトル・親仕様を変更すると、元の状態が DRAFT 以外なら DRAFT に戻ります。同時に Front Matter を READY にしても、元の状態が DRAFT 以外なら意味内容の変更による DRAFT 化が優先されます。

最終内容に変更がなければ Working State は変更しません。Agent は Front Matter の状態を変更できません。人間は状態のみを変更できます。

## 過去を復元する

```sh
sloop view '1#1'
sloop edit '1#1'
```

現在の Head と異なる Revision を指定すると確認が出ます。

```text
Revision 2 is currently the latest revision.
Restore the content of revision 1 (a1b2c3d4) as a new revision and open it for editing? [y/N]
```

`y` または `yes` で復元します。それ以外は変更せず終了します。未記録の現在内容をまず保存し、現在 Head の子として復元 Revision を作ってからエディターを開きます。復元対象はタイトル・本文・状態・親仕様・Reference です。過去の著者を装うのではなく、今回の操作の著者で記録します。

:::note[編集前の記録は残ります]
エディターが異常終了しても、`--record`、著者切り替え、復元によってエディター起動前に作られた Revision は残ります。復元の確認後は、エディターを終了するだけで復元操作が取り消されるわけではありません。
:::

現在の Head 自体を指定した場合は復元せず Working State を編集します。履歴の枝を作る `--branch-from` は未提供です。

存在しない仕様・Revision、曖昧なハッシュ、不正な Markdown、ID 変更、Agent による状態の直接編集は拒否されます。
