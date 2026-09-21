# Repo scripts

Scripts developers run, most also used by CI. Prefer them over hand-rolled equivalents:
they encode repo-wide conventions (file lists, ignore patterns, header formats) that are
easy to get wrong by hand.

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

- `check_versions.sh` — verifies every npm and PyPI package manifest shares one minor,
  and that the minor is the Core's latest stable `core/v*` tag or the next one (the
  train rule). Needs tags fetched.
- `bump_versions.sh <version>` — sets a new semver (`X.Y.Z`) across those manifests.
  Release tooling; don't run ad hoc against a dirty tree.

## Workflow-only scripts (`.github/scripts/`)

Scripts only GitHub Actions runs live beside the workflows, each documented in its
header: `resolve_version.sh`, `latest_version.sh`, and `verify_checks.sh` behind the
release actions (pytest coverage beside them), `check_artifact_cache.sh`,
`generate_os_matrix.sh`, `verify_build_config.sh`, `import_apple_certificate.sh`,
`pin_internal_deps.sh`, `prune_published.py`, and the Windows installer inputs.

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
