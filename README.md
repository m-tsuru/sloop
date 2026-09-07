<div align="center">
    <h1>Sloop</h1>
</div>

## What is this?

Sloop は実装意図をソースコードから独立して管理し、コーディング担当「者」に関連する仕様情報のみを展開するための実装ツールです。

Sloop stores implementation intent independently from source code and exposes only the relevant specification to coding agents.

## Purpose

以下の課題感を解決するつもりで作っています。

- 私が仕様を書き、仕様とテストのコード範囲（かコミットハッシュ）を明確に紐づけて、それを元にコーディングエージェントに書かせたい
- 指示仕様は私が書く環境にも AI が茶々を入れる形にもできるようにしたい。コーディングエージェントが実装時に困るような行間を見つけた時は、補完せずに REJECT し明確に指摘してほしい
- Claude と ChatGPT、GitHub Copilot のうちどれか一つだけにベンダーロックインしたくない
- リポジトリの /docs や AGENTS.md に LLM に与える仕様書を書く方式だと、そのタスクの本題ではない部分を読みにいくことになって、コンテキスト・トークン使用量が増えるのではないか
- すべてを CI に載せたり、GitHub Issues や Notion でそれを管理する方法はワークフロー的に良さそうに見えるが、アカウント BAN やデータの可搬性的にあまりうれしくない
- ユーザに提供する成果物のドキュメンテーションを LLM に投げた仕様からいい感じに活用したい。生成したいわけではない。
- Git のような分散型でありたい（インターネットがない場所でも安定して書きたい）

## Usage

全てのドキュメントは、以下のページにあります。

https://sloop-docs.s8i.jp.eu.org/
