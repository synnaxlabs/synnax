authority 200

start_sim_cmd => main

sequence main {
    stage open_ox_mpv {
        1 -> ox_mpv_cmd,
        wait{duration=500ms} => next,
    }
    stage open_fuel_mpv {
        1 -> fuel_mpv_cmd,
    }
}