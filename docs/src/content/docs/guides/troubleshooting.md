---
title: "トラブルシューティング"
description: "よくある CLI エラーと現在の実装上の制約への対処。"
---

## sloop が見つからない

`go install ./cmd/sloop` をリポジトリルートで実行し、`GOBIN` または `GOPATH/bin` が `PATH` に含まれているか確認します。ビルドできない場合は `go version` と `go.mod` の要求バージョンを確認してください。

## not a Sloop project

現在地から親ディレクトリに `.sloop/config.yaml` がありません。管理対象のリポジトリに移動するか、未初期化なら `sloop init <slug> .` を実行します。

## neither SLOOP_EDITOR nor EDITOR is set

```sh
export SLOOP_EDITOR='vi'
```

GUI エディターでは `code --wait` のような待機オプションを設定してください。エディターが異常終了した場合、編集後の内容は取り込みません。ただし `--record` や復元で編集前に作られた履歴は残ります。

## has unrecorded working changes

固定 Context に対応する Revision がありません。内容を確認して READY にするか、状態を変えずに記録したい場合は `sloop edit 1 --record` を実行します。確認だけなら `sloop context 1 --working` を使います。

同じ状態の再指定は何もしません。例えば未記録の DRAFT に `sloop draft 1` を実行しても記録されません。

## Agent の状態変更が拒否される

`--reason` に判断根拠を書き、設定先を IMPLEMENTED / VERIFIED に限定します。現在の Recorded Revision が必要です。READY・FORCEREADY・COMPLETED は人間が判断します。Agent の Markdown 編集で status を変更しても拒否されます。

## FORCEREADY を reject できない

FORCEREADY は人間が仕様不足を許容した状態なので、仕様不足の REJECTED Review は記録できません。環境の技術的失敗は `agent-run failed` を使用します。

## Section が見つからない

`## 目的 {#goal}` のように見出し末尾に ID があるか、スペル・大文字小文字が一致するか確認します。`view --section` は存在しない ID で失敗し、`query` は該当 Section のない仕様をスキップします。

## Reference 追加後に DRAFT になった

Reference も仕様内容です。正常な動作なので、変更内容を確認して必要な状態へ人間が変更します。実装コミットとの関連だけを記録したい場合は `commit-record` を使います。

## 仕様やコミットの関連が見えない

`project.id`、`spec-prefix`、現在のディレクトリを確認します。prefix 変更前の仕様には `sloop view old-prefix-1` のように完全な ID を指定します。

Git Relation は Revision ごとに記録されるため、状態変更後の Context には以前の Revision の関連が出ません。必要なら現在の Revision に対して `sloop cr 1 HEAD` を実行します。

## revision prefix is ambiguous

短縮ハッシュが複数 Revision に一致しています。より長いハッシュ、完全なハッシュ、または `'myapp-1#2'` のような Revision 番号を使います。

## Git Notes を記録できない

Git リポジトリ内か、コミットが存在するか、Git の `user.name` と `user.email` が設定されているか確認します。`--author.name` は Sloop の著者であり、Git Notes のコミットに必要な Git 設定の代わりにはなりません。既存の非 Sloop Notes は `commit-record` で上書きせずエラーになります。

## index rebuild が失敗する

破損 object・欠けた親・別プロジェクト・複数 Head の可能性があります。object を手で書き換えず、[バックアップ](/guides/storage/)からの復元を検討してください。履歴の分岐を自動で統合する機能はありません。
