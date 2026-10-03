TypeScript development rules for this package: @../docs/claude/toolchains/typescript.md

## Namespace casing encodes thread safety

- **lowercase** namespaces (`aether`, `flux`, `lineplot` from `@/lineplot/aether`) are
  safe on the aether worker thread: no React, no DOM.
- **PascalCase** namespaces (`Button`, `LinePlot`, `Status`) are main-thread only: they
  use React and/or the DOM. Never import them from worker code.

A feature often ships both: `LinePlot` (React components) and `lineplot` (its aether
worker counterpart). Follow the casing when adding a module — a worker-safe package
exported as PascalCase (or the reverse) is a defect.

## Live-Core tests

Query/flux specs (e.g. `node/queries.spec.ts`, `task/queries.spec.ts`,
`synnax/aether/provider.spec.ts`) connect to a real Core at `localhost:9090`. Check for
one and start it if missing per "Live-Core Tests" in `docs/claude/testing.md`.
