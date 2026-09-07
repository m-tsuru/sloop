# Sloop

## 1. 概要

Sloop は、LLM に与える実装仕様と、その仕様に関連するソースコード・テスト・Git Commit・Coding Agent との関係をローカルで管理する CLI アプリケーションである。

Sloop は、以下の課題を解決することを目的とする。

- ユーザ自身が仕様を書き、その仕様とソースコード・テスト・Git Commit を明確に紐づけ、それを Coding Agent へ渡せるようにする。
- 仕様全文をユーザ自身が記述するワークフローと、AI が仕様レビューへ介入するワークフローの双方を許容する。
- Coding Agent が実装に必要な仕様不足を発見した場合、その内容を推測して補完せず、REJECT できるようにする。
- Claude、ChatGPT、GitHub Copilot 等の特定ベンダーにロックインしない。
- `/docs` や `AGENTS.md` にタスク固有の仕様を蓄積し、無関係な情報まで Coding Agent のコンテキストへ投入することを避ける。
- GitHub Issues、Notion、CI 等の外部サービスを仕様の正本としない。
- アカウント停止・サービス終了等の影響を受けず、データを利用者自身が保持できるようにする。
- Sloop に記録された仕様を、ユーザ向けドキュメント作成時の情報源として再利用できるようにする。ただし、Sloop 自身をドキュメントジェネレータにはしない。
- Git と同様にローカルファーストかつ分散型とし、インターネット接続がなくても仕様の作成・編集・参照・履歴管理を行えるようにする。

---

# 2. 基本原則

## 2.1 ローカルファースト

Sloop の通常操作はインターネット接続を必要としてはならない。

以下はオフラインで動作しなければならない。

- `sloop init`
- `sloop new`
- `sloop edit`
- `sloop view`
- `sloop list`
- `sloop status`
- `sloop ref`
- Revision 管理
- Git Commit との関連付け
- Coding Agent 用 Context の出力

v0.1 の必須操作はすべてオフラインで動作する。分散同期機能は将来仕様とする。

---

## 2.2 Vendor Neutral

Sloop Core は特定の LLM ベンダー API を必須依存としてはならない。

Coding Agent との連携は CLI、標準入出力、JSON、MCP 等の交換可能なインターフェースを介して行う。

Sloop の仕様データは、Claude、ChatGPT、GitHub Copilot その他の Coding Agent から同一の意味で参照可能でなければならない。

---

## 2.3 Specification First

Coding Agent は仕様に記載されていない、外部から観測可能な挙動を独自に決定してはならない。

例えば以下は、仕様で必要な場合には明示されなければならない。

- エラー時の挙動
- API のレスポンス
- デフォルト値
- 削除方式
- timeout
- retry
- UI の挙動
- 永続化方式
- 互換性要件

Coding Agent が実装に必要な仕様不足を発見した場合は、推測して補完するのではなく Review Result として `REJECTED` を返さなければならない。

ただし、仕様が `FORCEREADY` の場合はこの限りではない。

---

# 3. 実装技術

Sloop Core は Go で実装する。

ローカル状態・検索用インデックス・Working State の保存には SQLite を利用する。

外部の DB daemon を必要としてはならない。

macOS および Linux をサポートする。

---

# 4. Project

## 4.1 初期化

Sloop Project は次のコマンドで初期化する。

```console
$ sloop init <project-slug> <repository-path>
```

例:

```console
$ pwd
/Users/tsuru/sloop

$ sloop init sloop .
Project 'sloop' is created.
Project ID: sloop-2deb2e1c-925f-40df-a9b0-4e7ac25b124d
```

Project ID は、

```text
<project-slug>-<UUID>
```

の形式とする。

UUID は UUID v4 とする。

---

## 4.2 `.sloop` directory

Repository root に以下を作成する。

```text
.sloop/
├── config.yaml
└── templates/
    └── default.md
```

`.sloop/` にはプロジェクト共有可能な設定のみを配置する。

SQLite database、検索インデックス、Embedding 等のローカル生成物を `.sloop/` に保存してはならない。

---

## 4.3 Local Data Directory

Linux では以下を使用する。

```text
${XDG_DATA_HOME:-~/.local/share}/sloop/<project-id>/
```

macOS では以下を使用する。

```text
~/Library/Application Support/sloop/<project-id>/
```

最低限、次のデータを保持する。

```text
sloop.db
objects/
cache/
```

---

# 5. Data Model

Sloop は以下を区別する。

```text
Working State
Recorded Revision
Specification
Reference
Review Result
Agent Run
Git Relation
```

---

# 6. Specification

Specification は一つの実装単位を表す。

Specification は内部的な永続 Identity と、ユーザ向けの表示・参照用 ID を持つ。

内部的な永続 Identity には UUID v4 を使用する。Specification UUID は Specification 作成時に生成し、その後変更してはならない。

概念上、以下の形式とする。

```yaml
specification:
  uuid: 19b95238-e270-49ab-a139-ea9ca467b7de
  id: sloop-15
```

`uuid` は Specification の永続 Identity とする。

`id` は Project 内でユーザが利用する表示・参照用 Identifier とし、

```text
<spec-prefix>-<number>
```

の形式とする。

デフォルトの `spec-prefix` は Project Slug とする。

例えば、

```text
sloop-1
sloop-2
sloop-3
```

となる。

Prefix は `.sloop/config.yaml` で変更可能とする。

通常の CLI 操作では UUID を表示・入力する必要はない。

```console
$ sloop edit 15
$ sloop view sloop-15
```

のように表示用 ID を利用できる。

