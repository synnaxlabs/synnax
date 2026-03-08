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

// Segment A: 3000 RPM
SEGMENT_A_TRANSIENT_LOAD_SP f32 := 1200
SEGMENT_A_TRANSIENT_THRESHOLD f32 := 1140
SEGMENT_A_BASE_LOAD_SP f32 := 1000
SEGMENT_A_BASE_THRESHOLD f32 := 1050
SEGMENT_A_CYCLES u32 := 100
SEGMENT_A_DRIVE_SP f32 := 3000
SEGMENT_A_DRIVE_THRESHOLD f32 := 2850

// Segment B: 2500 RPM
SEGMENT_B_TRANSIENT_LOAD_SP f32 := 1000
SEGMENT_B_TRANSIENT_THRESHOLD f32 := 950
SEGMENT_B_BASE_LOAD_SP f32 := 800
SEGMENT_B_BASE_THRESHOLD f32 := 850
SEGMENT_B_CYCLES u32 := 100
SEGMENT_B_DRIVE_SP f32 := 2500
SEGMENT_B_DRIVE_THRESHOLD f32 := 2550

// Segment C: 2000 RPM
SEGMENT_C_TRANSIENT_LOAD_SP f32 := 900
SEGMENT_C_TRANSIENT_THRESHOLD f32 := 855
SEGMENT_C_BASE_LOAD_SP f32 := 700
SEGMENT_C_BASE_THRESHOLD f32 := 750
SEGMENT_C_CYCLES u32 := 100
SEGMENT_C_DRIVE_SP f32 := 2000
SEGMENT_C_DRIVE_THRESHOLD f32 := 2050

// Segment D: 1500 RPM
SEGMENT_D_TRANSIENT_LOAD_SP f32 := 700
SEGMENT_D_TRANSIENT_THRESHOLD f32 := 665
SEGMENT_D_BASE_LOAD_SP f32 := 500
SEGMENT_D_BASE_THRESHOLD f32 := 550
SEGMENT_D_CYCLES u32 := 100
SEGMENT_D_DRIVE_SP f32 := 1500
SEGMENT_D_DRIVE_THRESHOLD f32 := 1550

// Segment E: 1000 RPM
SEGMENT_E_TRANSIENT_LOAD_SP f32 := 500
SEGMENT_E_TRANSIENT_THRESHOLD f32 := 475
SEGMENT_E_BASE_LOAD_SP f32 := 300
SEGMENT_E_BASE_THRESHOLD f32 := 350
SEGMENT_E_CYCLES u32 := 100
SEGMENT_E_DRIVE_SP f32 := 1000
SEGMENT_E_DRIVE_THRESHOLD f32 := 1050

