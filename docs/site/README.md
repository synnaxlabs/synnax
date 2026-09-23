# Synnax docs site

The documentation site at [docs.synnaxlabs.com](https://docs.synnaxlabs.com), built with
Astro and served by Vercel. Pages render on the server per request.

## Commands

```bash
pnpm dev                                 # dev server
pnpm build                               # astro check + astro build
pnpm test                                # vitest
pnpm check-types                         # astro check
pnpm lint                                # eslint
pnpm check-site:build && pnpm check-site # static build + link and media crawler
```

## Release lookups

Download links, the version in the header, and the Console updater routes under
`/releases/console/` read the per-product GitHub releases (`console/vX.Y.Z`,
`core/vX.Y.Z`, `driver/vX.Y.Z`) at request time through `src/util/releases.ts`. The
listing is cached for five minutes per warm function. `DOCS_GITHUB_TOKEN` in the Vercel
project raises the GitHub API rate limit; the site works without it.

## Feature flags

A flag is a static build-time boolean that hides unfinished docs in production. Flags
are off by default, and preview deploys turn every flag on so reviewers see dark work. A
flip is an environment variable plus a redeploy.

To add a flag named `example`:

1. Declare `FLAG_EXAMPLE` in the `env.schema` block of `astro.config.ts` as a public
   client boolean with `default: false`.
2. Register it in `src/flags.ts` as `example: flag(FLAG_EXAMPLE)`, with a comment naming
   the owner and the release that removes the flag.
3. Put `flag: "example"` in the frontmatter of each page behind it. The page returns 404
   while the flag is off. Give the page's `PageNavNode` the same `flag`, and the nav
   hides the node and its children while the flag is off.
4. To turn it on in production, set `FLAG_EXAMPLE=true` in the Vercel project's
   Production environment and redeploy. Set the same name as a repository variable in
   GitHub Actions, or the search index job drops the flagged pages.
5. When the work ships, delete the flag from all four places.

An unknown flag name throws at render time, so a typo cannot hide a page silently.
