// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <cstdlib>
#include <memory>
#include <new>
#include <string>
#include <unordered_map>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "benchmark/benchmark.h"

namespace {
// Atomic counters incremented by the global operator new/new[] overrides
// defined below. Benchmarks snapshot these before and after the
// measurement loop to derive per-iteration allocation stats that surface
// in state.counters, matching Go's testing.B.ReportAllocs format.
std::atomic<int64_t> g_alloc_count{0};
std::atomic<int64_t> g_alloc_bytes{0};
}

void *operator new(const std::size_t n) {
    g_alloc_count.fetch_add(1, std::memory_order_relaxed);
    g_alloc_bytes.fetch_add(static_cast<int64_t>(n), std::memory_order_relaxed);
    if (auto *p = std::malloc(n)) return p;
    throw std::bad_alloc();
}

void *operator new[](const std::size_t n) {
    g_alloc_count.fetch_add(1, std::memory_order_relaxed);
    g_alloc_bytes.fetch_add(static_cast<int64_t>(n), std::memory_order_relaxed);
    if (auto *p = std::malloc(n)) return p;
    throw std::bad_alloc();
}

void operator delete(void *p) noexcept {
    std::free(p);
}
void operator delete[](void *p) noexcept {
    std::free(p);
}
void operator delete(void *p, std::size_t) noexcept {
    std::free(p);
}
void operator delete[](void *p, std::size_t) noexcept {
    std::free(p);
}

namespace arc::runtime::scheduler { namespace {

/// @brief records allocations attributed to the benchmark loop body and
/// publishes them as state.counters so the default console reporter
/// displays per-iteration allocs and bytes alongside time/op.
template<typename F>
void run_with_alloc_tracking(benchmark::State &state, F &&body) {
    const auto start_count = g_alloc_count.load(std::memory_order_relaxed);
    const auto start_bytes = g_alloc_bytes.load(std::memory_order_relaxed);
    for (auto _: state)
        body();
    state.counters["allocs/op"] = benchmark::Counter(
        static_cast<double>(
            g_alloc_count.load(std::memory_order_relaxed) - start_count
        ),
        benchmark::Counter::kAvgIterations
    );
    state.counters["bytes/op"] = benchmark::Counter(
        static_cast<double>(
            g_alloc_bytes.load(std::memory_order_relaxed) - start_bytes
        ),
        benchmark::Counter::kAvgIterations
    );
}

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
    current.liveness = ir::Liveness::Gated;
    current.strata.push_back(ir::Members{ir::node_member("leaf")});

    for (size_t i = 1; i < depth; ++i) {
        ir::Scope outer;
        outer.key = "s" + std::to_string(i);
        outer.mode = ir::ScopeMode::Parallel;
        outer.liveness = ir::Liveness::Gated;
        outer.strata.push_back(ir::Members{ir::scope_member(std::move(current))});
        current = std::move(outer);
    }
    ir::Handle act{"trigger", "go"};
    current.activation = act;

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

}}