将来、複数 Sloop Client 間で同一の表示 ID を持つ別 Specification が作成された場合も、UUID により区別できなければならない。

---

# 7. Specification の作成

次のコマンドで Specification を作成する。

```console
$ sloop new
```

Sloop は一時 Markdown ファイルを作成し、環境変数 `$EDITOR` で指定された Editor を起動する。

`$SLOOP_EDITOR` が設定されている場合は `$EDITOR` より優先する。

どちらも設定されていない場合はエラーとする。

一時ファイルは Repository 内へ保存してはならない。

Editor が正常終了した場合のみ内容を Sloop に反映する。

---

## 7.1 Editable Markdown Format

Editor に渡される Markdown は以下の形式とする。

```md
---
id: "sloop-1"
title: "sloop での Markdown 仕様書について"
status: DRAFT
parents: []
---

## 目的 {#goal}

## 仕様 {#specification}

## 完了条件 {#acceptance-criteria}
```

Editable Markdown に内部履歴情報を含めてはならない。

特に以下はユーザが直接編集できない。

- Revision Hash
- Revision Log
- Git Relations
- Review Results
- Agent Runs

---

## 7.2 Front Matter

Front Matter のフィールドは最低限以下とする。

### `id`

Specification ID。

通常は変更不可。

### `title`

Specification のタイトル。

### `status`

Specification Status。

### `parents`

関連する親 Specification の ID。

例:

```yaml
parents:
  - sloop-1
  - sloop-3
```

`children` は保存しない。

子 Specification は、他 Specification の `parents` を逆引きして算出する。

---

# 8. Markdown Section ID

Markdown heading には次の形式で Section ID を指定できる。

```md
## 目的 {#goal}
```

v0.1 では以下を予約する。

```text
goal
specification
acceptance-criteria
```

それぞれ、

```text
goal
    Specification の目的

specification
    実装仕様

acceptance-criteria
    完了・検証条件
```

を表す。

その他の Section ID も保存・検索可能とする。

未知の Section ID をエラーにしてはならない。

---

# 9. Template

デフォルト Template は、

```text
.sloop/templates/default.md
```

に保存する。

Template には YAML Front Matter を含めない。

例:

```md
## 目的 {#goal}

## 仕様 {#specification}

## 完了条件 {#acceptance-criteria}
```

別 Template は、

```console
$ sloop new --template hoge
```

で利用する。

この場合、

```text
.sloop/templates/hoge.md
```

を読み込む。

---

# 10. Specification Status

Specification Status は以下とする。

```text
DRAFT
READY
FORCEREADY
IMPLEMENTED
VERIFIED
CANCELED
COMPLETED
```

`REJECTED` は Specification Status ではない。

`REJECTED` は Coding Agent の Review Result とする。

## 10.1 DRAFT

Specification が編集・検討中であり、ユーザによって `READY` または `FORCEREADY` と確認されていない状態を表す。

DRAFT の Specification は人間または Agent のどちらからも編集可能である。

## 10.2 READY

ユーザが、

> この仕様は Coding Agent が実装可能な程度まで定義されている

と確認した状態。

READY の最終判断主体はユーザとする。

## 10.3 FORCEREADY

ユーザが、

> 仕様不足が存在しても Coding Agent に実装を続行させる

と明示した状態。

FORCEREADY の Agent Run では、仕様不足を理由とした `REJECTED` Review Result を記録してはならない。

ただし、実行環境のエラー等による技術的失敗は `FAILED` Agent Result として記録できる。

## 10.4 IMPLEMENTED

Specification に基づく実装が生成されたと記録された状態を表す。

通常は Coding Agent がこの Status を設定するが、人間も Human Transition として明示的に設定できる。

## 10.5 VERIFIED

Specification の Acceptance Criteria を満たしていると確認された状態を表す。

通常は Coding Agent または Verification Process がこの Status を設定するが、人間も Human Transition として明示的に設定できる。

VERIFIED は COMPLETED と同義ではない。

## 10.6 CANCELED

その Specification に対する実装を継続しないとユーザが判断した状態を表す。

## 10.7 COMPLETED

ユーザが Specification に対する作業を完了したと最終確認した状態を表す。

COMPLETED の最終判断主体はユーザとする。

Status は「誰が操作したか」ではなく Specification の意味上の状態を表し、操作主体の制約は Status Transition の仕様で定義する。

---

# 11. Status Transition

Sloop は、人間による Status Transition と Agent による Status Transition を区別する。

## 11.1 Human Transition

人間は Specification の所有者として、原則として任意の Status へ直接変更できる。

Sloop は、人間に対して厳密な lifecycle の順序を強制しない。

例えば以下の遷移をすべて許可する。

```text
DRAFT → READY
DRAFT → COMPLETED
COMPLETED → DRAFT
VERIFIED → READY
READY → CANCELED
```

不自然な状態遷移について Sloop は warning を表示してもよいが、人間が明示的に要求した Status Transition を拒否してはならない。

Status は workflow を補助する metadata であり、人間の操作を拘束する workflow engine として扱ってはならない。

---

## 11.2 Agent Transition

Agent が Specification の Status を変更する場合、Status Transition Record として最低限以下の3項目を記録しなければならない。

```text
Specification
Status
Reason
```

それぞれ次の意味を持つ。

### Specification

状態変更の対象となる Specification。

Specification ID および、その変更判断の基準となった Recorded Revision を特定できなければならない。

内部的には、その判断の基準となった Revision を `based-on-revision` として保持する。

概念上、以下の形式とする。

