# Testing guide

Per-language frameworks, rules, and examples live in the toolchain docs
(`docs/claude/toolchains/*.md`): Vitest (TS), Ginkgo/Gomega (Go), pytest (Python),
Google Test + testutil (C++). All BDD-style; tests co-located with source (TS/Go/C++) or
in `tests/` (Python).

## Universal practices

- Descriptive "should"-style names; AAA (arrange, act, assert); one behavior per test.
- Fast (unit tests <100ms), deterministic (no flaky tests, no random data), independent
  (no inter-test dependencies).
- Mock via interfaces / dependency injection, not by patching internals.

## Benchmark gate

A change to a hot path (write, read, stream, codec, calculation) must hold the gate
benchmarks. `scripts/compare_benchmarks.py` measures two Git refs and gives the verdict:

```bash
uv run --project scripts scripts/compare_benchmarks.py --base main --head HEAD
```

- It builds each ref once, then runs each group once per ref in each of 10 rounds, back
  to back. A benchmark fails on a `sec/op` rise over 5% at p < 0.05, on an `allocs/op`
  rise of any size at p < 0.05 (Go and C++), or when head does not have it.
- `GROUPS` in the script lists the gate. A new gate benchmark needs a row there.
  `--only <regex>` limits a run to the groups a change touches.
- The `cpu` suite runs on any quiet host. The `io` suite (`--suite io --io-dir <dir>`)
  measures the disk that holds `<dir>`, so run it only on a host with a local NVMe
  drive.
- On a new host, first run with `main` as both refs. A failure there means the host is
  too noisy for the gate.
- Never run two benchmark processes at once. Never compare numbers from two hosts or
  from two runs.
- The `BenchmarkFloor*` benchmarks in `cesium` measure what the host can do: sync, copy,
  checksum, and sequential write and read. A storage target is a ratio to a floor from
  the same run. A floor that breaks the rule means the host is noisy.

## Live-Core tests

Client (TS/Python/C++) tests, Pluto and Console live-Core specs, and Driver tests
connect to a Core (login `synnax`/`seldon`). They read `SYNNAX_TEST_HOST` and
`SYNNAX_TEST_PORT`, and default to `localhost:9090`.

Parallel sessions share this machine. Build a Core from your worktree and run it on a
free port. Never use, stop, or replace a Core that you did not start.

```bash
cd core && go build -o synnax . && cd ..
lsof -nP -iTCP:9190 -sTCP:LISTEN          # no output means the port is free
./core/synnax start -mi -l localhost:9190 # -m in-memory storage, -i insecure
```

Start it in the background; in-memory state is lost on restart. Then set
`SYNNAX_TEST_PORT` on every test run, because a directory target can include live-Core
specs:

```bash
SYNNAX_TEST_PORT=9190 pnpm test:console
cd client/py && SYNNAX_TEST_PORT=9190 uv run pytest tests/test_channel.py
SYNNAX_TEST_PORT=9190 bazel test //client/cpp/channel:all
```

- `SYNNAX_TEST_PORT` does not move a Core. A Core reads `-l` (`--listen`) or
  `SYNNAX_LISTEN`.
- A Core needs a license to answer API calls. It reads `SYNNAX_LICENSE_KEY`. Look for
  `license active` in its log.
- A Core older than your branch fails specs in ways that look like regressions. Rebuild
  it after a merge.
- `//driver/rack:config_test` and `channel/calc_type_cascade` need a Core on 9090. They
  ignore `SYNNAX_TEST_PORT`.

### Dev Console on a free port

```bash
SYNNAX_TEST_PORT=9190 SYNNAX_CONSOLE_PORT=5190 pnpm dev:console-vite
```

`SYNNAX_TEST_PORT` names the Core, and `SYNNAX_CONSOLE_PORT` names the Vite port
(default 5173). Vite exits if its port is taken. A Core that the browser already
selected wins over `SYNNAX_TEST_PORT`, so a new `SYNNAX_CONSOLE_PORT` gives a clean
start.

`pnpm dev:console` stays on 5173, and its single-instance plugin blocks a second one.
Use `pnpm dev:console-vite` for parallel work.

## Integration testing (`/integration/`)

A custom Python **test conductor** orchestrates the full stack: Go server +
TS/Python/C++ clients. Console UI tests use Playwright.

### Build & run sequence

Integration tests need a running Synnax server with the embedded Console:

```bash
VITE_FLAGS_ENABLED=true pnpm build:console-vite                         # 1. build Console web assets
cp -r console/dist/* core/pkg/console/dist/                             # 2. copy into Core's embed dir
cd core && go build -tags=console -ldflags="-w -s" -o synnax . && cd .. # 3.
mkdir -p ~/synnax-data && cd ~/synnax-data                              # 4. start server (in-memory)
./path/to/synnax start -mi &
cd integration && uv run tc console # 5. run tests
```

The `-tags=console` build tag activates `core/pkg/console/enabled.go`
(`//go:embed all:dist`).

### Test conductor CLI (`uv run tc`)

- `tc console` — all cases in `console_tests.json` (1-part: file)
- `tc console/label` — cases matching "label" (2-part: file + case filter)
- `tc console/channel/calc` — file + sequence + case filter (3-part)
- `tc -f modbus` — global filter across all test files
- Options: `--headed` (Playwright headed mode), `--slow-mo <ms>` (delay per Playwright
  action), `-d <rack>` (driver rack name), `--logs` (stream case logs as they happen
  instead of only after a failure), `--port <port>` (the Core to test, default 9090)
- `--browsers <n>` limits how many Console tests run at once.
- `--cores <n> --core-binary <path>` runs sequences on n Cores at once. The conductor
  starts the extra Cores on the ports after `--port` and stops them at exit. A test that
  holds NI runs on the first Core, and an `isolated` sequence runs alone at the end.
- `--shard <k>/<n>` runs part k of n of the selected tests. Shards split whole sequences
  by the test times in `tests/durations.json`, so give each shard the same other flags.
  A sequence marked `independent` splits into single tests.
  `scripts/update_durations.py` rebuilds the file from CI test results.

### Organization

JSON sequence files in `/integration/tests/`: `console_tests.json`, `driver_tests.json`,
`arc_tests.json`, `client_tests.json`. Each defines sequences of cases running
sequentially or async with configurable pool sizes. A case runs for at most its
`timeout` seconds (default 300), then the conductor marks it TIMEOUT and moves on.

The conductor runs each failed case one more time. A failed case of an asynchronous
sequence runs again with the whole sequence, because it can need its peers, and each
case keeps the result of its last run. Mark the sequence `independent` to run only the
failed cases again.

Console test helpers in `/integration/console/`: `layout.py` (navigation, toolbars,
keyboard), `context_menu.py`, `ranges.py` / `tasks.py` / `statuses.py` (resource
helpers), `case.py` (base `ConsoleCase` with Playwright setup).

### Environment dependencies

- Console tests: only the Synnax server (most tests).
- Driver tests: hardware simulators (OPC UA, Modbus, ...). The Console OPC UA and Modbus
  task tests run from `driver_tests.json` to balance job time; they and task lifecycle
  need `OPCUASim` or `ModbusSim`.
- NI form tests may fail with notification overlays when no driver is connected.

### Debugging failures

Every run writes a debug bundle to `integration/tests/results/run-<ts>[-<name>]/` (also
via the `latest` symlink): `summary.json`, `server.log`, and per-test `tests/<case>/`
dirs with `trace.zip` (on failure), a sliced `server.log`, and screenshots/exports. CI
uploads the directory as a `test-results-<os>-<name>` artifact.

For triage workflow (including pulling CI bundles via `gh run download`), use the
`synnax-integration-debug` skill (`.agents/skills/synnax-integration-debug/SKILL.md`).

For writing integration tests — especially Arc reactive-runtime gotchas — see
`docs/claude/integration-test.md`.
