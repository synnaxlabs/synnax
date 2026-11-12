# Synnax docs site

The documentation site at [docs.synnaxlabs.com](https://docs.synnaxlabs.com), built with
Astro and served by Vercel. Pages render at build time. The 404 page and the updater
routes render per request.

## Commands

```bash
pnpm dev                                 # dev server
pnpm build                               # astro check + astro build
pnpm test                                # vitest
pnpm check-types                         # astro check
pnpm lint                                # eslint
pnpm check-site:build && pnpm check-site # static build + link and media crawler
```

`pnpm check-site` fails on a broken internal link. A broken external link is a warning,
because the cause can be the host. With `--strict`, which the nightly run and the CI of
a release set, a broken external link fails too.

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
that tag. 0.60 and 0.61 shipped while `main` held the docs, so they build from
`release/docs-0.60` and `release/docs-0.61`. 0.60 starts at the `main` of 2026-11-08,
and 0.61 at the `core/v0.61.0` tag. Set `ref` to build one version from another ref. A
push to `main` deploys `next`. A stable release deploys `next`, `latest`, `previous`,
and the train it released. An older site keeps the download links of its last deploy.

To fix the docs of a released train, merge the fix into `main`, cherry-pick it onto the
train's release branch (`release/docs-X.Y` for 0.60 and 0.61), and deploy the train. If
a patch of the train shipped from `main`, first merge its tag into the branch. A 0.60 or
0.61 release deploys its `release/docs-X.Y` branch as it is, so pick the docs of a patch
onto that branch before the release.
