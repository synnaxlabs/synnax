# Arc Documentation Session Prompts

This document guides Claude through 5 documentation sessions for Arc. Each session
builds on the previous one.

---

## How to Use This Document

### Before Starting Any Session

1. **Read the research folder** in this order:
   - `01-arc-language-deep-dive.md` — Technical understanding of Arc
   - `02-documentation-best-practices.md` — Structure, vocabulary, and content decisions
   - `03-synnax-docs-structure.md` — Where docs live, page format, navigation patterns
   - `04-writing-style.md` — Voice, tone, and the session prompt to use

2. **Read existing Synnax docs** to match patterns:
   - `/docs/site/src/pages/reference/control/python/` — Similar control docs
   - `/docs/site/src/pages/reference/control/embedded/` — Another control language

3. **Read Arc source** as needed for accuracy:

   **Always read first:**
   - `/arc/docs/spec.md` — Language specification (authoritative reference)

   **For syntax and grammar details:**
   - `/arc/go/parser/ArcParser.g4` — Parser grammar (exact syntax rules)
   - `/arc/go/parser/ArcLexer.g4` — Lexer grammar (tokens, keywords, operators)

   **For type system and validation:**
   - `/arc/go/analyzer/analyzer.go` — Type checking, validation rules
   - `/arc/go/types/` — Type definitions, unit system, constraints

   **For execution model and semantics:**
   - `/arc/go/stratifier/` — How execution order is computed
   - `/arc/go/ir/` — Intermediate representation (nodes, edges, dataflow)

   **For runtime behavior:**
   - `/arc/go/runtime/` — Go runtime (WASM execution, host functions)
   - `/driver/arc/` — C++ runtime (real-time execution, timing modes)

   **For real working examples:**
   - `/arc/go/compiler/compiler_test.go` — 2000+ lines of test cases
   - `/arc/go/analyzer/analyzer_test.go` — Validation test cases

   **For built-in functions:**
   - `/arc/go/compiler/` — Look for built-in node implementations
   - `/arc/go/ir/` — Node definitions (interval, wait, avg, etc.)

   **For error codes and messages:**
   - `/arc/go/analyzer/` — Error definitions and validation messages
   - `/arc/go/symbol/` — Symbol resolution errors

   **For Console integration:**
   - `/console/src/arc/` — Editor, toolbar, deployment UI

### After Completing a Session

1. **Update this document**:
   - Mark the completed session as `[DONE]`
   - Add a "Learnings" section under the session with:
     - Any terminology decisions made
     - Patterns that worked well
     - Things to avoid in future sessions
     - Cross-references that should be added

2. **Update subsequent session prompts** if:
   - You discovered a better way to explain something
   - You made a terminology choice that affects later sessions
   - You found content that should move to a different session

3. **Stage the docs** you wrote using `git add`

---

## Writing Process for Each Session

1. Read the session prompt below
2. **Read `01-arc-language-deep-dive.md` thoroughly** — This is the most important
   research document. It contains accurate syntax, APIs, semantics, and examples that
   have been verified against the source code. Treat it as your primary reference.
3. Review `04-writing-style.md` for voice and tone
4. Check `02-documentation-best-practices.md` for vocabulary (use consistent terms)
5. **Research any gaps**: If you need to document syntax, APIs, or behavior not covered
   in the research docs, read the Arc source files listed above. Do not guess or invent
   syntax. Verify against:
   - `/arc/docs/spec.md` for language semantics
   - `/arc/go/parser/ArcParser.g4` for exact syntax
   - `/arc/go/compiler/compiler_test.go` for working examples
6. Write the pages in the order listed
7. Use the MDX format from `03-synnax-docs-structure.md`
8. Place files in `/docs/site/src/pages/reference/control/arc/`
9. Update `_nav.ts` files as needed

---

## Session 1: Foundation

**Status**: [x] DONE

**Output**: 2 pages

- `index.astro` (redirect to get-started)
- `get-started.mdx`

