// Conditional Routing: Syntax Comparison
// Which routing syntax best expresses conditional logic?

// ============================================================================
// Scenario: Route sensor data based on operating mode
// ============================================================================

stage mode_router{} (value f32, mode u8) {
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

// ============================================================================
// OPTION 1: Multiple Statements
// ============================================================================

// Explicit but verbose
sensor -> mode_router{};
mode_signal -> mode_router{};

mode_router{} -> startup_out -> startup_handler{};
mode_router{} -> normal_out -> normal_handler{};
mode_router{} -> shutdown_out -> shutdown_handler{};

// PROS:
// - Each path clearly listed
// - Easy to comment individual paths
// - Simple to add/remove paths

// CONS:
// - Repetitive "mode_router{}" references
// - Relationship between paths not obvious
// - Hard to see this is a conditional fan-out

// ============================================================================
// OPTION 2: Named Output Routing (Current Recommendation)
// ============================================================================

sensor -> mode_router{} -> {;
    startup_out -> startup_handler{},;
    normal_out -> normal_handler{},;
    shutdown_out -> shutdown_handler{};
}

mode_signal -> mode_router{};

// PROS:
// - Clear semantic mapping: output name → handler
// - Grouped under one router
// - Easy to see conditional fan-out pattern
// - Maps naturally to multi-output stage concept

// CONS:
// - Requires named output syntax support
// - More complex parsing

// ============================================================================
// OPTION 3: Brackets + Named Outputs (If Both Supported)
// ============================================================================

// When multiple stages consume same conditional output
sensor -> mode_router{} -> {;
    startup_out -> [startup_logger{}, startup_monitor{}],;
    normal_out -> [normal_controller{}, normal_logger{}],;
    shutdown_out -> [shutdown_sequence{}, shutdown_log{}];
}

mode_signal -> mode_router{};

// PROS:
// - Combines benefits: semantic names + concise fan-out
// - Very readable for complex routing
// - Clear which outputs go together

// CONS:
// - Most complex syntax
// - Requires both named outputs AND brackets

// ============================================================================
// OPTION 4: Explicit Tee (Wrong Tool)
// ============================================================================

// Tee doesn't make decisions - it broadcasts to ALL outputs
// Wrong pattern for conditional routing

sensor -> mode_router{} -> tee{;
    startup_handler{},
    normal_handler{},
    shutdown_handler{}
}

// PROBLEM: All handlers receive data from ALL outputs!
// This would require filter stages after tee:

sensor -> mode_router{} -> tee{;
    filter{output: startup} -> startup_handler{},;
    filter{output: normal} -> normal_handler{},;
    filter{output: shutdown} -> shutdown_handler{};
}

// This is verbose and error-prone

// ============================================================================
// Complex Example: Nested Conditionals
// ============================================================================

stage priority_router{} (value f32, priority u8, enabled u8) {
    high_priority_out f32
    low_priority_out f32
    disabled_out f32
} {
    if (!enabled) {
        disabled_out = value
        high_priority_out = 0.0
        low_priority_out = 0.0
    } else if (priority > 5) {
        high_priority_out = value
        low_priority_out = 0.0
        disabled_out = 0.0
    } else {
        low_priority_out = value
        high_priority_out = 0.0
        disabled_out = 0.0
    }
}

// With named outputs - clear structure
sensor -> priority_router{} -> {;
    high_priority_out -> [;
        critical_alarm{},
        immediate_log{},
        emergency_shutdown{}
    ],
    low_priority_out -> [;
        normal_log{},
        trend_monitor{}
    ],
    disabled_out -> [;
        diagnostic_only{}
    ]
}

priority_signal -> priority_router{};
enabled_signal -> priority_router{};

// With multiple statements - structure lost
sensor -> priority_router{};
priority_signal -> priority_router{};
enabled_signal -> priority_router{};

priority_router{} -> high_priority_out -> critical_alarm{};
priority_router{} -> high_priority_out -> immediate_log{};
priority_router{} -> high_priority_out -> emergency_shutdown{};
priority_router{} -> low_priority_out -> normal_log{};
priority_router{} -> low_priority_out -> trend_monitor{};
priority_router{} -> disabled_out -> diagnostic_only{};

// Hard to see groupings!

// ============================================================================
// Real-World: Valve Control with Multiple Conditions
// ============================================================================

stage valve_controller{} (
    pressure f32,
    flow_rate f32,
    override_open u8,
    override_close u8,
    auto_mode u8
) {
    open_command f32
    close_command f32
    hold_command f32
    fault_command f32
} {
    if (override_open) {
        open_command = 1.0
        close_command = 0.0
        hold_command = 0.0
        fault_command = 0.0
    } else if (override_close) {
        open_command = 0.0
        close_command = 1.0
        hold_command = 0.0
        fault_command = 0.0
    } else if (!auto_mode) {
        open_command = 0.0
        close_command = 0.0
        hold_command = 1.0
        fault_command = 0.0
    } else if (pressure > 600.0 || flow_rate > 100.0) {
        open_command = 0.0
        close_command = 0.0
        hold_command = 0.0
        fault_command = 1.0
    } else if (pressure < 400.0) {
        open_command = 1.0
        close_command = 0.0
        hold_command = 0.0
        fault_command = 0.0
    } else if (pressure > 550.0) {
        open_command = 0.0
        close_command = 1.0
        hold_command = 0.0
        fault_command = 0.0
    } else {
        open_command = 0.0
        close_command = 0.0
        hold_command = 1.0
        fault_command = 0.0
    }
}

// Named outputs make this readable
pressure_sensor -> valve_controller{};
flow_sensor -> valve_controller{};
override_open_button -> valve_controller{};
override_close_button -> valve_controller{};
auto_mode_switch -> valve_controller{};

valve_controller{} -> {;
    open_command -> valve_actuator{direction: open},;
    close_command -> valve_actuator{direction: close},;
    hold_command -> valve_actuator{direction: hold},;
    fault_command -> [;
        emergency_shutdown{},
        alarm_panel{},
        fault_logger{}
    ]
}

// ============================================================================
// VERDICT: Named Outputs Win for Conditional Routing
// ============================================================================

// Why named outputs are best:
// 1. Semantic clarity - output names match conditions
// 2. Grouped routing - all paths under one router
// 3. Visual structure - easy to see decision tree
// 4. Natural for multi-output stages
// 5. Works well with brackets for sub-fan-out

// Multiple statements work but lose structure:
// - OK for simple cases
// - Poor for complex conditionals
// - Hard to maintain

// Tee is wrong tool:
// - Broadcasts to all, doesn't decide
// - Would need filter stages (verbose)

// Recommendation: Prioritize named output routing syntax
// It's not just about fan-out - it's about expressing CONDITIONAL flow
