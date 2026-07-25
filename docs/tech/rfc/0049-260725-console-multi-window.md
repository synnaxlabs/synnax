# 49 - Console Multi-Window

**Feature Name**: Console Multi-Window Opening and Movement <br /> **Status**: Draft
<br /> **Start Date**: 2026-07-25 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

The panel refactor left the console with working multi-window plumbing and no way to
reach it. Secondary windows render a viewport shell
(`console/src/app/window/Secondary.tsx:24`), per-window session state exists
(`console/src/session/panel/slice.ts:27`), and window arrangement persists per project
through the drift partition (RFC 46 §5.9), but nothing in the UI creates a window: the
only `Drift.createWindow` caller is the context-swap restore path
(`console/src/session/store.ts:195`), and a restored secondary window has no panel
selector, so it renders the empty state.

This RFC restores window creation and tab movement on top of panels. A window is an
unbound viewport: it carries its own panel selector strip and can show any panel in the
project, including one another window is already showing. Windows are created by an
explicit command, from a panel pill, or by tearing a tab or a pill onto the desktop.
Tearing a tab out mints a project panel to hold it. Dragging a tab from one window into
another moves it between the two windows' panels.

# 1 - Motivation

The instrumentation engineer configuring a test and the operator running it both work
across monitors: a schematic on the wall display, a plot on the desk, task configuration
on a laptop. Before panels this was routine — a tab dragged onto the desktop opened a
mosaic window, and `Layout.moveMosaicTab` handled tabs crossing windows
(`console/src/layout/slice.ts:296` on `main`). Panels replaced the per-window Redux
mosaic with a shared server document and deleted both affordances (`useDropOutside.ts`,
`useOpenInNewWindow.ts`, removed in `1dc03e9e46`) because their semantics no longer
held: they moved a tab between two Redux trees, and there is only one tree now, owned by
the cluster.

The consequence today, if we ship as-is: a control room running two displays cannot put
two views side by side at all. That is a regression against the shipped console, not a
missing enhancement.

# 2 - Vocabulary

**Window** — an OS window managed by drift. **Main window** — the window labeled `main`;
owns persistence (`console/src/session/persist/state.ts:105`). **Aux window** — any
other window. **Panel** — a project-owned document holding a tab tree; shared by
everyone with access to the project. **Viewport** — a window's view of one panel; the
binding lives in per-window session state, not in the panel. **Tear-off** — ending a
drag over the desktop rather than over a window.

# 3 - Prior Art

VS Code's auxiliary windows (1.85+) are created by dragging an editor onto the desktop
or via "Move Editor into New Window", and host an editor _group_ — a container — rather
than a bare file; window layout does not survive a restart, a known complaint. Chrome
tears a tab into a new window container and closing the window closes its tabs. Ignition
Perspective Workstation is the operator-station norm and the closest analog to our
deployment: multi-monitor is _configured_, each display assigned a **page**, stable
across restarts.

Our panel is their editor group and their page. We take the container model from all
three, the ad-hoc tear-off gesture from VS Code and Chrome, and beat both on
persistence: a panel is a server document and window arrangement is an L2 partition, so
a torn-off window comes back where it was.

# 4 - Principles

1. **A window is a viewport, never an owner.** No window-scoped document state. Anything
   a window knows about a panel lives in `session/panel`, keyed by drift window key.
2. **Session state is per-window; document state is shared.** Which panel a window shows
   and which tab is selected in each leaf are session; the tree is the panel.
3. **Nothing is created implicitly except where the user's gesture demands a
   container.** Tear-off is the single case: a torn tab must land in some panel.
4. **Closing a window destroys nothing.** Panels outlive the windows that spawned them.
5. **Reuse the proven drag machinery.** Haul mirrors drag state through Redux and drift
   syncs it to every window (`console/src/session/haul/slice.ts:15`, wired at
   `console/src/app/pluto/Context.tsx:31`); target-window drops are ordinary Haul drop
   targets. Only desktop drops need the drag-end interceptor.

# 5 - Design

## 5.0 - The window model

Every window carries a panel selector strip and may select any panel in the active
project. Two windows may show the same panel: both render it, both write through the
same Flux dispatches, and each keeps its own tab selection because `selectedTabs` is
stored per window per panel (`console/src/session/panel/slice.ts:29`). No ownership
bookkeeping, no disabled pills, nothing to reconcile when a window closes.

This supersedes the "move-only, no co-view" half of panels architecture decision #7.
That decision predates the shipped session slice, which already keys selection by window
and therefore already tolerates two windows on one panel.

## 5.1 - The aux window shell

