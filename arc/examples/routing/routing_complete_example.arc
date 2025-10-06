// Complete Routing Example: Rocket Engine Control
// Demonstrates all routing patterns with recommended semantics

// ============================================================================
// Multi-Output Stage Definitions
// ============================================================================

// Simple threshold demux
stage threshold_demux{
    threshold f64
} (value f32) {
    above f32
    below f32
} {
    // Reactive semantics: Only assigned outputs fire
    if (value > f32(threshold)) {
        above = value
        // 'below' not assigned → downstream doesn't execute
    } else {
        below = value
        // 'above' not assigned → downstream doesn't execute
    }
}

// Three-way range classifier
stage range_classifier{
    low f64
    high f64
} (value f32) {
    below_range f32
    in_range f32
    above_range f32
} {
    if (value < f32(low)) {
        below_range = value
    } else if (value > f32(high)) {
        above_range = value
    } else {
        in_range = value
    }
    // Only one output assigned per execution
}

// State-based router
stage state_router{} (value f32, state u8) {
    idle_out f32
    prestart_out f32
    running_out f32
    shutdown_out f32
    fault_out f32
} {
    // Route based on engine state
    // State: 0=idle, 1=prestart, 2=running, 3=shutdown, 4=fault

    if (state == 0) {
        idle_out = value
    } else if (state == 1) {
        prestart_out = value
    } else if (state == 2) {
        running_out = value
    } else if (state == 3) {
        shutdown_out = value
    } else {
        fault_out = value
    }
}

// Redundant sensor selector
stage sensor_selector{} (
    sensor_1 f32,
    sensor_2 f32,
    sensor_3 f32,
    health_1 u8,
    health_2 u8,
    health_3 u8
) {
    selected f32
    fault_detected u8
} {
    // Always output something
    if (health_1) {
        selected = sensor_1
        fault_detected = 0
    } else if (health_2) {
        selected = sensor_2
        fault_detected = 1  // Primary failed
    } else if (health_3) {
        selected = sensor_3
        fault_detected = 1  // Primary failed
    } else {
        selected = 0.0  // All failed
        fault_detected = 1
    }
    // Both outputs always fire
}

// Conditional logger (log rate changes with mode)
stage adaptive_logger{} (value f32, critical_mode u8) {
    high_rate_out f32
    low_rate_out f32
} {
    if (critical_mode) {
        high_rate_out = value
    } else {
        low_rate_out = value
    }
}

// ============================================================================
// System Architecture
// ============================================================================

// Oxidizer pressure monitoring
ox_pressure_1 -> sensor_selector{}
ox_pressure_2 -> sensor_selector{}
ox_pressure_3 -> sensor_selector{}
ox_health_monitor{} -> health_1 -> sensor_selector{}
ox_health_monitor{} -> health_2 -> sensor_selector{}
ox_health_monitor{} -> health_3 -> sensor_selector{}

sensor_selector{} -> {
    selected -> ox_validated,
    fault_detected -> sensor_fault_alarm{}
}

// State-dependent routing for oxidizer pressure
ox_validated -> state_router{} -> {
    idle_out -> [
        idle_monitor{rate: 1hz},
        idle_display{}
    ],
    prestart_out -> [
        prestart_monitor{rate: 10hz},
        prestart_safety_check{}
    ],
    running_out -> [
        running_logger{rate: 1khz},
        running_controller{},
        running_safety{}
    ],
    shutdown_out -> [
        shutdown_sequence{},
        shutdown_logger{rate: 100hz}
    ],
    fault_out -> [
        emergency_shutdown{},
        fault_logger{},
        fault_alarm{}
    ]
}

engine_state{} -> state_router{}

// Fuel pressure with range-based alarm
fuel_pressure -> range_classifier{low: 50.0, high: 500.0} -> {
    below_range -> [
        low_pressure_alarm{severity: 2},
        low_pressure_log{}
    ],
    in_range -> [
        normal_controller{},
        normal_display{}
    ],
    above_range -> [
        high_pressure_alarm{severity: 3},
        emergency_vent{}
    ]
}

// Chamber temperature with adaptive logging
chamber_temp -> adaptive_logger{} -> {
    high_rate_out -> critical_logger{rate: 10khz},
    low_rate_out -> normal_logger{rate: 10hz}
}

critical_mode_flag{} -> adaptive_logger{}

// Thrust with threshold monitoring
thrust_sensor -> threshold_demux{threshold: 1000.0} -> {
    above -> [
        nominal_display{},
        performance_logger{}
    ],
    below -> [
        low_thrust_alarm{},
        thrust_correction{}
    ]
}

// ============================================================================
// Example Execution Timeline
// ============================================================================