```yaml
specification:
  id: sloop-1
  based-on-revision: a1b2c3d4
```

`based-on-revision` は、

> Agent がどの内容の Specification を読んでその判断を行ったか

を表す。

Agent Status Transition は `based-on-revision` の Recorded Revision を直接変更してはならない。Status Transition によって新しい Recorded Revision を生成する。

例えば、

```text
a1b2c3d4 (READY)
      ↓
e5f6a7b8 (IMPLEMENTED)
```

となる。

新しい Revision の parent は `based-on-revision` とする。

### Status

Agent が Specification に設定する新しい Status。

Agent が設定できる Status は以下に限定する。

```text
IMPLEMENTED
VERIFIED
```

`FORCEREADY` についても同様に、

```text
FORCEREADY
    ↓
IMPLEMENTED
    ↓
VERIFIED
```

と遷移できる。

Agent は以下の Status を設定してはならない。

```text
DRAFT
READY
FORCEREADY
CANCELED
COMPLETED
```

これらはユーザの意思または編集操作によって決定される Status とする。

### Reason

Agent がその Status Transition を行う根拠を自然言語で記述する。

Reason は空文字列であってはならない。

Reason は単に、

```text
done
completed
success
```

のような状態名の言い換えだけであってはならず、その Status を設定できると判断した根拠を説明する。

例えば `IMPLEMENTED` への変更では、

```text
Implemented the parser described by the specification and added the required tests.
```

のように記録できる。

`VERIFIED` への変更では、

```text
All acceptance criteria are covered by tests and `go test ./...` completed successfully.
```

のように、その検証根拠を記録する。

---

## 11.3 Agent Transition Record

Agent による Status Transition は履歴として保存する。

概念上、以下の形式とする。

```yaml
transition:
  specification: sloop-1
  based-on-revision: a1b2c3d4
  resulting-revision: e5f6a7b8
  status: IMPLEMENTED
  reason: >
    Implemented the Markdown parser and added tests covering
    the behavior defined in the specification.
```

`based-on-revision` は Agent が判断に利用した Recorded Revision、`resulting-revision` は Status Transition により生成された Recorded Revision を表す。

Author、timestamp 等の provenance metadata は Sloop が別途記録してよい。

ただし、Agent が Status Transition を行うために入力すべき意味上必須の情報は、

```text
Specification
Status
Reason
```

の3項目とする。`based-on-revision` は対象 Specification の current Recorded Revision から Sloop が解決してよい。

---

## 11.4 Specification Rejection

仕様不足を発見した Agent は Specification Status を変更してはならない。

この場合は Status Transition ではなく Review Result として、

```text
REJECTED
```

を記録する。

REJECTED Review についても対象 Specification と Reason を必須とする。

Review Result は必ず、Agent が判断対象とした Recorded Revision に紐づける。

概念上、

```yaml
review:
  specification: sloop-1
  revision: a1b2c3d4
  result: REJECTED
  reason: >
    The behavior when the configuration file already exists
    is not specified.
```

とする。

`REJECTED` は Specification Status ではないため、

```text
READY → REJECTED
```

のような Status Transition は存在しない。

新しい Recorded Revision が生成された場合、旧 Revision に対する REJECTED Review は履歴として保持するが、新 Revision に対する Review Result として扱ってはならない。

---

# 12. Status CLI

人間による Status の変更は以下の形式とする。

```console
$ sloop status <specification> <status>
```

例:

```console
$ sloop status 1 ready
```

複数 Specification を指定できる。

```console
$ sloop status 1,2,3 draft
```

完全な Specification ID も使用可能とする。

```console
$ sloop status sloop-1 completed
```

---

## 12.1 Human Status Transition

人間による Status Transition では `--reason` を必須としない。

例えば、

```console
$ sloop status 1 completed
```

だけで状態を変更できる。

必要であれば、人間も任意で Reason を記録できる。

```console
$ sloop status 1 completed \
    --reason "Manual verification completed."
```

---

## 12.2 Agent Status Transition

Agent が Status を変更する場合は、

```text
Specification
Status
Reason
```

の3項目を必須とする。

CLI では Specification と Status を positional argument、Reason を `--reason` で指定する。

例:

```console
$ sloop status 1 implemented \
    --reason "Implemented the requested parser and added corresponding tests." \
    --author.name coding-agent \
    --author.agent true
```

または、

```console
$ sloop status 1 verified \
    --reason "All acceptance criteria passed and go test ./... succeeded." \
    --author.name coding-agent \
    --author.agent true
```

`--author.agent true` が指定されているにもかかわらず `--reason` が存在しない場合、Sloop は操作を拒否しなければならない。

例:

```text
Error: agent status transitions require a reason.
```

Agent が許可されていない Status を指定した場合も拒否する。

例えば、

```console
$ sloop status 1 completed \
    --reason "Implementation is finished." \
    --author.name coding-agent \
    --author.agent true
```

は失敗する。

```text
Error: an agent cannot set specification status to COMPLETED.
```

---

## 12.3 Status Alias

以下の alias command を提供する。

```console
$ sloop draft 1
$ sloop ready 1
$ sloop force-ready 1
$ sloop implemented 1
$ sloop verified 1
$ sloop cancel 1
$ sloop complete 1
```

これらは `sloop status` の alias とする。

例えば、

```console
$ sloop ready 1
```

は、

```console
$ sloop status 1 ready
```

と同義である。

Agent が alias を使用する場合にも Agent Transition の制約を適用する。

したがって、

```console
$ sloop implemented 1 \
    --reason "Implemented all behavior defined by the specification." \
    --author.name coding-agent \
    --author.agent true
```

