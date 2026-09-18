# Replicating the Screen Studio look: technical decomposition and implementation guide

Scope: synthesize every Screen Studio (SS) polish effect in post from (a) a raw
cursor-free screen capture and (b) a machine-readable input-event timeline. Primary
hard-data sources: SS's own docs/changelog, SS's landing-page bundle (which ships the
literal spring constants of the in-app cursor engine), the Cap open-source recorder
(whose Rust renderer explicitly implements "Screen Studio semantics" and encodes
concrete constants), and the reverse-engineered SS project format in
crafter-station/open-screenstudio.

Key architectural fact (confirmed by SS's author and every clone): **nothing is baked
in at record time.** SS records raw screen pixels plus separate JSON channels of
cursor moves (~120 Hz, with `cursorId` and hotspot per macOS cursor type), clicks, and
keystrokes; the cursor is re-rendered as a sprite along a synthetic path, and a
"virtual camera" (zoom+pan) is computed in the editor. Our pipeline (capture without
cursor + scripted event timeline) is exactly this architecture, with the advantage of
perfect future knowledge.

---

## Part A — Effect inventory with concrete parameters

### A1. Cursor re-rendering and path smoothing

The single highest-leverage effect. The real cursor is hidden/ignored; a sprite is
animated along a **spring-mass-damper simulation chasing the raw cursor path**.

**The exact default spring, from SS's own landing-page code** (module `27647` in chunk
`9764-*.js`, which powers the "behaves exactly like Screen Studio defaults" demo the
author tweeted about):

| Preset (role) | stiffness k | damping c | mass m | ω₀=√(k/m) | ζ=c/2√(km) | steady-state lag c/k |
|---|---|---|---|---|---|---|
| default cursor movement | **470** | **70** | **3** | 12.5 rad/s | **0.93** | 149 ms |
| click / snappy spring | **530** | **40** | **1** | 23.0 | 0.87 | 75 ms |
| "slow" variant | 170 | 50 | 3 | 7.5 | 1.11 (overdamped) | 294 ms |
| camera/other | 340 | 60 | 3 | 10.7 | 0.94 | 176 ms |

Units: k in N/m-equivalents with position in px or normalized UV (the sim is
scale-free), c in N·s/m, dt in seconds. 470/70/3 is independently confirmed three
ways: SS landing bundle, the reverse-engineered SS project JSON (`"spring":
{"stiffness": 470, "damping": 70, "mass": 3}`), and Cap's default "Mellow" preset.
Cap's other user presets: Slow 200/40/2.25, Smooth 80/28/2.5, Fast 380/30/1, plus an
internal Drag profile 1000/40/1 used while the primary button is held.

**Integration:** semi-implicit Euler at fixed step is fine (`a=(-k·x_disp - c·v)/m;
v+=a·dt; x+=v·dt`), 60 Hz steps; Cap uses the closed-form damped-harmonic solution
(handles under/critical/overdamped branches) at a fixed 16.67 ms step, then lerps
between precomputed samples at render time. Precompute the whole timeline once; it
makes playback/seek/export bit-identical.

**Cap's production-grade refinements (all directly reusable):**
- **Click anticipation:** when the next click is ≤ **500 ms** away, retarget the
  spring at the click coordinates immediately (glide arrives early); **175 ms** before
  the click, switch to the stiffer click spring (530/40/1). After the click, revert.
  With a scripted timeline this comes for free.
- **Phase-lead compensation:** a spring chasing a moving target trails it by **c/k
  seconds** (149 ms at defaults). Sample the raw path `c/k` ahead of render time as
  the spring target so the smoothed cursor sits where the real cursor is "now".
  Smooth the lead when the profile changes (one-pole, ~130 ms time constant,
  coefficient 0.12 per 60 Hz step).
- **Shake filter:** drop a sample if it reverses direction (dot product < 0 with
  neighbors) and both displacements are < **0.015 UV** within a **100 ms** window.
- **Decimation:** thin raw moves to ≤60 Hz and drop moves < 1/1920 UV apart.
- **Teleport handling:** if a jump between samples exceeds ~**500 px**, snap the
  spring instead of smoothing (open-screenstudio threshold).
- **Idle gaps:** treat gaps > ~67 ms in the move stream as holds (no interpolation
  across them).
