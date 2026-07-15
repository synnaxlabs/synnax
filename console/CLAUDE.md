TypeScript development rules: @../docs/claude/toolchains/typescript.md

# Console Application

Cross-platform desktop app: Tauri 2.8+ (Rust) + React 19 + TypeScript, Redux Toolkit for
state, Drift for multi-window sync, Pluto for visualization, Vite for dev/build.
Drag-and-drop mosaic dashboards.

## Layered Architecture (`console/src/`)

Four strictly-ordered layers; every domain (schematic, range, task, ...) is split across
them. A layer imports only layers below it — never above:

1. **`session/`** (lowest) — pure Redux: slices, selectors, persistence, migrations. No
   React.
2. **`platform/`** — substrate: high fan-in capabilities other domains depend on.
   Frameworks (ontology, layout, palette, link, export/import, modals) and
   cross-domain-shared widgets/hooks. Never imports `feature/`.
3. **`feature/`** — isolated leaves: a domain's full widget (renderer, toolbar,
   controls, `useCreate`) plus its ontology/palette/link glue. Imports platform +
   session, **never a sibling feature** — cross-domain wiring bottoms out in `session/`,
   the client SDK, or a thin `platform/<domain>/layout.ts` token.
4. **`app/`** (highest) — composition root: aggregates every domain's exports into
   global registries + the shell chrome. No domain logic.

Rules:

- Cross-domain imports within a layer are fine unless circular; a cycle means the
  placement is wrong.
- Not every domain needs every layer.
- Platform vs feature is decided by fan-in: if other domains depend on it, it's
  platform; if it's a leaf, it's feature.
- Barrels everywhere: each domain folder in each layer has `index.ts` doing
  `export * as Domain from "@/<layer>/<domain>/external"`.

## Dev Modes

- `pnpm dev:console` — full Tauri: native windows, Tauri APIs, backend hot reload. Use
  for integration/window-management work.
- `pnpm dev:console-vite` — frontend only: faster, single window, no Tauri APIs. Use for
  UI work.

## Multi-Window Architecture (Drift)

Main window holds authoritative Redux state; child windows request initial state on
startup. Every action is applied locally then emitted to all windows via Tauri IPC
(`drift://action`), so all stores stay identical. Actions carry an `emitter` to prevent
circular propagation; `async-mutex` serializes window operations.

Windows are managed declaratively via Redux: `Drift.createWindow({key, ...})`,
`Drift.closeWindow`, `Drift.setWindowProps`. Window keys must be unique across all
windows.

**Pre-rendering**: Drift keeps invisible pre-render windows in the background and reuses
one on `createWindow`, so new windows appear instantly (`enablePrerender: true`).

**Process registration** blocks window closure during long-running operations:
`Drift.registerProcess({windowKey, processKey, blocking: true})` /
`Drift.unregisterProcess`. Always unregister before closing.

Hooks from `@synnaxlabs/drift/react`: `useWindowLifecycle({key, onMount, onUnmount})`,
`useSelectWindow(key)`.

## State Management

Modular slices (`cluster`, `layout`, `linePlot`, `schematic`, `table`, `workspace`,
`drift`, ...), each with `SLICE_NAME`, `SliceState`, `ZERO_SLICE_STATE`, and
`createSlice` reducers. Keep slices focused and independent; side effects go in Redux
middleware.

**Persistence** — main window only: JSON to the user data dir via Tauri fs, last 4
versions kept, 250ms debounce, automatic v1→v2 migration. Transient state is excluded
(`layout.**.nav`, `layout.**.hauling`, `palette.activeTheme`). Add migration logic when
changing state shape.

## Layout System (Mosaic)

Tab layouts are a mosaic tree: leaf nodes hold `tabs`, split nodes hold
`first`/`second` + `direction` + `size`. Tabs contain any visualization, move between
windows via `moveMosaicTab`, and rearrange via drag-and-drop. **Each window has its own
independent mosaic.** Per-window navigation drawer state (`activeItem`, `hoveredItem`,
`expanded`).

Workspaces are saved layouts (`Layout.setWorkspace`) persisted to disk,
exportable/importable.

## Live-Core Tests

Live-core specs (cluster, flux query paths, user badges, ...) connect to a real core at
`localhost:9090` through the production query path — no store-poking. Check for a
running core and start one if missing per "Live-Core Tests" in `docs/claude/testing.md`.
