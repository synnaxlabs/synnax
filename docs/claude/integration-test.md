# Writing Integration Tests

This guide covers how to write integration tests well, with a focus on Arc tests where
the runtime's reactive semantics make naive tests flaky or silently wrong. For running
tests via the test conductor and reading debug bundles, see @docs/claude/testing.md.

## Where Tests Live

- Test cases are Python files under `integration/tests/<area>/` (e.g.
  `integration/tests/arc/`).
- Each case is registered in `integration/tests/<area>_tests.json` as
  `"<area>/<file_stem>"` (e.g. `"arc/variables"`).
- The conductor maps the case name to the file by stem and auto-discovers the single
  concrete `TestCase` subclass in it. The class name does not need to match the file.

## Structure Conventions

Model new Arc tests on the existing ones, which are the source of truth for both API and
style:

- `inline_bodies.py` for inline `stage {}` / `sequence {}` bodies, nesting, and sibling
  sequences.
- `stl_string.py` and `stl_math.py` for the shared-trigger pattern and `@dataclass` case
  lists.
- `stage_routing.py` and `thermal_monitor.py` for sequence/stage transitions.

Headless Arc tests extend `ArcCase` (`tests.arc.arc`), drive the program through the
Python client, and must define `arc_source`, `arc_name_prefix`, `start_cmd_channel`, and
`subscribe_channels`. Conventions that keep a broad test readable:

- Break the Arc source into clearly labeled sections with a single-line divider, not a
  multi-line block:

  ```
  // ──────────────────────────── value variables ────────────────────────────
  ```

  Keep the label short and descriptive. Do not number sections (`Group A`, etc.).

- Group output channel names into per-section lists and concatenate them into one
  `OUTPUTS`. Set `subscribe_channels = OUTPUTS` (only the channels you assert on).
- Create channels in typed buckets and build them with `create_virtual_channels`.
- Give each section its own `_verify_*` method, called in order from
  `verify_sequence_execution`.
- Use a `@dataclass` case list plus a loop when the same assertion repeats over varying
  inputs, the same way the stl tests do.

## Triggers: Shared Start by Default

The base class fires `start_cmd_channel` once when it loads the program. Make that the
default activation for everything: point every sequence's entry at it.

```
vars_start => a_main
vars_start => b_main
vars_start => c_main
```

All sequences then activate concurrently off one trigger, and each `_verify_*` method
only drives its own inputs. Only introduce a separate trigger channel when a section
genuinely needs independent activation, for example when one source would otherwise fan
out into two conflicting transitions, or when a flow must stay gated until a later step.

## Arc Runtime Semantics (the traps)

These are the behaviors that make Arc integration tests fail in non-obvious ways.

- **Stage flows run concurrently.** Every flow in a stage executes asynchronously each
  cycle. Do not rely on the order of lines within a stage for a value to be ready.
- **No same-tick read-after-write.** A flow cannot observe a variable that was written
  on the same tick. Latch a value on one tick, then read it on a later tick.
- **Stage entry ignores pre-activation writes.** A channel value written before its
  stage is active is dropped. Write a stage's inputs only after the stage is active.
  Emit a readiness marker on entry (`1 -> a_ready`) and `wait_for_eq("a_ready", 1)`
  before driving that stage.
- **First truthy transition wins.** When several `=>` transitions in a stage can be
  truthy in the same cycle, the first in line order fires. Keep transitions mutually
  exclusive, or split them across stages so each assertion is unambiguous.
- **Variable kinds.** _literal_ (`:=` / `$=`, a stateful value cell), _channel read_
  (read-only stream over channels), and _channel read/write_ (aliases a channel it reads
  and writes). A `:=` literal re-seeds to its declared value on every scope entry; `$=`
  persists.

## No Timing Hacks

- Do not call `time.now()` in the Arc program unless the timing itself is what the test
  verifies. It makes results nondeterministic.
- Never use `sy.sleep` or `time.sleep` in a test. Use the bounded polling helpers
  (`wait_for_eq`, `wait_for_gt`, `wait_for_ge`, etc.), which fail fast with a clear
  message instead of hanging.
- Do not assert a value written on the same tick as a transition that leaves the stage;
  that is a race. Assert the definite downstream effect instead.

## Test Vectoring

- Derive the matrix of cases from proven behavior (the feature's own unit tests), not
  from assumptions about how the runtime should behave.
- Give each vector its own input (but prefer the shared start trigger where possible)
  and output so a failure points at exactly one line rather than a shared channel.
- Cover the axes that matter for the feature: scope, variable kind, source kind, sink
  kind, and data type.
