authority 0

start_sim_cmd => main


sequence main {
    stage listen {
        ox_pt_1 > 200 and ox_pt_2 > 200 => next,
        ox_tc_1 -> stable_for{duration=1s} => next,
    }
    stage abort {
        set_authority{value=255},
        0 -> ox_mpv_cmd,
        1 -> ox_vent_cmd,
        0 -> ox_tpc_cmd,
        wait{duration=50s} => next,
    }
    stage safe {
        set_authority{value=0}
    }
}