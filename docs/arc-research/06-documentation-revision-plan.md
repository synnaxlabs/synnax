# Arc Documentation Revision Plan

This document captures research and decisions for revising Arc documentation to be more
beginner-friendly while serving advanced users.

---

## 1. Target Audience

### Primary Persona

Two overlapping profiles:

1. **Visual Programming Background** (LabVIEW users)
   - Comfortable with control systems concepts
   - Experience with visual/block-based programming
   - Little to no text-based coding experience
   - Understands what they want to accomplish, unfamiliar with syntax

2. **PLC Configuration Background** (Rockwell, Siemens users)
   - Deep control systems knowledge
   - Configures hardware through vendor software
   - May have written ladder logic or function blocks
   - Text-based programming syntax is foreign

**Key insight:** These users understand valves, pressure transducers, control loops, and
safety interlocks. They don't need to learn _what_ to do, they need to learn _how to
express it_ in Arc syntax.

### Secondary Audience

**Experienced programmers new to Arc/Synnax:**

- LabVIEW power users comfortable with complex systems
- Low-level C programmers from embedded/firmware backgrounds
- Software engineers integrating with hardware systems

**Key insight:** This audience can navigate documentation independently. They'll jump
directly to reference material and skip tutorials. They need:

- Quick syntax lookup (reference pages)
- Conceptual differences from what they know
- Not hand-holding, but clear technical accuracy

**Implication:** Don't dumb down reference material to serve beginners. Instead, create
clear separation between learning paths (tutorials/concepts) and lookup resources
(reference). Advanced users will self-select into reference.

### Industries

**Active verticals:**

- Aerospace (rocket test stands, propulsion systems)
- Industrial manufacturing (PLC automation)
- Scientific instrumentation (quantum dilution refrigerators)

**Implication for examples:**

- Use generic examples by default (temperature, pressure, valve, pump)
- Avoid aerospace-specific jargon in introductory material (`ox_pt_1` → `tank_pressure`)
- Can include aerospace-specific examples in advanced how-to guides as ONE industry
- Examples should translate across industries without domain expertise

---

## 2. User Journey & Entry Points

### How Users Discover Arc

**Primary entry paths:**

1. **Synnax docs homepage → navigation**
   - Browsing to understand Synnax capabilities
   - Need: "What is Arc?" and "Is this right for me?"
   - May not have decided to use Arc yet

2. **Console → "Create Arc automation" → docs link**
   - Already in the product, ready to try Arc
   - Need: Practical "now what do I do" guidance
   - Decision already made, want to execute

**Implication:** Need two distinct early pages:

- Introduction/overview (serves path 1): What is Arc, when to use it, comparison to
  alternatives
- Get Started tutorial (serves path 2): You're in Console, here's your first automation

### In-App Guidance

**Current state:**

- User clicks "Create Arc automation"
- Dropped into empty editor (no template, no example code)
- Link to documentation is available in editor

**Implication:** When users click the docs link, they're thinking:

- "I'm staring at a blank screen"
- "What do I type first?"
- "Give me something I can copy and modify"

**Get-started page must:**

- Provide immediately copy-pasteable code
- Show the minimal viable example upfront
- Not require reading 3 pages of concepts first

### Prior Synnax Knowledge

**Users arriving at Arc typically already know:**

- Channels (creating, naming, data types)
- Data acquisition tasks and drivers
- Schematics and visualizations
- Manual control through Console
- Ranges for storing data

**Arc is an advanced feature** in the typical user journey:

1. Set up data acquisition
2. Build schematics for manual control
3. Visualize and store data
4. → Then explore automation (Arc)

**Implications:**

- DON'T need to explain basic Synnax concepts (channels, drivers)
- CAN assume familiarity with Console UI
- DO need to explain how Arc differs from manual control they already know
- Frame Arc as "automating what you've been doing manually"

---

## 3. Prerequisites & Context

### Quality of Prerequisite Docs (Channels, Drivers)

**Approach:** Link to prerequisite docs as a safety net, but don't re-explain basics.

A brief "Before you start" note with links is sufficient:

- Most users already know this material
- Links serve users who jumped ahead
- Arc docs shouldn't duplicate Synnax fundamentals

### Synnax 101 / Core Concepts

**Existing resource:** `/reference/concepts/` directory covers Synnax fundamentals.

**Approach for Arc docs:**

- Link to concepts directory for foundational knowledge (channels, control authority,
  etc.)
- Arc concepts section focuses only on Arc-specific concepts (reactive execution,
  stateful variables, sequences)
