# Panels Architecture Decisions (SY-4193)

Status -> Decided, pre-implementation. Companion to
`0032-260329-workspace-reorg-scratch.md`. This doc records the data-model and
architecture decisions for the panels work on `sy-4193-panels`. The scratch RFC is
useful context, not gospel; where they disagree, this doc wins.

## The problem

Production Synnax runs all layouting through the Redux layout slice and placer. The
panels work introduced a second, Pluto/Flux/server-backed system that also owns
arrangement and content. Two systems claiming the same responsibility, gated by a
per-window cursor, is a split brain: two render pipelines, two placement APIs, two
identity models, two persistence regimes, with no convergence.

The fix is not "remove Redux." Redux holding session state and core holding shared
document state is the correct separation. The split brain is only where that boundary is
drawn wrong, duplicated, or violated. The work is to classify every piece of layout
state as session or document, put it in the right tier, and collapse the duplicated
pipelines into one.

## The keystone: two-layer model

Tiling arrangement is shared; which panels you have open is personal.

- Inner layer (a panel's mosaic: tiling, splits, sizes, tab content) -> shared document,
  stored in core. One operator configures a panel, every operator sees that arrangement.
- Outer layer (which panels are open, which is active, where) -> per-user session, in
  the narrowed Redux layout slice.

This gives shared layouts with per-operator flexibility, and it forces the rest of the
model: if a panel is shared and operator B must render operator A's plot, the plot's
config cannot live in A's Redux. It must be a shared core resource.

## Decisions

1. **Tab content identity is a union.** A tab references either
   `{ resource: ontology.ID }` (content that has its own core document, e.g. line plot,
   schematic) or `{ view: { type, args } }` (self-describing app-views/tools with no
   backing document, e.g. docs, explorers, about, the picker). One renderer registry
   keyed by a type string serves both arms. Restores the expressiveness the legacy
   `type + args` model had while keeping data content unified on `ontology.ID`.

2. **Visualization config lives in core; ephemeral view state is session.** All viz
   config (channels, axes, ranges, rules, cells, nodes) is a core document keyed by
   ontology key. Ephemeral view state (viewport pan/zoom, selection, hover) is session,
   keyed per tab-instance, never shared, never blocks render. Forced by the keystone:
   shared panels require shared config. Table and schematic are already moved; line plot
   and log follow. These types do not carry Redux args, so this is not a blocker.

3. **One render registry, self-loading, Console-owned, injected into Pluto.** A single
   registry keyed by type string resolves both union arms. Resource renderers receive
   the `ontology.ID` and self-load config from Flux; view renderers receive inline
   `args`. The registry lives in Console (views are Console concepts that cannot live in
   Pluto) and is injected into the generic Pluto panel host through a content-renderer
   interface. Dependency injection, not a layering tether: Pluto stays domain-agnostic,
   Console supplies meaning. Lets each viz type self-load as its config lands in core
   without the panel host knowing.

4. **Placement stays one router, panel vs overlay.** `usePlacer` remains the single
   placement entry point. It routes by destination: document content -> dispatch into
   the active panel (core); modal/window -> session overlay in the narrowed layout
   slice. The rip-out removes only the placer's legacy-mosaic arm (the branch that wrote
   into `state.mosaics`). No call-site migration to new verbs.

5. **The layout slice narrows to a pure session store.** Once mosaic tabs move to
   panels, the only things left in `state.layouts` are modals and window/viewport state,
   which are already session. The slice sheds its document role and becomes
   session-only: modals (stay in the slice, not extracted), window viewports,
   active-panel/active-tab cursors, nav drawer, focus. Focus stops being persisted into
   any document.

6. **One panel model; draft vs project is a flag.** A draft and a project panel are the
   same resource, same schema, same Flux store, same reducer, same dispatch, same
   TypeScript surface. The only difference is ownership: a draft is parented in the
   ontology to its creator; promotion (implicit on rename from "Untitled") reparents it
   to the project. Drafts are scoped per-user (they follow you to any console you log
   into), visible only to their creator until promoted. No second state model anywhere,
   no serialize-on-promote seam. The existing project-scoped listing query already walks
   ontology parents, so "my drafts + this project's panels" falls out of it.

