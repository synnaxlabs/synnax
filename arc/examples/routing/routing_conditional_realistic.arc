// Realistic Conditional Routing: Rocket Engine Test Stand
// Shows how conditional routing works in practice

// ============================================================================
// Scenario: Engine state-dependent sensor routing
// ============================================================================

// Engine states
stage engine_state{} () u8 {
    // 0=safe, 1=pre_ignition, 2=ignition, 3=steady_state, 4=shutdown, 5=abort
    state u8 $= 0
    // ... state machine logic ...
    return state
}

// ============================================================================
// Pattern: Different Logging Rates Based on State
// ============================================================================

stage adaptive_logger{} (value f32, state u8) {
    high_rate_out f32
    low_rate_out f32
} {
    // During critical phases, log at high rate
    // During safe phases, log at low rate
    if (state == 2 || state == 4) {  // Ignition or shutdown
        high_rate_out = value
        low_rate_out = 0.0
    } else {
        high_rate_out = 0.0
        low_rate_out = value
    }
}

chamber_pressure -> adaptive_logger{} -> {
    high_rate_out -> logger{rate: 1khz},
    low_rate_out -> logger{rate: 10hz}
}

engine_state{} -> adaptive_logger{}

// ============================================================================
// Pattern: Safety Thresholds Change with State
// ============================================================================

stage state_aware_alarm{} (value f32, state u8) {
    safe_range_alarm f32
    startup_alarm f32
    steady_alarm f32
} {
    if (state == 0) {
        // Safe mode: tight limits
        if (value > 10.0) {
            safe_range_alarm = value
        } else {
            safe_range_alarm = 0.0
        }
        startup_alarm = 0.0
        steady_alarm = 0.0
    } else if (state == 1 || state == 2) {
        // Startup/ignition: allow higher values
        if (value > 500.0) {
            startup_alarm = value
        } else {
            startup_alarm = 0.0
        }
        safe_range_alarm = 0.0
        steady_alarm = 0.0
    } else if (state == 3) {
        // Steady state: expected high values
        if (value > 1000.0 || value < 400.0) {
            steady_alarm = value
        } else {
            steady_alarm = 0.0
        }
        safe_range_alarm = 0.0
        startup_alarm = 0.0
    }
}

ox_pressure -> state_aware_alarm{} -> {
    safe_range_alarm -> immediate_shutdown{},
    startup_alarm -> abort_sequence{},
    steady_alarm -> controlled_shutdown{}
}

engine_state{} -> state_aware_alarm{}

// ============================================================================
// Pattern: Sensor Selection Based on Operating Range
// ============================================================================

stage sensor_selector{} (
    low_range f32,
    mid_range f32,
    high_range f32,
    state u8
) f32 {
    // Choose appropriate sensor for current operating regime
    if (state == 0) {
        return low_range  // Safe mode: use sensitive sensor
    } else if (state == 1 || state == 2) {
        return mid_range  // Startup: use mid-range sensor
    } else {
        return high_range  // Running: use high-range sensor
    }
}

low_range_sensor -> sensor_selector{}
mid_range_sensor -> sensor_selector{}
high_range_sensor -> sensor_selector{}
engine_state{} -> sensor_selector{} -> controller{}

// ============================================================================
// Pattern: Abort Condition Router
// ============================================================================

stage abort_router{} (
    ox_pressure f32,
    fuel_pressure f32,
    chamber_temp f32,
    state u8
) {
    immediate_abort f32  // Hard stop
    graceful_abort f32   // Controlled shutdown
    continue_normal f32  // No action
} {
    abort_level u8 := 0  // 0=none, 1=graceful, 2=immediate

    // Critical failures: immediate abort in any state
    if (ox_pressure > 600.0 || fuel_pressure > 600.0 || chamber_temp > 3500.0) {
        abort_level = 2
    } else if (state == 3) {
        // In steady state, moderate deviations trigger graceful abort
        if (chamber_temp > 3200.0 || chamber_temp < 2800.0) {
            abort_level = 1
        }
    }

    if (abort_level == 2) {
        immediate_abort = 1.0
        graceful_abort = 0.0
        continue_normal = 0.0
    } else if (abort_level == 1) {
        immediate_abort = 0.0
        graceful_abort = 1.0
        continue_normal = 0.0
    } else {
        immediate_abort = 0.0
        graceful_abort = 0.0
        continue_normal = 1.0
    }
}

