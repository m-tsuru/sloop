---
title: "データ保存とバックアップ"
description: "共有設定、SQLite、immutable objects、Git Notes の保存範囲と復旧の限界。"
---

## 保存先

リポジトリ内で共有するのは設定とテンプレートです。

```text
.sloop/
├── config.yaml
└── templates/
    └── default.md
```

仕様の本文と履歴は、次のローカルディレクトリにあります。

| OS | 保存先 |
| --- | --- |
| macOS | `~/Library/Application Support/sloop/<project-id>/` |
| Linux | `${XDG_DATA_HOME:-~/.local/share}/sloop/<project-id>/` |

```text
<project-id>/
├── sloop.db
├── objects/
│   └── sha256/
│       └── ab/
│           └── 残り62桁のハッシュ
└── cache/
```

SQLite は Working State・索引・レビュー・Agent Run などを保持します。Recorded Revision は別の immutable object として保存し、ハッシュで識別します。Git Commit との関連は Git Notes にも記録されます。

## バックアップするもの

1. Sloop の編集や書き込みを終了し、動作中のエディターも閉じます。
2. 対象プロジェクトの**ローカルデータディレクトリ全体**をコピーします。SQLite の付随ファイルがあればそれも含めます。
3. リポジトリの `.sloop/`、ソースコード、Git の履歴と利用している Notes ref も保存します。
4. 復元後に `sloop list`、`sloop log 1`、`sloop view 1` で内容を確認します。

書き込み中の SQLite ファイルだけを単純にコピーする運用は避け、プロセスを止めて整合した状態を保存してください。`cache/` は再生成可能ですが、それ以外を「キャッシュ」とみなして削除しないでください。

## Object Store から復元できる範囲

```sh
sloop index rebuild
```

Revision とその固定内容を再索引化できます。一方、SQLite にしかない未記録の編集・Review・Agent Run・状態変更理由などは、Object Store だけでは復元できません。`index rebuild` は完全バックアップの代わりではありません。

## 別の端末に移す場合

`.sloop/` の Git clone だけでは仕様本体は移りません。現在は Sloop の端末間同期コマンドを提供していないため、必要なら書き込みを停止したうえで設定とローカルデータを一緒に移す運用になります。`project.id` を保ち、移行先 OS の対応する保存先へ配置します。二つの端末で進んだ履歴を統合する機能はありません。

Git Notes を共有しても、Notes に仕様の object は含まれません。Notes と Specification 自体の移行・同期は別の問題です。