// Timing
DRIVE_RAMP_TIMEOUT i64 ns := 60s
LOAD_RAMP_TIMEOUT i64 ns := 30s
TRANSIENT_HOLD i64 ns := 30s
BASE_HOLD i64 ns := 60s

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

    // ---- Segment A: 3000 RPM ----

    stage segment_a_drive_speed {
        2 -> endurance_test_state,
        0 -> endurance_test_segment_cycle,
        SEGMENT_A_DRIVE_SP => endurance_desired_speed,
        SEGMENT_A_DRIVE_SP => drive_speed_sp,
        drive_speed_fb > SEGMENT_A_DRIVE_THRESHOLD => next,
        wait{duration=DRIVE_RAMP_TIMEOUT} => idle,
    }

    stage segment_a_load_trans {
        3 -> endurance_test_state,
        increment_cycle{ch=endurance_test_segment_cycle},
        f32(SEGMENT_A_TRANSIENT_LOAD_SP) => endurance_desired_load,
        SEGMENT_A_TRANSIENT_LOAD_SP => dc_lb_load_sp,
        Load_Current > SEGMENT_A_TRANSIENT_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_a_bad_cycle,
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
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_a_bad_cycle,
    }

    stage segment_a_base_wait {
        6 -> endurance_test_state,
        increment_good_cycle{},
        wait{duration=BASE_HOLD} => next
    }

    stage segment_a_bad_cycle {
        increment_bad_cycle{},
        1 -> next
    }

    stage segment_a_pass_fail {
        7 -> endurance_test_state,
        wait{duration=50ms} => endurance_test_segment_cycle >= SEGMENT_A_CYCLES => segment_b_drive_speed,
        wait{duration=50ms} => endurance_test_segment_cycle < SEGMENT_A_CYCLES => segment_a_load_trans,
    }

    // ---- Segment B: 2500 RPM ----

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
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_b_bad_cycle,
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
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_b_bad_cycle,
    }

    stage segment_b_base_wait {
        12 -> endurance_test_state,
        increment_good_cycle{},
        wait{duration=BASE_HOLD} => next
    }

    stage segment_b_bad_cycle {
        increment_bad_cycle{},
        1 -> next
    }

    stage segment_b_pass_fail {
        13 -> endurance_test_state,
        wait{duration=50ms} => endurance_test_segment_cycle >= SEGMENT_B_CYCLES => segment_c_drive_speed,
        wait{duration=50ms} => endurance_test_segment_cycle < SEGMENT_B_CYCLES => segment_b_load_trans,
    }

    // ---- Segment C: 2000 RPM ----

    stage segment_c_drive_speed {
        14 -> endurance_test_state,
        0 -> endurance_test_segment_cycle,
        SEGMENT_C_DRIVE_SP => endurance_desired_speed,
        SEGMENT_C_DRIVE_SP => drive_speed_sp,
        drive_speed_fb < SEGMENT_C_DRIVE_THRESHOLD => next,
        wait{duration=DRIVE_RAMP_TIMEOUT} => idle,
    }

    stage segment_c_load_trans {
        15 -> endurance_test_state,
        increment_cycle{ch=endurance_test_segment_cycle},
        f32(SEGMENT_C_TRANSIENT_LOAD_SP) => endurance_desired_load,
        SEGMENT_C_TRANSIENT_LOAD_SP => dc_lb_load_sp,
        Load_Current > SEGMENT_C_TRANSIENT_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_c_bad_cycle,
    }

    stage segment_c_transient_wait {
        16 -> endurance_test_state,
        wait{duration=TRANSIENT_HOLD} => next
    }

    stage segment_c_load_base {
        17 -> endurance_test_state,
        SEGMENT_C_BASE_LOAD_SP -> endurance_desired_load,
        SEGMENT_C_BASE_LOAD_SP -> dc_lb_load_sp,
        Load_Current < SEGMENT_C_BASE_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_c_bad_cycle,
    }

    stage segment_c_base_wait {
        18 -> endurance_test_state,
        increment_good_cycle{},
        wait{duration=BASE_HOLD} => next
    }

    stage segment_c_bad_cycle {
        increment_bad_cycle{},
        1 -> next
    }

    stage segment_c_pass_fail {
        19 -> endurance_test_state,
        wait{duration=50ms} => endurance_test_segment_cycle >= SEGMENT_C_CYCLES => segment_d_drive_speed,
        wait{duration=50ms} => endurance_test_segment_cycle < SEGMENT_C_CYCLES => segment_c_load_trans,
    }

    // ---- Segment D: 1500 RPM ----

    stage segment_d_drive_speed {
        20 -> endurance_test_state,
        0 -> endurance_test_segment_cycle,
        SEGMENT_D_DRIVE_SP => endurance_desired_speed,
        SEGMENT_D_DRIVE_SP => drive_speed_sp,
        drive_speed_fb < SEGMENT_D_DRIVE_THRESHOLD => next,
        wait{duration=DRIVE_RAMP_TIMEOUT} => idle,
    }

    stage segment_d_load_trans {
        21 -> endurance_test_state,
        increment_cycle{ch=endurance_test_segment_cycle},
        f32(SEGMENT_D_TRANSIENT_LOAD_SP) => endurance_desired_load,
        SEGMENT_D_TRANSIENT_LOAD_SP => dc_lb_load_sp,
        Load_Current > SEGMENT_D_TRANSIENT_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_d_bad_cycle,
    }

    stage segment_d_transient_wait {
        22 -> endurance_test_state,
        wait{duration=TRANSIENT_HOLD} => next
    }

    stage segment_d_load_base {
        23 -> endurance_test_state,
        SEGMENT_D_BASE_LOAD_SP -> endurance_desired_load,
        SEGMENT_D_BASE_LOAD_SP -> dc_lb_load_sp,
        Load_Current < SEGMENT_D_BASE_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_d_bad_cycle,
    }

    stage segment_d_base_wait {
        24 -> endurance_test_state,
        increment_good_cycle{},
        wait{duration=BASE_HOLD} => next
    }

    stage segment_d_bad_cycle {
        increment_bad_cycle{},
        1 -> next
    }

    stage segment_d_pass_fail {
        25 -> endurance_test_state,
        wait{duration=50ms} => endurance_test_segment_cycle >= SEGMENT_D_CYCLES => segment_e_drive_speed,
        wait{duration=50ms} => endurance_test_segment_cycle < SEGMENT_D_CYCLES => segment_d_load_trans,
    }

    // ---- Segment E: 1000 RPM ----

    stage segment_e_drive_speed {
        26 -> endurance_test_state,
        0 -> endurance_test_segment_cycle,
        SEGMENT_E_DRIVE_SP => endurance_desired_speed,
        SEGMENT_E_DRIVE_SP => drive_speed_sp,
        drive_speed_fb < SEGMENT_E_DRIVE_THRESHOLD => next,
        wait{duration=DRIVE_RAMP_TIMEOUT} => idle,
    }

    stage segment_e_load_trans {
        27 -> endurance_test_state,
        increment_cycle{ch=endurance_test_segment_cycle},
        f32(SEGMENT_E_TRANSIENT_LOAD_SP) => endurance_desired_load,
        SEGMENT_E_TRANSIENT_LOAD_SP => dc_lb_load_sp,
        Load_Current > SEGMENT_E_TRANSIENT_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_e_bad_cycle,
    }

    stage segment_e_transient_wait {
        28 -> endurance_test_state,
        wait{duration=TRANSIENT_HOLD} => next
    }

    stage segment_e_load_base {
        29 -> endurance_test_state,
        SEGMENT_E_BASE_LOAD_SP -> endurance_desired_load,
        SEGMENT_E_BASE_LOAD_SP -> dc_lb_load_sp,
        Load_Current < SEGMENT_E_BASE_THRESHOLD => next,
        wait{duration=LOAD_RAMP_TIMEOUT} => segment_e_bad_cycle,
    }

    stage segment_e_base_wait {
        30 -> endurance_test_state,
        increment_good_cycle{},
        wait{duration=BASE_HOLD} => next
    }

    stage segment_e_bad_cycle {
        increment_bad_cycle{},
        1 -> next
    }

    stage segment_e_pass_fail {
        31 -> endurance_test_state,
        wait{duration=50ms} => endurance_test_segment_cycle >= SEGMENT_E_CYCLES => idle,
        wait{duration=50ms} => endurance_test_segment_cycle < SEGMENT_E_CYCLES => segment_e_load_trans,
    }

    // ---- Idle ----

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
