# Synnax docs site

The documentation site at [docs.synnaxlabs.com](https://docs.synnaxlabs.com), built with
Astro and served by Vercel. Pages render at build time. The 404 page, the updater
routes, and pages behind a flag render per request.

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

Download links and the version in the header read the per-product GitHub releases
(`console/vX.Y.Z`, `core/vX.Y.Z`, `driver/vX.Y.Z`) at build time through
`src/util/releases.ts`, so a release shows on the site after the next deploy. The
release workflows redeploy the docs after each stable release. The updater routes under
`/releases/` read the releases per request, and a warm function caches the listing for
five minutes. `DOCS_GITHUB_TOKEN` in the Vercel project raises the GitHub API rate
limit; the site works without it.

## Versions

`DOCS_VERSION` names the version a build describes: a train such as `0.61`, or `next`
for `main`. Trains before 0.60 predate versioned docs and have no site. The newest train
with a stable Core release is the latest and lives at `docs.synnaxlabs.com`. An earlier
train lives at `v0-60.docs.synnaxlabs.com`, and `next` at `next.docs.synnaxlabs.com`.
Old and next builds show a banner, ask search engines not to index them, and send the
blog, release notes, reliability, and legal pages to the root. Download links keep to
the build's train. Each version has its own search index. A build without
`DOCS_VERSION`, such as a dev server or a preview, shows no menu and searches the index
of `next`. A site other than the newest names the root as Latest, not by its train,
because a newer train does not always rebuild it.

To see a version locally, run `DOCS_VERSION=0.60 pnpm dev`.

To deploy, run the `Deploy - Docs` workflow with one or more versions: a train such as
`0.61`, `next`, `latest`, or `previous`. `next` builds from `main`. A train builds from
its newest stable Core tag, or from its `release/core-X.Y` branch if the branch holds
that tag. Set `ref` to build one version from another ref. A push to `main` deploys
`next`. A stable release deploys `next`, `latest`, `previous`, and the train it
released. An older site keeps the download links of its last deploy.

To fix the docs of a released train, merge the fix into `main`, cherry-pick it onto the
train's release branch, and deploy the train. If a patch of the train shipped from
`main`, first merge its tag into the branch.

## Feature flags

A flag is a static build-time boolean that hides unfinished docs in production. Flags
are off by default, and preview deploys turn every flag on so reviewers see dark work. A
flip is an environment variable plus a redeploy.

To add a flag named `example`:

1. Declare `FLAG_EXAMPLE` in the `env.schema` block of `astro.config.ts` as a public
   client boolean with `default: false`.
2. Register it in `src/flags.ts` as `example: flag(FLAG_EXAMPLE)`, with a comment naming
   the owner and the release that removes the flag.
3. Put `flag: "example"` in the frontmatter of each page behind it. The page renders per
   request and returns 404 while the flag is off. Give the page's `PageNavNode` the same
   `flag`, and the nav hides the node and its children while the flag is off.
4. To turn it on in production, set `FLAG_EXAMPLE=true` in the Vercel project's
   Production environment and redeploy. Set the same name as a repository variable in
   GitHub Actions, or the search index job drops the flagged pages.
5. When the work ships, delete the flag from all four places.

An unknown flag name throws at render time, so a typo cannot hide a page silently.
