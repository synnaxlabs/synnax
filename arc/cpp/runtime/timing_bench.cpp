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
#include <cstdint>
#include <cstdlib>
#include <iomanip>
#include <iostream>
#include <memory>
#include <string>
#include <thread>
#include <unordered_map>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/runtime.h"
#include "arc/cpp/runtime/testutil/timing.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime {
namespace {
/// @brief Periods measured for each interval.
constexpr size_t TICKS = 500;
/// @brief Maximum median distance between a tick and its period on an idle machine.
const auto MEDIAN_BOUND = x::telem::MILLISECOND;
/// @brief Time the runtime gets to start before the first tick.
const auto START_BOUND = 5 * x::telem::SECOND;
const std::string TIMER_KEY = "timer";

/// @brief Runs a timer node and records the cycle of each fire.
struct Recorder final : public node::Node {
    std::unique_ptr<node::Node> timer;
    /// @brief the elapsed time and the stamp of each fire. Sized up front so the
    /// runtime thread never allocates. Read them only after the runtime stops.
    std::vector<x::telem::TimeSpan> elapsed;
    std::vector<x::telem::TimeStamp> stamps;
    /// @brief how many fires are recorded. Safe to read from any thread.
    std::atomic<size_t> count{0};

    Recorder(std::unique_ptr<node::Node> timer, const size_t capacity):
        timer(std::move(timer)), elapsed(capacity), stamps(capacity) {}

    x::errors::Error next(node::Context &ctx) override {
        bool fired = false;
        auto mark = std::move(ctx.mark_changed);
        ctx.mark_changed = [&fired, &mark](const size_t output_idx) {
            fired = true;
            mark(output_idx);
        };
        const auto err = this->timer->next(ctx);
        ctx.mark_changed = std::move(mark);
        const auto i = this->count.load(std::memory_order_relaxed);
        if (fired && i < this->elapsed.size()) {
            this->elapsed[i] = ctx.cycle.elapsed;
            this->stamps[i] = ctx.cycle.now;
            this->count.store(i + 1, std::memory_order_release);
        }
        return err;
    }

    void reset(node::Context &ctx) override { this->timer->reset(ctx); }

