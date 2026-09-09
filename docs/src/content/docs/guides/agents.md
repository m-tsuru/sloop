---
title: "Coding Agent との連携"
description: "Agent に仕様を渡し、レビュー・実装・検証の根拠を残す手順。"
---

## 人間が実装開始を判断する

```sh
sloop view 1
sloop ready 1
sloop context 1 --json > /tmp/sloop-context.json
```

生成した Context を利用する Agent に渡します。Sloop Core は特定ベンダーの API を呼ばず、CLI の出力で連携します。MCP Server や自動実行 Connector は現在ありません。

この Repository には Codex plugin と `sloop-implement` Skill も含まれます。Plugin 導入後は `/sloop implement sloop-1` または同等の自然言語で Workflow を開始できます。

次のような作業指示を Context とともに渡せます。

```text
この Context の Revision を基準に作業してください。
参照されたコード・テストを読み、仕様にない外部挙動を勝手に決めないでください。
READY で必要な挙動が未定義なら、具体的な不足を reject に記録して実装を止めてください。
Agent の書き込み操作では --author.name と --author.agent true を指定してください。
実装後は入力 Revision に agent-run の結果を記録し、根拠付きで implemented にしてください。
完了条件を実際に確認した後で verified にしてください。
complete は人間が実行します。
```

Reference は参照先のメタデータです。対象ファイルを Agent が読める環境で実行するか、明示的に必要な内容を渡します。

Agent が Feature に対応する新しい Function/Method または Test Symbol を作成した場合、状態変更前に Binding を記録します。

```sh
sloop bind add 1 markdownlint --impl internal/cmd/cmd.go:LintMarkdown \
  --author.name coding-agent --author.agent true
sloop bind add 1 markdownlint --test internal/cmd/cmd_test.go:TestLintMarkdown \
  --author.name coding-agent --author.agent true
```

Locator は Symbol 作成後にのみ追加でき、Sloop が `RESOLVED` と判定できない追加は拒否されます。

## 仕様不足を見つけた場合

```sh
sloop reject 1 --author.name coding-agent --author.agent true \
  --reason '入力が空の場合の戻り値と終了コードが未定義。期待する挙動を指定してください。'
```

状態は READY のままです。人間が編集し、判断し直します。

```sh
sloop edit 1
sloop ready 1
sloop context 1 --json
```

編集によって DRAFT に戻り、READY にすることで新しい Revision に基づく Context ができます。古い REJECTED Review は履歴に残り、新しい Revision には適用されません。

## 不足を許容して続行する場合

```sh
sloop force-ready 1
sloop context 1 --json
```

これは人間の明示的な判断です。FORCEREADY に対する仕様不足の `reject` は拒否されます。実行環境が壊れている場合などは `sloop agent-run 1 failed --reason '...'` で技術的失敗を記録できます。

## 編集と状態変更を分ける

Agent も仕様を編集できますが、Front Matter の `status` を直接変更できません。

```sh
sloop edit 1 --author.name coding-agent --author.agent true
```

本文等が変わると自動的に DRAFT になるので、人間が必要に応じて READY に戻します。Agent が `--author.agent` を省いて人間の操作として実行しないようにしてください。このフラグは操作の申告であり、認証の仕組みではありません。

## 作業中の Revision を確認する

状態変更・レビュー・Agent Run は、そのコマンド実行時の現在の Revision に記録されます。Context に書かれた Full Hash を残し、別の編集が入っていないか `context --json` で比較してから記録します。基準 Revision を固定指定するフラグは現行 CLI にありません。

Agent の実行中に同じ仕様を別の利用者が変更した場合は、そのまま結果を記録せず新しい Context との整合を確認してください。現在の CLI は複数 Agent の作業を自動調停するものではありません。
