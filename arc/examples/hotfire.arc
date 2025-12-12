// Main sequence with primary path
sequence main {
    start_cmd => precheck: stage {
        {
            if !verify_connections() { return connection_fail }
            if !verify_sensors() { return sensor_fail }
            return ok
        } => match {
            ok => next,
            connection_fail => abort,
            sensor_fail => abort
        },
        abort_btn => abort,
        hold_btn => precheck_hold
    } => pressurization: stage {
        interval{100ms} -> ox_press_control{target=ox_target_psi}
        interval{100ms} -> fuel_press_control{target=fuel_target_psi}
        hold_button => pressurization_hold
        copv_temp > 350 => abort
        ox_tank_psi > ox_max_psi => abort
        fuel_tank_psi > fuel_max_psi => abort
        wait{30s} => abort
        ox_tank_psi > ox_target_psi && fuel_tank_psi > fuel_target_psi => next
    } => igniter: stage {
        1 -> igniter_cmd
        wait{2s} => abort
        flame_detected => next
        chamber_pressure > abort_pressure => abort
    } => main_engine_start: stage {
        1 -> ox_valve_cmd
        wait{10ms} -> 1 -> fuel_valve_cmd
        interval{10ms} -> ox_valve_ramp{rate=10}
        interval{10ms} -> fuel_valve_ramp{rate=10}
        chamber_pressure > abort_pressure => abort
        wait{5s} => abort
        ox_valve_state == 1 && fuel_valve_state == 1 && chamber_pressure > min_chamber => next
    } => steady_state: stage {
        interval{10ms} -> ox_valve_control{target=100}
        interval{10ms} -> fuel_valve_control{target=100}
        chamber_pressure > abort_pressure => abort
        chamber_pressure < min_chamber => abort
        ox_inlet_temp > max_ox_temp => abort
        wait{burn_duration} => next
    } => shutdown: stage {
        0 -> ox_valve_cmd, fuel_valve_cmd, igniter_cmd
        wait{10s} => abort
        ox_valve_state == 0 && fuel_valve_state == 0 && chamber_pressure < 50 => next
    } => safing: stage {
        1 -> ox_vent_cmd, fuel_vent_cmd
        ox_tank_psi < 20 && fuel_tank_psi < 20 => next
    } => complete: stage {
        log{"Test complete - nominal"}
    }
}

// Hold stages (standalone definitions)
stage precheck_hold {
    resume_btn => precheck
    abort_btn => abort
}

stage pressurization_hold {
    interval{100ms} -> ox_press_maintain{}
    interval{100ms} -> fuel_press_maintain{}
    resume_btn => pressurization
    abort_btn => abort
    ox_tank_psi < ox_min_hold_psi => abort
    fuel_tank_psi < fuel_min_hold_psi => abort
}

// Abort sequence
sequence abort {
    stage {
        0 -> ox_valve_cmd, fuel_valve_cmd, igniter_cmd
        wait{2s} => next
        ox_valve_cmd == 0 && fuel_valve_cmd == 0 && igniter_cmd == 0 => next
    } => abort_safing: stage {
        1 -> ox_vent_cmd, fuel_vent_cmd, emergency_vent_cmd
        interval{100ms} -> abort_safing_log{}
        ox_tank_psi < 50 && fuel_tank_psi < 50 => next
        wait{60s} => next
    } => abort_complete: stage {
        log{"ABORT COMPLETE - SYSTEM SAFED"}
    }
}
