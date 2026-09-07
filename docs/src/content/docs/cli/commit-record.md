---
title: "commit-record / cr"
description: "Recorded Revision と Git Commit を Git Notes で関連付ける。"
sidebar:
  order: 13
---

```sh
sloop commit-record <specification[,specification...]> <commit[,commit...]> [著者フラグ]
sloop cr <specification[,specification...]> <commit[,commit...]> [著者フラグ]
```

```sh
sloop commit-record 1 HEAD
sloop cr 1,2 HEAD,HEAD~1
```

すべての仕様とすべてのコミットの組み合わせを関連付けます。Git が解決できるコミット ID・短縮 ID・`HEAD` 等を指定します。参照先は完全なコミット ID として保存します。

## 記録されるもの

未記録の Working State があれば先に Recorded Revision を生成し、その Head とコミットを関連付けます。新しい Git Commit を作る操作ではなく、既存コミットのメッセージやハッシュは変えません。仕様の状態も変更しません。

Git Notes と SQLite の Git Relation に `implementation` 関係を記録します。Notes ref は config の `git.notes-ref`（既定 `refs/notes/sloop`）です。同じ組み合わせを指定しても同一の Notes 関係を重複追加しません。

```sh
git log --notes=refs/notes/sloop
git notes --ref=refs/notes/sloop show HEAD
```

通常の Git log にも表示したい場合は、リポジトリ内で次を設定します。

```sh
git config notes.displayRef refs/notes/sloop
```

著者フラグは、未記録内容を記録するときの著者に使います。明示しなければ Working State の著者を使います。Notes を書くための Git 著者設定は Git 側で必要です。

## Revision の違いに注意する

関連は仕様全体ではなく、記録時点の Revision に付きます。後から状態や本文を変更すると、現在の `context` には以前の Revision の関連が出ません。必要なら変更後にも `cr` を実行します。

無効なコミット、Git リポジトリ外での操作、Git Notes 書き込み失敗、対象に既存の非 Sloop Notes がある場合はエラーになります。複数の記録処理は途中で失敗すると一部が既に保存されている可能性があるため、表示結果と Notes を確認してください。

Notes の共有は Git で別途行います。Notes には仕様全文がないため、Notes の push / fetch だけで他端末の Sloop の仕様一覧を復元することはできません。
