# 58 Release workflow

- **Author**: Patrick Dotson
- **Date**: 2026-09-21
- **Related**:
  [RFC 0020 - Engineering process standardization](0020-engineering-workflow.md)

## 0 Summary

The Core, Console, and Driver release on every push to `main` or `rc`, their versions
are hand-edited files, and the Console updater polls a JSON file a bot force-pushes to
`main`. This RFC moves to trunk-based development on `main`, one dispatched release
workflow per binary product with a train workflow over them, Git tags as the only
version source for those products, an updater manifest hosted on the GitHub release, and
static feature flags. Python and TypeScript packages keep their current flow: versioned
in their manifests, published on merge. It supersedes RFC 0020 §2, §3, §4, §5, and §7,
and retires the `rc.md` checklist that RFC 0020 §6 links. From §4 only the shared minor
survives: every product and package shares one minor, and patches move independently.

## 1 Motivation

- **`rc` is a second mainline**: 76 commits ahead of `main`, 26 open PRs, 16 back-merges
  from `main` in July and August 2026.
- **Release is a side effect of a merge**: `deploy.synnax.yaml:12-89` publishes on any
  push to `main` or `rc` that touches a product path. Hotfix branches cannot release.
- **Binary versions are hand-maintained**: `core/pkg/version/VERSION` and
  `tauri.conf.json` are edited by hand and checked against the 12 package manifests for
  major.minor agreement only.
- **`console/release-spec.json` is a production endpoint**: force-pushed to `main` by CI
  and polled every 30 s by every shipped Console.
- **Docs links go stale**: `fetchVersion.ts:33` memoizes the releases lookup for the
  life of a warm Vercel function.
- **Nothing can ship dark**: the Console, Core, and docs site have no feature flags.

## 2 Design

Three binary products release by dispatch: Console, Core, Driver. The Python and
TypeScript packages publish on merge, as today. Compatible products and packages share a
major.minor, the train, such as `0.59`. `main` is always releasable: unfinished work
ships dark behind a flag. No binary publishes on push; a person dispatches every
release. No file in Git carries a binary product version; the tag is the version. A
package manifest carries its own version.

### 2.0 Branching

`main` is the only long-lived branch and every PR targets it. `rc`, the `rc.md` PR
template, and every `branches: [main, rc]` trigger are deleted. Pushes to `main` and
`release/**` build and test only.

A hotfix is fixed on `main` first, then cherry-picked by PR onto
`release/<product>-X.Y`, cut from the product's latest stable tag. The release workflow
runs from that branch and resolves its minor from the tags reachable from it. The branch
is deleted afterwards.

### 2.1 Release workflows

One `workflow_dispatch` workflow per binary product, `release.<product>.yaml`, with
inputs:

- **`bump`**: `patch` (default) or `minor`.
- **`prerelease`**: Tag `-rc.N` and mark the release pre-release.
- **`console_version`, `driver_version`** (Core only): Releases to embed. Default: the
  newest stable on the resolved minor, or the newest pre-release on it when `prerelease`
  is set, so a hotfix from `release/core-0.59` never picks up 0.60.

Each runs four stages:

1. **Resolve**: A composite action `.github/actions/resolve-version` takes the minor
   from the highest stable product tag reachable from `HEAD`, then the patch from the
   highest stable product tag on that minor anywhere in the repo, applies `bump`, and
   enforces the train rule (§2.2). Candidates never set the base: `-rc.N` counts up from
   the candidates already tagged for that version, and promoting one repeats the same
   `bump`. The action also emits the previous stable product tag; a candidate uses the
   same baseline, so its notes cover every change since the last stable. Reachability
   picks the train; the repo-wide scan keeps a hotfix tag from being reissued.
2. **Verify**: The commit's required checks must have passed. The integration suite runs
   as a `workflow_call` job.
3. **Build**: `build.synnax.yaml` with only that product enabled and `version` passed
   through, signed.
4. **Publish**: Draft release under the tag, upload assets, `generate_release_notes`
   with categories from `.github/release.yml` and `previous_tag_name` set to the
   previous stable product tag, since GitHub's default is the repo's last release of any
   product, then clear the draft. The release creates the tag, so a failed build leaves
   none. Concurrency group `release-<product>` serializes a product's releases.

Assets per product:

