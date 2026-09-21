# 58 Release workflow

- **Author**: Patrick Dotson
- **Date**: 2026-09-21
- **Related**:
  [RFC 0020 - Engineering process standardization](0020-engineering-workflow.md)

## 0 Summary

The Core, Console, and Driver release on every push to `main` or `rc`, versions live in
fourteen files, and the Console updater polls a JSON file a bot force-pushes to `main`.
This RFC moves to trunk-based development on `main`, one dispatched release workflow per
product, Git tags as the only version source, an updater manifest hosted on the GitHub
release, and static feature flags. It supersedes RFC 0020 §2, §3, §5, and §7, and
retires the `rc.md` checklist that §6 links.

## 1 Motivation

- **`rc` is a second mainline**: 76 commits ahead of `main`, 26 open PRs, sixteen
  back-merges from `main` in July and August 2026.
- **Release is a side effect of a merge**: `deploy.synnax.yaml:12-89` publishes on any
  push to `main` or `rc` that touches a product path. Hotfix branches cannot release.
- **Versions are hand-maintained**: `core/pkg/version/VERSION`, `tauri.conf.json`, eight
  `package.json`, four `pyproject.toml`, checked for major.minor agreement only.
- **`console/release-spec.json` is a production endpoint**: force-pushed to `main` by CI
  and polled every 30 s by every shipped Console.
- **Docs links go stale**: `fetchVersion.ts:33` memoizes the releases lookup for the
  life of a warm Vercel function.
- **Nothing can ship dark**: the Console, Core, and docs site have no feature flags.

## 2 Design

Five products release separately: Console, Core, Driver, Python, TypeScript. Compatible
products share a major.minor, the train, such as `0.59`. `main` is always releasable:
unfinished work ships dark behind a flag. Nothing publishes on push; a person dispatches
every release. No file in Git carries a product version; the tag is the version.

### 2.0 Branching

`main` is the only long-lived branch and every PR targets it. `rc`, the `rc.md` PR
template, and every `branches: [main, rc]` trigger are deleted. Pushes to `main` and
`release/**` build and test only.

A hotfix is fixed on `main` first, then cherry-picked by PR onto
`release/<product>-X.Y`, cut from the product's latest stable tag. The release workflow
runs from that branch and computes the next patch from the tags reachable from it. The
branch is deleted afterwards.

### 2.1 Release workflows

One `workflow_dispatch` workflow per product, `release.<product>.yaml`, with inputs:

- **`bump`**: `patch` (default) or `minor`.
- **`prerelease`**: Tag `-rc.N` and mark the release pre-release.
- **`console_version`, `driver_version`** (Core only): Releases to embed. Default: the
  newest stable, or the newest pre-release when `prerelease` is set.

Each runs five stages:

1. **Resolve**: A composite action `.github/actions/resolve-version` reads the tags with
   the product prefix reachable from `HEAD`, applies `bump` and `-rc.N`, and enforces
   the train rule (§2.2).
2. **Verify**: The commit's required checks must have passed. The integration suite runs
   as a `workflow_call` job.
3. **Build**: `build.synnax.yaml` with only that product enabled and `version` passed
   through, signed. Python and TypeScript inject the version and build in place.
4. **Publish**: Draft release under the tag, upload assets, clear the draft. The release
   creates the tag, so a failed build leaves none. Concurrency group `release-<product>`
   serializes a product's releases.
5. **Notes**: `generate_release_notes` with categories from `.github/release.yml`.

Assets per product:

- **Console** (`console/vX.Y.Z`): `console-web-vX.Y.Z.tar.gz`, the DMG, MSI, and NSIS
  installers, their `.sig` files, and `latest.json`.
- **Driver** (`driver/vX.Y.Z`): The four platform binaries and the NI install script.
- **Core** (`core/vX.Y.Z`): Downloads the Console bundle and Driver binaries from their
  releases into `core/pkg/console/dist/` and `core/pkg/driver/assets/`, then builds the
  binaries, Windows installer, and Docker image (`latest` for stable, `next` for
  pre-release). No workflow rebuilds another product. The notes name the embedded
  versions.
- **Python** (`py/vX.Y.Z`), **TypeScript** (`ts/vX.Y.Z`): Version injection, then the
  existing `uv publish` and `pnpm publish -r`, moved to OIDC trusted publishing.

`deploy.synnax.yaml`, `deploy.ts.yaml`, and `deploy.py.yaml` are deleted.

### 2.2 Versions

Train rule: a product may bump to at most one minor ahead of the Core's latest stable,
and the Core release fails unless the Console and Driver it embeds share its minor.
Console and Driver open a train; the Core closes it.

Every manifest carries `0.0.0` and the build injects the resolved `version`:

- **Core**: The existing `-ldflags -X` (`build.synnax.yaml:621-626`). The `VERSION` file
  and `//go:embed` fallback in `get.go` are deleted; `Prod()` returns `0.0.0-dev` when
  unset.
- **Driver**: Bazel `--stamp` with a `--workspace_status_command` emitting
  `STABLE_SYNNAX_VERSION`; the `version.h` genrule reads `stable-status.txt` and takes
  the timestamp from `volatile-status.txt`.
