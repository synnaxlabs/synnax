// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <functional>
#include <string>
#include <tuple>
#include <utility>
#include <vector>

#include "benchmark/benchmark.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/bench/bench.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/graph/types.gen.h"
#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/testutil/compile.h"
#include "arc/cpp/runtime/testutil/harness.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime { namespace {

using x::bench::run_with_alloc_tracking;

/// @brief compiles a program and runs it through the scheduler. Every declared
/// channel is created under a generated name and substituted into the source, so
/// repeated runs against one cluster never collide. Construction is the only
/// part that touches the network; tick and flush are local.
class Program {
    synnax::Synnax client = new_test_client();
    testutil::Channels channels;
    testutil::Harness harness;

public:
    Program(const std::string &source, const std::vector<testutil::ChannelSpec> &specs):
        channels(this->client, specs),
        harness(this->client, this->channels.substitute(source)) {}

    /// @brief compiles the source that build makes from the created channels.
    Program(
        const std::vector<testutil::ChannelSpec> &specs,
        const std::function<synnax::arc::Source(const testutil::Channels &)> &build
    ):
        channels(this->client, specs),
        harness(this->client, testutil::compile(this->client, build(this->channels))) {}

    /// @brief ticks the scheduler, then flushes and drops the cycle's writes, matching
    /// what the runtime loop does after every tick.
    void advance(const x::telem::TimeSpan elapsed) {
        this->harness.tick(elapsed);
        std::ignore = this->harness.flush();
    }

    /// @brief ingests data onto the channel declared as name.
    void ingest(const std::string &name, x::telem::Series &&data) {
        this->harness.ingest(this->channels.key(name), std::move(data));
    }

    /// @brief ingests a u8=1 onto the given channel and ticks long enough for
    /// the on-channel-read -> entry -> step cascade to settle.
    void trigger(const std::string &name) {
        this->harness.ingest(
            this->channels.key(name),
            x::telem::Series(std::uint8_t(1))
        );
        for (int i = 0; i < 5; i++)
            this->advance(x::telem::MILLISECOND);
    }
};

const std::vector<testutil::ChannelSpec> SPECS = {
    {"start_cmd", x::telem::UINT8_T},
    {"out", x::telem::INT64_T},
    {"tick", x::telem::UINT8_T},
};

// The plainest shape a program can use a variable in: declare, read, write out.
const std::string VARIABLE_READ_SRC = R"(
    sequence main {
        a i64 := 5
        a -> %out%
    }
    %start_cmd% => main)";

// A `$=` variable keeps its value across stage entries where a `:=` is restored
// to its initial, so the two take different reset paths on re-entry.
const std::string STATEFUL_SRC = R"(
    sequence main {
        count $= 0
        stage s {
            count + 1 -> count
            count -> %out%
        }
    }
    %start_cmd% => main)";

/// @brief builds n counters, each self-writing through an add node and feeding
/// the next. Every write lands on a register that already ran, so n sets how
/// many such writes one cycle carries.
std::string counter_chain_src(const int n) {
    std::string decls;
    std::string body;
    for (int i = 0; i < n; i++) {
        const auto v = "a" + std::to_string(i);
        decls += "        " + v + " i64 := 0\n";
        body += "            " + v + " + " +
                (i == 0 ? std::string("1") : "a" + std::to_string(i - 1)) + " -> " + v +
                "\n";
    }
    return "\n    sequence main {\n" + decls + "        stage s {\n" + body +
           "            a" + std::to_string(n - 1) + " -> %out%\n" +
           "        }\n    }\n    %start_cmd% => main";
}

// The reassignment keeps rate out of reach of constant folding, so the interval
// genuinely reads it live. Without it both timer sources compile to the same
// program and the pair measures nothing. Its wait is mirrored in the const
// source so the two differ only in where the period comes from.
const std::string TIMER_VAR_SRC = R"(import time
    sequence main {
        rate := i64 ns(100ms)
        stage s {
            time.interval(rate) -> %tick%
            time.wait(1s) -> sequence {
                rate = i64 ns(200ms)
            }
        }
    }
    %start_cmd% => main)";

const std::string TIMER_CONST_SRC = R"(import time
    sequence main {
        stage s {
            time.interval(100ms) -> %tick%
            time.wait(1s) -> %tick%
        }
    }
    %start_cmd% => main)";

/// @brief triggers the sequence, then measures steady-state ticks. Elapsed
/// advances every iteration so timer nodes reach their deadlines instead of
/// short-circuiting on a frozen clock.
void run_tick_bench(benchmark::State &state, const std::string &source) {
    Program p(source, SPECS);
    p.trigger("start_cmd");
    x::telem::TimeSpan elapsed = 5 * x::telem::MILLISECOND;
    run_with_alloc_tracking(state, [&] {
        elapsed += x::telem::MILLISECOND;
        p.advance(elapsed);
    });
}

