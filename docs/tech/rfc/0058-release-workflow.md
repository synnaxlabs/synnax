# 58 Release workflow

- **Author**: Patrick Dotson
- **Date**: 2026-09-21
- **Related**:
  [RFC 0020 - Engineering process standardization](0020-engineering-workflow.md),
  [RFC 0111 - Fast releases](0111-fast-releases.md)

## 0 Summary

The Core, Console, and Driver release on every push to `main` or `rc`, their versions
are hand-edited files, and the Console updater polls a JSON file a bot force-pushes to
`main`. This RFC moves to trunk-based development on `main`, one dispatched release
workflow with a checkbox per product, Git tags as the only version source, an updater
manifest hosted on the GitHub release, and static feature flags. Python and TypeScript
packages publish from the same workflow. It supersedes RFC 0020 §2, §3, §4, §5, and §7,
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
TypeScript packages publish from the same dispatch. Compatible products and packages
share a major.minor, the train, such as `0.59`. `main` is always releasable: unfinished
work ships dark behind a flag. No binary publishes on push; a person dispatches every
release. No file in Git carries a version; the tag is the version, and every manifest
carries `0.0.0`.

### 2.0 Branching

`main` is the only long-lived branch and every PR targets it. `rc`, the `rc.md` PR
template, and every `branches: [main, rc]` trigger are deleted. Pushes to `main` and
`release/**` build and test only.

Every release, candidates included, runs only from `main` or a release branch,
`release/X.Y`, one per train; Desktop also allows `release/desktop-X.Y`. RFC 0111 §3.0
changed the name from `release/<console|core|driver>-X.Y`, which now fails. Desktop
takes a minor or major bump from `main` and only a patch bump from its release branch,
once `X.Y` has a stable tag. Any other branch fails. The Console, Core, and Driver fail
when any stable tag of the product names `HEAD`, so no commit ships twice. A release
branch fails when it does not hold the newest tag of the product on `X.Y`, so no patch
drops an earlier fix.

The branch sets the version of the Console, Core, Driver, and packages. A release of the
Console, Core, or Driver from `main` is always the next minor: when the newest stable
tag is `X.Y.Z`, it is `X.(Y+1).0`, and its candidates are `X.(Y+1).0-rc.N`. A patch from
`main` is not possible. Release branches exist only for patches: when train `X.Y` needs
one, `release/X.Y` is cut from the `X.Y.0` tag, and the fix lands on `main` first, then
is cherry-picked by PR onto the branch. A release from the branch is always the next
patch on `X.Y`, or `X.Y.0` when the product has no tag on a train that another product
opened from `main`. A branch never opens a train.

Packages share the Core's minor, since the TypeScript and Python clients connect only to
a Core with the same major.minor. From `main`, packages release only in the same run as
the Core, at the next patch on the minor the Core releases; a packages-only run from
`main` fails and asks for `release/X.Y`. From a release branch, packages release the
next patch on `X.Y`, with or without the Core, but only once a Core is released on
`X.Y`.

### 2.1 Release workflow

One `workflow_dispatch` workflow, `release.yaml`, with inputs:

- **`console`, `driver`, `core`**: The products to release, all on by default. One
  dispatch ships a whole train; a hotfix ticks one product.
- **`typescript`, `python`**: Publish the npm and PyPI packages, both on by default, at
  the version the branch sets (§2.0). A candidate run never publishes them. An npm
  publish on an older train takes the `train-X.Y` dist-tag, so `latest` never moves
  back.
- **`prerelease`**: Tag `-rc.N` and mark the releases pre-release.
- **`console_version`, `driver_version`**: Releases the Core embeds. Default: the one
  this run releases, else the newest on the resolved minor (candidates included when
  `prerelease` is set), so a hotfix from `release/0.59` never picks up 0.60.

Each selected product runs four stages:

