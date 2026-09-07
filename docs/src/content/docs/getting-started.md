---
title: "Getting Started"
description: "Sloop のインストールから最初の仕様の作成まで。"
---

## 必要な環境

macOS または Linux と、`go.mod` が要求する **Go 1.26.5 以降**を用意します。ソース取得・依存パッケージの初回ダウンロードにはネットワークが必要です。インストール後の通常操作はオフラインで実行できます。Git との関連付けには Git が必要です。

## ソースからインストールする

```sh
git clone https://github.com/m-tsuru/sloop.git
cd sloop
go install ./cmd/sloop
```

既にチェックアウト済みなら、そのルートで `go install ./cmd/sloop` を実行してください。バイナリの保存先は `GOBIN`、未設定なら `GOPATH/bin`（通常 `~/go/bin`）です。そのディレクトリを `PATH` に追加します。

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
sloop --help
```

`GOBIN` を設定している場合は、その保存先を `PATH` に追加してください。

## エディターを設定する

```sh
export SLOOP_EDITOR='vi'
```

VS Code を使う場合は、編集が終わるまで待機するようにします。

```sh
export SLOOP_EDITOR='code --wait'
```

`SLOOP_EDITOR` が空なら `EDITOR` を使い、両方が空ならエラーになります。エディターが正常終了した時点で最終内容が保存されます。

## プロジェクトを初期化する

管理したいリポジトリのルートに移動して実行します。

```sh
sloop init myapp .
```

`.sloop/config.yaml` と `.sloop/templates/default.md` が作られます。仕様データ用の SQLite と Object Store はリポジトリの外に作られます。以後はこのディレクトリ、またはその子ディレクトリから操作します。

## 最初の仕様を作る

```sh
sloop new
```

エディターで、生成された `id` はそのままにし、タイトルと本文を記述します。

```md
---
id: "myapp-1"
title: "ヘルプに利用例を表示する"
status: DRAFT
parents: []
---

## 目的 {#goal}
初めて使う人が実行方法を理解できるようにする。

## 仕様 {#specification}
`myapp --help` の標準出力の末尾に `Example: myapp hello` を表示する。
終了コードは 0 とし、ファイルやネットワークを変更しない。

## 完了条件 {#acceptance-criteria}
`myapp --help` の出力に指定した文字列があり、終了コードが 0 である。
```

保存してエディターを終了したら、内容を確認します。

```sh
sloop list
sloop view 1
```

人間が実装可能と確認してから、Agent 用の Context を出力します。

```sh
sloop ready 1
sloop context 1
sloop context 1 --json > /tmp/myapp-context.json
```

`ready` によってその内容の Recorded Revision が作られます。出力を使う Agent へ明示的に渡してください。Sloop が自動送信することはありません。

続いて [Tutorial](/tutorial/) で実装・検証結果とコミットの関連付けまで体験できます。
