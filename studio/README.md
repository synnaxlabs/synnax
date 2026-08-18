# Studio

Automated production pipeline for Synnax docs videos: scripted Console sessions in,
Screen Studio-grade MP4s out. Research and architecture live in
`docs/tech/video-automation/`.

## Pipeline

```
video script (scripts/*.ts)
   -> capture   deterministic clock-stepped Playwright session:
                lossless PNG per 60fps tick + input-event timeline
   -> director  pure math: event timeline -> virtual camera + synthetic cursor
   -> compositor  Remotion: zoom/pan, cursor sprite, click ripples
   -> encode    H.264 yuv420p, bt709, 60fps
```

The timeline (`src/timeline/`) is the source-agnostic contract: producers other than
the capture rig (hand-authored promo compositions) can feed the same director and
compositor.

Layering rule: `timeline/` and `director/` are pure and must not import Playwright or
Remotion. `capture/` owns Playwright; `remotion/` owns compositing; `fixtures/` owns
environment provisioning (the ephemeral core, telemetry seeding); `cli/` glues. All
cinematography policy (springs, zoom amounts, motion blur, idle fade) lives in the
director so it is unit-testable; the compositor mechanically applies director output.

The package exports one namespace per module (`capture`, `director`, `fixtures`,
`timeline`), matching the monorepo convention. Video scripts consume that surface:

```ts
import { capture, fixtures } from "@/index";

export default async (session: capture.CaptureSession) => {
  await capture.login(session, { username: "synnax", password: "seldon" }, project);
};
```

## Producing docs videos

`videos.ts` is the production manifest: one entry per docs video, keyed by the
id the docs site's Video component uses (the CDN serves
`docs/<id>-light.mp4` / `docs/<id>-dark.mp4`).

```bash
# use `pnpm run` for flagged invocations: the bare `pnpm <script>` shorthand
# consumes flags pnpm itself knows (--force, --dry-run, ...)
pnpm batch                    # produce every out-of-date manifest entry
pnpm batch ranges             # only ids containing "ranges"
pnpm run batch --draft        # fast 1080p review renders
pnpm run batch --list         # show what would run
pnpm gallery                  # write out/gallery.html for batch review
pnpm run upload --dry-run     # preview the CDN upload plan
pnpm upload ranges            # push finals (needs DO_SPACES_KEY / DO_SPACES_SECRET)
pnpm clean                    # free disk: drop capture frames, keep videos
pnpm run clean --videos       # also drop rendered videos and stamps
pnpm run clean --all          # remove out/ entirely
```

For each entry, batch captures light and dark against fresh ephemeral cores,
renders the pair to `out/<id>-light.mp4` / `out/<id>-dark.mp4`, and stamps
`out/<id>/produce.json` with a hash of the script and options; unchanged
entries are skipped on the next run (`--force` overrides). Captured frames run
to gigabytes per theme, so batch prunes them after a successful render
(`--keep-frames` retains them for `produce --skip-capture` iteration). Draft
renders are refused by `pnpm upload` unless `--allow-draft` is passed.

## Usage

Requires the Console dev server (`pnpm dev:console-vite`) served from a worktree
that includes the dev-connection port override in
`console/src/cluster/detectConnection.ts` (this one does); pass its URL via
`--url` when it isn't on the default localhost:5173. The core is managed by the
studio: each capture spawns a fresh in-memory core (`--no-driver`, logs to
`<out>/core.log`) on the studio's own port 9095 and stops it when the capture
ends, so every capture sees an empty cluster and never contends with the shared
dev core on 9090. The rig points the dev Console at the capture core through
the override (a localStorage key set before the app boots), and fixtures pick
the port up automatically. A long-lived shared core pollutes shots: stale
driver racks surface warning notifications, integration test runs leave ranges
that render as annotations on any plot whose window overlaps them, and every
capture adds a project; that is why produce refuses if the capture port is
already in use. Pass `--core external` to capture against a running core (on
9090 unless `--port` says otherwise), and `--core-bin <path>` (or
`SYNNAX_CORE_BIN`) if `core/synnax` isn't built at the repo root.

Build `core/synnax` in the same worktree that serves the Console: a core from
another branch can return schema the Console fails to parse (permissions being
the dangerous one, since palette commands are permission-gated and silently
vanish when the retrieve fails).

```bash
# capture + direct + render
pnpm produce --script scripts/line-plot.ts --out out/line-plot

# re-render without recapturing
pnpm produce --script scripts/line-plot.ts --out out/line-plot --skip-capture

# dark theme pair
pnpm produce --script scripts/line-plot.ts --out out/line-plot-dark --theme dark

# output resolution target (default: native capture resolution, width*dsf)
pnpm produce --script scripts/line-plot.ts --out out/line-plot --target 1080p
```

## Resolution

Capture happens at `--width x --height` CSS pixels (default 1920x1080) with device
scale factor `--dsf` (default 2), so native frames are 3840x2160. `--target`
(`1080p | 1440p | 4k | <pixels>`) sets the rendered video's width independently of
the capture. Targets below native are supersampled, which also buys zoom headroom:
crisp zoom tops out at `(width * dsf) / target`, so a 1080p target from a dsf-2
capture keeps 2x zooms pixel-perfect, while a 4k target upscales during them. For
deeper crisp zooms at 4k, capture with `--dsf 3`.

## Camera

Auto-zoom frames each click on the clicked element, not the click point: capture
records the target's bounding rect into the timeline, and the director picks a zoom
amount that fits the rect plus a margin (capped at the Screen Studio default), then
positions the crop to contain it. Scripts can also author the camera directly:
`session.zoom(locator)` opens an override framing the element (amount derived from
its rect unless given) and suppresses auto-zoom until `session.endZoom()`.

The synthetic cursor fades out after sitting still for 1.5s and fades back in the
moment it moves or presses again, matching Screen Studio's idle behavior. The fade
is computed in the director from actual sample motion, so click anticipation and
spring settle count as movement.

Captures run under a fixed virtual-clock epoch, so main-thread timestamps are
identical across runs and between the light/dark videos of a pair. Plot axis labels
are not: they come from the Aether worker, which the virtual clock does not reach
(the same seam noted under Determinism), so they show wall time until Pluto gains a
steppable time source.

## Pacing

`session.setSpeed(factor)` changes presentation speed mid-capture: each output frame
samples `factor` frames of app time, so `> 1` fast-forwards (speed through typing)
and `< 1` is slow motion. Because the clock is virtual, slow motion is exact
resampling, not frame interpolation. Durations passed to `hold`/`type` stay in app
time. `commandPalette` accepts `{ typeSpeed }` (default 1.5) and restores natural
speed before the selection.

Docs videos ship as themed pairs (`<id>-light.mp4` / `<id>-dark.mp4`) uploaded to the
docs CDN; `pnpm batch` runs each manifest script once per theme.

## Determinism

Capture steps a virtual clock one frame at a time (`page.clock` + a Web Animations
stepper), so output is independent of machine load. The director and compositor are
pure functions of the timeline. Known limitation: the Aether worker's render loop has
its own clock; static tutorials are unaffected (the loop is request-driven and idle),
but live-telemetry shots need the planned steppable time source in Pluto before they
are frame-stable.