- **Console** (`console/vX.Y.Z`): `console-web-vX.Y.Z.tar.gz`, the DMG, the updater
  artifacts (`Synnax.app.tar.gz`, the MSI, and the NSIS installer) with their `.sig`
  files, and `latest.json`, which points at the updater artifacts. The DMG is for first
  installs only and carries no signature.
- **Driver** (`driver/vX.Y.Z`): The four platform binaries and the NI install script.
- **Core** (`core/vX.Y.Z`): Downloads the Console bundle and Driver binaries from their
  releases into `core/pkg/console/dist/` and `core/pkg/driver/assets/`, then builds the
  binaries, Windows installer, and Docker image (`latest` for stable, `next` for
  pre-release). No workflow rebuilds another product. The notes name the embedded
  versions.

A fourth workflow, `release.train.yaml`, is the everyday path. It takes the same `bump`
and `prerelease` inputs and calls the three product workflows, which also expose
`workflow_call` and a `version` output: Console and Driver in parallel, then the Core
with their outputs as `console_version` and `driver_version`. One dispatch releases a
whole train. A product that fails leaves no tag and the rest stand; rerun that product
alone, and a Core rerun's defaults pick up the others. The per-product workflows stay
for hotfix patches.

`deploy.synnax.yaml` is deleted; every binary release is a new version, so nothing is
ever already published.

Python and TypeScript keep `deploy.py.yaml` and `deploy.ts.yaml`. A merge to `main` that
touches a package publishes every package whose manifest version is not yet on the
registry; `uv publish --check-url` and `pnpm publish -r` skip the rest. The version bump
in the PR is the release decision, and a package that did not change is never
republished. Python moves to OIDC trusted publishing; TypeScript already has
`id-token: write`. Packages have no candidate channel.

### 2.2 Versions

Train rule: a product may bump to at most one minor ahead of the Core's latest stable,
and the Core release fails unless the Console and Driver it embeds share its minor.
Console and Driver open a train; the Core closes it. Every product and every package
shares the train's minor; patches are independent, so a Console hotfix ships as `0.59.1`
while the Core stays at `0.59.0`, and Pluto `0.59.2` works with every client `0.59.x`.
`check_versions.sh` enforces the rule for the packages: every manifest shares one minor,
and that minor is the Core's latest stable minor or the next one, read from the `core/`
tags instead of the `VERSION` file. A split, with some manifests on each train, fails
the check, so a package minor bump is one PR that moves every manifest to the new train.
The catalog pins internal deps as `workspace:^`, which pnpm rewrites to `^X.Y.Z` at
publish, so any patch mix inside a train resolves; `pluto/package.json` pins
`@synnaxlabs/freighter` and `@synnaxlabs/media` as `workspace:*`, which publishes exact
versions, and both move to the catalog.

Every binary manifest carries `0.0.0` and the build injects the resolved `version`:

- **Core**: The existing `-ldflags -X` (`build.synnax.yaml:621-626`). The `VERSION` file
  and `//go:embed` fallback in `get.go` are deleted; `Prod()` returns `0.0.0-dev` when
  unset.
- **Driver**: Bazel `--stamp` with a `--workspace_status_command` emitting
  `STABLE_SYNNAX_VERSION`. The `//core/pkg/version` genrule is already stamped; it
  switches from the `VERSION` file and `date` to `stable-status.txt` and
  `volatile-status.txt`.
- **Console**: `tauri build --config '{"version":"X.Y.Z"}'`.

Dev binaries therefore run at `0.0.0`. The client compatibility checks (`isCompatible`
in `client/ts/src/connection/status.ts`, `_versions_compatible` in
`client/py/synnax/connection.py`, and `versions_compatible` in
`client/cpp/connection/checker.cpp`, which the Driver ships) require an equal
major.minor today and gain one rule: a `0.0` on either side is compatible. A dev Console
or Driver then connects to any Core, and any client connects to a dev Core, without a
mismatch warning.

`bump_versions.sh` drops the `VERSION` and `tauri.conf.json` edits and bumps only the
package manifests.

### 2.3 Updater manifest and docs site

The `publish-console-update` job and the `rc` endpoint rewrite in
`build.synnax.yaml:891-897` are deleted; `release-spec.json` follows once 0.58 is out of
support (§3). Consoles built from `rc` poll a URL that dies with the branch, so those
machines reinstall. `latest.json` lives on the Console release.

