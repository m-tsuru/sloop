---
title: "Markdown とテンプレート"
description: "編集できる Front Matter、Section ID、テンプレートの作り方。"
---

## 編集形式

`new` と `edit` は次のような一時 Markdown をエディターへ渡します。

```md
---
id: "myapp-1"
title: "入力の検証"
status: DRAFT
parents: []
---

## 目的 {#goal}
入力間違いを利用者に伝える。

## 仕様 {#specification}
空文字列を受け取った場合の挙動をここに定義する。

## 完了条件 {#acceptance-criteria}
正常系と空文字列のテスト条件をここに定義する。
```

| フィールド | 書き方・制約 |
| --- | --- |
| `id` | 生成された表示 ID。変更不可 |
| `title` | 仕様のタイトルを文字列で指定 |
| `status` | [仕様の状態](/cli/status/)を指定。Agent は直接変更不可 |
| `parents` | 親仕様の完全な表示 ID を文字列配列で指定。省略時は空配列 |

```yaml
parents:
  - myapp-2
  - myapp-3
```

親仕様は関連付けの情報です。子を `children` として保存するフィールドはありません。Context に親の本文が自動展開されるわけでもありません。

`project`、`revision`、`revision-hash`、`log`、`git-relations`、`review-results`、`agent-runs` は編集不可です。`view` が出力する参照用 Front Matter を編集形式へそのまま貼り付けると拒否されます。Reference は `ref` コマンドで管理してください。

## Section ID

見出し末尾の `{#...}` が抽出用の ID です。

| ID | 用途 |
| --- | --- |
| `goal` | 目的 |
| `specification` | 実装仕様 |
| `acceptance-criteria` | 完了・検証条件 |

追加の ID も利用できます。英数字・`_`・`-` を使い、同じ文書では重複させないでください。

```md
## ユーザーインターフェース {#user-interface}
表示内容をここに記述する。
```

```sh
sloop view 1 --section user-interface
sloop query --section user-interface --filter status:completed
```

現行の抽出処理は、ID の付いた見出しの次の行から、次の ID 付き見出しの直前までを返します。ID のない小見出しはその内容に含まれます。重複 ID は後の内容で上書きされるため、ID を一意にしてください。

## テンプレート

既定のテンプレートは `.sloop/templates/default.md` です。**YAML Front Matter を含めず**、本文のみ保存します。

`.sloop/templates/bugfix.md` の例です。

```md
## 目的 {#goal}

## 再現手順 {#reproduction}

## 仕様 {#specification}

## 完了条件 {#acceptance-criteria}
```

```sh
sloop new --template bugfix
```

指定するのは `.md` を除く名前です。パスの指定はできません。テンプレートを変更しても、既存仕様は変更されません。
