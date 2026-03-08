endurance_test_start => main

func increment_cycle{ch chan u32}() {
    ch = ch + 1
}

func increment_bad_cycle{}() {
    cycle_count_endurance_bad = cycle_count_endurance_bad + 1
}

func increment_good_cycle{}() {
    cycle_count_endurance_good = cycle_count_endurance_good + 1
}

SEGMENT_A_TRANSIENT_LOAD_SP f32 := 1200
SEGMENT_A_TRANSIENT_THRESHOLD f32 := 1140
SEGMENT_A_BASE_LOAD_SP f32 := 1000
SEGMENT_A_BASE_THRESHOLD f32 := 1050
SEGMENT_A_CYCLES u32 := 10
SEGMENT_A_DRIVE_SP f32 := 3000
SEGMENT_A_DRIVE_THRESHOLD f32 := 2850

SEGMENT_B_TRANSIENT_LOAD_SP f32 := 800
SEGMENT_B_TRANSIENT_THRESHOLD f32 := 760
SEGMENT_B_BASE_LOAD_SP f32 := 600
SEGMENT_B_BASE_THRESHOLD f32 := 650
SEGMENT_B_CYCLES u32 := 10
SEGMENT_B_DRIVE_SP f32 := 2500
SEGMENT_B_DRIVE_THRESHOLD f32 := 2550

DRIVE_RAMP_TIMEOUT i64 ns := 5s
LOAD_RAMP_TIMEOUT i64 ns := 5s
TRANSIENT_HOLD i64 ns := 500ms
BASE_HOLD i64 ns := 500ms

sequence main {

    stage init {
        1 -> endurance_test_state,
        0 -> cycle_count_endurance_good,
        0 -> cycle_count_endurance_bad,
        0 -> shutdown,
        0 -> dc_lb_load_sp,
        1 -> dc_lb_enable_sp,
        1 -> batt_contactor_sp,
        1 -> dc_contactor_sp,
        1 -> gen_field_relay_sp,
        1 -> next
    }

    stage segment_a_drive_speed {
        2 -> endurance_test_state,
        0 -> cycle_count_segment_a,
        SEGMENT_A_DRIVE_SP => endurance_desired_speed,
        SEGMENT_A_DRIVE_SP => drive_speed_sp,
        drive_speed_fb > SEGMENT_A_DRIVE_THRESHOLD => next,
        wait{duration=DRIVE_RAMP_TIMEOUT} => idle,
    }

    stage segment_a_load_trans {
        3 -> endurance_test_state,
        increment_cycle{ch=cycle_count_segment_a},
        f32(SEGMENT_A_TRANSIENT_LOAD_SP) => endurance_desired_load,
        SEGMENT_A_TRANSIENT_LOAD_SP => dc_lb_load_sp,
        Load_Current > SEGMENT_A_TRANSIENT_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => idle,
    }

    stage segment_a_transient_wait {
        4 -> endurance_test_state,
        wait{duration=TRANSIENT_HOLD} => next
    }

    stage segment_a_load_base {
        5 -> endurance_test_state,
        SEGMENT_A_BASE_LOAD_SP -> endurance_desired_load,
        SEGMENT_A_BASE_LOAD_SP -> dc_lb_load_sp,
        Load_Current < SEGMENT_A_BASE_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => idle,
    }

    stage segment_a_base_wait {
        6 -> endurance_test_state,
        increment_good_cycle{},
        wait{duration=BASE_HOLD} => next
    }

    stage segment_a_pass_fail {
        7 -> endurance_test_state,
        wait{duration=50ms} => cycle_count_segment_a >= SEGMENT_A_CYCLES => segment_b_drive_speed,
        wait{duration=50ms} => cycle_count_segment_a < SEGMENT_A_CYCLES => segment_a_load_trans,
    }

    stage segment_b_drive_speed {
        8 -> endurance_test_state,
        0 -> endurance_test_segment_cycle,
        SEGMENT_B_DRIVE_SP => endurance_desired_speed,
        SEGMENT_B_DRIVE_SP => drive_speed_sp,
        drive_speed_fb < SEGMENT_B_DRIVE_THRESHOLD => next,
        wait{duration=DRIVE_RAMP_TIMEOUT} => idle,
    }

    stage segment_b_load_trans {
        9 -> endurance_test_state,
        increment_cycle{ch=endurance_test_segment_cycle},
        f32(SEGMENT_B_TRANSIENT_LOAD_SP) => endurance_desired_load,
        SEGMENT_B_TRANSIENT_LOAD_SP => dc_lb_load_sp,
        Load_Current > SEGMENT_B_TRANSIENT_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => idle,
    }

    stage segment_b_transient_wait {
        10 -> endurance_test_state,
        wait{duration=TRANSIENT_HOLD} => next
    }

    stage segment_b_load_base {
        11 -> endurance_test_state,
        SEGMENT_B_BASE_LOAD_SP -> endurance_desired_load,
        SEGMENT_B_BASE_LOAD_SP -> dc_lb_load_sp,
        Load_Current < SEGMENT_B_BASE_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => idle,
    }

    stage segment_b_base_wait {
        12 -> endurance_test_state,
        increment_good_cycle{},
        wait{duration=BASE_HOLD} => next
    }

    stage segment_b_pass_fail {
        13 -> endurance_test_state,
        wait{duration=50ms} => endurance_test_segment_cycle >= SEGMENT_B_CYCLES => idle,
        wait{duration=50ms} => endurance_test_segment_cycle < SEGMENT_B_CYCLES => segment_b_load_trans,
    }

    stage idle {
        0 -> endurance_test_state,
        0 -> drive_speed_sp,
        0 -> dc_lb_load_sp,
        0 -> dc_lb_enable_sp,
        0 -> dc_contactor_sp,
        0 -> gen_field_relay_sp,
        0 -> batt_contactor_sp,
        1 -> shutdown,
        endurance_test_start => init
    }
}