- Don't duplicate general Synnax concepts within Arc docs

---

## 4. Graph Mode vs Text Mode

### Graph Mode Readiness

**Current state:**

- Graph mode IS stable and used in production
- Documentation is deferred (not prioritized now)
- Need to design structure that accommodates Graph mode docs later

### Recommended Entry Point for Beginners

**Include in Introduction/Overview page:**

- What Graph mode is (visual block-based editor)
- What Text mode is (code editor)
- When to use each:
  - Graph mode: Simple alarms, threshold monitoring, basic logic
  - Text mode: Complex sequences, stateful logic, advanced control
- Graph mode is discoverable in Console UI (user chooses when creating automation)

**Current docs focus:** Text mode (full documentation) **Graph mode:** Explained
conceptually, detailed docs deferred

### Strategy for Graph Mode Integration (Future)

**Structure to accommodate future Graph mode docs:**

- Introduction page covers both modes (already planned above)
- Graph mode docs would live as parallel section: `/arc/graph/`
- Concepts that apply to both modes stay in shared `/arc/concepts/`
- Mode-specific how-to guides in respective sections

**Placeholder approach:**

- Mention Graph mode exists in intro
- Note that Graph mode docs are "coming soon" or similar
- Text mode docs are complete and standalone

---

## 5. Use Cases & Examples

### Top Use Cases (in priority order)

1. **Multi-step test sequences** ← Most common use case
2. **Threshold alarms / notifications**
3. **Simple on/off control** (bang-bang, valve control)
4. **Data transformations** (unit conversions, derived values)
5. **Rate-of-change monitoring**

