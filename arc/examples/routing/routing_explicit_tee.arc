// Approach 4: Explicit Tee Stage
// Using standard library tee{} stage (spec line 1112)

// Basic tee usage
sensor -> tee{;
    controller{},
    logger{},
    display{}
}

// Tee with nested stages
ox_pressure -> tee{;
    alarm{threshold: 500},
    logger{rate: 10hz},
    pid{kp: 1.5, ki: 0.1, kd: 0.05} -> valve_cmd;
}

// Multiple tees in pipeline
sensor -> filter{} -> tee{;
    path_a{} -> processor_a{},;
    path_b{} -> processor_b{};
} -> merger{};

// Tee appears as actual node in graph
// Advantage: The tee node is visible in visual editor
// Can be configured, debugged, monitored

// Tee with configuration
sensor -> tee{;
    // Could add tee-specific config in future:
    // synchronous: true,  // All outputs receive value simultaneously
    // buffer_policy: "drop_oldest"
} {
    critical_path{},
    monitoring_path{},
    archive_path{}
}

// Complex multi-stage with tees
main_sensor -> preprocessor{} -> tee{;
    realtime_display{},
    tee{
        archive_local{},
        archive_remote{}
    },
    analyzer{} -> alarm{};
}

// Tee for broadcast pattern
heartbeat -> tee{;
    subsystem_a{},
    subsystem_b{},
    subsystem_c{},
    subsystem_d{}
}

// Comparison with implicit approaches:

// Explicit tee:
sensor -> tee{a{}, b{}, c{}};
// Creates nodes: [sensor, tee_1, a, b, c]
// Creates edges: [sensor->tee_1, tee_1->a, tee_1->b, tee_1->c];

// Implicit (if brackets desugar):
sensor -> [a{}, b{}, c{}];
// Creates nodes: [sensor, a, b, c]
// Creates edges: [sensor->a, sensor->b, sensor->c];

// Pros:
// - Explicit about what's happening
// - Node appears in graph (can be inspected)
// - Could add tee-specific behavior/config
// - Already in spec

// Cons:
// - More verbose
// - Implementation detail in user code
// - Extra node in graph clutter
