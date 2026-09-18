# Demo-video post-production stack: research report

Scope: composite a raw high-DPI screen capture (or PNG frame sequence) plus a JSON
event timeline into a Screen Studio-grade video (virtual camera zoom/pan, synthetic
cursor, click ripples, key overlays, padding/shadow/background, motion blur) at 4K60
H.264/VP9, rendered in batch/CI.

---

## 1. Remotion

### Licensing (verified against remotion.dev and remotion.pro, August 2026)

- **Free license**: individuals, for-profit companies with **up to 3 employees**, and
  non-profits. Unlimited commercial use, local rendering included. Source:
  [LICENSE.md](https://github.com/remotion-dev/remotion/blob/main/LICENSE.md),
  [License FAQ](https://www.remotion.dev/docs/license/faq).
- **4+ employees = paid Company License**, one of two plans
  ([remotion.pro/license](https://www.remotion.pro/license)):
  - **Creators**: **$25/seat/month**, no minimum seat count, no minimum spend. A Seat
    = "one person who writes Remotion code themselves or uses agentic coding tools."
    Marketed for "low volume video creations through coding and prompting." Renders
    are not metered on this plan.
  - **Automators**: **$0.01 per render, $100/month minimum** (10,000 renders included
    at the minimum). For "automations such as video editors, prompt-to-video tools,
    automated video pipelines." Developer seats not required on this plan. "1 Render
    is the successful generation of a video, audio, GIF, PDF or still image"; Studio
    previews do not count.
  - **Enterprise**: from $500/month.
- **Does CI rendering count?** Yes, every successful CI render is a Render. Which plan
  applies is the real question: an unattended docs-video pipeline reads as an
  "automated video pipeline" (Automators, $100/mo floor). A docs pipeline where your
  own devs author each video and CI merely renders it is arguably Creators usage ($25
  to $50/mo for 1 to 2 seats). Remotion's public docs do not draw this line crisply;
  confirm with hi@remotion.dev before committing. Note the LICENSE.md flags "a
  licensing change coming in Remotion 5.0," so re-check at adoption time.

### Rendering model and throughput

- Architecture: each output frame = seek headless Chromium to a frame, React renders,
  DevTools screenshot (PNG/JPEG), frames piped to FFmpeg. Concurrency = N browser tabs
  rendering different frames in parallel
  ([performance docs](https://www.remotion.dev/docs/performance/)).
- Concrete numbers found:
  - [Issue #4783 "Remotion is too slow to use in production"](https://github.com/remotion-dev/remotion/issues/4783)
    (Overlap, YC S24): server-side rendering on 8 vCPU / 24 GB / GPU-enabled Cloud Run
    runs at **2x to 4x realtime** for animation-heavy content (a 1-minute video takes
    2 to 4 minutes). Closed without a maintainer fix; the screenshot model is the
    structural cost.
  - Third-party guide figure: ~8 to 15 s per 150 frames at 1080p, roughly **10 to 19
    rendered fps**
    ([PkgPulse comparison](https://www.pkgpulse.com/guides/remotion-vs-motion-canvas-vs-revideo-programmatic-video-2026)).
  - [Discussion #3070](https://github.com/orgs/remotion-dev/discussions/3070) and
    [#3088](https://github.com/remotion-dev/remotion/issues/3088): long OffthreadVideo
    renders degrade to **~5 fps** after 20 to 30% progress (cache eviction; mitigated
    by raising the OffthreadVideo cache and splitting renders).
    [#4949](https://github.com/remotion-dev/remotion/issues/4949): throwing cores at
    one render (224-core box) does not scale; concurrency has an optimum, find it with
    [`npx remotion benchmark`](https://www.remotion.dev/docs/cli/benchmark).
  - `<OffthreadVideo>` (Rust FFmpeg frame extractor) got 281% faster in v4
    ([official 4.0 benchmark](https://github.com/remotion-dev/4-0-benchmark): 4K
    source video, M2 Air, concurrency 4: 75 s vs 286 s). For our case a **PNG frame
    sequence via `<Img src={staticFile(...)}>` is simpler and avoids video-decode
    entirely**; the cost left is React render + 4K screenshot per frame.
  - Estimate for our workload (one 4K `<Img>` + CSS transform camera + SVG overlays,
    concurrency 8, desktop-class CPU): expect roughly 15 to 40 rendered fps, so a 90 s
    4K60 video (5400 frames) lands around **3 to 8 minutes**, before motion blur.
- **Motion blur**:
  [`@remotion/motion-blur`](https://www.remotion.dev/docs/motion-blur/) gives
  `<Trail>` and
  [`<CameraMotionBlur>`](https://www.remotion.dev/docs/motion-blur/camera-motion-blur).
  CameraMotionBlur renders the subtree at N time offsets and averages them: **render
  cost multiplies by `samples` (default 10)**, and the docs warn the averaging "is
  destructive to colors." Applied only during camera moves (say 25% of frames at
  samples=6) it roughly 2x's total render time. This is the exact Screen Studio look,
  but it is brute-force.
- **GPU**: [`--gl=angle`](https://www.remotion.dev/docs/gl-options) substantially
  speeds canvas/WebGL content, including
  [@remotion/skia](https://www.remotion.dev/docs/skia); known **memory leaks on long
  renders** (split renders) and **no GPU on stock GitHub Actions runners** (use
  `swangle` fallback or a GPU runner). Linux GPU needs `chrome-for-testing` mode.
- Screencast-style prior art exists and is healthy:
  [remotion-cinematic](https://github.com/codeverbojan/remotion-cinematic) (SaaS demo
  template: cursor targeting elements by ID, per-scene auto-zoom, camera movement),
  cursor/zoom template collections
  ([reactvideoeditor/remotion-templates](https://github.com/reactvideoeditor/remotion-templates),
  [ali-abassi/remotion-templates](https://github.com/ali-abassi/remotion-templates)).

**Verdict on the model**: for 60 to 90 s docs videos in batch, the browser-render
model is fast enough (minutes per video). For hour-scale output it is the wrong tool
(a 60-minute 4K60 video would take 6+ hours per worker and hit the long-render
degradation issues).

## 2. Motion Canvas (and Revideo)

- **Motion Canvas is abandoned.** MIT-licensed (it moved off its earlier
  non-commercial license years ago), but the last commit was **December 2024**,
  motioncanvas.io has been down, and the author stopped when his YouTube channel (the
  project's motivator) ended. Community fork **Canvas Commons** (canvascommons.io,
  archive at archive.canvascommons.io, 2800+ member Discord) continues it. Sources:
  [HN thread](https://news.ycombinator.com/item?id=47191103),
  [repo](https://github.com/motion-canvas/motion-canvas).
- **Revideo** ([midrender/revideo](https://github.com/midrender/revideo)) is the
  production fork that matters: **MIT**, adds a headless server-side rendering API,
  `<Video/>` with frame-accurate sync, `renderPartialVideo()` for parallelizing across
  workers, and it is **actively maintained** (v0.11.0 released July 10, 2026; steady
  commits through July 2026). Docs note a render job wants 8 to 10 GB RAM
  ([production docs](https://docs.re.video/rendering-in-production/)). The company
  pivoted to midrender.com (hosted rendering), so watch bus-factor, but the OSS repo
  is alive.
- As a compositor: canvas-draw-call engine (no DOM/CSS), generator-based animation
  API. Perfectly capable of drawImage + transforms + cursor sprites, and faster than
  Remotion for canvas-only scenes, but a smaller ecosystem, an unfamiliar generator
  programming model, and the upstream-abandonment cloud. Choose Revideo, never
  upstream Motion Canvas, if we go this way.

## 3. Custom compositor options

### (a) Node + skia-canvas / @napi-rs/canvas piping raw frames to FFmpeg

- Feasible and the cheapest to own. Loop: decode source PNG frame, `drawImage` the
  camera crop with easing, draw cursor sprite/ripples/key overlays, draw
  background/padding/shadow, write raw RGBA to an FFmpeg `rawvideo` stdin pipe.
- Libraries: [@napi-rs/canvas](https://github.com/Brooooooklyn/canvas) (Skia,
  prebuilt, fastest in the common benchmark: 68 ops/s vs node-canvas 60 vs skia-canvas
  47 on a draw+PNG-export test) and
  [skia-canvas](https://github.com/samizdatco/skia-canvas) (multi-threaded,
  **GPU-backed via Metal/Vulkan**, closer output parity with Chrome). No published
  4K60-pipeline benchmarks exist; budget a spike. Rough envelope: a 4K RGBA frame is
  33 MB; PNG decode of the source frame will dominate (tens of ms/frame
  single-threaded), so decode on a worker pool and expect 20 to 60 composited fps,
  with x264 encode running concurrently in FFmpeg.
- **Resampling**: Canvas 2D gives `imageSmoothingQuality` of `low`/`medium`/`high`,
  which in Skia maps to bilinear / bilinear+mipmaps / bicubic (Catmull-Rom). **No
  Lanczos**
  ([MDN](https://developer.mozilla.org/en-US/docs/Web/API/CanvasRenderingContext2D/imageSmoothingEnabled),
  [WHATWG](https://html.spec.whatwg.org/dev/canvas.html)). This is fine, and matches
  what Screen Studio-class tools ship, **as long as we never magnify past source
  native pixels** (see section 4). Bicubic upscaling of UI text beyond ~1.3x visibly
  softens; Lanczos would not save us there either.
- Motion blur: N sub-samples per output frame along the camera path,
  alpha-accumulated. Same brute force as Remotion but each sample is one GPU/CPU
  drawImage, not a full React render + screenshot, so it is an order of magnitude
  cheaper.

### (b) Pure FFmpeg filtergraph (zoompan/overlay/sendcmd)

Dismiss as the primary. `zoompan` **rounds x/y to integers per frame, producing
visible jitter on slow pans**; the standard workaround is pre-upscaling the source so
rounding error shrinks
([analysis](https://usercomp.com/news/1214682/smoothing-out-ffmpeg-zoom)), which at 4K
source means huge intermediates. Eased multi-keyframe camera paths become unreadable
expression strings; `sendcmd` scripts are write-only. The `geq` alternative is 100 to
1000x slower
([ffmpeg examples](https://hhsprings.bitbucket.io/docs/programming/examples/ffmpeg/manipulating_video_colors/use_of_geq_as_zoompan_alternative.html)).
Keep FFmpeg for decode/encode only.

### (c) GPU compositor

- **wgpu (Cap's approach)**: [Cap](https://github.com/CapSoftware/cap) renders its
  zoom/cursor/background pipeline with **custom wgpu forks + WGSL shaders**, FFmpeg
  via rust-ffmpeg fork, exports up to 4K60. Recent releases show exactly our feature
  set: "auto zooms ease out more naturally, zoom focus follows the smoothed cursor
  path, composited frames with smoother antialiasing"
  ([release notes](https://github.com/CapSoftware/Cap/releases)). Its rendering crates
  are the best reference implementation in the space. Cost: writing Rust + WGSL and
  owning a GPU pipeline; highest ceiling (shader motion blur, mipmapped/anisotropic
  sampling), highest build cost.
- **Compositor as a web page in the existing deterministic capture rig**: with a
  deterministic-browser rig, load a compositor page (Canvas2D or WebGL), feed it the
  frame sequence + event JSON, step virtual time, and capture. Prior art:
  [WebVideoCreator](https://github.com/Vinlic/WebVideoCreator) (virtual time +
  BeginFrame capture, Node+Puppeteer+FFmpeg) and
  [Replit's renderer](https://blog.replit.com/browsers-dont-want-to-be-cameras)
  (1200-line time-API shim, BeginFrame capture, WebCodecs for media; they rejected
  Remotion for flexibility reasons). Two capture paths:
  - DevTools screenshots: same throughput ceiling as Remotion (that is what Remotion
    is).
  - **In-page WebCodecs `VideoEncoder`**: hardware H.264 encode without pixels ever
    leaving the GPU; mixed real-world numbers (one report: ~25 fps at 4K on a 2018 MBP
    via WebCodecs vs 65-70 fps via FFmpeg+VideoToolbox,
    [w3c/webcodecs#492](https://github.com/w3c/webcodecs/issues/492); modern hardware
    does 4K60). Works, but muxing is re-implemented and codec flexibility is lost (VP9
    encode support is spottier).
  - Also note the OSS Screen Studio clone cluster proving the pattern:
    [OpenScreen](https://github.com/siddharthvaddem/openscreen),
    [recordly](https://github.com/WizardofTryout/recordly),
    [open-recorder](https://github.com/imbhargav5/open-recorder) ("generates automatic
    zooms from recorded click telemetry"),
    [screen-demo](https://github.com/njraladdin/screen-demo).

### (d) GStreamer / Blender VSE

Dismiss. GStreamer+GL gives a pipeline framework but the compositor would still be
hand-written GLSL with far worse dev ergonomics than any option above and no ecosystem
for this use case. Blender VSE is an interactive NLE; scripting eased camera paths,
cursor sprites, and click effects through its Python API is slower to build, slow to
render, and heavy to deploy in CI.

## 4. Image scaling quality for zoomed UI text

The governing inequality: with CSS viewport width `W_css`, capture `deviceScaleFactor
d`, output width `W_out`, a zoom factor `z` stays pixel-crisp iff `z ≤ (W_css × d) /
W_out` (the sampled region never magnifies past native pixels).

- **1080p output, capture at d=2** (e.g. 1920×1080 CSS → 3840×2160 native): crisp zoom
  budget up to **2.0x**. At **d=3** (5760×3240 native): budget **3.0x**. This covers
  everything Screen Studio-style videos do.
- **4K output** halves the budget: d=3 over a 1920 CSS viewport gives only 1.5x.
  Pixel-perfect 2.5x zoom at 4K would need d=5 (9600 px wide), which is impractical.
  Screen Studio itself upscales past native on deep zooms at 4K; bicubic magnification
  of text that was *rendered* at 2x-3x DPR degrades gracefully (it looks like slightly
  soft retina text, not ringing pixel soup), and motion plus blur hide it. Practical
  policy: **capture at d=3, master at 1080p60 or 1440p60 where the budget is real, and
  let 4K be a bonus container**, or cap zoom at the budget for the 4K master.
- **Is `deviceScaleFactor: 3` viable in Chromium?** Yes. Playwright/CDP support
  arbitrary `deviceScaleFactor`; screenshots come out at viewport×d
  ([Playwright #6188](https://github.com/microsoft/playwright/issues/6188),
  [guide](https://screenshotone.com/blog/how-to-render-screenshots-with-playwright/)).
  Raster cost grows ~d², which is irrelevant in a deterministic frame-stepped rig (not
  realtime). Stay under Chromium's ~16384 px surface limits; 5760×3240 is fine.
- **Resampler choice**: for minification and 1:1, mipmapped bilinear or bicubic
  (`imageSmoothingQuality: "high"`, Skia Catmull-Rom) is fully adequate; Lanczos's
  advantage is anti-aliased downscaling
  ([reference](https://upliftorch.com/tools/image-resize/en/blog/image-interpolation.html)),
  which supersampled capture already solves. **Supersampling the capture is worth more
  than any filter.** One extra option unique to this pipeline: the replay is
  deterministic, so specific segments can be re-captured at higher d or with CSS zoom
  applied if a hero shot ever needs true 3x at 4K.

## 5. Cursor rendering

- **Do not ship Apple's cursor images.** The macOS SLA prohibits redistribution and
  derivative works of Apple software components, and cursor bitmaps extracted from
  macOS are exactly that
  ([Apple SLA](https://www.apple.com/legal/sla/docs/macOSSequoia.pdf)); the Apple
  Design Resources license likewise restricts asset redistribution. Repos like
  [macOS-cursors-for-Windows](https://github.com/antiden/macOS-cursors-for-Windows)
  redistribute extracted assets without a real license; avoid.
- Options, in order of preference:
  1. **Draw our own vector cursor.** The arrow, pointing hand, and I-beam are trivial
     SVG shapes (white fill, black outline, soft drop shadow). Infinitely scalable for
     zooms, no licensing, and it is what the credible tools do (Cap renders its own
     cursor sprites; the OSS Screen Studio clones ship their own cursor pipelines).
  2. **[ful1e5/apple_cursor](https://github.com/ful1e5/apple_cursor)**: hand-recreated
     macOS-style cursors, **GPL-3.0**. Fine inside an internal tool; the GPL on the
     art is a nuisance we do not need given how simple the shapes are.
- Standard practice in this space: synthetic cursor drawn from the event timeline
  (position spline-smoothed, e.g. Catmull-Rom or critically-damped spring through the
  recorded points), scale-bounce on click, radial ripple on click. Cap's release notes
  confirm "zoom focus follows the smoothed cursor path" as the norm.

## 6. Encode settings

For crisp UI at 4K60 / 1080p60
([x264 guidance](https://forum.videohelp.com/threads/398764-x264-x265-Most-efficient-settings-for-Screen-recording),
[ASWF encoding guidelines](https://academysoftwarefoundation.github.io/EncodingGuidelines/Encodeh264.html),
[FFmpeg VP9 wiki](https://trac.ffmpeg.org/wiki/Encode/VP9),
[Google VP9 VOD settings](https://developers.google.com/media/vp9/settings/vod)):

**H.264 (compatibility master):**
```
ffmpeg -framerate 60 -i frames/%06d.png \
  -c:v libx264 -preset slow -crf 17 -profile:v high -level 5.2 \
  -pix_fmt yuv420p -vf "scale=out_color_matrix=bt709" -color_range tv \
  -colorspace bt709 -color_primaries bt709 -color_trc bt709 \
  -g 120 -movflags +faststart out_4k60.mp4
```
- CRF 17 to 18 at 4K, 18 to 20 at 1080p; UI content compresses extremely well (flat
  regions, static frames), so err low, the cost is small.
- **Chroma subsampling is the text killer**: 4:2:0 halves chroma resolution and
  fringes colored text. Mitigations: supersampled source (4:2:0 at 4K downscaled to a
  1080p display beats 4:4:4 at 1080p), or a `yuv444p` variant for docs pages we
  control (Chrome/Firefox/Edge play High 4:4:4 H.264; Safari historically does not, so
  keep 4:2:0 as the default deliverable).
- `+faststart` is mandatory for web docs (moov atom up front).

**VP9 (size-optimized web deliverable), two-pass constant quality:**
```
ffmpeg -framerate 60 -i frames/%06d.png -c:v libvpx-vp9 -b:v 0 -crf 15 \
  -pass 1 -row-mt 1 -tile-columns 2 -threads 8 -speed 4 -an -f null /dev/null
ffmpeg -framerate 60 -i frames/%06d.png -c:v libvpx-vp9 -b:v 0 -crf 15 \
  -pass 2 -row-mt 1 -tile-columns 2 -threads 8 -speed 1 out_4k60.webm
```
- Google's recommendation: CRF ~15 at 2160p, ~31 at 1080p, two-pass CQ mode,
  `-quality good`. `-row-mt 1` matters for encode speed. VP9 also supports 4:4:4 in
  profile 1 for a max-fidelity web variant.
- **Expected file sizes for a 60 to 90 s UI demo** (estimates; UI content bitrates are
  far below film): H.264 4K60 CRF 17: ~8 to 20 Mbps → **60 to 220 MB**. H.264 1080p60
  CRF 18: ~3 to 6 Mbps → **25 to 70 MB**. VP9 lands roughly 30 to 50% smaller at equal
  quality. If docs-page weight matters, serve 1080p60 VP9 (~15 to 40 MB) with the 4K
  H.264 as the "watch full quality" link.

---

## Comparison

| | Quality ceiling | Dev velocity | Throughput (90 s 4K60 job) | Licensing/cost | Maintenance risk |
|---|---|---|---|---|---|
| **Remotion** | High (full CSS/SVG/React; motion blur via multi-sampling; color-averaging artifact) | **Best**: templates for exactly this exist | ~3 to 8 min; 2x more with blur; degrades on hour-scale renders | Free ≤3 employees; else $25/seat/mo (Creators) or $0.01/render + $100/mo min (Automators); CI renders count; v5 license change pending | Low (well funded, active) |
| **Motion Canvas** | Medium-high (canvas only) | Medium (generator API) | Faster than Remotion per frame | MIT | **High: abandoned** (last commit Dec 2024); use Canvas Commons or Revideo |
| **Revideo** | Medium-high (canvas only, video-in supported) | Medium | Good; `renderPartialVideo` parallelism; 8-10 GB RAM/job | MIT, free self-hosted | Medium (active July 2026, but company pivoted to midrender.com) |
| **Node + @napi-rs/canvas or skia-canvas + FFmpeg pipe** | High (bicubic sampling; supersampled source makes it moot) | Medium: ~1-3 weeks for the full effect set | Best CPU option: est. 20 to 60 fps composite, encode concurrent | Free (MIT/MPL) | Low: ~1k lines we own, two boring deps |
| **Pure FFmpeg filtergraph** | Low (zoompan integer jitter) | Poor (write-only expressions) | Fast | Free | Unmaintainable; **dismissed** |
| **wgpu compositor (Cap-style)** | **Highest** (shader blur, aniso sampling) | Slowest (Rust + WGSL from scratch) | Fastest (GPU, realtime+) | Reference code AGPL (copy math only) | We own a GPU pipeline; overkill for docs videos |
| **Compositor web page in the capture rig** | High | Good (we have the rig) | Screenshot path = Remotion speed; WebCodecs path fast but H.264-only, custom muxing | Free | Rebuilds what Remotion already is |

## Recommendation

**Primary: Remotion**, PNG-frame-sequence source (`<Img src={staticFile(frame)}>`),
camera as an eased CSS transform, `<Trail>`/`<CameraMotionBlur>` applied only inside
camera-move sequences, cursor/ripples/keystrokes as React components driven directly
by the event JSON, background/padding/shadow as plain CSS. It is the only option where
the entire feature list is assembled from existing, documented parts, and the videos
are 60 to 90 s, which sits squarely inside the browser-render model's comfort zone
(minutes per video in CI). Mitigations: pin OffthreadVideo out (not needed), split any
long render, tune `--concurrency` via `npx remotion benchmark`, keep `samples` at 5 or
6 for blur, and get written confirmation from Remotion on Creators-vs-Automators
classification (worst case $100/month).

**Fallback (and eventual replacement if render minutes or licensing chafe): the custom
Node compositor**, @napi-rs/canvas (or skia-canvas for GPU) piping raw RGBA into
`libx264`. Identical output quality given a supersampled source, no license, no
browser in the loop, and the OSS Screen Studio clones as reference for zoom easing and
cursor smoothing math. The two share everything upstream (capture at dSF 3, event
JSON, camera-path solver, encode settings), so a later swap replaces only the
frame-drawing layer.

### Architecture sketch

```
capture rig (deterministic Chromium)
  viewport 1920x1080 CSS, deviceScaleFactor 3  → frames/%06d.png (5760x3240)
  event tap                                    → events.json (clicks/scroll/keys, t+xy)
        │
camera-path solver (pure TS, shared)
  events.json → zoom targets (cluster clicks, dwell windows)
  → keyframes {t, cx, cy, z} → C² eased spline (damped spring), clamp z ≤ budget
  → cursor spline (Catmull-Rom through samples, click bounce)
        │
compositor = Remotion composition (primary)
  <Img frame> in a transformed AbsoluteFill (camera)
  cursor SVG (own art) + ripple + key-overlay components
  CameraMotionBlur wrapper, active only during camera moves
  padding/background/shadow: CSS
  npx remotion render --concurrency=N
        │
encode: x264 crf17 high@5.2 yuv420p +faststart (master)
        libvpx-vp9 2-pass crf15/31 row-mt (web)
```

### Sources

- Remotion licensing: [LICENSE.md](https://github.com/remotion-dev/remotion/blob/main/LICENSE.md) · [License FAQ](https://www.remotion.dev/docs/license/faq) · [remotion.pro/license](https://www.remotion.pro/license)
- Remotion performance: [issue #4783](https://github.com/remotion-dev/remotion/issues/4783) · [issue #4949](https://github.com/remotion-dev/remotion/issues/4949) · [discussion #3070](https://github.com/orgs/remotion-dev/discussions/3070) · [4.0 benchmark](https://github.com/remotion-dev/4-0-benchmark) · [performance docs](https://www.remotion.dev/docs/performance/) · [benchmark CLI](https://www.remotion.dev/docs/cli/benchmark) · [gl options](https://www.remotion.dev/docs/gl-options) · [@remotion/skia](https://www.remotion.dev/docs/skia)
- Remotion motion blur: [@remotion/motion-blur](https://www.remotion.dev/docs/motion-blur/) · [CameraMotionBlur](https://www.remotion.dev/docs/motion-blur/camera-motion-blur) · [Trail](https://www.remotion.dev/docs/motion-blur/trail)
- Remotion screencast prior art: [remotion-cinematic](https://github.com/codeverbojan/remotion-cinematic) · [reactvideoeditor/remotion-templates](https://github.com/reactvideoeditor/remotion-templates)
- Motion Canvas / Revideo: [HN abandonment thread](https://news.ycombinator.com/item?id=47191103) · [motion-canvas repo](https://github.com/motion-canvas/motion-canvas) · [midrender/revideo](https://github.com/midrender/revideo) · [Revideo production docs](https://docs.re.video/rendering-in-production/) · [PkgPulse comparison](https://www.pkgpulse.com/guides/remotion-vs-motion-canvas-vs-revideo-programmatic-video-2026)
- Canvas libraries: [@napi-rs/canvas](https://github.com/Brooooooklyn/canvas) · [skia-canvas](https://github.com/samizdatco/skia-canvas) · [PkgPulse canvas benchmark](https://www.pkgpulse.com/guides/node-canvas-vs-napi-rs-canvas-vs-skia-canvas-server-2026)
- Resampling: [MDN imageSmoothingEnabled](https://developer.mozilla.org/en-US/docs/Web/API/CanvasRenderingContext2D/imageSmoothingEnabled) · [WHATWG canvas spec](https://html.spec.whatwg.org/dev/canvas.html) · [interpolation explainer](https://upliftorch.com/tools/image-resize/en/blog/image-interpolation.html)
- FFmpeg zoompan: [jitter analysis](https://usercomp.com/news/1214682/smoothing-out-ffmpeg-zoom) · [geq alternative cost](https://hhsprings.bitbucket.io/docs/programming/examples/ffmpeg/manipulating_video_colors/use_of_geq_as_zoompan_alternative.html)
- GPU / capture-rig compositors: [Cap repo](https://github.com/CapSoftware/cap) · [Cap releases](https://github.com/CapSoftware/Cap/releases) · [Replit rendering engine](https://blog.replit.com/browsers-dont-want-to-be-cameras) · [WebVideoCreator](https://github.com/Vinlic/WebVideoCreator) · [WebCodecs perf issue #492](https://github.com/w3c/webcodecs/issues/492)
- OSS Screen Studio clones: [OpenScreen](https://github.com/siddharthvaddem/openscreen) · [recordly](https://github.com/WizardofTryout/recordly) · [open-recorder](https://github.com/imbhargav5/open-recorder) · [screen-demo](https://github.com/njraladdin/screen-demo)
- High-DPI capture: [Playwright #6188](https://github.com/microsoft/playwright/issues/6188) · [Playwright screenshot guide](https://screenshotone.com/blog/how-to-render-screenshots-with-playwright/)
- Cursors: [macOS SLA](https://www.apple.com/legal/sla/docs/macOSSequoia.pdf) · [Apple Design Resources license](https://developer.apple.com/support/downloads/terms/apple-design-resources/Apple-Design-Resources-License-20230621-English.pdf) · [ful1e5/apple_cursor (GPL-3.0)](https://github.com/ful1e5/apple_cursor)
- Encoding: [x264 screen recording thread](https://forum.videohelp.com/threads/398764-x264-x265-Most-efficient-settings-for-Screen-recording) · [ASWF H.264 guidelines](https://academysoftwarefoundation.github.io/EncodingGuidelines/Encodeh264.html) · [FFmpeg VP9 wiki](https://trac.ffmpeg.org/wiki/Encode/VP9) · [Google VP9 VOD settings](https://developers.google.com/media/vp9/settings/vod)