- **Settle tail:** run the sim **300 ms** past the last move so it comes to rest.

**For fully synthetic movement** (an input timeline with only endpoints, e.g. scripted
`moveTo`): generate the inter-point path with a **minimum-jerk trajectory** (Flash &
Hogan 1985): `x(τ)=x₀+(x₁−x₀)(10τ³−15τ⁴+6τ⁵)`, τ=t/T. Duration from a Fitts-like
rule: `T = 0.25 s + 0.35 s · (d/D)^0.5` (d = travel distance, D = screen diagonal),
clamped to [0.3 s, 1.1 s]; add a slight curvature (offset the midpoint control
perpendicular by 2–5% of distance) to avoid dead-straight robotic lines, then
optionally run the spring over it anyway — the spring is what gives the SS "weight".

### A2. Cursor presentation

- **Size:** SS scales the cursor up; observed default in the reverse-engineered
  project is **1.5×** natural size; UI allows up to roughly 2–3×. Cap stores size as
  percent, default 100, and normalizes cursor sprite height against a standard cursor
  height, scaled by 1/zoom-crop so cursor size stays constant on screen while zoomed
  (SS behaves the same).
- **Render as vector/hi-res sprite** (SS ships its own cursor sets; Cap rasterizes SVG
  at 200 px height). Respect the per-cursor **hotspot**. Option "always use default
  arrow" avoids I-beam flicker; SS also has "optimize original cursor types" to
  suppress rapid cursor-type changes.
- **Click press animation:** cursor scales to **0.8×** on mouse-down, back on release;
  Cap: `CLICK_SHRINK_SIZE 0.8` over **130 ms** (`CURSOR_CLICK_DURATION 0.13 s`). The
  SS landing demo interpolates scale over [0.8, 1.0] with the click spring — same
  numbers.
- **Rotate while moving:** small heading-following rotation, default amount **0.15**
  (Cap `rotation_amount`), i.e. cursor tilts ~a few degrees toward its velocity
  vector; SS shipped this in 2.7.3.
- **Hide when idle:** fade out after **2 s** idle (default delay), fade duration **400
  ms**, fade back in with **250 ms** lookahead before motion resumes; minimum delay
  floor 500 ms (Cap constants).
- **Loop cursor position:** optional; near the video end, animate the cursor back to
  its initial position for perfect loops (SS docs).
- **Stop cursor movement at end:** freeze cursor for the last N seconds.

### A3. Click highlight effects

SS options: **none / circle / ripple / shockwave** (changelog 2.25.25, 2.26.0), plus
optional **mouse click sound**.

Ripple spec (inferred from SS output + standard practice; SS publishes no numbers):
- Spawn a circle at click point, below the cursor sprite.
- Expand radius from ~0 to **40–70 px** (at 1× zoom, scale with cursor size), over
  **400–600 ms**, ease-out (cubic or `1−(1−t)³`).
- Fill: white or accent color at **25–35% start opacity → 0**; or ring stroke 2–3 px
  that fades without much expansion ("circle" variant = ring at full size fading ~300
  ms).
- "Shockwave" = same but larger radius (~1.5–2×) with a thin ring and slight
  background displacement; treat as ripple with stroke-only and higher initial scale
  velocity.
- Optional press-state darkening under the cursor while button is down.

### A4. Auto zoom (the virtual camera) — when and where

