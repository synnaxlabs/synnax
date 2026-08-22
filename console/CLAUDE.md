TypeScript development rules: @../docs/claude/toolchains/typescript.md

# Console Application

Cross-platform desktop app: Tauri 2.8+ (Rust) + React 19 + TypeScript, Redux Toolkit for
state, Drift for multi-window sync, Pluto for visualization, Vite for dev/build.
Drag-and-drop mosaic dashboards.

## Layered Architecture (`console/src/`)

Four strictly-ordered layers; every domain (schematic, range, task, ...) is split across
them. A layer imports only layers below it — never above:

1. **`session/`** (lowest) — Redux state: slices, selectors, persistence, plus the
   synchronizer hooks that keep session state consistent with the Core. No components or
   rendering.
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

## Mounting Side Effects

Hotkey handlers, synchronizers and window catchers render nothing but must be mounted to
run. Nothing fails loudly when one is dropped, so placement follows three rules.

**Effects that act on one document mount with that document.** Pluto's trigger provider
fires every registered callback with no arbitration, and the mosaic keeps background
tabs mounted, so hidden instances would listen too. Focus is handled for you: every
tab's content mounts inside a `Triggers.Scope` (`feature/panel/Mosaic.tsx`) that
switches triggers off for background tabs and while a modal is open. Never rebuild that
predicate in a component. A component passes `enabled` (or `enableTriggers`) only for
conditions it alone owns, like whether its content is editable.

**Everything else mounts as high in the tree as its dependencies allow, inside a named
`SideEffect` component.** Never a bare hook call in a component that draws — that is
exactly how the line plot hold trigger was silently dropped for five days. Tree position
is load-bearing: `App.tsx` sits above the crash screen, `Session.Settled.Provider`
outlives what it repairs, `ProjectSideEffect` needs a selected project.

**A hotkey spec renders the component that owns the mount**, never `renderHook` on the
effect itself. `app/nav/bar/Top.spec.tsx` is the pattern.

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

Modular slices (`core`, `nav`, `panels`, `lineplot`, `schematic`, `table`, `project`,
`drift`, ...), each with `SLICE_NAME`, `sliceStateZ`, `SliceState`, `ZERO_SLICE_STATE`,
and `createSlice` reducers. Keep slices focused and independent; side effects go in
Redux middleware.

Every persisted slice carries `version: z.literal(N)` in its `sliceStateZ`. Schema
evolution is Zod's job: widen the schema, bump the literal, and give absent fields
defaults. There is no migrator framework — a stored slice that fails its schema falls
back to its initial state.

**Persistence** (`session/persist/`) — main window only, 250ms debounce. Desktop writes
`session.json` in the Tauri app data dir; the browser uses an IndexedDB database. Each
slice lives in exactly one scope, declared with its schema in `PERSIST_SCOPES`:

| Scope     | Slices                                                                  |
| --------- | ----------------------------------------------------------------------- |
| `global`  | core, color, theme                                                      |
| `core`    | project                                                                 |
| `project` | arc, drift, lineplot, log, nav, panels, range, schematic, status, table |

`haul` and `persist` are declared `transient` and never written. `Persist.open` throws
when a slice is in none of the four, so adding a slice forces a decision about its
durability. Each partition keeps a four-slot ring behind a `.slot` pointer, backing
revert. Switching Core or project flushes the outgoing partitions and hydrates the
target's without a reload.

`persisted-state.json` is the 0.56 store. It is read once, to seed a fresh install, and
never written or cleared — a rollback to 0.56 must still find its state.

**Cores are keyed by `host:port`.** The key is the address, so it never changes
underneath a running session and two entries at one address collapse into one.

## Windows Are Viewports

Panel documents live on the Core; a window is a view onto them, and any number of
windows may show one panel at once. So everything about _how this window looks at a
document_ is keyed by window key: `nav`, `panels`, and the per-document view slices
(`arc`, `lineplot`, `log`, `schematic`, `table`), all shaped
`windows: Record<windowKey, ...>`.

Building one: take the schema helpers from `session/window/keyed.ts` —
`createWithDocumentHandler` for a per-document reducer, `createDocumentInitializer` for
its create, `createInjectKeyMiddleware` to fill `windowKey` on dispatch, and
`selectDocument` so selectors resolve the current window themselves. Add
`extraReducers: Window.handleRemoved` or the slice keeps an entry for every window ever
opened — window keys are minted fresh per open.

## Live-Core Tests

Live-Core specs (Core, flux query paths, user badges, ...) connect to a real Core at
`localhost:9090` through the production query path — no store-poking. Check for a
running Core and start one if missing per "Live-Core Tests" in `docs/claude/testing.md`.

Specs over a window-keyed slice build their store with `createSliceStore` from
`session/window/testutil.ts`, which adds the drift slice the selectors read the current
window from and runs the slice's own middleware. `inWindow` and `documentIn` write and
read one window's documents.