An aux window renders the panel strip, the mosaic, and the bottom visualization drawer
(`Toolbars.BOTTOM`, which is `Panel.TOOLBAR`) — nothing else. No left toolbars, no
palette, no project or cluster chrome. Window controls and the drag region live in a
slim bar alongside the strip, matching the shape `main`'s mosaic windows had
(`console/src/layouts/Mosaic.tsx:398` on `main`).

`Secondary.tsx` grows the strip and the drawer; `Primary.tsx` is unchanged. Both mount
`Triggers.use()`, so tab close, focus, and window close keys work in every window; today
triggers mount only in `Primary`'s `SideEffect`. Per-window nav state already exists
(`console/src/session/nav/slice.ts:38`), so the drawer's collapsed size is remembered
per window with no new state.

## 5.2 - Creating windows

| Surface                 | Gesture                                     | Result                                                 |
| ----------------------- | ------------------------------------------- | ------------------------------------------------------ |
| Palette                 | "Open a new window" (`Panel.COMMANDS`, NEW) | New window showing the spawning window's panel         |
| Panel pill context menu | "Open in New Window"                        | New window showing that panel; source window unchanged |
| Panel pill              | Drag onto the desktop                       | Same as above                                          |
| Tab context menu        | "Move to New Window"                        | Mints a panel holding that tab; new window shows it    |
| Tab                     | Drag onto the desktop                       | Same as above                                          |

All five paths bottom out in one hook, `Panel.useOpenWindow` (NEW, `feature/panel`),
which dispatches `Drift.createWindow({key: id.create(), ...})` and seeds the new
window's session state with `Session.Panel.select({windowKey, key})`. Window keys are
generated, not panel keys: with co-view legal there is nothing to key on, and a
generated key keeps drift's key-collision focus behavior (`drift/src/state.ts:248`) out
of the way.

The window's title tracks the name of the panel it shows, via `Drift.setWindowTitle`
from the strip, so the OS window list is legible.

The tab context menu regains a window item. The 2026-07-21 decision that "window items
are gone for good" was correct against the panel tree as it then stood, where a
tab-level move had no destination; tear-off minting a panel supplies one, and the menu
item is the discoverable twin of an undiscoverable drag.

## 5.3 - Moving tabs between windows

A drag from window A into window B's mosaic is an ordinary Haul drop in B: B's drop
target fires, B knows the payload from the Redux-mirrored dragging state, and drop
indicators are leaf-precise. This is exactly how `main` does it — its handler dispatches
`moveMosaicTab` with B's own window key and the reducer detects the cross-window case
(`console/src/layout/slice.ts:296`). Nothing about that mechanism changes.

What changes is the dispatch. B's selected panel is a different document from A's, so
the drop is a cross-panel move: `insertTab` on B's panel followed by `removeTab` on A's,
the two-dispatch form the schema already specifies
(`client/ts/src/panel/actions.gen.ts:69`). Insert runs first: a failed second dispatch
leaves the tab in both panels, which the user can see and fix, where the reverse order
loses it. Same-panel drops keep using `moveTab`.

To know A's panel, the haul item carries it. `Mosaic.createTabDropHaulItem` has an
unused data slot (`pluto/src/mosaic/haul.ts:20`), and `Panel.Mosaic` fills it with the
source panel key; the tab's content is then read from the client cache
(`client.panels.getCached`) to build the `NewTab` for the insert.

Dropping into a window with no panel selected is rejected — the drag returns — rather
than minting a panel. Tear-off is the only implicit-creation gesture.

## 5.4 - Tear-off

Ending a drag over the desktop is the one case with no drop target, and it is resolved
by the interceptor `useDropOutside` used to own. Both halves come back unchanged in
mechanism: `Haul.bind` on Windows and Linux, where `dragend` reports the final screen
position, and the Tauri `mouse_up` event on macOS, which is still emitted today with no
listener (`console/src-tauri/src/main.rs:122`). "Outside" is still computed from drift's
window boxes (`Drift.selectWindows`, each with `position` and `size`).

One correction to the old implementation. The macOS event is broadcast to every window,
so every window's listener sees the same shared dragging state and would each mint a
window; the old hook dodged this by acting only when the source layout lived in the main
window, which is why tear-off from an aux window never worked on macOS. Instead,
`setHauled` gets window-keyed through the existing middleware
(`Window.createInjectKeyMiddleware([setHauled])`,
`console/src/session/window/keyed.ts:90`) and each listener acts only on its own drags.
Tear-off then works from any window.

A torn tab lands in a freshly created panel named after the tab, parented to the active
project exactly as the strip's create button does (`feature/panel/Selector.tsx:113`),
and the tab moves there by the same insert-then-remove pair as §5.3. A torn pill mints
nothing; it opens a second window on the panel and leaves the source window alone.

## 5.5 - Lifecycle

