func press() bool {
   curr_target $= ox_pt_1
   curr_target = curr_target + 50
   return ox_pt_1 > curr_target
}

func initialize() true {
    tpc_cmd = 0
    mpv_cmd = 0
    supply_cmd = 0
    vent_cmd = 1
    return true
}

sequence tpc {
    stage initialize {
        initialize{} => next,
    }
    stage press_open_valves {
        1 -> ox_press_cmd,
        press{} => press_wait
    }
    stage press_wait {
        ox_pt_1 > max_target => next,
        wait{5s} => press_open_valves
    }
    stage open_mpv {
        1 -> ox_mpv_cmd ,
        1 -> press_iso_cmd,
        interval{100ms} -> tpc,
        wait{test_duration} => next
    }
    stage safe {
        1 -> ox_tpc_cmd,
        1 -> gas_booster_iso_cmd,
        1 -> ox_vent_cmd,
        1 -> ox_mpv_cmd,
    }
}