は許可される。

一方、

```console
$ sloop complete 1 \
    --reason "Everything is finished." \
    --author.name coding-agent \
    --author.agent true
```

は拒否される。

Alias 自体に独自の Status Transition semantics を持たせてはならない。


---

# 13. Editing and Working State

既存 Specification は次のコマンドで編集する。

```console
$ sloop edit 1
```

または、

```console
$ sloop edit sloop-1
```

Sloop は DB 内の Working State を一時 Markdown として Editor に渡す。

Editor 終了後、その内容を再度 Sloop 内へ保存する。

通常編集では、同一著者による細かな編集のたびに Recorded Revision を増加させない。

## 13.1 Meaningful Edit と DRAFT

`READY`、`FORCEREADY`、`IMPLEMENTED`、`VERIFIED`、`CANCELED`、`COMPLETED` の Specification の意味内容が変更された場合、その Working State の Status は自動的に `DRAFT` に変更する。

意味内容の変更には最低限以下を含む。

- Markdown 本文の変更
- `title` の変更
- `parents` の変更
- Reference の追加
- Reference の削除

`status` のみを変更した場合はこの規則を適用しない。

Editor を開いたが最終内容に差分がない場合も適用しない。

例えば、

```text
READY
  ↓
sloop edit
  ↓
本文変更
  ↓
DRAFT
```

となる。

人間が編集後も実装可能と判断した場合は、

```console
$ sloop ready 1
```

によって再び READY にできる。

## 13.2 Agent による Markdown 編集と Status

Editable Markdown の Front Matter に `status` を含めることは許可する。

ただし Agent Author による編集では `status` を直接変更してはならない。

Agent が `sloop edit` によって Specification を編集し、Editor 終了後に Front Matter の `status` が変更されていた場合は操作を拒否する。

例:

```text
Error: agents cannot change specification status through Markdown editing.
Use `sloop status` with a reason instead.
```

Agent による Status Transition は必ず `sloop status` または対応する Status Alias を利用し、Agent Transition の制約に従わなければならない。

人間による Markdown 編集では `status` の直接変更を許可する。

---

# 14. Recorded Revision

Sloop は、

```text
Working State
```

と、

```text
Recorded Revision
```

を区別する。

Working State は現在編集中の内容であり、変更可能である。

Recorded Revision は明示的に記録された immutable snapshot であり、一度作成された内容を変更してはならない。

これにより、

> ログを無制限に増加させない

ことと、

> 過去の Revision を確実に識別する

ことを両立する。

---

# 15. `--record`

ユーザが現在の内容を明示的に履歴へ残したい場合、

```console
$ sloop edit 1 --record
```

を実行する。

処理順は以下とする。

```text
current Working State
        ↓
immutable Recorded Revision を作成
        ↓
Editor を開く
        ↓
編集
        ↓
新しい Working State を保存
        ↓
変更が存在する場合、新しい Recorded Revision を作成
```

したがって `--record` は、編集前と編集後の境界を明示的に履歴として残す操作となる。

Editor 内で何回ファイル保存を行っても、Sloop が観測するのは Editor process 終了時の最終内容のみとする。

---

# 16. Author Boundary

Working State の Author と異なる Author が編集を開始した場合は、Author Boundary とみなす。

この場合、現在の Working State が未記録であれば自動的に Recorded Revision を作成する。

新しい Author による編集結果も Recorded Revision として記録する。

これにより、

```text
Human
↓
AI
↓
Human
```

のような変更履歴を追跡可能にする。

---

# 17. Status と Revision

Status を変更した場合、その状態変更は履歴上重要なイベントとして Recorded Revision に記録する。

同一 content であっても Status が異なる場合は異なる Revision とする。

---

# 18. Revision Identity

Recorded Revision は immutable とする。

各 Revision は `revision-hash` を持つ。

`revision-hash` は Revision 内容を正規化した canonical representation に対する SHA-256 とする。

Hash の対象には最低限以下を含む。

```text
project-id
specification-uuid
specification-id
parent revision hashes
author
content
status
parents
references
```

Timestamp および表示用 Revision Number は revision-hash の生成対象に含めない。

Specification の永続 Identity を保証するため、`specification-uuid` を hash 対象へ含める。

Canonical representation は UTF-8、LF 改行、安定した key ordering を用いる。

複数の parent revision hash が存在する場合、canonical representation 上では parent hash を SHA-256 の文字列表現による昇順で並べる。parent の指定順によって Revision Hash が変化してはならない。

## 18.1 Full Hash と Short Hash

Revision の永続 Identity として利用する SHA-256 は常に完全な 64 桁 hexadecimal string とする。

Database、Object Store、JSON interface、Connector protocol では Full Hash を使用する。

CLI の human-readable output では Short Hash を表示できる。

デフォルトの Short Hash 長は 8 桁とする。

CLI input では Full Hash または一意に Revision を特定できる Short Hash を指定できる。

```console
$ sloop edit a1b2c3d4
```

Short Hash が複数 Revision に一致する場合は操作を拒否する。

```text
Error: revision prefix 'a1b2c3d4' is ambiguous.
Please provide a longer revision hash.
```

Short Hash は表示・入力上の省略表現であり、Recorded Revision の Identity として保存してはならない。

---

# 19. Revision Number

ユーザ向けには Revision を連番で表示する。

例:

```text
sloop-1#1
sloop-1#2
sloop-1#3
```

ただし Revision Number は表示用であり、永続 Identity ではない。

永続 Identity は Revision Hash とする。

