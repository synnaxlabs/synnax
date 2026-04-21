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

Additionally, device simulators are moved from `client/py/examples/` into
`integration/simulators/`, and the client package stops exporting `examples`.

# 1 - Motivation

PR #2236 exposed the structural flaw: the migration tests fail because `pymodbus` was
upgraded from 3.12 to 3.13 on `rc`, but the old client wheel (0.54.x) bundles simulator
code written for the 3.12 API. The current migration conductor installs the old wheel
into a `venv` but always uses current-repo test code (via `PYTHONPATH`), so any API
drift — in the client itself or in its dependencies — causes failures unrelated to
actual migration.

More broadly, if `client.channels.create()` gains a new required parameter or
`sy.modbus.ReadTask` changes its constructor signature, every old version's `setup`
phase breaks because the current test code calls methods that don't exist in the old
client.

The fix must ensure that setup code for version X only ever runs against the version X
client, with dependencies resolved by the package manager — not pinned manually.

# 2 - Design

## 2.0 - Architecture

Migration testing is split into two independent concerns:

**Setup chain** (`run_migration_chain.sh`): A shell script that loops through old
versions. For each version it downloads the Core binary, starts it against a shared data
directory, creates an isolated venv with `synnax=={version}` from PyPI, runs that
version's `setup.py`, and stops the Core. Pure orchestration — no Python framework
imports.

**Verification** (`uv run tc migration_verify`): A standard test conductor invocation
against the latest Core (started separately). Uses the current workspace's `synnax`
client via `uv run`. Full access to test infrastructure — Playwright, assertions, result
reporting.

## 2.1 - Version Folder Setup Scripts

Each `v0_XX/setup.py` is a standalone Python script. It connects to Core at
`localhost:9090` via `import synnax`, creates resources (channels, ranges, tasks, etc.),
and exits. No `TestCase` inheritance, no framework imports, no assertions.

Setup scripts should NOT start simulators. They use the Synnax API to create and
configure tasks directly. The verify phase handles starting simulators and validating
that tasks actually run.

When a new version is released, a snapshot of the current resource-creation logic is
frozen into a new version folder. Once committed, it is never modified.

## 2.2 - Dependency Resolution

When the shell script runs `uv pip install synnax==0.54.0`, the resolver automatically
pulls the correct transitive dependencies (`pymodbus`, `asyncua`, `numpy`, etc.) that
0.54.0 declared in its package metadata. No manual `deps.txt` or version pinning is
needed.

For the verify phase, `uv run` resolves `import synnax` to the local workspace code via
the uv workspace configuration. No wheels, no build step.

## 2.3 - Verify Tests

Files in `verify/` are standard test conductor test cases. They inherit from existing
base classes (`ReadTaskMigrationVerify`, `SimulatorCase`, etc.) and use the full test
infrastructure. They always run with the latest client and current-repo code.

Test discovery uses `tests/migration_verify_tests.json` with cases like
`{"case": "migration/verify/channels"}`. Nested directory paths already work with the
existing `config_client.py` discovery mechanism.

## 2.4 - Simulator Relocation

Device simulators are test infrastructure, not client library code. They are moved from
`client/py/examples/` into `integration/simulators/`. The client package
(`client/py/pyproject.toml`) stops exporting `examples` — the directory remains as
user-facing example scripts but is no longer packaged.

All import sites in `integration/` (~20 files across `tests/driver/`, `tests/arc/`,
`tests/console/`) are updated from `from examples.*` to `from simulators.*`.

# 3 - Directory Structure

```
integration/
├── simulators/                         # Moved from client/py/examples/
│   ├── __init__.py
│   ├── simulator.py
│   ├── device_sim.py
│   ├── simdaq.py
│   ├── modbus_sim.py
│   ├── opcua_sim.py
│   ├── http_sim.py
│   ├── press.py
│   ├── thermal.py
│   ├── tpc.py
│   └── load_current.py
├── scripts/
│   └── run_migration_chain.sh
├── tests/migration/
│   ├── v0_54/
│   │   └── setup.py
│   ├── verify/
│   │   ├── __init__.py
│   │   ├── channels.py
│   │   ├── channels_calc.py
│   │   ├── labels.py
│   │   ├── ranges.py
│   │   ├── rbac.py
│   │   ├── task_modbus.py
│   │   ├── task_ni.py
│   │   ├── task_opc.py
│   │   └── workspace.py
│   └── task.py                         # Shared base classes (verify-side only)
├── tests/migration_verify_tests.json
└── (old flat migration files removed)
```

# 4 - Shell Script: `run_migration_chain.sh`

