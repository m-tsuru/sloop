import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: process.env.SITE_URL || undefined,
  output: 'static',
  trailingSlash: 'always',
  integrations: [
    starlight({
      title: 'Sloop',
      description: '仕様を起点に、コード・テスト・Git・Coding Agent をつなぐローカルファースト CLI。',
      defaultLocale: 'root',
      locales: { root: { label: '日本語', lang: 'ja' } },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/m-tsuru/sloop',
        },
      ],
      editLink: {
        baseUrl: 'https://github.com/m-tsuru/sloop/edit/main/docs/',
      },
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        {
          label: 'はじめに',
          items: [
            { label: 'Sloop とは', slug: '' },
            { label: 'Getting Started', slug: 'getting-started' },
            { label: 'Tutorial：仕様から実装へ', slug: 'tutorial' },
            { label: '基本概念と履歴', slug: 'guides/concepts' },
          ],
        },
        {
          label: 'CLI リファレンス',
          items: [
            {
              autogenerate: {
                directory: 'cli',
              },
            },
          ],
        },
        {
          label: '設定と運用',
          items: [
            { label: 'config.yaml と環境変数', slug: 'configuration' },
            { label: 'Markdown とテンプレート', slug: 'guides/markdown' },
            { label: 'Coding Agent との連携', slug: 'guides/agents' },
            { label: 'データ保存とバックアップ', slug: 'guides/storage' },
            { label: 'トラブルシューティング', slug: 'guides/troubleshooting' },
            { label: 'Cloudflare Workers への公開', slug: 'guides/deployment' },
          ],
        },
      ],
    }),
  ],
});
