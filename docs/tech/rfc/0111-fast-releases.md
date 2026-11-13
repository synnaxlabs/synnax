# 111 Fast releases

- **Author**: Patrick Dotson
- **Date**: 2026-11-12
- **Related**: [RFC 0058 - Release workflow](0058-release-workflow.md)

## 0 Summary

The 0.61 release took 10 hours of workflow time after the last fix merged. One dispatch
now builds every product once, in parallel, on runners with a warm cache, and CI gates
only the publish. The target is 1 hour from dispatch to a published Synnax Desktop. This
RFC amends RFC 0058 §2.0 and §2.1 and deletes `release.desktop.yaml`.

## 1 Motivation

The last 0.61 fix merged at 02:06 UTC on 2026-11-12. Synnax Desktop published at 12:09
UTC. Six release runs did the work:

| Sink                        | Minutes | Cause                                               |
| --------------------------- | ------- | --------------------------------------------------- |
| macOS Driver build, 3 times | 339     | Hosted runner, no Bazel cache                       |
| Full CI, 6 times            | 173     | Every run starts with `ci.yaml` at `full: true`     |
| Lock wait                   | 182     | Three dispatches behind `concurrency: release`      |
| Failed Desktop runs         | 190     | Each retry repeats CI and the Driver build          |
| npm stage                   | by hand | `pnpm stage publish --batch` fails (fixed by #5371) |

- **Repeat CI**: Commit `6363a146` passed the full CI four times in 3.5 hours, at 185
  jobs and about 10 runner-hours each.
- **Serial dispatches**: The Console run waited 129 minutes for the Driver run, and the
  Core run 53 minutes (`release.yaml:62-64`).
- **Cold macOS builds**: A signed build runs on `macos-latest`
  (`generate_os_matrix.sh:27-31`). The Bazel remote cache has a private address, and the
  macOS step passes no cache flags (`build.synnax.yaml:427-435`). Bazel took 87 minutes
  for the Driver release and 110 minutes for Synnax Desktop. The Linux bot took 19.
- **Repeat builds**: Synnax Desktop builds the Driver from source
  (`release.desktop.yaml:150-155`), minutes after a release built the same code.
- **Serial job graph**: The packages wait for `publish-console`, `publish-driver`, and
  `publish-core` (`release.yaml:410`), and the Core waits for both publishes
  (`release.yaml:616`). Only the Core has a real input from the other products.
- **Stale Go caches**: The full CI took 36 minutes on a release branch and 21 minutes on
  `main`, where a nightly run had saved the cache for the same commit.

## 2 Principles

1. **CI gates the publish, never the build**: A build starts at dispatch. Nothing
   becomes public until the commit has a green full CI run.
2. **One green run per commit**: A commit that passed the full CI does not run it again.
3. **Build once**: A run builds each distinct binary one time. Every consumer takes that
   artifact. The Core of Desktop is a distinct binary: it embeds no Console.
4. **A job needs only what it reads**: A `needs` edge exists for an artifact or a
   verdict, never for order alone.
5. **Publish last, together**: All builds pass before any product, package, or image
   goes out.

## 3 Design

### 3.0 One dispatch, one branch

`release.yaml` releases every product, Synnax Desktop included. `release.desktop.yaml`
is deleted. Desktop gains two inputs in `release.yaml`: `desktop` and `desktop_bump`.

A release runs from `main` or from `release/X.Y`. One branch per train replaces
`release/<console|core|driver>-X.Y`. `resolve_version.sh:81` requires a product in the
name but uses only the `X.Y`, so the product carries no meaning. The docs deploy also
reads the branch name (`site/docs/src/util/deploy/plan.ts:82`).

Desktop joins a run with the other products only on `main`. A Desktop patch is its own
dispatch of `release.yaml` from `release/desktop-X.Y`, with only `desktop` set.

Desktop builds from the commit of its run, so it can ship a change before the Core or
the Driver releases it, or with build flags of its own:

- **Driver**: The Driver build of the run when the run releases both the Driver and the
  Core, since the commit and the minor are the same. Otherwise Desktop builds the Driver
  from source at `HEAD` on a build bot (§3.3), as today but with a warm cache.
- **Core**: Built from source at `HEAD`, as today. The released Core is 250 MB to 308
  MB, since it embeds the Console and its installers, and the Desktop installer is 45 MB
  to 70 MB. The Go build takes 3 to 4 minutes on a bot.
- **Version stamp**: The Core, the clients, and a Driver built from source carry one
  version, so the compatibility check passes (`client/cpp/connection/checker.cpp:36`).
  It is the Core version of the run, or else the newest stable Core tag reachable from
  `HEAD`, as today (`release.desktop.yaml:104-116`).

### 3.1 Job graph

```
resolve ─┬─ verify (green run lookup, else full CI) ───────────────┐
         ├─ build Driver (Windows, macOS, Linux, NI Linux RT) ──┐  │
         │    └─ build Desktop (own Core, Tauri app)            │  │
         ├─ build Console web ──────────────────────────────────┤  │
         ├─ build Console app (macOS, Windows)                  │  │
         ├─ build packages (npm, PyPI)                          │  │
         │                                    build Core ◄──────┘  │
         │                                      ├─ Windows installer
         │                                      └─ Core image       │
         └────────────────────────────── publish (all) ◄───────────┘
                                             └─ deploy docs
```

- **Drafts hold the artifacts**: Each build uploads to its draft release, as today. The
  Core build downloads the Driver and the Console web bundle from the drafts of the run.
  A run that does not release a product downloads its newest release on the train, as
  today (`release.yaml:211-238`).
- **Desktop**: The Desktop build takes the Driver that §3.0 names, builds its Core on a
  bot, and compiles the Tauri app.
- **Packages build early**: The npm and PyPI jobs build at dispatch. The stage and the
  upload move to the publish stage.
- **Publish**: One stage, after `verify` and every build pass. It clears the drafts,
  pushes the image, stages npm, and uploads to PyPI. The Core publishes last, so a
  public Core never embeds a release that is still a draft. A failed build leaves only
  drafts, which the next run clears (`draft-release/action.yaml:55`).
- **Docs**: One deploy per run, after the publish. The docs pages are prerendered, so
  the deploy sets the download links and the dates of the release notes.
- **Release notes date**: For 0.59 and later, `Release.astro` reads the publish date of
  `core/vX.Y.0` from the releases listing, which gains `published_at`. Those notes lose
  the `date` prop, so no pull request dates a release. Older notes keep the prop: their
  tags predate `core/v*`.

### 3.2 Verify

The `green-run` job looks for a successful run of `ci.yaml` with `full: true` on the
commit of the run. Nightly runs and earlier release runs both count. When none exists,
the `ci` job calls `ci.yaml`, in parallel with the builds. The `verify` job passes when
a green run exists or `ci` passed, and every publish needs it. A retry of a failed
release finds the green run and skips CI.

The lookup is a new script, `.github/scripts/green_run.sh`. It lists the runs of
`nightly.yaml`, `release.yaml`, and `release.desktop.yaml` on the commit and accepts one
whose `CI / OK` job passed in any attempt, so a rerun of all jobs finds the first
attempt of its own run. A full run has about 185 jobs, so the script pages the job list,
50 jobs per page: a page of 100 at times gets an HTTP 502. An API error counts as no
green run, and CI runs.

### 3.3 macOS builds on the build bots, signing on hosted runners

The Driver and the Core build unsigned on `macos-build-bot`, which keeps a warm Bazel
output base: `driver / Test (macos-build-bot)` takes 1 to 2 minutes. A job on
`macos-latest` then downloads the binaries, signs them, and notarizes them. The signing
key never reaches a bot, since the bots also run pull request jobs. The Tauri apps stay
on hosted runners: they sign inside the build and take 12 to 18 minutes.

### 3.4 Retries

- **CI**: §3.2 reuses the green run.
- **Builds**: A rerun of failed jobs keeps the passed builds. A new dispatch builds
  again, which costs about 20 minutes.

### 3.5 Development loop

- **Fail fast counts a cancelled job**: A runner that never took a job left a merge
  group alive for 32 minutes on #5367 (run 43131752204). `fail-fast` reads only
  `failure` and `timed_out` (`ci.yaml:1205`).
- **Sedosaur reports every finding in one round**: #5337 took 7 pushes in 68 minutes at
  one finding per round, and each push cancelled the CI in progress.
- **The queue skips what the pull request proved**: When the tree of a merge group
  equals the tree of the pull request head, a queue job passes at once if the same job
  passed on that head. Jobs that only the queue runs, such as integration, still run.

- **An outage run reruns itself**: A GitHub outage on 2026-11-11 failed 22 runs in which
  no job other than `OK` failed: jobs never got a runner, or `OK` saw `abandoned` needs
  (run 43090120253). Such a run can end before `OK` wakes the shepherd, so the sweep
  (`relay.sweep.yaml`) finds the failed `CI` runs of open pull requests and reruns each
  once.
- **External links warn on a pull request**: A 404 from `datatracker.ietf.org` failed
  `docs-site` on 4 runs. A broken external link fails only the nightly run.

## 4 Expected time

| Stage                                       | Minutes |
| ------------------------------------------- | ------- |
| Driver, Console, and package builds         | 20      |
| Core build, installer, image, Desktop build | 25      |
| Publish and docs                            | 5       |

`verify` takes 0 minutes after a green run and 21 to 36 minutes otherwise, in parallel.
The optimized macOS Driver build on a bot is not measured yet. Signing adds about 2
minutes.

## 5 What this RFC does not cover

- Version rules, tags, the train rule, and package staging. RFC 0058 holds them.
- The speed of the CI jobs themselves. `verify` runs in parallel with the builds, so it
  is off the critical path.
- The migration audit. It stays an input, in parallel with the builds.

## 6 Implementation phases

Phases 1 to 4 remove about 80% of the time. Each phase leaves `release.yaml` able to
release. Phase 9 saves 3 to 4 minutes per Windows build.

| Phase | Scope                                                    | Touches                                                                                                                                                                                                                           | Depends on |
| ----- | -------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Gate the release publish jobs on CI, not the builds      | `.github/workflows/release.yaml`, `.github/workflows/release.desktop.yaml`                                                                                                                                                        | -          |
| 2     | Reuse a green full CI run of the commit in a release     | `.github/scripts/green_run.sh`, `.github/workflows/release.yaml`                                                                                                                                                                  | 1          |
| 3     | Build macOS binaries on the bots and sign them on hosted | `.github/scripts/generate_os_matrix.sh`, `.github/workflows/build.synnax.yaml`                                                                                                                                                    | -          |
| 4     | Build the Core from drafts and publish all products last | `.github/workflows/release.yaml`, `.github/workflows/build.synnax.yaml`                                                                                                                                                           | 1          |
| 5     | Build the packages at dispatch and publish them last     | `.github/workflows/release.yaml`                                                                                                                                                                                                  | 4          |
| 6     | Release Synnax Desktop in the release workflow           | `.github/workflows/release.yaml`, `.github/workflows/build.tauri.yaml`, `.github/scripts/resolve_version.sh`                                                                                                                      | 4          |
| 7     | Release each train from one `release/X.Y` branch         | `.github/scripts/resolve_version.sh`, `.github/actions/resolve-version`, `site/docs/src/util/deploy/plan.ts`, `AGENTS.md`, `CONTRIBUTING.md`, `docs/tech/git.md`, `site/docs/README.md`, `docs/tech/rfc/0058-release-workflow.md` | -          |
| 8     | Read the release notes date from the Core release        | `site/docs/src/components/releases/Release.astro`, `site/docs/src/releases`                                                                                                                                                       | -          |
| 9     | Cache `trusted-signing-cli` on the Windows builds        | `.github/workflows/build.tauri.yaml`, `.github/workflows/build.synnax.yaml`                                                                                                                                                       | -          |
| 10    | Cancel a merge group when a job is cancelled             | `.github/workflows/ci.yaml`                                                                                                                                                                                                       | -          |
| 11    | Report every Sedosaur finding in one review              | `.sedosaur/prompt.md`                                                                                                                                                                                                             | -          |
| 12    | Skip queue jobs that passed on the same tree             | `.github/workflows/ci.yaml`, `.github/scripts/changed_jobs.py`                                                                                                                                                                    | -          |
| 13    | Rerun a failed CI run in which only `OK` failed          | `.github/workflows/relay.sweep.yaml`                                                                                                                                                                                              | -          |
| 14    | Fail external docs links only in the nightly run         | `site/docs/src/util/checks`, `.github/workflows/test.docs.yaml`                                                                                                                                                                   | -          |

## 7 Resolved decisions

1. **CI in parallel, not removed**: The pull request and the queue test the Core on
   Ubuntu only. The full run is the first on Windows and macOS, and it caught SY-5838
   during 0.61. Dropping it was rejected. The trade is real: a build can finish for a
   commit that then fails CI, which wastes runner time and publishes nothing.
2. **Lookup before a full run on every merge**: A full run per merge costs about 10
   runner-hours each, over 100 times a day. The lookup takes the nightly run when it
   matches and pays for one run otherwise.
3. **Build on the bots, sign on hosted runners**: Signing on the bots was rejected: the
   keychain stays on a machine that runs pull request jobs. A Bazel cache that hosted
   runners can reach was rejected for now: it needs a public endpoint with
   authentication. The trade is real: two bots limit a run to two macOS builds at a
   time.
4. **One publish stage**: A publish per product let the Driver and Console go out before
   a Core build failed. One stage keeps a failed run invisible. The cost is that no
   product ships until the slowest build ends.
5. **Desktop in the same workflow, from source**: A second workflow repeated CI and the
   2-hour Driver build. Released binaries for Desktop were rejected: Desktop could not
   ship a change ahead of a Core or Driver release, and the released Core is four times
   the size of the installer. The build bots remove the cost, not the bundling.
6. **One branch per train**: Three branches at one commit gave three dispatches. This
   reverses the branch names of RFC 0058 §2.0 and keeps its version rules.
7. **Docs deploy stays**: The pages are prerendered, so the deploy sets the links. It
   runs once per run, after the publish.
8. **No draft reuse across dispatches**: The version does not change after a failed run,
   so a draft under the same tag can come from an older commit. A rebuild costs 20
   minutes and carries no such risk.

## 8 Open questions

1. **Full run cadence on `main`**: Nightly today. A run every few hours when `main`
   moved makes the lookup hit more often, at about 10 runner-hours each.
2. **Queue skip and the Core tests**: The pull request runs the Core tests cached in one
   job, and the queue runs three uncached shards (`ci.yaml:428`). Phase 12 skips them
   only if the pull request runs the same job.
3. **Go caches on release branches**: The full CI takes 36 minutes on a release branch
   and 21 minutes with warm caches. A cache per shard and ref needs a measured size
   against the repository cache limit first.
4. **Third Mac bot**: Needed if the macOS builds of a run wait on each other.
