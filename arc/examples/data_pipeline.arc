// Data Pipeline Example
// Demonstrates complex data flows with multiple routing points

// Data validation stage
stage validator {
    min_value f64
    max_value f64
} (raw_data f64) {
    valid_data f64
    invalid_data f64
    out_of_range f64
} {
    if (raw_data < min_value || raw_data > max_value) {
        out_of_range = raw_data
    } else if (raw_data == 0.0) {
        invalid_data = raw_data
    } else {
        valid_data = raw_data
    }
}

// Statistical aggregator
stage aggregator {} (value_1 f64, value_2 f64, value_3 f64) {
    average f64
    max_value f64
    min_value f64
} {
    sum := value_1 + value_2 + value_3
    average = sum / 3.0

    // Find max
    max_value = value_1
    if (value_2 > max_value) {
        max_value = value_2
    }
    if (value_3 > max_value) {
        max_value = value_3
    }

    // Find min
    min_value = value_1
    if (value_2 < min_value) {
        min_value = value_2
    }
    if (value_3 < min_value) {
        min_value = value_3
    }
}

// Quality classifier
stage quality_check {
    warning_threshold f64
    critical_threshold f64
} (variance f64) {
    good_quality f64
    warning_quality f64
    critical_quality f64
} {
    if (variance < warning_threshold) {
        good_quality = variance
    } else if (variance < critical_threshold) {
        warning_quality = variance
    } else {
        critical_quality = variance
    }
}

// Processing stages
stage filter {} (value f64) f64 { return value * 0.9 }
stage amplify {} (value f64) f64 { return value * 1.5 }
stage normalize {} (value f64) f64 { return value / 100.0 }

// Archive stages
stage database_writer {} (data f64) {}
stage error_logger {} (error f64) {}
stage alert_system {} (alert f64) {}
stage dashboard {} (metric f64) {}

// Pipeline Step 1: Validate incoming data from three sensors
sensor_1_raw -> validator{min_value=0.0, max_value=1000.0} -> {
    valid_data -> sensor_1_clean,
    invalid_data -> error_logger{},
    out_of_range -> alert_system{}
}

sensor_2_raw -> validator{min_value=0.0, max_value=1000.0} -> {
    valid_data -> sensor_2_clean,
    invalid_data -> error_logger{},
    out_of_range -> alert_system{}
}

sensor_3_raw -> validator{min_value=0.0, max_value=1000.0} -> {
    valid_data -> sensor_3_clean,
    invalid_data -> error_logger{},
    out_of_range -> alert_system{}
}

// Pipeline Step 2: Aggregate validated data
// Input routing to combine three processed sensors
{
    value_1 -> sensor_1_clean,
    value_2 -> sensor_2_clean,
    value_3 -> sensor_3_clean
} -> aggregator{} -> {
    average -> database_writer{},
    max_value -> dashboard{},
    min_value -> dashboard{}
}

// Pipeline Step 3: Process with different filters and route based on quality
// Input routing with different preprocessing per input
{
    value_1 -> sensor_1_clean -> filter{},
    value_2 -> sensor_2_clean -> amplify{},
    value_3 -> sensor_3_clean -> normalize{}
} -> aggregator{} -> {
    average -> variance_calculator,
    max_value -> max_tracker,
    min_value -> min_tracker
}

// Pipeline Step 4: Quality assessment and conditional archiving
variance_calculator -> quality_check{warning_threshold=10.0, critical_threshold=50.0} -> {
    good_quality -> database_writer{},
    warning_quality -> alert_system{},
    critical_quality -> error_logger{}
}

// Pipeline Step 5: Fan-out pattern - one source to multiple destinations via routing
stage broadcaster {} (input f64) {
    stream_1 f64
    stream_2 f64
    stream_3 f64
    stream_4 f64
} {
    stream_1 = input
    stream_2 = input
    stream_3 = input
    stream_4 = input
}

master_sensor -> broadcaster{} -> {
    stream_1 -> database_writer{},
    stream_2 -> dashboard{},
    stream_3 -> alert_system{},
    stream_4 -> backup_writer
}