**Also create**:

- `_nav.ts` for the arc section

### Prompt

```
You are writing Arc documentation for Synnax.

Read the arc-research folder:
- 01-arc-language-deep-dive.md
- 02-documentation-best-practices.md
- 03-synnax-docs-structure.md
- 04-writing-style.md
- 05-session-prompts.md (this file, for context)

Read existing control docs for patterns:
- /docs/site/src/pages/reference/control/python/get-started.mdx
- /docs/site/src/pages/reference/control/embedded/get-started.mdx

Read Arc source for accuracy:
- /arc/docs/spec.md — Overview and basics
- /console/src/arc/ — Console integration (editor, deployment)

Write the following pages:

1. **get-started.mdx** — First contact with Arc
   - What Arc is (1-2 paragraphs, not a wall of text)
   - Beta notice (brief, not a banner)
   - Creating your first Arc automation in Console
   - A minimal working example (sensor -> calculation -> output)
   - Deploying to a driver
   - What to learn next (link to concepts)

2. **_nav.ts** — Navigation for the Arc section
   - Follow the pattern from python/_nav.ts
   - Include placeholders for future sections (concepts, how-to, reference)

3. **index.astro** — Redirect to get-started
   - Follow the pattern from python/index.astro

Keep it short. Someone should be able to read this in 5 minutes and have something running.

Use the writing style from 04-writing-style.md. No "this guide will show you" or
"in this section". Just get to the point.
```

### Learnings

**Completed**: 2026-01-28

**Files created**:

- `/docs/site/src/pages/reference/control/arc/_nav.ts`
- `/docs/site/src/pages/reference/control/arc/index.astro`
- `/docs/site/src/pages/reference/control/arc/get-started.mdx`

**Files modified**:

- `/docs/site/src/pages/reference/control/_nav.ts` (added ARC_NAV import and reference)

**Terminology decisions**:

- Used "Arc automation" consistently (not "program" or "script")
- Used "driver" for deployment target (not "rack")
- Used "config parameter" for `{}` syntax (matches 02-documentation-best-practices.md)

**Patterns that worked well**:

- Starting with a minimal 3-line example before the more realistic alarm example
- Explaining the reactive model briefly in the intro without going deep
- Linking to driver docs for driver explanation rather than re-explaining

**Terminology correction (post-session)**:

- Changed "rack" to "driver" throughout all docs
- The UI may still show "rack" in some places (e.g., `Rack.SelectSingle` component), but
  user-facing documentation should use "driver"
- Updated: 01-arc-language-deep-dive.md, 02-documentation-best-practices.md,
  get-started.mdx, and this file

**Things to note for future sessions**:

- Video placeholders added for `create-automation` and `deploy-automation`
- The "What's Next" section references concepts, how-to guides, and reference that don't
  exist yet — these will be linked properly in Session 2+
- Kept beta notice concise (single Note, not a banner)

---

## Session 2: Concepts

**Status**: [x] DONE

**Depends on**: Session 1 complete

**Output**: 4 pages + navigation

- `concepts/_nav.ts`
- `concepts/reactive-execution.mdx`
- `concepts/channels-and-series.mdx`
- `concepts/stateful-variables.mdx`
- `concepts/sequences-and-stages.mdx`

### Prompt

