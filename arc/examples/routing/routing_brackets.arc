// Approach 2: Bracket Syntax for Fan-out
// Concise grouping of multiple targets

// Simple fan-out using brackets
ox_pressure -> [;
    alarm_high{},
    alarm_low{},
    logger{},
    display{}
]

// Chained with brackets mid-flow
fuel_sensor -> filter{cutoff: 100} -> [;
    pid_controller{kp: 1.5, ki: 0.1, kd: 0.05},
    trend_logger{},
    safety_monitor{}
]

// Multiple sources to bracket target
[ox_ready, fuel_ready, power_ready] -> all{} -> ignition_sequencer{};

// Brackets at multiple points in chain
sensor -> preprocessor{} -> [;
    path_a{} -> logger_a{},;
    path_b{} -> logger_b{};
] -> merger{};

// Fan-out then reconverge
temperature -> [;
    avg{},
    max{},
    min{}
] -> statistical_display{};

// Nested brackets for complex routing
main_sensor -> [;
    critical_path -> [;
        alarm{},
        shutdown{}
    ],
    monitoring_path -> [;
        logger{},
        display{},
        archive{}
    ]
]

// Design Questions:
// Q1: Does this desugar to multiple edges?
//     ox_pressure -> [a, b];
//     => ox_pressure -> a; ox_pressure -> b;
//
// Q2: Or does it create an implicit tee node?
//     ox_pressure -> [a, b];
//     => ox_pressure -> tee{a, b};
//
// Q3: What about bracket sources?
//     [a, b] -> c;
//     => a -> c; b -> c  (fan-in)?;
//     => merge{a, b} -> c (implicit merge)?;

// Pros:
// - Visual grouping makes relationships clear
// - Concise for common patterns
// - Natural symmetry with array literals

// Cons:
// - Ambiguous semantics (sugar vs node?)
// - Parsing complexity (nesting, precedence)
// - Mixed fan-in/fan-out could be confusing
