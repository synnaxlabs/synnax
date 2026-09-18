# Automated docs videos: research synthesis

Goal: fully automated, Screen Studio-grade tutorial videos for the docs site,
generated from scripted Console sessions. Six research reports live in `research/`;
this document locks the conclusions.

## Verdict

Feasible, and the quality bar is reachable. The three pillars:

1. **Capture** is lossless and deterministic via clock-stepped screenshot rendering
   (Playwright `page.clock` + `document.getAnimations()` seek + CDP
   `Page.captureScreenshot` per 1/60 s tick, PNG, deviceScaleFactor 2–3). Playwright
   `recordVideo` and CDP screencast are disqualified (1 Mbps VP8, variable frame
   rate). See `research/capture-technology.md`.
2. **The Screen Studio look is fully specified math**, not taste. Its exact spring
   constants leaked in its own landing bundle (cursor 470/70/3, click 530/40/1) and
   Cap's MIT-visible renderer confirms the complete recipe: auto-zoom segments
   [click−0.3 s, click+2.5 s] at 2.0x merged at ≤2.5 s gaps, camera spring
   200/40/2.25, dead-zone cursor follow (50%x70% of viewport), edge snap 0.25,
   velocity-smear motion blur, click shrink 0.8x/130 ms. See
   `research/screen-studio-anatomy.md` and `research/prior-art.md`.
3. **Compositing** runs in Remotion (PNG-sequence source, camera as CSS transform,
   cursor/ripples as React components), with a custom @napi-rs/canvas + ffmpeg
   compositor as the licensing/perf fallback. A 90 s 4K60 render lands in minutes.
   See `research/render-stack.md`.

Nothing commercial does app-driving + video polish end to end; the closest OSS
(argo, playwright-recast, supercut) validates the shape but tops out at
ffmpeg-filter quality. We would be assembling known parts into an unoccupied slot.

## Locked architecture

```
action script (TS, semantic steps; reuses integration page-object semantics)
   │
executor: Playwright vs fresh Core + seeded data (SimDAQ), Console web build
   viewport 1920x1080 CSS, deviceScaleFactor 2 (3 if zooms >2x needed)
   clock-stepped 60fps capture -> frames/%06d.png (lossless)
   event tap -> events.json {t, type, x, y, key, targetRect}
   run twice: light theme + dark theme (docs Video component requires the pair)
   │
director (pure TS, deterministic): events.json ->
   zoom segments (Cap recipe) -> camera spring sim (8 ms steps)
   cursor path (minimum-jerk between waypoints + 470/70/3 spring,
   click anticipation 500 ms, click spring 530/40/1)
   │
compositor (Remotion): frames + camera/cursor tracks ->
   zoom/pan crop, synthetic SVG cursor (own art; never Apple's bitmaps),
   click ripples, click shrink, motion blur on camera moves only,
   optional padding/background/shadow card
   │
encode: H.264 yuv420p CRF 16-17, bt709 tagged, +faststart, 60fps
   deliverables: 4K60 master + 1080p60; upload pair to DO Spaces CDN
```

No audio track: docs videos are `muted loop`; narration is out of scope
(`research/tts-narration.md` retained as reference only).

## Decisions locked by the research

- **Capture = clock-stepped screenshots**, not real-time recording and not
  BeginFrameControl (macOS-unsupported, legacy binary). BFC on Linux stays the
  fallback if residual nondeterminism appears.
- **Cursor is synthesized in post** from the event timeline; nothing cursor-shaped is
  rendered in the page during capture.
- **Compositor = Remotion first** (fastest to quality; confirm Creators vs Automators
  licensing with remotion.dev, worst case $100/mo), custom Node canvas compositor as
  the planned escape hatch; both share the director's camera/cursor tracks.
- **Effect math ported from Cap conceptually** (AGPL: reimplement, never copy code).
- **Zoom budget rule**: zoom z stays crisp iff z ≤ (CSS width x DSF) / output width.
  DSF 2 gives 2.0x at 1080p output; capture DSF 3 if a script needs more; 4K master
  accepts mild softening at deep zooms (Screen Studio does the same).
- **Encode**: H.264 High yuv420p, CRF 16-17, explicit bt709 tags; supersampled 4K
  source is the chroma-subsampling mitigation. VP9/AV1 variants optional later.

## Known risks and their answers

- **Aether worker clock**: the render loop (`pluto/src/vis/render/loop.ts`) is
  rAF-driven inside a real Worker; `page.clock` does not reach it. Mitigation A: the
  loop is request-driven, so without live telemetry the worker is idle and capture is
  already clean. Mitigation B (needed for live-chart shots): a small injected,
  steppable time source for the worker render path + telemetry `TimeStamp.now`,
  advanced per tick via an exposed binding. This is the only app change the pipeline
  requires.
- **CSS transitions** (~25 declarations in Pluto, no reduced-motion switch): seeked
  per tick via `document.getAnimations()`.
- **Zoom taste**: encode the pacing rules (hold ≥600 ms, ≤1 move per 3-4 s, merge
  segments, zoom where the cursor settles) as director defaults; a per-script
  override file handles exceptions.

## Prototype (next step)

One hardcoded flow, no agents, no upload:
1. Capture rig: launch flags (`--force-color-profile=srgb`, `--hide-scrollbars`),
   `page.clock` install, tick loop with animation seek, CDP PNG per tick, event tap.
   Drive a real flow against a dev Core (for example: open palette, create a line
   plot, add a channel, watch data).
2. Director: event timeline -> camera + cursor tracks (the constants above).
3. Remotion composition: frames + tracks -> 1080p60 H.264, light theme only.
4. Judge output against a hand-made Screen Studio recording of the same flow.

Success criterion: a reviewer cannot tell within three seconds which video was
hand-recorded. Then: dark theme, agent-authored scripts, QA loop, CDN upload.
