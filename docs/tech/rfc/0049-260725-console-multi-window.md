# 49 - Console Multi-Window

**Feature Name**: Console Multi-Window Opening and Movement <br /> **Status**:
Implemented <br /> **Start Date**: 2026-07-25 <br /> **Authors**: Emiliano Bonilla
<br />

# 0 - Summary

The panel refactor left the console with working multi-window plumbing and no way to
reach it: secondary windows rendered a viewport shell, per-window session state existed,
and window arrangement persisted per project through the drift partition (RFC 0046), but
nothing in the UI created a window. This RFC restores window creation and tab movement
on top of panels. A window is an unbound viewport: it carries its own panel selector
strip and can show any panel in the project, including one another window is already
showing. Windows are created by an explicit command, from a panel pill, or by tearing a
tab or a pill onto the desktop. Tearing a tab out mints a project panel to hold it.
Dragging a tab from one window into another moves it between the two windows' panels.

# 1 - Motivation

The instrumentation engineer configuring a test and the operator running it both work
across monitors: a schematic on the wall display, a plot on the desk, task configuration
on a laptop. Before panels this was routine: a tab dragged onto the desktop opened a
mosaic window, and the layout slice handled tabs crossing windows. Panels replaced the
per-window Redux mosaic with a shared server document and deleted both affordances,
because their semantics no longer held: they moved a tab between two Redux trees, and
there is only one tree now, owned by the cluster.

Shipping without a replacement would mean a control room running two displays cannot put
two views side by side at all. That is a regression against the shipped console, not a
missing enhancement.

# 2 - Vocabulary

- **Window** -> an OS window managed by Drift. The **main window** owns persistence;
  every other window is an **aux window**.
- **Panel** -> a project-owned document holding a tab tree; shared by everyone with
  access to the project.
- **Viewport** -> a window's view of one panel; the binding lives in per-window session
  state, not in the panel.
- **Tear-off** -> ending a drag over the desktop rather than over a window.
- **Ordinal** -> a window's stable number, assigned by Drift when the window is reserved
  and never reused.

# 3 - Prior Art

VS Code's auxiliary windows are created by dragging an editor onto the desktop or via
"Move Editor into New Window", and host an editor group, a container, rather than a bare
file; window layout does not survive a restart, a known complaint. Chrome tears a tab
into a new window container, and closing the window closes its tabs. Ignition
Perspective Workstation is the operator-station norm: multi-monitor is configured, each
display assigned a page, stable across restarts.

Our panel is their editor group and their page. We take the container model from all
three, the ad-hoc tear-off gesture from VS Code and Chrome, and beat both on
persistence: a panel is a server document and window arrangement persists in the project
scope, so a torn-off window comes back where it was.

# 4 - Principles

1. **A window is a viewport, never an owner.** No window-scoped document state. Anything
   a window knows about a panel lives in `session/panel`, keyed by drift window key.
2. **Session state is per-window; document state is shared.** Which panel a window shows
   and which tab is selected in each leaf are session; the tree is the panel.
3. **Nothing is created implicitly except where the user's gesture demands a
   container.** Tear-off is the single case: a torn tab must land in some panel.
4. **Closing a window destroys nothing.** Panels outlive the windows that spawned them.
5. **Reuse the proven drag machinery.** Haul mirrors drag state through Redux and drift
   syncs it to every window; target-window drops are ordinary Haul drop targets. Only
   desktop drops need a drag-end interceptor.

# 5 - Design

## 5.0 - The window model

Every window carries a panel selector strip and may select any panel in the active
project. Two windows may show the same panel: both render it, both write through the
same dispatches, and each keeps its own tab selection because leaf selection is stored
per window per panel. No ownership bookkeeping, no disabled pills, nothing to reconcile
when a window closes.

This supersedes the "move-only, no co-view" half of panels architecture decision #7.
That decision predates the shipped session slice, which already keys selection by window
and therefore already tolerates two windows on one panel.

## 5.1 - The aux window shell

