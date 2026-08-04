// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstdint>

#include "benchmark/benchmark.h"

namespace x::bench {
// Atomic counters incremented by the global operator new/new[] overrides in
// bench.cpp. Benchmarks snapshot these before and after the measurement loop
// to derive per-iteration allocation stats that surface in state.counters,
// matching Go's testing.B.ReportAllocs format.
extern std::atomic<int64_t> alloc_count;
extern std::atomic<int64_t> alloc_bytes;

/// @brief records allocations attributed to the benchmark loop body and
/// publishes them as state.counters so the default console reporter
/// displays per-iteration allocs and bytes alongside time/op.
template<typename F>
void run_with_alloc_tracking(benchmark::State &state, F &&body) {
    const auto start_count = alloc_count.load(std::memory_order_relaxed);
    const auto start_bytes = alloc_bytes.load(std::memory_order_relaxed);
    for (auto _: state)
        body();
    state.counters["allocs/op"] = benchmark::Counter(
        static_cast<double>(alloc_count.load(std::memory_order_relaxed) - start_count),
        benchmark::Counter::kAvgIterations
    );
    state.counters["bytes/op"] = benchmark::Counter(
        static_cast<double>(alloc_bytes.load(std::memory_order_relaxed) - start_bytes),
        benchmark::Counter::kAvgIterations
    );
}
}
