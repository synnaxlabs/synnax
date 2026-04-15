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
#include <unordered_set>
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
class BenchNode final : public node::Node {
    std::vector<std::string> output_names;
    std::unordered_set<std::string> truthy;

public:
    explicit BenchNode(std::vector<std::string> ts = {}):
        output_names(ts), truthy(ts.begin(), ts.end()) {}

    x::errors::Error next(node::Context &ctx) override {
        for (size_t i = 0; i < this->output_names.size(); ++i)
            ctx.mark_changed(i);
        return x::errors::NIL;
    }

    [[nodiscard]] std::vector<std::string> outputs() const override {
        return this->output_names;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &p) const override {
        return this->truthy.contains(p);
    }
};

ir::Member node_ref_member(const std::string &key) {
    ir::Member m;
    m.key = key;
    ir::NodeRef ref;
    ref.key = key;
    m.node_ref = std::move(ref);
    return m;
}

ir::Member scope_member(ir::Scope s) {
    ir::Member m;
    m.key = s.key;
    m.scope = x::mem::indirect<ir::Scope>(std::move(s));
    return m;
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
    std::vector<ir::Member> members;
    members.reserve(n);
    for (size_t i = 0; i < n; ++i) {
        const auto k = "n" + std::to_string(i);
        ir::Node node;
        node.key = k;
        p.ir.nodes.push_back(std::move(node));
        members.push_back(node_ref_member(k));
        p.nodes[k] = std::make_unique<BenchNode>();
    }
    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    ir::Phase ph;
    ph.members = std::move(members);
    p.ir.root.phases.push_back(std::move(ph));
    return p;
}

Program build_fanout_chain(size_t n) {
    if (n < 2) n = 2;
    Program p;
    ir::Node src_node;
    src_node.key = "src";
    p.ir.nodes.push_back(std::move(src_node));
    p.nodes["src"] = std::make_unique<BenchNode>(std::vector<std::string>{"out"});

    std::vector<ir::Member> p0 = {node_ref_member("src")};
    std::vector<ir::Member> p1;
    p1.reserve(n - 1);
    for (size_t i = 1; i < n; ++i) {
        const auto k = "t" + std::to_string(i);
        ir::Node tn;
        tn.key = k;
        p.ir.nodes.push_back(std::move(tn));
        p1.push_back(node_ref_member(k));
        p.ir.edges.push_back(continuous_edge("src", "out", k, "in"));
        p.nodes[k] = std::make_unique<BenchNode>();
    }
    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    ir::Phase phase0;
    phase0.members = std::move(p0);
    ir::Phase phase1;
    phase1.members = std::move(p1);
    p.ir.root.phases.push_back(std::move(phase0));
    p.ir.root.phases.push_back(std::move(phase1));
    return p;
}

Program build_deep_nested(size_t depth) {
    Program p;
    ir::Node leaf;
    leaf.key = "leaf";
    p.ir.nodes.push_back(std::move(leaf));
    p.nodes["leaf"] = std::make_unique<BenchNode>();
    ir::Node trig;
    trig.key = "trigger";
    p.ir.nodes.push_back(std::move(trig));
    p.nodes["trigger"] = std::make_unique<BenchNode>(std::vector<std::string>{"go"});

    ir::Scope current;
    current.key = "s0";
    current.mode = ir::ScopeMode::Parallel;
    current.liveness = ir::Liveness::Gated;
    ir::Phase ph0;
    ph0.members = {node_ref_member("leaf")};
    current.phases.push_back(std::move(ph0));

    for (size_t i = 1; i < depth; ++i) {
        ir::Scope outer;
        outer.key = "s" + std::to_string(i);
        outer.mode = ir::ScopeMode::Parallel;
        outer.liveness = ir::Liveness::Gated;
        ir::Phase ph;
        ph.members = {scope_member(std::move(current))};
        outer.phases.push_back(std::move(ph));
        current = std::move(outer);
    }
    ir::Handle act{"trigger", "go"};
    current.activation = act;

    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    ir::Phase rphase;
    rphase.members = {node_ref_member("trigger"), scope_member(std::move(current))};
    p.ir.root.phases.push_back(std::move(rphase));
    return p;
}

Program build_sequential_chain(size_t n) {
    Program p;
    ir::Node trig;
    trig.key = "trigger";
    p.ir.nodes.push_back(std::move(trig));
    p.nodes["trigger"] = std::make_unique<BenchNode>(std::vector<std::string>{"go"});

    std::vector<ir::Member> members;
    std::vector<ir::Transition> transitions;
    members.reserve(n);
    transitions.reserve(n);
    for (size_t i = 0; i < n; ++i) {
        const auto k = "m" + std::to_string(i);
        ir::Node mn;
        mn.key = k;
        p.ir.nodes.push_back(std::move(mn));
        members.push_back(node_ref_member(k));
        p.nodes[k] = std::make_unique<BenchNode>(std::vector<std::string>{"next"});
        ir::Transition t;
        t.on = ir::Handle{k, "next"};
        if (i + 1 < n)
            t.target.member_key = "m" + std::to_string(i + 1);
        else
            t.target.exit = true;
        transitions.push_back(std::move(t));
    }

    ir::Scope seq;
    seq.key = "seq";
    seq.mode = ir::ScopeMode::Sequential;
    seq.liveness = ir::Liveness::Gated;
    seq.members = std::move(members);
    seq.transitions = std::move(transitions);
    ir::Handle act{"trigger", "go"};
    seq.activation = act;

    p.ir.root.mode = ir::ScopeMode::Parallel;
    p.ir.root.liveness = ir::Liveness::Always;
    ir::Phase rphase;
    rphase.members = {node_ref_member("trigger"), scope_member(std::move(seq))};
    p.ir.root.phases.push_back(std::move(rphase));
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

/// @brief positive-control benchmark that deliberately allocates once per
/// iteration so we can confirm the alloc-tracking infrastructure reports
/// the expected allocs/op and bytes/op. Should read 1 alloc/op and 64
/// bytes/op. If it reads 0 either value, the overrides or counters are
/// broken and every other benchmark's zero-alloc claim is untrustworthy.
void BM_AllocSanityCheck(benchmark::State &state) {
    run_with_alloc_tracking(state, [] {
        auto *p = new int[16];
        benchmark::DoNotOptimize(p);
        delete[] p;
    });
}
BENCHMARK(BM_AllocSanityCheck);

}}