```bash
#!/usr/bin/env bash
# Usage: run_migration_chain.sh "0.54.0,0.54.4"
set -euo pipefail

CHAIN="$1"
DATA_DIR="${HOME}/synnax-data"
BINARY_DIR="${HOME}/synnax-binaries"
CACHE_DIR="${HOME}/synnax-binary-cache"
VENV_DIR="${HOME}/migration-client-env"
PORT=9090
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INTEGRATION_DIR="$(dirname "$SCRIPT_DIR")"

rm -rf "$DATA_DIR"

IFS=',' read -ra VERSIONS <<< "$CHAIN"
for version in "${VERSIONS[@]}"; do
  version=$(echo "$version" | xargs)

  # 1. Download/install Core binary from GitHub releases
  # (cache in CACHE_DIR, copy to BINARY_DIR)

  # 2. Start Core against shared data directory
  mkdir -p "$DATA_DIR"
  "$BINARY_DIR/synnax" start -i &
  CORE_PID=$!
  # Poll localhost:PORT until ready ...

  # 3. Create isolated venv with old client from PyPI
  rm -rf "$VENV_DIR"
  uv venv "$VENV_DIR"
  uv pip install --python "$VENV_DIR/bin/python" "synnax==$version"

  # 4. Run version-specific setup script
  major_minor="v${version%.*}"
  major_minor="${major_minor//./_}"
  "$VENV_DIR/bin/python" "$INTEGRATION_DIR/tests/migration/$major_minor/setup.py"

  # 5. Stop Core
  kill "$CORE_PID" && wait "$CORE_PID" || true
  # Poll until port released ...
done
```

# 5 - CI Integration

The `test.migration.yaml` workflow changes:

- The `build-clients` job is removed entirely — no more wheel building.
- The test job splits into two steps: run the setup chain, then run verify.

```yaml
- name: Run Setup Chain
  run: integration/scripts/run_migration_chain.sh "${{ matrix.chain }}"

- name: Start Latest Core
  run: |
    mkdir -p ~/synnax-data
    ~/synnax-binaries/synnax start -i &
    # wait for port to be ready ...

- name: Run Migration Verify
  working-directory: integration
  run: uv run tc migration_verify

- name: Stop Core
  if: always()
  run: kill $(lsof -ti:9090) || true
```

The matrix chain format is adjusted so it only includes old versions (not `latest`),
since `latest` is handled by the `verify` step.

`generate_migration_matrix.sh` removes the `VERSIONS` output (was used by the deleted
`build-clients` job).

# 6 - Files Changed

| File                                                    | Change                                              |
| ------------------------------------------------------- | --------------------------------------------------- |
| `integration/simulators/`                               | New — simulators moved from `client/py/examples/`   |
| `integration/tests/driver/*.py`, `tests/arc/*.py`, etc. | Update imports `examples.*` → `simulators.*`        |
| `client/py/pyproject.toml`                              | Remove `examples` from `packages`                   |
| `integration/tests/migration/v0_54/setup.py`            | New — standalone setup script for 0.54.x            |
| `integration/tests/migration/verify/*.py`               | New — verify test cases from flat files             |
| `integration/tests/migration_verify_tests.json`         | New — test sequence for verify                      |
| `integration/scripts/run_migration_chain.sh`            | New — shell script chain orchestrator               |
| `integration/migration_conductor.py`                    | Deleted                                             |
| `integration/tests/migration/{channels,labels,...}.py`  | Deleted — split into v0_54/ and verify/             |
| `integration/tests/migration_tests.json`                | Deleted                                             |
| `integration/scripts/build_client_wheels.sh`            | Deleted                                             |
| `integration/scripts/transform_schematic.py`            | Deleted                                             |
| `integration/scripts/sim_from_task_configs.py`          | Deleted                                             |
| `.github/workflows/test.migration.yaml`                 | Remove `build-clients` job, split into chain+verify |
| `integration/scripts/generate_migration_matrix.sh`      | Remove `VERSIONS` output                            |
| `integration/pyproject.toml`                            | Remove `migration-conductor`/`mc` entry points      |

# 7 - Adding a New Version

If you want to add new setup code, say for v0.55, you can do the following:

1. Create `integration/tests/migration/v0_55/setup.py` — snapshot the current
   resource-creation logic as a standalone script written against the 0.55 client API.
2. The `verify` folder, shell script, and CI workflow need no changes unless you want to
   add new tests.
3. `generate_migration_matrix.sh` automatically discovers the new version from GitHub
   releases.

This way, you can add new setup code for a new version without having to modify the
existing setup code.
