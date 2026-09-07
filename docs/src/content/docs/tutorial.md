---
title: "Tutorial：仕様から実装へ"
description: "小さな Go プロジェクトで仕様・実装・検証・コミットを結び付ける。"
---

[Sloop のインストールとエディター設定](/getting-started/) を済ませてから進めます。ここでは新しい練習用リポジトリで、標準出力に挨拶する Go プログラムを作ります。

## 1. 練習用プロジェクトを作る

```sh
mkdir sloop-tutorial
cd sloop-tutorial
git init
go mod init example.com/hello
sloop init hello .
```

Git の `user.name` と `user.email` が未設定なら、このリポジトリに設定してください。以下は自分の値に置き換えます。

```sh
git config user.name 'Your Name'
git config user.email 'you@example.com'
```

## 2. 実装仕様を書く

```sh
sloop new
```

エディターの内容を次のようにします。

```md
---
id: "hello-1"
title: "挨拶を標準出力に表示する"
status: DRAFT
parents: []
---

## 目的 {#goal}
実行できる最小限の挨拶プログラムを作る。

## 仕様 {#specification}
Go の main package をリポジトリルートに置く。
実行すると標準出力に `Hello, Sloop!` と LF 改行を 1 回出力する。
標準エラー出力は空、正常終了コードは 0 とする。
引数は無視する。ファイルへの書き込みとネットワーク通信は行わない。
`greeting() string` は改行なしの挨拶文字列を返す。

## 完了条件 {#acceptance-criteria}
`go run .` の出力が指定の挨拶と LF 改行に一致する。
引数を追加しても出力が変化しない。
`greeting()` の返り値をテストし、`go test ./...` が成功する。
```

関連ファイルを Context の参照に加えます。

```sh
sloop ref add 1 --kind context go.mod
sloop ref list 1
sloop view 1
```

## 3. 人間が READY にして、Agent に渡す

```sh
sloop ready 1
sloop log 1
sloop context 1 --json > /tmp/hello-context.json
```

利用する Agent にこの JSON と必要なファイルを渡し、仕様に従う実装を依頼します。Reference は参照先の情報であり、ファイル全文が JSON に埋め込まれるわけではありません。

仕様不足があれば、Agent は `reject` に具体的な理由を記録し、人間に修正を求めます。修正後はもう一度 `ready` にして Context を取り直します。[レビュー手順](/guides/agents/)を参照してください。

## 4. 実装する

この例は手動でも再現できます。`main.go` を作成します。

```go
package main

import "fmt"

func greeting() string { return "Hello, Sloop!" }

func main() { fmt.Println(greeting()) }
```

`main_test.go` を作成します。

```go
package main

import "testing"

func TestGreeting(t *testing.T) {
    if got := greeting(); got != "Hello, Sloop!" {
        t.Fatalf("unexpected greeting: %q", got)
    }
}
```

Agent による作業なら、結果と状態をそれぞれ記録します。手動で実装した場合は `--author.name` を自分の名前に置き換え、`--author.agent true` を省略してください。`agent-run` は Agent が実行した場合だけ記録します。

```sh
sloop agent-run 1 implemented --author.name coding-agent \
  --reason 'main.go と greeting の単体テストを作成した。'
sloop implemented 1 --author.name coding-agent --author.agent true \
  --reason '指定された挨拶の出力処理と greeting の単体テストを実装した。'
```

`agent-run` は入力の Revision に実行結果を残し、`implemented` は新しい状態の Revision を作ります。

## 5. 検証する

```sh
go test ./...
go build -o /tmp/sloop-tutorial-hello .
printf 'Hello, Sloop!\n' > /tmp/sloop-tutorial-expected.txt
/tmp/sloop-tutorial-hello > /tmp/sloop-tutorial-actual.txt 2> /tmp/sloop-tutorial-stderr.txt
test "$?" -eq 0
cmp /tmp/sloop-tutorial-expected.txt /tmp/sloop-tutorial-actual.txt
test ! -s /tmp/sloop-tutorial-stderr.txt
/tmp/sloop-tutorial-hello ignored > /tmp/sloop-tutorial-actual.txt 2> /tmp/sloop-tutorial-stderr.txt
test "$?" -eq 0
cmp /tmp/sloop-tutorial-expected.txt /tmp/sloop-tutorial-actual.txt
test ! -s /tmp/sloop-tutorial-stderr.txt
```

各コマンドの成功と、コードがファイル書き込み・ネットワーク通信を行わないことを確認します。成功を確認したときだけ次を実行します。

```sh
sloop verified 1 --author.name coding-agent --author.agent true \
  --reason 'go test が成功し、通常実行と引数付き実行の出力・終了コード・空の標準エラーを確認した。コードにファイル書き込みや通信がないことも確認した。'
```

Sloop はこのコマンドでテストを自動実行しません。

## 6. コミットを関連付け、人間が完了する

```sh
git add go.mod main.go main_test.go .sloop
git commit -m 'Add greeting program'
sloop commit-record 1 HEAD
git log -1 --notes=refs/notes/sloop
```

その時点の VERIFIED Revision と Git Commit が関連付きます。

```sh
sloop complete 1 --reason '実装と検証結果を確認し、この仕様の作業を完了した。'
sloop list --filter status:completed
sloop log 1
```

最後の `complete` は人間の判断です。COMPLETED への変更でさらに Revision が増えるため、現在の Context にもコミットの関連を付けたい場合は `sloop cr 1 HEAD` を再度実行します。

## 7. 実装ファイルも Reference に加える場合

```sh
sloop ref add 1 --kind code main.go --at HEAD
sloop ref add 1 --kind test main_test.go --at HEAD
```

Reference は仕様内容の一部なので、追加すると Working State は DRAFT に戻ります。単に Git Commit との関連を記録したい場合は `commit-record` を使います。Reference を変更した後は内容を確認し、必要な状態へ人間が明示的に変更してください。
