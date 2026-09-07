---
title: "Cloudflare Workers への公開"
description: "Starlight の静的ドキュメントを Workers Static Assets でビルド・確認・公開する。"
---

このドキュメントサイトは Astro Starlight で静的 HTML を生成し、Cloudflare Workers の Static Assets で配信します。[Starlight の構成](https://starlight.astro.build/manual-setup/)と[Workers Static Assets](https://developers.cloudflare.com/workers/static-assets/get-started/)に対応した設定を `docs/` に含めています。

## ローカル開発

Node.js 22.12.0 以降と npm を使用します。リポジトリの `.node-version`（`docs/` 内）は Node.js 24 を指定しています。

```sh
cd docs
npm ci
npm run dev
```

ターミナルに表示された URL を開きます。本文は `src/content/docs/` の Markdown、メニューや日本語設定は `astro.config.mjs` にあります。

```sh
npm run check
npm run build
npm run preview
```

`dist/` に HTML・CSS・JavaScript・Pagefind の検索インデックスを生成します。検索は本番ビルドのプレビューで確認してください。

## Workers の設定

`wrangler.jsonc` は次の構成です。

```json
{
  "$schema": "node_modules/wrangler/config-schema.json",
  "name": "sloop-docs",
  "compatibility_date": "2026-09-08",
  "assets": {
    "directory": "./dist",
    "html_handling": "force-trailing-slash",
    "not_found_handling": "404-page"
  }
}
```

静的ファイルのみを配信するので Worker の `main` や SSR アダプターは不要です。末尾スラッシュを Astro の生成形式と揃え、存在しない URL は 404 ページにします。[HTML の配信規則](https://developers.cloudflare.com/workers/static-assets/routing/advanced/html-handling/)も参照してください。

Worker 名 `sloop-docs` は公開先アカウントで使用する名前です。別の名前を使う場合はこの値を変更します。

## 公開前に確認する

```sh
npm run deploy:check
npm run preview:workers
```

`deploy:check` はビルドと Wrangler の dry-run を行い、公開しません。`preview:workers` はビルド後に Workers のローカル開発サーバーを起動します。表示された URL で本文・検索・存在しないパスを確認し、終了するには Ctrl+C を押します。

## Cloudflare に公開する

Cloudflare アカウントにログインし、公開先の Worker 名を確認します。

```sh
npx wrangler login
npm run deploy
```

`deploy` は毎回ビルドしてから Wrangler で公開します。初回公開で表示された `workers.dev` の URL、または設定済みのカスタムドメインを `SITE_URL` に設定して再ビルド・公開すると、canonical URL 等に公開先を反映できます。

```sh
SITE_URL=https://sloop-docs.your-subdomain.workers.dev npm run deploy
```

上記のホスト名は例です。実際の URL に置き換えてください。ローカル開発では `SITE_URL` は省略できます。必要なら `docs/.env.example` を参考に `docs/.env` を作ります。

## Workers Builds から公開する

Cloudflare の Git 連携を利用する場合、ビルド対象をこのリポジトリの `docs` ディレクトリに設定します。

| 項目 | 設定 |
| --- | --- |
| ルートディレクトリ | `docs` |
| ビルドコマンド | `npm ci && npm run check && npm run build` |
| デプロイコマンド | `npx wrangler deploy` |
| Node.js | 24（少なくとも 22.12.0 以降） |
| ビルド環境変数 | 公開先が確定したら `SITE_URL` |

Worker 名と `wrangler.jsonc` の `name` を揃えます。外部 CI から Wrangler を利用する場合は Cloudflare の認証用環境変数を CI の秘密情報として設定し、リポジトリに認証情報を書き込まないでください。[Workers Builds の設定](https://developers.cloudflare.com/workers/ci-cd/builds/configuration/)を参照してください。

## ドキュメントを更新する

Markdown の Front Matter に `title` と `description` を付け、CLI ページは `sidebar.order` で順序を指定します。内部リンクは `/cli/context/` のようなサイトのパスを使います。CLI を変更したときは、引数・フラグ・既定値・エラー条件・具体例を一緒に更新してください。
