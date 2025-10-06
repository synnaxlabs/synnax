// SRM Hotfire Test Sequence
// Focuses on the actual test execution sequence

// =============================================================================
// Test Sequencer - Main state machine
// =============================================================================
stage test_sequencer {
    ignition_delay_ms i64
    burn_duration_ms i64
} (trigger f64) {
    idle f64
    armed f64
    t_minus_10 f64
    ignition f64
    burning f64
    shutdown f64
} {
    // State transitions based on timing and conditions
    // This will get more sophisticated with actual timing logic
    if (trigger == 0.0) {
        idle = trigger
    } else if (trigger == 1.0) {
        armed = trigger
    } else if (trigger == 2.0) {
        t_minus_10 = trigger
    } else if (trigger == 3.0) {
        ignition = trigger
    } else if (trigger == 4.0) {
        burning = trigger
    } else {
        shutdown = trigger
    }
}

// =============================================================================
// Ignition Sequence - Multi-step pyro firing
// =============================================================================
stage ignition_sequencer {} (fire_signal f64) {
    prime_igniters f64
    fire_primary f64
    fire_secondary f64
} {
    // T-1.0s: Prime igniters
    // T-0.5s: Fire primary igniter
    // T-0.0s: Fire secondary igniter
    if (fire_signal < 0.5) {
        prime_igniters = fire_signal
    } else if (fire_signal < 1.0) {
        fire_primary = fire_signal
    } else {
        fire_secondary = fire_signal
    }
}

// =============================================================================
// Valve Sequencing - Pressurant/purge system
// =============================================================================
stage valve_sequencer {} (sequence_state f64) {
    open_pressurant f64
    open_purge f64
    close_all f64
} {
    if (sequence_state == 1.0) {
        open_pressurant = sequence_state
    } else if (sequence_state == 2.0) {
        open_purge = sequence_state
    } else {
        close_all = sequence_state
    }
}

// =============================================================================
// Actuator Control Stages
// =============================================================================
stage pyro_controller {} (fire_command f64) {
    // Send voltage pulse to pyro channels
}

stage pressurant_valve {} (command f64) {
    // Open/close pressurant tank valve
}

stage purge_valve {} (command f64) {
    // Open/close nitrogen purge valve
}

stage data_acquisition {} (value f64) {
    // High-speed DAQ for test data
}

stage abort_sequence {} (abort_signal f64) {
    // Emergency shutdown procedures
}

// =============================================================================
// Test Sequence Flow
// =============================================================================

// Main test state machine
test_control_signal -> test_sequencer{
    ignition_delay_ms: 10000,
    burn_duration_ms: 5000
} -> {
    idle -> data_acquisition{},
    armed -> valve_sequencer{} -> {
        open_pressurant -> pressurant_valve{},
        open_purge -> purge_valve{},
        close_all -> pressurant_valve{} -> purge_valve{}
    },
    t_minus_10 -> data_acquisition{},
    ignition -> ignition_sequencer{} -> {
        prime_igniters -> pyro_controller{},
        fire_primary -> pyro_controller{},
        fire_secondary -> pyro_controller{}
    },
    burning -> data_acquisition{},
    shutdown -> valve_sequencer{} -> {
        close_all -> pressurant_valve{} -> purge_valve{}
    }
}

// Abort handling
abort_button -> abort_sequence{} -> valve_sequencer{} -> {
    close_all -> pressurant_valve{} -> purge_valve{}
}
