---
title: "view"
description: "現在の仕様・過去の Revision・特定 Section を Markdown で表示する。"
sidebar:
  order: 4
---

```sh
sloop view <specification-or-revision> [--section <id>]
```

```sh
sloop view 1
sloop view 'myapp-1#2'
sloop view a1b2c3d4
sloop view 1 --section acceptance-criteria
sloop view 1 > /tmp/specification.md
```

番号と ID は Working State、Revision 指定はその固定内容を表示します。`--section` の既定値は空で、指定した場合は見出しと Front Matter を除いた Section の本文だけを出力します。存在しない Section はエラーです。

全文の出力には、project、id、title、status、parents、および必要に応じて references・log・revision・revision-hash を含む参照用 Front Matter が付きます。未記録の Working State には現在内容の Revision 番号・ハッシュを付けません。過去を指定したときの log は、その Revision までです。

表示用ハッシュは短縮形です。完全なハッシュを機械処理する用途には `context --json` を使います。参照用 Front Matter は読み取り専用項目を含むため、`edit` へそのまま入力できません。

出力は装飾のない plain text で、診断は stderr です。macOS では `sloop view 1 | pbcopy` でコピーできます。
