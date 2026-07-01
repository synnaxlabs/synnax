# Console Application

The Console is a cross-platform desktop application built with Tauri, React, and
TypeScript. It provides a drag-and-drop interface for building custom control and
monitoring dashboards.

## Technology Stack

- **Tauri 2.8+** (Rust backend) + **React 19** (frontend)
- **Redux Toolkit** for state management
- **Drift** for multi-window state synchronization
- **Pluto** for high-performance visualization components
- **Vite** for development and building

## Layered Architecture

**Status: mid-migration** (branch
`sy-4443-refactor-remaining-console-slices-into-layered-architecture`). Some domains
below follow these rules exactly (`table`, `schematic`, `lineplot`, `log`, `arc`,
`task`/`device`/`rack`); most others are still on the pre-refactor structure and will
not type-check cleanly until migrated. Do not assume an unmigrated domain follows these
rules — check its actual imports first.

Console's source (`console/src/`) is organized into four layers with a **strict,
one-directional import order**:

```
1. session/    lowest  — pure Redux state
2. component/          — UI + self-instantiation
3. service/            — cross-cutting app integration
4. app/        highest — composition root + app shell
```

**A layer may only import from a layer with a strictly lower number.** `component/` must
never import from `service/`. `session/` must never import from `component/` or
`service/`. This is a hard rule.

Within a single layer, any domain folder may import any other domain folder in that same
layer (`service/table` importing `service/group`, `service/link`, `service/ontology`;
`component/table` importing `component/cluster`, `component/project`), **as long as it
does not create a circular dependency**. If two domains in the same layer end up
depending on each other, that is a signal to reconsider which layer the code belongs in
— not something to route around with a lazy import or a type-only import.

### What belongs in each layer

**`session/<domain>/`** — Redux only: zod-typed state schema, `createSlice`
reducers/actions, selectors, persistence purge helpers (`PERSIST_EXCLUDE`). No React, no
knowledge of `component/` or `service/`.

**`component/<domain>/`** — the domain's actual rendered UI, plus the minimal
state-writing needed to instantiate itself into the layout: `layout.ts` (`LAYOUT_TYPE`

- the `create()` layout-placer, which does dispatch into `session/`) and `useCreate.ts`.
  The dividing question for anything ambiguous: **does this code render/spawn the
  domain's own widget?** If yes, `component/`.

**`service/<domain>/`** — everything that plugs the domain into some other
cross-cutting, app-wide system: the ontology tree (`ontology.tsx` — context menus,
select/drag-drop handlers), the command palette (`palette.tsx`), the deep-link router
(`link.ts`), import/export (`import.ts`/`export.ts`, including versioned state
migrations for legacy exports). The dividing question: **does this code hook the domain
into the rest of the app?** If yes, `service/`. `service/` may freely import from
`component/` (its own domain's and others') and `session/` to do this — e.g.
`service/table/ontology.tsx` pulls in `component/cluster`, `component/context-menu`, and
`component/table` to build its tree context menu.

**`app/`** — two things only: (1) pure aggregation files (`commands.ts`, `links.ts`,
`extractors.ts`, `ingesters.ts`, `services.tsx`) that each collect every domain's
`COMMANDS` / `useLinks` / `EXTRACTORS` / `FILE_INGESTERS` / `ONTOLOGY_SERVICE` exports
into one global registry, and (2) the small set of extremely high-level, effectively
singleton shell components — the main app entry, the mosaic host, the top/aux nav bars,
the notification host. `app/` owns the _arrangement_ of that chrome; any actual
domain-flavored widget rendered inside it (e.g. a cluster-connection badge in the nav
bar) still lives in that domain's own `component/`, not in `app/`.

### Not every domain needs every layer

A domain only gets the layers it actually needs — a domain with no persisted state skips
`session/`, a domain with nothing to hook into the app skips `service/`. Nothing is
required to exist in all four layers.

### Framework domains vs. product domains

Some folders aren't product domains at all — they're the framework other domains plug
into: `ontology` (defines the `Service` type + `NOOP_SERVICE` every domain's
`ONTOLOGY_SERVICE` implements), `link`, `layout`, `export`, `import`, `palette`,
`persist`, `access`. The same pattern repeats one level down inside hardware:
`task`/`device`/`rack` are framework domains that the vendor integrations
(`ni`/`opc`/`modbus`/`labjack`/`ethercat`/`http`/`pagerduty`) plug into — e.g.
`service/task/layouts.ts` merges every vendor's `ZERO_LAYOUTS` map into one dispatch
table (`...NI.Task.ZERO_LAYOUTS, ...Modbus.Task.ZERO_LAYOUTS, ...`), and
`component/task/` holds the generic, vendor-agnostic form shell every vendor's task UI
renders into.