    [[nodiscard]] bool is_output_truthy(const size_t output_idx) const override {
        return this->timer->is_output_truthy(output_idx);
    }
};

/// @brief Returns a program that holds one interval of period at its root.
ir::IR interval_program(const x::telem::TimeSpan period) {
    types::Param output;
    output.name = "output";
    output.type = types::Type{.kind = types::Kind::U8};
    types::Param input;
    input.name = "period";
    input.type = types::Type{.kind = types::Kind::I64};
    input.value = period.nanoseconds();
    ir::Node timer;
    timer.key = TIMER_KEY;
    timer.type = "interval";
    timer.outputs.push_back(output);
    timer.inputs.push_back(input);
    ir::Members stratum;
    stratum.push_back(ir::node_member(TIMER_KEY));
    ir::IR prog;
    prog.root.mode = ir::ScopeMode::Parallel;
    prog.root.liveness = ir::Liveness::Always;
    prog.root.strata.push_back(std::move(stratum));
    prog.nodes.push_back(timer);
    return prog;
}

/// @brief Runs one interval of period on a runtime with a real scheduler and loop,
/// and prints how far each period was from the nominal period. The drift is the time
/// the last tick is late after all the periods.
/// @param median_ns receives the median period error in nanoseconds.
void measure_interval(const x::telem::TimeSpan period, std::int64_t &median_ns) {
    const auto prog = interval_program(period);
    auto state = std::make_shared<state::State>(
        state::Config{.ir = prog, .channels = {}}
    );
    auto time_module = std::make_shared<stl::time::Module>();
    auto recorder = std::make_unique<Recorder>(
        ASSERT_NIL_P(time_module->create(
            node::Config(prog, prog.nodes[0], ASSERT_NIL_P(state->node(TIMER_KEY)))
        )),
        TICKS + 1
    );
    const auto *ticks = recorder.get();
    std::unordered_map<std::string, std::unique_ptr<node::Node>> nodes;
    nodes[TIMER_KEY] = std::move(recorder);

    loop::Config loop_cfg;
    // The Driver does not pin the loop thread on Windows.
    loop_cfg.cpu_affinity = loop::CPU_AFFINITY_NONE;
    loop_cfg = loop_cfg.apply_defaults(time_module->shortest_span());
    auto sched = std::make_unique<scheduler::Scheduler>(prog, nodes);
    const Config cfg{
        .program = {},
        .breaker = x::breaker::Config{},
        .retrieve_channels = nullptr,
        .input_queue_capacity = 256,
        .output_queue_capacity = 256,
        .loop = {},
    };
    const auto runtime = std::make_shared<Runtime>(
        cfg,
        nullptr,
        state,
        std::move(sched),
        loop::create(loop_cfg),
        time_module,
        std::vector<types::ChannelKey>{},
        std::vector<types::ChannelKey>{}
    );

    ASSERT_TRUE(runtime->start());
    const auto limit = START_BOUND + period * (2 * TICKS);
    const auto sw = x::telem::Stopwatch();
    while (ticks->count.load(std::memory_order_acquire) <= TICKS &&
           sw.elapsed() < limit)
        std::this_thread::sleep_for(x::telem::MILLISECOND.chrono());
    ASSERT_TRUE(runtime->stop());
    ASSERT_EQ(ticks->count.load(), TICKS + 1);

    std::vector<std::int64_t> errors_ns;
    errors_ns.reserve(TICKS);
    std::int64_t skew_ns = 0;
    for (size_t i = 1; i <= TICKS; i++) {
        const auto held = ticks->elapsed[i] - ticks->elapsed[i - 1];
        const auto stamped_ns = ticks->stamps[i].nanoseconds() -
                                ticks->stamps[i - 1].nanoseconds();
        skew_ns = std::max(skew_ns, std::abs(stamped_ns - held.nanoseconds()));
        errors_ns.push_back((held - period).nanoseconds());
    }
    const auto drift = ticks->elapsed[TICKS] - ticks->elapsed[0] - period * TICKS;
    const testutil::Spread spread(std::move(errors_ns));
    std::cout << std::fixed << std::setprecision(1) << "interval " << period << ", "
              << loop_cfg.mode << ", " << TICKS << " periods, error: " << spread
              << ", drift " << drift.microseconds() << " us, clock skew "
              << skew_ns / 1e3 << " us\n";
    median_ns = spread.at(50);
}
}

/// @brief Timing of one interval on a runtime with a real scheduler and loop.
class IntervalTimingTest : public testing::TestWithParam<x::telem::TimeSpan> {};

/// @brief On an idle machine, an interval should hold its period.
TEST_P(IntervalTimingTest, HoldsItsPeriodWhenIdle) {
    std::int64_t median_ns = 0;
    ASSERT_NO_FATAL_FAILURE(measure_interval(this->GetParam(), median_ns));
    EXPECT_LE(std::abs(median_ns), MEDIAN_BOUND.nanoseconds());
}

/// @brief With every core busy, an interval should still be measured. The printed
/// spread shows what the scheduler adds.
TEST_P(IntervalTimingTest, MeasuresUnderLoad) {
    const testutil::Load load;
    std::int64_t median_ns = 0;
    ASSERT_NO_FATAL_FAILURE(measure_interval(this->GetParam(), median_ns));
}

INSTANTIATE_TEST_SUITE_P(
    Periods,
    IntervalTimingTest,
    testing::Values(
        x::telem::MILLISECOND,
        2 * x::telem::MILLISECOND,
        5 * x::telem::MILLISECOND,
        10 * x::telem::MILLISECOND,
        20 * x::telem::MILLISECOND
    )
);
}