1. **Resolve**: A composite action `.github/actions/resolve-version` takes the version
   from the branch (§2.0) and the product's stable tags anywhere in the repo, so a patch
   tag on a release branch is never reissued. It enforces the train rule (§2.2).
   Candidates never set the base: `-rc.N` counts up from the candidates already tagged
   for that version, and a stable release from the same branch promotes it. The action
   also emits the previous stable product tag; a candidate uses the same baseline, so
   its notes cover every change since the last stable.
2. **Verify**: The commit needs one green run of `ci.yaml` with `full`. A nightly run or
   an earlier release run of the commit counts. If there is none, `ci.yaml` runs as a
   `workflow_call` job. The migration audit runs when its input is set. Both run in
   parallel with the builds and gate only the publish (RFC 0111 §2 and §3.2).
3. **Build**: `build.synnax.yaml` with only that product enabled and `version` passed
   through, signed.
4. **Publish**: Draft release under the tag, upload assets, `generate_release_notes`
   with categories from `.github/release.yml` and `previous_tag_name` set to the
   previous stable product tag, since GitHub's default is the repo's last release of any
   product. One `publish` job clears the drafts of the run, the Core last, after Verify
   and every build pass. The release creates the tag, so a failed build or a failed
   Verify leaves only drafts and no tag. Concurrency group `release` serializes
   releases.

Assets per product:

- **Console** (`console/vX.Y.Z`): `console-web-vX.Y.Z.tar.gz`, the DMG, the updater
  artifacts (`Synnax.app.tar.gz`, the MSI, and the NSIS installer) with their `.sig`
  files, and `latest.json`, which points at the updater artifacts. The DMG is for first
  installs only and carries no signature.
- **Driver** (`driver/vX.Y.Z`): The four platform binaries and the NI install script.
- **Core** (`core/vX.Y.Z`): Downloads the Console bundle and Driver binaries into
  `core/pkg/console/dist/` and `core/pkg/driver/assets/`, from the drafts of the run or,
  for a product the run does not release, from its published release. It then builds the
  binaries, Windows installer, and Docker image (`next` for a pre-release that no Core
  tag outranks, `latest` for a stable that no other stable Core tag outranks, so a
  hotfix on an old train never moves either one backwards). No workflow rebuilds another
  product. The notes name the embedded versions. The image pushes in its own job after
  the publish, and no job waits on it, so a failed push blocks only the image. It reads
  the tags at push time, so a late rerun never moves a channel back.

The Console and the Driver build in parallel. The Core embeds them, so its build waits
for both builds and for the updater manifest, and never runs after one fails. A failed
build publishes no product of the run. A rerun of the failed jobs keeps the drafts that
the passed builds filled, and a new dispatch clears them and builds again.

`deploy.synnax.yaml` is deleted; every binary release is a new version, so nothing is
ever already published.

Each language is a product with one lockstep tag: `ts/vX.Y.Z` for the npm packages and
`py/vX.Y.Z` for the PyPI packages. No package depends on one in the other language, so
each language releases alone. A language takes the Core's minor (§2.0): from `main`, the
next patch on the minor of the run's Core; from `release/X.Y`, the next patch on `X.Y`.
A version a registry once held is never reused, even after a yank, so the next patch
skips it: with `ts/v0.61.0` and `py/v0.61.0` yanked, the run that ships Core 0.61.0
publishes packages 0.61.1. The publish job waits for every product the run releases,
then runs `stamp_versions.sh` to write the version into that language's manifests. It
stops when the version's tag names another commit, or when the registry has the version
without a tag, so one version never holds packages from two commits.

The Python job claims the version by pushing its tag at the run's commit, then
publishes. To finish a partial publish, rerun the failed jobs, which keep the resolved
versions, or dispatch the packages from `release/X.Y` cut at the run's commit: the
resolve stage keeps a tag on the train that names `HEAD`, and PyPI skips what already
went up.

