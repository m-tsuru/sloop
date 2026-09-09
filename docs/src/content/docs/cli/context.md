---
title: "context"
description: "固定 Revision または Working State から Agent 用情報を出力する。"
sidebar:
  order: 10
---

```sh
sloop context <specification> [--json] [--working]
```

| フラグ | 既定値 | 意味 |
| --- | --- | --- |
| `--json` | `false` | 構造化 JSON を出力 |
| `--working` | `false` | 固定 Revision ではなく現在の Working State を使用 |

```sh
sloop ready 1
sloop context 1
sloop context 1 --json > /tmp/context.json
```

既定では現在の immutable Recorded Revision を使用します。未記録の変更がある、または Head がない場合は失敗します。未記録内容の確認だけなら `sloop context 1 --working --json` を使います。

## 出力内容

テキスト出力には仕様 ID・UUID・タイトル・Full Hash・状態・記録済みかどうか・親 ID・目的・仕様・完了条件・Reference・Feature Binding・Git Relation・レビュー方針を含みます。

JSON の主なフィールドは次のとおりです。

| キー | 型・内容 |
| --- | --- |
| `project_id`, `specification_id`, `specification_uuid` | Project ID、表示用 Specification ID、永続 Specification UUID |
| `revision_hash` | 64 桁の SHA-256。Working Context では省略 |
| `recorded` | 固定 Revision なら `true`、Working Context は `false` |
| `status`, `title` | 状態とタイトル |
| `goal`, `specification`, `acceptance_criteria` | 予約 Section の本文 |
| `sections` | カスタム ID を含む Section ID → 本文の map |
| `parents` | 親仕様の ID 配列 |
| `references` | ID・kind・path・行範囲・コミット等を持つ配列 |
| `features` | Feature ID、Section ID、Implementation/Test Locator と現在の Resolution Status |
| `git_relations` | 対象 Revision とコミットの関連配列 |
| `reviews` | 対象 Revision の Review 配列。なければ省略 |
| `review_policy` | 状態に応じたレビュー方針の文字列 |

Review は JSON に含まれますが、現行のテキスト出力には個別の Review 一覧を表示しません。Working Context には Recorded Revision の Review や Git Relation を含めません。

## Context の利用範囲

Reference の**ファイル全文は含まれません**。Agent が必要なファイルを別途読めるようにしてください。無関係な仕様や親の本文を自動的に全件取り込むこともありません。

READY の方針は仕様不足をレビューで拒否すること、FORCEREADY の方針は不足を許容して実装を継続することです。その他の状態は人間による実装可能の確認を意味しません。固定されていても DRAFT の Context を READY 相当として扱わないでください。

READY / FORCEREADY の非空 Feature Binding が現在の Repository で解決できない場合、Context 生成は失敗します。過去 Revision 内の Binding は変更されません。