7. **Multi-window: windows are pure viewports; `MOSAIC_WINDOW_TYPE` dies.** A window is
   a session viewport with its own active-panel/active-tab cursor (`windowPanels`).
   Panels are window-agnostic shared documents. Dragging a panel from the tab strip to a
   new window moves the viewport (the new window's cursor points at that panel; the
   document is untouched). Dragging a tab out creates a draft panel from that tab, shown
   in the new window. Move only, no co-view: a panel has exactly one viewing window at a
   time, which kills the two-windows-fighting-over-active-tab problem. The window/mosaic
   lifecycle middleware retires.

8. **View args in core are an opaque blob.** Core stores
   `view: { type, args: <opaque> }` and never interprets `args`; it syncs them as an
   opaque payload. Console owns the meaning and validates on read. Keeps core decoupled
   from Console's view vocabulary; new view types never touch the schema or trigger
   cross-language regen. Dependency: oracle needs an opaque/JSON field type for `args`.

9. **Zero-coexistence cutover.** The Redux mosaic is ripped out completely. No feature
   flag, no dual-rendering, no time-boxed coexistence. We have confidence in the
   end-state, so we commit. The split brain cannot calcify because the legacy path stops
   existing. Prerequisite: every viz type renders from core through the injected
   registry before cutover (largely done per decision 2).

## Parked (non-blocking, solve later)

- Session cursor integrity -> the `windowPanels` cursor can dangle when another client
  deletes the panel/tab it points at. Handle via advisory-cursor (validate on read) or
  reactive repair later. Not a blocker.
- Content resource lifecycle -> working assumption is independent lifecycle (removing a
  tab removes only the reference, never the resource). Reference-counting or
  recent/drafts auto-archive is a later cleanup policy.

## End-state summary

- Document (core, shared) -> panel trees (tiling + tab content refs), viz config as
  ontology resources.
- Session (narrowed Redux layout slice, per-user) -> which panels open where,
  active-panel/tab cursors, modals, window viewports, nav, focus, ephemeral view state.
- One render registry (Console-owned, injected into the Pluto panel host) mounting
  already-core-backed connected components for resources and view components for views.
- One placer routing panel vs overlay. One panel model with a draft/project ownership
  flag. Windows as pure viewports. Redux mosaic deleted.

## Follow-up decisions

- Open-in-panel has two intents -> the default "open / peek" lands in a DRAFT panel
  (personal, close = gone, never mutates shared state); "add to panel" is an explicit
  gesture that composes into the active panel (including shared project panels). The
  placer's document arm routes by intent.
- Draft promotion -> implicit, on first rename away from the default name, to the active
  project of the window doing the rename (client passes the project key). No active
  project -> stays a draft. Explicit "Save to project" exists for the cross-project
  case.
- Focus (Ctrl+L fullscreen one tab) -> kept, as per-window SESSION state (relocated from
  `mosaic.focused` into the session `modalFocus` map). Never persisted to the document.
- Existing-mosaic migration -> PARKED. Eventually migrate an old workspace's mosaic into
  a project panel (workspaces were already saved shared documents, so project panel is
  the right target). Not required for the cutover; the v11 migration just drops
  `state.mosaics` for now. Resources survive regardless.
- Undo/redo + concurrent edits on shared panels -> OUT OF SCOPE. Leave the undoable
  store as-is. Not part of this work.

## Implementation findings (from code research)

Verified against current code on `sy-4193-panels`. File:line citations are the state at
research time; re-verify before editing.

### De-riskers (already exist)

- Opaque view args -> oracle's `record` primitive exists (`oracle/resolution/types.go`)
  and is already used by `schematic`/`table`/`log` schemas. `args record??` generates TS
  `unknown` with faithful JSON round-trip across Go/Py/C++. No generator change.
- Draft-to-user parenting -> `user.OntologyID(key)` exists
  (`core/pkg/service/user/ontology.go:29`) and panel `Writer.Create(ctx, p, parentID)`
  already accepts a parent. The API create handler passes the current subject's user
  ontology ID when no project is given.
