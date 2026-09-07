---
title: "list"
description: "仕様を一覧・絞り込み・JSON・CSV で取得する。"
sidebar:
  order: 5
---

```sh
sloop list [--filter <key:value>] [--json | --csv]
```

Working State の一覧を最終更新日時の降順で表示します。同時刻なら ID 順です。通常出力はプロジェクト情報と ID・Title・Status・Last Updated・Last Author の表です。

| フラグ | 既定値 | 用途 |
| --- | --- | --- |
| `--filter` | なし | `status:<状態>` または `title:<部分文字列>`。複数回指定可能 |
| `--json` | `false` | 配列の JSON を出力 |
| `--csv` | `false` | ヘッダー付き CSV を出力 |

## 絞り込み

```sh
sloop list --filter status:draft --filter status:ready
sloop list --filter status:ready --filter 'title:Markdown'
```

同じキーは OR、異なるキーは AND です。状態は大文字・小文字を区別しません。タイトルは大文字・小文字を区別する部分一致です。未知のキー、空の値、不正な状態はエラーになります。

## JSON と CSV

```sh
sloop list --json
sloop list --csv > /tmp/specifications.csv
```

JSON は次の形です（値は例）。0 件なら `[]` です。

```json
[
  {
    "id": "myapp-1",
    "title": "入力の検証",
    "status": "DRAFT",
    "last_updated": "2026-09-08T09:00:00+09:00",
    "last_author": { "name": "your-name", "email": "you@example.com", "agent": false }
  }
]
```

CSV のカラムは `id,title,status,last_updated,last_author_name,last_author_email,last_author_agent` です。0 件でも CSV ヘッダーを出力します。いずれも人間向けプロジェクト見出しは付きません。`--json` と `--csv` の同時指定はエラーです。
