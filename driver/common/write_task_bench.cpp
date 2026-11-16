// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <chrono>
#include <random>
#include <thread>
#include <vector>

#include "benchmark/benchmark.h"

#include "driver/common/write_task.h"

namespace {
using std::chrono::steady_clock;

class Sink final : public driver::common::Sink {
public:
    using driver::common::Sink::Sink;

    x::errors::Error write(x::telem::Frame &frame) override {
        this->set_state(frame);
        return x::errors::NIL;
    }
};

void spin_for(const std::chrono::nanoseconds d) {
    const auto until = steady_clock::now() + d;
    while (steady_clock::now() < until) {}
}
}

/// Measures how long a command takes to show up in a state frame. Arg 0 is the state
/// rate in Hz. Arg 1 is the time in microseconds that the state loop spends writing
/// each state frame to the Core. Commands arrive at random points in the state period.
static void BM_CommandToState(benchmark::State &state) {
    const auto rate = x::telem::HERTZ * static_cast<float>(state.range(0));
    const auto write_cost = std::chrono::microseconds(state.range(1));
    const auto period = rate.period().chrono();
    synnax::channel::Channel state_ch;
    state_ch.key = 2;
    state_ch.data_type = x::telem::INT64_T;
    Sink sink(rate, {{1, state_ch}}, false);
    std::atomic<bool> done{false};
    std::atomic<int64_t> seen{0};
    std::atomic<int64_t> seen_at_ns{0};
    std::thread state_loop([&] {
        x::breaker::Breaker breaker;
        driver::pipeline::Authorities authorities;
        x::telem::Frame fr;
        while (!done.load()) {
            static_cast<void>(sink.read(breaker, fr, authorities));
            const auto value = fr.at<int64_t>(2, 0);
            if (value != seen.load()) {
                seen_at_ns.store(steady_clock::now().time_since_epoch().count());
                seen.store(value);
            }
            spin_for(write_cost);
        }
    });
    // NOLINTNEXTLINE(bugprone-random-generator-seed): repeatable command offsets.
    std::mt19937 rng(42);
    std::uniform_int_distribution<int64_t> offset_ns(0, period.count());
    std::vector<double> latencies_ms;
    int64_t i = 0;
    for (auto _: state) {
        i++;
        std::this_thread::sleep_for(std::chrono::nanoseconds(offset_ns(rng)));
        x::telem::Frame cmd(1);
        cmd.emplace(1, x::telem::Series(i));
        const auto sent_at = steady_clock::now();
        static_cast<void>(sink.write(cmd));
        while (seen.load() < i) {}
        const steady_clock::time_point seen_at(
            steady_clock::duration(seen_at_ns.load())
        );
        const std::chrono::duration<double> latency = seen_at - sent_at;
        state.SetIterationTime(latency.count());
        latencies_ms.push_back(latency.count() * 1e3);
    }
    done.store(true);
    state_loop.join();
    std::ranges::sort(latencies_ms);
    const auto period_ms = std::chrono::duration<double, std::milli>(period).count();
    state.counters["p99_ms"] = latencies_ms[latencies_ms.size() * 99 / 100];
    state.counters["max_ms"] = latencies_ms.back();
    state.counters["period_ms"] = period_ms;
    state.counters["slow_over_half_period"] = static_cast<double>(
        latencies_ms.end() - std::ranges::upper_bound(latencies_ms, period_ms / 2)
    );
}

BENCHMARK(BM_CommandToState)->UseManualTime()->Args({2000, 50})->Iterations(5000);
BENCHMARK(BM_CommandToState)->UseManualTime()->Args({100, 50})->Iterations(500);
BENCHMARK(BM_CommandToState)->UseManualTime()->Args({10, 50})->Iterations(100);
