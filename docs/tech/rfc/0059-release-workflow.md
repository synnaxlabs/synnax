# 59 Release workflow

- **Author**: Patrick Dotson
- **Date**: 2026-09-21
- **Related**:
  [RFC 0020 - Engineering process standardization](0020-engineering-workflow.md)

## 0 Summary

Synnax releases fire on every push to `main` or `rc`, version numbers live in fourteen
files, the Console updater polls a JSON file that a bot force-pushes to `main`, and the
docs site races the release pipeline to learn which version to link. This RFC replaces
that with trunk-based development on `main`, one manually dispatched release workflow
per product, Git tags as the only version source, an updater manifest hosted on the
GitHub release and resolved through the docs site, request-time docs links, and static
feature flags in the Console and the docs site. It supersedes RFC 0020 §2, §3, and §5.

## 1 Motivation

- **`rc` is a second mainline**: 76 commits ahead of `main` with 26 open PRs, merged
  into `main` every two to three weeks after a 260-line manual checklist. The Git log
  shows ten `main` into `rc` back-merges in July and August 2026 alone.
- **Release is a side effect of a merge**: `deploy.synnax.yaml:12-89` publishes on any
  push to `main` or `rc` that touches a product path. A docs-only commit that brushes a
  shared path can re-release the Core.
- **Hotfixes cannot release from a branch**: `deploy.synnax.yaml:110-135` recognizes
  only `main` and `rc`, so `sy-release-0-58-2` produced nothing until its commits were
  landed on `main`.
- **Versions are hand-maintained**: `core/pkg/version/VERSION`, `tauri.conf.json`, eight
  `package.json` files, and four `pyproject.toml` files, checked for major.minor
  agreement only by `scripts/check_versions.sh:34-47`. The Console patch bump is a
  manual commit.
- **`console/release-spec.json` is a production endpoint**: `deploy.synnax.yaml:414-421`
  force-pushes it to `main`, and every shipped Console polls that raw URL every 30 s
  (`console/src/platform/version/Updater.tsx:24`).
- **Docs links go stale**: `docs/site/src/util/fetchVersion.ts:33` memoizes the GitHub
  releases lookup for the life of a warm Vercel function, so a release can stay
  invisible on the site until the function recycles.
- **Nothing can ship dark**: neither the Console, the Core, nor the docs site has a
  feature flag. A new desktop product cannot be developed on `main` without appearing in
  the next release.

## 2 Vocabulary

- **Product**: A separately released unit. Five today: Console, Core, Driver, the Python
  packages, and the TypeScript packages. A future desktop application is a sixth.
- **Train**: A shared major.minor across products, `0.59`. Products in one train are
  compatible with each other.
- **Release workflow**: A `workflow_dispatch` workflow in `.github/workflows/` named
  `release.<product>.yaml`.
- **Stable release**: A GitHub release with tag `<product>/vX.Y.Z`.
- **Pre-release**: A GitHub release with tag `<product>/vX.Y.Z-rc.N`, marked pre-release
  on GitHub, invisible to `/releases/latest` and to the docs site.
- **Updater manifest**: The Tauri `latest.json` document the Console polls.
- **Flag**: A static, build-time boolean that hides unfinished work in a production
  build.

## 3 Principles

1. **`main` is always releasable**: Every commit on `main` passes CI, and unfinished
   work ships dark behind a flag. There is no integration branch.
2. **A release is an intent, not a side effect**: Nothing publishes on push. A person
   dispatches a product's release workflow.
3. **The tag is the version**: No file in Git carries a product version. Every artifact
   learns its version at build time from the tag the workflow is about to create.
4. **Build once, embed everywhere**: The Core embeds the Console web bundle and the
   Driver binary from their published releases. No release workflow rebuilds another
   product.
5. **Products version independently within a train**: Patch numbers diverge freely; the
   minor is shared, and the Core release verifies the train.
6. **The docs site reads the release, never the repo**: Every version the site renders
   comes from the GitHub releases API at request time.

## 4 Design

### 4.0 Branching

`main` is the only long-lived branch. Feature and fix PRs target `main`. The `rc`
branch, the `rc.md` PR template, and every `branches: [main, rc]` trigger are deleted.

