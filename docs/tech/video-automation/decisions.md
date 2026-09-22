# Automated docs videos: locked decisions

Docs videos are produced by the studio package (`studio/`): scripted Console sessions
in, Screen Studio-grade MP4 pairs out. `studio/README.md` documents the shipped pipeline
and `studio/AUTHORING.md` the authoring rules. This file records the decisions behind
that design, the alternatives they rejected, and the provenance of the cinematography
constants in `studio/src/director/constants.ts`. `shot-list.md` lists the docs videos to
produce.

## Decisions

**Capture is clock-stepped screenshots.** Playwright `page.clock` plus a Web Animations
stepper advance app time one frame at a time; CDP `Page.captureScreenshot` writes a
lossless PNG per tick at `deviceScaleFactor` 2. Output is independent of machine load,
and both videos of a themed pair carry identical on-screen timestamps.

**The cursor is synthesized in post.** Nothing cursor-shaped renders during capture. The
rig logs input events to the timeline; the director draws the path from them. The
sprites are own artwork, never OS cursor bitmaps.

**The compositor is Remotion.** PNG sequence as source, the camera as a CSS transform,
cursor and ripples as React components.

**The effect math is ported conceptually from Cap, never copied.** Cap is AGPL; the
constants below are re-derived, and the implementation is ours.

**All cinematography policy lives in the director.** `timeline/` and `director/` are
pure and unit-tested; `capture/` owns Playwright and `remotion/` owns compositing. The
compositor mechanically applies director output.

**Zoom budget.** A zoom `z` stays crisp only while
`z <= (CSS width * dsf) / output width`. A dsf-2 capture rendered at 1080p keeps 2x
zooms pixel-perfect; deeper crisp zooms at 4k need `--dsf 3`.

**Encode.** H.264 yuv420p, explicit bt709 tags, 60fps. Supersampling from a
higher-resolution capture is the chroma-subsampling mitigation.

**No audio.** Docs videos render `muted loop`, so narration and TTS are out of scope.

## Rejected

- **Playwright `recordVideo` and CDP screencast**: hardcoded 1 Mbps VP8, variable frame
  rate, and a silent 800x450 downscale. The quality floor is far below the bar.
- **`HeadlessExperimental.beginFrame` (BeginFrameControl)**: the strongest determinism
  guarantee, but macOS-unsupported and confined to a legacy binary. It stays the Linux
  CI fallback if residual nondeterminism appears.
- **Real-time capture** (Xvfb + x11grab, macOS ScreenCaptureKit): drops frames under
  load. Useful for quick previews, never for final renders.
- **MediaRecorder / `getDisplayMedia`**: variable frame rate and silent frame drops.
- **Motion Canvas and a pure ffmpeg filtergraph**: neither reaches the polish bar
  without rebuilding the compositor anyway.

## Known limitation

The Aether worker's render loop has its own rAF clock, which `page.clock` does not
reach. Static tutorials are unaffected because the loop is request-driven and idle. Live
telemetry shots run on wall time while the video runs in virtual time, so plot axis
labels show wall time and a point pinned early scrolls off before a later beat. A
steppable time source in Pluto is what would close the seam.

## Provenance

The constants in `studio/src/director/constants.ts` come from three hard sources.

**Screen Studio's own landing bundle** ships the literal spring presets of its in-app
cursor engine: `https://screen.studio/_next/static/chunks/9764-19903045e53be0b5.js`
(module 27647: 170/50/3, **470/70/3**, **530/40/1**, 340/60/3; click scale interpolated
over [0.8, 1]). Its [guide](https://screen.studio/guide) and
[changelog](https://screen.studio/changelog) date the effects (motion-blur engine
2.25.18, ripple 2.26.0, shortcut timeline 2.22.0).

**Cap** (AGPL, read for behavior only) encodes the same recipe in concrete numbers:
[cursor_interpolation.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/cursor_interpolation.rs)
(500 ms click lookahead, 175 ms stiffen, phase lead c/k),
[zoom_spring.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/zoom_spring.rs)
(8 ms simulation steps, 0.5x0.7 dead-zone follow, pre-aim while zoomed out),
[configuration.rs](https://github.com/CapSoftware/Cap/blob/main/crates/project/src/configuration.rs)
(camera spring 200/40/2.25, edge snap),
[recording.rs](https://github.com/CapSoftware/Cap/blob/main/apps/desktop/src-tauri/src/recording.rs)
(auto-zoom segments at click-0.3 s to click+2.5 s, 2.0x, merged at 2.5 s gaps), and
[layers/cursor.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/layers/cursor.rs)
(click shrink 0.8x over 130 ms, idle fade, velocity smear).

**The reverse-engineered Screen Studio project format** in
[crafter-station/open-screenstudio](https://github.com/crafter-station/open-screenstudio/blob/main/docs/TECHNICAL_PLAN.md)
confirms the record-time channels: 120 Hz cursor samples, spring 470/70/3, cursor size
1.5x, `follow-cursor` zoom with `snapToEdges` 0.25.

Supporting math: minimum-jerk travel is Flash and Hogan, _The coordination of arm
movements_, J. Neurosci. 5(7), 1985. Spring parameters use the react-spring convention
(omega0 = sqrt(k/m), zeta = c / 2*sqrt(k*m)).

The studio's values diverge where captured app footage asked for it: a softer camera
spring (130/42/3), a shorter post-click hold, and rect-fitted zoom amounts rather than a
fixed 2.0x, since the director knows the clicked element's bounding rect and Screen
Studio does not.
