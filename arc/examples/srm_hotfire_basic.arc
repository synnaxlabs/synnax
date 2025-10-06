// Basic Solid Rocket Motor (SRM) Hotfire Test
// Demonstrates control flow for a simple static fire test

// =============================================================================
// Safety Monitoring - Check if pressure is within safe limits
// =============================================================================
stage pressure_safety_check {
    max_pressure f64
    min_pressure f64
} (pressure f64) {
    safe f64
    high_abort f64
    low_abort f64
} {
    if (pressure > max_pressure) {
        high_abort = pressure
    } else if (pressure < min_pressure) {
        low_abort = pressure
    } else {
        safe = pressure
    }
}

// =============================================================================
// Temperature Safety - Monitor thermocouple readings
// =============================================================================
stage temperature_safety_check {
    max_temp f64
} (temperature f64) {
    safe f64
    abort f64
} {
    if (temperature > max_temp) {
        abort = temperature
    } else {
        safe = temperature
    }
}

// =============================================================================
// Thrust Measurement - Convert load cell reading to thrust
// =============================================================================
stage thrust_calculator {} (load_cell_volts f64) f64 {
    // Simple calibration=1000 lbf per volt
    return load_cell_volts * 1000.0
}

// =============================================================================
// Data Logging Stages
// =============================================================================
stage safe_data_logger {} (value f64) {
    // Log normal operational data
}

stage abort_logger {} (value f64) {
    // Log abort conditions for post-test analysis
}

stage alarm_trigger {} (value f64) {
    // Trigger visual/audio alarms
}

// =============================================================================
// Test Sequence Stages
// =============================================================================
stage igniter_controller {} () {
    // Control igniter firing sequence
}

stage valve_controller {} (command f64) {
    // Control propellant valves (if applicable)
}

// =============================================================================
// Flow Statements - Wire up the test system
// =============================================================================

// Pressure monitoring with safety checks
chamber_pressure -> pressure_safety_check{;
    max_pressure=1000.0,  // psi
    min_pressure=0.0
} -> {;
    safe -> safe_data_logger{},;
    high_abort -> abort_logger{} -> alarm_trigger{},;
    low_abort -> abort_logger{} -> alarm_trigger{};
}

// Temperature monitoring
nozzle_temperature -> temperature_safety_check{;
    max_temp=3500.0  // °F
} -> {;
    safe -> safe_data_logger{},;
    abort -> abort_logger{} -> alarm_trigger{};
}

// Thrust measurement and logging
load_cell -> thrust_calculator{} -> safe_data_logger{};