// t=0s: Engine idle (state=0)
//   ox_validated=100 → state_router
//     idle_out=100 fires → idle_monitor, idle_display execute
//     prestart_out not assigned → prestart stages DON'T execute
//     running_out not assigned → running stages DON'T execute
//
//   fuel_pressure=60 → range_classifier
//     in_range=60 fires → normal_controller, normal_display execute
//     below_range not assigned → alarm stages DON'T execute
//
//   chamber_temp=25 → adaptive_logger (critical_mode=0)
//     low_rate_out=25 fires → normal_logger executes at 10hz
//     high_rate_out not assigned → critical_logger DON'T execute

// t=5s: Engine prestart (state=1)
//   ox_validated=150 → state_router
//     idle_out not assigned → idle stages DON'T execute
//     prestart_out=150 fires → prestart_monitor, prestart_safety execute
//     running_out not assigned → running stages DON'T execute

// t=10s: Ignition (state=2, critical_mode=1)
//   ox_validated=450 → state_router
//     running_out=450 fires → running_logger(1khz), controller, safety
//     All other outputs not assigned → other stages DON'T execute
//
//   chamber_temp=2500 → adaptive_logger (critical_mode=1)
//     high_rate_out=2500 fires → critical_logger executes at 10khz
//     low_rate_out not assigned → normal_logger DON'T execute

// t=15s: Fault detected (sensor_3 fails, state=4)
//   sensor_selector: health_1=0, health_2=0, health_3=0
//     selected=0.0 fires → routing continues
//     fault_detected=1 fires → sensor_fault_alarm executes
//
//   ox_validated=0 → state_router
//     fault_out=0 fires → emergency_shutdown, fault_logger, fault_alarm
//     All other outputs not assigned → other stages DON'T execute

// ============================================================================
// Execution Efficiency
// ============================================================================

// Traditional broadcast (tee):
//   - 15 stages always execute
//   - Waste computation in wrong state

// Conditional routing with reactive semantics:
//   - Only 3-5 stages execute per state
//   - 70% reduction in unnecessary execution
//   - Critical for real-time systems!

// ============================================================================
// Static Analysis Examples
// ============================================================================

// Good: All paths assign something
stage good_demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
    } else {
        low = value
    }
}
// ✅ Compiler: OK - all outputs assigned in some path

// Warning: Output might never fire
stage sparse_demux{threshold f64} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
    }
    // Neither output assigned if value <= threshold
}
// ⚠️  Compiler: Warning - 'low' never assigned, might not fire

// Error: Output never assigned
stage broken_stage{} (value f32) {
    out f32
} {
    // Nothing assigned
}
// ❌ Compiler: Error - 'out' never assigned in any path

// Good: Multiple outputs in same execution
stage fan_out{} (value f32) {
    doubled f32
    squared f32
} {
    doubled = value * 2.0
    squared = value * value
}
// ✅ Both outputs fire every execution

// ============================================================================
// Visual Graph Representation
// ============================================================================

// Text:
sensor -> state_router{} -> {
    idle_out -> idle_handler{},
    running_out -> running_handler{},
    fault_out -> fault_handler{}
}

// Graph:
//
//    ┌────────┐      ┌───────────────┐
//    │ sensor │      │ engine_state  │
//    └───┬────┘      └───────┬───────┘
//        │                   │
//        v                   v
//    ┌────────────────────────────┐
//    │     state_router{}         │
//    │  ┌──────────────────────┐  │
//    │  │ •idle_out            │──┼────> [idle_handler]
//    │  │ •running_out         │──┼────> [running_handler]
//    │  │ •fault_out           │──┼────> [fault_handler]
//    │  └──────────────────────┘  │
//    └────────────────────────────┘
//
// During idle: green glow on idle_out port and edge
// During running: green glow on running_out port and edge
// During fault: red glow on fault_out port and edge
// Inactive ports: gray

// ============================================================================
// Syntax Summary
// ============================================================================

// Multi-output stage declaration:
stage my_stage{} (input f32) {
    output_1 f32
    output_2 f32
} {
    // Assign to outputs conditionally
}

// Named output routing:
source -> my_stage{} -> {
    output_1 -> target_1{},
    output_2 -> target_2{}
}

// Reactive semantics:
// - Assigned outputs fire (downstream executes)
// - Unassigned outputs don't fire (downstream skips)
// - Efficient for conditional routing
// - Natural for dataflow

// ============================================================================
// Key Takeaways
// ============================================================================

// 1. Named output routing is ESSENTIAL for control systems
// 2. Reactive semantics (assigned = fire) provides efficiency
// 3. Static analysis catches missing assignments
// 4. Visual editor shows active/inactive outputs at runtime
// 5. No optional types or sentinel values needed
// 6. Matches Arc's dataflow model perfectly
