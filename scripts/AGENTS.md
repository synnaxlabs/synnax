# Repo scripts

Scripts developers run, most also used by CI. Prefer them over hand-rolled equivalents:
they encode repo-wide conventions (file lists, ignore patterns, header formats) that are
easy to get wrong by hand.

## Formatting

- `fix.sh` — adds copyright headers to files and formats them with Prettier, Ruff,
  `golangci-lint fmt`, and clang-format. With no arguments it fixes files changed
  against the merge base with `origin/main`; `--staged` fixes staged files and stages
  the fixes; file arguments fix only those files. It skips a formatter it cannot find.
- Go has no repo script: format with `golangci-lint fmt` in the module (dry run:
  `--diff`), driven by the formatters in the root `.golangci.yaml`.
- `clang_format.sh <path>` / `clang_format.sh --files <f1> <f2> ...` — formats
  `.cpp`/`.hpp`/`.h`/`.cc` in place, parallelized across cores. Excludes come from the
  root `.clang-format-ignore`. `check_clang_format.sh <path>` — dry run, non-zero exit
  on diffs.
- `install_clang_format.sh` — installs the pinned clang-format with `uv tool install`.
  Always use this over a package manager, so the version matches CI. Bazel fetches
  clang-tidy itself (`MODULE.bazel`).

## Linting

- `clang_tidy_fix.sh <bazel targets...>` — applies the fixes clang-tidy offers for the
  C++ files in the targets, then formats the changed code. Lint without fixing through
  `bazel build --config=clang-tidy <targets>`.
- `clang_tidy_wrapper.sh` — the clang-tidy the Bazel aspect runs. Never call it
  directly.

Prettier (TS) and Ruff (Python) run via each toolchain's own CLI, not a repo script.

- `uv run --project scripts scripts/check_exports_py.py` — reports top-level Python
  names without a leading underscore that only their own module or test code imports.
  Tests may import names from a test utility (`testutil`, `test`, or `mock`). Fix a
  finding with a leading underscore, or move the helper into a test utility. The package
  list is `PACKAGES` in the script.
- `python3 scripts/check_exports_ts.py` — reports TypeScript exports that only their own
  module or test code use. It runs Knip in production mode, then traces each finding. An
  export that only other modules' tests use is allowed only in a test utility directory
  (`testutil`, `test`, or `mock`).

## Copyright headers

- `update_copyrights.sh` — rewrites or inserts the BSL header (from
  `licenses/headers/template.txt`) on every tracked file, repo-wide. Comment style and
  leading-line handling (shebangs, `@echo off`, astro frontmatter fences) are resolved
  per file extension. Reads `.copyrightignore` for exclusions. Never call its `__batch`
  worker mode directly.
- `check_copyrights.sh [subdir]` — read-only check of the same rules, scoped to an
  optional subdirectory; non-zero exit on missing, stale-year, malformed, or duplicate
  headers. Run before push after adding or copying files.

Neither script touches `.oracle` schema files.

## Codegen

- `check_go_generate.sh <path>` — runs `go generate ./...` in `<path>` (repo-root
  relative), then `git add --intent-to-add` so new generated files show up in
  `git diff`. Verifies generated Go output is checked in and current; not a fix-up.

## Agents

- `dispatch_agent_task.sh [--dry-run] [--agent claude|codex] SY-####` — sends an
  `agent-task` dispatch that asks a remote agent to fix a Linear issue, then prints the
  run link. Used by the `bug-from-email` skill.

## Versioning and release

- `check_versions.sh` — verifies every npm and PyPI package manifest and the C++
  client's `VERSION` file share one minor, and that the minor is the Core's latest
  stable `core/v*` tag or the next one (the train rule). Needs tags fetched.
- `bump_versions.sh <version>` — sets a new semver (`X.Y.Z`) across those manifests.
  Release tooling; don't run ad hoc against a dirty tree.
- `uv run reliability.py <run-url> --version <X.Y.Z>`: builds the docs reliability page
  data from a CI run and uploads it to DigitalOcean Spaces. Redeploy the docs to publish
  it. Needs `gh` auth, `DO_SPACES_KEY`, and `DO_SPACES_SECRET`.
  `--manual-pass <skip reason>` notes that the tests skipped for that reason were run by
  hand and passed; `--manual-note <text>` says why, on the page.

## Bazel

- `run_bazel.sh <bazel arguments...>` — runs Bazel, and recovers once from an external
  repo that a repo contents cache deletion left unloadable, or from a link against
  archives a dependency bump made stale. Only the Windows CI steps use it, because those
  runners hold one output base for weeks.

## Coding agents

- `agent/run` — runs Claude Code or Codex headless, then opens a PR, comments on a PR,
  or writes a report under a GitHub App bot identity. `--dry-run` pushes nothing. The
  module docstring in `agent/agent.py` lists the flags and the environment.

## Toolchain bootstrap

- `install_antlr4.sh` — installs a pinned, checksum-verified antlr4 4.13.2 onto PATH,
  bypassing `antlr4-tools`' unreliable "latest version" lookup. Used by Arc grammar
  codegen (`arc/AGENTS.md`).

## Sanitizers (`sanitizers/`)

`asan_suppressions.txt`, `lsan_suppressions.txt`, `ubsan_suppressions.txt` — suppression
lists for ASan/LSan/UBSan runs against the C++ driver. Add an entry here instead of
disabling a sanitizer wholesale.

## Runner fleet (`runners/`)

The controller for the self-hosted CI runners, deployed to the `Sandbox-Windows` EC2
instance. `controller.py` polls GitHub and EC2, writes fleet state to the Synnax Core on
that box, and runs the scaler; `dispatch.py` turns the scaler's commands into EC2 starts
and stops. Its own uv project and CI job. A pool in `scaler.py` must use the label the
workflows request (`windows-test-bot`, not `windows-latest`), or its jobs never start an
instance.

Deploy with the `Deploy - Runners` workflow: it copies the `RUNNERS_*` repo secrets into
Parameter Store under `/runners/` and runs `deploy.ps1` on the box over SSM. Never start
the scripts by hand on the box.

The `Sync - Runner webhook ranges` workflow runs `hook_ranges.py` daily. It keeps the
controller's TCP 9091 rules in sync with the `hooks` ranges in GitHub's meta API.
