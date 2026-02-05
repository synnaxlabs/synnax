start_seq_cmd => main

func control_tpc() {
    if ox_pt_1 > 25 {
        if ox_press_state {
            ox_press_cmd = 0
        }
    } else if ox_pt_1 < 20 {
        if not ox_press_state {
            ox_press_cmd = 1
        }
    }
}


sequence main {
    stage initialize {
        0 -> ox_press_cmd,
        0 -> ox_mpv_cmd,
        0 -> gas_booster_iso_cmd,
        1 -> ox_vent_cmd,
        ox_vent_state == 1 => next,
    }
    stage press {
        1 -> press_iso_cmd,
        1 -> ox_press_cmd,
        1 -> gas_booster_iso_cmd,
        ox_pt_1 > 50 => next,
    } 
    stage press_high {
        0 -> ox_press_cmd,
        0 -> press_iso_cmd,
        1 -> gas_booster_iso_cmd,
        press_pt_1 > 150 => next,
    } 
    stage hold {
        0 -> gas_booster_iso_cmd,
        wait{duration=2s} => next,
    }
    stage tpc {
        1 -> ox_mpv_cmd,
        1 -> press_iso_cmd,
        interval{period=100ms} -> control_tpc{},
        ox_pt_1 < 15 => next,
    }
    stage safe {
        0 -> ox_press_cmd,
        0 -> ox_mpv_cmd,
        0 -> gas_booster_iso_cmd,
        1 -> ox_vent_cmd,
    }
}