The docs site gains two Astro endpoints, `/releases/console/latest.json` and
`/releases/console/next.json`, with `s-maxage` so the Vercel CDN absorbs the polling.
The first 302s to the manifest of the highest stable Console version; the second to the
highest version on either channel. Highest means semver order, never release date, so a
hotfix on an old train never outranks the current one. Stable builds point at the first;
pre-release builds get the second through `--config`. Semver ranks `0.59.0` above
`0.59.0-rc.3`, so a QA machine follows candidates, installs the next stable, and lands
on the stable endpoint.

`fetchVersion.ts` becomes `releases.ts`: `latest(product)` queries the releases API with
a server-side token, filters by tag prefix, skips drafts and pre-releases, returns the
highest semver, and caches for a few minutes. The API lists by creation date, so a
hotfix on an old train can sit first. Every download component passes its product; the
header shows the Core's train. The site is server-rendered, so links flip within the
cache window with no deploy hook. The docs site reads the release, never the repo.

### 2.4 Feature flags

A flag is a static build-time boolean, default off, that hides unfinished work in
production. One registry per surface names each flag's owner and the release that
removes it:

- **Console**: `console/src/flags.ts` builds `const FLAGS` from
  `import.meta.env.VITE_FLAG_*`, which Vite replaces statically, so production
  tree-shakes dark code. `IS_DEV` folds in.
- **Docs site**: `docs/site/src/flags.ts` reads `FLAG_*` through `astro:env`. Pages opt
  in with `flag` frontmatter (404 when off); `PageNavNode` gains `flag`. Preview deploys
  set every flag on.

### 2.5 Out of scope

Release cadence and QA procedure (the `rc.md` checklist becomes a release issue
template), migration testing between trains, and any future desktop product beyond its
flag and workflow file.

## 3 Implementation phases

- **Phase 1: Docs site.** One manual release per binary product under its new tag, so
  `latest(product)` needs no legacy map and the first dispatched release computes its
  version from a real tag. Core, Driver, and Console at 0.58.2 with the assets copied
  from `synnax-v0.58.2` and `console-v0.58.2`, plus a `latest.json` written from
  `release-spec.json`, which is the only place the manifest exists today. Every
  bootstrap tag sits on the `synnax-v0.58.2` commit on `main`, so Resolve reaches it
  from `HEAD`. Then `releases.ts`, the updater routes, per-product consumers, and docs
  flags, all deployable before any workflow change.
- **Phase 2: Cutover.** One PR: the four `release.*.yaml`, `resolve-version`,
  `.github/release.yml`, `deploy.synnax.yaml` deleted, `rc` removed from every trigger,
  the updater endpoint swapped, `CLAUDE.md` rewritten. Version files stay and injection
  overrides them, so no version file changes. Then merge `rc` into `main` (publishes
  nothing), `gh pr edit --base main` for open PRs, delete `rc`.
- **Phase 3: Versions and Console flags.** Delete the binary version literals; point
  `check_versions.sh` at the `core/` tags and trim `bump_versions.sh`; move Pluto's two
  `workspace:*` pins to the catalog; add the Bazel status script, the `0.0`
  compatibility rule in all three clients, and `console/src/flags.ts`.
- **Phase 4: First releases.** One PR bumps every package manifest to 0.59, then one
  `release.train.yaml` dispatch with `bump: minor` opens train 0.59. After the Console
  release, one manual commit copies its `latest.json` into `release-spec.json`, so
  installed 0.58 builds upgrade once and land on the new endpoint. The file stays on
  `main` until 0.58 is out of support.

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
10. **Packages keep the merge-publish flow**: Tag-resolved lockstep versions rejected
    for Python and TypeScript; they republished every package on every patch and cost a
    product tag, a workflow, and version injection to replace a flow that already works.
    Per-package tags rejected because the cascade through internal ranges then needs a
    person or Changesets. The manifest version is the release decision, the registry's
    skip of published versions is the change detection, and the published `^X.Y.0`
    ranges lock the minor on 0.x, so any patch mix inside a train resolves.
11. **One dispatch per train**: `release.train.yaml` composes the three product
    workflows so a minor ships with one click; separate dispatches were rejected as
    three clicks in a forced order. The products keep their own tags, releases, and
    hotfix workflows.

## 5 Open questions

- Tag prefix separator: `console/v`, the shape the repo's own tags had through 0.13
  (`synnax/v0.13.1`), or `console-v`.
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