Framework domains and product domains live **flat, side by side**, in the same layer —
there is no `core/` or similar subfolder to separate them structurally. Consumers
already see the distinction at the import site (`Service.Ontology` vs `Service.Table`).

### Barrel convention

Every domain folder that has enough surface area pairs `index.ts` with `external.ts`:

```typescript
// index.ts
export * as Table from "@/service/table/external";

// external.ts
export * from "@/service/table/ontology";
export * from "@/service/table/palette";
// ...
```

This is the default for every domain in every layer it touches. Deviate only for a
specific, deliberate reason (e.g. a domain with a single trivial file).

### Why `service/` looks inconsistent right now

`service/` is not a new folder invented for this refactor — it predates it by years and
used to hold a domain's entire slice (state, logic, and UI all mixed together). The
migration works by moving files **out of** `service/<domain>/` into new
`session/<domain>/` and `component/<domain>/` folders, leaving behind only what's
genuinely layer-3. An unmigrated domain (e.g. `ni`) still has full form components
sitting under `service/ni/task/*.tsx` with imports pointing at deleted paths like
`@/hardware/ni/...` — that's expected mid-migration state, not a bug to route around.

## Testing

When writing or editing any console unit test (`console/src/**/*.spec.ts[x]`), follow
the **`console-testing` skill** — it carries the full rules with correct/incorrect
examples. The four highest-value ones:

- **Test through the public namespace, never a domain's internal file** — including the
  thing under test itself (`Nav.Bar.Top`, not `@/app/nav/bar/Top`). The only exemption
  is a co-located `testutil` file. This is blackbox testing, the same discipline as Go
  `package foo_test`.
