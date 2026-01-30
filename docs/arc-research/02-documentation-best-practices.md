# Arc Documentation Plan

This document captures decisions about how Arc documentation should be structured, what
content to include, and style guidelines to follow.

---

## 1. Documentation Framework

Arc docs follow the **Diátaxis framework** as a guide, with flexibility to blend
categories where it makes sense:

| Type              | Purpose                | Arc Application                   |
| ----------------- | ---------------------- | --------------------------------- |
| **Tutorials**     | Learning-oriented      | Get Started, progressive How-To's |
| **How-To Guides** | Problem-oriented       | Task-focused guides               |
| **Explanations**  | Understanding-oriented | Concepts section                  |
| **Reference**     | Information-oriented   | Language Reference                |

**Key principle**: Denser pages with related concepts grouped together, not one concept
per page. Fewer pages, more substance on each.

---

## 2. Arc Documentation Structure

```
Arc Documentation
│
├── Introduction
│   └── What is Arc, why it exists, beta notice
│
├── Get Started
│   └── Console workflow → create Arc → first program → deploy
│
├── Concepts
│   ├── Reactive Execution Model
│   │   └── Includes continuous (`->`) vs one-shot (`=>`) edges
│   ├── Channels and Series
│   ├── Stateful Variables
│   └── Sequences and Stages
│
├── How-To Guides
│   ├── Unit Conversions
│   ├── Sensor Averaging & Filtering
│   ├── Derived Calculations
│   ├── Rate of Change Detection
│   ├── Alarms (threshold → deadband → multi-condition, progressive)
│   ├── Bang-Bang Control
│   ├── Test Sequence Automation (basic → timed → abort → conditional, progressive)
│   └── Sensor Validation
│
├── Effective Arc
│   └── Best practices, common pitfalls (single tight page)
│
└── Language Reference
    ├── Syntax Overview
    ├── Types (primitives, channels, series)
    ├── Operators
    ├── Functions (declaration, config, inputs, outputs)
    ├── Sequences and Stages
    ├── Built-In Functions (grouped by category with examples)
    └── Error Codes
```

### Writing Order

**Phase 1: Get people running**

1. Introduction (what is Arc, beta notice)
2. Get Started (Console workflow → first program → deploy)

**Phase 2: Core understanding** 3. Reactive Execution Model 4. Channels and Series 5.
Stateful Variables 6. Sequences and Stages

**Phase 3: Practical application** 7. How-To Guides (starting with simpler ones) 8.
Effective Arc (best practices)

**Phase 4: Reference** 9. Language Reference (syntax, types, operators, built-ins, error
codes)

---

## 3. Content Decisions

### What to Include

- **Text mode only** for initial docs (graph mode documented later)
- **Introduction** separate from **Get Started** (conceptual vs practical entry points)
- **Effective Arc** page with best practices and common pitfalls
- **Error codes** documented in Language Reference (current codes, expand over time)
- **Examples embedded** in Tutorials and How-To guides (not a separate Examples section)

### What to Exclude (for now)

- Graph mode documentation
- Compilation to WebAssembly details (implementation detail)
- Version numbers in examples
- Separate Troubleshooting section (fold error codes into Reference)

### Beta Status

- Note in Introduction only (not a banner on every page)
- Be upfront about limitations where relevant in each section

---

## 4. Style Guidelines

### Tone

- **"You"-focused** with descriptive concept introductions (matches existing Synnax
  docs)
- Occasional "we" where natural
- No heavy use of "let's" or first-person plural
- Conversational but precise
- Encouraging but honest about limitations

### Code Examples

- **Code first, then explanation** (default)
- Prose first only when context is genuinely needed upfront
- Show expected output using:
  - Channel values
  - Annotated diagrams (data flow with values)
  - Text descriptions
- No Console screenshots (high maintenance)

### Page Structure

- Denser pages with related concepts grouped
- Use existing `next`/`prev` navigation (no explicit Prerequisites sections)
- Moderate linking (not every term, but helpful links)
- Callouts (`<Note.Note>`) for common mistakes
- Video placeholders: `[VIDEO: Description of what to record]`