SS creates zoom segments automatically from clicks ("Create initial zooms
automatically", per-click "Auto zoom (zoom to your clicks)"); typing bursts also
attract focus in newer versions. Manual zooms set an explicit target point; "Instant
zoom" cuts with no animation.

**Cap's generation algorithm (clean-room match of SS behavior), all constants
explicit:**
- Default zoom amount: **2.0×** (`DEFAULT_AUTO_ZOOM_AMOUNT`).
- Per click: segment = [click − **300 ms**, click + **2500 ms**].
- Merge overlapping/nearby segments when gap ≤ **2500 ms**.
- Ignore clicks in the last **1000 ms** of the video; clamp segment ends to duration −
  **800 ms**.
- Segment fields: amount, mode (auto = aim at cursor/clicks, manual = fixed x,y),
  glide direction/speed (default 0.5), `instant` flag, `edgeSnapRatio` **0.25**.

**SS observed behavior/limits:** minimum zoom segment length **0.1 s** (was 1 s before
v3.5.0); zoom level adjustable with 0–9 keys while hovering a segment; manual zoom
picker snaps at 50% steps; zoom types auto / manual / follow-cursor (`"type":
"follow-cursor"`, `"snapToEdges": 0.25` in the project format).

**Zoom factor guidance** (community + SS practice): default **2.0×**; use
**1.25–1.5×** for already-readable UI, up to **2.5–3×** for dense detail, but 3× shows
pixels unless source is 4K/Retina 2× — always record at ≥2× the pixel density of the
final crop. FollowCursor exposes 1.25–2.5×.

### A5. Zoom/pan transition dynamics — how the camera moves

SS's camera is **spring-driven, not duration-eased** ("Speed controls from slow to
rapid, with advanced tension, friction, and mass parameters" — SS UI; two
screen-animation modes: "Focused" = stabilizes quickly for readability, "Smooth" =
more fluid). Cap's current engine models exactly this:

- Three independently sprung channels: **amount** (zoom scale), **center** (2D framing
  target), **activity** (0/1 any-zoom-active flag).
- Screen movement spring default: **stiffness 200, damping 40, mass 2.25** (ζ≈0.94,
  visually ~0.8–1.0 s to settle; SS's equivalent is plausibly its 340/60/3 spring,
  ζ≈0.94, ~0.7 s). Both are *just barely underdamped*: a 2–3% overshoot that reads as
  "cinematic" without bounce.
- Targets are **step functions**; retargeting happens every 8 ms sim step (125 Hz
  precompute) and velocity always carries across retargets — segment start/end/re-aim
  are just new targets, so motion is continuous with no boundary special cases.
- **Pre-aiming:** while amount ≈ 1 (≤1.0005), the center channel snaps instantly to
  its upcoming target, so a zoom-in scales *straight toward* its focus rather than
  zooming center-then-panning.
- **Instant zooms** snap all channels within a ±0.1 s window of the segment (hard cut;
  also suppress motion blur across it).
- Geometry: viewport = `from_amount_center(amount, center)` where center ∈ [0,1]² is a
  *proportional travel-space* coordinate (0 = viewport flush to left/top edge, 1 =
  flush right/bottom); clamping is inherent, the crop can never leave the frame.
  `edge_snap_ratio 0.25` remaps focus so anything in the outer 25% band pins flush to
  the edge.
- If using fixed-duration easing instead of springs: community consensus for the SS
  look is **ease-in-out (or ease-out for zoom-out), 350–600 ms** for the scale
  transition; a 200 ms snap reads jarring. Older Cap used a fixed `ZOOM_DURATION` ≈
  0.6 s. But springs + velocity carry-over handle overlapping/retargeted zooms far
  more gracefully.

### A6. Camera pan while zoomed (follow cursor / glide)

- **Dead-zone cluster follow (Cap, matches SS "follow cursor")**: while a zoom segment
  is active, build greedy clusters of cursor positions with a bounding box limited to
  **50% width × 70% height of the visible zoomed viewport** (i.e. 0.5/amount ×
  0.7/amount in content UV). The camera aims at the active cluster's center; when the
  cursor exits the box, a new cluster starts and the center spring re-aims. Result:
  the camera holds still during local mousing and glides only on real relocations —
  the core anti-nausea mechanism.
- **Glide (SS)**: optional slow drift while zoomed (direction + speed, default speed
  0.5) to keep long holds alive; a few px/s of travel, purely aesthetic.
- **Fallback focus**: center (0.5, 0.5) when no cursor data exists in the segment.

### A7. Motion blur

SS: "cinematic motion blur", enabled by default, one slider plus advanced per-channel
amounts for **cursor movement / screen zoom / screen panning** (SS animations guide;
new engine in 2.25.18). Cap replicates it with documented "Screen Studio semantics":

- **Semantics:** the user amount scales the **length of the blur kernel** linearly;
  the output is the fully blurred result, never a crossfade with a sharp copy (that
  reads as ghosting). Zero-length kernel = identity, so blur fades in/out with
  velocity continuously — no activation ramp needed.