A hotfix follows the trunk-based shape: fix on `main` first, then cut
`release/<product>-X.Y` from the product's latest stable tag, cherry-pick, and dispatch
the product's release workflow from that branch. The workflow computes the next patch
from that branch's tag lineage, so the branch needs no special handling. The branch is
abandoned after the release; the next stable from `main` supersedes it.

### 4.1 Release workflows

Each product has one workflow, `release.<product>.yaml`, with these inputs:

| Input             | Type    | Products | Meaning                                         |
| ----------------- | ------- | -------- | ----------------------------------------------- |
| `bump`            | choice  | all      | `patch` (default) or `minor`                    |
| `prerelease`      | boolean | all      | Tag `-rc.N` and mark the release pre-release    |
| `console_version` | string  | core     | Web bundle to embed; default latest in train    |
| `driver_version`  | string  | core     | Driver binary to embed; default latest in train |

Every workflow runs the same five stages:

1. **Resolve**: A composite action `.github/actions/resolve-version` lists tags with the
   product prefix reachable from `HEAD`, applies `bump`, appends `-rc.N` when
   `prerelease` is set (`N` is one past the highest existing candidate for that
   version), and enforces the train rule of §4.2. It outputs `version` and `tag`.
2. **Verify**: The commit's required checks must have succeeded. The workflow queries
   the checks API for `HEAD` and fails if any required check is missing or failed, so a
   release never re-runs the unit suites that every push to `main` already ran. The
   integration suite, which runs only on PRs today, is called as a `workflow_call` job.
3. **Build**: A call into `build.synnax.yaml` with only that product's inputs enabled
   and `version` passed through. Signing runs for every release, stable or pre-release.
4. **Publish**: The release is created as a draft under `tag` at `HEAD`, assets are
   uploaded, and the draft flag is cleared last. The tag is created by the release, so a
   failed build leaves no tag behind. Concurrency group `release-<product>` with
   `cancel-in-progress: false` serializes releases of one product.
5. **Notes**: `generate_release_notes: true` with categories from `.github/release.yml`,
   which maps PR labels to sections and excludes bot authors.

Per product:

- **Console** (`console/vX.Y.Z`): Web bundle as `console-web-vX.Y.Z.tar.gz`, the Tauri
  DMG, MSI, and NSIS installers, their `.sig` files, and `latest.json`. The manifest is
  generated by the workflow from the `.sig` files, as `deploy.synnax.yaml:365-412` does
  today, with `linux-x86_64` omitted rather than faked.
- **Driver** (`driver/vX.Y.Z`): The Windows, macOS, Linux, and NI Linux RT binaries and
  the NI install script.
- **Core** (`core/vX.Y.Z`): Downloads `console-web-vX.Y.Z.tar.gz` and the Driver
  binaries from their releases into `core/pkg/console/dist/` and
  `core/pkg/driver/assets/`, then builds the Core binaries, the Windows installer, and
  the Docker image. Docker tags: the version, `latest` for stable, `next` for
  pre-release. The release notes name the embedded Console and Driver versions.
- **Python** (`py/vX.Y.Z`) and **TypeScript** (`ts/vX.Y.Z`): Version injection per §4.2,
  then the existing `uv publish` and `pnpm publish -r` steps. Both registries support
  OIDC trusted publishing; the workflows adopt it and the stored tokens are removed.

The push-triggered `deploy.synnax.yaml`, `deploy.ts.yaml`, and `deploy.py.yaml` are
deleted. `deploy.docs.yaml` keeps the Algolia reindex. Pushes to `main` build and test
only.

### 4.2 Versions

The train rule: a product's `minor` bump may put it at most one minor ahead of the
Core's latest stable, and the Core release fails if the Console or Driver it embeds is
not in its own minor. This lets Console and Driver open a train before the Core closes
it, and the Core release is the proof that the train is complete.

Version injection per product, all from the `version` output of §4.1 stage 1:

- **Core**: `-ldflags -X .../pkg/version.Version=` already exists
  (`build.synnax.yaml:621-626`). `core/pkg/version/VERSION` and the `//go:embed`
  fallback in `get.go:22-62` are deleted; `Prod()` returns `Version` or `0.0.0-dev`.
