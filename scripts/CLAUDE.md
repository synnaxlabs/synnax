# Repo scripts

Shell scripts used locally and by CI. Prefer these over hand-rolled equivalents: they
encode repo-wide conventions (file lists, ignore patterns, per-extension header formats)
that are easy to get wrong by hand.

## Formatting

- Go has no repo script: format with `golangci-lint fmt` in the module (dry run:
  `--diff`), driven by the formatters in the root `.golangci.yaml`.
- `clang_format.sh <path>` / `clang_format.sh --files <f1> <f2> ...` — formats
  `.cpp`/`.hpp`/`.h`/`.cc` in place, parallelized across cores. Excludes come from the
  root `.clang-format-ignore`. `check_clang_format.sh <path>` — dry run, non-zero exit
  on diffs.
- `install_clang_format.sh` — installs the pinned LLVM 22 clang-format via apt. Always
  use this over a plain `apt install clang-format`, so the version matches CI.

Prettier (TS) and Ruff (Python) run via each toolchain's own CLI, not a repo script.

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

## Versioning and release

- `check_versions.sh` — verifies version strings are consistent across the repo's
  version-bearing files.
- `bump_versions.sh <version>` — sets a new semver (`X.Y.Z`) across those same files.
  Release tooling; don't run ad hoc against a dirty tree.
- `pin_internal_deps.sh` — pins internal workspace dependency ranges in each Python
  package's `pyproject.toml` before `uv build` in the deploy pipeline. Never commit its
  output.
- `resolve_version.sh <product> <bump> [prerelease]` — resolves the next `console`,
  `core`, or `driver` version from Git tags and prints workflow outputs (`version`,
  `tag`, `minor`, `previous_tag`). Backs `.github/actions/resolve-version`; tests in
  `test_resolve_version.py`.
- `latest_version.sh <product> <major.minor> [candidates]` — prints the highest product
  version on a train from Git tags, or nothing; with `candidates=true`, `-rc.N` tags
  count. Backs the Core release's embedded-version defaults.
- `verify_checks.sh <sha> <run_id> <branch>` — fails unless every GitHub Actions check
  on the commit passed, ignoring the given run, and the newest push run of each test,
  lint, and check workflow that ran on the branch passed. Backs
  `.github/actions/verify-checks`; tests in `test_verify_checks.py`.
- CI-only, called by the workflows and documented in their headers:
  `verify_build_config.sh`, `generate_os_matrix.sh`, `check_artifact_cache.sh`,
  `import_apple_certificate.sh`.

## Bazel

- `run_bazel.sh <bazel arguments...>` — runs Bazel, and recovers once from an external
  repo that a repo contents cache deletion left unloadable. Only the Windows CI steps
  use it, because those runners hold one output base for weeks.

## Toolchain bootstrap

- `install_antlr4.sh` — installs a pinned, checksum-verified antlr4 4.13.2 onto PATH,
  bypassing `antlr4-tools`' unreliable "latest version" lookup. Used by Arc grammar
  codegen (`arc/CLAUDE.md`).

## Sanitizers (`sanitizers/`)

`asan_suppressions.txt`, `lsan_suppressions.txt`, `ubsan_suppressions.txt` — suppression
lists for ASan/LSan/UBSan runs against the C++ driver. Add an entry here instead of
disabling a sanitizer wholesale.
