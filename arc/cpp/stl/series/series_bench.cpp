// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

#include "benchmark/benchmark.h"
#include "wasmtime.hh"

#include "x/cpp/bench/bench.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/stl/series/series.h"
#include "arc/cpp/stl/series/state.h"

namespace arc::stl::series { namespace {

using x::bench::run_with_alloc_tracking;

const std::string_view LOGICAL_WAT = R"wat(
(module
  (import "series" "and" (func $and (param i32 i32) (result i32)))
  (import "series" "or" (func $or (param i32 i32) (result i32)))
  (import "series" "and_scalar" (func $and_scalar (param i32 i32) (result i32)))
  (import "series" "or_scalar" (func $or_scalar (param i32 i32) (result i32)))
  (import "series" "not" (func $not (param i32) (result i32)))
  (func (export "and") (param i32 i32) (result i32)
    (call $and (local.get 0) (local.get 1)))
  (func (export "or") (param i32 i32) (result i32)
    (call $or (local.get 0) (local.get 1)))
  (func (export "and_scalar") (param i32 i32) (result i32)
    (call $and_scalar (local.get 0) (local.get 1)))
  (func (export "or_scalar") (param i32 i32) (result i32)
    (call $or_scalar (local.get 0) (local.get 1)))
  (func (export "not") (param i32) (result i32)
    (call $not (local.get 0)))
)
)wat";

/// @brief owns the handle store and a WASM instance whose exports call through
/// to the logical host funcs.
struct Fixture {
    std::shared_ptr<State> state;
    Module mod;
    wasmtime::Engine engine;
    wasmtime::Store store;
    wasmtime::Linker linker;
    wasmtime::Instance instance;

    Fixture():
        state(std::make_shared<State>()),
        mod(state),
        store(engine),
        linker(engine),
        instance(setup()) {}

    wasmtime::Func get(const std::string &name) {
        return std::get<wasmtime::Func>(*instance.get(store, name));
    }

private:
    wasmtime::Instance setup() {
        mod.bind_to(linker, store);
        auto wasm_mod = wasmtime::Module::compile(engine, std::string(LOGICAL_WAT))
                            .unwrap();
        return linker.instantiate(store, wasm_mod).unwrap();
    }
};

/// @brief builds a bool series of the given length with alternating values.
x::telem::Series bool_series(const size_t size) {
    std::vector<uint8_t> vals(size);
    for (size_t i = 0; i < size; ++i)
        vals[i] = i % 2 == 0;
    return x::telem::Series(vals, x::telem::BOOLEAN_T);
}

/// @brief stores a shallow copy so each iteration re-stores inputs, matching
/// the runtime's per-cycle store/clear pattern, without re-copying input data.
int32_t store(Fixture &f, const x::telem::Series &s) {
    return static_cast<int32_t>(f.state->store(s.shallow_copy()));
}

void run_binary(benchmark::State &bstate, const std::string &fn_name) {
    Fixture f;
    const auto size = static_cast<size_t>(bstate.range(0));
    const auto lhs = bool_series(size);
    const auto rhs = bool_series(size);
    auto fn = f.get(fn_name);
    run_with_alloc_tracking(bstate, [&] {
        f.state->clear();
        (void) fn
            .call(f.store, {wasmtime::Val(store(f, lhs)), wasmtime::Val(store(f, rhs))})
            .unwrap();
    });
}

void run_scalar(benchmark::State &bstate, const std::string &fn_name) {
    Fixture f;
    const auto lhs = bool_series(static_cast<size_t>(bstate.range(0)));
    auto fn = f.get(fn_name);
    run_with_alloc_tracking(bstate, [&] {
        f.state->clear();
        (void) fn
            .call(f.store, {wasmtime::Val(store(f, lhs)), wasmtime::Val(int32_t{1})})
            .unwrap();
    });
}

void BM_And(benchmark::State &state) {
    run_binary(state, "and");
}
BENCHMARK(BM_And)->Arg(1)->Arg(1024);

void BM_Or(benchmark::State &state) {
    run_binary(state, "or");
}
BENCHMARK(BM_Or)->Arg(1)->Arg(1024);

void BM_AndScalar(benchmark::State &state) {
    run_scalar(state, "and_scalar");
}
BENCHMARK(BM_AndScalar)->Arg(1)->Arg(1024);

void BM_OrScalar(benchmark::State &state) {
    run_scalar(state, "or_scalar");
}
BENCHMARK(BM_OrScalar)->Arg(1)->Arg(1024);

void BM_Not(benchmark::State &state) {
    Fixture f;
    const auto lhs = bool_series(static_cast<size_t>(state.range(0)));
    auto fn = f.get("not");
    run_with_alloc_tracking(state, [&] {
        f.state->clear();
        (void) fn.call(f.store, {wasmtime::Val(store(f, lhs))}).unwrap();
    });
}
BENCHMARK(BM_Not)->Arg(1)->Arg(1024);

}}