分散同期により Revision Number の衝突が発生した場合、ローカル表示番号は再割当してよい。

Revision Number の変更によって Revision Hash は変化しない。

---

# 20. Revision DAG

Revision の内部構造は、将来の分散同期・merge に対応できるよう DAG として表現可能にする。

各 Recorded Revision は 0 個以上の parent revision hash を持つ。

- Specification の最初の Recorded Revision は parent を持たない。
- v0.1 の通常の線形編集で生成される Revision は 1 個の parent を持つ。
- 将来的に merge 等を実装する場合は 2 個以上の parent を持つことができる。

v0.1 の通常操作では単一 Head の線形履歴のみを生成する。

```text
A → B → C → D
```

Multiple Head、Explicit Branching、Divergence Resolution は v0.1 では実装せず、分散同期機能とともに将来仕様として定義する。

---

# 21. 過去 Revision の復元

過去 Revision を指定して編集できる。

```console
$ sloop edit 1#1
```

または、

```console
$ sloop edit a1b2c3d4
```

現在 Head が Revision 2 の場合は、例えば次の確認を表示する。

```text
Revision 2 is currently the latest revision.
Restore the content of revision 1 (a1b2c3d4) as a new revision and open it for editing? [y/N]
```

Yes の場合、過去 Revision 自体は変更しない。

現在 Head を parent とする新しい Revision を作成し、その content として選択 Revision の内容を利用する。

つまり、

```text
A → B → C
```

から A を復元した場合、

```text
A → B → C → D
            content = A
```

となる。

これにより、通常の restore では履歴を線形に維持する。

---

# 22. Explicit Branching

Explicit Branching は v0.1 では実装しない。

したがって、以下の CLI は v0.1 では提供しない。

```console
$ sloop edit 1#1 --branch-from
```

過去 Revision の復元は常に現在 Head の子として新 Revision を生成し、v0.1 の通常操作から複数 Head を生成してはならない。

Explicit Branching は分散同期・Multiple Head の仕様とともに将来仕様として定義する。

---

# 23. Diverged History

Diverged History は v0.1 では発生させない。

複数 Sloop Client 間の同期、Multiple Head の検出、merge / resolution は将来仕様とする。

Sloop の内部データモデルは将来の DAG を表現可能であってよいが、v0.1 の CLI は divergence を生成・解決する機能を提供しない。

---

# 24. Revision History

Revision 履歴は次のコマンドで表示する。

```console
$ sloop log 1
```

例:

```text
Specification: sloop-1

Rev  Hash       Status       Author                 Date
1    a1b2c3d4   DRAFT        m-tsuru                2026-09-07 21:29
2    e5f6a7b8   READY        m-tsuru                2026-09-07 21:45
3    17af05de   IMPLEMENTED  codex-5.6-sol          2026-09-07 22:03
```

---

# 25. Revision Pruning

v0.1 の通常操作では単一 Head の線形履歴のみを生成するため、履歴 branch を対象とした Revision Pruning は実装しない。

将来、Explicit Branching または分散同期を実装する際に、不要な branch の Recorded Revision を明示的に削除する機能として別途仕様を定義する。

なお、Reference を失った内部 cache object の削除は `sloop gc` の責務とする。

---

# 26. Garbage Collection

Reference を失った内部 object は、

```console
$ sloop gc
```

で削除できる。

`gc` は意味のある Recorded Revision を削除してはならない。

---

# 27. SQLite の役割

SQLite は以下に使用する。

- Working State
- Specification metadata index
- Revision index
- Search index
- Git Relation index
- Reference index
- Review Result
- Agent Run
- Cache

Recorded Revision の canonical data は immutable object として保存する。

SQLite database を Specification History の唯一の正本としてはならない。

SQLite index は、immutable object store から再構築可能でなければならない。

```console
$ sloop index rebuild
```

で再構築できるものとする。ただし、その Object Store が端末間でどう交換されるかは未定とする。ローカル永続化方式と分散 transport は分離できる

---

# 28. Content-addressed Object Store

Recorded Revision は content-addressed object として保存する。

概念的には、

```text
objects/
└── sha256/
    ├── a1/
    │   └── b2c3...
    └── f4/
        └── e5d6...
```

のように管理する。

ユーザはこの object store を直接編集する必要がない。

Object の Identity は SHA-256 とする。

---

# 33. Specification Reference

Specification は Repository 内のコードやテストを明示的に Reference として保持できる。

Reference Kind は v0.1 では以下とする。

```text
code
test
context
```

Reference は Specification の意味内容の一部として扱う。

Reference の追加・削除は Specification Working State の変更として扱い、それだけを理由に Recorded Revision を自動生成する必要はない。

ただし Recorded Revision を生成する場合、その時点の Reference 一覧を Revision snapshot に含めなければならない。

一度生成された Recorded Revision の Reference を後から変更してはならない。

Reference の追加・削除により意味内容が変化した場合、Working State の Status は `DRAFT` に戻る。

## 33.1 Reference の追加

```console
$ sloop ref add 1 \
    --kind code \
    src/parser/parser.go:20-80
```

テストの場合:

```console
$ sloop ref add 1 \
    --kind test \
    internal/parser/parser_test.go:10-75
```

ファイル全体も指定可能とする。

```console
$ sloop ref add 1 \
    --kind context \
    go.mod
```

## 33.2 Commit を基準とした Reference

Line Range の意味を固定するため、Git Commit を指定できる。

```console
$ sloop ref add 1 \
    --kind test \
    internal/parser/parser_test.go:10-75 \
    --at a1b2c3d4
```