void BM_TickVariableRead(benchmark::State &state) {
    run_tick_bench(state, VARIABLE_READ_SRC);
}
BENCHMARK(BM_TickVariableRead);

void BM_TickStatefulVariable(benchmark::State &state) {
    run_tick_bench(state, STATEFUL_SRC);
}
BENCHMARK(BM_TickStatefulVariable);

void BM_TickCounterChain(benchmark::State &state) {
    run_tick_bench(state, counter_chain_src(static_cast<int>(state.range(0))));
}
BENCHMARK(BM_TickCounterChain)->Arg(1)->Arg(3)->Arg(10)->Arg(100);

void BM_TickTimerVarPeriod(benchmark::State &state) {
    run_tick_bench(state, TIMER_VAR_SRC);
}
BENCHMARK(BM_TickTimerVarPeriod);

void BM_TickTimerConstPeriod(benchmark::State &state) {
    run_tick_bench(state, TIMER_CONST_SRC);
}
BENCHMARK(BM_TickTimerConstPeriod);

const std::vector<testutil::ChannelSpec> CALC_SPECS = {
    {"a", x::telem::FLOAT64_T},
    {"b", x::telem::FLOAT64_T},
    {"out", x::telem::FLOAT64_T},
};

types::Type f64() {
    types::Type t;
    t.kind = types::Kind::F64;
    return t;
}

void add_node(
    graph::Graph &g,
    const std::string &key,
    const std::string &type,
    x::json::json::object_t inputs = {}
) {
    g.nodes.push_back(graph::Node{.key = key});
    inputs["type"] = type;
    g.inputs[key] = std::move(inputs);
}

void add_edge(graph::Graph &g, const ir::Handle &source, const ir::Handle &target) {
    graph::Edge e;
    e.source = source;
    e.target = target;
    g.edges.push_back(e);
}

/// @brief builds the graph the Core compiles for a calculated channel: one on node
/// per read channel feeding a WASM function with the given body, then the given
/// operations in order, then a write to the out channel.
synnax::arc::Source calculation(
    const testutil::Channels &channels,
    const std::vector<std::string> &reads,
    const std::string &body,
    const std::vector<std::string> &ops = {}
) {
    graph::Graph g;
    ir::Function fn{.key = "calculation", .body = {.raw = "{" + body + "}"}};
    fn.outputs.push_back(types::Param{.name = "output", .type = f64()});
    add_node(g, "calculation", "calculation");
    for (const auto &name: reads) {
        fn.inputs.push_back(types::Param{.name = name, .type = f64()});
        add_node(g, name, "on", {{"channel", channels.key(name)}});
        add_edge(
            g,
            {.node = name, .param = "output"},
            {.node = "calculation", .param = name}
        );
    }
    g.functions.push_back(fn);
    std::string prev = "calculation";
    for (size_t i = 0; i < ops.size(); i++) {
        const auto key = "op_" + std::to_string(i);
        add_node(g, key, ops[i], {{"duration", 0}});
        add_edge(g, {.node = prev, .param = "output"}, {.node = key, .param = "input"});
        prev = key;
    }
    add_node(g, "write", "write", {{"channel", channels.key("out")}});
    add_edge(g, {.node = prev, .param = "output"}, {.node = "write", .param = "input"});
    return synnax::arc::GraphSource{.graph = std::move(g)};
}

/// @brief measures one cycle that ingests state.range(0) samples onto each read
/// channel, then runs and drains it. per_sample is the cost of one sample.
void run_sample_bench(
    benchmark::State &state,
    const std::vector<std::string> &reads,
    const std::string &body,
    const std::vector<std::string> &ops = {}
) {
    Program p(CALC_SPECS, [&](const testutil::Channels &channels) {
        return calculation(channels, reads, body, ops);
    });
    const auto n = static_cast<size_t>(state.range(0));
    std::vector<double> values(n);
    for (size_t i = 0; i < n; i++)
        values[i] = static_cast<double>(i);
    const x::telem::Series data(values);
    x::telem::TimeSpan elapsed;
    run_with_alloc_tracking(state, [&] {
        for (const auto &name: reads)
            p.ingest(name, data.deep_copy());
        elapsed += x::telem::MILLISECOND;
        p.advance(elapsed);
    });
    state.counters["per_sample"] = benchmark::Counter(
        static_cast<double>(n),
        benchmark::Counter::kIsIterationInvariantRate | benchmark::Counter::kInvert
    );
}

void BM_SampleAdd(benchmark::State &state) {
    run_sample_bench(state, {"a", "b"}, "return a + b");
}
BENCHMARK(BM_SampleAdd)->Arg(1)->Arg(1000)->Arg(10000);

void BM_SampleAvg(benchmark::State &state) {
    run_sample_bench(state, {"a"}, "return a", {"avg"});
}
BENCHMARK(BM_SampleAvg)->Arg(1)->Arg(1000)->Arg(10000);

}}
