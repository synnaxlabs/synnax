// Alarm System Example
// Simple example showing both routing types in a practical alarm scenario

// Dual-threshold alarm detector
stage alarm_detector {
    low_threshold f64
    high_threshold f64
} (current_value f64) {
    low_alarm f64
    high_alarm f64
    normal f64
} {
    if (current_value < low_threshold) {
        low_alarm = current_value
    } else if (current_value > high_threshold) {
        high_alarm = current_value
    } else {
        normal = current_value
    }
}

// Alarm handlers
stage low_alarm_handler {} (value f64) {
    // Handle low alarm - maybe turn on heater
}

stage high_alarm_handler {} (value f64) {
    // Handle high alarm - maybe turn on cooler
}

stage normal_logger {} (value f64) {
    // Just log normal values
}

// Stage that combines two sensor readings
stage sensor_averager {} (sensor_a f64, sensor_b f64) f64 {
    return (sensor_a + sensor_b) / 2.0
}

// Example 1: Simple output routing
// Monitor a single sensor and route to different handlers
temp_sensor_1 -> alarm_detector{low_threshold=15.0, high_threshold=30.0} -> {
    low_alarm: low_alarm_handler{},
    high_alarm: high_alarm_handler{},
    normal: normal_logger{}
}

// Example 2: Input routing followed by output routing
// Combine two sensors, then check thresholds and route
{
    temp_sensor_1: sensor_a,
    temp_sensor_2: sensor_b
} -> sensor_averager{} -> alarm_detector{low_threshold=18.0, high_threshold=28.0} -> {
    low_alarm: low_alarm_handler{},
    high_alarm: high_alarm_handler{},
    normal: normal_logger{}
}

// Example 3: Input routing with preprocessing
// Process sensors differently before combining
stage noise_filter {
    alpha f64
} (input f64) f64 {
    return input * alpha
}

{
    temp_sensor_1: noise_filter{alpha=0.95}: sensor_a,
    temp_sensor_2: noise_filter{alpha=0.90}: sensor_b
} -> sensor_averager{} -> alarm_detector{low_threshold=20.0, high_threshold=25.0} -> {
    low_alarm: low_alarm_channel,
    high_alarm: high_alarm_channel,
    normal: normal_log_channel
}

// Example 4: Chaining - route to channels, then another routing stage
stage severity_classifier {} (alarm_value f64) {
    minor f64
    major f64
} {
    if (alarm_value < 10.0) {
        minor = alarm_value
    } else {
        major = alarm_value
    }
}

stage minor_alert {} (value f64) {}
stage major_alert {} (value f64) {}
stage critical_alert {} (value f64) {}

// First routing: route detector outputs to different destinations
temperature_sensor -> alarm_detector{low_threshold=15.0, high_threshold=35.0} -> {
    low_alarm: low_alarm_staging_channel,
    high_alarm: critical_alert{},
    normal: normal_logger{}
}

// Second routing: classify low alarms by severity
low_alarm_staging_channel -> severity_classifier{} -> {
    minor: minor_alert{},
    major: major_alert{}
}
