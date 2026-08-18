# Testing Guide

Per-language frameworks, rules, and examples live in the toolchain docs
(`docs/claude/toolchains/*.md`): Vitest (TS), Ginkgo/Gomega (Go), pytest (Python),
Google Test + xtest (C++). All BDD-style; tests co-located with source (TS/Go/C++) or in
`tests/` (Python).

## Universal Practices

- Descriptive "should"-style names; AAA (arrange, act, assert); one behavior per test.
- Fast (unit tests <100ms), deterministic (no flaky tests, no random data), independent
  (no inter-test dependencies).
- Mock via interfaces / dependency injection, not by patching internals.

## Live-Core Tests

Client (TS/Python/C++) tests, Pluto and Console live-core specs, and Driver tests
connect to a core at `localhost:9090` (login `synnax`/`seldon`). Before running any of
them, check whether a core is already up:

```bash
curl -s -X POST http://localhost:9090/api/v1/connectivity/check
```

A JSON response means a core is running — use it. No response: build and start one
yourself (in-memory, insecure):

```bash
cd core && go build -o synnax . && cd ..
./core/synnax start -mi # -m in-memory storage, -i insecure; listens on localhost:9090
```

Start it in the background; in-memory state is lost on restart.

## Integration Testing (`/integration/`)

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
- Options: `--headed` (Playwright headed mode), `-d <rack>` (driver rack name)

### Organization

JSON sequence files in `/integration/tests/`: `console_tests.json`, `driver_tests.json`,
`arc_tests.json`, `client_tests.json`. Each defines sequences of cases running
sequentially or async with configurable pool sizes.

Console test helpers in `/integration/console/`: `layout.py` (navigation, toolbars,
keyboard), `context_menu.py`, `ranges.py` / `tasks.py` / `statuses.py` (resource
helpers), `case.py` (base `ConsoleCase` with Playwright setup).

### Environment dependencies

- Console tests: only the Synnax server (most tests).
- Driver tests: hardware simulators (OPC UA, Modbus, ...). Task lifecycle tests need
  `OPCUASim`.
- NI form tests may fail with notification overlays when no driver is connected.

### Debugging failures

Every run writes a debug bundle to `integration/tests/results/run-<ts>[-<name>]/` (also
via the `latest` symlink): `summary.json`, `server.log`, and per-test `tests/<name>/`
dirs with `trace.zip` (on failure), a sliced `server.log`, and screenshots/exports. CI
uploads the directory as a `test-results-<os>-<name>` artifact.

For triage workflow (including pulling CI bundles via `gh run download`), use the
`synnax-integration-debug` skill (`.claude/skills/synnax-integration-debug/SKILL.md`).

For writing integration tests — especially Arc reactive-runtime gotchas — see
`docs/claude/integration-test.md`.
