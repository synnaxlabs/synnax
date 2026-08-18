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
Remotion. `capture/` owns Playwright; `remotion/` owns compositing; `cli/` glues.

## Usage

Requires a running Core (localhost:9090) and the Console dev server
(`pnpm dev:console-vite`, localhost:5173).

```bash
# capture + direct + render
pnpm produce --script scripts/line-plot.ts --out out/line-plot

# re-render without recapturing
pnpm produce --script scripts/line-plot.ts --out out/line-plot --skip-capture

# dark theme pair
pnpm produce --script scripts/line-plot.ts --out out/line-plot-dark --theme dark
```

Docs videos ship as themed pairs (`<id>-light.mp4` / `<id>-dark.mp4`) uploaded to the
docs CDN; run each script once per theme.

## Determinism

Capture steps a virtual clock one frame at a time (`page.clock` + a Web Animations
stepper), so output is independent of machine load. The director and compositor are
pure functions of the timeline. Known limitation: the Aether worker's render loop has
its own clock; static tutorials are unaffected (the loop is request-driven and idle),
but live-telemetry shots need the planned steppable time source in Pluto before they
are frame-stable.
