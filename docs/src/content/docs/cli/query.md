---
title: "query"
description: "複数の仕様から Section の本文を取り出す。"
sidebar:
  order: 9
---

```sh
sloop query --section <id> [--filter <key:value> ...]
```

`--section` は必須、`--filter` は任意・複数指定可能です。フィルターは [list](/cli/list/) と同じで、同じキーを OR、異なるキーを AND で評価します。

```sh
sloop query --section acceptance-criteria --filter status:completed
sloop query --section user-interface --filter 'title:画面'
```

現在の Working State を最終更新日時の降順で検索し、一致した仕様ごとに `## [myapp-1] タイトル` と Section 本文を Markdown で出力します。指定 Section のない仕様はスキップします。該当がなければ出力せず成功します。

無効なフィルターと `--section` の未指定はエラーです。`view --section` は存在しない Section に対してエラーにする点が異なります。JSON / CSV 出力や Revision 指定には対応していません。

取得した本文をドキュメント作成ツールに渡せますが、Sloop 自体は利用者向けドキュメントを自動生成しません。