```
You are continuing Arc documentation for Synnax.

Read the arc-research folder (all files).
Read the get-started.mdx you wrote in Session 1.

Read Arc source for technical accuracy:
- /arc/docs/spec.md — Language specification
- /arc/go/stratifier/ — Execution order computation (for reactive-execution.mdx)
- /arc/go/ir/ — Node/edge definitions (for understanding dataflow)
- /arc/go/analyzer/analyzer.go — Type checking (for channels-and-series.mdx)
- /arc/go/compiler/compiler_test.go — Working examples of sequences and stages

Write 4 concept pages:

1. **reactive-execution.mdx** — How Arc executes
   - Stratified execution (why it matters for safety)
   - Continuous edges (->) vs one-shot edges (=>)
   - Snapshot consistency
   - No loops — use reactive patterns instead
   - Include a diagram showing data flow

2. **channels-and-series.mdx** — Data in Arc
   - Channels as connections to Synnax telemetry
   - Reading from channels (reactive vs imperative context)
   - Writing to channels
   - Series (arrays) and element-wise operations
   - Channel discovery in the editor

3. **stateful-variables.mdx** — Persistence across executions
   - Local variables (:=) vs stateful variables ($=)
   - When to use stateful variables
   - Common patterns (counters, accumulators, previous value tracking)
   - Stateful variables reset on stage re-entry

4. **sequences-and-stages.mdx** — State machines
   - What sequences and stages are
   - Concurrency within stages (all flows run simultaneously)
   - Line order determines priority for transitions
   - Triggering sequences from channels
   - Transition targets (next, stage_name, sequence_name)
   - A realistic example (pressurize -> ignite -> shutdown)

Each page should be self-contained but link to related concepts.
Use the vocabulary from 02-documentation-best-practices.md consistently.
```

### Learnings

**Completed**: 2026-01-28

**Files created**:

- `/docs/site/src/pages/reference/control/arc/concepts/_nav.ts`
- `/docs/site/src/pages/reference/control/arc/concepts/reactive-execution.mdx`
- `/docs/site/src/pages/reference/control/arc/concepts/channels-and-series.mdx`
- `/docs/site/src/pages/reference/control/arc/concepts/stateful-variables.mdx`
- `/docs/site/src/pages/reference/control/arc/concepts/sequences-and-stages.mdx`

**Files modified**:

