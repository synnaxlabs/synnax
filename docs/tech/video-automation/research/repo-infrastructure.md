# Existing Synnax infrastructure for the video pipeline

Surveyed in-repo (worktree of `synnaxlabs/synnax`), August 2026.

## 1. Integration conductor (tc) already drives the Console with Playwright

- `integration/console/case.py` (`ConsoleCase`): Playwright **sync API**, Chromium
  only (Firefox/WebKit disabled per SY-2928), `headed` / `slow_mo` /
  `default_timeout` params, clipboard permissions, and **tracing always on**
  (screenshots + snapshots + sources; persisted as `trace.zip` on failure).
- Console discovery: navigates to the Core's own address first (the Core can serve an
  **embedded console build**); falls back to the vite dev server on **port 5173**.
- Login is a plain web form (`.pluto-field__username`, `.pluto-field__password`, "Log
  In" button); readiness signal is `.console-palette button`; every test selects the
  `TestSpace` workspace and closes all tabs.
- A rich page-object layer already exists in `integration/console/`: `console.py`,
  `workspace.py`, `schematic/`, `plot.py`, `tasks.py`, `channels.py`, `ranges.py`,
  `layout.py`, `tree.py`, `context_menu.py`, etc. A video action script can reuse
  these semantics directly (or a TS equivalent of them).
- No `playwright.config.ts` anywhere; all driver config lives in the Python case
  class. No viewport/deviceScaleFactor is set today (Playwright defaults).

## 2. Console web build

Runs in a plain browser: the login form + fallback to `:5173` proves no Tauri
dependency for core flows. Connection params are typed into the login form, not query
params. The Core can serve an embedded console build from its own port.

## 3. Aether render loop (the deterministic-capture seam)

- The visualization tree runs in a real module `Worker`:
  `console/src/Console.tsx:60` imports `@/worker?worker&url` and passes it to
  `Pluto.Provider`; `pluto/src/aether/store.ts:208` does
  `new Worker(workerURL, { type: "module" })`.
- The render loop is `pluto/src/vis/render/loop.ts` (`Loop`), constructed in
  `pluto/src/vis/render/context.ts` on the worker side (OffscreenCanvas). It is a
  **request-driven rAF loop**: `requestAnimationFrame` fires continuously, but
  `render()` returns immediately when the request queue is empty. Static scenes are
  therefore idle; only telemetry updates and interactions enqueue renders.
- Consequence for capture: `page.clock` (main thread) does not reach the worker. A
  steppable time seam must cover (a) the worker's rAF cadence and (b) whatever clock
  stamps live-telemetry windows (`TimeStamp.now` usage in telem code). The
  request-driven design helps: with no live telemetry playing, the worker renders
  nothing between interactions, so plain clock-stepped capture may already be clean
  for non-streaming tutorials.
- `pluto/src/vis/render/performance.ts` uses `performance.now` for instrumentation
  only.

## 4. Animations

- ~25 `transition:` declarations across Pluto CSS; a composed token
  `--pluto-btn-transition` in `pluto/src/theming/theme.css:39`. Durations are
  hand-set per rule (no global duration token).
- **No reduced-motion or animation-disable switch exists** in Pluto or the Console.
  CSS transitions must be handled by the capture rig's `document.getAnimations()`
  seek (they are few and short, so this is tractable).

## 5. Docs site video embedding (the output contract)

- `docs/site/src/components/media/Media.tsx`: `Video` component loads
  `https://synnax.nyc3.cdn.digitaloceanspaces.com/docs/<id>-<theme>.mp4` where theme
  is `light`/`dark` (a `themed=false` escape hatch exists). MP4 only.
- Playback: `loop muted`, played/paused by an IntersectionObserver at 0.85 threshold.
  No audio track is ever used. No captions machinery.
- **Every themed tutorial video therefore ships as a light + dark pair** — the
  pipeline must render each script twice, once per Console theme.
- Assets are uploaded to DigitalOcean Spaces out-of-band (no in-repo upload tooling
  found).
- Usage is broad: most pages under `docs/site/src/pages/reference/console/*.mdx`
  embed `Video` (channels, line-plots, schematics, workspaces, tables, logs, users,
  calculated-channels, cores...).

## 6. Demo/seed data

- `integration/scripts/sim_from_task_configs.py` (`SimDAQ`): creates
  index/analog/digital channels from task configs and generates realistic sensor
  values — a ready-made data simulator for live-chart shots.
- Integration tests assume a seeded `TestSpace` workspace; driver has full simulated
  device support used by integration runs (no hardware needed).

## 7. Existing recording tooling

None beyond Playwright tracing (`trace.zip` on failure). No screenshot/video scripts
in `scripts/` or CI workflows. The perf suite does not record video. Greenfield.
