// Comprehensive Routing Example
// Rocket engine test stand with multiple routing patterns

// ============================================================================
// Stage Definitions
// ============================================================================

stage pid_controller{
    kp f64
    ki f64
    kd f64
} (setpoint f32, measured f32) f32 {
    integral f32 $= 0.0

    error := setpoint - measured
    integral = integral + error

    output := (error * f32(kp)) + (integral * f32(ki))
    return output
}

stage safety_monitor{
    low_limit f32
    high_limit f32
} (value f32) {
    status string
    severity u8
} {
    if (value < low_limit) {
        status = "BELOW_LIMIT"
        severity = 2
    } else if (value > high_limit) {
        status = "ABOVE_LIMIT"
        severity = 3
    } else {
        status = "NORMAL"
        severity = 0
    }
}

// ============================================================================
// Pattern 1: Multiple Statements for Simple Clarity
// ============================================================================

// Each critical sensor feeds multiple systems independently
ox_tank_pressure -> main_controller{};
ox_tank_pressure -> backup_controller{};
ox_tank_pressure -> data_logger{};
ox_tank_pressure -> realtime_display{};

fuel_tank_pressure -> main_controller{};
fuel_tank_pressure -> backup_controller{};
fuel_tank_pressure -> data_logger{};
fuel_tank_pressure -> realtime_display{};

// ============================================================================
// Pattern 2: Bracket Syntax for Grouped Fan-out (if implemented)
// ============================================================================

// Pre-processed sensor feeds multiple consumers as a group
chamber_pressure -> filter{cutoff: 1000} -> [;
    pid_controller{kp: 1.5, ki: 0.1, kd: 0.05},
    statistical_analyzer{},
    high_speed_logger{rate: 10khz}
]

// ============================================================================
// Pattern 3: Named Output Routing for Multi-output Stages
// ============================================================================

// Safety monitor produces multiple outputs routed differently
ox_line_pressure -> safety_monitor{;
    low_limit: 50.0,
    high_limit: 500.0
} -> {;
    status -> operator_display{},;
    severity -> [;
        alarm_system{},
        abort_controller{}
    ]
}

// Alternative binding syntax for complex routing
fuel_line_pressure -> safety_monitor{;
    low_limit: 45.0,
    high_limit: 480.0
} as fuel_safety

fuel_safety.status -> operator_display{};
fuel_safety.status -> data_archive{};
fuel_safety.severity -> alarm_system{};
fuel_safety.severity -> abort_controller{};

// ============================================================================
// Pattern 4: Explicit Tee for Broadcast Semantics
// ============================================================================

// System heartbeat broadcasts to all subsystems
interval{period: 1s} -> tee{;
    ox_subsystem{},
    fuel_subsystem{},
    ignition_subsystem{},
    data_acquisition{}
}

// ============================================================================
// Mixed Patterns: Real-world Complexity
// ============================================================================

// Start signal triggers parallel sequences
start_command -> [;
    ox_fill_sequence{},
    fuel_fill_sequence{},
    power_up_sequence{}
]

// Convergence from multiple sources
ox_fill_sequence{} -> all{ox_fill_sequence{}, fuel_fill_sequence{}, power_up_sequence{}} -> ignition_enable{};
fuel_fill_sequence{} -> all{ox_fill_sequence{}, fuel_fill_sequence{}, power_up_sequence{}};
power_up_sequence{} -> all{ox_fill_sequence{}, fuel_fill_sequence{}, power_up_sequence{}};

// Complex multi-stage processing
chamber_temp -> preprocessor{} -> safety_monitor{;
    low_limit: -20.0,
    high_limit: 3000.0
} as chamber_safety

// Critical path: Immediate response
chamber_safety.severity -> emergency_shutdown{};

// Monitoring paths: Multiple consumers
chamber_safety.status -> tee{;
    operator_hmi{},
    mission_control_telemetry{},
    local_data_logger{}
}

// Analytical path: Post-processing
chamber_safety.severity -> statistical_analyzer{} -> trend_predictor{};

// ============================================================================
// Abort Logic: Fan-in from Multiple Conditions
// ============================================================================

// Multiple abort conditions
ox_overpressure -> abort_or{};
fuel_overpressure -> abort_or{};
chamber_overtemp -> abort_or{};
loss_of_comms -> abort_or{};

// Abort triggers shutdown sequence
abort_or{} -> shutdown_sequence{} -> [;
    ox_valve_close{},
    fuel_valve_close{},
    ignition_cutoff{},
    purge_activate{}
]

// ============================================================================
// Pattern Comparison in Same System
// ============================================================================

// Approach A: Multiple statements (explicit, verbose)
thrust_sensor -> controller_a{};
thrust_sensor -> controller_b{};
thrust_sensor -> logger_a{};

// Approach B: Brackets (concise, grouped)
thrust_sensor -> [;
    controller_c{},
    controller_d{},
    logger_b{}
]

// Approach C: Explicit tee (semantic, visible node)
thrust_sensor -> tee{;
    controller_e{},
    controller_f{},
    logger_c{}
}

// All three should produce identical runtime behavior
// Choice depends on developer preference and tooling
