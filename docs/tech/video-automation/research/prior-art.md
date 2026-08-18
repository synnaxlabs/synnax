# Prior art survey: auto-generated Screen Studio-quality demo videos

## Part 1: Open-source Screen Studio alternatives (rendering internals)

### Cap (cap.so) — the single most valuable codebase for us
- **Repo**: https://github.com/CapSoftware/Cap — ~21k stars, very active. Tauri v2 +
  SolidStart UI, all rendering/export in **Rust with wgpu** (plus a newer
  `rendering-skia` crate).
- **License**: **AGPLv3** for the core (MIT only for `cap-camera*`/`scap-*` capture
  crates). So: copy techniques and math, do not link as a dependency unless AGPL is
  acceptable.
- **Rendering crate** (`crates/rendering/src/`) is exactly our problem domain,
  decomposed cleanly:
  - `cursor_interpolation.rs` — cursor smoothing is a **spring-mass-damper
    simulation**, not curve fitting. Key constants: click spring tension=530,
    mass=1.0, friction=40; drag spring tension=1000; sim step 16.67 ms. Three spring
    profiles switched dynamically (default / "snappy" within 175 ms of a click / drag
    while button held). **Click lookahead**: target snaps to an upcoming click
    position within 500 ms so the cursor arrives on time. **Phase-lead compensation**:
    target leads the raw path by `friction/tension` seconds to cancel spring lag.
    Pre-processing: shake filter (drops reversals < 0.015 UV) and burst decimation.
    Exposes velocity per frame (feeds motion blur).
  - `zoom.rs` + `zoom_spring.rs` + `spring_mass_damper.rs` — zoom is `SegmentBounds`
    computed from `(amount, center)` with `from_amount_center`, a spring-driven
    activity parameter `t` in [0,1], **edge snapping** (`snap_to_edges` remaps focus
    so corners are reachable) and clamping so the viewport never leaves the source
    frame. Zoom focus follows the smoothed cursor path.
  - `layers/` — background.rs (colors/gradients/wallpaper), blur.rs, cursor.rs,
    camera.rs, captions.rs, color_grade.rs, display.rs, frame.rs (padding/rounded
    corners/shadow), keyboard.rs (keystroke viz), mask.rs, notch.rs, text.rs. A clean
    layered compositor design worth mirroring.
  - `layers/cursor.rs` motion blur: **velocity-based smear** (linear stretch along
    per-frame travel × strength, strength clamped to 4.0, travel capped at 480 px to
    kill teleport artifacts). Click animation: smoothstep scale to 0.8 over 0.13 s
    (not a spring). Cursor texture: SVG assets rasterized at 200 px, or captured
    cursor image, or a programmatic soft circle. Size normalization formula:
    `STANDARD_CURSOR_HEIGHT * (source_h/1080) * (frame_h/crop_h) * size_factor *
    click_scale`.
  - Export: `crates/export`, `enc-ffmpeg` / `enc-avfoundation` / `enc-mediafoundation`
    / `enc-gif`, `cap-muxer`, `ffmpeg-hw-device`, `frame-converter`, `gpu-converters`.
- **Verdict**: **copy techniques** (the spring constants, lookahead/phase-lead trick,
  edge-snap zoom math, velocity-smear blur are all directly portable). AGPL blocks
  dependency use.

### OpenScreen (and the Recordly fork family)
- Original: https://github.com/siddharthvaddem/openscreen (~40k stars, MIT) —
  **archived June 2026**; community fork at
  https://github.com/EtienneLescot/openscreen; a differently-owned continuation at
  https://github.com/getopenscreen/openscreen. **Electron + React/TypeScript +
  PixiJS** compositor. Auto or manual zooms with adjustable depth/duration/easing;
  cursor themes, click effects, smoothing; MP4/GIF export; ScreenCaptureKit (macOS) /
  Windows Graphics Capture, browser pipeline on Linux.
- Recordly (https://recordly.dev, https://github.com/WizardofTryout/recordly, plus
  several mirror accounts like DougNix/recordly and webadderallorg/Recordly — the
  topic page is polluted with mirrors, treat star counts skeptically): "substantially
  modifies the OpenScreen foundation", adds a fuller cursor animation/rendering
  pipeline and "zoom animations faithful to Screen Studio".
- **Verdict**: **copy techniques**. MIT-licensed, and it proves the whole effect stack
  (zoom, cursor, background) is doable in a TypeScript/canvas (PixiJS) 2D compositor —
  closest architectural template if we render in Node/web rather than Rust.

