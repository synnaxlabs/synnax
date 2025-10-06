// Input Routing Example
// Demonstrates routing multiple sources to specific parameters of a stage

// A stage that adds two values
stage add {} (a f32, b f32) f32 {
    return a + b
}

// A stage that computes weighted average
stage weighted_avg {} (value1 f32, value2 f32, weight1 f32, weight2 f32) f32 {
    return (value1 * weight1 + value2 * weight2) / (weight1 + weight2)
}

// A lowpass filter
stage lowpass {
    cutoff f32
} (input f32) f32 {
    // Simplified lowpass filter
    return input * cutoff
}

// A scale stage
stage scale {
    factor f32
} (input f32) f32 {
    return input * factor
}

// Example 1: Simple input routing
// Route sensor1 to parameter 'a' and sensor2 to parameter 'b'
{
    sensor1 -> a,
    sensor2 -> b
} -> add{}

// Example 2: Input routing with flow chains
// Process each input before routing
{
    sensor1 -> lowpass{cutoff=0.5} -> a,
    sensor2 -> scale{factor=2.0} -> b
} -> add{}

// Example 3: Multiple parameters with processing
{
    temp_sensor1 -> value1,
    temp_sensor2 -> value2,
    constant_a -> weight1,
    constant_b -> weight2
} -> weighted_avg{}

// Example 4: Complex flow chains in input routing
{
    sensor1 -> lowpass{cutoff=0.5} -> scale{factor=1.5} -> a,
    sensor2 -> scale{factor=0.5} -> lowpass{cutoff=0.8} -> b
} -> add{} -> output_channel
