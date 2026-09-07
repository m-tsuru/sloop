---
title: "index rebuild"
description: "immutable Object Store からローカル Revision 索引を再構築する。"
sidebar:
  order: 14
---

```sh
sloop index rebuild
```

追加引数・フラグはありません。Object Store の Revision を読み、ハッシュ・プロジェクト・親関係を確認して、SQLite の Revision 索引と親関係を再構築します。

```text
Rebuilt index from 3 revision objects.
```

索引から失われた仕様は Head の内容で復元します。既存の未記録 Working State は維持し、記録済みの Working State は Head に合わせます。Revision 番号は表示用なので、再構築で割り当て直す場合があります。

破損した object、別プロジェクトの object、欠けた親、複数 Head などは失敗します。分岐を自動解決する機能ではありません。

:::caution[バックアップの代わりにはなりません]
現行の再構築対象は主に Revision 索引と仕様・Reference です。SQLite を失うと、未記録の内容、Review、Agent Run、状態変更 Reason、Git Relation の索引などをこのコマンドだけで復元できません。DB を削除する前提で運用しないでください。
:::

保存するデータの範囲は[バックアップ](/guides/storage/)を参照してください。
