// Control System Example
// Shows input and output routing in a practical control scenario

// PID controller with multiple inputs
stage pid_controller {
    kp f64
    ki f64
    kd f64
} (setpoint f64, measured f64, enable u8) {
    control_output f64
    error_output f64
} {
    error f64 := setpoint - measured

    // Simplified PID calculation
    proportional := error * kp
    control_output = proportional
    error_output = error
}

// Safety limiter that checks multiple conditions
stage safety_limiter {
    max_output f64
    min_output f64
} (control_signal f64, emergency_stop u8) {
    safe_output f64
    clipped_output f64
    emergency_shutdown f64
} {
    if (emergency_stop) {
        emergency_shutdown = 0.0
    } else if (control_signal > max_output) {
        clipped_output = max_output
    } else if (control_signal < min_output) {
        clipped_output = min_output
    } else {
        safe_output = control_signal
    }
}

// Actuator stages
stage valve_actuator {} (position f64) {}
stage alarm {} (message f64) {}
stage logger {} (value f64) {}
stage emergency_shutdown {} (signal f64) {}

// Example 1: Temperature control with safety
// Input routing: Route setpoint and sensor to controller
{
    setpoint -> temp_setpoint_channel,
    measured -> temp_sensor_channel,
    enable -> controller_enable_channel
} -> pid_controller{kp=1.0, ki=0.1, kd=0.05} -> {
    control_output -> safety_limiter{max_output=100.0, min_output=0.0},
    error_output -> logger{}
}

// Example 2: Complete control loop with emergency handling
// Input routing with preprocessing, then output routing to multiple destinations
{
    control_signal -> pid_output_channel,
    emergency_stop -> estop_button_channel
} -> safety_limiter{max_output=95.0, min_output=5.0} -> {
    safe_output -> valve_actuator{},
    clipped_output -> alarm{},
    emergency_shutdown -> emergency_shutdown{}
}

// Example 3: Nested routing - output of one routing becomes input to another
stage threshold_check {
    threshold f64
} (value f64) {
    above_threshold u8
    below_threshold u8
} {
    if (value > threshold) {
        above_threshold = u8(1)
    } else {
        below_threshold = u8(1)
    }
}

stage combiner {} (signal_a u8, signal_b u8) u8 {
    return signal_a || signal_b
}

// Check if either error or output is out of range
temp_sensor_channel -> threshold_check{threshold=75.0} -> {
    above_threshold -> warning_channel,
    below_threshold -> normal_channel
}

// Combine multiple alarm conditions
{
    signal_a=high_temp_alarm,
    signal_b: high_pressure_alarm
} -> combiner{} -> master_alarm_channel
