# Repo Scripts (`/scripts/`)

Shell scripts used locally and by CI. Prefer these over hand-rolled equivalents — they
encode repo-wide conventions (file lists, ignore patterns, per-extension header formats)
that are easy to get subtly wrong by hand.

## Formatting

- Go has no repo script: format with `golangci-lint fmt` in the module (dry run:
  `--diff`), driven by the formatters in the root `.golangci.yaml`.
- `clang_format.sh <path>` / `clang_format.sh --files <f1> <f2> ...` — formats
  `.cpp`/`.hpp`/`.h`/`.cc` in place, parallelized across cores. Excludes handled by the
  root `.clang-format-ignore`, not this script. `check_clang_format.sh <path>` — dry
  run, non-zero exit on diffs.
- `install_clang_format.sh` — installs the pinned LLVM 22 clang-format via apt. Always
  use this over a plain `apt install clang-format`, to keep the version consistent with
  CI.

Prettier (TS) and Ruff (Python) run via each toolchain's own CLI, not a repo script.

## Copyright Headers

- `update_copyrights.sh` — rewrites/inserts the BSL header (from
  `licenses/headers/template.txt`) on every tracked file, repo-wide (`git ls-files`, no
  path argument). Comment style, header size, and leading-line handling (shebangs,
  `@echo off`, astro frontmatter fences) are resolved per file extension. Parallelized
  via a `__batch` worker mode invoked through `xargs`; don't call `__batch` directly.
  Reads `.copyrightignore` for exclusions. Use this instead of typing headers by hand.
- `check_copyrights.sh [subdir]` — read-only check of the same header rules, scoped to
  an optional subdirectory; non-zero exit on missing/stale-year/malformed/duplicate
  headers. Run before push if you've added or copied files.

Neither script touches `.oracle` schema files — no header comments there.

## Codegen

- `check_go_generate.sh <path>` — runs `go generate ./...` in `<path>` (repo-root
  relative), then `git add --intent-to-add` so newly generated files show up in
  `git diff` alongside modifications. Used to verify generated Go output is checked in
  and current, not to be run as a silent fix-up.

## Versioning & Release

- `check_versions.sh` — verifies version strings are consistent across the repo's
  version-bearing files.
- `bump_versions.sh <version>` — sets a new semver (`X.Y.Z`) across those same files.
  CI/release tooling; don't run ad hoc against a dirty tree.
- `pin_internal_deps.sh` — rewrites internal workspace dependency constraints (alamos,
  synnax-freighter, synnax-x) in each Python package's `pyproject.toml` to a pinned
  range, derived from each dependency's own version. Runs in the deploy pipeline
  immediately before `uv build` against an ephemeral release checkout; not meant to be
  committed or run locally.
- `verify_build_config.sh` / `generate_os_matrix.sh` — CI-only, consume positional
  platform/build flags from the GitHub Actions workflow to validate the build matrix and
  emit its OS list. Not useful outside that workflow context.
- `import_apple_certificate.sh` — CI-only, imports the Apple Developer `.p12` into a
  per-runner keychain for macOS code signing. Requires runner-scoped env vars
  (`APPLE_CERTIFICATE*`, `KEYCHAIN_*`, `GITHUB_ENV`).

## Toolchain Bootstrap

- `install_antlr4.sh` — installs a pinned, checksum-verified antlr4 4.13.2 onto PATH,
  bypassing `antlr4-tools`' unreliable "latest version" lookup. Used by Arc grammar
  codegen (`arc/CLAUDE.md`).

## Sanitizers (`scripts/sanitizers/`)

`asan_suppressions.txt`, `lsan_suppressions.txt`, `ubsan_suppressions.txt` — suppression
lists for ASan/LSan/UBSan runs against the C++ driver. Add an entry here instead of
disabling a sanitizer wholesale.