**Implication:** Current docs bury sequences (the #1 use case) deep in the structure.
Consider:

- Introducing sequences earlier in the learning path
- Using a simple sequence as a "realistic first example"
- Making the how-to guide for sequences more prominent

### Example Domain (Aerospace vs Generic)

Use generic examples by default (see Section 1: Industries).

- `tank_pressure` instead of `ox_pt_1`
- `valve_cmd` instead of `press_vlv_cmd`
- Industry-specific examples only in advanced how-to guides

---

## 6. User Feedback & Learning Paths

### Beta User Feedback

**Source:** Firehawk Aerospace demo call (Jan 2026)

**Positive reactions:**

- "This would be pretty intuitive" (Jon, after walkthrough)
- "That's beautiful. Okay, that directly addresses my concern." (Jordan, after seeing
  function extraction)
- Syntax was well-received once explained
- Integration with Console UI (buttons, schematics) seen as major benefit

**Key confusion point - concurrent execution:**

> "Your UI shows line based execution, but logically it doesn't." — Jason (new engineer)

Jason asked "when is that wait actually being executed?" because the visual layout
suggests sequential execution, but everything in a stage runs concurrently. **This is
the #1 documentation gap.** Users understand it once explained, but it's not intuitive
from looking at the syntax.

**Concerns raised:**

- Jordan worried about maintainability: "if I wrote all the conditions out... maybe 50
  to 60 lines" across stages — addressed by function extraction pattern
- Questions about where code executes (console vs driver vs node)
- Integration with Python client for advanced features (ref prop, metadata, calculated
  channels)
- Timeout requirements for safety ("there should be a timeout on pretty much any
  sequence")

**Questions users asked:**

1. Can multiple sequences run at once? (Yes)
2. Can parameters be updated while running? (Yes, via channels)
3. How do I know what commands can exit a stage? (Visual exists, no button graying yet)
4. How does control authority work? (Still being designed)
5. Where does the code execute? (Driver, configurable)

**Documentation implications:**

- **MUST** explain concurrent execution within stages early and clearly
- Show function extraction pattern for managing complexity
- Explain the difference between line-by-line code (traditional) and concurrent flows
  (Arc)
- Address "where does it run" question
- Emphasize timeout patterns for safety

### Successful Learning Paths

From the call, the successful explanation pattern was:

1. Show a working example first (the tank pressurization sequence)
2. Walk through what happens when you run it
3. THEN explain the syntax and semantics
4. Address the "everything runs at once" concept explicitly

This matches good documentation practice: concrete example → behavior → syntax →
concepts

---

**Source:** Inversion Space demo call (Jan 2026)

**Explicit documentation request:**

> "Do you have docs set up for it yet? Like how fleshed out is ARC at this point?"
> "Especially when you guys launch the docs. I would love to dig into it to see what it
> can do." — Rob

**Priority/execution model questions came up immediately** - Same pattern as Firehawk.
Rob's first question: "What's the priority scheduling, how that generally works behind
the scenes?"

**Other questions raised:**

- Where does code execute? (Driver, configurable by OS)
- What provides real-time guarantees? (OS-dependent: NI Linux RT, Ubuntu RT)
- Version control for Arc files? (Text-based, git-friendly)
- Integration with Python client for advanced features
- Custom device/protocol support

**Key quote on importance:**

> "This is a very core tool that we would need to run in order to use your program." —
> Rob

**Documentation implications:**

- Docs are explicitly requested and anticipated
- Execution model / "where does it run" needs clear explanation
- Real-time capabilities and limitations should be documented
- Version control / git workflow is valued

---

**Source:** Northrop Grumman demo call (Kyle, Jan 2026)

**Syntax praised:**

> "The way that this sequence is expressed is like super readable and really elegant." —
> Kyle

**Use case distinction raised:**

- **Test automation**: Very defined, start from known state, if aborted restart from
  beginning
- **Control**: Going in and out, pausing, jumping states, want it to keep running

Kyle: "You kind of need both, because it depends on are you doing test automation... or
your control side."

**Timer/state transition question:** If you have "wait an hour" and pause 30 minutes in,
then resume, does the timer reset? Kyle said for sequencing "they would want that timer
to start over again" but noted both behaviors are useful.

**API/programmatic control requested:**

> "I would almost want buttons to actually hit the start and stop of the arc itself. Or
> API or SDK or something."

**Complexity concern validated:** Kyle agreed that having to write custom wait blocks
with time tracking "is just not super test engineer friendly."

**Documentation implication:**

- Document the distinction between test automation and control use cases
- Explain timer/state reset behavior clearly
- Keep syntax simple; avoid pushing complexity onto users

---

## 7. Technical & Practical Considerations

### Error Message Quality

**Error messages are beginner-friendly.** The compiler provides actionable guidance, not
just terse type errors.

Example: If someone tries to add `i32` and `f64`, the error explains how to fix it
(e.g., cast with `f64(x)`), not just "type mismatch."

**Source:** `/arc/go/analyzer/humanize.go` contains the diagnostic messages.

**Documentation implication:**

- Don't need extensive "common errors" troubleshooting section
- Can trust the compiler to guide users
- Docs should focus on teaching concepts, not decoding errors
- Reference section can list error codes briefly; compiler does the heavy lifting

### Video/Visual Assets

**Videos will be recorded before release.** Placeholders in docs will be filled.

**Key videos needed:**

- Creating an Arc automation (Console workflow)
- Deploying to a driver
- (Possibly) Running a sequence with buttons

**Documentation implication:**

- Keep video placeholders in docs
- Videos will address the "blank editor" problem for beginners
- Text should still be self-sufficient (videos are supplementary)

### Interactive Playground

**Console IS the playground.** Users experiment directly in the Console editor, which
provides syntax highlighting, autocomplete, and inline error feedback.

No separate web-based playground needed or planned.

**Implication:**

- Docs can assume users have Console open while learning
- Copy-pasteable examples let users try things immediately
- LSP/editor feedback helps catch errors as they type

---

## 8. Strategy & Constraints

### Timeline & Release Requirements

Docs should be ready for release. Videos will be recorded before launch.

### Resource Constraints

**Flexible.** Can do comprehensive rewrites or surgical fixes based on impact vs.
effort.

Recommendation will be provided in strategy section.

### Documentation Maintenance

**Engineering team maintains docs. Docs will be living/frequently updated.**

**Implication:**

- Don't need to make everything perfect upfront
- Can iterate based on user feedback
- Structure should be easy for engineers to update (clear organization, modular pages)
- Avoid overly complex cross-references that become stale

### Specific Requests

**Remove input/output routing table examples.** Some current docs use routing table
syntax like `{ sensor1: a, sensor2: b } -> func{}` or `-> { out1: channel1 }`. Remove
these examples; they add complexity for limited benefit.

---

## 9. Revised Documentation Strategy

### Key Insights Summary

1. **Primary audience**: PLC/LabVIEW users who understand control systems but not
   text-based programming syntax
2. **#1 confusion point**: Concurrent execution within stages ("your UI shows line-based
   execution, but logically it doesn't")
3. **#1 use case**: Multi-step test sequences (currently buried in docs)
4. **Users already know Synnax**: Channels, drivers, schematics, manual control
5. **Graph mode exists but docs deferred**: Should mention it, explain when to use each
   mode
6. **Examples should be generic**: Not aerospace-specific (`tank_pressure` not
   `ox_pt_1`)
7. **Error messages are good**: Compiler guides users; docs don't need extensive
   troubleshooting
8. **Videos coming**: Will help with "blank editor" problem

### Structural Changes

**1. Split Get-Started into two pages:**

| Current                           | Proposed                                                        |
| --------------------------------- | --------------------------------------------------------------- |
| `get-started.mdx` (does too much) | `introduction.mdx` - What is Arc, Graph vs Text, when to use it |
|                                   | `get-started.mdx` - Hands-on first automation                   |

**2. Reorder to surface sequences earlier:**

Current order buries sequences (the #1 use case) after 4 concept pages.

Proposed concept order:

1. Reactive Execution (shortened, concrete example first)
2. **Sequences and Stages** (move UP - this is what users want)
3. Channels and Series
4. Stateful Variables

**3. Add "Graph vs Text" section to introduction:**

Explain both modes exist, when to use each, that Graph mode docs are coming.

### Content Changes

**1. Introduction page (NEW):**

- What Arc is (automation language for Synnax)
- Who it's for (test engineers automating procedures)
- Graph mode vs Text mode (brief comparison, recommend based on use case)
- Link to Get Started when ready to try it

**2. Get-Started page (REWRITE):**

- Remove conceptual explanation (moved to Introduction)
- Start with "Open Console, create Arc automation" with video
- Minimal first example using GENERIC names (`temperature`, `heater_cmd`)
- Explain the example line by line (including `f64`, `->`, `{}`)
- Deploy and run
- "What's Next" links

**3. Reactive Execution (REVISE):**

- START with concrete example (two functions reading same sensor)
- THEN name the concept ("this is called stratified execution")
- Move the "why it matters" example to the TOP
- Shorten the theory; users learn by doing

**4. Sequences and Stages (REVISE + MOVE UP):**

- This is the #1 use case; should come earlier
- Add prominent callout: "Everything in a stage runs at the same time"
- The current content is good; just needs better positioning

**5. All How-To Guides (REVISE):**

- Replace aerospace jargon with generic names
- Remove routing table examples
- Add brief "what you'll learn" at top of each

**6. Examples throughout:**

- Replace `ox_pt_1` → `tank_pressure` or `pressure_sensor`
- Replace `fuel_tc_2` → `temperature` or `tank_temp`
- Replace `press_vlv_cmd` → `valve_cmd` or `heater_cmd`
- Keep ONE aerospace example in test-sequences.mdx as industry-specific example

### Files to Change

| File                                | Action                                        | Priority |
| ----------------------------------- | --------------------------------------------- | -------- |
| `introduction.mdx`                  | CREATE                                        | High     |
| `get-started.mdx`                   | REWRITE                                       | High     |
| `concepts/reactive-execution.mdx`   | REVISE (reorder)                              | High     |
| `concepts/sequences-and-stages.mdx` | REVISE (move up, add callout)                 | High     |
| `concepts/channels-and-series.mdx`  | REVISE (remove routing tables)                | Medium   |
| `how-to/*.mdx` (all)                | REVISE (generic names, remove routing tables) | Medium   |
| `reference/syntax.mdx`              | REVISE (remove routing tables)                | Low      |
| `_nav.ts` files                     | UPDATE (new structure)                        | High     |

### Routing Tables to Remove

Search for and remove/simplify:

- `{ sensor1: a, sensor2: b } -> func{}`
- `-> { output1: channel1, output2: channel2 }`
- Any multi-input or multi-output routing syntax

Replace with simpler patterns or multiple separate flows.

### Priority Order

**Phase 1: Critical Path (do first)**

1. Create `introduction.mdx`
2. Rewrite `get-started.mdx`
3. Revise `reactive-execution.mdx` (reorder for clarity)
4. Update navigation (`_nav.ts`)

**Phase 2: Structure Improvements** 5. Move sequences-and-stages.mdx up in nav order 6.
Add "concurrent execution" callout to sequences page 7. Remove routing table examples
from concepts

**Phase 3: Polish** 8. Replace aerospace jargon throughout how-to guides 9. Remove
routing table examples from how-to guides 10. Clean up reference pages

### Success Criteria

After revisions, a PLC engineer with no text programming experience should be able to:

1. Read Introduction and understand if Arc is right for them
2. Complete Get-Started and have a working automation in <15 minutes
3. Understand that "everything in a stage runs at the same time" before writing
   sequences
4. Write a basic test sequence without getting confused by syntax