- **Pan/movement blur (screen content):** directional blur along the per-frame
  viewport velocity vector. Kernel: **21 taps over [0, +v]** where `v = (bounds delta
  this frame) × amount`, alpha accumulated with the same kernel so card edges smear
  too. Default amount **1.0** (i.e. smear = one full frame of travel ≈ 360° shutter;
  set 0.5 for a film-standard 180° shutter look). Normalize by fps: strength ×
  (fps/60), clamp ≤ 4. Activation threshold: ≥1 px/frame movement.
- **Zoom (radial) blur:** ray toward the zoom center, per-pixel length =
  `dist_to_center × min(zoom_rate, 1)`, hard cap **0.10 UV**; **13 taps** with
  parabolic weights `4(p−p²)` peaked mid-ray and per-pixel interleaved-gradient-noise
  dither to kill banding. Typical real ray lengths 0.03–0.05 UV at far corners during
  a spring zoom.
- **Cursor smear:** the cursor sprite is blurred along its own per-frame travel: smear
  length = per-frame travel px × amount (linear, no curve), capped at **480 px** to
  bound teleports; fps-normalized as above.
- **Suppression:** never blur across a hard cut / instant zoom; zoom blur beats pan
  blur when zoom dominates (dominance check).
- **Alternative (highest quality, offline):** frame accumulation — render N subframes
  (N = 8–32) of the *virtual camera transform* across a shutter interval of `0.5/fps`
  (180° shutter convention) and average in **linear light**. Since the source is a
  discrete screen capture, subframe content is the nearest captured frame but the
  camera/cursor transforms interpolate continuously — accumulation of the transform is
  indistinguishable from true motion blur. The single-pass directional/radial shader
  above is the real-time approximation of the same integral.

### A8. Keystroke / shortcut overlay

- SS shows pressed shortcuts as a pill overlay (bottom-center), with a **size
  slider**, a dedicated shortcuts timeline (per-key disable), and a **Show single key
  shortcuts** toggle.
- Grouping logic: "tries to ignore keystrokes near to each other" — i.e. suppress
  overlays during typing bursts; show only modifier combos (⌘K etc.) by default;
  single keys only if enabled and temporally isolated (inferred gap threshold ~300–500
  ms).
- Render: rounded-rect pill, key glyphs in macOS style, fade/scale in ~150 ms, hold
  ~800–1200 ms after last key of the combo, fade out ~200–300 ms; successive shortcuts
  within the hold window replace/queue the pill's content.
- Related: SS "speed up typing segments" — detects typing stretches and offers to
  time-compress them (a pacing tool worth copying: compress inter-keystroke video to
  e.g. 3–5× while holding the zoom).

### A9. Background, padding, corners, shadow, inset

- **Background sources:** wallpaper (macOS-native set; default has been "Tahoe Light"
  since 3.0), generated gradient, flat color, custom image.
- **Padding:** slider; 0 = no background. Typical SS-look values: **6–12% of output
  min dimension** (SS marketing shots sit around 8–10%). Reverse-engineered schema
  stores per-side px.
- **Corner radius** on the recording card: SS "Rounded corners" slider; typical
  premium look **12–24 px at 1080p output (~1–2% of width)**; Cap supports round vs
  squircle (superellipse power 4) corner styles — SS corners read as squircle-ish at
  larger radii.
- **Shadow:** SS has presets flat→dramatic plus advanced size/opacity/blur. Concrete
  defaults from the two clones that copied SS's look:
  - Cap display shadow: master strength **73.6/100**, advanced `{size 33.9, opacity
    44.2, blur 10.5}` (percent-of-min-frame-dimension units in shader: reach =
    strength × blur% × minDim); generic default `{size 14.4, opacity 68.1, blur
    3.8}`.
  - open-screenstudio (SS project mirror): `{intensity 0.75, angle 90 (straight
    down), distance 25, blur 20}`.
  - Practical CSS-equivalent: `0 25px 50px rgba(0,0,0,0.35)` plus a tighter `0 5px
    15px rgba(0,0,0,0.20)`.
- **Inset:** an extra colored frame between video and background (SS 2.8.0) — a second
  padding layer with its own fill; typical 1–3%.
- **Border:** optional 1px light border (Cap default 80% opacity white-ish) helps the
  card read on dark backgrounds.