- Render seam (TODO #9) -> the existing console `Layout.Renderer` components
  (`LinePlot`, `Table`, `Schematic`, `Log`) already self-load config from core via
  `useLoadRemote`/`useEnsureState` keyed by `layoutKey`, and
  `layoutKey == ontology.key`. The injection seam exists: pluto `panel/Mosaic.tsx`
  exposes a `children(MosaicTabRenderProps)` render-prop. The Console registry maps
  `resource.type -> existing Layout.Renderer` mounted with `layoutKey = resource.key`.

### Assumption correction

- All four viz types (line plot, schematic, table, log) are HYBRID, not "table/schematic
  done." Each has a core Flux resource (server source of truth) AND a per-viz Redux
  slice the renderer reads after hydrating from core. Not a blocker: that working-copy
  slice is keyed by the ontology key, is separate from the layout slice, and the
  load-from-core path already lets one operator render another's panel.
  Pure-Flux-connected components are optional later cleanup, not a cutover prerequisite.

### Schema shape

- Oracle has no native union field type. Model Tab content as two optional fields:
  `resource ontology.ID?` and `view View?` (with
  `View struct { type string; args record?? }`), convention "exactly one set," mirroring
  `Node`'s `leaf??`/`split??`. Add a `SetTabView` action/handler alongside
  `SetTabResource`; each clears the other field.

### Rip-out inventory (layout slice -> pure session)

- Delete reducers -> `moveMosaicTab`, `selectMosaicTab`, `resizeMosaicTab`,
  `splitMosaicNode`. Delete helpers -> `ensureMosaic`, `purgeEmptyMosaics`,
  `reconcileMosaicLayouts`, `tabFromLayout`. Delete `MOSAIC_WINDOW_TYPE` +
  `createMosaicWindow`.
- Narrow `place`/`remove` to the window+modal arms (drop the `location: "mosaic"`
  branches). Valid locations become `"window" | "modal"`.
- Delete `state.mosaics`. New v11 migration drops `mosaics` and moves per-window focus
  into a session `modalFocus` map.
- Selectors -> delete `selectMosaic`/`useSelectMosaic`; rewire `selectFocused` to read
  `modalFocus`; rewire `selectActiveMosaicTabState` to read `windowPanels.activeTab`.
- Render -> delete the `Internal` and `MosaicWindow` components in `layouts/Mosaic.tsx`;
  simplify the gate to panel-only.
- Break sites to fix -> `layout/Menu.tsx` (split menu item), `useOpenInNewWindow`,
  `useDropOutside`, `layouts/external.ts`.

### Multi-window rewire

- Delete middleware effects -> `closeWindowOnEmptyMosaicEffect`,
  `createWindowOnPlaceEffect`, `createWindowsOnSetProjectEffect`,
  `deleteLayoutsOnMosaicCloseEffect`. Keep `injectNavDrawerWindowKey`. Keep
  `closeWindowOnRemoveEffect` only for `location: "window"` overlays.
- Rebuild `useDropOutside`/`useOpenInNewWindow` -> open a viewport window and set its
  `windowPanels` cursor instead of `createMosaicWindow` + `moveMosaicTab`.
- Build net-new -> panel-strip drag to a window (move the viewport cursor) and tab-tear
  (create a draft panel from the tab, open a viewport window on it). Haul gesture infra
  exists (`HAUL_DROP_TYPE` etc.); panel-level tear is new.

### Net-new (build, not rewire)

- Promotion-on-rename reparenting (draft user-parent -> project-parent). Documented in
  the schema, no code path yet.
- Draft parenting in the API create handler.
- The unified Console render registry (resource + view arms) wired into panel
  `TabContent`, replacing the `TODO(#9)` stub.
- View-arm renderers (docs, explorers, about) registered with inline args.
- Panel-tab drag + tab-tear gestures.

### Build order

Sequenced for a compiling branch; ships as one zero-coexistence cutover, not incremental
releases.

0. Schema -> View struct + Tab union + `args record`; regen (user runs oracle); add
   `SetTabView` reducer handler.
1. Render seam -> Console unified registry mounting existing `Layout.Renderer` by type
   with `layoutKey = resource.key`; wire the view arm; remove the `TODO(#9)` stub.
2. Placement -> repoint the `usePlacer` document-content arm to panel
   `insertTab`/`setTabResource`/`setTabView`; keep the modal/window arm.
3. Rip out the Redux mosaic -> apply the rip-out inventory; v11 migration; fix break
   sites.
4. Multi-window -> delete coupling middleware; rebuild `useDropOutside`/
   `useOpenInNewWindow` as viewport windows; build panel drag + tab-tear.
5. Drafts -> API create user-parenting + promotion-on-rename reparent; per-user draft
   scoping in the listing query.
