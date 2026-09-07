---
title: "config.yaml と環境変数"
description: "共有設定の全フィールド、既定値、エディターと著者の解決規則。"
---

## ファイルの場所と探索

`sloop init` がリポジトリルートの `.sloop/config.yaml` を作成します。通常のコマンドは実行ディレクトリから親方向へ探索し、最初に見つかったファイルを使用します。`--config` やグローバル設定ファイルを指定する CLI は現在ありません。

## 完全な例

```yaml
project:
  id: myapp-2deb2e1c-925f-40df-a9b0-4e7ac25b124d
  slug: myapp
  spec-prefix: myapp
repository:
  path: .
git:
  notes-ref: refs/notes/sloop
```

`id` の UUID は例です。実際には `init` が生成した値を維持してください。

| キー | 型 | 必須・既定値 | 意味 |
| --- | --- | --- | --- |
| `project.id` | 文字列 | 必須・空不可 | `<slug>-<UUID v4>`。ローカルデータ保存先を識別 |
| `project.slug` | 文字列 | 必須・空不可 | 一覧などに表示するプロジェクト名 |
| `project.spec-prefix` | 文字列 | 必須・空不可。初期値は slug | 新しい仕様 ID の prefix と番号指定の解決に使用 |
| `repository.path` | 文字列 | 初期値 `.` | 共有設定に保存するリポジトリパス。現行 CLI では操作対象の切り替えには使われない |
| `git.notes-ref` | 文字列 | 未指定・空なら `refs/notes/sloop` | `commit-record` が読み書きする Git Notes ref |

### 変更時の挙動

`project.id` を変えると別のローカルデータディレクトリを参照するため、既存の仕様が一覧から見えなくなります。移行やリネームの手段として変更しないでください。

`spec-prefix` の変更は既存仕様の ID を改名しません。例えば `myapp` から `task` に変更すると、番号指定 `sloop view 1` は `task-1` を探します。既存の仕様には `sloop view myapp-1` のように完全な ID を指定します。採番はプロジェクト全体で続きます。

現在の `repository.path` は保存フィールドです。実際の Git 操作・テンプレート・Reference の基準は `.sloop/config.yaml` が見つかったディレクトリです。別リポジトリを操作するには、そのリポジトリへ移動してください。

`notes-ref` を変更しても、古い ref の Notes は移動しません。共有に利用する ref を決め、Git 側の表示・push・fetch も同じ ref に合わせます。

## 環境変数

| 名前 | 用途 |
| --- | --- |
| `SLOOP_EDITOR` | 優先するエディターコマンド。空なら `EDITOR` |
| `EDITOR` | `SLOOP_EDITOR` が空のときのエディター |
| `USER` | 著者名を明示せず Git にも名前がない場合の代替 |
| `XDG_DATA_HOME` | Linux のローカル保存先の基点。空なら `~/.local/share` |
| `HOME` | ホームディレクトリの解決に使用 |
| `TMPDIR` | 一時 Markdown の配置先に影響する OS の一時ディレクトリ設定 |

macOS の保存先は `~/Library/Application Support/sloop/<project-id>/` で、`XDG_DATA_HOME` は使いません。一時 Markdown をリポジトリ内に置かないよう、`TMPDIR` はリポジトリ外に設定してください。

```sh
export SLOOP_EDITOR='code --wait'
```

## 著者情報

`new`、`edit`、状態変更とその alias、`accept`、`reject`、`agent-run`、`ref add/remove`、`commit-record` で次を指定できます。

| フラグ | 既定・解決順 |
| --- | --- |
| `--author.name` | 指定値 → Git `user.name` → `USER` → `unknown` |
| `--author.email` | 指定値 → Git `user.email` → 空文字列 |
| `--author.agent` | 通常 `false`。`true` で Agent の操作として記録 |

名前・メールは独立して解決します。名前・メール・Agent フラグのどれかが異なると編集時の Author Boundary になります。`agent-run` はフラグにかかわらず Agent として記録します。

```sh
sloop edit 1 --author.name coding-agent --author.agent true
```

端末固有の保存先や認証情報を共有 config に追加しないでください。現行実装が参照するキーは上記のみです。
