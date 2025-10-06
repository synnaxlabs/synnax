// How to "Not Send" to Named Outputs
// Critical design decision for conditional routing

// ============================================================================
// The Problem
// ============================================================================

stage demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        // What about 'low'? How do we say "don't send"?
    } else {
        low = value
        // What about 'high'?
    }
}

// ============================================================================
// OPTION 1: Sentinel Values (Current Examples)
// ============================================================================

stage demux_v1{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        low = 0.0  // Use 0 as "no data"
    } else {
        high = 0.0  // Use 0 as "no data"
        low = value
    }
}

// PROS:
// - Simple to implement
// - No language changes needed
// - Always produces a value

// CONS:
// - 0 might be a valid data value!
// - Downstream stage still executes (wastes computation)
// - Semantic confusion: is this 0 data or "no data"?
// - Doesn't work for all types (what's sentinel for string?)

// Example problem:
sensor -> demux_v1{threshold: 100} -> {
    high -> alarm{},
    low -> logger{}
}

// If sensor reads 0.0, logger receives 0.0
// If sensor reads 50.0, logger receives 50.0
// Can't distinguish between:
//   - Real zero value
//   - "No send" signal

// ============================================================================
// OPTION 2: Undefined Outputs (Don't Assign)
// ============================================================================

stage demux_v2{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        // Don't assign to 'low' at all
    } else {
        low = value
        // Don't assign to 'high' at all
    }
}

// Semantic: If output variable is never assigned, downstream doesn't execute

// PROS:
// - Natural semantics
// - No sentinel value confusion
// - Downstream stage skips execution (efficient)

// CONS:
// - Requires tracking "was this assigned?"
// - What if forgotten to assign in some branch?
// - Static analysis: ensure all paths assign something?

// Example:
sensor -> demux_v2{threshold: 100} -> {
    high -> alarm{},     // Executes only when value > 100
    low -> logger{}      // Executes only when value <= 100
}

// ============================================================================
// OPTION 3: Optional Types
// ============================================================================

stage demux_v3{threshold f64} (value f32) {
    high f32?  // Optional type (might be null/none)
    low f32?
} {
    if (value > f32(threshold)) {
        high = some(value)
        low = none
    } else {
        high = none
        low = some(value)
    }
}

// PROS:
// - Explicit in type system
// - Clear semantics: none = no data
// - Downstream must handle optional
// - Works for all types

// CONS:
// - Adds optional types to type system
// - More complex
// - Downstream needs unwrapping
// - Performance: allocate memory for optional wrapper?

// Example:
sensor -> demux_v3{threshold: 100} -> {
    high -> alarm{},     // alarm receives optional, must unwrap
    low -> logger{}      // logger receives optional, must unwrap
}

// Downstream stage must handle:
stage alarm{} (value f32?) {
    if (value.is_some()) {
        trigger := value.unwrap()
        // ...
    }
}

// ============================================================================
// OPTION 4: Explicit No-Op Channel
// ============================================================================

stage demux_v4{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        low = void  // Explicit "no send"
    } else {
        high = void
        low = value
    }
}

// PROS:
// - Explicit intent
// - No confusion with sentinel values
// - Could be zero-cost (compiler removes void sends)

// CONS:
// - New keyword/concept
// - Still need to assign to every output

// ============================================================================
// OPTION 5: Reactive Semantics (Implicit)
// ============================================================================

stage demux_v5{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
    } else {
        low = value
    }
}

// Semantic: Output only "fires" when assigned
// Reactive model: downstream only executes on output change

// PROS:
// - Clean reactive semantics
// - Natural for dataflow
// - Efficient (no unnecessary execution)

// CONS:
// - Must track "was assigned this execution"
// - What about initialization? (first execution)
// - Static analysis: warn if output never assigned?

// ============================================================================
// OPTION 6: Return Statement for Outputs
// ============================================================================

stage demux_v6{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        return high: value
    } else {
        return low: value
    }
}

// PROS:
// - Explicit control flow
// - Only one output per execution
// - Clear semantics

// CONS:
// - Can't send to multiple outputs in same execution
// - Awkward for stages that need to update multiple outputs

// ============================================================================
// OPTION 7: Enable/Disable Flags
// ============================================================================

stage demux_v7{threshold f64} (value f32) {
    high f32
    high_enabled u8
    low f32
    low_enabled u8
} {
    if (value > f32(threshold)) {
        high = value
        high_enabled = 1
        low = 0.0  // Doesn't matter
        low_enabled = 0
    } else {
        high = 0.0  // Doesn't matter
        high_enabled = 0
        low = value
        low_enabled = 1
    }
}

