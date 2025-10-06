// Conditional Routing in Arc
// How to route data differently based on runtime conditions

// ============================================================================
// Key Insight: Routing Topology vs Data Flow
// ============================================================================

// In Arc's reactive model:
// - Routing TOPOLOGY is STATIC (graph structure fixed at compile time)
// - Data FLOW is DYNAMIC (which edges carry data varies at runtime)

// This means we can't write:
//   if (condition) {
//       sensor -> path_a{}
//   } else {
//       sensor -> path_b{}
//   }
// Because flow statements aren't inside control flow!

// Instead, we use STAGES to make routing decisions

// ============================================================================
// Pattern 1: Filter Stage (Pass/Drop)
// ============================================================================

stage filter_high{
    threshold f64
} (value f32) f32 {
    if (value > f32(threshold)) {
        return value
    }
    // Don't return anything = downstream doesn't execute
    return 0.0  // Or could use optional/null pattern
}

// Static topology, dynamic flow
sensor -> filter_high{threshold: 100.0} -> alarm{}
sensor -> filter_low{threshold: 50.0} -> normal_display{}

// Data flows to alarm ONLY when sensor > 100
// Data flows to normal_display ONLY when sensor < 50

// ============================================================================
// Pattern 2: Demux Stage (Multi-Output Routing)
// ============================================================================

stage demux{
    threshold f64
} (value f32) {
    high f32
    low f32
} {
    if (value > f32(threshold)) {
        high = value
        low = 0.0
    } else {
        high = 0.0
        low = value
    }
}

// Route to different paths based on condition
sensor -> demux{threshold: 100.0} -> {
    high -> emergency_path{},
    low -> normal_path{}
}

// Both paths exist in graph, but only one receives non-zero data

// ============================================================================
// Pattern 3: Switch Stage (Enum-based Routing)
// ============================================================================

stage system_mode{} () u8 {
    // Returns: 0=startup, 1=normal, 2=shutdown
    mode u8 $= 0
    // ... logic to determine mode ...
    return mode
}

stage route_by_mode{} (value f32, mode u8) {
    startup_out f32
    normal_out f32
    shutdown_out f32
} {
    if (mode == 0) {
        startup_out = value
        normal_out = 0.0
        shutdown_out = 0.0
    } else if (mode == 1) {
        startup_out = 0.0
        normal_out = value
        shutdown_out = 0.0
    } else {
        startup_out = 0.0
        normal_out = 0.0
        shutdown_out = value
    }
}

// Static graph, dynamic routing
sensor -> route_by_mode{} -> {
    startup_out -> startup_handler{},
    normal_out -> normal_handler{},
    shutdown_out -> shutdown_handler{}
}

system_mode{} -> route_by_mode{}

// Only ONE handler receives non-zero data at a time

// ============================================================================
// Pattern 4: Guard Stage (Enable/Disable Paths)
// ============================================================================

stage gate{} (value f32, enable u8) f32 {
    if (enable) {
        return value
    }
    return 0.0  // Blocked
}

// Control which paths are active
sensor -> gate{} -> critical_processor{}
enable_signal -> gate{}

// When enable_signal is 0, critical_processor doesn't receive data

// ============================================================================
// Pattern 5: Priority Selector (Choose Between Sources)
// ============================================================================

stage select_source{} (primary f32, backup f32, use_backup u8) f32 {
    if (use_backup) {
        return backup
    }
    return primary
}

// Choose between redundant sensors
primary_sensor -> select_source{}
backup_sensor -> select_source{}
fault_detected -> select_source{} -> controller{}

// ============================================================================
// Pattern 6: Conditional Fan-out (Enable Multiple Paths)
// ============================================================================

stage conditional_broadcast{} (value f32, log_enabled u8, display_enabled u8) {
    log_out f32
    display_out f32
} {
    if (log_enabled) {
        log_out = value
    } else {
        log_out = 0.0
    }

    if (display_enabled) {
        display_out = value
    } else {
        display_out = 0.0
    }
}

sensor -> conditional_broadcast{} -> {
    log_out -> logger{},
    display_out -> display{}
}

logging_enabled -> conditional_broadcast{}
display_enabled -> conditional_broadcast{}

// ============================================================================
// Pattern 7: State Machine Router
// ============================================================================

stage state_router{} (input f32, state u8) {
    idle_out f32
    active_out f32
    error_out f32
} {
    // State: 0=idle, 1=active, 2=error
    if (state == 0) {
        idle_out = input
        active_out = 0.0
        error_out = 0.0
    } else if (state == 1) {
        idle_out = 0.0
        active_out = input
        error_out = 0.0
    } else {
        idle_out = 0.0
        active_out = 0.0
        error_out = input
    }
}

sensor -> state_router{} -> {
    idle_out -> idle_handler{},
    active_out -> active_handler{},
    error_out -> error_handler{}
}

system_state -> state_router{}

// ============================================================================
// Comparison: How Routing Syntax Affects Conditional Logic
// ============================================================================

// MULTIPLE STATEMENTS:
// - Each path explicitly listed
// - Conditional logic in stages
sensor -> demux{threshold: 100} -> high -> alarm{}
sensor -> demux{threshold: 100} -> low -> logger{}

// BRACKETS (if supported):
// - Still need demux stage for logic
// - Brackets just group the static topology
sensor -> demux{threshold: 100} -> {
    high -> [alarm{}, emergency_log{}],
    low -> [logger{}, normal_display{}]
}

// NAMED OUTPUTS:
// - Most natural for conditional routing
// - Each output has semantic meaning
sensor -> demux{threshold: 100} -> {
    high -> alarm{},
    low -> logger{}
}

// EXPLICIT TEE:
// - Tee doesn't make decisions, always broadcasts
// - Need filter stages after tee
sensor -> tee{
    gate{enable: emergency} -> alarm{},
    gate{enable: normal} -> logger{}
}

// ============================================================================
// Advanced: Nested Conditionals
// ============================================================================

stage two_level_router{} (value f32, primary_mode u8, secondary_mode u8) {
    path_a f32
    path_b f32
    path_c f32
    path_d f32
} {
    if (primary_mode == 0) {
        if (secondary_mode == 0) {
            path_a = value
        } else {
            path_b = value
        }
        path_c = 0.0
        path_d = 0.0
    } else {
        if (secondary_mode == 0) {
            path_c = value
        } else {
            path_d = value
        }
        path_a = 0.0
        path_b = 0.0
    }
}

sensor -> two_level_router{} -> {
    path_a -> handler_a{},
    path_b -> handler_b{},
    path_c -> handler_c{},
    path_d -> handler_d{}
}

// ============================================================================
// Standard Library Helper Stages (Could Be Built-in)
// ============================================================================

// Generic demux based on boolean
stage demux_bool{} (value f32, condition u8) {
    when_true f32
    when_false f32
} {
    if (condition) {
        when_true = value
        when_false = 0.0
    } else {
        when_true = 0.0
        when_false = value
    }
}

// Generic N-way switch
stage switch_u8{} (value f32, selector u8) {
    out_0 f32
    out_1 f32
    out_2 f32
    out_3 f32
} {
    out_0 = 0.0
    out_1 = 0.0
    out_2 = 0.0
    out_3 = 0.0

    if (selector == 0) {
        out_0 = value
    } else if (selector == 1) {
        out_1 = value
    } else if (selector == 2) {
        out_2 = value
    } else if (selector == 3) {
        out_3 = value
    }
}

// Usage:
sensor -> demux_bool{} -> {
    when_true -> critical_path{},
    when_false -> normal_path{}
}

condition_signal -> demux_bool{}
