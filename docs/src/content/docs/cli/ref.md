---
title: "ref add / list / remove"
description: "コード・テスト・補足情報への明示的な Reference を管理する。"
sidebar:
  order: 8
---

Reference は仕様内容の一部です。追加・削除すると Working State は DRAFT になり、未記録の変更として保存されます。既存の Recorded Revision は変更しません。参照だけの変更では自動的な Revision 生成は行いません。

## ref add

```sh
sloop ref add <specification> <path[:start-end]> --kind <code|test|context> [--at <commit>] [著者フラグ]
```

| フラグ | 必須・既定値 | 意味 |
| --- | --- | --- |
| `--kind` | 必須 | `code` は実装、`test` はテスト、`context` は補足情報 |
| `--at` | 省略時は Working Tree | 参照する Git Commit。保存時に完全なコミット ID に解決 |
| 著者フラグ | [著者設定](/configuration/#著者情報) | 変更者を指定 |

```sh
sloop ref add 1 --kind code src/parser.go:20-80
sloop ref add 1 --kind test internal/parser_test.go:10-75 --at HEAD
sloop ref add 1 --kind context go.mod
```

パスは実行ディレクトリではなく**プロジェクトルート基準**です。絶対パス・`../` などルート外の指定は拒否します。全体を指定するか、1 以上の開始行と、それ以上の終了行を `:20-80` のように指定します。1 行なら `:20-20` です。

Working Tree の対象ファイル、または指定コミット内の対象パスが必要です。現行 CLI は行番号の大小関係を検証しますが、実ファイルの行数以内かは検証しません。`--at` なしの行番号はファイルの変更で意味がずれるため、内容を固定するなら `--at` を指定します。

## ref list

```sh
sloop ref list 1
```

追加フラグはありません。現在の Reference ID・Kind・Target を表示します。Reference ID は仕様 ID とは別の UUID です。

## ref remove

```sh
sloop ref remove <specification> <reference-id> [著者フラグ]
```

`ref list` に表示された ID を指定します。シェル例では実際の UUID に置き換えてください。

```sh
sloop ref remove 1 19b95238-e270-49ab-a139-ea9ca467b7de
```

存在しない ID はエラーです。削除されるのは関連情報だけで、コードやテストのファイル自体は削除しません。
