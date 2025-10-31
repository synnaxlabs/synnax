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
