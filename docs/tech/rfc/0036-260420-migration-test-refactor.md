# 36 - Migration Test Refactor

**Feature Name**: Version-Specific Migration Test Setup <br /> **Status**: Draft <br />
**Start Date**: 2026-04-20 <br /> **Authors**: Patrick Dotson <br />

# 0 - Summary

The migration test infrastructure is refactored so that each released version gets a
frozen setup script written against that version's client API. A shell script
orchestrates walking through a chain of old versions, and the existing test conductor
verifies everything survived using the latest client.

This replaces the current architecture where a single Python conductor runs current-repo
test code against old client wheels — a design that breaks whenever the client API or
its transitive dependencies change.

# 1 - Motivation

PR #2236 exposed the structural flaw: the migration tests fail because `pymodbus` was
upgraded from 3.12 to 3.13 on `rc`, but the old client wheel (0.54.x) bundles code
written for the 3.12 API. The current migration conductor installs the old wheel into a
`venv` but always uses current-repo test code (via `PYTHONPATH`), so any API drift — in
the client itself or in its dependencies — causes failures unrelated to actual
migration.

More broadly, if `client.channels.create()` gains a new required parameter, every old
version's `setup` phase breaks because the current test code calls methods that don't
exist in the old client.

The fix must ensure that setup code for version X only ever runs against the version X
client, with dependencies resolved by the package manager — not pinned manually.

# 2 - Design

## 2.0 - Architecture

Migration testing is split into two independent concerns:

**Setup chain** (`run_migration_chain.sh`): A shell script that loops through old
versions in order. For each version it downloads the Core binary, starts it against a
shared data directory, creates an isolated venv with `synnax=={version}` from PyPI, runs
that version's `setup.py`, and stops Core.

**Verification** (`uv run tc migration`): A standard test conductor invocation against
the latest Core. Uses the current workspace's `synnax` client via `uv run`. Full access
to test infrastructure — assertions, result reporting, etc.

### Directory Structure

```
integration/
├── migration/
│   └── setup/
│       ├── v0_54/
│       │   └── setup.py          # Standalone script for 0.54.x
│       ├── v0_55/
│       │   └── setup.py          # Standalone script for 0.55.x
│       └── ...
├── scripts/
│   └── run_migration_chain.sh
└── tests/
    ├── migration/
    │   ├── channels.py            # Verify test cases
    │   ├── channels_calc.py
    │   ├── ranges.py
    │   ├── rbac.py
    │   ├── task.py
    │   ├── task_opc.py
    │   ├── task_modbus.py
    │   └── task_ni.py
    └── migration_tests.json
```

Verify tests remain under `tests/migration/` because the test conductor's discovery
mechanism prepends `tests/` to all case paths.

## 2.1 - Setup Scripts

Each `setup/v0_XX/setup.py` is a standalone Python script. It connects to Core at
`localhost:9090` via `import synnax`, creates resources (channels, ranges, tasks, etc.),
and exits. No `TestCase` inheritance, no framework imports, no assertions — only Python
stdlib and the `synnax` package.

When a new version is released, a snapshot of the current resource-creation logic is
frozen into a new version folder. Once committed, it is never modified.

### v1 Scope

In the initial implementation, all setup scripts run unconditionally — every version
folder is executed in order. A future iteration will add a `from_version` argument (e.g.
`from_version=v0.57`) that runs only setup scripts up to and including that version,
skipping `v0_58`, `v0_59`, etc. This enables testing migrations from a specific starting
point without running the full chain.

## 2.2 - Dependency Resolution

Each setup folder targets a minor version (e.g. `v0_54`), not a specific patch. The
shell script installs the latest patch within that minor — `synnax>=0.54,<0.55` — so
setup scripts automatically pick up bug-fix releases without needing a new folder. The
resolver pulls the correct transitive dependencies from the installed version's package
metadata. No manual `deps.txt` or version pinning is needed.

For the verify phase, `uv run` resolves `import synnax` to the local workspace code via
the uv workspace configuration.

## 2.3 - Verification Tests

Files in `tests/migration/` are standard test conductor test cases. They inherit from
existing integration base classes (e.g. `TestCase` from
`integration/framework/test_case.py`) and use the full test infrastructure. They always
run with the latest client and current-repo code.

Test discovery uses `tests/migration_tests.json` with cases like
`{"case": "migration/channels"}`.

## 2.4 - Shell Script

The shell script loops through the version chain, and for each version:

1. Downloads / caches the Core binary from GitHub releases.
2. Starts Core against a shared data directory.
3. Creates an isolated venv and installs `synnax=={version}` from PyPI.
4. Runs the version-specific `setup.py`.
5. Stops Core.

The data directory persists across versions so each setup builds on the previous
version's state — the same way a real deployment accumulates data across upgrades.

# 3 - CI Integration

The `test.migration.yaml` workflow changes:

- The `build-clients` job is removed — no more wheel building.
- The test job splits into two steps: run the setup chain, then run verify against the
  latest Core.
- `generate_migration_matrix.sh` automatically discovers versions from GitHub releases
  and removes the `VERSIONS` output (was used by the deleted `build-clients` job).

# 4 - Adding a New Version

1. Create `integration/migration/setup/v0_XX/setup.py` — snapshot the current
   resource-creation logic as a standalone script written against the v0.XX client API.
2. The `tests/` folder, shell script, and CI workflow need no changes unless you want to
   add new verification tests.
3. `generate_migration_matrix.sh` automatically discovers the new version from GitHub
   releases.
