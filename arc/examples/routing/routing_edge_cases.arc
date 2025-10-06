// Edge Cases and Ambiguities in Routing Syntax

// ============================================================================
// Edge Case 1: Bracket Parsing Ambiguities
// ============================================================================

// Problem: Config braces vs routing brackets
stage configurable{
    threshold f64
    channels chan f32  // If channels could be config params
} (input f32) f32 {
    return input
}

// Ambiguous parse:
sensor -> configurable{threshold: 100}, other{}
//                                   ^ Config comma or routing comma?

// Clearer with brackets:
sensor -> [configurable{threshold: 100}, other{}]
//        ^ Bracket clearly indicates routing

// ============================================================================
// Edge Case 2: Multi-level Nesting
// ============================================================================

// How deep can brackets nest?
sensor -> [
    path_a -> [
        sub_a1{},
        sub_a2{}
    ],
    path_b -> [
        sub_b1{},
        sub_b2 -> [
            deep_c1{},
            deep_c2{}
        ]
    ]
]

// Does this create a tree of nodes or a flat list of edges?

// ============================================================================
// Edge Case 3: Fan-in Semantics
// ============================================================================

// Multiple sources to one target - what does this mean?
[sensor_a, sensor_b, sensor_c] -> processor{}

// Option A: Implicit merge (any fires => processor runs)
// Equivalent to: merge{sensor_a, sensor_b, sensor_c} -> processor{}

// Option B: Implicit all (all required before processor runs)
// Equivalent to: all{sensor_a, sensor_b, sensor_c} -> processor{}

// Option C: Syntax error - must be explicit
sensor_a -> processor{}
sensor_b -> processor{}
sensor_c -> processor{}

// ============================================================================
// Edge Case 4: Mixed Fan-in and Fan-out
// ============================================================================

// What does this mean?
[a, b] -> processor{} -> [x, y, z]

// Interpretation 1: Cartesian product (6 edges)
// a->processor_1->x, a->processor_1->y, a->processor_1->z
// b->processor_2->x, b->processor_2->y, b->processor_2->z

// Interpretation 2: Parallel paths (2 instances)
// a->processor_1->{x,y,z}
// b->processor_2->{x,y,z}

// Interpretation 3: Merged input, split output
// merge{a,b}->processor_1->{x,y,z}

// ============================================================================
// Edge Case 5: Empty Brackets
// ============================================================================

// What does this mean?
sensor -> []

// Option A: No-op (valid but useless)
// Option B: Syntax error
// Option C: Explicit null sink

// ============================================================================
// Edge Case 6: Single Element Brackets
// ============================================================================

// Are these equivalent?
sensor -> target{}
sensor -> [target{}]

// If brackets are pure syntax sugar, yes
// If brackets create tee node, no:
//   First: sensor->target
//   Second: sensor->tee->target

// ============================================================================
// Edge Case 7: Bracket Precedence with Chaining
// ============================================================================

// Which parse is correct?
a -> b{} -> [c{}, d{}] -> e{}

// Parse A: (a -> b) -> ([c, d] -> e)
// Creates: a->b->c->e, a->b->d->e

// Parse B: a -> (b -> [c, d]) -> e
// Creates: a->b->c, a->b->d, then what feeds e?

// ============================================================================
// Edge Case 8: Named Outputs with Brackets
// ============================================================================

stage multi_out{} (input f32) {
    out_a f32
    out_b f32
} {
    out_a = input * 2.0
    out_b = input * 3.0
}

// Can you combine these?
sensor -> multi_out{} -> {
    out_a -> [target_a1{}, target_a2{}],
    out_b -> [target_b1{}, target_b2{}]
}

// ============================================================================
// Edge Case 9: Inline Expressions with Routing
// ============================================================================

// Inline expression to multiple targets
(ox_pressure + fuel_pressure) / 2 -> [display{}, logger{}]

// Does the expression evaluate once and fan out,
// or does it create multiple expression nodes?

// ============================================================================
// Edge Case 10: Stage Return Values in Brackets
// ============================================================================

// Stages with return values
stage calc{} (x f32) f32 {
    return x * 2.0
}

// Can returned channel be bracketed?
sensor -> calc{} -> [a{}, b{}]

// If calc returns anonymous channel, this should work
// But does it create tee or multiple edges?

// ============================================================================
// Recommendation: Explicit is Better
// ============================================================================

// When in doubt, use explicit forms:

// Fan-out: Multiple statements
sensor -> target_a{}
sensor -> target_b{}

// Or explicit tee
sensor -> tee{target_a{}, target_b{}}

// Fan-in: Explicit merge/all
merge{sensor_a, sensor_b} -> processor{}
all{sensor_a, sensor_b} -> processor{}

// Multi-output: Named routing
sensor -> multi_out{} -> {
    out_a -> target_a{},
    out_b -> target_b{}
}

// Brackets: Only when semantics are crystal clear
sensor -> [target_a{}, target_b{}]
// Desugars to: sensor->target_a; sensor->target_b