- **Driver**: Bazel `--stamp` with `--workspace_status_command=scripts/bazel_status.sh`
  emitting `STABLE_SYNNAX_VERSION`. The `version.h` genrule in
  `core/pkg/version/BUILD.bazel:12-23` reads `bazel-out/stable-status.txt` instead of
  the `VERSION` file, and takes the build timestamp from `volatile-status.txt` so a
  timestamp change does not invalidate the compile.
- **Console**: `tauri build --config '{"version":"X.Y.Z"}'`. `tauri.conf.json` carries
  `0.0.0`. `getVersion()` in `console/src/session/version/use.ts:20` keeps working.
- **TypeScript**: `pnpm -r exec npm version X.Y.Z --no-git-tag-version` before
  `pnpm publish -r`. Every `package.json` carries `0.0.0`; internal dependencies already
  resolve to `workspace:^` through the catalog, which pnpm rewrites at publish.
- **Python**: `uv version X.Y.Z` in each package before `scripts/pin_internal_deps.sh`
  and `uv build`. Every `pyproject.toml` carries `0.0.0`.

`scripts/check_versions.sh`, `scripts/bump_versions.sh`, and `test.updates.yaml` are
deleted. Dev builds of every product report `0.0.0-dev` plus the commit.

### 4.3 Updater manifest

`console/release-spec.json`, the `publish-console-update` job, and the `rc` endpoint
rewrite at `build.synnax.yaml:891-897` are deleted. The manifest lives on the
`console/vX.Y.Z` release as `latest.json`.

The docs site gains two Astro endpoints:

- `/releases/console/latest.json`: 302 to the `latest.json` asset of the newest stable
  Console release.
- `/releases/console/next.json`: 302 to the newest Console pre-release, falling back to
  stable when no pre-release is newer.

Both resolve through §4.4's release cache and set `Cache-Control` with a short
`s-maxage` so the Vercel CDN absorbs the 30 s polling of every installed Console.
`tauri.conf.json` points stable builds at the first route; the Console workflow passes
the second route through `--config` for pre-release builds, so a QA machine on a
candidate keeps receiving candidates until it installs a stable build. Tauri follows
redirects. The docs Console download button reads the same routes.

### 4.4 Docs site

`fetchVersion.ts` becomes `src/util/releases.ts`: `latest(product)` queries the GitHub
releases API with a server-side token, filters by tag prefix, skips drafts and
pre-releases, and caches per product for a few minutes. Every consumer
(`core/DownloadURL.astro`, `driver/DownloadURL.astro`, `code/MoveCommand.astro`,
`code/VersionOutput.astro`, `layout/Header.astro`) passes its product. The header shows
the Core's train. Because the site is server-rendered (`astro.config.ts:21`), links flip
within the cache window of a release with no deploy hook.

### 4.5 Feature flags

Static release toggles, one registry per surface, each entry naming an owner and the
release that removes it:

- **Console**: `console/src/flags.ts` exports `const FLAGS` built from `VITE_FLAG_*` env
  vars through a Vite `define`, default off, so production builds tree-shake dark code.
  The `IS_DEV` define at `console/vite.config.ts:157` folds into it.
- **Docs site**: `docs/site/src/flags.ts` reads `FLAG_*` through a typed `astro:env`
  schema. A page opts in with a `flag` frontmatter field and the reference layout
  answers 404 when it is off; `PageNavNode` gains an optional `flag` and the nav filters
  on it. Vercel preview deploys set every flag on.

Flipping a flag is a one-line PR to `main`. The Core gets no flag registry until a Core
feature needs one.

### 4.6 What this RFC does not cover

Release cadence and QA procedure (RFC 0020 §5 is superseded without replacement; the
`rc.md` checklist becomes a release issue template), migration testing between trains,
and the design of any future desktop product beyond its flag and its workflow file.

## 5 Implementation phases

- **Phase 1: Docs site.** `releases.ts`, the two updater routes, per-product consumers,
  the docs flag registry, `flag` on pages and nav. Works against today's `synnax-v` tags
  and tomorrow's `core/v` tags. Green on its own; deployable before any workflow
  changes.
- **Phase 2: Cutover.** One PR to `main`: the five `release.*.yaml` workflows,
  `resolve-version`, `.github/release.yml`, `workflow_call` on the integration suite,
  deletion of the three `deploy.*` publish workflows, `rc` removed from every trigger,
  `tauri.conf.json` endpoint swapped to the docs route, `CLAUDE.md` release-model and
  base-branch text rewritten. Version files stay in place and are overridden by
  injection, so the PR is workflow-only and bisects cleanly. Then: plain merge of `rc`
  into `main` (publishes nothing), `gh pr edit --base main` for the open PRs, delete
  `rc` and its protection.
