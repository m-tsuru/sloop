---
title: "log"
description: "記録された Revision の番号・ハッシュ・状態・著者・日時を表示する。"
sidebar:
  order: 6
---

```sh
sloop log <specification>
```

追加フラグはありません。現在の仕様の Recorded Revision を番号順に表示します。

```sh
sloop log 1
```

```text
Specification: myapp-1

Rev  Hash      Status       Author                  Date
1    a1b2c3d4  DRAFT        your-name               2026-09-08 09:00
2    e5f6a7b8  READY        your-name               2026-09-08 09:10
```

ハッシュは表示例です。Date はローカル時刻です。未記録の変更は行に現れず、履歴がまだなければヘッダーのみです。作業中の内容は `view`、過去の内容は `view '1#1'` で確認します。

このコマンドは Review、Agent Run、状態変更 Reason の一覧ではありません。専用の一覧コマンドや `log --json` は現在提供していません。対象 Revision の Review は `context --json` に含まれます。