An aux window renders the top bar in its secondary variant (window controls, drag
region, and the panel selector strip), the mosaic, and the bottom visualization drawer.
No left toolbars, no palette, no project or cluster chrome. Both shells mount the
trigger and tear-off side effects, so tab close, focus, window close keys, and tear-off
work in every window. Per-window nav state already exists, so the drawer's size is
remembered per window with no new state. The aux shell sits behind the same guard chain
as the main one (auth, connection, project), so RFC 0048's takeover and login regimes
render in every window.

## 5.2 - Creating windows

| Surface                 | Gesture               | Result                           |
| ----------------------- | --------------------- | -------------------------------- |
| Palette                 | "Open a new window"   | New window on the selected panel |
| Panel pill context menu | "Open in new window"  | New window showing that panel    |
| Panel pill              | Drag onto the desktop | Same as above; source unchanged  |
| Tab context menu        | "Move to new window"  | Mints a panel holding that tab   |
| Tab                     | Drag onto the desktop | Same as above, under the cursor  |

All five paths bottom out in `Panel.useOpenWindow`: it dispatches `Drift.createWindow`
under a generated key, seeds the window's title from its ordinal, and selects the panel
in the new window's session state (`Session.Panel.select({ windowKey, key })`). Window
keys are generated, not panel keys: with co-view legal there is nothing to key on, and a
generated key keeps drift's key-collision focus behavior out of the way. Size and
position come from drift's default window props; tear-offs pass a cursor-relative
position (offset `{ x: -80, y: -45 }`) so the torn tab lands where the pointer released
it.

Window titles are owned by the session panel synchronizer (RFC 0046): a window is titled
by its stable identity plus the selected panel's name (`Main - Ops`, `2 - Ops`), falling
back to `Synnax` or `Window N` when nothing is selected. Ordinals live in drift,
increment monotonically, and survive restarts, so the OS window list stays legible and
stable.

The tab context menu regains a window item. The earlier decision that "window items are
gone for good" was correct against the panel tree as it then stood, where a tab-level
move had no destination; tear-off minting a panel supplies one, and the menu item is the
discoverable twin of an undiscoverable drag.

## 5.3 - Moving tabs between windows

A drag from window A into window B's mosaic is an ordinary Haul drop in B: B's drop
target fires and drop indicators are leaf-precise. B's selected panel is a different
document from A's, so the drop is a cross-panel move: `insertTab` on B's panel followed
by `removeTab` on A's, the two-dispatch form the panel schema specifies. Insert runs
first: a failed second dispatch leaves the tab in both panels, which the user can see
and fix, where the reverse order loses it. Same-panel drops keep using `moveTab`, and
the moved tab is selected in the target window.

The haul payload carries the source panel key and the whole tab
(`Panel.createTabDragPayload`), because the drop may land in a window that has never
loaded the source panel. A window with no panel selected renders the empty state and
mounts no mosaic, so there is no drop target to hit: tear-off remains the only
implicit-creation gesture.

## 5.4 - Tear-off

Ending a drag over the desktop is the one case with no drop target, resolved by
`Window.useDropOutside`. The mechanism is platform-split: on Windows and Linux the
drag-end interceptor rides the Haul provider's `dragend`, while on macOS, which swallows
`dragend` for drops outside the window, a Tauri `mouse_up` event carries the final
screen position. "Outside" is computed against drift's window boxes, counting only
created, reserved windows.

The macOS event broadcasts to every window, and drift mirrors the dragging state
everywhere, so naively every window would act on the drop. Only the source window does:
the Haul provider's authoritative drag ref is populated only where the drag started, so
every other window's interceptor resolves nothing. Tear-off therefore works from any
window, which the deleted pre-panels implementation never managed on macOS.

A torn tab mints its panel create-first: a new panel is created with the tab already in
its root, named after the tab's resource (falling back to "New Panel" for view tabs or
when the name cannot be read), parented to the active project; then the tab is removed
from the source panel; then the window opens showing the minted panel. If the create
fails, nothing was removed and the tab stays where it was. A torn pill mints nothing; it
opens a second window on the panel and leaves the source window alone.

## 5.5 - Lifecycle