- **Console**: `tauri build --config '{"version":"X.Y.Z"}'`.
- **TypeScript**: `pnpm -r exec npm version X.Y.Z --no-git-tag-version`. Internal deps
  already resolve to `workspace:^`, which pnpm rewrites at publish.
- **Python**: `uv version X.Y.Z` per package before `pin_internal_deps.sh`.

`check_versions.sh`, `bump_versions.sh`, and `test.updates.yaml` are deleted.

### 2.3 Updater manifest and docs site

`release-spec.json`, the `publish-console-update` job, and the `rc` endpoint rewrite in
`build.synnax.yaml:891-897` are deleted. `latest.json` lives on the Console release.

The docs site gains two Astro endpoints, `/releases/console/latest.json` and
`/releases/console/next.json`, that 302 to the manifest of the newest stable or
pre-release Console, with `s-maxage` so the Vercel CDN absorbs the polling. Stable
builds point at the first; pre-release builds get the second through `--config`, so a QA
machine follows candidates until it installs a stable build.

`fetchVersion.ts` becomes `releases.ts`: `latest(product)` queries the releases API with
a server-side token, filters by tag prefix, skips drafts and pre-releases, and caches
for a few minutes. Every download component passes its product; the header shows the
Core's train. The site is server-rendered, so links flip within the cache window with no
deploy hook. The docs site reads the release, never the repo.

### 2.4 Feature flags

A flag is a static build-time boolean, default off, that hides unfinished work in
production. One registry per surface names each flag's owner and the release that
removes it:

- **Console**: `console/src/flags.ts` builds `const FLAGS` from `VITE_FLAG_*` through a
  Vite `define`, so production tree-shakes dark code. `IS_DEV` folds in.
- **Docs site**: `docs/site/src/flags.ts` reads `FLAG_*` through `astro:env`. Pages opt
  in with `flag` frontmatter (404 when off); `PageNavNode` gains `flag`. Preview deploys
  set every flag on.

### 2.5 Out of scope

Release cadence and QA procedure (the `rc.md` checklist becomes a release issue
template), migration testing between trains, and any future desktop product beyond its
flag and workflow file.

## 3 Implementation phases

- **Phase 1: Docs site.** `releases.ts`, the updater routes, per-product consumers, docs
  flags. Works with today's `synnax-v` tags and deploys before any workflow change.
- **Phase 2: Cutover.** One PR: the five `release.*.yaml`, `resolve-version`,
  `.github/release.yml`, `deploy.*` deleted, `rc` removed from every trigger, the
  updater endpoint swapped, `CLAUDE.md` rewritten. Version files stay and injection
  overrides them, so the PR is workflow-only. Then merge `rc` into `main` (publishes
  nothing), `gh pr edit --base main` for open PRs, delete `rc`.
- **Phase 3: Versions and Console flags.** Delete the version literals, scripts, and
  `test.updates.yaml`; add the Bazel status script and `console/src/flags.ts`.
- **Phase 4: First releases.** Console and Driver with `bump: minor`, then the Core,
  opening train 0.59. After the Console release, one manual commit copies its
  `latest.json` into `release-spec.json`, so installed 0.58 builds upgrade once and land
  on the new endpoint. The file stays on `main` until 0.58 is out of support.

## 4 Resolved decisions

1. **Shared minor, independent patch**: Independent semver rejected; users could not
   answer cross-product compatibility questions.
2. **Candidate channel is a release property**: A `prerelease` input, not a branch. QA
   needs signed candidates and the updater needs a channel.
3. **Three binary products, built once**: A single Core plus Driver release rejected; a
   Go-only change must not re-release the Driver or Console.
4. **No approval gate**: Environment reviewers rejected; the dispatch is the approval.
5. **`bump` choice**: release-please rejected because it writes version files; typed
   versions rejected because they invite skipped numbers.
6. **Updater endpoint on the docs site**: `releases/latest/download` is repo-wide and
   breaks under per-product tags; a rolling `console/latest` release abuses tags.
7. **Request-time docs versions**: The site is already SSR; a deploy hook fixes a race
   that does not exist.
8. **Static flags per surface**: Version-gated docs pages rejected because unshipped
   features have no version; runtime flag services rejected because nothing needs a flip
   without a deploy.
9. **Cutover order**: Workflows first, then merge `rc`, then retarget and delete, so no
   PR pays a rebase and the first release ships a whole train.

## 5 Open questions

- Tag prefix separator: `console/v` or `console-v`.
- Cache TTLs on the docs site and updater routes.
- Whether the Driver release also publishes a Docker image.
- Which PR labels `.github/release.yml` categorizes on.

## 6 Prior art

- Fowler, branching patterns: https://martinfowler.com/articles/branching-patterns.html
- trunkbaseddevelopment.com, Branch for release:
  https://trunkbaseddevelopment.com/branch-for-release/
- Hodgson, Feature toggles: https://martinfowler.com/articles/feature-toggles.html
- Tauri v2 updater: https://v2.tauri.app/plugin/updater/
- grafana/shared-workflows, per-component tags in a polyglot monorepo:
  https://github.com/grafana/shared-workflows
- npm and PyPI trusted publishing: https://docs.npmjs.com/trusted-publishers/,
  https://docs.pypi.org/trusted-publishers/