`--at` を省略した場合は現在の Working Tree に対する Reference とする。

## 33.3 Reference の表示

```console
$ sloop ref list 1
```

## 33.4 Reference の削除

```console
$ sloop ref remove 1 <reference-id>
```

---

# 34. List

Specification を一覧表示できる。

```console
$ sloop list
Project: sloop <sloop-2deb2e1c-925f-40df-a9b0-4e7ac25b124d>

ID         Title                                Status   Last Updated   Last Author
[sloop-1]  sloop での Markdown 仕様書について    DRAFT    4 seconds ago  m-tsuru
```

デフォルトでは `Last Updated` 降順とする。

---

# 35. Structured Output

以下をサポートする。

```console
$ sloop list --json
$ sloop list --csv
```

`--json` と `--csv` は同時指定不可とする。

Machine-readable output を指定した場合、装飾や human readable header を含めてはならない。

---

# 36. Filter

```console
$ sloop list --filter="status:draft"
```

```console
$ sloop list --filter="title:Markdown"
```

複数指定可能とする。

同じ key の filter は OR とする。

```console
$ sloop list \
    --filter="status:draft" \
    --filter="status:ready"
```

は、

```text
status = DRAFT OR status = READY
```

とする。

異なる key は AND とする。

```console
$ sloop list \
    --filter="status:draft" \
    --filter="title:Markdown"
```

は、

```text
status = DRAFT AND title contains "Markdown"
```

とする。

Status は case-insensitive とする。

Title は substring match とする。

---

# 37. `sloop view`

特定 Specification を標準出力へ Markdown として出力する。

```console
$ sloop view 1
```

出力は piping 可能な plain text とする。

診断メッセージは stderr に出力し、Specification 本文と混在させてはならない。

したがって、

```console
$ sloop view 1 | pbcopy
```

や、

```console
$ sloop view 1 | another-command
```

を利用できる。

---

# 38. View Front Matter

`sloop view` の出力には編集時には存在しない履歴情報を追加する。

例:

```yaml
---
project: "sloop-2deb2e1c-925f-40df-a9b0-4e7ac25b124d"
id: "sloop-1"
title: "sloop での Markdown 仕様書について"
status: READY
revision: 2
revision-hash: e5f6a7b8
parents: []

log:
  - author:
      name: m-tsuru
      email: tsuru@example.com
      agent: false
    revision-hash: a1b2c3d4
    status: DRAFT
    date: 2026-09-07T21:29:47+09:00

  - author:
      name: m-tsuru
      email: tsuru@example.com
      agent: false
    revision-hash: e5f6a7b8
    status: READY
    date: 2026-09-07T21:45:03+09:00
---
```

この `log` は表示用 projection であり、ユーザが直接編集するデータではない。

---

# 39. Section Retrieval

ユーザ向けドキュメンテーションや Coding Agent から一部 Section だけ利用できるようにする。

```console
$ sloop view 1 --section goal
```

```console
$ sloop view 1 --section specification
```

```console
$ sloop view 1 --section acceptance-criteria
```

未知の Section ID も、

```console
$ sloop view 1 --section user-interface
```

のように指定可能とする。

---

# 40. Query

複数 Specification から特定 Section を取得できる。

```console
$ sloop query \
    --section user-interface \
    --filter="status:completed"
```

これは Documentation Tool 等が Sloop の情報を再利用するためのインターフェースとして利用できる。

Sloop 自身は取得した内容からユーザ向け Documentation を自動生成しない。

---

# 41. Coding Agent Context

Coding Agent 用には、

```console
$ sloop context 1
```

を提供する。

Coding Agent が実装に利用するデフォルトの Context は immutable Recorded Revision から生成する。

Context には最低限以下を含める。

```text
Specification ID
Revision Hash
Status
Goal
Specification
Acceptance Criteria
Explicit References
Git Relations
Review Policy
```

Machine-readable output として、

```console
$ sloop context 1 --json
```

を提供する。

現在の Working State に未記録の変更が存在し、対応する Recorded Revision が存在しない場合、デフォルトの `sloop context` は失敗する。

例:

```text
Error: sloop-1 has unrecorded working changes.
Record or mark the specification READY before generating an agent context.
```

`READY` または `FORCEREADY` へ Status Transition する場合、その時点の Working State が未記録であれば自動的に Recorded Revision を生成する。

したがって通常は、

```console
$ sloop ready 1
$ sloop context 1
```

の順で利用できる。

DRAFT の Working State を確認・レビュー目的で出力したい場合は、

```console
$ sloop context 1 --working
```

を使用できる。

`--working` で生成された Context は immutable Revision に基づく Context ではないことを Machine-readable output に明示しなければならない。

例:

```json
{
  "recorded": false,
  "status": "DRAFT"
}
```

`--working` Context を Coding Agent の正式な Implementation Input として扱うことは推奨しない。

Sloop は無関係な他 Specification をデフォルト Context に含めてはならない。

親 Specification 等の関連情報は明示的な relation に基づき必要なもののみ含める。

---

# 42. Specification Review

Coding Agent は `READY` Specification を実装する前に、仕様不足を確認できる。

Review Result は以下とする。

```text
ACCEPTED
REJECTED
```

REJECTED は Specification Status ではない。

---

# 43. REJECTED Review

Coding Agent が以下を発見した場合、REJECTED を返すことができる。

```text
仕様から実装時の外部観測可能な挙動を一意に決定できない
```

REJECT は最低限以下を含まなければならない。

```text
問題となる仕様箇所
未定義の判断事項
その判断が実装へ与える影響
```

