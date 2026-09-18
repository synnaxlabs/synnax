# Capturing pixel-perfect 60fps video of a scripted Chromium session: research report

Scope: deterministic, stutter-free, high-DPI (DSF 2, up to 4K) capture of a scripted
Playwright session driving the Synnax Console web build, post-processed into a demo
video.

---

## 1. Deterministic frame capture: `HeadlessExperimental.beginFrame` + BeginFrameControl

### What it is
BeginFrames are Chromium's internal vsync signal. BeginFrameControl (BFC) replaces the
vsync with manual CDP-issued frames: each `HeadlessExperimental.beginFrame` call runs
one atomic layout -> paint -> composite -> screenshot cycle and returns `{ hasDamage,
screenshotData }`. Parameters: `frameTimeTicks` (renderer clock timestamp in ms),
`interval` (reported frame interval, default 16.666), `noDisplayUpdates` (run side
effects without producing a frame), `screenshot: { format: png|jpeg|webp, quality,
optimizeForSpeed }`. Sources: [CDP HeadlessExperimental docs](https://chromedevtools.github.io/devtools-protocol/tot/HeadlessExperimental/),
[headless-dev: rendering control](https://groups.google.com/a/chromium.org/g/headless-dev/c/S5CoLs46AiE),
[headless-dev: frame control examples](https://groups.google.com/a/chromium.org/g/headless-dev/c/WZtYOO-x1Hc).

### Current status (this is the critical part)
- **`--headless=new` never supported it.** BFC requires `Target.createTarget` with
  `enableBeginFrameControl: true`, which only the old headless architecture implements
  ([puppeteer #11315](https://github.com/puppeteer/puppeteer/issues/11315),
  [puppeteer #3411](https://github.com/puppeteer/puppeteer/issues/3411)).
- Old headless was split out of the main binary as **`chrome-headless-shell`** (Chrome
  for Testing artifact since M118); `--headless=old` stopped working in the main
  binary at **M132** ([Chromium headless README](https://chromium.googlesource.com/chromium/src/+/lkgr/headless/README.md),
  [Chrome headless docs](https://developer.chrome.com/docs/chromium/headless)).
- **Chromium 147+ removed `HeadlessExperimental.beginFrame` from the main binary
  entirely**; tools that need it must ship `chrome-headless-shell`, which still
  supports it ([HyperFrames troubleshooting](https://github.com/NousResearch/hermes-agent/blob/main/optional-skills/creative/hyperframes/references/troubleshooting.md),
  [hyperframes issue #294](https://github.com/heygen-com/hyperframes/issues/294)). So
  today: **`chrome-headless-shell` only**, a legacy-architecture binary with an
  uncertain long-term future.
- **Platform limits: not supported on macOS.** `enableBeginFrameControl` works only on
  Windows and Linux ([headless-dev thread](https://groups.google.com/a/chromium.org/g/headless-dev/c/S5CoLs46AiE));
  HyperFrames reports it is **reliable on Linux only**, with macOS/Windows falling
  back to `Page.captureScreenshot`
  ([HeyGen HyperFrames writeup](https://www.heygen.com/research/html-to-video));
  puppeteer-capture reports Linux + Windows, explicitly not macOS
  ([why puppeteer-capture](https://alexey-pelykh.com/blog/why-i-built-puppeteer-capture/)).

### Required flags (the HyperFrames/WebVideoCreator recipe)
```
chrome-headless-shell \
  --deterministic-mode \
  --enable-begin-frame-control \
  --run-all-compositor-stages-before-draw \
  --disable-threaded-animation \
  --disable-threaded-scrolling \
  --disable-checker-imaging \
  --disable-image-animation-resync \
  --enable-surface-synchronization \
  --enable-unsafe-swiftshader        # WebGL without GPU (Chrome 137+ removed auto fallback)
```
Under `--deterministic-mode`, `performance.now()` is driven by the `frameTimeTicks`
passed to `beginFrame`, not the system clock, so rAF, CSS animations, and WAAPI all
advance exactly one tick per captured frame. Byte-identical output across runs is
achievable ([puppeteer-capture](https://alexey-pelykh.com/blog/why-i-built-puppeteer-capture/),
[HyperFrames](https://www.heygen.com/research/html-to-video)).

### Known gotchas
- **Event loop stalls**: with BFC on, the event loop stops ticking autonomously;
  `document.fonts.ready` and similar promises hang. Fix: a warmup loop firing
  `beginFrame` with `noDisplayUpdates: true` during load (HyperFrames, and Replit's
  ~30fps invisible warmup loop)
  ([Replit engine writeup](https://blog.replit.com/browsers-dont-want-to-be-cameras)).
- **`<video>` elements do not obey virtual/frame time.** Every production system
  pre-extracts video frames with ffmpeg and swaps in `<img>`/canvas per tick (Replit's
  5-layer mp4box.js + WebCodecs pipeline; HyperFrames' JPEG-per-frame injection). Not
  relevant for the Console unless a demo embeds video.
- **Workers and OffscreenCanvas: direct risk for Synnax.** Replit explicitly
  **disabled `OffscreenCanvas` and `transferControlToOffscreen`** because
  worker-thread rendering bypasses deterministic capture
  ([Replit](https://blog.replit.com/browsers-dont-want-to-be-cameras)). The Console
  renders charts to canvas/WebGL in the Aether offscreen worker; the worker has its
  own `performance.now`/rAF clock that neither JS time patching nor
  `--deterministic-mode` guarantees to quantize with the main thread. Worker commits
  do ride the BeginFrame signal, but the worker's notion of "now" and any internal
  timers are not seeked per tick. **Any deterministic approach needs an app-level
  seam: a capture mode where the Aether render loop takes its timestamps from an
  injected, steppable time source (or renders on the main thread).**
- **Rendering is SwiftShader (software)** in `chrome-headless-shell` without a GPU:
  correct but slow WebGL, sometimes lower shader quality, and the GPU process can
  busy-spin a CPU core in containers
  ([SwiftShader docs](https://chromium.googlesource.com/chromium/src/+/main/docs/gpu/swiftshader.md),
  [chromedp #1073](https://github.com/chromedp/chromedp/issues/1073),
  [Intent to Remove SwiftShader fallback](https://groups.google.com/a/chromium.org/g/blink-dev/c/yhFguWS_3pM)).
  Text/AA rendering can differ slightly from GPU output; render final frames on one
  fixed platform.
- Chromium 117-120 had intermittent crashes with the deterministic flags
  ([puppeteer-capture](https://alexey-pelykh.com/blog/why-i-built-puppeteer-capture/));
  long since fixed but pin the browser version regardless.
- 4K at DSF 2 means a 7680x4320 software-rasterized surface per frame; expect frame
  times in the hundreds of ms. This is offline rendering: throughput does not affect
  output quality.

### Playwright vs Puppeteer/raw CDP
Playwright cannot use this. It never passes `enableBeginFrameControl` when creating
targets and does not expose the domain. Playwright does ship its own "chromium
headless shell" for headless runs
([Playwright browsers doc](https://playwright.dev/docs/browsers)) but its target
creation still lacks the flag. The proven stacks are **Puppeteer +
`chrome-headless-shell`** (WebVideoCreator, timecut, puppeteer-capture, HyperFrames,
Replit) or raw CDP (`Target.createTarget({ enableBeginFrameControl: true })`, attach,
then drive). A Playwright script would have to be ported, or the session driven with
raw CDP input events. Ecosystem references:
[WebVideoCreator](https://github.com/Vinlic/WebVideoCreator) (the pioneering
implementation; built against old headless, needs the headless-shell binary today),
[puppeteer-capture](https://alexey-pelykh.com/blog/why-i-built-puppeteer-capture/)
(60fps H.264 MP4 default, byte-identical output).

### Virtual time (`Emulation.setVirtualTimePolicy`) on its own
Policies: `advance`, `pause`, `pauseIfNetworkFetchesPending`, plus `budget` (fires
`virtualTimeBudgetExpired`) and `maxVirtualTimeTaskStarvationCount` to break
timer-loop deadlock
([CDP Emulation domain](https://chromedevtools.github.io/devtools-protocol/tot/Emulation/)).
Virtual time advances **timers, Date, and task scheduling**, and pauses on pending
fetches (good for settling pages). But rendering (rAF cadence, CSS animation clock,
compositor) stays wall-clock unless combined with BFC; the sanctioned pattern is
virtual time **plus** manual BeginFrames, per Eric Seckler on headless-dev
([60fps WebGL thread](https://groups.google.com/a/chromium.org/g/headless-dev/c/s8ttGCh8jzM),
[CSS animation capture thread](https://groups.google.com/a/chromium.org/g/headless-dev/c/QBQEm5Yd3_E)).
Known deadlocks: budget never expiring on pages with pathological timers
([chrome-headless-render-pdf #29](https://github.com/Szpadel/chrome-headless-render-pdf/issues/29)).
Virtual time does not fix `<video>`; media playback ignores it.

### JS time patching: timecut/timesnap/timeweb (tungs)
[timeweb](https://github.com/tungs/timecut) monkey-patches `Date`, `Date.now`,
`performance.now`, `requestAnimationFrame`, `setTimeout`, `setInterval` (via
`evaluateOnNewDocument`), then timesnap screenshots per virtual tick and timecut runs
ffmpeg. Fidelity limits, from its own README:
- **CSS transitions/animations are NOT captured correctly**: "pages where changes
  occur via other means (e.g. through transitions/animations from CSS rules) will
  likely not render as intended"
  ([timecut README](https://github.com/tungs/timecut/blob/main/README.md)). The CSS
  animation clock is not a JS function; patching JS does not touch it. For a React app
  leaning on CSS transitions this is disqualifying unless every animation is
  additionally seeked via `document.getAnimations()` per tick.
- **Main thread only.** Workers keep real clocks: the Synnax offscreen chart worker
  would free-run against the frozen main-thread clock. Same flag as above: needs an
  app seam.
- Throughput is screenshot-bound (same as approach 5).

Playwright's built-in [Clock API](https://playwright.dev/docs/clock)
(`page.clock.install()` before navigation) is the same idea, maintained and
integrated: it fakes `Date`, `performance.now`, `setTimeout`, `setInterval`, and
`requestAnimationFrame`, with `pauseAt`/`runFor`/`fastForward`
([clock.md](https://github.com/microsoft/playwright/blob/main/docs/src/clock.md)).
Same two blind spots: CSS animation clock and workers.

---

## 2. `Page.startScreencast` (and Playwright's video recorder)

Mechanics: browser pushes base64 JPEG/PNG `screencastFrame` events; each is acked with
`screencastFrameAck` (the ack is the throttle). `maxWidth`/`maxHeight` downscale the
surface; `everyNthFrame` subsamples
([CDP Page domain](https://chromedevtools.github.io/devtools-protocol/tot/Page/)).

Hard limits:
- **Variable frame rate, ~30fps practical ceiling** over the CDP websocket; idle pages
  emit no frames at all; a long-standing feature request for an fps option was never
  implemented ([devtools-protocol #63](https://github.com/ChromeDevTools/devtools-protocol/issues/63),
  [screencast rate thread](https://groups.google.com/a/chromium.org/g/headless-dev/c/5rxhZntIYSM),
  [vercel agent-browser #632](https://github.com/vercel-labs/agent-browser/issues/632)
  on HiDPI maxWidth). Frames arrive when the compositor damages, timed by wall clock:
  system load = jitter.
- **Playwright `recordVideo` quality problems (documented)**: VP8 WebM with
  **hardcoded ffmpeg args: ~1 Mbps target bitrate, single thread, realtime CPU cap**,
  visible mosquito noise around glyphs; maintainers repeatedly declined to expose
  tuning ([#31424](https://github.com/microsoft/playwright/issues/31424),
  [#12056](https://github.com/microsoft/playwright/issues/12056),
  [#7246 blurry/wrong resolution](https://github.com/microsoft/playwright/issues/7246)).
  The screencast server **locks frame size to the first client**: if tracing
  screenshots start first, the recorder gets an **800x450** stream regardless of the
  requested size ([#34282](https://github.com/microsoft/playwright/issues/34282)).
  VFR-to-CFR conversion duplicates frames by wall-clock deltas, with an
  accumulated-timestamp-error bug causing unstable frame rate
  ([#35776](https://github.com/microsoft/playwright/issues/35776)).
- [playwright-recorder-plus](https://github.com/MuTsunTsai/playwright-recorder-plus)
  (wraps the public `page.screencast` in Playwright 1.59+) fixes the encode side
  (two-pass H.264/VP9/AV1, presets) but cannot fix the source: still wall-clock,
  still VFR, still JPEG-frame-fed.

**Verdict: fine for CI artifacts, disqualified for this project.** Fidelity ceiling is
JPEG-compressed frames at uneven ~15-30fps.

---

## 3. Real-time capture of a headed browser

### Linux: Xvfb + ffmpeg x11grab
Run headed Chromium on an Xvfb display sized 3840x2160 (24-bit), grab with `ffmpeg -f
x11grab -framerate 60 -i :99`. Realities:
- Raw 4K60 BGRA is ~1.9 GB/s. Real-time encode needs `libx264 -preset ultrafast -qp 0`
  (or `-crf 18`) with many threads, or dump rawvideo/lossless first and transcode
  after; single-threaded encode pegs a core and drops frames
  ([Arch forum framedrop thread](https://bbs.archlinux.org/viewtopic.php?id=205920),
  [x11grab framerate as input option](https://bbs.archlinux.org/viewtopic.php?id=160150)).
  CI runners (4-8 shared vCPUs, no GPU) will drop frames at 4K60; 1080p60 is the
  realistic ceiling without NVENC
  ([x11docker GPU recording issue](https://github.com/mviereck/x11docker/issues/199),
  [jperl/record-screen](https://github.com/jperl/record-screen)).
- Xvfb has no vsync/compositor: x11grab samples the framebuffer asynchronously, so it
  can catch mid-paint states, and the app itself renders at whatever rate the loaded
  CPU manages. Non-deterministic by construction: every run differs.
- DSF 2 works (`--force-device-scale-factor=2`) but doubles pixel throughput again.

### macOS (dev machines)
- **ffmpeg avfoundation** (`-f avfoundation -framerate 60 -capture_cursor 0 -i
  "N:none"`) uses the **deprecated `AVCaptureScreenInput`**: display-scoped only (no
  window scoping), increasingly broken on Sequoia (multi-display capture broken, TCC
  alerts for deprecated capture APIs)
  ([Apple forums: Sequoia capture](https://developer.apple.com/forums/thread/769214),
  [SCK vs AVFoundation](https://developer.apple.com/forums/thread/736022),
  [AVDevice ignoring 60fps](https://developer.apple.com/forums/thread/765249)).
  Retina capture works (captures physical pixels) but treat it as legacy.
- **ScreenCaptureKit** (macOS 12.3+) is the correct API: GPU-backed, low CPU,
  **window-scoped capture** (`SCWindow`), full Retina resolution, solid 60fps into
  `AVAssetWriter` with `h264_videotoolbox`/HEVC. ffmpeg has no SCK input device; use a
  small Swift capture utility or an SCK-based recorder
  ([ScreenCaptureKit docs](https://developer.apple.com/documentation/screencapturekit),
  [screencapturekit-rs](https://github.com/svtlabs/screencapturekit-rs),
  [ffmpeg+SCK writeup](https://www.ashleyarthur.co.uk/posts/2025/ffmpeg_rust_screencapture/)).
  Beware SCK throttling to content change rate; request explicit
  `minimumFrameInterval`
  ([SCK 7fps thread](https://developer.apple.com/forums/thread/811300)).
- Capturing a headed Playwright-driven Chromium window on a Mac Studio is genuinely
  viable for **near-final** quality: real GPU rendering, real font smoothing, 60fps
  window-scoped SCK capture. But it is wall-clock: any GC pause, network hiccup, or
  busy core becomes a permanent stutter in the recording, and repeat runs never match
  cut-for-cut.

**Verdict: best "live" ceiling, zero determinism. Acceptable only if re-recording
takes like a human screen-recorder would is acceptable.**

---

## 4. In-page capture

- **`canvas.captureStream()` + MediaRecorder**: captures one canvas, not the DOM UI,
  so it cannot record the Console. For completeness: VFR output with highly
  inconsistent frame rate, frames dropped silently under load, no "prefer consistent
  fps" knob
  ([w3c/mediacapture-record #177](https://github.com/w3c/mediacapture-record/issues/177),
  [Mozilla bug 1231848](https://bugzilla.mozilla.org/show_bug.cgi?id=1231848)),
  background-tab throttling
  ([Mozilla bug 1344524](https://bugzilla.mozilla.org/show_bug.cgi?id=1344524)),
  large-canvas encoder failures
  ([Chrome bug 897727](https://paul.kinlan.me/chrome-bug-897727mediarecorder-using-canvas-capturestreamfails-for-large-canvas-elements-on-android/)).
- **getDisplayMedia / `chrome.tabCapture` in an extension**
  ([puppeteer-stream](https://www.npmjs.com/package/puppeteer-stream)): requires
  headful (getDisplayMedia is unsupported headless,
  [puppeteer #4404](https://github.com/puppeteer/puppeteer/issues/4404)); realtime
  MediaRecorder encode with the same VFR/drop problems; 60fps unreliable
  ([puppeteer-stream #27](https://github.com/samuelscheit/puppeteer-stream/issues/27)).
- **WebCodecs `VideoEncoder`**: excellent offline encoder (hardware-accelerated,
  encode at your own pace, strict timestamp control; a 3-minute video in ~30s on
  modern hardware)
  ([Chrome WebCodecs guide](https://developer.chrome.com/docs/web-platform/best-practices/webcodecs),
  [canvas-to-WebCodecs walkthrough](https://sabigara.com/blog/capture-frames-and-encode)).
  But the page cannot screenshot itself: WebCodecs needs a frame source, so for
  whole-app capture it degenerates into "CDP screenshots piped into an encoder".

**Verdict: not applicable to whole-app capture, except WebCodecs as an optional encode
stage.**

---

## 5. Screenshot stepping (pause/step the clock, PNG per 1/60s tick)

The cross-platform deterministic approach, and the one that keeps Playwright as the
driver.

### Mechanics
1. `page.clock.install()` **before** `goto` (scripts capture timer refs at eval time)
   ([Playwright clock docs](https://playwright.dev/docs/clock)).
2. Drive the session; at recording sections: `page.clock.pauseAt(t)`, then per frame
   `page.clock.runFor(16.6667)` which fires timers and rAF callbacks deterministically.
3. Per tick, seek compositor-owned animations that the fake clock does not reach:
   `document.getAnimations().forEach(a => a.currentTime = t)` covers CSS transitions,
   CSS animations, and WAAPI. (This is the fix for the timeweb CSS blind spot.)
4. Screenshot per tick via raw CDP for speed: `Page.captureScreenshot { format:
   "png", fromSurface: true, optimizeForSpeed: true, captureBeyondViewport: false }`
   on a `page.context().newCDPSession(page)`. Avoid `page.screenshot()` in the hot
   loop; it adds per-call CDP round-trips (layout metrics, stability waits)
   ([puppeteer #3502](https://github.com/puppeteer/puppeteer/issues/3502),
   [screenshotone speed guide](https://screenshotone.com/blog/optimize-for-speed-when-rendering-screenshots-in-puppeteer-and-chrome-devtools-protocol/)).
5. Assemble with ffmpeg (section 6).

### Throughput expectations
Reported figures: 60-90 ms per screenshot typical, requests above ~10/s can back up
([puppeteer #1656](https://github.com/puppeteer/puppeteer/issues/1656)); large
viewports are much slower
([puppeteer #736](https://github.com/puppeteer/puppeteer/issues/736)); historic ~1/6 s
floor on macOS headful `fromSurface` paths
([puppeteer #476](https://github.com/puppeteer/puppeteer/issues/476)). PNG encode
dominates at high resolution; `optimizeForSpeed: true` helps materially
([8 tips, Bannerbear](https://www.bannerbear.com/blog/ways-to-speed-up-puppeteer-screenshots/)).
Realistic planning numbers at 1920x1080 CSS x DSF 2 (3840x2160 physical): **4-10 fps
capture rate**, so a 60 s / 3600-frame video renders in **6-15 minutes**. At native 4K
CSS x2 (7680x4320): 2-4x slower. Disk: 3600 PNGs at 4K UI content ≈ 2-8 MB each ≈
10-25 GB; use a scratch SSD or pipe frames straight into ffmpeg via `image2pipe`.

### Gotchas
- **Each `captureScreenshot fromSurface:true` forces a composite**; with the clock
  paused between ticks the page is static, so the extra composite is idempotent and
  harmless. Do not let any real-time animation (spinner driven by an un-patched clock,
  worker) run between tick and shot.
- **Workers again**: `page.clock` does not reach the Aether offscreen worker. Without
  an app seam, charts will either freeze (worker waiting on rAF that never fires
  meaningfully) or free-run (real `performance.now`). **Concrete recommendation: add a
  capture-mode time source to the Console/Pluto telemetry + render loop, injected at
  construction, that the Playwright script advances via an exposed binding each
  tick.** Then chart sweep lines, streaming data windows, and decimation all step in
  lockstep with the captured clock.
- **Text rendering consistency**: perfectly consistent within one run and one
  platform. Across platforms, font rasterization differs (Linux FreeType vs macOS
  CoreText); render all final takes on one platform.
- **GPU vs SwiftShader for the WebGL charts**: headless=new supports real GPU
  (`--use-angle=metal` on macOS just works on dev machines); Linux CI without GPU
  needs `--enable-unsafe-swiftshader` since Chrome 137 removed automatic SwiftShader
  fallback ([Intent to Remove](https://groups.google.com/a/chromium.org/g/blink-dev/c/yhFguWS_3pM),
  [SwiftShader doc](https://chromium.googlesource.com/chromium/src/+/main/docs/gpu/swiftshader.md)).
  GPU and SwiftShader output differ subtly in AA and shader precision; again, one
  platform for final renders. Docker: raise `/dev/shm` above the 64 MB default or
  WebGL crashes.
- **Determinism grade**: "visually deterministic", not byte-identical like BFC (GPU
  rasterization and thread scheduling can produce sub-pixel-identical but not
  bit-identical frames). For a demo video this is indistinguishable.
- **No OS cursor in headless**: inject a DOM cursor element and move it from the
  script, or synthesize the cursor entirely in post from the event timeline.

---

## 6. Color and encode chain

### sRGB correctness
- Launch Chromium with `--force-color-profile=srgb` so rendering does not adapt to the
  host display profile; screenshots then come out as untagged sRGB and are identical
  on every machine.
- Do not let ffmpeg guess color metadata. PNGs are full-range RGB; convert explicitly
  to limited-range BT.709 (the assumption every browser/player makes for HD/UHD
  video):

```
ffmpeg -framerate 60 -i frames/%05d.png \
  -vf "scale=in_range=full:out_range=tv:out_color_matrix=bt709:flags=lanczos+accurate_rnd+full_chroma_int,format=yuv420p" \
  -c:v libx264 -preset slow -crf 16 -profile:v high -g 120 \
  -color_primaries bt709 -color_trc bt709 -colorspace bt709 -color_range tv \
  -movflags +faststart out.mp4
```

### Chroma subsampling: yuv444 vs yuv420
- yuv444 keeps UI text and 1px hairlines exactly; yuv420 halves chroma resolution,
  softening colored text edges (red/blue text on contrasting backgrounds suffers
  most).
- **But yuv444 H.264 (High 4:4:4 Predictive) does not play in Safari or Firefox, and
  most hardware decoders reject it**; only Chromium's bundled software decoder handles
  it ([Mozilla bug 1368063](https://bugzilla.mozilla.org/show_bug.cgi?id=1368063),
  [motioneye #1067](https://github.com/ccrisan/motioneye/issues/1067),
  [ASWF encoding guidelines](https://academysoftwarefoundation.github.io/EncodingGuidelines/Encodeh264.html),
  [NVDEC no-444](https://forums.developer.nvidia.com/t/using-yuv444-for-encoding/323606)).
  yuv444 is a non-starter for docs-site embedding.
- **The standard mitigation is supersampling: encode at 4K (2x the CSS layout size,
  which the DSF-2 capture already gives) and display in a 1x-sized `<video>`
  element.** The player's downscale averages 4 luma + 1 chroma sample per displayed
  pixel, effectively restoring full-chroma text at display size.

### Codec/settings recommendations for crisp UI at 4K60
- **H.264 (universal fallback)**: libx264, `-preset slow`/`veryslow`, **CRF 15-17**
  for UI (text needs lower CRF than film), High profile, yuv420p, no `-tune`. Expect
  ~10-25 Mbps at 4K60 for UI content: **~75-190 MB per minute**.
- **VP9 (webm)**: `-c:v libvpx-vp9 -b:v 0 -crf 24 -row-mt 1 -deadline good -cpu-used
  2`; ~40-50% smaller than x264 at matched quality; slow to encode.
- **AV1 (best size, modern docs sites)**: `-c:v libsvtav1 -preset 5 -crf 30
  -svtav1-params tune=0`; roughly half of VP9 again. Playback support is
  near-universal in 2026 browsers, but older Safari/hardware lacks AV1 decode, so
  never ship AV1-only.
- **Docs-site delivery convention**: `<video muted autoplay loop playsinline>` with
  `<source type="video/mp4">` H.264 as the guaranteed path, optionally an AV1/MP4 or
  VP9/WebM `<source>` first for size. H.264-in-MP4 remains the only format that plays
  everywhere ([browser support summary](https://www.lambdatest.com/web-technologies/mpeg4)).
  For a 4K60 embed, also produce a 1080p60 variant.
- H.265/HEVC: skip; no Firefox support, patent-encumbered, AV1 beats it for this use.

---

## Comparison matrix

| Approach | Fidelity ceiling | Determinism | fps consistency | DSF 2 / 4K | Canvas/WebGL (worker) | Complexity | CI-friendly | Playwright-native |
|---|---|---|---|---|---|---|---|---|
| 1. beginFrame + BFC (`chrome-headless-shell`) | Lossless PNG per frame | Byte-identical | Perfect (offline) | Yes (slow, SwiftShader) | WebGL ok via SwiftShader; **worker rendering must be disabled/seamed** | High (raw CDP/Puppeteer, warmup loop, legacy binary) | Linux only | No (port to Puppeteer/CDP) |
| 1b. Virtual time alone | n/a (not a capture method) | Timers/Date only; rendering stays wall-clock | n/a | n/a | n/a | Low | Yes | Via CDP session |
| 1c. timeweb/timecut | Lossless PNG | High for JS-driven motion; **CSS transitions wrong**; workers wrong | Perfect (offline) | Yes | Main-thread only | Medium | Yes | No (Puppeteer) |
| 2. `Page.startScreencast` / Playwright recordVideo | JPEG frames, VP8 1 Mbps | None | ~15-30fps, VFR, idle gaps | maxWidth downscale traps | Whatever renders | Low | Yes | Yes (built in) |
| 3a. Xvfb + x11grab | Encoder-limited; real render | None | Drops at 4K60 on CI CPUs | Yes at high CPU cost | Full (real pipeline) | Medium | Yes but flaky | Yes (headed) |
| 3b. macOS ScreenCaptureKit (window) | Excellent (GPU, Retina, 60fps) | None (wall clock) | Good on a strong Mac | Native Retina | Full | Medium (Swift helper) | No (macOS runner needed) | Yes (headed) |
| 4. MediaRecorder / getDisplayMedia / WebCodecs | Realtime-encoder limited (except WebCodecs offline) | None | VFR, silent drops | Canvas-size limits | Canvas-only or tab capture | Medium | Poor (headful needed) | Partially |
| 5. Clock-step + `Page.captureScreenshot` | Lossless PNG per frame | Visually deterministic (not byte-exact) | Perfect (offline) | Yes (4-10 cap fps at 2x1080p) | WebGL ok (real GPU on dev Macs); **worker needs app seam** | Medium | Yes (macOS + Linux) | **Yes** (`page.clock` + CDP session) |

---

## Recommendation

### Primary: approach 5, clock-stepped lossless screenshot rendering, driven by Playwright
It is the only approach that is simultaneously deterministic, lossless, 60fps-exact,
DSF-2/4K capable, cross-platform (dev on macOS, CI on Linux), and keeps the existing
Playwright scripting model. The offline render cost (~10-15 min per 60 s of video) is
irrelevant for demo production.

Sketch:

```ts
const browser = await chromium.launch({
  args: [
    "--force-color-profile=srgb",
    "--force-device-scale-factor=2",   // belt and braces with deviceScaleFactor
    "--disable-lcd-text",              // optional: subpixel AA fringes survive scaling badly
    "--enable-unsafe-swiftshader",     // Linux CI without GPU only
    "--hide-scrollbars",
  ],
});
const page = await browser.newPage({
  viewport: { width: 1920, height: 1080 },
  deviceScaleFactor: 2,                 // 3840x2160 physical frames
});
await page.clock.install({ time: T0 }); // BEFORE goto
await page.goto(consoleUrl);
// ... scripted setup ...

const cdp = await page.context().newCDPSession(page);
const TICK = 1000 / 60;
for (let f = 0; f < 3600; f++) {
  await page.clock.runFor(TICK);                       // timers + rAF
  await page.evaluate((t) => {                          // CSS transitions/animations
    document.getAnimations().forEach((a) => { try { a.currentTime = t; } catch {} });
  }, T0ms + f * TICK);
  await page.evaluate((t) => window.__synnaxCapture?.tick(t), f * TICK); // app seam: step Aether worker
  const { data } = await cdp.send("Page.captureScreenshot", {
    format: "png", fromSurface: true, optimizeForSpeed: true, captureBeyondViewport: false,
  });
  await writeFrame(f, data);
}
```

Then the ffmpeg command from section 6 (H.264 CRF 16 master, plus an AV1/WebM variant
for the docs site).

**The one mandatory engineering task regardless of approach: a capture-mode time seam
for the Aether offscreen-canvas worker.** Every deterministic technique (BFC, timeweb,
Playwright clock) stops at the worker boundary; Replit had to disable OffscreenCanvas
entirely. An injected, steppable time source for the worker render loop (exposed to
the script via `exposeBinding`/`postMessage`) turns the biggest risk into a non-issue.

Interleave scripted interactions between ticks (Playwright input events dispatch
immediately; hover/active states then render on the next tick). Add a DOM cursor
element for pointer visualization, or synthesize the cursor in post.

### Fallback / maximal-determinism variant: approach 1, `chrome-headless-shell` + BeginFrameControl on Linux CI
If the clock-step pipeline shows residual nondeterminism (compositor-driven effects
that `getAnimations()` cannot seek, scroll animations, GIF/image animations), move
final rendering to the WebVideoCreator/HyperFrames stack: Puppeteer +
`chrome-headless-shell` + the flag set above + `HeadlessExperimental.beginFrame` with
a PNG screenshot per frame and a `noDisplayUpdates` warmup loop. Costs: Linux-only,
port the driver script off Playwright, SwiftShader rendering, and a dependency on a
legacy binary whose API was already deleted from mainline Chromium 147+. Study
[WebVideoCreator](https://github.com/Vinlic/WebVideoCreator) and
[puppeteer-capture](https://alexey-pelykh.com/blog/why-i-built-puppeteer-capture/)
before building.

### Explicitly rejected
- Playwright `recordVideo`/screencast: hardcoded 1 Mbps VP8, VFR, 800x450 downscale
  trap; quality floor far below requirement.
- Real-time capture (Xvfb/x11grab, macOS SCK): keep macOS SCK window capture in the
  toolbox for quick previews, never for the final deterministic renders.
- MediaRecorder/getDisplayMedia: VFR and silent frame drops are unfixable.

### Source index
Deterministic rendering: [CDP HeadlessExperimental](https://chromedevtools.github.io/devtools-protocol/tot/HeadlessExperimental/) | [CDP Emulation (virtual time)](https://chromedevtools.github.io/devtools-protocol/tot/Emulation/) | [headless-dev rendering control](https://groups.google.com/a/chromium.org/g/headless-dev/c/S5CoLs46AiE) | [headless-dev BFC examples](https://groups.google.com/a/chromium.org/g/headless-dev/c/WZtYOO-x1Hc) | [headless-dev 60fps WebGL](https://groups.google.com/a/chromium.org/g/headless-dev/c/s8ttGCh8jzM) | [headless-dev CSS animation capture](https://groups.google.com/a/chromium.org/g/headless-dev/c/QBQEm5Yd3_E) | [Chromium headless README](https://chromium.googlesource.com/chromium/src/+/lkgr/headless/README.md) | [Chrome headless guide](https://developer.chrome.com/docs/chromium/headless) | [HyperFrames writeup](https://www.heygen.com/research/html-to-video) | [HyperFrames beginFrame removal note](https://github.com/NousResearch/hermes-agent/blob/main/optional-skills/creative/hyperframes/references/troubleshooting.md) | [hyperframes #294](https://github.com/heygen-com/hyperframes/issues/294) | [Replit render engine](https://blog.replit.com/browsers-dont-want-to-be-cameras) | [WebVideoCreator](https://github.com/Vinlic/WebVideoCreator) | [puppeteer-capture](https://alexey-pelykh.com/blog/why-i-built-puppeteer-capture/) | [timecut](https://github.com/tungs/timecut) | [puppeteer #11315](https://github.com/puppeteer/puppeteer/issues/11315) | [puppeteer #3411](https://github.com/puppeteer/puppeteer/issues/3411).
Screencast/Playwright: [CDP Page](https://chromedevtools.github.io/devtools-protocol/tot/Page/) | [devtools-protocol #63](https://github.com/ChromeDevTools/devtools-protocol/issues/63) | [playwright #31424](https://github.com/microsoft/playwright/issues/31424) | [#12056](https://github.com/microsoft/playwright/issues/12056) | [#7246](https://github.com/microsoft/playwright/issues/7246) | [#34282](https://github.com/microsoft/playwright/issues/34282) | [#35776](https://github.com/microsoft/playwright/issues/35776) | [playwright-recorder-plus](https://github.com/MuTsunTsai/playwright-recorder-plus) | [Playwright clock](https://playwright.dev/docs/clock) | [Playwright browsers](https://playwright.dev/docs/browsers).
Screenshots: [puppeteer #1656](https://github.com/puppeteer/puppeteer/issues/1656) | [#736](https://github.com/puppeteer/puppeteer/issues/736) | [#476](https://github.com/puppeteer/puppeteer/issues/476) | [#3502 burst mode](https://github.com/puppeteer/puppeteer/issues/3502) | [screenshotone speed guide](https://screenshotone.com/blog/optimize-for-speed-when-rendering-screenshots-in-puppeteer-and-chrome-devtools-protocol/) | [Bannerbear tips](https://www.bannerbear.com/blog/ways-to-speed-up-puppeteer-screenshots/).
Real-time capture: [Arch framedrop](https://bbs.archlinux.org/viewtopic.php?id=205920) | [x11docker #199](https://github.com/mviereck/x11docker/issues/199) | [record-screen](https://github.com/jperl/record-screen) | [ScreenCaptureKit](https://developer.apple.com/documentation/screencapturekit) | [SCK vs AVFoundation](https://developer.apple.com/forums/thread/736022) | [Sequoia multi-display](https://developer.apple.com/forums/thread/769214) | [SCK throttling](https://developer.apple.com/forums/thread/811300) | [ffmpeg+SCK Rust writeup](https://www.ashleyarthur.co.uk/posts/2025/ffmpeg_rust_screencapture/).
In-page: [mediacapture-record #177](https://github.com/w3c/mediacapture-record/issues/177) | [Mozilla 1231848](https://bugzilla.mozilla.org/show_bug.cgi?id=1231848) | [Mozilla 1344524](https://bugzilla.mozilla.org/show_bug.cgi?id=1344524) | [Chrome bug 897727](https://paul.kinlan.me/chrome-bug-897727mediarecorder-using-canvas-capturestreamfails-for-large-canvas-elements-on-android/) | [puppeteer #4404](https://github.com/puppeteer/puppeteer/issues/4404) | [puppeteer-stream #27](https://github.com/samuelscheit/puppeteer-stream/issues/27) | [WebCodecs guide](https://developer.chrome.com/docs/web-platform/best-practices/webcodecs) | [WebCodecs canvas encode](https://sabigara.com/blog/capture-frames-and-encode).
WebGL/GPU: [SwiftShader doc](https://chromium.googlesource.com/chromium/src/+/main/docs/gpu/swiftshader.md) | [Intent to Remove SwiftShader fallback](https://groups.google.com/a/chromium.org/g/blink-dev/c/yhFguWS_3pM) | [chromedp #1073](https://github.com/chromedp/chromedp/issues/1073) | [GPU-in-headless issue](https://issues.chromium.org/issues/40540071) | [WebGPU/WebGL headless testing](https://developer.chrome.com/blog/supercharge-web-ai-testing).
Encoding: [ASWF H.264 guidelines](https://academysoftwarefoundation.github.io/EncodingGuidelines/Encodeh264.html) | [Mozilla 1368063 (yuv444)](https://bugzilla.mozilla.org/show_bug.cgi?id=1368063) | [motioneye #1067](https://github.com/ccrisan/motioneye/issues/1067) | [VideoHelp screen-recording settings](https://forum.videohelp.com/threads/398764-x264-x265-Most-efficient-settings-for-Screen-recording) | [MP4 browser support](https://www.lambdatest.com/web-technologies/mpeg4).
