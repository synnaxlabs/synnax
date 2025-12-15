# Arc Channel I/O Semantics - Design Analysis

## Problem Statement

Arc currently uses the same syntax (`->`) for channel operations in two fundamentally different contexts:

### Reactive Context (Flow/Sequence Layer)
```arc
// Trigger: Run function when channel changes
sensor_value -> process_reading{}

// Dataflow: Connect output of one function to input of another
sensor -> transform{} -> output_channel
```

### Imperative Context (Function Body)
```arc
func initialize() true {
    0 -> tpc_cmd      // Write value to channel (side effect)
    1 -> vent_cmd     // Same operation
    return true
}
```

### The Problem
- **Syntactic ambiguity**: Same `->` operator looks reactive but executes imperatively
- **Semantic confusion**: In a function body, `0 -> tpc_cmd` looks like a trigger but it's a write operation
- **Readability**: Readers can't immediately tell the difference between dataflow connection and imperative I/O
- **Mental model inconsistency**: Imperative code shouldn't look reactive

---

## Current Semantics Analysis

### How `->` Currently Works

**In reactive context (analyzers/flow/flow.go):**
```
channel -> function{}  →  Create reactive edge, trigger function on channel change
func{} -> output_chan  →  Connect function output to channel in IR
```

**In imperative context (analyzers/statement/statement.go):**
```
value -> channel  →  analyzeChannelWrite()
  - Tracks channel as "written to" in function
  - Compiles to WASM: read value, call host binding to write channel
  - Side effect - value discarded after write
```

**The disconnect**: Users read imperative `0 -> tpc_cmd` and think "trigger something with value 0" but it actually means "queue value 0 to tpc_cmd, then proceed".

---

## Solution Approaches

### Option A: Explicit Imperative Syntax - Assignment

**Proposal**: Use assignment for imperative channel writes
```arc
func initialize() true {
    tpc_cmd = 0       // Write to channel
    mpv_cmd = 0       // Same
    vent_cmd = 1
    return true
}
```

**Advantages**:
- ✅ Clearly imperative (assignment is imperative in all languages)
- ✅ Semantically distinct from reactive `->`
- ✅ Familiar to all programmers
- ✅ Reads naturally: "set tpc_cmd to 0"
- ✅ Minimal syntax change

**Disadvantages**:
- ❌ Semantically misleading: channels are queues, not variables
- ❌ Suggests replacement rather than enqueuing
- ❌ Users might wonder why it's not mutation (since channels are queues, not scalar state)
- ❌ Inconsistent with reactive model where channels are communication primitives

**Semantic Issue**:
Assignment suggests "replace the value" but channels are queues that accumulate writes. A reader might expect this to behave like variable assignment (last write wins) when it actually enqueues.

---

### Option B: Explicit Function Call

**Proposal**: Use `send()` or `write()` function
```arc
func initialize() true {
    send(tpc_cmd, 0)    // Explicit: write value to channel
    send(mpv_cmd, 0)
    send(vent_cmd, 1)
    return true
}
```

**Alternatively with method syntax:**
```arc
func initialize() true {
    tpc_cmd.send(0)      // Method style
    mpv_cmd.send(0)
    vent_cmd.send(1)
    return true
}
```

**Advantages**:
- ✅ Completely unambiguous intent (explicit function call)
- ✅ Clearly a side effect / I/O operation
- ✅ No confusion with reactive syntax
- ✅ Extensible (could also have `send_series()`, `send_batch()`, etc.)
- ✅ Semantically correct (explicit "send" matches channel semantics)
- ✅ IDE-friendly (autocomplete, documentation)

**Disadvantages**:
- ❌ More verbose (7 chars vs 3 for `->`)
- ❌ Breaks consistency with reactive `->` syntax elsewhere
- ❌ Function-call style might feel inconsistent with operator-based reactive style
- ❌ Harder to write in quick/dirty scripts

**Semantic Advantage**:
`send()` explicitly names the operation, making channel semantics (queuing, not replacing) clear.

---

### Option C: Contextual Operator Overloading

**Proposal**: Keep `->` but make semantics context-dependent (current approach, refined)

Make it explicit through scoping:
```arc
// In stage/sequence (reactive context)
stage control {
    sensor -> process{}       // Clearly: trigger function
    process{} -> output       // Clearly: dataflow connection
}

// In function body (imperative context)
func do_things() bool {
    0 -> tpc_cmd              // Less clear: is this a trigger?
    return true
}
```

**Advantages**:
- ✅ Minimizes syntax change (existing programs work)
- ✅ Consistent appearance
- ✅ Concise

