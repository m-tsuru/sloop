# Sloop 日本語ドキュメント

Astro Starlight で生成し、Cloudflare Workers Static Assets で公開するドキュメントサイトです。

## 開発

Node.js 24（最低 22.12.0）と npm を使用します。

```sh
cd docs
npm ci
npm run dev
```

- `npm run check`：Astro / TypeScript の検証
- `npm run build`：静的サイトと検索インデックスを `dist/` に生成
- `npm run preview`：ビルド済みサイトのプレビュー
- `npm run preview:workers`：Workers のローカル配信で確認
- `npm run deploy:check`：ビルドと Wrangler dry-run（公開なし）
- `npm run deploy`：ビルドして Cloudflare Workers に公開

## 公開

`wrangler.jsonc` の Worker 名を確認し、`npx wrangler login` 後に `npm run deploy` を実行します。公開先が決まったら `SITE_URL` に実際のオリジンを指定してビルドしてください。SSR アダプターや Worker スクリプトは不要です。

Workers Builds のルートは `docs`、ビルドコマンドは `npm ci && npm run check && npm run build`、デプロイコマンドは `npx wrangler deploy` です。

詳細は [公開手順](src/content/docs/guides/deployment.md) に記載しています。

## 編集

- 本文：`src/content/docs/`
- ナビゲーション・日本語設定：`astro.config.mjs`
- テーマ：`src/styles/custom.css`
- 配信設定：`wrangler.jsonc`

Getting Started、Tutorial、CLI の全サブコマンドと alias、config、Markdown、Agent 連携、バックアップ、トラブルシューティングを収録しています。説明の基準は `../internal/cli/` と `../internal/sloop/` の実装です。将来構想を実装済み機能として記載しないでください。
