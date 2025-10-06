// Approach 3: Named Output Routing
// For stages with multiple return values

// Stage with multiple named outputs
stage valve_controller{
    setpoint f64
} (pressure f32, enable u8) {
    command f32
    status string
    diagnostics f32
} {
    if (!enable) {
        command = 0.0
        status = "DISABLED"
        diagnostics = 0.0
        return
    }

    error := setpoint - f64(pressure)
    command = f32(error * 1.5)

    if (pressure > setpoint) {
        status = "ABOVE_TARGET"
    } else {
        status = "BELOW_TARGET"
    }

    diagnostics = pressure / f32(setpoint)
}

// Named output routing with table syntax
sensor -> valve_controller{setpoint: 100.0} -> {;
    command -> actuator{},;
    status -> display{},;
    diagnostics -> logger{};
}

// Alternative: Binding syntax
sensor -> valve_controller{setpoint: 100.0} as vc;

vc.command -> actuator{};
vc.status -> display{};
vc.diagnostics -> logger{};
vc.diagnostics -> trend_plot{};

// Stage with conditional outputs
stage splitter{
    threshold f64
} (input f32) {
    high f32
    low f32
} {
    if (input > f32(threshold)) {
        high = input
        low = 0.0
    } else {
        high = 0.0
        low = input
    }
}

// Routing different outputs to different targets
ox_pressure -> splitter{threshold: 500.0} -> {;
    high -> [alarm{}, emergency_shutdown{}],;
    low -> [logger{}, normal_display{}];
}

// Multi-stage pipeline with named routing
stage signal_analyzer{} (input f32) {
    mean f32
    peak f32
    rms f32
} {
    // ... analysis logic ...
}

sensor -> signal_analyzer{} -> {;
    mean -> baseline_tracker{},;
    peak -> alarm_checker{threshold: 100.0},;
    rms -> power_monitor{};
}

// Combining named outputs with fan-out
sensor -> signal_analyzer{} as sa;

sa.mean -> trend_logger{};
sa.mean -> display_panel{};

sa.peak -> alarm_high{};
sa.peak -> data_archive{};

sa.rms -> power_calculation{};

// This maps cleanly to IR Handles:
// {
//   "source": {"node": "signal_analyzer_1", "param": "mean"},
//   "target": {"node": "trend_logger", "param": "input"}
// }

// Pros:
// - Handles multi-output stages elegantly
// - Clear which data flows where
// - Maps directly to graph Handle concept
// - Enables selective routing

// Cons:
// - More complex syntax
// - Requires output name declarations
// - Binding syntax adds language complexity
