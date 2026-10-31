TypeScript development rules for this package: @../docs/claude/toolchains/typescript.md

CSS rules: @../docs/claude/toolchains/css.md

## Namespace casing encodes thread safety

- **lowercase** namespaces (`aether`, `flux`, `lineplot` from `@/lineplot/aether`) are
  safe on the aether worker thread: no React, no DOM.
- **PascalCase** namespaces (`Button`, `LinePlot`, `Status`) are main-thread only: they
  use React and/or the DOM. Never import them from worker code.

A feature often ships both: `LinePlot` (React components) and `lineplot` (its aether
worker counterpart). Aether names worker functions like hooks, so the React Compiler and
its lint rules skip every lowercase namespace (`configs/eslint/compiler.ts`). Follow the
casing when adding a module: a hook in a lowercase namespace is never compiled or
linted, and a hook-named worker function in a PascalCase one crashes the worker.

## Live-Core tests

Query/flux specs (e.g. `node/queries.spec.ts`, `task/queries.spec.ts`,
`synnax/aether/provider.spec.ts`) connect to a real Core at `localhost:9090`. Check for
one and start it if missing per "Live-Core Tests" in `docs/claude/testing.md`.
