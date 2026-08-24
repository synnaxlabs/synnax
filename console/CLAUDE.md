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
   Frameworks (tree, mosaic, panel, palette, link, import/export, modals) and
   cross-domain-shared widgets/hooks. Never imports `feature/`.
3. **`feature/`** — isolated leaves: a domain's full widget (renderer, toolbar,
   controls, `useCreate`) plus its ontology/palette/link glue. Imports platform +
   session, **never a sibling feature** — cross-domain wiring bottoms out in `session/`,
   the client SDK, or `platform/panel`.
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
is load-bearing: `App.tsx` sits above the crash screen, `Session.SettledProvider`
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
`Drift.closeWindow`, `Drift.setWindowProps`. Keys must be unique, and `useOpenWindow`
mints a fresh one per open. Drift keeps invisible pre-render windows in the background
and claims one on `createWindow`, so new windows appear instantly.

## State Management

Modular slices (`core`, `nav`, `panels`, `lineplot`, `schematic`, `table`, `project`,
`drift`, ...), each with `SLICE_NAME`, `sliceStateZ`, `SliceState`, `ZERO_SLICE_STATE`,
and `createSlice` reducers. Side effects go in middleware.

**A Core record's key is opaque** — `LOCAL`, `DEMO`, `SERVED`, a UUID for the rest — so
editing an address never moves an entry. Stored state is partitioned by the **cluster
key** the record caches on connect, so two records reaching one cluster share its state.
That key is local, so deep links name the cluster instead
(`synnax://cluster/<cluster-key>/<type>/<key>`) and resolve to whatever record this
machine reaches it through.

### Persistence (`session/persist/`)

Main window only, 250ms debounce. Every slice is declared in exactly one scope in
`PERSIST_SCOPES`, alongside the schema its stored bytes are parsed through:

| Scope       | Slices                                            |
| ----------- | ------------------------------------------------- |
| `global`    | core, color, theme                                |
| `core`      | project                                           |
| `project`   | drift, panelOrder, range, status                  |
| `window`    | arc, lineplot, log, nav, panels, schematic, table |
| `transient` | haul, persist — never written                     |

`Persist.open` throws when a slice is in none of them, so a new slice forces a decision
about its durability.

A `window`-scoped slice gets one partition per window
(`window.<clusterKey>.<projectKey>.<windowKey>`) holding the slice narrowed to that
window, so its bytes still parse through the slice's own schema.
`session/window/lens.ts` owns the narrow and widen; persistence never reaches into a
slice itself. Closing a window deletes its partition.

Each partition keeps a four-slot ring behind a `.slot` pointer, backing revert. A
partition whose slices did not change is left alone, so the ring holds sessions rather
than the last second of writes.

**Evolve a slice additively, or lose it.** There is no migrator framework: a schema
parses one version and drops silently to the initial state otherwise, an older Console
reading a newer slice included. To reshape or rename, either keep accepting the old
shape or accept the reset — say which in the PR.

### Where it lands

`STORE_NAME` (`"session"`) names both backends, picked by `Runtime.ENGINE`:

- **Tauri** — `session.json` in the **local** app data dir. `TauriKV` names an absolute
  path because the store plugin resolves a relative one against the roaming dir, which
  must not carry machine-local session state.
- **Browser** — IndexedDB database `session`, object store `kv`, scoped to the page's
  origin, so a Console served from two ports is two sessions. Not localStorage: twelve
  state slots outgrow its cap. `localStorage` holds one thing, the deep-link ignore flag
  in `platform/link/markIgnored.ts`; keep it that way.

The 0.56 store (`persisted-state.json` in the roaming dir; the `persisted-state.json:`
localStorage prefix) is read once to seed a fresh install, never written, so a rollback
to 0.56 finds its state.

## Windows Are Viewports

Panels live on the Core and any number of windows may show one at once, so everything
about _how this window looks at a document_ is keyed by window: `nav`, `panels`, and the
per-document view slices (`arc`, `lineplot`, `log`, `schematic`, `table`), all shaped
`windows: Record<windowKey, ...>`.

Build one from `session/window/keyed.ts`: `createWithDocumentHandler` for a per-document
reducer, `createDocumentInitializer` for its create, `createInjectKeyMiddleware` to fill
`windowKey` on dispatch, `selectDocument` so selectors resolve the window themselves.
Window keys are minted fresh per open, so add `extraReducers: Window.handleRemoved` or
the slice keeps an entry for every window ever opened.

## Live-Core Tests

Live-Core specs (Core, flux query paths, user badges, ...) connect to a real Core at
`localhost:9090` through the production query path — no store-poking. Check for a
running Core and start one if missing per "Live-Core Tests" in `docs/claude/testing.md`.

Window-keyed slice specs build their store with `createSliceStore` from
`session/window/testutil.ts` — it adds the drift slice the selectors read the window
from and runs the slice's middleware. `inWindow` and `documentIn` write and read one
window's documents.
