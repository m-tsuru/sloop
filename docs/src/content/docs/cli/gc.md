---
title: "gc"
description: "再生成可能なローカル cache を削除する。"
sidebar:
  order: 15
---

```sh
sloop gc
```

追加引数・フラグはありません。現行実装では、プロジェクトのローカルデータディレクトリにある `cache/` の内容を削除します。削除されたファイル数を表示します。

```text
Removed 0 cache objects.
```

Recorded Revision の object、Working State、SQLite の仕様データ、Reference の参照先ファイルは削除しません。意味のある履歴を短くするコマンドではありません。

cache の読み取り・削除に失敗した場合はエラーとなり、それまでに削除できた内容は戻りません。将来の再生成に必要な cache のみをこのディレクトリへ置き、手作業の原本を保存しないでください。
