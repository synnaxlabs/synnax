// Multi-Output Routing Example
// Demonstrates named outputs and routing tables for conditional data flow

// A stage that demultiplexes sensor values based on a threshold
stage threshold_demux {
    threshold f64
} (value f64) {
    high f64
    low f64
} {
    if (value > threshold) {
        high = value
    } else {
        low = value
    }
}

// A stage that classifies values into three ranges
stage range_classifier {} (value f64) {
    below_range f64
    in_range f64
    above_range f64
} {
    if (value < 20.0) {
        below_range = value
    } else if (value <= 80.0) {
        in_range = value
    } else {
        above_range = value
    }
}

// Target stages for different outputs
stage low_alarm {} (value f64) {
    // Handle low values
}

stage high_alarm {} (value f64) {
    // Handle high values
}

stage normal_logger {} (value f64) {
    // Log normal values
}

stage data_processor {} (value f64) f64 {
    return value * 1.5
}

// Example 1: Simple routing table
temperature_sensor -> threshold_demux{threshold: 100.0} -> {
    high -> high_alarm{},
    low -> low_alarm{}
}

// Example 2: Three-way routing
pressure_sensor -> range_classifier{} -> {
    below_range -> low_alarm{},
    in_range -> normal_logger{},
    above_range -> high_alarm{}
}

// Example 3: Routing with chained processing
flow_sensor -> threshold_demux{threshold: 50.0} -> {
    high -> data_processor{} -> high_alarm{},
    low -> normal_logger{}
}

// Example 4: Routing to channels
voltage_sensor -> threshold_demux{threshold: 12.0} -> {
    high -> overvoltage_channel,
    low -> normal_voltage_channel
}