Closing a window closes nothing else: its panel stays in the strip, reachable from any
window and deleted only through the pill's existing Delete item. A torn-off panel that
outlives its window is ordinary project clutter, not a special case.

Window keys are stable across a restart because `Drift.restoreWindows` reopens stored
windows with their stored props and ordinals (RFC 0046), so a restored window keeps its
panel selection and its drawer sizing. Per-window session entries (`panels.windows`,
`nav.windows`) are keyed by window key and are not pruned when a window closes; see open
questions. No persisted-state migration was required: the per-window slices and the
drift slice already persisted in the project scope.

# 6 - Implementation

Landed with the SY-4511 panel UX batch: the aux shell, `Panel.useOpenWindow` and the
five creation surfaces, the tab and pill haul payloads, the cross-panel move in the
pluto panel mosaic's drop handler, the tear-off interceptor, window titles and ordinals,
and specs covering the cross-panel drag payload and drop semantics, the open-window
hook, and the move-to-new-window menu flow.

# 7 - What This RFC Does Not Cover

- **Per-monitor assignment.** Perspective-style "this display always shows this page" is
  a configuration model on top of this one; window arrangement persisting per project is
  as far as we go.
- **Detaching a single tab into a chrome-less window.** Every window is a panel
  viewport.
- **Cross-project windows.** A window shows a panel of the active project; project
  switch swaps every window's context through the scoped persistence swap (RFC 0046).
- **Undo across a cross-panel move.** See open questions.

# 8 - Resolved Decisions

1. **A window is not bound to a panel.** Rejected: binding each window to one panel and
   keying the drift window with the panel key, which would have made drift's
   key-collision focus behavior enforce single-viewer for free. An aux window would have
   no way to change what it shows, the strip would be main-window-only chrome, and "open
   this panel" would have to first evict it from whatever window held it. The trade is
   real: we give up structural enforcement and accept that two windows can render the
   same expensive schematic twice.
2. **Co-view is legal**, superseding panels decision #7. Rejected: forbidding it by
   graying out pills another window holds and focusing that window instead. That needs
   an ownership map in session state, a release path on every window close, and a
   conflict rule when two restored windows both claim a panel: bookkeeping in exchange
   for preventing something the user may legitimately want.
3. **Tear-off mints a project panel.** Rejected: an overlay-focused window on the source
   panel, minting nothing. The tab would keep living in its original panel and render in
   both windows (two renderers, two aether workers, doubled telemetry for one plot), and
   a "move" that leaves the tab where it was is not a move. The trade is real: idle
   tear-offs leave pills in a shared strip that teammates see.
4. **A cross-window drop moves the tab**, mutating the source panel for everyone
   watching it. Rejected: adding to the target while leaving the source intact, which is
   not what a drag means anywhere else.
5. **Torn-off panels persist when their window closes.** Rejected: folding back, i.e.
   returning the tab to its origin panel and deleting the minted one when it is
   untouched. That buys a tidy strip with a heuristic and a destructive close path, and
   the heuristic fails the moment someone adds a second tab.
6. **The aux shell is trimmed**, not full parity. Rejected: rendering the main shell in
   every window, which is less code and more predictable, at the cost of mounting every
   toolbar's queries in a window that exists to show one schematic.
7. **A bare "new window" inherits the spawning window's selected panel** rather than
   opening empty and letting the strip's auto-select fall to the project's first panel.
8. **Minted-panel name collisions are allowed.** A second tear-off of the same resource
   mints a second panel with the same name; suffixing was considered and dropped as
   bookkeeping for a state the user can see and rename.

# 9 - Open Questions

1. **Undo granularity across a cross-panel move.** Undo is per panel, so a move produces
   two independent entries and undo in the target window un-inserts without un-removing.
   Options: leave it, suppress the insert from the undo stack, or extend the actions
   layer to a multi-resource transaction. Parameter, not shape; the move semantics hold
   either way.
2. **Pruning per-window session entries.** `panels.windows` and `nav.windows` entries
   outlive their windows and persist. Restarts reuse stored window keys, so the leak is
   bounded in practice, but closed-window entries accumulate until a pruning reducer
   lands.
