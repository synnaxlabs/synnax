# Arc

DSL for hardware control sequences: reactive event-driven stages, channel-based
communication, stateful variables. Compiles to WebAssembly; runtimes provide host
functions. Language spec: `arc/docs/spec.md`.

## Layout (multi-language)

- `arc/go` — parser → analyzer → compiler pipeline, LSP, formatter, STL, Go runtime.
  Entry: `arc.CompileText`/`arc.CompileGraph` against a caller-built root scope
  (`symbol.NewRoot`).
- `arc/cpp` — embedded runtime (WASM execution, scheduler, state) used by the driver's
  Arc task (`driver/arc/`).
- `arc/ts` — `@synnaxlabs/arc`: grammar/syntax utilities for editors.

## Language Gotchas

- No `true`/`false` keywords — use `1`/`0`.
- Stage flows run concurrently each cycle; no same-tick read-after-write; first truthy
  `=>` transition in line order wins; stage entry ignores pre-activation channel writes.
  Details + test patterns: `docs/claude/integration-test.md`.