### Reference Format

Mix of expanded entries and grouped categories:

````markdown
## Statistical Functions

These functions operate on series data.

### avg

Returns the average of a series.

```arc
readings -> avg() -> avg_reading
```
````

**Parameters:** `series` (any numeric series) **Returns:** `f64`

### min, max

Return the minimum or maximum value in a series.

```arc
readings -> min() -> min_reading
readings -> max() -> max_reading
```

```

---

## 5. Arc Vocabulary (Consistent Terminology)

Use these terms consistently throughout all documentation.

### Program Structure

| Term               | Definition                                      |
| ------------------ | ----------------------------------------------- |
| **Arc automation** | The complete program file                       |
| **Sequence**       | A state machine containing ordered stages       |
| **Stage**          | A state within a sequence                       |

### Dataflow

| Term                       | Definition                                           |
| -------------------------- | ---------------------------------------------------- |
| **Node**                   | A function instantiation in the dataflow graph       |
| **Edge**                   | A connection between nodes (`->` or `=>`)            |
| **Continuous edge** (`->`) | Reactive connection that fires on every input        |
| **One-shot edge** (`=>`)   | Fires once when condition becomes true, then stops   |
| **Flow statement**         | A chain of nodes connected by edges                  |

### Functions

| Term                 | Definition                              |
| -------------------- | --------------------------------------- |
| **Function**         | A reusable block definition             |
| **Config parameter** | Parameter set at instantiation time (`{}`) |
| **Input parameter**  | Parameter received at runtime (`()`)    |
| **Output**           | Return value(s) from a function         |

### Variables

| Term                  | Definition                                |
| --------------------- | ----------------------------------------- |
| **Variable**          | A named value                             |
| **Local variable**    | Variable that resets each execution (`:=`) |
| **Stateful variable** | Variable that persists across executions (`$=`) |

### Data Types

| Term          | Definition                              |
| ------------- | --------------------------------------- |
| **Channel**   | External data source/sink from Synnax   |
| **Series**    | Array of homogeneous values             |
| **Primitive** | Basic type (`i64`, `f64`, `str`, etc.)  |

### Execution

| Term                     | Definition                                    |
| ------------------------ | --------------------------------------------- |
| **Reactive execution**   | Code runs in response to channel events       |
| **Stratified execution** | Deterministic ordering of node execution      |
| **Trigger**              | Event that starts a sequence                  |

### Console/Deployment

| Term        | Definition                                   |
| ----------- | -------------------------------------------- |
| **Deploy**  | Upload and activate an automation            |
| **Rack**    | The driver/location where automation executes |
| **Toolbar** | UI controls for the automation               |

### Terms to AVOID

| Don't Use              | Use Instead              |
| ---------------------- | ------------------------ |
| Program, script        | Arc automation           |
| Reactive edge          | Continuous edge (`->`)   |
| Transition edge        | One-shot edge (`=>`)     |
| Pipe, connection       | Edge                     |
| Block (for nodes)      | Node                     |
| State (for stages)     | Stage                    |
| Persistent variable    | Stateful variable        |

---

## 6. Error Code Documentation

### Error Code Format

`ARC<category><number>`

### Categories

| Category    | Range    | Description                          |
| ----------- | -------- | ------------------------------------ |
| Type System | ARC2xxx  | Type mismatches and constraint violations |
| Functions   | ARC3xxx  | Argument count/type errors           |
| Symbols     | ARC4xxx  | Undefined or redefined symbols       |

### Current Error Codes

| Code        | Name                      | Description                              |
| ----------- | ------------------------- | ---------------------------------------- |
| **ARC2001** | TypeMismatch              | Incompatible types in operation or assignment |
| **ARC2003** | TypeConstraintViolation   | Type doesn't satisfy constraint          |
| **ARC3001** | FuncArgCount              | Wrong number of arguments                |
| **ARC3002** | FuncArgType               | Function argument has wrong type         |
| **ARC4001** | SymbolUndefined           | Referenced symbol not found              |
| **ARC4002** | SymbolRedefined           | Symbol declared twice in same scope      |

### Severity Levels

- **Error** — Prevents compilation
- **Warning** — Potential problem, compiles anyway
- **Info** — Informational
- **Hint** — Suggested improvement

### Common Error Messages

| Message                                              | Meaning                                    |
| ---------------------------------------------------- | ------------------------------------------ |
| `"X is not a channel"`                               | Using non-channel in flow                  |
| `"X is not a function"`                              | Calling non-function                       |
| `"X is not a channel or sequence"`                   | Invalid flow target                        |
| `"X has more than one parameter"`                    | Anonymous config on multi-param function   |
| `"input routing table must precede a func invocation"` | Misplaced routing table                 |
| `"compound assignment requires numeric element type"` | Type restriction on `+=`, `-=`, etc.      |

---

## 7. How-To Guide Specifications

### Guide 1: Unit Conversions

**Goal:** Convert sensor readings between units
**Features:** Basic flow chains, arithmetic nodes
**Complexity:** Simple

**Example:** Fahrenheit to Celsius, PSI to bar, voltage to engineering units

### Guide 2: Sensor Averaging & Filtering

**Goal:** Smooth noisy sensor data
**Features:** Series functions (`avg`, `min`, `max`)
**Complexity:** Simple

**Example:** Rolling average, bounds checking

### Guide 3: Derived Calculations

**Goal:** Compute values from multiple inputs
**Features:** Multi-input functions
**Complexity:** Simple

**Example:** Mass flow rate from pressure and temperature

### Guide 4: Rate of Change Detection

**Goal:** Calculate how fast a value is changing
**Features:** Stateful variables
**Complexity:** Medium

**Example:** Pressure rise rate for leak detection

### Guide 5: Alarms (Progressive)

**Goal:** Trigger warnings when values exceed limits
**Features:** Status system, stateful logic, boolean combinations
**Complexity:** Simple → Medium

**Progression:**
1. Basic threshold alarm
2. Deadband alarm (prevent chatter)
3. Multi-condition alarm (AND/OR logic)

### Guide 6: Bang-Bang Control

**Goal:** Simple on/off control with hysteresis
**Features:** Conditional flows, stateful deadband
**Complexity:** Simple

**Example:** Heater control

### Guide 7: Test Sequence Automation (Progressive)

**Goal:** Step through ordered stages
**Features:** Sequences, stages, transitions, timing
**Complexity:** Medium

**Progression:**
1. Basic sequence (idle → pressurize → hold → complete)
2. Timed sequence (automatic stage advancement)
3. Abort handling (emergency interrupt)
4. Conditional progression (advance when conditions met)

### Guide 8: Sensor Validation

**Goal:** Check if sensor readings are within expected bounds
**Features:** Input validation patterns
**Complexity:** Simple

**Example:** Reject out-of-range readings before control logic

---

## 8. Effective Arc Page Outline

A single, tight page with best practices and common pitfalls.

### Best Practices

- Put safety conditions first (line order matters for priority)
- Use continuous edges (`->`) for streaming data
- Use one-shot edges (`=>`) for state transitions
- Keep stages focused on one responsibility
- Use stateful variables for values that need to persist
- Name channels clearly (snake_case)

### Common Pitfalls

- Forgetting that line order determines priority
- Using `->` when `=>` is needed (or vice versa)
- Not initializing stateful variables
- Connecting incompatible types
- Expecting loops (use stateful variables + reactive execution instead)

### Performance Considerations

- Target 1kHz for control loop rates
- Keep flow chains short where possible
- Use `interval` for precise timing

---

## 9. Sources and References

### Documentation Frameworks

- [Diátaxis Framework](https://diataxis.fr/)

### Language Documentation Examples

- **Rust (The Book)**: Progressive learning, "why" explanations
- **Go**: Tour + Effective Go idioms guide
- **TypeScript**: Handbook approach, gradual complexity
- **Stripe**: Task-based organization, code samples everywhere

### Synnax Documentation

- Location: `/docs/site/`
- Framework: Astro 5.16 + MDX + React 19
- Arc location: `/docs/site/src/pages/reference/control/arc/`
- Arc syntax highlighting: Already configured in Astro
```