// Downstream checks enabled flag:
sensor -> demux_v7{threshold: 100} -> {
    high -> gate{} -> alarm{},
    high_enabled -> gate{},
    low -> gate{} -> logger{},
    low_enabled -> gate{}
}

stage gate{} (value f32, enabled u8) f32 {
    if (enabled) {
        return value
    }
    return 0.0  // Or don't return?
}

// PROS:
// - Explicit control
// - Works with current type system

// CONS:
// - Verbose
// - Boilerplate for every output
// - Need gate stages everywhere

// ============================================================================
// Comparison Matrix
// ============================================================================

// | Approach         | Explicit | Efficient | Type Safe | Simple |
// |------------------|----------|-----------|-----------|--------|
// | Sentinel (0)     | No       | No        | No        | Yes    |
// | Undefined        | No       | Yes       | Partial   | Yes    |
// | Optional Types   | Yes      | Partial   | Yes       | No     |
// | Void Keyword     | Yes      | Yes       | Yes       | Yes    |
// | Reactive         | No       | Yes       | Partial   | Yes    |
// | Return           | Yes      | Yes       | Yes       | Partial|
// | Enable Flags     | Yes      | Partial   | Yes       | No     |

// ============================================================================
// Recommendation Analysis
// ============================================================================

// For Arc's reactive dataflow model, OPTION 2 (Undefined) or OPTION 5 (Reactive)
// are most natural:

// REACTIVE SEMANTICS (Recommended):

stage demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        // 'low' not assigned this execution → downstream doesn't run
    } else {
        low = value
        // 'high' not assigned this execution → downstream doesn't run
    }
}

// Rules:
// 1. Output variables declared but not assigned → downstream doesn't execute
// 2. Only assigned outputs "fire" this execution
// 3. Compiler warning if output never assigned in any path
// 4. Runtime tracks "dirty" flags per output

// Example execution:
// Iteration 1: sensor=120 → high=120, low undefined → alarm runs, logger doesn't
// Iteration 2: sensor=50  → high undefined, low=50 → alarm doesn't, logger runs
// Iteration 3: sensor=120 → high=120, low undefined → alarm runs, logger doesn't

// ============================================================================
// Special Case: Multiple Outputs in Same Execution
// ============================================================================

stage fan_out{} (value f32) {
    raw f32
    doubled f32
    squared f32
} {
    // All outputs fire every execution
    raw = value
    doubled = value * 2.0
    squared = value * value
}

sensor -> fan_out{} -> {
    raw -> logger{},
    doubled -> display{},
    squared -> analyzer{}
}

// All three downstream stages execute every time

// ============================================================================
// Special Case: Stateful Outputs
// ============================================================================

stage accumulator{} (value f32, reset u8) {
    sum f32
    count u32
} {
    total f32 $= 0.0
    counter u32 $= 0

    if (reset) {
        total = 0.0
        counter = 0
    } else {
        total = total + value
        counter = counter + 1
    }

    sum = total
    count = counter
    // Both outputs fire every execution
}

// ============================================================================
// Implementation: Dirty Flags
// ============================================================================

// Compiler generates:

// struct demux_outputs {
//     f32 high;
//     bool high_dirty;
//     f32 low;
//     bool low_dirty;
// };
//
// void demux_execute(demux_outputs* out, f32 value, f64 threshold) {
//     out->high_dirty = false;
//     out->low_dirty = false;
//
//     if (value > (f32)threshold) {
//         out->high = value;
//         out->high_dirty = true;
//     } else {
//         out->low = value;
//         out->low_dirty = true;
//     }
// }
//
// // In stratified execution:
// demux_execute(&demux_state, sensor_value, 100.0);
// if (demux_state.high_dirty) {
//     alarm_execute(demux_state.high);
// }
// if (demux_state.low_dirty) {
//     logger_execute(demux_state.low);
// }

// ============================================================================
// Static Analysis: Ensure Outputs Assigned
// ============================================================================

// Compiler should warn:

stage bad_demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        // Warning: 'low' never assigned in this branch
    }
    // Warning: 'high' might not be assigned in all paths
}

// Compiler error if output CAN'T be assigned in any execution:

stage broken{} (value f32) {
    out f32
} {
    // Error: output 'out' never assigned
}

// ============================================================================
// Recommended Approach
// ============================================================================

// Use REACTIVE SEMANTICS with dirty flags:
// 1. Outputs only "fire" when assigned
// 2. Compiler tracks assignments per branch
// 3. Runtime uses dirty flags for efficiency
// 4. Static analysis warns about unassigned outputs
// 5. No sentinel values, no optional types needed

// This matches Arc's reactive dataflow model and provides:
// - Natural semantics
// - Efficient execution
// - Type safety
// - Clear intent