**Disadvantages**:
- ❌ Semantic overloading persists (same syntax, different meanings)
- ❌ Requires understanding context to read code
- ❌ Harder to refactor between reactive/imperative
- ❌ Parser must disambiguate based on position in AST
- ❌ Least explicit solution
- ❌ Doesn't solve the user's concern about readability

---

### Option D: Unified Syntax with Direction Semantics

**Proposal**: Different arrows for different operations
```arc
// Reactive context - unchanged
sensor -> process{}         // One-directional flow/trigger

// Imperative context - new syntax
0 ==> tpc_cmd              // Imperative write (looks different)
```

Or with left-arrow (mirroring Go):
```arc
0 <- tpc_cmd               // Read from channel
tpc_cmd <- 0               // Write to channel (same syntax as flow, but different direction)
```

**Advantages**:
- ✅ Different visual appearance for different semantics
- ✅ Minimal learning curve (borrowed from Go)

**Disadvantages**:
- ❌ Adds new operators (more to learn)
- ❌ `<-` doesn't read naturally in imperative context
- ❌ Inconsistent with reactive `->` appearance
- ❌ Could confuse users (both look like operators)

---

### Option E: Keyword-based Distinction

**Proposal**: Use `emit` for reactive, implicit function calls for imperative
```arc
// Reactive context
stage press {
    emit interval{100ms} -> tpc    // Explicit "emit" to reactive
    emit press{} => press_wait      // State transition
}

// Imperative context
func initialize() true {
    0 -> tpc_cmd                    // Unchanged, but now in function context it's clear
    return true
}
```

Or explicitly:
```arc
stage press {
    interval{100ms} -> tpc          // Reactive (in stage context)
    press{} => press_wait
}

func initialize() true {
    emit(tpc_cmd, 0)                // Explicit emit function
    emit(mpv_cmd, 0)
    return true
}
```

**Advantages**:
- ✅ `emit` keyword makes reactive operations explicit
- ✅ Distinguishes between reactive and imperative
- ✅ Reads naturally in both contexts

**Disadvantages**:
- ❌ Adds a new keyword
- ❌ Reactive code slightly more verbose
- ❌ Either requires changing reactive syntax OR having it optional (inconsistent)

---

## Comparison Matrix

| Aspect | Option A: Assign | Option B: send() | Option C: Context | Option D: Dir | Option E: emit |
|--------|------------------|------------------|-------------------|---------------|----------------|
| **Clarity** | Medium | High | Low | Medium | High |
| **Verbosity** | Low | Medium | Low | Low | Medium |
| **Semantic correctness** | Low | High | Low | Medium | High |
| **Consistency** | Medium | Low | High | Medium | Medium |
| **Backward compat** | No | No | Yes | No | No |
| **IDE friendliness** | High | High | Medium | High | High |
| **Learning curve** | Low | Low | Medium | Medium | Low |
| **Extensibility** | Medium | High | Medium | Low | High |

---

## Recommendation Analysis

### Best for Clarity & Correctness: **Option B (send function)**

```arc
func initialize() true {
    send(tpc_cmd, 0)
    send(mpv_cmd, 0)
    send(vent_cmd, 1)
    return true
}
```

**Why**:
- Makes imperative intent crystal clear
- Semantically correct (send matches channel operations)
- Completely disambiguates from reactive context
- IDE-friendly (autocomplete, docs)
- Scales well (send_series, send_batch, etc. in future)
- Readers immediately understand this is I/O, not logic

**Against current concern**:
- "This feels bad" because it's more explicit, but that's actually good for code clarity
- Slightly more verbose, but trade-off is worth it

### Second Best: **Option A (assignment)**

```arc
func initialize() true {
    tpc_cmd = 0
    mpv_cmd = 0
    vent_cmd = 1
    return true
}
```

**Why**:
- Very concise
- Reads naturally to imperative programmers
- Clear distinction from reactive `->`

**Why not best**:
- Misleading semantics (assignment suggests value replacement, not enqueuing)
- Users will misunderstand channel behavior
- Semantically inconsistent with reactive model

### Viable but Not Ideal: **Option C (context overloading)**

Keep current approach but improve documentation/formatting:

```arc
// In sequence stages (reactive)
stage press {
    interval{100ms} -> tpc       // Trigger function on interval

    // Channel write in reaction (part of dataflow)
    1 -> ox_press_cmd
}

// In functions (imperative)
func initialize() true {
    // Note: These are imperative writes, not reactive triggers
    0 -> tpc_cmd
    1 -> vent_cmd
    return true
}
```

**Why viable**:
- No breaking changes
- Can be disambiguated by parsing context (statement vs flow level)

**Why not ideal**:
- Doesn't solve the fundamental confusion
- Requires context awareness to read
- Harder to refactor

