---
title: "bind"
description: "Feature と現在存在する Implementation/Test Symbol の Binding を管理する。"
sidebar:
  order: 8
---

Feature Binding は Specification 内の `func:<feature-id>` と Repository 上の Symbol の事実上の対応です。未来の実装場所や実装要求は表しません。

## 追加

```sh
sloop bind add <specification> <feature> --impl <path:function>
sloop bind add <specification> <feature> --impl <path:type:method>
sloop bind add <specification> <feature> --test <path:test-symbol>
```

`--impl` と `--test` は複数回、または同時に指定できます。Go (`.go`) と Python (`.py`) の Function/Method を解決し、`RESOLVED` の Locator だけを追加します。存在しない Symbol、複数候補、未対応拡張子は拒否されます。

## 削除

```sh
sloop bind remove <specification> <feature> --impl <locator>
sloop bind remove <specification> <feature> --test <locator>
```

存在しない Binding の削除は成功扱いです。不正な Locator はエラーになります。

## 一覧と Resolution Status

```sh
sloop bind list <specification>
sloop bind list <specification> <feature>
sloop bind list <specification> --json
```

各 Locator に `RESOLVED`、`MISSING`、`AMBIGUOUS`、`UNSUPPORTED` のいずれかを表示します。Feature を指定した JSON は単一 object、指定しない JSON は Feature object の配列です。

Binding の変更は Working State の意味変更として扱われ、必要に応じて DRAFT に戻ります。非 DRAFT からの解決済み Binding 変更と Agent による Binding 変更は DRAFT Revision として記録されるため、Coding Agent は複数回に分けて Binding を更新した後でも、根拠付きで IMPLEMENTED へ進めます。