- **Phase 3: Versions and Console flags.** Delete `VERSION`, the `//go:embed`, the
  fourteen version literals, `check_versions.sh`, `bump_versions.sh`,
  `test.updates.yaml`, and `release-spec.json`; add the Bazel status script and
  `console/src/flags.ts`. Source-only, isolated from Phase 2's CI risk.
- **Phase 4: First releases.** Dispatch Console and Driver with `bump: minor`, then the
  Core, opening train 0.59. Python and TypeScript follow.

Compatibility: shipped Consoles on 0.58 poll the old raw URL. `release-spec.json` stays
on `main` through Phase 3 pointing at the last 0.58 build; the first 0.59 Console
release also writes one final `release-spec.json` so 0.58 installs upgrade once and land
on the new endpoint. After that the file is deleted.

## 6 Resolved decisions

1. **Shared minor, independent patch**: Fully independent semver rejected; users could
   not answer cross-product compatibility questions. The trade is a train rule the Core
   release enforces.
2. **Candidate channel is a release property, not a branch**: A `prerelease` input on
   every workflow. Releasing straight to stable only was rejected; QA needs signed,
   installable candidates and the updater needs a channel to follow.
3. **Three binary products, built once**: A single `synnax` release for Core plus Driver
   was rejected; a Go-only change should not re-release the Driver or Console. The Core
   embeds published bundles instead of rebuilding them.
4. **No approval gate**: GitHub Environment reviewers rejected; the dispatch is the
   approval. The workflow verifies checks and runs integration tests instead.
5. **`bump` choice, not a typed version or a release bot**: release-please writes
   version files back into the repo, which principle 3 forbids; typed versions invite
   skipped numbers.
6. **Updater endpoint on the docs site**: `releases/latest/download` rejected because
   GitHub's `latest` is repo-wide and breaks under per-product tags; a rolling
   `console/latest` release rejected as tag abuse. The trade is that docs availability
   gates update checks, which are non-critical.
7. **Request-time docs versions, no deploy hook**: The site is already SSR; a deploy
   hook would fix the symptom of a build-time race that does not exist.
8. **Static flags per surface**: Version-gated docs pages rejected because unshipped
   features have no version; runtime flag services rejected because nothing needs a flip
   without a deploy.
9. **Cutover order**: Workflows first, then merge `rc`, then retarget and delete.
   Cutting over with `rc` unmerged rejected; every open PR would pay a rebase and the
   first release could ship half a train.

## 7 Open questions

- Exact tag prefixes: `console/v` vs `console-v`. GitHub sorts either; the docs regex
  and `resolve-version` take it as one constant.
- Release cache TTL on the docs site and `s-maxage` on the updater routes.
- Whether the Driver release should also publish a Docker image, now that it is
  separable from the Core.
- Which labels `.github/release.yml` categorizes on, given PR titles carry Linear
  numbers rather than conventional-commit types.

## 8 Prior art

- Fowler, Patterns for managing source code branches:
  https://martinfowler.com/articles/branching-patterns.html
- trunkbaseddevelopment.com, Branch for release and You're doing it wrong:
  https://trunkbaseddevelopment.com/branch-for-release/,
  https://trunkbaseddevelopment.com/youre-doing-it-wrong/
- DORA, Trunk-based development: https://dora.dev/capabilities/trunk-based-development/
- Hodgson, Feature toggles: https://martinfowler.com/articles/feature-toggles.html
- GitHub, `workflow_dispatch` and reusable workflows:
  https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows,
  https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows
- Tauri v2 updater and tauri-action: https://v2.tauri.app/plugin/updater/,
  https://github.com/tauri-apps/tauri-action
- Bazel stamping: https://bazel.build/docs/user-manual
- grafana/shared-workflows, per-component tags in a polyglot monorepo:
  https://github.com/grafana/shared-workflows
- npm and PyPI trusted publishing: https://docs.npmjs.com/trusted-publishers/,
  https://docs.pypi.org/trusted-publishers/
- GitHub auto-generated release notes:
  https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes
