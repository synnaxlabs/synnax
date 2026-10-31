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

## Live-Core tests

Client (TS/Python/C++) tests, Pluto and Console live-Core specs, and Driver tests
connect to a Core (login `synnax`/`seldon`). They read `SYNNAX_HOST` and `SYNNAX_PORT`,
and default to `localhost:9090`.

Parallel sessions share this machine. Build a Core from your worktree and run it on a
free port. Never use, stop, or replace a Core that you did not start.

```bash
cd core && go build -o synnax . && cd ..
lsof -nP -iTCP:9190 -sTCP:LISTEN          # no output means the port is free
./core/synnax start -mi -l localhost:9190 # -m in-memory storage, -i insecure
```

Start it in the background; in-memory state is lost on restart. Then set `SYNNAX_PORT`
on every test run, because a directory target can include live-Core specs:

```bash
SYNNAX_PORT=9190 pnpm test:console
cd client/py && SYNNAX_PORT=9190 uv run pytest tests/test_channel.py
SYNNAX_PORT=9190 bazel test //client/cpp/channel:all
```

- `SYNNAX_PORT` does not move a Core. A Core reads `-l` (`--listen`) or `SYNNAX_LISTEN`.
- A Core needs a license to answer API calls. It reads `SYNNAX_LICENSE_KEY`. Look for
  `license active` in its log.
- A Core older than your branch fails specs in ways that look like regressions. Rebuild
  it after a merge.
- `//driver/rack:config_test`, `cluster/login_forms`, and `channel/calc_type_cascade`
  need a Core on 9090. They ignore `SYNNAX_PORT`.

### Dev Console on a free port

```bash
SYNNAX_PORT=9190 SYNNAX_CONSOLE_PORT=5190 pnpm dev:console-vite
```

`SYNNAX_PORT` names the Core, and `SYNNAX_CONSOLE_PORT` names the Vite port (default
5173). Vite exits if its port is taken. A Core that the browser already selected wins
over `SYNNAX_PORT`, so a new `SYNNAX_CONSOLE_PORT` gives a clean start.

`pnpm dev:console` stays on 5173, and its single-instance plugin blocks a second one.
Use `pnpm dev:console-vite` for parallel work.

## Integration testing (`/integration/`)

A custom Python **test conductor** orchestrates the full stack: Go server +
TS/Python/C++ clients. Console UI tests use Playwright.

### Build & run sequence

Integration tests need a running Synnax server with the embedded Console:

```bash
pnpm build:console-vite                                                 # 1. build Console web assets
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

### Organization

JSON sequence files in `/integration/tests/`: `console_tests.json`, `driver_tests.json`,
`arc_tests.json`, `client_tests.json`. Each defines sequences of cases running
sequentially or async with configurable pool sizes. A case runs for at most its
`timeout` seconds (default 300), then the conductor marks it TIMEOUT and moves on.

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
via the `latest` symlink): `summary.json`, `server.log`, and per-test `tests/<name>/`
dirs with `trace.zip` (on failure), a sliced `server.log`, and screenshots/exports. CI
uploads the directory as a `test-results-<os>-<name>` artifact.

For triage workflow (including pulling CI bundles via `gh run download`), use the
`synnax-integration-debug` skill (`.agents/skills/synnax-integration-debug/SKILL.md`).

For writing integration tests — especially Arc reactive-runtime gotchas — see
`docs/claude/integration-test.md`.
