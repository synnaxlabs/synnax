# Writing Integration Tests

How to write integration tests well, focused on Arc, where reactive-runtime semantics
make naive tests flaky or silently wrong. For running tests and reading debug bundles,
see `docs/claude/testing.md`.

## Where Tests Live

- Cases are Python files under `integration/tests/<area>/`, registered in
  `integration/tests/<area>_tests.json` as `"<area>/<file_stem>"`.
- The conductor maps case name → file by stem and auto-discovers the single concrete
  `TestCase` subclass (class name need not match the file).

## Structure Conventions

Model new Arc tests on the existing ones — they are the source of truth for API and
style: `inline_bodies.py` (inline `stage {}`/`sequence {}` bodies, nesting),
`stl_string.py` / `stl_math.py` (shared-trigger pattern, `@dataclass` case lists),
`stage_routing.py` / `thermal_monitor.py` (sequence/stage transitions).

Headless Arc tests extend `ArcCase` (`tests.arc.arc`) and must define `arc_source`,
`arc_name_prefix`, `start_cmd_channel`, `subscribe_channels`. Keep broad tests readable:

- Break Arc source into labeled sections with a single-line divider:
  `// ──── value variables ────`. Short labels, no numbering (`Group A`, etc.).
- Group output channels into per-section lists concatenated into one `OUTPUTS`;
  `subscribe_channels = OUTPUTS` (only channels you assert on).
- Create channels in typed buckets via `create_virtual_channels`.
- One `_verify_*` method per section, called in order from `verify_sequence_execution`.
- `@dataclass` case list + loop when the same assertion repeats over varying inputs (as
  in the stl tests).

## Triggers: Shared Start by Default

The base class fires `start_cmd_channel` once on program load. Point every sequence's
entry at it (`vars_start => a_main`, `=> b_main`, ...) so all sequences activate
concurrently off one trigger; each `_verify_*` drives only its own inputs. Separate
trigger channels only when a section genuinely needs independent activation (one source
would fan out into conflicting transitions, or a flow must stay gated until a later
step).

## Arc Runtime Semantics (the traps)

- **Stage flows run concurrently.** Every flow executes asynchronously each cycle —
  never rely on line order within a stage for a value to be ready.
- **No same-tick read-after-write.** A flow cannot observe a variable written on the
  same tick. Latch on one tick, read on a later one.
- **Stage entry ignores pre-activation writes.** Channel values written before a stage
  is active are dropped. Emit a readiness marker on entry (`1 -> a_ready`) and
  `wait_for_eq("a_ready", 1)` before driving that stage.
- **First truthy transition wins.** When several `=>` transitions can be truthy in one
  cycle, the first in line order fires. Keep transitions mutually exclusive or split
  across stages.

## No Timing Hacks

- No `time.now()` in the Arc program unless timing is what's being verified.
- Never `sy.sleep` / `time.sleep` — use bounded polling helpers (`wait_for_eq`,
  `wait_for_gt`, `wait_for_ge`, ...), which fail fast with a clear message.
- Don't assert a value written on the same tick as a transition leaving the stage (race)
  — assert the definite downstream effect instead.

## Test Vectoring

- Derive the case matrix from proven behavior (the feature's own unit tests), not
  assumptions about the runtime.
- Each vector gets its own input (prefer the shared start trigger) and output so a
  failure points at exactly one line.
- Cover the axes that matter: scope, source kind, sink kind, data type.
