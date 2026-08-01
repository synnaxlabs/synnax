// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "benchmark/benchmark.h"

#include "x/cpp/bench/bench.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"

namespace arc::runtime::scheduler { namespace {

using x::bench::run_with_alloc_tracking;

/// @brief minimal node implementation for benchmarks. Avoids the tracking
/// overhead of the test MockNode so measurements reflect scheduler cost.
/// Output names live in ir::Node::outputs; this node fires mark_changed
/// for every declared truthy ordinal each cycle.
class BenchNode final : public node::Node {
public:
    std::vector<bool> truthy;

    explicit BenchNode(std::vector<bool> t = {}): truthy(std::move(t)) {}

    x::errors::Error next(node::Context &ctx) override {
        for (size_t i = 0; i < this->truthy.size(); ++i)
            if (this->truthy[i]) ctx.mark_changed(i);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        if (output_idx >= this->truthy.size()) return false;
        return this->truthy[output_idx];
    }
};

/// @brief builds an ir::Node with the given key and ordered output
/// names. Production runtime nodes never declare names; the scheduler
/// reads them exclusively from ir::Node::outputs.
ir::Node
ir_node(const std::string &key, std::initializer_list<std::string> outputs = {}) {
    ir::Node n;
    n.key = key;
    for (const auto &name: outputs)
        n.outputs.push_back(arc::types::Param{.name = name});
    return n;
}

ir::Edge continuous_edge(
    const std::string &src,
    const std::string &sp,
    const std::string &tgt,
    const std::string &tp
) {
    return ir::Edge{ir::Handle{src, sp}, ir::Handle{tgt, tp}, ir::EdgeKind::Continuous};
}

struct Program {
    ir::IR ir;
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
};

Program build_flat_parallel(size_t n) {
    Program p;
    ir::Members stratum;
    stratum.reserve(n);
    for (size_t i = 0; i < n; ++i) {
        const auto k = "n" + std::to_string(i);
        p.ir.nodes.push_back(ir_node(k));
        stratum.push_back(ir::node_member(k));
        p.nodes[k] = std::make_unique<BenchNode>();
    }
    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    p.ir.root.strata.push_back(std::move(stratum));
    return p;
}

Program build_fanout_chain(size_t n) {
    if (n < 2) n = 2;
    Program p;
    p.ir.nodes.push_back(ir_node("src", {"out"}));
    p.nodes["src"] = std::make_unique<BenchNode>(std::vector<bool>{true});

    ir::Members s0 = {ir::node_member("src")};
    ir::Members s1;
    s1.reserve(n - 1);
    for (size_t i = 1; i < n; ++i) {
        const auto k = "t" + std::to_string(i);
        p.ir.nodes.push_back(ir_node(k));
        s1.push_back(ir::node_member(k));
        p.ir.edges.push_back(continuous_edge("src", "out", k, "in"));
        p.nodes[k] = std::make_unique<BenchNode>();
    }
    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    p.ir.root.strata.push_back(std::move(s0));
    p.ir.root.strata.push_back(std::move(s1));
    return p;
}

Program build_deep_nested(size_t depth) {
    Program p;
    p.ir.nodes.push_back(ir_node("leaf"));
    p.nodes["leaf"] = std::make_unique<BenchNode>();
    p.ir.nodes.push_back(ir_node("trigger", {"go"}));
    p.nodes["trigger"] = std::make_unique<BenchNode>(std::vector<bool>{true});

    ir::Scope current;
    current.key = "s0";
    current.mode = ir::ScopeMode::Parallel;
    current.liveness = ir::Liveness::Always;
    current.strata.push_back(ir::Members{ir::node_member("leaf")});

    for (size_t i = 1; i < depth; ++i) {
        ir::Scope outer;
        outer.key = "s" + std::to_string(i);
        outer.mode = ir::ScopeMode::Parallel;
        outer.liveness = ir::Liveness::Always;
        outer.strata.push_back(ir::Members{ir::scope_member(std::move(current))});
        current = std::move(outer);
    }
    current.liveness = ir::Liveness::Gated;
    current.activation = ir::Handle{"trigger", "go"};

    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    p.ir.root.strata.push_back(
        ir::Members{ir::node_member("trigger"), ir::scope_member(std::move(current))}
    );
    return p;
}

Program build_sequential_chain(size_t n) {
    Program p;
    p.ir.nodes.push_back(ir_node("trigger", {"go"}));
    p.nodes["trigger"] = std::make_unique<BenchNode>(std::vector<bool>{true});

    ir::Members steps;
    std::vector<ir::Transition> transitions;
    steps.reserve(n);
    transitions.reserve(n);
    for (size_t i = 0; i < n; ++i) {
        const auto k = "m" + std::to_string(i);
        p.ir.nodes.push_back(ir_node(k, {"next"}));
        steps.push_back(ir::node_member(k));
        p.nodes[k] = std::make_unique<BenchNode>(std::vector<bool>{true});
        ir::Transition t;
        t.on = ir::Handle{k, "next"};
        if (i + 1 < n) t.target_key = "m" + std::to_string(i + 1);
        // leaving target_key unset signals exit for the terminal step.
        transitions.push_back(std::move(t));
    }

    ir::Scope seq;
    seq.key = "seq";
    seq.mode = ir::ScopeMode::Sequential;
    seq.liveness = ir::Liveness::Gated;
    seq.steps = std::move(steps);
    seq.transitions = std::move(transitions);
    ir::Handle act{"trigger", "go"};
    seq.activation = act;

    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    p.ir.root.strata.push_back(
        ir::Members{ir::node_member("trigger"), ir::scope_member(std::move(seq))}
    );
    return p;
}

/// @brief builds an n-node chain, one node per stratum, wired forward and then
/// closed with a backward edge from the last node to the first. Every node
/// marks every cycle, so each pass lands a change on the already-visited head
/// and the cycle runs to the settle bound of n+1 passes.
///
/// This is the bound, not a workload: no compiled Arc program re-fires
/// unconditionally like BenchNode does, and measured Arc counter chains stay
/// linear. Read the result as a guard on the bound logic, not as a cost real
/// programs pay.
Program build_settle_chain(size_t n) {
    if (n < 2) n = 2;
    Program p;
    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    for (size_t i = 0; i < n; ++i) {
        const auto k = "s" + std::to_string(i);
        p.ir.nodes.push_back(ir_node(k, {"out"}));
        p.nodes[k] = std::make_unique<BenchNode>(std::vector<bool>{true});
        p.ir.root.strata.push_back(ir::Members{ir::node_member(k)});
        if (i > 0)
            p.ir.edges.push_back(
                continuous_edge("s" + std::to_string(i - 1), "out", k, "in")
            );
    }
    p.ir.edges.push_back(
        continuous_edge("s" + std::to_string(n - 1), "out", "s0", "in")
    );
    return p;
}

/// @brief the settle-chain shape without its backward edge, so every change
/// lands on an unvisited node and the cycle settles in one pass. Differs from
/// build_settle_chain by exactly one edge, isolating the cost of re-walking
/// from the per-pass visited-flag clear.
Program build_settle_noop(size_t n) {
    if (n < 2) n = 2;
    Program p;
    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    for (size_t i = 0; i < n; ++i) {
        const auto k = "s" + std::to_string(i);
        p.ir.nodes.push_back(ir_node(k, {"out"}));
        p.nodes[k] = std::make_unique<BenchNode>(std::vector<bool>{true});
        p.ir.root.strata.push_back(ir::Members{ir::node_member(k)});
        if (i > 0)
            p.ir.edges.push_back(
                continuous_edge("s" + std::to_string(i - 1), "out", k, "in")
            );
    }
    return p;
}

/// @brief a sequential chain of n steps carrying n variable nodes as trailing
/// strata members. Those members run once per settle pass, so their cost
/// multiplies with the number of passes the cascade needs.
Program build_sequential_with_vars(size_t n) {
    Program p;
    p.ir.nodes.push_back(ir_node("trigger", {"go"}));
    p.nodes["trigger"] = std::make_unique<BenchNode>(std::vector<bool>{true});

    ir::Members steps;
    std::vector<ir::Transition> transitions;
    steps.reserve(n);
    transitions.reserve(n);
    for (size_t i = 0; i < n; ++i) {
        const auto k = "m" + std::to_string(i);
        p.ir.nodes.push_back(ir_node(k, {"next"}));
        steps.push_back(ir::node_member(k));
        p.nodes[k] = std::make_unique<BenchNode>(std::vector<bool>{true});
        ir::Transition t;
        t.on = ir::Handle{k, "next"};
        if (i + 1 < n) t.target_key = "m" + std::to_string(i + 1);
        transitions.push_back(std::move(t));
    }

    ir::Members vars;
    vars.reserve(n);
    for (size_t i = 0; i < n; ++i) {
        const auto k = "v" + std::to_string(i);
        p.ir.nodes.push_back(ir_node(k));
        vars.push_back(ir::node_member(k));
        p.nodes[k] = std::make_unique<BenchNode>();
    }

    ir::Scope seq;
    seq.key = "seq";
    seq.mode = ir::ScopeMode::Sequential;
    seq.liveness = ir::Liveness::Gated;
    seq.steps = std::move(steps);
    seq.transitions = std::move(transitions);
    seq.strata.push_back(std::move(vars));
    seq.activation = ir::Handle{"trigger", "go"};

    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    p.ir.root.strata.push_back(
        ir::Members{ir::node_member("trigger"), ir::scope_member(std::move(seq))}
    );
    return p;
}

void run_tick_bench(benchmark::State &state, Program p) {
    Scheduler sched(std::move(p.ir), p.nodes, x::telem::TimeSpan(0));
    run_with_alloc_tracking(state, [&] {
        sched.next(x::telem::MICROSECOND, node::RunReason::TimerTick);
    });
}

void BM_TickFlatParallel(benchmark::State &state) {
    run_tick_bench(state, build_flat_parallel(state.range(0)));
}
BENCHMARK(BM_TickFlatParallel)->Arg(10)->Arg(100)->Arg(1000);

void BM_TickFanoutChain(benchmark::State &state) {
    run_tick_bench(state, build_fanout_chain(state.range(0)));
}
BENCHMARK(BM_TickFanoutChain)->Arg(10)->Arg(100)->Arg(1000);

void BM_TickDeepNestedScopes(benchmark::State &state) {
    run_tick_bench(state, build_deep_nested(state.range(0)));
}
BENCHMARK(BM_TickDeepNestedScopes)->Arg(4)->Arg(16)->Arg(64);

void BM_TickSequentialCascade(benchmark::State &state) {
    run_tick_bench(state, build_sequential_chain(state.range(0)));
}
BENCHMARK(BM_TickSequentialCascade)->Arg(4)->Arg(16)->Arg(64);

void BM_Construction(benchmark::State &state) {
    for (auto _: state) {
        state.PauseTiming();
        auto p = build_fanout_chain(state.range(0));
        state.ResumeTiming();
        Scheduler sched(std::move(p.ir), p.nodes, x::telem::TimeSpan(0));
        benchmark::DoNotOptimize(sched);
    }
}
BENCHMARK(BM_Construction)->Arg(1000)->Arg(10000);

void BM_MarkChangedTruthy(benchmark::State &state) {
    run_tick_bench(state, build_fanout_chain(65));
}
BENCHMARK(BM_MarkChangedTruthy);

void BM_TickSettlePasses(benchmark::State &state) {
    run_tick_bench(state, build_settle_chain(state.range(0)));
}
BENCHMARK(BM_TickSettlePasses)->Arg(2)->Arg(8)->Arg(32)->Arg(128);

void BM_TickSettleNoop(benchmark::State &state) {
    run_tick_bench(state, build_settle_noop(state.range(0)));
}
BENCHMARK(BM_TickSettleNoop)->Arg(2)->Arg(8)->Arg(32)->Arg(128);

void BM_TickSequentialWithVars(benchmark::State &state) {
    run_tick_bench(state, build_sequential_with_vars(state.range(0)));
}
BENCHMARK(BM_TickSequentialWithVars)->Arg(4)->Arg(16)->Arg(64)->Arg(256);

}}