### Reframed
- https://github.com/jkuri/reframed — Swift/SwiftUI, macOS-native, MIT, 139 stars,
  active. Auto-zoom **keyframe generation from click clusters + dwell-time analysis**;
  cursor smoothing via spring physics; backgrounds; WhisperKit captions; MP4/MOV/GIF,
  H.264/H.265/ProRes. HN thread: https://news.ycombinator.com/item?id=47141255.
- **Verdict**: copy the auto-zoom segmentation heuristic (click clustering + dwell
  time); Swift code otherwise not reusable for us.

### screenstudio-alt (headless, agent-oriented — conceptually closest to our plan)
- https://github.com/connerkward/screenstudio-alternative-skill — MIT. **CLI, no GUI
  recording app**: input is a raw `.mov` **plus a `.jsonl` event log** (clicks,
  keystrokes, cursor positions from a small Swift logger). Python `polish.py`
  auto-detects action zones, generates zoom keyframes with easing, renders cursor,
  ripples, keystroke chips, idle speed-up, callouts, vertical export — **in a single
  ffmpeg render pass**, plus a local web timeline editor and FCPXML export. Built as a
  Claude Code skill for agent-driven workflows.
- **Verdict**: **study closely** — same architecture as ours (clean capture + event
  timeline → synthetic post-production), just fed by a screen recorder instead of
  Playwright.

### Others in the space
- **Screenity** — https://github.com/alyssaxuu/screenity, 18.5k stars, GPL-3.0 (v3+),
  Chrome extension (MV3, ffmpeg.wasm). Annotation-first; has zoom smoothing and cursor
  highlight but browser-extension capture quality ceiling; solo-maintained. Verdict:
  ignore (GPL + wrong architecture).
- **Kap** — https://github.com/wulkano/kap, ~19k stars, MIT, Electron/macOS. Capture +
  plugin export only; **no auto-zoom, no cursor synthesis**. Verdict: ignore.
- **Screenize** — https://github.com/syi0808/screenize (macOS, auto-zoom, paused
  development). Ignore.
- **screen-demo** — https://github.com/njraladdin/screen-demo (Tauri + React, MIT, 49
  stars, manual "add zoom at playhead"). Ignore.
- **open-recorder** — https://github.com/imbhargav5/open-recorder (native Swift,
  generates zooms from click telemetry). Minor reference.
- From the `screen-studio` GitHub topic (https://github.com/topics/screen-studio):
  capptivo (SECHAK-AG, TS, ~700 stars), lumirec, focuscut (browser-based zoom follow),
  dolly (Rust/Windows), mochi-recorder (Linux), Focuso (Swift), OpenScreenStudio,
  nuvideo-el, inkast-suite. Also see
  https://github.com/topics/screen-studio-alternative. None add techniques beyond
  Cap/OpenScreen.
- **screenstudio-agent** — https://github.com/HyperfocuSam/screenstudio-agent (fork of
  ShawnPana/screenstudio-cli), MIT. Drives **the real Screen Studio app via CDP** (it
  is Electron) from the CLI: record, split, speed, add zoom ranges, export. Fragile
  (unsupported debug interface) but a viable shortcut if we ever wanted Screen
  Studio's actual renderer in an automated pipeline on macOS.
- **screenstudio-to-mp4** exporters (salatech, vignesh-sabhahit) — parse Screen Studio
  project files with Python/ffmpeg; useful only as documentation of Screen Studio's
  project format.

## Part 2: Browser sessions → polished video

- **timecut / timesnap (tungs)** — https://github.com/tungs/timecut,
  https://github.com/tungs/timesnap. Virtual-time capture: overrides
  `requestAnimationFrame`/timers via timeweb, screenshots each deterministic frame,
  ffmpeg-encodes. Guarantees **perfectly smooth 60 fps regardless of machine speed** —
  the key capture trick for quality. Supports Puppeteer and (since v0.3.1) Playwright.
  npm v0.3.3 is ~4 years old; effectively dormant but small and stable. Verdict:
  **copy the virtual-time technique** (or use Remotion, which does the same thing
  internally).
- **puppeteer-screen-recorder** —
  https://github.com/prasanaworld/puppeteer-screen-recorder. CDP
  `Page.screencast`-based, real-time capture, MP4/AVI/MOV/WebM. Maintenance
  **inactive** (no release in 12+ months). Quality ceiling: variable-fps screencast
  frames, no effects. Verdict: ignore.