The npm trusted publishers allow staging only. The npm job stages each package with
`pnpm stage publish -r`, since pnpm cannot stage a batch, then pushes the tag. Nothing
installs until a maintainer approves each package on npmjs.com with 2FA, in dependency
order: `x`, `alamos`, `freighter`, then `client`. A staged version is hidden from the
registry, so a run whose tag already names its commit skips the stage. While `client`
waits for approval at the newest `ts/v*` version, overall or on the run's train, the job
stages nothing, so approvals never move a dist-tag back. Three cases need a person. To
reject a release, reject every staged package and delete its `ts/v*` tag. If a stage
fails part way, reject what staged and rerun. No tag guards those packages, so a later
approval of one can move a dist-tag back; the run summary asks for the reject. If the
tag push fails after a stage, push the tag at the run's commit by hand and rerun.

A merge publishes nothing. Both publish through OIDC trusted publishing with
`id-token: write`, so no registry token lives in the repo's secrets. The registries
trust `release.yaml` by name, so the publish jobs live in it, not in a called workflow.
Packages have no candidate channel: a candidate run never publishes them.

### 2.2 Versions

Train rule: a product may bump to at most one minor ahead of the Core's latest stable,
and the Core release fails unless the Console and Driver it embeds share its minor.
Console and Driver open a train; the Core closes it. A Console or Driver run from `main`
leaves out the packages, and the closing run releases only the Core and packages, since
the Console and Driver would resolve a second minor ahead. Every product and every
package shares the train's minor; patches are independent, so a Console hotfix ships as
`0.59.1` while the Core stays at `0.59.0`, and Pluto `0.59.2` works with every client
`0.59.x`. Each language releases its packages together at one version, on the Core's
minor. The C++ client and the C library are on no registry and ship with the Driver, so
they carry the Driver's `SYNNAX_DRIVER_VERSION` define and have no manifest. The catalog
pins internal deps as `workspace:^`, which pnpm rewrites to `^X.Y.Z` at publish, so any
patch mix inside a train resolves; `pluto/package.json` pins `@synnaxlabs/freighter` and
`@synnaxlabs/media` as `workspace:*`, which publishes exact versions, and both move to
the catalog.

Every binary manifest carries `0.0.0` and the build injects the resolved `version`:

- **Core**: The existing `-ldflags -X` in `build.synnax.yaml`. The `VERSION` file and
  `//go:embed` fallback in `get.go` are deleted; `Get()` returns `0.0.0` when unset.
- **Driver**: A `SYNNAX_DRIVER_VERSION` Bazel define, `0.0.0` in `.bazelrc`, that the
  release build overrides; the `//driver/version` genrule reads it.
- **Console**: `tauri build --config '{"version":"X.Y.Z"}'`. A candidate runs as app
  version `X.Y.Z-N`, since the MSI bundler accepts only a numeric pre-release; its tag
  stays `X.Y.Z-rc.N` and its manifest carries the app version.

Dev binaries therefore run at `0.0.0`. The client compatibility checks (`isCompatible`
in `client/ts/src/connection/status.ts`, `_versions_compatible` in
`client/py/synnax/connection.py`, and `versions_compatible` in
`client/cpp/connection/checker.cpp`, which the Driver ships) require an equal
major.minor today and gain one rule: a `0.0` on either side is compatible. Any client
then connects to a dev Core without a mismatch warning. A release build of the Console
runs `stamp_versions.sh`, which writes its version into the package manifests, so the
Console's guard reports the Console's version. Each Desktop build stamps the version of
the Core that it embeds.

`check_versions.sh` and `bump_versions.sh` are deleted. A dev build of a client runs at
`0.0.0`, like a dev binary.

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
- **Docs site**: `site/docs/src/flags.ts` reads `FLAG_*` through `astro:env`. Pages opt
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
- **Phase 2: Cutover.** One PR: `release.yaml`, `resolve-version`,
  `.github/release.yml`, `deploy.synnax.yaml` deleted, `rc` removed from every trigger,
  the updater endpoint swapped, `CLAUDE.md` rewritten. Version files stay and injection
  overrides them, so no version file changes. Then merge `rc` into `main` (publishes
  nothing), `gh pr edit --base main` for open PRs, delete `rc`.
