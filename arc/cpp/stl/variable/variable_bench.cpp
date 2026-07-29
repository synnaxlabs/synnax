// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/bench/bench.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/variable/variable.h"
#include "benchmark/benchmark.h"

namespace arc::stl::variable { namespace {

using x::bench::run_with_alloc_tracking;

/// @brief builds a param with the given name, type, and configured value.
types::Param
param(std::string name, const types::Kind kind, x::json::json value = nullptr) {
    types::Param p;
    p.name = std::move(name);
    p.type.kind = kind;
    p.value = std::move(value);
    return p;
}

/// @brief builds an IR node with the given key, type, inputs, and outputs.
ir::Node make_node(
    std::string key,
    std::string type,
    const std::vector<types::Param> &inputs,
    const std::vector<types::Param> &outputs
) {
    ir::Node n;
    n.key = std::move(key);
    n.type = std::move(type);
    for (const auto &p: inputs)
        n.inputs.push_back(p);
    for (const auto &p: outputs)
        n.outputs.push_back(p);
    return n;
}

/// @brief builds a register variable "v" with a value-carrying first param and
/// n feeder nodes edged into the params after it. last_changed scans every
/// feeder on each fire, so n sets the scan width.
ir::IR register_ir(const size_t feeders) {
    ir::IR prog;
    std::vector<types::Param> inputs{param("value", types::Kind::I64, 42)};
    for (size_t i = 0; i < feeders; ++i) {
        const auto f = "f" + std::to_string(i);
        prog.nodes.push_back(make_node(
            f,
            "feeder",
            {},
            {param(ir::default_output_param, types::Kind::I64)}
        ));
        inputs.push_back(param("in" + std::to_string(i), types::Kind::I64));
    }
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        inputs,
        {param(ir::default_output_param, types::Kind::I64)}
    ));
    for (size_t i = 0; i < feeders; ++i)
        prog.edges.emplace_back(
            ir::Handle{"f" + std::to_string(i), ir::default_output_param},
            ir::Handle{"v", "in" + std::to_string(i)},
            ir::EdgeKind::Continuous
        );
    return prog;
}

/// @brief builds an ExprRead variable "v" whose value is edge-fed by a
/// dispatcher stand-in "d". With sel, a register stand-in "selsrc" drives the
/// re-point input the deref checks before its value.
ir::IR expr_read_ir(const bool with_sel) {
    ir::IR prog;
    prog.nodes.push_back(
        make_node("d", "d", {}, {param(ir::default_output_param, types::Kind::I64)})
    );
    std::vector<types::Param> inputs{param("value", types::Kind::I64)};
    if (with_sel) {
        prog.nodes.push_back(make_node(
            "selsrc",
            "stateful_variable",
            {},
            {param(ir::default_output_param, types::Kind::U32)}
        ));
        inputs.push_back(param("sel", types::Kind::U32));
    }
    prog.nodes.push_back(make_node(
        "v",
        "variable",
        inputs,
        {param(ir::default_output_param, types::Kind::I64)}
    ));
    prog.edges.emplace_back(
        ir::Handle{"d", ir::default_output_param},
        ir::Handle{"v", "value"},
        ir::EdgeKind::Continuous
    );
    if (with_sel)
        prog.edges.emplace_back(
            ir::Handle{"selsrc", ir::default_output_param},
            ir::Handle{"v", "sel"},
            ir::EdgeKind::Continuous
        );
    return prog;
}

/// @brief owns an IR, the state built from it, and the variable node under
/// measurement. All three outlive the loop, so a Program must stay put for the
/// benchmark's duration.
class Program {
    ir::IR prog;
    runtime::state::State state;
    Module module;

public:
    std::unique_ptr<runtime::node::Node> node;

    explicit Program(ir::IR ir):
        prog(std::move(ir)),
        state(
            runtime::state::Config{.ir = prog, .channels = {}},
            runtime::errors::noop_handler
        ) {
        auto [ns, ns_err] = this->state.node("v");
        if (ns_err) throw std::runtime_error(ns_err.message());
        auto [created, err] = this->module.create(
            runtime::node::Config(this->prog, this->prog.node("v"), std::move(ns))
        );
        if (err) throw std::runtime_error(err.message());
        this->node = std::move(created);
    }

    Program(const Program &) = delete;
    Program &operator=(const Program &) = delete;

    runtime::state::Node source(const std::string &key) {
        auto [n, err] = this->state.node(key);
        if (err) throw std::runtime_error(err.message());
        return std::move(n);
    }
};

runtime::node::Context bench_context() {
    return runtime::node::Context{
        .elapsed = x::telem::SECOND,
        .mark_changed = [](size_t) {},
        .mark_self_changed = [] {},
        .set_deadline = [](x::telem::TimeSpan) {},
        .report_error = [](const x::errors::Error &) {},
    };
}

/// @brief sizes src's first output to a single sample so the loop can overwrite
/// it in place. Reassigning the series each iteration would allocate and swamp
/// the counters.
void prime(const runtime::state::Node &src) {
    *src.output(0) = x::telem::Series(std::vector<int64_t>{0});
    *src.output_time(0) = x::telem::Series(x::telem::TimeStamp(0));
}

/// @brief drives src with a fresh value and timestamp each iteration. Without a
/// newer timestamp the node treats its input as consumed and returns early,
/// measuring nothing.
void run_feed_bench(
    benchmark::State &state,
    Program &p,
    const runtime::state::Node &src
) {
    auto ctx = bench_context();
    int64_t t = 0;
    run_with_alloc_tracking(state, [&] {
        ++t;
        src.output(0)->set(0, t);
        src.output_time(0)->set(0, x::telem::TimeStamp(t * x::telem::SECOND));
        p.node->next(ctx);
    });
}

void BM_RegisterNext(benchmark::State &state) {
    Program p(register_ir(static_cast<size_t>(state.range(0))));
    const auto src = p.source("f0");
    prime(src);
    run_feed_bench(state, p, src);
}
BENCHMARK(BM_RegisterNext)->Arg(1)->Arg(4)->Arg(16);

void BM_ExprReadNext(benchmark::State &state) {
    Program p(expr_read_ir(false));
    const auto src = p.source("d");
    prime(src);
    run_feed_bench(state, p, src);
}
BENCHMARK(BM_ExprReadNext);

void BM_ExprReadNextSel(benchmark::State &state) {
    Program p(expr_read_ir(true));
    const auto src = p.source("d");
    prime(src);
    // sel stays put: a fresh sel re-points the deref and suppresses the value.
    prime(p.source("selsrc"));
    run_feed_bench(state, p, src);
}
BENCHMARK(BM_ExprReadNextSel);

}}