### A10. Export / delivery specs

- SS export: **MP4 (H.264) or GIF** (GIF advised < 1 min); quality presets **Studio /
  Social media / Web / Web low**; **60 fps default** ("all exports default to 60fps");
  fps choices 24/30/60; resolutions up to 4K (4K ≈ 4× HD export time). SS records the
  screen at up to 60 fps and cursor input at ~120 Hz.
- Recommended targets for the premium look: **1080p or 1440p at 60 fps, H.264 high
  profile, CRF 18–20 (or 12–16 Mbps VBR); 4K60 at CRF 18 ≈ 40–60 Mbps** for crisp
  text; H.265 at roughly half the bitrate where supported. 60 fps is load-bearing: the
  smooth cursor/camera glides are what sell the effect, and at 30 fps spring motion
  visibly stutters. For docs embeds where file size dominates, 30 fps + shutter-0.5
  motion blur is an acceptable fallback (blur hides the lower temporal rate).
- Render pipeline: composite in linear or at least premultiplied sRGB at the *output*
  resolution with the source at native (2×) density, so zooms ≤2× never upsample.

### A11. Pacing rules (anti-nausea / readability)

Compiled from SS behavior, its docs, and practitioner guidance:
- Zoom hold: **≥ 600 ms always** (below reads as a glitch); 800 ms–1.2 s for quick
  feedback actions; **1.5–2 s** for text-heavy targets. SS default post-click hold 2.5
  s (Cap constant) with merge, min segment 0.1 s available for intentional quick hits.
- Frequency: at most ~1 camera move per **3–4 s**; merge zooms rather than pumping
  in/out (both SS and Cap merge when gaps ≤ 2.5 s).
- Never zoom during fast cursor transit; zoom where the cursor *settles*, i.e. on
  clicks/typing/dwell.
- Zoom-out to 1× when: no interaction for > ~3 s, a scroll begins, a window/context
  switch occurs, or the next focus point is far (> ~60% of frame) — pan only within a
  zoom for nearby targets, cut or zoom-out-then-in for far ones.
- Camera velocity: spring params above inherently bound velocity; with easing, keep
  peak pan speed under ~1.5 frame-widths/s and scale rate under ~1.5×/s.
- Slight underdamping (ζ 0.87–0.95) everywhere; anything bouncier reads as toy-like,
  anything overdamped reads sluggish.

---

## Part B — Implementation notes for a look-ahead compositor

We have what SS never has: the full event timeline before rendering. Plan globally,
then simulate.

**Pass 1 — segmentation (plan the shot list).**
1. Extract interaction beats from the event timeline: clicks, typing bursts (group
   keydowns with gaps < ~1 s), scroll runs, drags, dwells.
2. Emit zoom segments per the Cap recipe (A4): [beat−0.3 s, beat+2.5 s] at 2.0×, merge
   ≤2.5 s gaps, clamp ends. Assign each segment a focus track, not a point: the
   sequence of beat coordinates inside it.
3. Classify each segment's zoom level by target size with UI metadata (small control →
   2–2.5×, form region → 1.5×, full-window context → 1×); otherwise keep 2×.
4. Insert explicit zoom-outs (amount 1, centered) between segments; mark far-jump
   transitions (> ~0.6 frame) as either instant cuts or zoom-out-through moves.
5. Snap focus centers through the edge-band remap (outer 25% → flush).

**Pass 2 — camera simulation.**
Run three spring channels (amount, center.x/y, activity) at 8 ms steps over the whole
timeline with step-function targets from Pass 1, velocity carried across retargets,
center pre-aim while amount≈1, dead-zone cluster re-aiming (0.5/amount × 0.7/amount
box) inside segments. Cache all samples; render is index+lerp.

**Pass 3 — cursor simulation.**
Since input is scripted there may be only sparse waypoints: synthesize inter-waypoint
paths (minimum-jerk, A1), then run the 470/70/3 spring with the 530/40/1
click-anticipation profile (500 ms target snap, 175 ms stiffen) and c/k phase-lead
compensation. Precompute at 60 Hz + settle tail. Derive per-frame cursor velocity from
adjacent samples for the smear.