- `/docs/site/src/pages/reference/control/arc/_nav.ts` (added CONCEPTS_NAV import)
- `/docs/site/src/pages/reference/control/arc/get-started.mdx` (added next/prev links,
  updated What's Next section with proper links)

**Terminology decisions**:

- Used "continuous edge" and "one-shot edge" consistently (per
  02-documentation-best-practices.md)
- Used "node" for function instantiations in the dataflow graph
- Used "stateful variable" not "persistent variable"
- Used "stage" not "state" for sequence stages

**Patterns that worked well**:

- Explaining stratified execution with a concrete example (two functions reading same
  sensor)
- The "concurrency within stages" callout with safety conditions listed first
- Providing practical examples for each stateful variable pattern (counter, accumulator,
  rate of change, running max, toggle)
- The comprehensive test sequence example in sequences-and-stages.mdx

**Things to note for future sessions**:

- No diagrams were added (mentioned in prompt but deferred) — could add ASCII diagrams
  or image placeholders later
- The "Expressions in Flows" section could be expanded in the reference docs
- The channel types section is brief — more detail goes in the Language Reference
- Test sequence example in sequences-and-stages.mdx can be referenced in How-To guides

**Cross-references established**:

- reactive-execution.mdx → stateful-variables.mdx (for loop replacement)
- stateful-variables.mdx → sequences-and-stages.mdx (for reset on stage re-entry)
- get-started.mdx → all concept pages (in What's Next section)

---

## Session 3: How-To Guides — Data Processing

**Status**: [x] DONE

**Depends on**: Session 2 complete

**Output**: 5 guides + navigation

- `how-to/_nav.ts`
- `how-to/unit-conversions.mdx`
- `how-to/sensor-averaging.mdx`
- `how-to/derived-calculations.mdx`
- `how-to/rate-of-change.mdx`
- `how-to/sensor-validation.mdx`

### Prompt

```
You are continuing Arc documentation for Synnax.

Read the arc-research folder (all files).
Read the concept pages from Session 2.

Read Arc source for built-in functions:
- /arc/docs/spec.md — Function signatures and behavior
- /arc/go/ir/ — Built-in node definitions (avg, min, max, etc.)
- /arc/go/compiler/compiler_test.go — Examples using built-ins

Write 5 how-to guides for data processing:

1. **unit-conversions.mdx**
   - Converting between units (F to C, PSI to bar, voltage to engineering units)
   - Using Arc's unit system for type safety
   - Simple flow chains

2. **sensor-averaging.mdx**
   - Smoothing noisy sensor data
   - Using avg, min, max with duration/count resets
   - Rolling statistics

3. **derived-calculations.mdx**
   - Computing values from multiple inputs
   - Multi-input functions
   - Example: mass flow rate from pressure and temperature

4. **rate-of-change.mdx**
   - Calculating how fast a value is changing
   - Using stateful variables to track previous values
   - Example: pressure rise rate for leak detection

5. **sensor-validation.mdx**
   - Checking if readings are within expected bounds
   - Rejecting out-of-range values before control logic
   - Input validation patterns

Each guide should:
- Start with the problem being solved
- Show the Arc code first, then explain
- Use realistic channel names (ox_pt_1, fuel_tc_2, not "sensor1")
- Be completable in under 10 minutes
```

### Learnings

**Completed**: 2026-01-28

**Files created**:

- `/docs/site/src/pages/reference/control/arc/how-to/_nav.ts`
- `/docs/site/src/pages/reference/control/arc/how-to/unit-conversions.mdx`
- `/docs/site/src/pages/reference/control/arc/how-to/sensor-averaging.mdx`
- `/docs/site/src/pages/reference/control/arc/how-to/derived-calculations.mdx`
- `/docs/site/src/pages/reference/control/arc/how-to/rate-of-change.mdx`
- `/docs/site/src/pages/reference/control/arc/how-to/sensor-validation.mdx`

**Files modified**:

- `/docs/site/src/pages/reference/control/arc/_nav.ts` (added HOW_TO_NAV import)
- `/docs/site/src/pages/reference/control/arc/concepts/sequences-and-stages.mdx` (added
  next link to how-to guides)

**Patterns that worked well**:

- Starting each guide with the problem being solved before showing code
- Using realistic channel names (ox_pt_1, tank_tc_1, fuel_flow, etc.)
- Providing multiple examples per guide that build on each other
- Including edge case handling (division by zero, negative values, etc.)
- Showing how to use derived values in sequences (rate-of-change.mdx)

**Terminology decisions**:

- Used "stateful variables" consistently (not "persistent")
- Used "config parameters" for `{}` syntax
- Used "validation" not "filtering" for sensor checking
- Used "range check" for min/max validation
- Used "spike filter" for rate-based outlier rejection

**Things to note for future sessions**:

- The sensor-averaging guide notes that Arc lacks loops, limiting true moving average
  implementation. Consider documenting workarounds or mentioning built-in nodes if they
  become available.
- Several guides show absolute value calculation as `if x < 0 { x = 0.0 - x }` since Arc
  doesn't have an abs() function. Consider mentioning in reference docs.
- The derived calculations guide notes that mass flow equation is simplified. Real
  aerospace users will need proper isentropic flow equations.
- The voting/median patterns could be referenced in the "Effective Arc" page (Session 4)

**Cross-references established**:

- unit-conversions.mdx linked from sequences-and-stages.mdx (next)
- Each how-to guide links to next/prev guides in sequence
- rate-of-change.mdx includes example using rates in sequences
- sensor-validation.mdx includes example using validation in sequences

---

## Session 4: How-To Guides — Control & Alarms + Effective Arc

**Status**: [x] DONE

**Depends on**: Session 3 complete

**Output**: 4 pages

- `how-to/alarms.mdx`
- `how-to/bang-bang-control.mdx`
- `how-to/test-sequences.mdx`
- `effective-arc.mdx`

### Prompt

```
Read /docs/arc-research/ and then execute Session 4 from 05-session-prompts.md
```

### Details

Write 3 progressive how-to guides + 1 best practices page:

1. **alarms.mdx** — Basic threshold alarm (ox_pt_1 > 500 -> set_status), deadband alarm
   (prevent chatter with stateful hysteresis), multi-condition alarm (AND/OR logic).
   Build each example on the previous.

2. **bang-bang-control.mdx** — Simple on/off control with hysteresis. Example: heater
   control maintaining temperature band. Using stateful variables for deadband state.

3. **test-sequences.mdx** — Progressive: basic sequence (idle -> pressurize -> hold ->
   complete), timed sequence (with wait), abort handling, conditional progression. Build
   toward realistic rocket test stand sequence.

4. **effective-arc.mdx** — Best practices: safety conditions first (line order =
   priority), use -> for streaming and => for transitions, keep stages focused. Common
   mistakes: using -> when => needed, forgetting stateful variable initialization,
   expecting loops. Performance: 1kHz max, keep flows short. Single tight page.

Read Arc source for sequences and timing: /arc/docs/spec.md,
/arc/go/compiler/compiler_test.go, /arc/go/ir/ (interval, wait, set_status),
/driver/arc/ (C++ runtime timing modes).

### Learnings

**Completed**: 2026-01-28

**Files created**:

- `/docs/site/src/pages/reference/control/arc/how-to/alarms.mdx`
- `/docs/site/src/pages/reference/control/arc/how-to/bang-bang-control.mdx`
- `/docs/site/src/pages/reference/control/arc/how-to/test-sequences.mdx`
- `/docs/site/src/pages/reference/control/arc/effective-arc.mdx`

**Files modified**:

- `/docs/site/src/pages/reference/control/arc/how-to/_nav.ts` (added alarms,
  bang-bang-control, test-sequences)
- `/docs/site/src/pages/reference/control/arc/_nav.ts` (added effective-arc link)
- `/docs/site/src/pages/reference/control/arc/how-to/sensor-validation.mdx` (added next
  link to alarms)

**Patterns that worked well**:

- Progressive examples in alarms.mdx (basic threshold → deadband → multi-condition →
  latching)
- Progressive examples in test-sequences.mdx (basic → timed → abort → complete rocket
  test)
- Concrete aerospace examples (rocket engine test sequence) make concepts tangible
- Repeating safety-first principle across multiple pages reinforces best practice
- Effective-arc.mdx as a tight reference combines best practices with common mistakes

**Terminology decisions**:

- Used "deadband" consistently for hysteresis regions
- Used "latching alarm" for alarms that require acknowledgment
- Used "bang-bang control" (industry standard term)
- Avoided "PID" since Arc doesn't support it natively yet

**Things to note for future sessions**:

- The effective-arc page references error messages that should be documented in detail
  in the reference/errors.mdx (Session 5)
- Bang-bang control page notes that more sophisticated control algorithms would need
  future Arc library additions
- Test sequences page could be referenced from Console documentation when explaining how
  to wire buttons to channels
- The complete rocket test sequence example in test-sequences.mdx could be reused as a
  downloadable example

**Cross-references established**:

- sensor-validation.mdx → alarms.mdx (next)
- alarms.mdx → bang-bang-control.mdx (next)
- bang-bang-control.mdx → test-sequences.mdx (next)
- test-sequences.mdx → effective-arc.mdx (next)
- effective-arc.mdx links back to concepts (stateful variables, sequences) implicitly

---

## Session 5: Language Reference

**Status**: [x] DONE

**Depends on**: Session 4 complete

**Output**: 7 reference pages + navigation

- `reference/_nav.ts`
- `reference/syntax.mdx`
- `reference/types.mdx`
- `reference/operators.mdx`
- `reference/functions.mdx`
- `reference/sequences.mdx`
- `reference/built-ins.mdx`
- `reference/errors.mdx`

### Prompt

```
Read /docs/arc-research/ and then execute Session 5 from 05-session-prompts.md
```

### Details

Write 7 reference pages. These are lookup references, not tutorials. Use tables, keep
examples minimal (1-3 lines each).

**Source files to read for accuracy:**

- `/arc/docs/spec.md` — Authoritative spec (READ FIRST)
- `/arc/go/parser/ArcParser.g4` — Exact syntax rules
- `/arc/go/parser/ArcLexer.g4` — Tokens, keywords, operators
- `/arc/go/analyzer/` — Error codes and validation rules

**Pages:**

1. **syntax.mdx** — Comments, identifiers, literals, statement structure, flow
   statements
2. **types.mdx** — Primitives, channels, series, type defaults, unit annotations,
   casting
3. **operators.mdx** — Arithmetic, comparison, logical, assignment, flow operators,
   precedence table
4. **functions.mdx** — Declaration syntax, config `{}`, inputs `()`, outputs, optional
   params
5. **sequences.mdx** — Sequence/stage declaration, transitions (`next`, named,
   cross-sequence), entry points
6. **built-ins.mdx** — Group by category: timing (interval, wait), statistics (avg, min,
   max), I/O (on, write), signal (select, stable_for), status (set_status), utility
   (constant, len, now)
7. **errors.mdx** — Error code format (ARC<cat><num>), current codes from
   02-documentation-best-practices.md, common messages from effective-arc.mdx

**Notes from previous sessions:**

- Arc lacks `abs()`, `sqrt()`, trig functions (note in built-ins)
- No loops (mention in syntax, link to stateful-variables concept)
- `^` is exponentiation not XOR (highlight in operators)
- Boolean is `u8` (0=false, non-zero=true)

### Learnings

**Completed**: 2026-01-28

**Files created**:

- `/docs/site/src/pages/reference/control/arc/reference/_nav.ts`
- `/docs/site/src/pages/reference/control/arc/reference/syntax.mdx`
- `/docs/site/src/pages/reference/control/arc/reference/types.mdx`
- `/docs/site/src/pages/reference/control/arc/reference/operators.mdx`
- `/docs/site/src/pages/reference/control/arc/reference/functions.mdx`
- `/docs/site/src/pages/reference/control/arc/reference/sequences.mdx`
- `/docs/site/src/pages/reference/control/arc/reference/built-ins.mdx`
- `/docs/site/src/pages/reference/control/arc/reference/errors.mdx`

**Files modified**:

- `/docs/site/src/pages/reference/control/arc/_nav.ts` (added REFERENCE_NAV import)
- `/docs/site/src/pages/reference/control/arc/effective-arc.mdx` (added next link to
  syntax)

**Patterns that worked well**:

- Tables for quick lookup (operators, types, error codes)
- Minimal examples (1-3 lines) with clear purpose
- Grouping related content (e.g., all operators in one place)
- Cross-linking between reference pages (types → operators → functions)
- Including "common fixes" table in errors.mdx for quick troubleshooting

**Terminology consistency**:

- Used "config parameter" for `{}` syntax throughout
- Used "input parameter" for `()` syntax
- Used "continuous edge" and "one-shot edge" for flow operators
- Used "truthy/falsy" for boolean semantics
- Used "element-wise" for series operations

**Things to note**:

- The built-ins.mdx notes that `abs()`, `sqrt()`, and trig functions are planned
- Errors.mdx consolidates error codes from spec and common messages from effective-arc
- All reference pages are lookup-focused, not tutorial-style
- Build passes successfully

**Cross-references established**:

- effective-arc.mdx → syntax.mdx (next)
- syntax.mdx → types.mdx (next)
- types.mdx → operators.mdx (next)
- operators.mdx → functions.mdx (next)
- functions.mdx → sequences.mdx (next)
- sequences.mdx → built-ins.mdx (next)
- built-ins.mdx → errors.mdx (next)
- syntax.mdx links to stateful-variables concept for loops explanation

---

## Final Checklist

After all sessions are complete:

- [x] All pages written and staged
- [x] Navigation updated in `/docs/site/src/pages/reference/control/_nav.ts`
- [x] Cross-references between pages verified
- [x] Arc syntax highlighting working in code blocks
- [x] Build passes: `cd docs/site && pnpm build`
- [ ] Manual review of rendered pages
