# 71 Studio - Feature films

- **Author**: Emiliano Bonilla
- **Date**: 2026-09-27
- **Related**:
  [PR #2777 - SY-4551: Add the docs video studio](https://github.com/synnaxlabs/synnax/pull/2777)

## 0 Summary

The studio (`studio/`) turns scripted Console sessions into docs videos. This RFC adds
feature films: short, cinematic clips that show one feature, in the style of Linear's
launch films. A film is made only from real captured pixels. The compositor stages the
capture on a near-black set: a tilted 3D plane, depth of field, a spotlight on the
feature, text cards, film grain, and hard cuts. Films share one house style built from
ten shot types with tunable settings. A film file picks the shots, their order, and
their settings, so an agent can write and render a film without new design work. All
films are shot in one demo world, a rocket engine test stand, and live plots are frame
perfect because Pluto gains a clock the studio can step.

## 1 Motivation

- **The studio has no film layer**: the one composition
  (`studio/src/remotion/Root.tsx:30`) sizes its canvas from the capture and draws the
  capture, a cursor, and click ripples on black
  (`studio/src/remotion/StudioVideo.tsx:57`). It has no background, text, cut, audio, or
  frame shape other than the capture's own.
- **Fixtures look like demos**: telemetry comes from `sineTelemetry`
  (`studio/src/fixtures/telemetry.ts:32`). An engineer reads a sine wave as a toy, no
  matter how the shot is lit.
- **Live plots run on the wrong clock**: capture steps a virtual clock one frame at a
  time, but the Aether worker keeps wall time. A live plot in a finished video scrolls
  too fast and jumps (`studio/README.md:151`).
- **Per-film design does not scale**: a film a few times a week is only possible when
  the craft lives in reusable shots, not in each film.

## 2 Vocabulary

- **Film**: One feature clip, 20 to 35 s, 4:5, dark theme. Defined by one film file.
- **Film file**: A module under `studio/films/` that exports the capture script and the
  edit.
- **Capture**: The recorded Console session a film cuts from. It is an ordinary studio
  capture with marks.
- **Mark**: A named point in a capture, with an optional screen rect, recorded by
  `session.mark(name, target?)`.
- **Edit**: The ordered list of shots in a film.
- **Shot**: One piece of the edit. A capture shot plays a span of the capture between
  two marks. A card shot draws typography with no capture.
- **Stage**: The set every shot sits on: the black ground, the vignette, the rim light,
  and the grain.
- **World**: The fixture set that fills the Core before a capture: channels, data,
  schematics, and ranges.

## 3 Principles

1. **Real pixels only**: every image of the Console in a film comes from a capture. The
   UI is never redrawn, recreated, or mocked. Typography on cards and labels is not UI
   and is allowed.
2. **One house style**: films differ only in which shots they use, in what order, and
   with what settings. A look that no shot type can make becomes a new shot type, never
   a one-off in a film file.
3. **Policy lives in the director**: the director plans every camera, focus, and light
   value. The compositor applies those values mechanically. This conforms to the rule in
   `docs/tech/video-automation/decisions.md` and extends it to the stage.
4. **Silent first**: a film tells its whole story with the sound off. Music is a bed,
   never a carrier.
5. **Deterministic**: the same film file renders the same video, frame for frame.
6. **Fail loud**: a shot that asks for more than the capture can give, such as a zoom
   past the sharpness budget or a mark that was never recorded, is an error at plan
   time. A film never ships a soft or empty frame in silence.

## 4 Design

### 4.0 The pipeline

A film runs the existing pipeline with one new stage between capture and render:

```
film file (studio/films/<id>.ts)
  -> capture  the existing rig, plus marks          -> frames/, timeline.json
  -> plan     director: edit + timeline -> stage tracks per output frame
  -> render   Remotion "film" composition, 1080x1350, 60 fps -> out/films/<id>.mp4
```

`timeline/` and `director/` stay pure. The film schema is a new pure module, `film/`.
The capture layer gains marks. The Remotion layer gains a second composition.

### 4.1 Marks

`CaptureSession.mark(name, target?)` records a `mark` event at the current frame.
`target` is a locator or a rect. A locator stores its bounding rect, measured the same
way `zoom` measures it today (`studio/src/capture/rig.ts:507`). Mark names are unique
within a capture; a duplicate throws. The timeline schema gains `markEventZ` next to
`zoomOverrideEventZ` (`studio/src/timeline/timeline.ts:79`).

### 4.2 The film file

```ts
import { capture, film } from "@/index";

export const viewport = { width: 1440, height: 900 };

export default async (session: capture.CaptureSession) => {
  await capture.login(session);
  session.startRecording();
  const group = session.page.getByText("Oxidizer feed");
  await session.mark("open", session.page.locator(".console-mosaic"));
  await session.click(group);
  await session.mark("group", group);
  await session.hold(1500);
  await session.mark("end");
};

export const edit = film.edit([
  { type: "card", lines: ["Group a feed system.", "Move it as one."] },
  {
    type: "take",
    from: "open",
    to: "end",
    beats: [{ at: "open", wide: true }, { at: "group", fill: 0.7 }],
  },
  { type: "spotlight", from: "group", to: "end", target: "group" },
  { type: "end", tagline: "Schematics that scale with your stand." },
]);
```

`film.edit` validates the list against a Zod schema at load time. Every shot takes only
the settings its type defines. Each setting is optional, and the director fills the
house default, so most shots need only their marks.

### 4.3 Shots and the camera

| Shot        | What it shows                                             | Key settings, house defaults                            |
| ----------- | --------------------------------------------------------- | ------------------------------------------------------- |
| `take`      | The window on a tilted plane, a camera moving on beats    | rest tilt X 4°, Z -2°; perspective 1800 px              |
| `isolate`   | One panel cropped out, the rest falls to black            | panel fills 68% of frame width                          |
| `spotlight` | The target sharp and lit, everything else blurred and dim | blur 16 px, brightness 0.5, fade 350 ms                 |
| `focus`     | A depth-of-field band that pulls between two marks        | band 30% of frame height, blur 8 px                     |
| `card`      | One or two lines of text on the stage                     | Inter 68 px, rise 10 px over 550 ms                     |
| `label`     | A monospace uppercase tag pinned beside a mark            | tracking 0.2 em, 60% white                              |
| `stat`      | One large number with a short line                        | number at 160 px, line at 40 px                         |
| `end`       | Grainy Synnax-blue gradient, white wordmark, tagline      | 3.5 s                                                   |

A take is one continuous camera move, and its beats say what the camera looks at. Each
beat names a mark. At that mark's frame the camera moves to frame the mark's rect, or
the whole window. The film script sets marks where the content changes: the click on a
button, a value that starts to climb, the log line of a stage change. The camera then
moves because something happened, not on a schedule. The motion follows four rules:

- **Springs, not curves**: position and zoom follow critically damped springs. A move
  starts gently, settles long, and never overshoots. A beat that lands mid-move
  retargets with the move's momentum.
- **Pace follows distance**: a beat's move settles in 0.8 to 1.9 s. The time grows with
  the distance traveled and the zoom change, so small reframes are quick and long pans
  take time.
- **The plane leans into travel**: tilt follows camera velocity, as a camera on a jib
  banks into a pan, and returns to rest when the camera holds. Only a reveal opens at
  a strong tilt (X 22°, Z -10°) and settles over 1.8 s.
- **Holds breathe**: while it holds on a beat, the camera pushes in at a per-beat rate
  (default 1% per second), so a still frame is never frozen.

The frame keeps one accent color, and shots join with hard cuts, never a wipe or a
dissolve.

### 4.4 The stage

Every frame draws, from back to front: a ground of `#060607`, the shot, a radial
vignette from 35% to 85% of the frame radius, a rim light (a 1 px top-left edge gradient
on the plane plus a 6% radial glow in `screen` blend), and grain. The grain is SVG
`feTurbulence` at base frequency 0.85 and 4% opacity in `overlay` blend. Its seed is the
frame index, which keeps renders deterministic and breaks up the banding H.264 adds to
dark gradients. Captures run in the dark theme only.

### 4.5 Sharpness budget

A shot stays sharp while its zoom `z` meets
`z <= (source px across the view) / (output px across the view)`, the rule in
`decisions.md` applied to the shot's crop. Film captures default to `dsf` 3, so a 1440
CSS px capture carries 4320 source px. The director checks the zoom of every frame of a take and throws
when one exceeds the budget, with the shot, the beat, and the `dsf` that would fix it.

### 4.6 The demo world

`fixtures.testStand` (`studio/src/fixtures/world.ts`) builds one bipropellant test stand
on the capture's Core:

- **Project**: the internal TPC operator workspace, committed to `studio/worlds/tpc/`
  with channel names in place of keys. The fixture binds the names to the new keys and
  imports the folder through `client.projects.import`, so the Core rebuilds the
  schematic, the thermocouple table, the pressure plots, the sequence log, and the panel
  layout.
- **Channels**: tank, pressurant, and pneumatic pressures, feed and pressurant
  temperatures, a command and state pair for each of 14 valves, the Start Sequence
  command, and the sequence log.
- **Simulation**: valves follow their commands. Start Sequence runs a pressurize, flow,
  and safe sequence in which the TPC valves regulate tank pressure around a setpoint.
  Tank pressures follow the valves, with sensor noise. The stand writes 60 s of idle
  history first, so plots open full.

Every film file starts from `fixtures.testStand`. Docs scripts keep their own fixtures.

### 4.7 Live data on the virtual clock

A live plot must advance one frame of data per captured frame. A capture runs many times
slower than real time, so data on wall time scrolls that many times too fast. Two pieces
fix this, both in the rig:

- **The worker clock**: `page.clock` fakes time in the page but never reaches its
  workers, and the Aether worker windows live series with `TimeStamp.now()`, which reads
  `new Date()`. The rig installs a script in each worker that replaces `Date` and
  `performance.now` with a clock it sets on every tick. The page and its workers then
  read one virtual time.
- **Virtual sampling**: fixtures take the session's clock (`session.now()`) and write
  samples at a fixed rate in virtual time, so each captured frame holds the same amount
  of data.

Pluto and the Console do not change. This matches how the rig already handles the page:
it fakes the platform clock instead of adding a clock seam to the product.

## 5 Implementation phases

Each phase is one PR into `main`.

- **Phase 1: Film layer and proof film.** Marks, the `film/` schema, the director's
  stage plan, the `film` composition with the stage, the `wide`, `card`, and `end`
  shots, and `pnpm film`. One proof film from an existing docs capture. Specs cover mark
  recording and the plan of each shot.
- **Phase 2: Studio docs cleanup.** `studio/README.md` still describes click auto-zoom,
  a 1512x945 default, and a 1.5 s idle fade. Remove the dead auto-zoom constants and the
  auto-zoom dwell in `rig.ts`. Mechanical, `review/bot`.
- **Phase 3: The remaining shots.** `isolate`, `spotlight`, `focus`, `macro`, `follow`,
  `label`, and `stat`, with the sharpness budget of §4.5.
- **Phase 4: The demo world and live data.** The test stand of §4.6, the worker clock of
  §4.7, and the hotfire film.
- **Phase 5: Music and stills.** The music layer and `--still`.

## 6 Resolved decisions

**6.0 Real captures over redrawn UI.** Redrawing the Console in a motion tool can make
looks a capture cannot: a UI slab with thickness, macro shots past the capture's detail,
and abstract concept animations. It costs a second copy of the UI kept in sync by hand,
and it shows engineers a rendering instead of the product. Linear's own Diffs film shows
the look comes from light, camera, and focus, which apply to captures. The trade is
real: those three looks are out.

**6.1 Motivated camera moves over fixed shot recipes.** The first cut gave every
capture shot the same motion: open tilted, ease to rest, dolly 1.00 to 1.06. Every
shot then moved the same way whatever it showed, and the film read as generated. Beats
tie each move to a change in the content. The trade is real: each film script now
places marks with intent, which a recipe did for free.

**6.2 A shot vocabulary over per-film design.** A film designed from scratch can fit its
feature exactly, but the craft then gets spent 40 times instead of once, and quality
drifts. The trade is real: films may start to feel alike. The answer is a new shot type,
never a one-off.

**6.3 4:5 over 16:9.** 16:9 matches Linear and fits the full Console window. Most
viewers watch on phones, where 4:5 fills more of the screen, and the shots frame one
panel at a time, which fits a tall frame. Wide shots pay with a stronger tilt and crop.
No public data settles which shape performs better.

**6.4 Dark only.** Both themes would match the docs videos. A light Console on a black
stage reads as a bright rectangle and breaks the lit-object look, and one theme halves
render time. Light-theme users do not see their theme in films.

**6.5 One world over per-film fixtures.** Per-film fixtures fit each feature, but a
shared stand makes 40 films read as one story and replaces sine waves with data an
engineer believes. Rocketry is one domain; engineers from other fields may relate less.

**6.6 A faked worker clock over a Pluto clock seam.** A clock injected into the Aether
worker, with a dev-only switch in the Console, would make virtual time an explicit
product feature. It touches the worker the whole Console depends on, for a need only the
studio has. Faking the worker's `Date` follows `page.clock` and changes no product code.
The trade is real: a Console change that reads time some other way (a
`SharedArrayBuffer` clock, a server timestamp) escapes the fake, and films show it only
as plots that scroll too fast.

## 7 Open questions

- **Film viewport**: 1440x900 CSS px is the starting point. Tune it on the proof film.
- **Tagline and end card copy**: set per film, with a house default chosen on the proof
  film.
- **Track set**: pick the first three tracks from a licensed library.