---

## Practical Example Comparison

### Current Arc (Context-dependent)
```arc
sequence tpc {
    stage initialize {
        initialize{} => next           // Function invocation
    }

    stage press_open_valves {
        1 -> ox_press_cmd              // Looks reactive but is imperative in stage
        press{} => press_wait           // State transition
    }
}

func initialize() true {
    0 -> tpc_cmd                       // Looks reactive but is imperative in function
    0 -> mpv_cmd
    0 -> supply_cmd
    1 -> vent_cmd
    return true
}
```

### With Option B (send function)
```arc
sequence tpc {
    stage initialize {
        initialize{} => next
    }

    stage press_open_valves {
        send(ox_press_cmd, 1)          // Explicitly imperative
        press{} => press_wait
    }
}

func initialize() true {
    send(tpc_cmd, 0)                   // Clearly imperative
    send(mpv_cmd, 0)
    send(supply_cmd, 0)
    send(vent_cmd, 1)
    return true
}
```

### With Option A (assignment)
```arc
sequence tpc {
    stage initialize {
        initialize{} => next
    }

    stage press_open_valves {
        ox_press_cmd = 1               // Looks imperative
        press{} => press_wait
    }
}

func initialize() true {
    tpc_cmd = 0                        // Familiar syntax
    mpv_cmd = 0
    supply_cmd = 0
    vent_cmd = 1
    return true
}
```

---

## Implementation Notes

### For Option B (Recommended)

**Changes needed:**
1. Add `send(channel, value)` to standard library (host function)
2. Update grammar: allow function calls as statements
3. Analyzer: Treat `send()` as having side effects
4. Compiler: Generate WASM binding to `send_builtin`
5. Update documentation with examples

**Backward compatibility:**
- Could keep `->` as syntactic sugar for `send()` in statement context during transition
- Deprecation path: `->` generates warning in function bodies

### For Option A (Alternative)

**Changes needed:**
1. Update grammar: `assignment` can target channels
2. Analyzer: Treat channel assignment as write operation
3. Compiler: Reuse `send()` code path
4. Document semantics carefully (channel ≠ variable)

**Backward compatibility:**
- Would require migration of existing code
- Could support both syntaxes during transition

---

## Recommendation

**Go with Option B: Explicit `send()` Function**

**Rationale**:
1. **Clarity is paramount** for domain-specific languages for hardware automation
2. **Semantic correctness** prevents user misconceptions about channel behavior
3. **Extensibility** enables future features (send_series, send_batch, conditional sends)
4. **IDE support** is better (autocomplete, type hints, documentation)
5. **No false positives** - can't accidentally look reactive when imperative
6. **Domain appropriate** - explicit I/O operations match hardware control semantics

**Implementation plan**:
1. Add `send()` and `read()` as first-class built-in functions
2. Deprecate `->` in imperative contexts (with helpful compiler error)
3. Keep `->` unchanged in reactive/flow contexts
4. Update all examples and documentation
5. Consider backward-compatible syntax: `0 -> tpc_cmd` → `send(tpc_cmd, 0)` migration message

**Example migration**:
```arc
// Old (deprecated)
func initialize() true {
    0 -> tpc_cmd
}

// New (clear)
func initialize() true {
    send(tpc_cmd, 0)
}
```

---

## Alternative: Hybrid Approach

If you want to minimize breaking changes while improving clarity, use **Option A with clear documentation**:

```arc
// In function bodies, channel assignment is imperative channel write
func initialize() true {
    tpc_cmd = 0      // Imperative: write 0 to channel queue
    vent_cmd = 1     // Imperative: write 1 to channel queue
    return true
}

// Note: Channel assignment means "enqueue to channel", not "replace channel value"
// Channels accumulate writes and maintain FIFO ordering
```

This is a middle ground:
- Minimal syntax change
- Clear visual distinction from `->`
- Familiar to imperative programmers
- Can later migrate to `send()` if confusion arises

---

## Summary Table

| Option | Syntax | Clarity | Semantics | Verbosity | Implementation |
|--------|--------|---------|-----------|-----------|-----------------|
| **A: Assign** | `chan = val` | 3/5 | 2/5 | 1/5 | Small change |
| **B: send()** | `send(chan, val)` | 5/5 | 5/5 | 3/5 | Medium change |
| **C: Context** | `val -> chan` | 2/5 | 1/5 | 1/5 | None |
| **D: Direction** | `val <-> chan` | 3/5 | 3/5 | 1/5 | Medium change |
| **E: emit** | `emit(chan, val)` | 4/5 | 4/5 | 2/5 | Medium change |

**Recommendation**: **Option B for production use** | **Option A for quick adoption**
