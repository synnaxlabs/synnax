// Combined Routing Example
// Demonstrates using both input and output routing in the same program

// Stage that takes two sensor inputs and computes their average
stage sensor_fusion {} (sensor_a f64, sensor_b f64) f64 {
    return (sensor_a + sensor_b) / 2.0
}

// Stage that classifies a value into three ranges
stage range_classifier {
    low_threshold f64
    high_threshold f64
} (value f64) {
    below f64
    normal f64
    above f64
} {
    if (value < low_threshold) {
        below = value
    } else if (value > high_threshold) {
        above = value
    } else {
        normal = value
    }
}

// Processing stages for different ranges
stage low_handler {} (value f64) {
    // Handle low values
}

stage normal_handler {} (value f64) {
    // Handle normal values
}

stage high_handler {} (value f64) {
    // Handle high values
}

// Example 1: Input routing followed by output routing
// Route two sensors to fusion, then classify and route the result
{
    sensor_a -> temp_sensor_1,
    sensor_b -> temp_sensor_2
} -> sensor_fusion{} -> range_classifier{low_threshold=20.0, high_threshold=80.0} -> {
    below -> low_handler{},
    normal -> normal_handler{},
    above -> high_handler{}
}

// Example 2: Complex flow with preprocessing before input routing
stage lowpass {
    cutoff f64
} (input f64) f64 {
    return input * cutoff
}

stage scale {
    factor f64
} (input f64) f64 {
    return input * factor
}

{
    sensor_a -> temp_sensor_1 -> lowpass{cutoff=0.8},
    sensor_b -> temp_sensor_2 -> scale{factor=1.2}
} -> sensor_fusion{} -> range_classifier{low_threshold=15.0, high_threshold=75.0} -> {
    below -> low_alarm_channel,
    normal -> normal_log_channel,
    above -> high_alarm_channel
}