**Pass 4 — render per output frame.**
1. Sample camera → crop rect; sample cursor → position/velocity/cursor-type.
2. Draw background (wallpaper/gradient) at output res.
3. Draw the screen card: crop source (bicubic/Lanczos), rounded corners (SDF or mask,
   squircle power 4), shadow (two-layer gaussian per A9), optional border/inset.
4. Apply motion blur on the card: directional 21-tap along per-frame crop velocity ×
   amount, or radial 13-tap toward zoom center while amount is changing; skip across
   cuts. (Or the accumulation variant: evaluate steps 1–3 at N=16 subframe times
   spanning 0.5/fps and average in linear.)
5. Composite click ripples (world-space, i.e. attached to screen coordinates so they
   zoom with content).
6. Draw cursor sprite: hotspot-anchored, size 1.5×/zoom-compensated, click-shrink 0.8
   (130 ms), heading rotation 0.15, idle fade, velocity smear (≤480 px).
7. Draw keystroke pill in output space (not zoomed).
8. Encode H.264 60 fps.

**Ordering constraints:** ripple under cursor; keystroke overlay over everything; blur
applies to card content + edges but not to overlay chrome; cursor is blurred by its
own velocity only (it lives above the card).

**Determinism:** all three sims are pure functions of (timeline, config); precompute
once, render frames in parallel.

---

## Part C — Sources