可能な選択肢を提示してもよい。

ただし、Coding Agent 自身がその選択肢を採用して実装を続行してはならない。

---

# 44. Review Result の記録

Review Result を記録する CLI を提供する。

```console
$ sloop reject 1 \
    --reason "Deletion semantics are not defined." \
    --author.name coding-agent \
    --author.agent true
```

または、

```console
$ sloop accept 1 \
    --author.name coding-agent \
    --author.agent true
```

Review Result は Specification 全体ではなく、Agent が判断対象とした特定の Recorded Revision に対する判断として記録する。

概念上、

```yaml
review:
  specification: sloop-1
  revision: a1b2c3d4
  result: REJECTED
  reason: >
    Deletion semantics are not defined.
```

とする。

`sloop reject` は Specification Status を `REJECTED` に変更しない。

Specification は `READY` のままとする。

ユーザが REJECT の内容を受けて Specification を編集した場合は、その Working State の Status を自動的に `DRAFT` とする。

新しい Recorded Revision が生成された場合、以前の Review Result は履歴として保持するが、新 Revision に対する Review Result として扱ってはならない。

---

# 45. FORCEREADY Review

Specification Status が `FORCEREADY` の場合、

```console
$ sloop reject 1 ...
```

は失敗しなければならない。

例:

```text
Error: specification sloop-1 is FORCEREADY and cannot be rejected for specification ambiguity.
```

Coding Agent は実装を続行する。

ただし、tool failure、build failure、resource failure 等の仕様不足とは異なる問題は Agent Run の `FAILED` として記録できる。

---

# 46. Agent Run

Coding Agent の一回の実行を Agent Run として記録できる。

Agent Run は、Agent が実際に入力として利用した Recorded Revision に必ず紐づける。

概念上、

```yaml
agent-run:
  specification: sloop-1
  revision: a1b2c3d4
  result: IMPLEMENTED
```

とする。

Agent Run Result は最低限以下とする。

```text
IMPLEMENTED
FAILED
```

Review Result とは独立する。

概念上、

```text
READY
  ↓
Review
  ├── REJECTED
  └── ACCEPTED
         ↓
       Agent Run
         ├── IMPLEMENTED
         └── FAILED
```

となる。

FORCEREADY の場合:

```text
FORCEREADY
    ↓
Agent Run
  ├── IMPLEMENTED
  └── FAILED
```

となる。

新しい Recorded Revision が生成された場合、旧 Revision に紐づく Agent Run は履歴として保持するが、新 Revision に対する実行結果として扱ってはならない。

---

# 47. Git Commit Integration

Git への過度な Integration は行わない。

通常の commit 操作はユーザまたは Coding Agent が通常通り行う。

Sloop は既存 Commit と Specification Revision の relation を記録する。

---

# 48. `commit-record`

```console
$ sloop commit-record <specification> <commit>
```

alias として、

```console
$ sloop cr <specification> <commit>
```

を提供する。

例:

```console
$ sloop cr 1 a1b2c3d4
```

複数指定可能とする。

```console
$ sloop cr 1,2 a1b2c3d4,a1b2c3d5
```

複数 Specification と複数 Commit を同時指定した場合は、指定された組み合わせすべてについて relation を作成する。

---

# 49. Commit Record と Revision

`commit-record` 実行時、現在の Working State が未記録であれば自動的に Recorded Revision を作成する。

Git Commit は必ず immutable Recorded Revision と関連付ける。

Working State そのものとは関連付けない。

---

# 50. Git Notes

Git Commit 側から Sloop relation を参照できるよう、

```text
refs/notes/sloop
```

を使用する。

Git Note は機械可読な YAML とする。

例:

```yaml
sloop:
  - id: sloop-1
    revision-hash: a1b2c3d4
    relation: implementation
```

一つの Commit に複数 Specification を関連付けることができる。

---

# 51. Git Log

専用 notes ref であるため、

```console
$ git log --show-notes=sloop
```

で表示可能とする。

必要であればユーザは Git repository local config に、

```console
$ git config notes.displayRef refs/notes/sloop
```

を設定できる。

Sloop はこの設定を必須とはしない。

---

# 52. Sloop 側の Git Relation

// 将来仕様

---

# 53. Source of Truth

Sloop におけるデータの分類は以下とする。

```text
Immutable Specification Objects = authoritative
SQLite = local working state / index
Git Notes = optional Git relation metadata
```

Generated Agent Context を正本として保存してはならない。

---

# 54. Cache

以下は再生成可能な Cache として扱う。

- full-text search index
- Vector index
- Embedding
- Repository analysis cache
- generated context cache

これらは Specification Revision と同じ永続性を要求しない。

---

# 55. Documentation Reuse

Sloop に保存された Specification は、ユーザ向け Documentation を作成する際の情報源として利用できる。

ただし、

```text
Sloop Specification
        ↓
Documentation Generator
```

を Sloop Core の責務としてはならない。

代わりに、

```text
sloop view
sloop query
sloop context
```

を通じて必要な情報を取得可能にする。

これにより外部 LLM や Documentation Tool が必要な情報のみ参照できる。

---

# 56. Config

`.sloop/config.yaml` には最低限以下を保存する。

例:

```yaml
project:
  id: sloop-2deb2e1c-925f-40df-a9b0-4e7ac25b124d
  slug: sloop
  spec-prefix: sloop

repository:
  path: .

git:
  notes-ref: refs/notes/sloop
```

ローカル端末固有情報を共有 config に保存してはならない。

---

# 57. Connector Architecture

v0.1 の Sloop Core は特定 Vendor Connector を実装必須とはしない。

