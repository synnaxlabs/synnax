// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <string>
#include <vector>

#include "benchmark/benchmark.h"

#include "x/cpp/bench/bench.h"
#include "x/cpp/telem/series.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::state { namespace {

using x::bench::run_with_alloc_tracking;

constexpr int64_t PERIOD_NS = 100LL * 1000 * 1000;

/// @brief reports err on state and returns true when the setup failed.
bool failed(benchmark::State &state, const x::errors::Error &err) {
    if (!err) return false;
    const auto msg = err.message();
    state.SkipWithError(msg.c_str());
    return true;
}

/// @brief builds a literal i64 param valued as the compiler emits it: a JSON number.
types::Param literal_param(std::string name, const int64_t value) {
    types::Param p;
    p.name = std::move(name);
    p.type = types::Type{.kind = types::Kind::I64};
    p.value = value;
    return p;
}

/// @brief builds an i64 param bound to variable node var, carrying its declared
/// initial.
types::Param
var_param(std::string name, const std::string &var, const int64_t initial) {
    types::Param p;
    p.name = std::move(name);
    p.type = types::Type{
        .kind = types::Kind::VarRef,
        .name = var,
        .elem = x::mem::indirect<types::Type>(types::Type{.kind = types::Kind::I64})
    };
    p.value = initial;
    return p;
}

/// @brief consumer c takes a literal "period" and a var-bound "rate" (variable node
/// v): the two ways a timer node receives its span.
ir::IR program() {
    ir::Node v;
    v.key = "v";
    v.type = "variable";
    v.outputs.push_back(literal_param(ir::default_output_param, 0));
    ir::Node c;
    c.key = "c";
    c.type = "consumer";
    c.inputs.push_back(literal_param("period", PERIOD_NS));
    c.inputs.push_back(var_param("rate", "v", PERIOD_NS));
    ir::IR prog;
    prog.nodes.push_back(v);
    prog.nodes.push_back(c);
    return prog;
}

/// @brief the read every timer node performs today: resolve by name, then parse
/// the literal out of its JSON.
void BM_NumericInputLiteral(benchmark::State &state) {
    State s(Config{.ir = program()});
    auto [c, err] = s.node("c");
    if (failed(state, err)) return;
    run_with_alloc_tracking(state, [&] {
        benchmark::DoNotOptimize(c.numeric_input<int64_t>("period"));
    });
}
BENCHMARK(BM_NumericInputLiteral);

/// @brief the same value read from the typed series node construction already
/// built for the literal, with the index resolved once.
void BM_InputSeriesLiteral(benchmark::State &state) {
    State s(Config{.ir = program()});
    auto [c, err] = s.node("c");
    if (failed(state, err)) return;
    const auto [idx, idx_err] = c.resolve_input("period");
    if (failed(state, idx_err)) return;
    run_with_alloc_tracking(state, [&] {
        benchmark::DoNotOptimize(c.input(idx)->at<int64_t>(-1));
    });
}
BENCHMARK(BM_InputSeriesLiteral);

/// @brief the by-name lookup alone.
void BM_ResolveInput(benchmark::State &state) {
    State s(Config{.ir = program()});
    auto [c, err] = s.node("c");
    if (failed(state, err)) return;
    run_with_alloc_tracking(state, [&] {
        benchmark::DoNotOptimize(c.resolve_input("period").first);
    });
}
BENCHMARK(BM_ResolveInput);

/// @brief the JSON parse and cast alone.
void BM_ToSampleValue(benchmark::State &state) {
    const auto p = literal_param("period", PERIOD_NS);
    run_with_alloc_tracking(state, [&] {
        benchmark::DoNotOptimize(
            x::telem::cast<int64_t>(*types::to_sample_value(p.value, p.type))
        );
    });
}
BENCHMARK(BM_ToSampleValue);

/// @brief a var-bound read, served from the variable's output series. The
/// scheduler fills that slot before the first cycle, so this is the only state
/// a real program reaches.
void BM_NumericInputVar(benchmark::State &state) {
    State s(Config{.ir = program()});
    auto [v, v_err] = s.node("v");
    if (failed(state, v_err)) return;
    auto [c, c_err] = s.node("c");
    if (failed(state, c_err)) return;
    *v.output(0) = x::telem::Series(std::vector<int64_t>{PERIOD_NS});
    run_with_alloc_tracking(state, [&] {
        benchmark::DoNotOptimize(c.numeric_input<int64_t>("rate"));
    });
}
BENCHMARK(BM_NumericInputVar);

}}
