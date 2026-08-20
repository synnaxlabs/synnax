# Authoring a shot

How to turn one docs `<Video id="..." />` into a light/dark pair. Pipeline
mechanics, flags, and camera theory live in `README.md`; this file is the
rulebook and the trap list.

## The loop

1. Read the id's line in `docs/tech/video-automation/shot-list.md`. That
   description is the contract: the shot shows exactly that flow, nothing more.
2. Write `scripts/<name>.ts` (default export `async (session) => {}`).
3. Register the id in `videos.ts`.
4. Draft it:

   ```bash
   cd studio
   caffeinate -i pnpm exec tsx src/cli/batch.ts "<id>" --draft --force \
     --url http://localhost:5174 --port <your port>
   ```

5. Verify frames (below). Iterate until the end frame shows the finished state
   and no mid frame shows a broken camera.

A pair takes 5 to 6 minutes. Draft renders are review-only.

## Rules

These come from user review and are not open for reinterpretation.

- **Natural path, not the palette.** Use the discoverable UI: the toolbar "+",
  context menus, the component selector. The exception is a shot whose subject
  is the palette itself.
- **Create shots start from an empty panel.** `capture.clearPanel` in setup,
  then `capture.clickPanelCreate` on camera so the viewer sees the component
  selector open.
- **Form dialogs frame the whole dialog.** `session.zoom(".console-modal")`
  before it opens, `session.endZoom()` after it closes. Never zoom a button
  that the dialog already covered.
- **Context menus never zoom.** The menu and the row it came from must both
  stay in frame.
- **View-swapping clicks pass `{ zoom: false }`.** A click that replaces the
  panel contents leaves the camera nothing to punch into, and the receding zoom
  smears the next beat. Same for every `session.drag`.
- **End on an additive change.** The last beat adds information to the screen.
  Toggling a column off is a subtractive ending and gets rejected.
- **Type the full word into dropdown searches.** They do not prefix-match, and
  list order is nondeterministic.
- **Rows measured by text.** Clicks on flex-grown rows pass `{ text: true }`,
  or the cursor lands on the row box instead of the glyphs.
- **Never touch what the rig injects, and leave no scaffolding.** The rig hides
  the notification feed and pins the caret through one injected stylesheet.
  Rewriting it from a script unhides environment noise (the Core reports clock
  skew and an unknown embedded driver status on every capture). Inspect the DOM
  in a throwaway file you delete, never in the shot.

## Traps

- The window panel bar renders `.pluto-tabs__tab` too. Scope mosaic tab
  locators to `.console-mosaic`.
- Menu items carry a shortcut indicator, so `getByText("Close", { exact: true })`
  misses. Match a prefix regex.
- The inline editable class is `pluto-text--editable`. `CSS.BM` is
  `block--modifier`, not `__`.
- The range overview's own title input answers `getByPlaceholder("Name")`.
  Scope modal locators to `.console-modal`.
- The Axes tab opens on X1, the time axis. Click Y1 before setting bounds.
- A line plot hold snapshots the axis the moment it turns on, and the empty
  default is an hour-wide window at page load. Wait for real samples
  (`settleWall`) before clicking pause.
- Live data moves in wall time while the video runs in virtual time, so a point
  pinned early scrolls off before a later beat. Pause the plot, or place the
  interaction late.
- `session.hold` advances the virtual clock only. `session.settleWall` waits
  real time, which is what live telemetry buffers need.
- Workspaces are projects on rc. `console/workspaces/*` has no counterpart yet.

## Verifying

The Remotion ffmpeg needs its own directory as the working directory:

```bash
cd node_modules/.pnpm/@remotion+compositor-darwin-arm64@*/node_modules/@remotion/compositor-darwin-arm64
./ffmpeg -y -sseof -1 -i <video> -update 1 -frames:v 1 /tmp/end.png   # last frame
./ffmpeg -y -ss 4.0  -i <video> -update 1 -frames:v 1 /tmp/mid.png    # by time
```

`-vf select=eq(n\,N)` with a `%d` output fails here; extract one frame per call
with `-update 1`. Read both PNGs and check: the end state matches the shot
description, the camera holds the subject, no dialog is cropped, and no cursor
sits over the thing it just changed.

## Running in parallel

Each capture spawns its own in-memory core and refuses to start if the port is
taken, so concurrent runs need distinct ports (`--port`). One Console dev
server serves every worker: the rig points each browser at its own core through
a localStorage override. Do not edit `videos.ts`, `src/capture/*`, or
`src/cli/*` while another worker is running; write your own script file, and
report a helper you need rather than adding it to a shared module.