Closing a window closes nothing else: its panel stays in the strip, reachable from any
window and deleted only through the pill's existing Delete item. A torn-off panel that
outlives its window is ordinary project clutter, not a special case.

Closed windows do leave session garbage, since `panels.windows` and `nav.windows` are
keyed by window key. Both are pruned when a window closes. Window keys are stable across
a restart because `swapDriftWindows` reopens stored windows with their stored props
(`console/src/session/store.ts:191`), so a restored window keeps its panel selection and
its drawer sizing.

No persisted-state migration is required: `panels.windows`, `nav.windows`, and the drift
slice already exist and already persist in the L2 partition.

# 6 - Implementation

One PR. The aux window shell (strip, mosaic, visualization drawer, window controls,
triggers), `Panel.useOpenWindow` and the four creation surfaces feeding it, window
titles, session pruning on close, the source panel key on the haul item, the cross-panel
move in `Panel.Mosaic`'s drop handler, window-keying `setHauled`, and the drag-end
interceptor for tab and pill tear-off.

Creation and movement are separable in principle, but a shell-only landing ships windows
that cannot exchange tabs and a movement-only landing has no second window to drag into,
so neither half is worth reviewing or bisecting alone. The diff is bounded: two shell
files, one hook, two menu items, one drop handler, one interceptor, one middleware
registration.

Integration coverage lands with it in `integration/tests/console/`, whose
`layout/mosaic_operations.py` on `main` is the precedent for driving two windows. No
persisted-state migration, per §5.5.

# 7 - What This RFC Does Not Cover

- **Per-monitor assignment.** Perspective-style "this display always shows this page" is
  a configuration model on top of this one; window arrangement persisting per project is
  as far as we go.
- **Detaching a single tab into a chrome-less window.** Every window is a panel
  viewport.
- **Cross-project windows.** A window shows a panel of the active project; project
  switch swaps every window's context per RFC 46 §5.9.
- **Undo across a cross-panel move.** See open questions.
- **`console/CLAUDE.md`'s stale "Layout System (Mosaic)" section**, which still
  describes per-window Redux mosaics and workspaces; it needs rewriting independent of
  this work.

# 8 - Resolved Decisions

**A window is not bound to a panel.** Rejected: binding each window to one panel and
keying the drift window with the panel key, which would have made drift's key-collision
focus behavior enforce single-viewer for free. Downsides that killed it: an aux window
would have no way to change what it shows, the strip would be main-window-only chrome,
and "open this panel" would have to first evict it from whatever window held it. The
trade is real — we give up structural enforcement and accept that two windows can render
the same expensive schematic twice.

**Co-view is legal**, superseding decision #7. Rejected: forbidding it by graying out
pills another window holds and focusing that window instead. That needs an ownership map
in session state, a release path on every window close, and a conflict rule when two
restored windows both claim a panel — bookkeeping in exchange for preventing something
the user may legitimately want.

**Tear-off mints a project panel.** Rejected: reusing `isOverlaid` so the torn window
shows the source panel focused on that tab, minting nothing. The tab would keep living
in its original panel and render in both windows — two renderers, two aether workers,
doubled telemetry for one plot — and a "move" that leaves the tab where it was is not a
move. The trade is real: idle tear-offs leave pills in a shared strip that teammates
see.

**A cross-window drop moves the tab**, mutating the source panel for everyone watching
it. Rejected: adding to the target while leaving the source intact, which is not what a
drag means anywhere else.

**Torn-off panels persist when their window closes.** Rejected: folding back — returning
the tab to its origin panel and deleting the minted one when it is untouched and
unrenamed. That buys a tidy strip with a heuristic and a destructive close path, and the
heuristic fails the moment someone adds a second tab.

**The aux shell is trimmed**, not full parity. Rejected: rendering `Primary` in every
window, which is less code and more predictable, at the cost of mounting every toolbar's
queries in a window that exists to show one schematic.

**A bare New Window inherits the spawning window's panel** rather than opening empty and
letting the strip's auto-select fall to the project's first panel.

# 9 - Open Questions

1. **Undo granularity across a cross-panel move.** Undo is per panel (`Panel.useUndo`),
   so a move produces two independent entries and `Ctrl+Z` in the target window
   un-inserts without un-removing. Options: leave it, suppress the insert from the undo
   stack, or extend the actions layer to a multi-resource transaction. Parameter, not
   shape — the move semantics hold either way.
2. **Minted panel naming collisions.** A torn "Pump Overview" tab mints a "Pump
   Overview" panel; a second tear-off of the same resource collides. Suffix, or allow
   duplicates.
3. **Default aux window size and position.** `main` used a cursor offset of
   `{x: -80, y: -45}` for tear-offs; drift otherwise centers on the main window.