Core は最低限、

```text
CLI
Markdown stdout
JSON stdout
```

を安定した interface として提供する。

将来的に、

```text
ChatGPT Connector
Claude Connector
GitHub Copilot Connector
MCP Server
```

を Core の上に実装可能とする。

Connector は Specification の正本を独自に保持してはならない。

---

# 58. Context Locality

Sloop の Coding Agent integration は、対象 Specification に必要な情報のみを Agent へ渡すことを原則とする。

デフォルトで以下を無条件に全文投入してはならない。

- 他の全 Specification
- Repository 全体
- `/docs` 全体
- `AGENTS.md` 全体
- 過去の全 Agent Run
- 全 Revision history

対象 Specification、明示 References、必要な relation を優先する。

---

# 60. Non-Goals

Sloop v0.1 は以下を目的としない。

- Coding Agent 自体を実装すること
- 特定 LLM API を必須にすること
- GitHub Issues を置き換えること
- Project Management Tool になること
- User Documentation を自動生成すること
- Repository 全体を毎回 LLM Context に投入すること
- 仕様不足を Coding Agent が自動的に補完すること

---

# 61. 重要な不変条件

実装は以下を常に満たさなければならない。

```text
1. Recorded Revision は immutable である。

2. Revision Hash は content-addressed identity であり、永続形式では Full SHA-256 を使用する。

3. Specification は UUID v4 による永続 Identity を持ち、表示用 Specification ID と区別する。

4. SQLite が失われても Recorded Revision から index を再構築できる。

5. Specification は通常の source working tree に Markdown として保存されない。

6. Coding Agent Context は Specification の正本ではない。

7. READY の最終決定主体はユーザである。

8. REJECTED は Specification Status ではなく Review Result である。

9. READY の Agent は仕様不足を推測して補完してはならない。

10. FORCEREADY の Agent は仕様不足を理由に REJECT してはならない。

11. COMPLETED の最終決定主体はユーザである。

12. Git Commit は mutable Working State ではなく Recorded Revision に関連付ける。

13. Agent の Status Transition、Review Result、Agent Run は判断対象となった Recorded Revision に紐づける。

14. Agent は Markdown Front Matter の編集によって Status Transition の制約を迂回してはならない。

15. Specification の意味内容を編集した Working State は DRAFT に戻る。

16. v0.1 の通常操作では Specification History を単一 Head の線形履歴として扱う。

17. 将来の分散同期時の履歴分岐を Sloop が勝手に解決してはならない。

18. Network connection がなくても仕様の作成・編集・参照・Revision 管理が可能である。

19. Core functionality は特定 LLM vendor に依存してはならない。
```

---

# 59. 将来実装

以下は v0.1 の必須機能ではない。

## CI Integration

CI 上で Sloop の Specification Object / Revision の変更を検出する。

特定 Revision と Branch / Commit の関係から、main branch への Pull Request 作成等へ連携できるようにする。

SQLite database 自体の変更を CI の同期単位として扱ってはならない。

## Vendor Connector

ChatGPT、Claude Code、GitHub Copilot 等から Sloop を直接参照できる Connector を提供する。

MCP Server を利用してもよい。

## Semantic Search

簡易的な Vector Index / Embedding Search を追加できるようにする。

Vector Search は Sloop Specification の正本ではなく、検索用の派生 Index とする。

## Repository Analyzer

将来的には Source Code の symbol、AST、依存関係等を解析し、明示 Reference と組み合わせて Coding Agent の Context を構築できるようにする。


## Revision Branching / Divergence

Explicit Branching、Multiple Head、Diverged History、Revision Pruning は v0.1 では実装しない。

これらは分散同期および merge / resolution の仕様と合わせて将来バージョンで定義する。

## 分散・同期に関する v0.1 の扱い

Sloop は、将来的に Git と同様の分散型仕様管理を実現することを目標とする。

ただし v0.1 では、複数 Sloop Client 間で Specification Object を同期するための remote、origin、push、pull、sync の仕様を定義・実装しない。

v0.1 における Specification の正本は、各 Client のローカルに存在する Sloop Object Store とする。

Local Sloop Client

Working State
      ↓
Recorded Revision
      ↓
Sloop Object Store

SQLite は引き続き Working State および索引用データベースとして利用する。

### 将来の分散同期

将来バージョンでは、以下を別仕様として定義する。

Sloop Remote の概念
Remote の発見・登録方法
origin に相当する default remote
Specification Object の交換形式
Specification Head の交換方法
push
pull
sync
divergent history の検出
merge / resolution
authentication
Git repository を transport として利用する場合の Git ref
Git を利用しない transport
removable media 等による offline object exchange

したがって、v0.1 では以下のコマンドを実装しない。

sloop push
sloop pull
sloop sync

また、以下のような Sloop Specification 同期用 Git ref についても v0.1 の仕様から除外する。

refs/sloop/<project-id>

Git Notes による「Git Commit と Specification Revision の関連付け」は、Specification 自体の同期とは独立した機能として v0.1 に残すことができる。

Sloop Remote の設計上の前提

将来の Remote 実装では、特定の中央サーバを必須としてはならない。

Sloop Object は、特定サービスのアカウントを持たなくても交換可能であることを目標とする。

概念上、

Client A
   │
   │ Sloop Objects
   ▼
Transport
   │
   ▼
Client B

として扱い、Transport と Specification Storage を分離する。

Transport の候補には以下がある。

Git remote
filesystem
SSH
removable media
Sloop server

これらの具体的な仕様は v0.1 では決定しない。