**Screen Studio (first-party)**
- Docs: [cursor settings](https://screen.studio/guide/cursor), [adding/editing zooms](https://screen.studio/guide/adding-editing-zooms), [auto zoom](https://screen.studio/guide/auto-zoom), [manual zoom](https://screen.studio/guide/manual-zoom), [animations (motion blur, speed presets, Focused/Smooth)](https://screen.studio/guide/animations), [background](https://screen.studio/guide/background), [shortcut overlay](https://screen.studio/guide/shortcuts), [export settings](https://screen.studio/guide/explanation-of-export-settings), [exporting](https://screen.studio/guide/exporting-the-video), [guide index](https://screen.studio/guide)
- [Changelog](https://screen.studio/changelog) (min zoom 1 s→0.1 s; motion-blur engine 2.25.18; ripple/shockwave 2.26.0; circle 2.25.25; cursor rotation 2.7.3; inset 2.8.0; shortcut timeline 2.22.0)
- Landing bundle spring constants: `https://screen.studio/_next/static/chunks/9764-19903045e53be0b5.js` (module 27647: 170/50/3, **470/70/3**, **530/40/1**, 340/60/3; click scale interp [0.8, 1])
- Feature hub: [custom auto-zoom range request](https://hub.screen.studio/p/custom-settings-for-default-automatic-zoom-range), [manual zoom by shortcut](https://hub.screen.studio/p/manual-zoom-by-shortcut)

**Author (Adam Pietrasiak)**
- [Show HN: Screen Studio](https://news.ycombinator.com/item?id=34045110)
- [HN comment: built in ~4 months; inspired by Stripe promo videos](https://news.ycombinator.com/item?id=33945843); [Electron/auto-update thread](https://news.ycombinator.com/item?id=35873727)
- [Tweet: Electron app with web tech](https://x.com/pie6k/status/1624535267401924611); [tweet: landing-page smooth-cursor demo behaves exactly like SS defaults](https://x.com/pie6k/status/1676337118488977408)
- [pie6k/motionblur](https://github.com/pie6k/motionblur)

**Open-source implementations (concrete constants)**
- Cap — [spring_mass_damper.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/spring_mass_damper.rs), [cursor_interpolation.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/cursor_interpolation.rs) (500 ms click lookahead, 175 ms stiffen, c/k lead, shake filter), [zoom_spring.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/zoom_spring.rs) (8 ms steps, 0.5×0.7 dead-zone clusters, pre-aim), [zoom.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/zoom.rs), [configuration.rs](https://github.com/CapSoftware/Cap/blob/main/crates/project/src/configuration.rs) (presets, screen spring 200/40/2.25, shadows, camera 0.7 scale-during-zoom), [recording.rs auto-zoom generation](https://github.com/CapSoftware/Cap/blob/main/apps/desktop/src-tauri/src/recording.rs) (300/2500/2500/1000/800 ms, 2.0×), [layers/cursor.rs](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/layers/cursor.rs) (click 130 ms/0.8×, idle fade, 480 px smear cap, fps normalization), [composite-video-frame.wgsl](https://github.com/CapSoftware/Cap/blob/main/crates/rendering/src/shaders/composite-video-frame.wgsl) (21-tap directional, 13-tap radial + parabolic weights + dither, 0.10 UV cap)
- [crafter-station/open-screenstudio](https://github.com/crafter-station/open-screenstudio) — [TECHNICAL_PLAN.md](https://github.com/crafter-station/open-screenstudio/blob/main/docs/TECHNICAL_PLAN.md) (SS-mirror project format: 120 Hz input, spring 470/70/3, cursor size 1.5, shadow 0.75/90°/25/20, zoom `follow-cursor` + `snapToEdges 0.25`), [phase-5 cursor smoothing plan](https://github.com/crafter-station/open-screenstudio/blob/main/.opencode/plans/phase-5-cursor-smoothing.md) (spring integration code, 500 px teleport)
- [jkuri/Reframed](https://github.com/jkuri/reframed) (spring presets, click-cluster/dwell auto-zoom, Hermite keyframe easing, 120 Hz cursor channel, ProRes export)

**Pacing / parameter guidance and clones**
- [Screenify: auto-zoom guide](https://www.screenify.studio/blog/2026-04-10-auto-zoom-screen-recording) (2× sweet spot, 600 ms floor, 800 ms–2 s holds, 350–500 ms ease-in-out, 1 zoom per 3–4 s), [smooth cursor guide](https://www.screenify.studio/blog/2026-04-20-smooth-cursor-recording) (50–150 ms perceived lag budget), [click highlight guide](https://www.screenify.studio/blog/2026-04-19-highlight-mouse-clicks-recording)
- [Building FollowCursor](https://sabbour.me/2026/03/23/building-followcursor.html) (settlement/typing/cluster triggers, 1.25–2.5×, ≤4-cluster pan chains)
- [Dave Swift SS review](https://daveswift.com/screen-studio/) (tension/friction/mass UI, per-channel motion blur defaults on, 60 fps default, presets Studio/Social/Web/Web-low, ~3× export time)
- [Rekort zoom-effect writeup](https://rekort.app/blog/screen-recording-with-zoom-effect); [Scribe SS review](https://scribehow.com/page/Screen_Studio_Review_2026_I_Tested_the_Auto-Zoom_Mac_Recorder_for_90_Days__Heres_the_Truth__0R7wu5TiSvqYAK3TzdygdQ) ("zooms on every click… drunk cameraman" failure mode to avoid); [HN clone threads](https://news.ycombinator.com/item?id=41827955), [ScreenKite](https://news.ycombinator.com/item?id=47016221)
- Bitrate refs: [UniFab 4K bitrate guide](https://unifab.ai/resource/4k-bitrate), [Canvid 4K60 export](https://www.canvid.com/features/export-4k-60fps)

**Math references**
- Minimum-jerk: Flash & Hogan, *The coordination of arm movements: an experimentally
  confirmed mathematical model*, J. Neurosci. 5(7), 1985
- Motion blur: [180° shutter convention](https://en.wikipedia.org/wiki/180-degree_shutter), [per-object motion blur (Chapman)](http://john-chapman-graphics.blogspot.com/2013/01/per-object-motion-blur.html), McGuire et al., *A Reconstruction Filter for Plausible Motion Blur*, I3D 2012
- Spring semantics identical to react-spring's `{tension/stiffness, friction/damping,
  mass}` physical model (ω₀=√(k/m), ζ=c/2√(km))

**Bottom line for the build:** cursor spring 470/70/3 with click spring 530/40/1 and
500 ms anticipation; camera springs ~200/40/2.25 (ζ≈0.94) over step targets at 2.0×
default zoom generated as click±(0.3 s pre / 2.5 s post) merged at 2.5 s; dead-zone
follow at 50%×70% of viewport; 21-tap directional + 13-tap radial blur with amount =
smear-per-frame-travel; cursor 1.5×, click shrink 0.8/130 ms; padding ~8–10%, radius
~16 px@1080p, soft down-shadow ~25 px offset/20 px blur/0.35 opacity; H.264 60 fps.
Those numbers reproduce the SS look to within tuning distance.