- **Prefer real infrastructure over mocks.** Default to a real client
  (`createTestClient`)
  - real flux stores via `@/testutil` for anything touching data; drop to a preloaded
    store for pure logic; inject spies with `vi.fn()` (that's DI, not a mock); reach for
    `vi.mock` only for unmockable runtime seams.
- **Never stub DOM primitives** (`getBoundingClientRect`, canvas) **or substitute a
  child component** with a placeholder. Render the real thing.
- **One home per shared helper.** Hoist to the nearest shared `testutil` on the second
  use; never copy scaffolding between spec files. Cross-package helpers must be
  vitest-free.

## Development Modes

### Tauri Development Mode

Full Tauri application with Rust backend:

```bash
pnpm dev:console
```

- Uses Tauri's development server
- Hot reload for both frontend and backend
- Native window management
- Access to Tauri APIs (fs, window, etc.)

### Vite-Only Mode

Frontend-only development without Tauri:

```bash
pnpm dev:console-vite
```

- Faster startup and reload
- Good for UI development
- No Tauri APIs available
- Single window only

## Multi-Window Architecture (Drift)

Console uses **Drift** to synchronize Redux state across multiple windows.

### How It Works

1. **Main Window Authority**: Main window holds authoritative state
2. **Child Windows**: Request initial state from main on startup
3. **Action Propagation**: All actions emitted to all windows via Tauri IPC
4. **Synchronized State**: Every window maintains identical Redux state

### Window Management

Windows are managed declaratively via Redux actions:

```typescript
// Create a new window
dispatch(
  Drift.createWindow({
    key: "schematic-1",
    type: "schematic",
    loc: "mosaic", // Where to render in layout
  }),
);

// Close a window
dispatch(Drift.closeWindow({ key: "schematic-1" }));

// Update window properties
dispatch(
  Drift.setWindowProps({
    key: "schematic-1",
    props: { title: "New Title", width: 800 },
  }),
);
```

### Pre-rendering Optimization

Drift creates invisible "pre-render" windows in the background:

- Main window creates pre-rendered windows on startup
- When `createWindow` is called, Drift reuses a pre-render window
- Makes new windows appear instantly (no React bootstrap delay)
- Configurable via `enablePrerender: true` option

### State Synchronization Flow

```
User Action in Window A
  ↓
Dispatch to local Redux store
  ↓
Middleware intercepts action
  ↓
Update local state
  ↓
Emit action to all windows (Tauri IPC: drift://action)
  ↓
Windows B, C, D receive action via event listener
  ↓
Each window updates its Redux store
  ↓
All windows now have synchronized state
```

### Mutex Protection

Drift uses `async-mutex` to prevent race conditions:

- Ensures ordered window operations
- Prevents concurrent window property updates
- Guarantees consistency across windows

## State Management

### Store Structure

Redux store uses modular slices:

```typescript
const store = {
  cluster: ClusterState, // Cluster connections
  layout: LayoutState, // Window layouts (mosaic)
  linePlot: LinePlotState, // Line plot visualizations
  schematic: SchematicState, // Schematic editor
  table: TableState, // Table views
  workspace: WorkspaceState, // Workspace management
  drift: DriftState, // Window state
  // ... more slices
};
```

### Slice Pattern

Each feature has its own slice:

```typescript
// cluster/slice.ts
export const SLICE_NAME = "cluster";

export interface SliceState {
  clusters: Record<string, Cluster>;
  activeCluster: string | null;
}

export const ZERO_SLICE_STATE: SliceState = {
  clusters: {},
  activeCluster: null,
};

const slice = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    add: (state, action) => {
      /* ... */
    },
    remove: (state, action) => {
      /* ... */
    },
    setActive: (state, action) => {
      /* ... */
    },
  },
});
```

### State Persistence

Only the **main window** persists state to disk:

- **Storage**: File-based via Tauri `fs` APIs (JSON format)
- **Location**: User data directory
- **Versioning**: Keeps last 4 versions for rollback
- **Migration**: Automatic migration from v1 (binary) to v2 (JSON)
- **Debouncing**: 250ms debounce to minimize disk I/O
- **Selective Persistence**: Excludes transient state (themes, nav drawers, hauling)

```typescript
const PERSIST_EXCLUDE = [
  "layout.**.nav", // Navigation drawer state
  "layout.**.hauling", // Drag-and-drop state
  "palette.activeTheme", // Active theme
];
```

## Layout System (Mosaic Pattern)

Console uses a **mosaic tree** structure for tab layouts:

### Mosaic Tree

```typescript
type MosaicNode = {
  key: string;
  tabs?: Tab[]; // Leaf node with tabs
  first?: MosaicNode; // Split node - first child
  second?: MosaicNode; // Split node - second child
  direction?: "row" | "column";
  size?: number; // Split ratio
};
```

### Tab Management

- Tabs can contain any visualization (line plot, schematic, table, etc.)
- Tabs can be moved between windows via `moveMosaicTab` action
- Drag-and-drop to rearrange tabs and splits
- Each window has its own independent mosaic

### Navigation Drawer

Per-window navigation drawer:

```typescript
type NavDrawerState = {
  activeItem: string | null;
  hoveredItem: string | null;
  expanded: boolean;
};
```

## Workspace Management

Workspaces are saved layouts that can be quickly switched:

```typescript
dispatch(
  Layout.setWorkspace({
    key: "workspace-1",
    name: "Telemetry Dashboard",
    layout: mosaicTree,
  }),
);
```

- Preserves window configurations
- Saved to disk with state persistence
- Can export/import workspaces

## Common Patterns

### Window Lifecycle Hooks

```typescript
import { useWindowLifecycle } from "@synnaxlabs/drift/react";

useWindowLifecycle({
  key: "my-window",
  onMount: () => {
    // Window created
  },
  onUnmount: () => {
    // Window destroyed
  },
});
```

### Process Registration

Prevent window closure during long-running operations:

```typescript
dispatch(
  Drift.registerProcess({
    windowKey: "schematic-1",
    processKey: "saving",
    blocking: true,
  }),
);

// Later...
dispatch(
  Drift.unregisterProcess({
    windowKey: "schematic-1",
    processKey: "saving",
  }),
);
```

### Selecting Window State

```typescript
import { useSelectWindow } from "@synnaxlabs/drift/react";

const MyComponent = () => {
  const window = useSelectWindow("my-window");
  return <div>Window: {window.title}</div>;
};
```

## Common Gotchas

- **Two dev modes**: `dev:console` (Tauri) vs `dev:console-vite` (frontend only)
- **Main window only persists**: Child windows don't save state to disk
- **Pre-rendering**: Hidden windows created in background for instant display
- **Window keys**: Must be unique across all windows
- **Mosaic state**: Per-window, not global
- **Action emitter metadata**: Actions carry `emitter` to prevent circular propagation
- **Mutex blocking**: Window operations are serialized to prevent races

## Development Best Practices

- **Use Vite mode for UI work**: Faster iteration without Tauri overhead
- **Use Tauri mode for integration**: Test full window management and IPC
- **Absolute imports**: Use `@/` prefix (configured in tsconfig)
- **Slice isolation**: Keep slices focused and independent
- **Middleware for side effects**: Use Redux middleware for async operations
- **Window cleanup**: Always unregister processes before closing
- **State migrations**: Add migration logic when changing state shape