- **playwright-recast (ThePatriczek)** — https://github.com/ThePatriczek/playwright-recast,
  MIT, TypeScript, ffmpeg. **Consumes Playwright trace.zip** (screenshots, timestamps,
  clicks, DOM snapshots) instead of video; chainable pipeline: hide steps, speed up
  idle, subtitles, TTS voiceover (ElevenLabs/OpenAI), **auto-zoom with easing and
  zoom-to-zoom panning, synthetic animated cursor traveling between click positions**.
  Write-up:
  https://dev.to/thepatriczek/i-was-tired-of-re-recording-product-demos-every-sprint-so-i-built-a-tool-that-turns-playwright-21od.
  Verdict: **study or use** — the closest existing "Playwright session → polished
  video" product; quality ceiling limited by trace screenshot cadence and
  ffmpeg-filter compositing.
- **argo (shreyaskarnik)** — https://github.com/shreyaskarnik/argo, MIT, 610 commits,
  active, ships a Claude Code skill. Playwright screencast (plus a "JPEG-stitch"
  high-fidelity mode) + `narration.mark()` timestamps → local TTS (Kokoro et al.)
  aligned to marks → ffmpeg export with **zoompan zoom-to-element, GSAP-animated DOM
  overlays, spotlight/focus ring, speed ramps, transitions, motion blur option,
  gradient background framing with padding/rounded corners**, multi-aspect export,
  preview editor with waveform. Verdict: **study or use** — most complete
  Playwright-native pipeline; but effects are ffmpeg-filter grade, below Screen
  Studio's GPU compositor quality.
- **testreel (greentfrapp)** — https://github.com/greentfrapp/testreel, MIT, 82 stars.
  JSON step definitions → Playwright → WebM/MP4/GIF with animated cursor + click
  ripples, zoom, window frame, backgrounds; Playwright test fixture integration and
  auth helpers. Verdict: study (JSON scene schema, fixture integration).
- **supercut (Co-Messi)** — https://github.com/Co-Messi/supercut, MIT, TypeScript.
  Five-stage pipeline: LLM analyzes repo + live DOM → schema-validated "filming
  recipe" JSON → deterministic Playwright recording → vision QC with retakes → ffmpeg
  render (cursor-following camera zooms, motion blur, music, wallpaper staging,
  1080p60). Verdict: **study** — it is our concept end to end (agent scripts the app,
  synthetic cinematography after), young codebase.
