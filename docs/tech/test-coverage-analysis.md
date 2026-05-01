# Test Coverage Analysis

A survey of test coverage across the Synnax monorepo, with prioritized
recommendations for where to invest test effort next.

## TL;DR

Coverage is uneven across toolchains. The highest-leverage gaps, ranked:

1. **Console Redux slices** — 8 of 12 slices (`workspace`, `layout`, `range`,
   `status`, `table`, `arc`, `docs`, `version`) have zero unit tests. These are
   the authoritative state for the desktop app.
2. **Console hardware integrations** — 226 untested files across `ni`, `modbus`,
   `opc`, `labjack`, `ethercat`, `http` device modules. Critical user-facing
   surface area.
3. **Core API layer (Go)** — `core/pkg/api/{auth,channel,ranger,access}` have
   no tests despite being the public HTTP/gRPC contract.
4. **Driver factories (C++)** — Zero tests across all 7 device factories
   (OPC, NI, LabJack, Modbus, HTTP, EtherCAT, Arc); these are the device
   initialization entry points.
5. **Integration test conductor** — The `/integration/framework/` Python harness
   itself has no unit tests, despite being ~3.5k LOC of test-orchestration code
   that the entire integration suite depends on.

## Coverage Snapshot

### Go

| Module        | Source | Test | Ratio |
| ------------- | ------ | ---- | ----- |
| arc/go        | 176    | 170  | 97%   |
| cesium        | 49     | 47   | 96%   |
| x/go          | 252    | 221  | 88%   |
| oracle        | 93     | 72   | 77%   |
| aspen         | 36     | 25   | 69%   |
| alamos/go     | 8      | 5    | 63%   |
| freighter/go  | 34     | 17   | 50%   |
| **core**      | 467    | 212  | **45%** |

### TypeScript

| Package          | Source | Spec | Ratio |
| ---------------- | ------ | ---- | ----- |
| client/ts        | 155    | 58   | 37%   |
| x/ts             | 162    | 64   | 40%   |
| freighter/ts     | 9      | 3    | 33%   |
| drift            | 20     | 6    | 30%   |
| pluto            | 692    | 121  | 17%   |
| alamos/ts        | 8      | 1    | 13%   |
| **console**      | 724    | 42   | **6%** |
| **x/media**      | 5      | 0    | **0%** |

### C++

| Component    | Source | Test | Ratio |
| ------------ | ------ | ---- | ----- |
| freighter    | 4      | 2    | 50%   |
| x/cpp        | 84     | 35   | 42%   |
| driver       | 171    | 65   | 38%   |
| **client**   | 58     | 15   | **26%** |

### Python

| Package        | Source | Test |
| -------------- | ------ | ---- |
| client/py      | 222    | 32   |
| freighter/py   | 12     | 4    |
| alamos/py      | 8      | 2    |
| **integration**| 96     | **0** |

## Cross-Cutting Themes

**The Console is dramatically undertested.** A 6% spec ratio on the primary
user-facing application is the single largest exposure in the codebase. Hardware
integrations and Redux state are the two worst sub-areas.

**Public API contracts lack unit tests.** Both the Go server's HTTP/gRPC API
layer (`core/pkg/api/*`) and the C++ client's transport/framer have low
coverage, despite being the boundary contracts other clients depend on.

**Device factories are a systematic gap.** The driver's factory pattern is
the entry point for every hardware integration, and not one of the seven has
a test. A factory bug bricks the integration before any pipeline test could
catch it.

**Test infrastructure isn't tested.** The `/integration/framework/` conductor
(~3.5k LOC) and the C++ `task::Manager` lifecycle have minimal coverage. When
the harness has bugs, every test on top of it is suspect.

**Weak error assertions in Go.** ~200 instances of `Expect(err).To(HaveOccurred())`
or `ToNot(HaveOccurred())` violate Rule 8 in `docs/claude/toolchains/go.md`.
These should be `MustSucceed` (success) or `MatchError(<sentinel>)` (failure)
to actually pin the error type.

## Priority Recommendations

### P0: Console Redux slices

`/console/src/{workspace,layout,range,status,table,arc,docs,version}/slice.ts`

Slices are pure reducers — easy to test, high-value. Each should get a
`slice.spec.ts` covering action handlers, edge cases (empty/undefined state),
and any selectors. Estimate: ~2–3 hours per slice.

### P0: Core API layer (Go)

`core/pkg/api/{access,auth,channel,ranger}/*.go`

These packages compose service-layer calls into the HTTP/gRPC contract.
Suggested approach: BDD-style suite per package using a fake distribution
layer; cover happy path, validation errors, and auth failures. The ranger
package (251 LOC) and channel package (389 LOC) are the largest — start
there.

### P0: Driver factories (C++)

`/driver/{opc,ni,labjack,modbus}/factory.cpp` plus `/driver/rack/factories.cpp`

Factory tests don't need real hardware — they exercise task configuration
parsing and dispatch. Each factory should verify: (a) accepts its own task
type, (b) rejects others (returns `{nullptr, false}`), (c) propagates
configuration errors clearly.

### P1: Console hardware integrations

`/console/src/hardware/{ni,modbus,opc,labjack,ethercat,http}/`

226 files is a lot — focus first on `common/task/fields/` (shared form field
logic) and `*/device/services/` (device discovery/CRUD). UI components can
follow with React Testing Library. Aim for spec coverage on validation logic
and form-state reducers first; visual components last.

### P1: C++ client framer

`/client/cpp/framer/{codec,streamer,writer}.cpp` and `/client/cpp/transport.cpp`

These are protocol code — test them against captured frames or a mock
freighter. Required for any confidence in C++ client compatibility.

### P1: Integration framework

`/integration/framework/{test_conductor,test_case,execution_client,config_client}.py`

The conductor parses JSON sequence files, schedules cases, and reports
results. Bugs here silently corrupt every integration run. Add `pytest` unit
tests covering: sequence parsing, case filtering (1-part, 2-part, 3-part
target syntax), pool sizing, and failure aggregation.

### P2: LSP protocol (Go)

`/x/go/lsp/protocol/server.go` (~1900 LOC, zero tests)

The Arc LSP depends on this. Generated protocol code is hard to test
end-to-end; focus on hand-written request/response routing and the
`server.go` dispatch logic.

### P2: Pluto Aether/canvas layer

`/pluto/src/vis/*`, `/pluto/src/lineplot/aether/*`

Worker-thread rendering is genuinely hard to unit test. Recommend leaning on
Playwright-based visual regression tests via `/integration/console/` rather
than expanding Vitest coverage here. New aether code should at minimum get
unit tests on the non-rendering helpers (axis tick generation,
range/zoom math, cursor coordinate translation).

### P3: Hygiene — fix weak error assertions

200 occurrences of `HaveOccurred()` across Go test files. A grep-and-fix pass
(largest concentrations: `x/go` 89, `core` 55, `cesium` 20) would bring the
existing test corpus into compliance with project conventions and make
regressions catch the right error rather than any error.

## What's Working Well

- **Go core libraries** (`cesium` 96%, `arc/go` 97%, `x/go` 88%) are
  well-covered.
- **No `.skip` / `.only` / `FIt` leftovers** anywhere in the codebase — the
  test suites that exist actually run.
- **Console Playwright integration tests** in `/integration/console/` cover
  many golden paths (plots, schematics, tasks, workspaces) that would be
  hard to unit test through the Aether worker layer.
- **Cesium's BDD parameterization** across filesystems is a model worth
  imitating in other storage-adjacent code.
