start_seq_cmd => main

sequence main {
    stage press {
        1 -> press_vlv_cmd,
        0 -> vent_vlv_cmd,
        press_pt > 30 => maintain,
        hold_cmd => press_hold,
        abort_cmd => abort
    }
    stage maintain {
        0 -> vent_vlv_cmd,
        0 -> press_vlv_cmd,
        wait{duration=2s} => vent,
        abort_cmd => abort
    }
    stage vent {
        0 -> press_vlv_cmd,
        1 -> vent_vlv_cmd,
        press_pt < 5 => press,
        abort_cmd => abort
    }
    stage press_hold {
        0 -> press_vlv_cmd,
        0 -> vent_vlv_cmd,
        resume_cmd => press,
        abort_cmd => abort
    }
    stage abort {
        0 -> press_vlv_cmd,
        1 -> vent_vlv_cmd
    }
}