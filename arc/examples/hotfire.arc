// Entry point
start_cmd => main

func ox_press_control() {
    valve_cmd_t = 1
    if (pressure > 30) {

    }
}

func prechecks() {
    if not verify_connections() { return false }
    sy.sleep(1)
    if not verify_sensors() { return false }
    sy.sleep(1)
    return true
}

// Main sequence - happy path stages first, then holds
sequence main {
    stage precheck {
        prechecks{} => match {
        }
        status_checks{} => match {
            true => abort
        }
        abort_btn => abort,
        hold_btn => precheck_hold
    }

    stage status_checks {}

    stage pressurization {
        ox_press_pt -> ox_press_control{target=ox_target_psi},
        ox_press_pt >= target_pressure => 0 -> ox_press_valve_cmd,
        interval{50ms} -> fuel_press_control{target=fuel_target_psi},
        copv_temp > 350 => abort,
        ox_tank_psi > ox_max_psi => abort,
        fuel_tank_psi > fuel_max_psi => abort,
        hold_button => pressurization_hold,
        ox_tank_psi > ox_target_psi and fuel_tank_psi > fuel_target_psi => next,
        interval{30s} -> send_slack_message{},
        wait{30s} => abort
    }

    stage igniter {
        // Reactive flows
        1 -> igniter_cmd,

        // Safety aborts
        chamber_pressure > abort_pressure => abort,

        // Nominal completion
        flame_detected => next,

        // Timeout
        wait{2s} => abort
    }

    stage main_engine_start (5s => abort) {
        // wait{5s} => abort,
        wait{2s} => hold,
        chamber_pressure > abort_pressure => abort,
        interval{20ms} => run_some_check{} => abort
        // Immediate and delayed commands
        1 -> ox_valve_cmd,
        wait{10ms} => 1 -> fuel_valve_cmd,
        interval{10ms} -> ox_valve_ramp{rate=10},
        interval{10ms} -> fuel_valve_ramp{rate=10},
        // Nominal completion
        ox_valve_state == 1 and fuel_valve_state == 1 and chamber_pressure > min_chamber => next,
    }

    stage steady_state {
        // Reactive flows
        interval{10ms} -> ox_valve_control{target=100},
        interval{10ms} -> fuel_valve_control{target=100},

        // Safety aborts
        chamber_pressure > abort_pressure => abort,
        chamber_pressure < min_chamber => abort,
        ox_inlet_temp > max_ox_temp => abort,

        // Nominal completion
        wait{burn_duration} => next
    }

    stage shutdown {
        // Reactive flows
        0 -> ox_valve_cmd,
        0 -> fuel_valve_cmd,
        0 -> igniter_cmd,

        // Nominal completion
        ox_valve_state == 0 and fuel_valve_state == 0 and chamber_pressure < 50 => next,

        // Timeout
        wait{10s} => abort
    }

    stage safing {
        1 -> ox_vent_cmd,
        1 -> fuel_vent_cmd,

        // Nominal completion
        ox_tank_psi < 20 and fuel_tank_psi < 20 => next,

        // Timeout - vent should complete within 60s
        wait{60s} => abort
    }

    stage complete {
        log{"Test complete - nominal"}
    }

    // ----- Hold stages (exception handlers) -----

    stage precheck_hold {
        resume_btn => precheck,
        abort_btn => abort
    }

    stage pressurization_hold {
        // Reactive flows
        interval{100ms} -> ox_press_maintain{},
        interval{100ms} -> fuel_press_maintain{},

        // Safety aborts
        ox_tank_psi < ox_min_hold_psi => abort,
        fuel_tank_psi < fuel_min_hold_psi => abort,

        // Resume
        resume_btn => pressurization,
        abort_btn => abort
    }
}

// Abort sequence
sequence abort {
    stage close_valves {
        0 -> ox_valve_cmd,
        0 -> fuel_valve_cmd,
        0 -> igniter_cmd,

        // Check actual valve STATE, not the command we just sent
        ox_valve_state == 0 and fuel_valve_state == 0 and igniter_state == 0 => next,
        wait{2s} => next
    }

    stage vent {
        1 -> ox_vent_cmd,
        1 -> fuel_vent_cmd,
        1 -> emergency_vent_cmd,
        interval{100ms} -> abort_safing_log{},

        ox_tank_psi < 50 and fuel_tank_psi < 50 => next,
        wait{60s} => next
    }

    stage safed {
        log{"ABORT COMPLETE - SYSTEM SAFED"}
    }
}
