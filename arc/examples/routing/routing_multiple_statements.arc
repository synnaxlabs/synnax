// Approach 1: Multiple Flow Statements
// Most explicit - one edge per line

// Simple sensor fan-out
ox_pressure -> alarm_high{};
ox_pressure -> alarm_low{};
ox_pressure -> logger{};
ox_pressure -> display{};

// Chained processing with multiple consumers
fuel_sensor -> filter{cutoff: 100} -> fuel_filtered;
fuel_filtered -> pid_controller{kp: 1.5, ki: 0.1, kd: 0.05};
fuel_filtered -> trend_logger{};
fuel_filtered -> safety_monitor{};

// Multiple inputs, multiple outputs
ox_pressure -> startup_sequencer{};
fuel_pressure -> startup_sequencer{};
ignition_command -> startup_sequencer{};

startup_sequencer{} -> ox_valve{};
startup_sequencer{} -> fuel_valve{};
startup_sequencer{} -> igniter{};
startup_sequencer{} -> status_display{};

// Complex DAG
sensor_1 -> preprocessor{gain: 2.0};
sensor_2 -> preprocessor{gain: 2.0};

preprocessor{gain: 2.0} -> comparator{};
preprocessor{gain: 2.0} -> data_logger{};
preprocessor{gain: 2.0} -> realtime_plot{};

// Pros:
// - Crystal clear which edges exist
// - Easy to comment individual connections
// - Trivial to parse
// - Direct mapping to IR edges

// Cons:
// - Repetitive for large fan-outs
// - Fan-out pattern not visually grouped
