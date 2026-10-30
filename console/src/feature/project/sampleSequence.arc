import control
import time

// The sequence outranks the schematic, so a Start click does not take the valves. An
// operator takes a valve with its control chip.
authority 100

sequence press {
    stage idle {
        0 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
        sample_start_cmd == 1 => pressurize
    }
    stage pressurize {
        control.set_authority(value=100)
        1 -> sample_press_vlv_cmd
        0 -> sample_vent_vlv_cmd
        sample_tank_pt > 450 => abort
        sample_abort_cmd == 1 => abort
        sample_tank_pt > 300 => hold
    }
    stage hold {
        0 -> sample_press_vlv_cmd
        sample_abort_cmd == 1 => abort
        time.wait(duration=10s) => vent
    }
    stage vent {
        1 -> sample_vent_vlv_cmd
        sample_tank_pt < 5 => idle
    }
    // Abort raises the sequence to the authority of a control chip, so its commands
    // reach a valve an operator holds.
    stage abort {
        control.set_authority(value=255)
        0 -> sample_press_vlv_cmd
        1 -> sample_vent_vlv_cmd
        sample_start_cmd == 1 => pressurize
    }
}

true => press