- **Phase 3: Versions and Console flags.** Delete the binary version literals; point
  `check_versions.sh` at the `core/` tags and trim `bump_versions.sh`; move Pluto's two
  `workspace:*` pins to the catalog; add the Bazel define, the `0.0` compatibility rule
  in all three clients, and `console/src/flags.ts`.
- **Phase 4: First releases.** One PR bumps every package manifest to 0.59, then one
  `release.yaml` dispatch from `main` opens train 0.59. After the Console release, one
  manual commit copies its `latest.json` into `release-spec.json`, so installed 0.58
  builds upgrade once and land on the new endpoint. The file stays on `main` until 0.58
  is out of support.

## 4 Resolved decisions

1. **Shared minor, independent patch**: Independent semver rejected; users could not
   answer cross-product compatibility questions.
2. **Candidate channel is a release property**: A `prerelease` input, not a branch. QA
   needs signed candidates and the updater needs a channel.
3. **Three binary products, built once**: A single Core plus Driver release rejected; a
   Go-only change must not re-release the Driver or Console.
4. **No approval gate**: Environment reviewers rejected; the dispatch is the approval.
5. **The branch sets the version**: `main` releases the next minor and a release branch
   the next patch; packages follow the Core's minor. Any other branch fails, so a
   feature branch never opens a train. A `bump` input rejected: a patch from `main`
   tagged next-minor code on the old train once that train had moved to its release
   branch. release-please rejected because it writes version files; typed versions
   rejected because they invite skipped numbers. Desktop keeps a `bump`, since its
   version line is apart from the train. RFC 0111 changed the release branch name from
   one per product, `release/<console|core|driver>-X.Y`, to one per train,
   `release/X.Y`.
6. **Updater endpoint on the docs site**: `releases/latest/download` is repo-wide and
   breaks under per-product tags; a rolling `console/latest` release abuses tags.
7. **Request-time docs versions**: The site is already SSR; a deploy hook fixes a race
   that does not exist.
8. **Static flags per surface**: Version-gated docs pages rejected because unshipped
   features have no version; runtime flag services rejected because nothing needs a flip
   without a deploy.
9. **Cutover order**: Workflows first, then merge `rc`, then retarget and delete, so no
   PR pays a rebase and the first release ships a whole train.
10. **Packages version from one lockstep tag per language**: Manifest versions rejected;
    a bump PR merged 0.61.0 into every manifest and published it before any 0.61 Core
    existed, and bump PRs are a second release decision outside the dispatch.
    Per-package tags rejected because the cascade through internal ranges then needs a
    person or Changesets. One tag for both languages rejected because a TypeScript fix
    would publish a Python version with no changes. Lockstep inside a language
    republishes unchanged packages, which is cheap and keeps every internal range
    satisfiable.
11. **One workflow, a checkbox per product**: `release.yaml` ships a minor with one
    click and a hotfix with one box ticked. A workflow per product plus a train over
    them was rejected as four files for one job; separate dispatches as three clicks in
    a forced order. The products keep their own tags and releases.
12. **Tag prefix `product/v`**: `console/v0.59.0`, the shape the repo's own tags had
    through 0.13 (`synnax/v0.13.1`). `console-v` rejected because the slash groups tags
    by product in every listing.
13. **Cache TTLs**: The updater routes send `s-maxage=300, stale-while-revalidate=60`
    and `releases.ts` caches a listing for five minutes, so the CDN absorbs the polling
    and a release lands on the site within minutes.
14. **Release note categories**: Each product's notes list only the PRs labeled for what
    it ships: the Console takes `console` and `ts-client`, the Driver `driver`, `arc`,
    and `cpp-client`, and the Core `core` and `arc`, with links to the Console and
    Driver it embeds. The configs live in `.github/release/`. Synnax Desktop keeps every
    category in `.github/release.yml`.

## 5 Open questions

- Whether the Driver release also publishes a Docker image.

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
