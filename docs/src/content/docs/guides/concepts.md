---
title: "基本概念と履歴"
description: "Working State、Recorded Revision、状態、レビューの違いを理解する。"
---

## 何を管理するか

| 概念 | 意味 |
| --- | --- |
| Project | `.sloop/config.yaml` で識別する管理単位 |
| Specification | 一つの実装単位の仕様。永続 UUID と表示 ID を持つ |
| Working State | 現在編集している変更可能な仕様 |
| Recorded Revision | 内容・状態・著者・Reference を固定した履歴 |
| Reference | 仕様に明示的に追加したコード・テスト・補足ファイルの参照 |
| Review Result | 特定 Revision に対する ACCEPTED / REJECTED の判断 |
| Agent Run | 特定 Revision を入力とした Agent の実行結果 |
| Git Relation | 特定 Revision と Git Commit の関連 |

`hello-1` は利用者向けの ID、内部 UUID は変わらない識別子です。`hello-1#2` は表示用の Revision 番号です。Revision 自体の永続 ID は正規化データから計算した 64 桁の SHA-256 です。

## 状態の意味

| 状態 | 意味 | Agent が直接設定 |
| --- | --- | --- |
| DRAFT | 編集・検討中 | 不可 |
| READY | 人間が実装可能と確認済み | 不可 |
| FORCEREADY | 人間が仕様不足を許容して続行を指示 | 不可 |
| IMPLEMENTED | 実装を生成したと記録済み | 可・理由必須 |
| VERIFIED | 完了条件を満たすことを確認済み | 可・理由必須 |
| CANCELED | 実装を継続しないと人間が判断 | 不可 |
| COMPLETED | 人間が作業完了を最終確認 | 不可 |

人間は任意の状態へ変更できます。順序を強制するワークフローエンジンではありません。REJECTED は状態ではなく、Revision に対するレビュー結果です。ACCEPTED も実装・検証の完了を意味しません。

## いつ履歴ができるか

通常、同じ著者が DRAFT を細かく編集するたびには履歴を増やしません。明示的に記録するなら `sloop edit 1 --record` を使います。

- `--record` は編集前を記録し、最終内容に変更があれば編集後も記録します。
- 著者が切り替わる場合は未記録の編集前を保存し、新しい著者の変更結果も記録します。
- 状態の変更は Revision を生成します。同じ状態への再指定は何もしません。
- `commit-record` は未記録の Working State を記録してからコミットに関連付けます。
- Reference の追加・削除だけでは Revision は自動生成されません。

本文・タイトル・親仕様・Reference が変わると、DRAFT 以外の状態は DRAFT に戻ります。Agent が本文を編集した結果の自動変更もこの対象です。

## 復元は新しい履歴になる

```text
A → B → C → D
            D の内容は A から復元
```

過去の Revision は変更しません。`sloop edit '1#1'` で復元を確認すると、現在の Head の子を作ります。未記録の変更は復元前に記録します。通常操作の履歴は線形で、分岐・マージ・履歴削除は提供していません。

[編集と復元の詳細](/cli/edit/)と[保存先とバックアップ](/guides/storage/)も参照してください。