- **Remotion + Remotion Recorder** — https://github.com/remotion-dev/remotion,
  https://github.com/remotion-dev/recorder, https://www.remotion.dev/docs/recorder/.
  Deterministic frame-by-frame rendering in headless Chrome → ffmpeg; the Recorder is
  a talking-head/screen video production template (Whisper captions, layouts), **not**
  an app-driving demo tool. License: free for individuals/small companies, **paid
  company license** above a size threshold — a real constraint for us. Why frames beat
  live recording (determinism, parallelism, crash resilience):
  https://github.com/orgs/remotion-dev/discussions/4351. Community templates:
  **remotion-cinematic** (https://github.com/codeverbojan/remotion-cinematic, MIT —
  geometry-aware cursor targeting element IDs with arc/linear/ease interpolation,
  per-scene AutoZoom; but renders **React mockups from JSON**, not the real app),
  remotion-saas-showcase (https://github.com/Raazi305/remotion-saas-showcase),
  remotion-yard (https://github.com/dsomel21/remotion-yard). There is also a
  "playwright-recording" Claude skill for feeding Playwright captures into Remotion:
  https://skills.rest/skill/playwright-recording.
- **editframe** — https://editframe.com, https://github.com/editframe. Commercial
  "video with code" API, still alive (repos updated 2025-2026), now marketing
  LLM-friendly composition. Closed SaaS; verdict: ignore.
- **product-launch-motion**
  (https://github.com/Clairvoyant-fenestration570/product-launch-motion) — Claude +
  GSAP + HTML deterministic motion; minor.
- **DIY blog**: Justin Abrahms, "Generating demo videos with Playwright"
  (https://justin.abrah.ms/blog/2026-02-12-generating-demo-videos-with-playwright.html)
  — injects a CSS-animated fake cursor into the page, `page.setContent()` title cards,
  ffmpeg WebM→H.264 + music. Confirms the pain points (headless Chromium renders no
  cursor; WebM output; no framing) and the overlay-injection approach's ceiling.
- **ffmpeg technique posts**: Creatomate on zoompan quality (upscale before zoompan to
  avoid jitter): https://creatomate.com/blog/how-to-zoom-images-and-videos-using-ffmpeg;
  Ken Burns with ffmpeg: https://mko.re/blog/ken-burns-ffmpeg/. Takeaway: raw
  `zoompan` is sub-pixel-jittery; every quality implementation (Cap, Screen Studio)
  resamples per-frame in a GPU/canvas compositor instead.

## Part 3: Commercial AI docs-video products

- **Screen Studio** (target quality bar, screen.studio) — macOS, Electron; smoothed
  cursor, auto zoom on click, motion blur, backgrounds. No public API; the CDP hack
  above exists.
- **Clueso** (https://www.clueso.io) — upload/record rough recording → AI adds **auto
  zoom-ins on detected UI actions**, spotlights, callouts, AI script + studio
  voiceover, and a parallel step-by-step article. Human records; no auto-driving of
  the app.
- **Trupeer** (https://www.trupeer.ai) — same shape: browser-extension recording →
  filler-word removal, script rewrite, 100+ TTS voices, **automatic smooth zoom
  transitions on every click**, avatars, doc export. Their "studio quality from rough
  recordings" is exactly our post-production stage: event-timeline-driven zoom/cursor
  synthesis + TTS narration; the "AI" is script cleanup and step detection, not novel
  rendering.
- **Guidde** (https://www.guidde.com) — "Magic Capture" extension/desktop logs every
  click/scroll/keystroke, then generates video + written guide in seconds with **auto
  pan-and-zoom at detected click areas**, 400+ TTS voices, Magic Mic narration
  cleanup. Human-driven capture. Their event-log-first architecture is the same as our
  Playwright event timeline.
- **Arcade** (https://www.arcade.software) — records once, exports **interactive demo
  + video + GIF simultaneously**, with auto-progression and pan/zoom in video exports
  (changelog: https://www.arcade.software/changelog). Human-driven.
- **Supademo** (https://supademo.com) — primarily interactive demos; has
  video/screenshot export modes, AI annotation text, voiceover; human-driven.
- **Storylane** (https://www.storylane.io) — interactive click-through demos first;
  video recording exists but video polish is not the product.
- **Colossyan** (https://www.colossyan.com) — AI-avatar presenter videos; screen
  recording lives inside the editor with manual zoom; different category (training
  videos with avatars), not auto-cinematography.
- **FocuSee** (iMobie/Gemoo, https://focusee.imobie.com) — desktop app, auto-detects
  click locations post-recording and generates zoom animations + cursor effects; no
  automation input.
- **Tella** (https://www.tella.com) — web + mac editor, Auto Zoom based on on-screen
  activity, up to 350% zoom, backgrounds, 4K export; manual/AI-assisted editing of
  human recordings.
- **Cursorful** (Chrome extension) — automatic smooth zooms/pans following cursor
  clicks; small-scale.
- **Key market fact**: none of these drive the application automatically. Every one
  requires a human recording. The auto-driven slot (Playwright as the "camera
  operator") is occupied only by the small OSS projects above (supercut, argo,
  testreel, playwright-recast) — so nothing commercial to buy, and the OSS ceiling is
  ffmpeg-filter quality.

## Synthesis: the 3-5 artifacts worth deep study

1. **Cap's `crates/rendering`** (AGPL — copy math, not code): the complete,
   production-hardened answer key for every hard effect. Spring-mass-damper cursor
   smoothing with click lookahead and phase-lead lag cancellation (tension 530 /
   friction 40 / mass 1; drag 1000/40), zoom-as-SegmentBounds with edge snapping and
   spring-driven activity, velocity-smear cursor motion blur with caps, layered wgpu
   compositor (background/display/cursor/captions/blur/color-grade), multi-backend
   hardware encode. Port this math into whatever renderer we choose.
2. **argo** (MIT): the most complete Playwright-native pipeline to reuse or fork
   outright: mark-based narration alignment, local TTS, GSAP overlays, multi-aspect
   export, preview editor, Claude Code skill. Its weakness (ffmpeg `zoompan`-grade
   camera) is exactly where Cap's math upgrades it.
3. **playwright-recast** (MIT): the trace-first idea — Playwright's trace.zip already
   is our "event timeline" format (clicks, timestamps, DOM snapshots); its pipeline
   API (hide steps, idle speed-up, zoom, cursor synthesis, TTS, subtitles) validates
   the whole product shape.
4. **screenstudio-alt skill + OpenScreen** (MIT): screenstudio-alt for the headless
   architecture (video + JSONL events in, single-pass polish out, FCPXML escape
   hatch); OpenScreen/Recordly for a proven TypeScript/PixiJS 2D compositor
   implementation of zoom/cursor/background if we want the renderer in our own TS
   stack rather than Rust or Remotion (whose company license is a cost for us).
5. **timecut's virtual-time capture** (plus supercut's LLM-director pipeline as a
   concept sketch): deterministic frame capture (override page clocks, screenshot per
   frame) is what separates "smooth 60 fps every time in CI" from flaky real-time
   screencasts; combine with a synthetic DOM cursor suppressed at capture and
   re-rendered in post.

**Recommended architecture from the evidence**: Playwright with
deterministic/virtual-time capture (timecut technique or CDP screencast at fixed
cadence) + a full event timeline (Playwright trace or custom JSONL) → our own
compositor (canvas/WebGL or Rust) implementing Cap's cursor-spring, zoom-bounds, and
velocity-smear math → ffmpeg hardware encode. Everything needed exists in MIT-licensed
form except the effect math itself, which Cap documents better than anyone and which
is small enough to reimplement cleanly.

Sources: [Cap repo](https://github.com/CapSoftware/Cap), [cursor_interpolation.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/cursor_interpolation.rs), [zoom.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/zoom.rs), [layers/cursor.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/layers/cursor.rs), [OpenScreen](https://github.com/siddharthvaddem/openscreen), [Reframed](https://github.com/jkuri/reframed), [Recordly](https://recordly.dev/), [screenstudio-alt](https://github.com/connerkward/screenstudio-alternative-skill), [Screenity](https://github.com/alyssaxuu/screenity), [Kap](https://github.com/wulkano/kap), [screen-studio topic](https://github.com/topics/screen-studio), [timecut](https://github.com/tungs/timecut), [timesnap](https://github.com/tungs/timesnap), [puppeteer-screen-recorder](https://github.com/prasanaworld/puppeteer-screen-recorder), [playwright-recast](https://github.com/ThePatriczek/playwright-recast) ([write-up](https://dev.to/thepatriczek/i-was-tired-of-re-recording-product-demos-every-sprint-so-i-built-a-tool-that-turns-playwright-21od)), [argo](https://github.com/shreyaskarnik/argo), [testreel](https://github.com/greentfrapp/testreel), [supercut](https://github.com/Co-Messi/supercut), [Remotion Recorder](https://github.com/remotion-dev/recorder) ([docs](https://www.remotion.dev/docs/recorder/)), [Remotion discussion #4351](https://github.com/orgs/remotion-dev/discussions/4351), [remotion-cinematic](https://github.com/codeverbojan/remotion-cinematic), [remotion-saas-showcase](https://github.com/Raazi305/remotion-saas-showcase), [playwright-recording skill](https://skills.rest/skill/playwright-recording), [Justin Abrahms blog](https://justin.abrah.ms/blog/2026-02-12-generating-demo-videos-with-playwright.html), [Creatomate ffmpeg zoom](https://creatomate.com/blog/how-to-zoom-images-and-videos-using-ffmpeg), [Ken Burns ffmpeg](https://mko.re/blog/ken-burns-ffmpeg/), [Clueso](https://www.clueso.io/vs/trupeer), [Trupeer](https://www.trupeer.ai/video), [Guidde](https://www.guidde.com/), [Guidde capture help](https://help.guidde.com/en/articles/9382933-getting-started-with-capturing-a-guidde), [Arcade changelog](https://www.arcade.software/changelog), [Storylane vs Arcade](https://www.storylane.io/blog/storylane-vs-arcade), [Supademo](https://supademo.com/blog/storylane-alternatives), [Colossyan screen recording](https://www.colossyan.com/screen-recording), [FocuSee auto zoom](https://focusee.imobie.com/features/auto-zoom-and-cursor-animation.htm), [Tella zoom](https://www.tella.com/features/zoom), [20 alternatives roundup](https://dev.to/justin3go/20-screen-studio-alternatives-everyone-should-know-1o1j), [screenstudio-agent](https://github.com/HyperfocuSam/screenstudio-agent), [editframe](https://editframe.com/), [Reframed HN](https://news.ycombinator.com/item?id=47141255).