ox_line_pressure -> abort_router{}
fuel_line_pressure -> abort_router{}
chamber_temperature -> abort_router{}
engine_state{} -> abort_router{} -> {
    immediate_abort -> emergency_shutdown{},
    graceful_abort -> controlled_shutdown{},
    continue_normal -> normal_operation{}
}

// ============================================================================
// Pattern: Redundancy Management
// ============================================================================

stage redundant_selector{} (
    sensor_1 f32,
    sensor_2 f32,
    sensor_3 f32,
    sensor_1_healthy u8,
    sensor_2_healthy u8,
    sensor_3_healthy u8
) f32 {
    // Select first healthy sensor, with priority order
    if (sensor_1_healthy) {
        return sensor_1
    } else if (sensor_2_healthy) {
        return sensor_2
    } else if (sensor_3_healthy) {
        return sensor_3
    } else {
        // All failed - return zero and trigger alarm elsewhere
        return 0.0
    }
}

chamber_temp_1 -> redundant_selector{}
chamber_temp_2 -> redundant_selector{}
chamber_temp_3 -> redundant_selector{}
health_monitor{} -> sensor_1_healthy -> redundant_selector{}
health_monitor{} -> sensor_2_healthy -> redundant_selector{}
health_monitor{} -> sensor_3_healthy -> redundant_selector{}

redundant_selector{} -> temperature_controller{}

// ============================================================================
// Pattern: Time-based Routing
// ============================================================================

stage time_gated_router{} (value f32, elapsed_time timestamp) {
    warmup_out f32
    active_out f32
} {
    // First 10 seconds: warmup monitoring
    // After 10s: active control
    if (elapsed_time < 10000000000) {  // 10s in nanoseconds
        warmup_out = value
        active_out = 0.0
    } else {
        warmup_out = 0.0
        active_out = value
    }
}

sensor -> time_gated_router{} -> {
    warmup_out -> warmup_monitor{},
    active_out -> active_controller{}
}

interval{period: 100ms} -> elapsed_timer{} -> time_gated_router{}

// ============================================================================
// Pattern: Multi-Condition Gating
// ============================================================================

stage multi_gate{} (
    value f32,
    enabled u8,
    in_safe_range u8,
    system_ready u8
) f32 {
    // Only pass data if ALL conditions met
    if (enabled && in_safe_range && system_ready) {
        return value
    }
    return 0.0
}

sensor -> multi_gate{} -> critical_actuator{}

enable_switch -> multi_gate{}
range_checker{} -> multi_gate{}
startup_complete{} -> multi_gate{}

// ============================================================================
// How Routing Syntax Affects Conditional Code
// ============================================================================

// NAMED OUTPUTS = MOST NATURAL FOR CONDITIONALS
// Each output has semantic meaning matching the condition

sensor -> state_router{} -> {
    idle_out -> idle_handler{},      // Clear which path for which state
    active_out -> active_handler{},
    error_out -> error_handler{}
}

// MULTIPLE STATEMENTS = WORKS BUT LESS CLEAR
// Relationship between paths not obvious

sensor -> state_router{} -> idle -> idle_handler{}
sensor -> state_router{} -> active -> active_handler{}
sensor -> state_router{} -> error -> error_handler{}

// BRACKETS = COULD GROUP RELATED PATHS
// Useful when multiple stages consume same conditional output

sensor -> state_router{} -> {
    idle_out -> [logger{}, display{}],
    active_out -> [controller{}, alarm_check{}]
}

// EXPLICIT TEE = WRONG TOOL FOR CONDITIONALS
// Tee broadcasts to all, doesn't make decisions
// Would need separate filter stages after tee

sensor -> tee{
    filter{condition: idle} -> idle_handler{},
    filter{condition: active} -> active_handler{}
}
// This works but is awkward - the filtering should happen once, not per path
