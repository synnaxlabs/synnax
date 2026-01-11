// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>

#include "gtest/gtest.h"

#include "driver/task/common/sample_clock.h"

/// @brief it should correctly use the system clock to time samples.
TEST(TestSampleClock, testSoftwareTimedSampleClock) {
    auto clock = driver::task::common::SoftwareTimedSampleClock(x::telem::HERTZ * 250);
    auto now = x::telem::TimeStamp::now();
    x::breaker::Breaker b;
    const auto start = clock.wait(b);
    EXPECT_GE(start, now);
    now = x::telem::TimeStamp::now();
    const auto end = clock.end();
    ASSERT_GE(end, now);
}

/// @brief it should correctly rely on steady sample spacing to time samples.
TEST(TestSampleClock, testHardwareTimedSampleClockNominal) {
    const auto sample_rate = x::telem::HERTZ * 2;
    const auto stream_rate = x::telem::HERTZ * 1;
    auto now_v = 0 * x::telem::SECOND;
    auto now_f = [&now_v] { return x::telem::TimeStamp(now_v); };
    auto clock = driver::task::common::HardwareTimedSampleClock(
        {.now = now_f,
         .sample_rate = sample_rate,
         .stream_rate = stream_rate,
         .k_p = 0,
         .k_i = 0,
         .k_d = 0}
    );
    x::breaker::Breaker b;

    auto start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(now_v));
    now_v = x::telem::SECOND * 1;
    auto end = clock.end();
    ASSERT_EQ(end, x::telem::TimeStamp(x::telem::SECOND * 1));

    start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(x::telem::SECOND * 1));
    now_v = x::telem::SECOND * 2;
    end = clock.end();
    ASSERT_EQ(end, x::telem::TimeStamp(x::telem::SECOND * 2));
}

/// @brief it should apply proportional correction when system time runs late.
TEST(TestSampleClock, testHardwareTimedSampleClockNowIsLater) {
    const auto sample_rate = x::telem::HERTZ * 2;
    const auto stream_rate = x::telem::HERTZ * 1;
    auto now_v = 0 * x::telem::SECOND;
    auto now_f = [&now_v] { return x::telem::TimeStamp(now_v); };
    constexpr double k_p = 0.1;
    auto clock = driver::task::common::HardwareTimedSampleClock(
        {.now = now_f,
         .sample_rate = sample_rate,
         .stream_rate = stream_rate,
         .k_p = k_p,
         .k_i = 0,
         .k_d = 0}
    );
    x::breaker::Breaker b;

    auto start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(now_v));
    now_v = x::telem::SECOND * 1;
    auto end = clock.end();
    ASSERT_EQ(end, x::telem::TimeStamp(x::telem::SECOND * 1));

    start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(x::telem::SECOND * 1));

    const auto skew = x::telem::MILLISECOND * 250;
    now_v = x::telem::SECOND * 2 + skew;
    end = clock.end();
    ASSERT_EQ(x::telem::TimeSpan(end.nanoseconds()), x::telem::SECOND * 2 + skew * k_p);
}

/// @brief it should reset clock state for new acquisition cycle.
TEST(TestSampleClock, testHardwareTimedSampleClockReset) {
    const auto sample_rate = x::telem::HERTZ * 2;
    const auto stream_rate = x::telem::HERTZ * 1;
    auto now_v = x::telem::SECOND * 5;
    auto now_f = [&now_v] { return x::telem::TimeStamp(now_v); };
    auto clock = driver::task::common::HardwareTimedSampleClock(
        {.now = now_f,
         .sample_rate = sample_rate,
         .stream_rate = stream_rate,
         .k_p = 0,
         .k_i = 0,
         .k_d = 0}
    );
    x::breaker::Breaker b;

    // First cycle
    auto start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(now_v));
    now_v += x::telem::SECOND * 1;
    auto end = clock.end();

    clock.reset();

    start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(now_v));
    now_v += x::telem::SECOND * 1;
    end = clock.end();
    ASSERT_EQ(end, x::telem::TimeStamp(now_v));
}

/// @brief it should apply PID correction when system runs slower than expected.
TEST(TestSampleClock, testHardwareTimedSampleClockPIDCorrection) {
    const auto sample_rate = x::telem::HERTZ * 2;
    const auto stream_rate = x::telem::HERTZ * 1;
    auto now_v = 0 * x::telem::SECOND;
    auto now_f = [&now_v] { return x::telem::TimeStamp(now_v); };
    auto clock = driver::task::common::HardwareTimedSampleClock(
        {.now = now_f,
         .sample_rate = sample_rate,
         .stream_rate = stream_rate,
         .k_p = 0.5,
         .k_i = 0.1,
         .k_d = 0.1}
    );
    x::breaker::Breaker b;

    // First sample - establish baseline
    const auto start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(now_v));

    // Simulate system running slower than expected
    now_v = x::telem::SECOND * 1 + x::telem::MILLISECOND * 100; // 100ms delay
    const auto end = clock.end();

    // The PID controller should attempt to correct for the delay
    // The exact value depends on the PID parameters, but it should be less than
    // the actual system time to compensate for the delay
    ASSERT_LT(end, x::telem::TimeStamp(now_v.nanoseconds()));
}

/// @brief it should maintain timing continuity across multiple consecutive cycles.
TEST(TestSampleClock, testHardwareTimedSampleClockConsecutiveCycles) {
    const auto sample_rate = x::telem::HERTZ * 2;
    const auto stream_rate = x::telem::HERTZ * 1;
    auto now_v = 0 * x::telem::SECOND;
    auto now_f = [&now_v] { return x::telem::TimeStamp(now_v); };
    auto clock = driver::task::common::HardwareTimedSampleClock(
        {.now = now_f,
         .sample_rate = sample_rate,
         .stream_rate = stream_rate,
         .k_p = 0,
         .k_i = 0,
         .k_d = 0}
    );
    x::breaker::Breaker b;

    // Test multiple consecutive cycles
    for (int i = 0; i < 3; i++) {
        auto start = clock.wait(b);
        ASSERT_EQ(start, x::telem::TimeStamp(now_v));
        now_v += x::telem::SECOND * 1; // Advance time by exactly one period
        auto end = clock.end();
        ASSERT_EQ(end, x::telem::TimeStamp(now_v));

        // Verify that the next start time matches the previous end time
        auto next_start = clock.wait(b);
        ASSERT_EQ(next_start, end);
    }
}

/// @brief it should limit back correction to prevent excessive timestamp adjustments.
TEST(TestSampleClock, testHardwareTimedSampleClockMaxBackCorrection) {
    const auto sample_rate = x::telem::HERTZ * 2;
    const auto stream_rate = x::telem::HERTZ * 1;
    auto now_v = 0 * x::telem::SECOND;
    auto now_f = [&now_v] { return x::telem::TimeStamp(now_v); };

    // Set a large P gain to ensure correction would exceed max if unconstrained
    constexpr double k_p = 2.0;
    constexpr double max_back_correction_factor = 0.1; // 10% of period
    auto clock = driver::task::common::HardwareTimedSampleClock(
        {.now = now_f,
         .sample_rate = sample_rate,
         .stream_rate = stream_rate,
         .k_p = k_p,
         .k_i = 0,
         .k_d = 0,
         .max_back_correction_factor = max_back_correction_factor}
    );
    x::breaker::Breaker b;

    const auto start = clock.wait(b);
    ASSERT_EQ(start, x::telem::TimeStamp(now_v));
    /// now is way way earlier than end.
    now_v = x::telem::MILLISECOND * 500;
    const auto end = clock.end();
    const auto expected_time = x::telem::TimeStamp(x::telem::MILLISECOND * 900);
    ASSERT_EQ(end, expected_time);
}

struct PIDTestParams {
    x::telem::Rate sample_rate;
    x::telem::Rate stream_rate;
    double k_p;
    double k_i;
    double k_d;
    // Timing error patterns
    x::telem::TimeSpan constant_offset;
    // Custom jitter function that takes cycle count and returns time offset
    std::function<x::telem::TimeSpan(int)> jitter_func;
    int n_cycles;
};

class HardwareTimedSampleClockPIDTest : public testing::TestWithParam<PIDTestParams> {
protected:
    x::telem::TimeSpan current_time = 0 * x::telem::SECOND;
    std::default_random_engine rng;
    int current_cycle = 0;

    [[nodiscard]] x::telem::TimeStamp now_func() const {
        return x::telem::TimeStamp(current_time);
    }

    void advance_system_time(const x::telem::TimeSpan expected_advance) {
        const auto &params = GetParam();
        auto actual_advance = expected_advance + params.constant_offset;

        if (params.jitter_func) { actual_advance += params.jitter_func(current_cycle); }

        current_time += actual_advance;
        current_cycle++;
    }
};

/// @brief it should converge timing errors using PID controller.
TEST_P(HardwareTimedSampleClockPIDTest, ConvergenceTest) {
    const auto &params = GetParam();

    auto clock = driver::task::common::HardwareTimedSampleClock(
        {.now = [this] { return this->now_func(); },
         .sample_rate = params.sample_rate,
         .stream_rate = params.stream_rate,
         .k_p = params.k_p,
         .k_i = params.k_i,
         .k_d = params.k_d}
    );

    x::breaker::Breaker b;
    std::vector<x::telem::TimeSpan> timing_errors;
    const int n_cycles = params.n_cycles;

    // Run the clock for multiple cycles
    for (int i = 0; i < n_cycles; i++) {
        const auto start = clock.wait(b);
        const auto expected_period = params.stream_rate.period();

        auto system_start = this->now_func();
        // Simulate system time advancing with error
        advance_system_time(expected_period);

        const auto system_end = this->now_func();
        const auto end = clock.end();

        // Calculate timing error (difference between expected and actual period)
        if (i > 0) {
            const auto sample_period = end - start;
            const auto actual_period = system_end - system_start;
            timing_errors.push_back(sample_period - actual_period);
        }
    }

    std::vector<long long> timing_errors_ns(timing_errors.size());
    for (size_t i = 0; i < timing_errors.size(); ++i)
        timing_errors_ns[i] = timing_errors[i].nanoseconds();

    // Analyze results
    // 1. Check if errors converge (later errors should be smaller)
    auto early_vs_late_count = n_cycles * 0.2;
    const auto early_avg_error = std::accumulate(
                                     timing_errors_ns.begin(),
                                     timing_errors_ns.begin() + early_vs_late_count,
                                     0
                                 ) /
                                 early_vs_late_count;

    const auto late_avg_error = std::accumulate(
                                    timing_errors_ns.end() - early_vs_late_count,
                                    timing_errors_ns.end(),
                                    0
                                ) /
                                early_vs_late_count;

    // System should improve over time
    EXPECT_LE(
        x::telem::TimeSpan(late_avg_error).abs(),
        x::telem::TimeSpan(early_avg_error).abs()
    );

    // 2. Check maximum error in steady state (last 20 samples)
    const auto max_steady_error = x::telem::TimeSpan(*std::max_element(
        timing_errors_ns.end() - n_cycles * 0.2,
        timing_errors_ns.end()
    ));

    // Maximum steady-state error should be reasonable (e.g., < 5% of period)
    EXPECT_LT(max_steady_error.abs(), params.stream_rate.period() * 0.05);
}

// Define test parameters
INSTANTIATE_TEST_SUITE_P(
    PIDTests,
    HardwareTimedSampleClockPIDTest,
    testing::Values(
        // Test case 1: Fast stream rate, constant jitter
        PIDTestParams{
            .sample_rate = x::telem::HERTZ * 1000,
            .stream_rate = x::telem::HERTZ * 100,
            .k_p = 0.1,
            .k_i = 0.01,
            .k_d = 0.001,
            .constant_offset = x::telem::MILLISECOND * 1,
            .jitter_func =
                [](int cycle) {
                    return x::telem::TimeSpan(0); // No jitter
                },
            .n_cycles = 1000
        },
        // Test case 2: Slow stream rate with sinusoidal jitter
        PIDTestParams{
            .sample_rate = x::telem::HERTZ * 100,
            .stream_rate = x::telem::HERTZ * 10,
            .k_p = 0.2,
            .k_i = 0.05,
            .k_d = 0.01,
            .constant_offset = x::telem::MILLISECOND * 2,
            .jitter_func =
                [](int cycle) {
                    // Sinusoidal jitter with 1ms amplitude and 100-cycle period
                    return x::telem::TimeSpan(
                        static_cast<int64_t>(
                            std::sin(2 * M_PI * cycle / 100.0) *
                            x::telem::MILLISECOND.nanoseconds()
                        )
                    );
                },
            .n_cycles = 1000
        },
        // Test case 3: Aggressive PID parameters
        PIDTestParams{
            .sample_rate = x::telem::HERTZ * 500,
            .stream_rate = x::telem::HERTZ * 50,
            .k_p = 0.5,
            .k_i = 0.1,
            .k_d = 0.05,
            .constant_offset = x::telem::MILLISECOND * 1,
            .jitter_func =
                [](int cycle) {
                    return x::telem::TimeSpan(0); // No jitter
                },
            .n_cycles = 1000
        },
        // Test case 4: Very slow rate with minimal correction
        PIDTestParams{
            .sample_rate = x::telem::HERTZ * 50,
            .stream_rate = x::telem::HERTZ * 1,
            .k_p = 0.05,
            .k_i = 0.005,
            .k_d = 0.001,
            .constant_offset = x::telem::MILLISECOND * 5,
            .jitter_func =
                [](int cycle) {
                    return x::telem::TimeSpan(0); // No jitter
                },
            .n_cycles = 100
        },
        // Test case 5: High frequency with tight timing
        PIDTestParams{
            .sample_rate = x::telem::HERTZ * 2000,
            .stream_rate = x::telem::HERTZ * 200,
            .k_p = 0.1,
            .k_i = 0.01,
            .k_d = 0.000,
            .constant_offset = x::telem::MICROSECOND * 500,
            .jitter_func =
                [](int cycle) {
                    return x::telem::TimeSpan(0); // No jitter
                },
            .n_cycles = 20000
        },
        // Test case 6: Steady then sudden jitter
        PIDTestParams{
            .sample_rate = x::telem::HERTZ * 1000,
            .stream_rate = x::telem::HERTZ * 100,
            .k_p = 0.3,
            .k_i = 0.02,
            .k_d = 0.05,
            .constant_offset = x::telem::MICROSECOND * 100,
            .jitter_func =
                [](int cycle) {
                    if (cycle < 10000) return x::telem::TimeSpan(0);
                    static std::random_device rd;
                    static std::mt19937 gen(rd());
                    static std::uniform_int_distribution<int64_t> dist(
                        -80 * x::telem::MICROSECOND.nanoseconds(),
                        80 * x::telem::MICROSECOND.nanoseconds()
                    );
                    return x::telem::TimeSpan(dist(gen));
                },
            .n_cycles = 15000
        }
    )
);

/// @brief it should generate evenly spaced timestamps for single index channel.
TEST(TestCommonReadTask, testGenerateIndexDataSingleIndex) {
    x::telem::Frame fr;
    fr.reserve(2); // 1 data channel + 1 index
    fr.emplace(1, x::telem::Series(x::telem::FLOAT64_T, 3)); // Data channel
    fr.emplace(2, x::telem::Series(x::telem::TIMESTAMP_T, 3)); // Index channel

    const std::set<synnax::channel::Key> index_keys = {2};
    const auto start = x::telem::TimeStamp(1000);
    const auto end = x::telem::TimeStamp(4000);
    constexpr size_t n_read = 3;
    constexpr size_t offset = 1; // Index starts after data channel

    driver::task::common::generate_index_data(
        fr,
        index_keys,
        start,
        end,
        n_read,
        offset
    );

    // Check index values are evenly spaced
    EXPECT_EQ(fr.series->at(1).at<x::telem::TimeStamp>(0), x::telem::TimeStamp(1000));
    EXPECT_EQ(fr.series->at(1).at<x::telem::TimeStamp>(1), x::telem::TimeStamp(2000));
    EXPECT_EQ(fr.series->at(1).at<x::telem::TimeStamp>(2), x::telem::TimeStamp(3000));
}

/// @brief it should generate identical timestamps for multiple index channels.
TEST(TestCommonReadTask, testGenerateIndexDataMultipleIndices) {
    x::telem::Frame fr;
    fr.reserve(3);
    fr.emplace(1, x::telem::Series(x::telem::FLOAT64_T, 3));
    fr.emplace(2, x::telem::Series(x::telem::TIMESTAMP_T, 3));
    fr.emplace(3, x::telem::Series(x::telem::TIMESTAMP_T, 3));

    const std::set<synnax::channel::Key> index_keys = {2, 3};
    const auto start = x::telem::TimeStamp(1000);
    const auto end = x::telem::TimeStamp(4000);
    constexpr size_t n_read = 3;
    constexpr size_t offset = 1;

    driver::task::common::generate_index_data(
        fr,
        index_keys,
        start,
        end,
        n_read,
        offset
    );

    for (size_t i = 1; i <= 2; i++) {
        EXPECT_EQ(
            fr.series->at(i).at<x::telem::TimeStamp>(0),
            x::telem::TimeStamp(1000)
        );
        EXPECT_EQ(
            fr.series->at(i).at<x::telem::TimeStamp>(1),
            x::telem::TimeStamp(2000)
        );
        EXPECT_EQ(
            fr.series->at(i).at<x::telem::TimeStamp>(2),
            x::telem::TimeStamp(3000)
        );
    }
}

/// @brief it should handle empty index keys without modification.
TEST(TestCommonReadTask, testGenerateIndexDataEmptyIndices) {
    x::telem::Frame fr;
    fr.reserve(1);
    fr.emplace(1, x::telem::Series(x::telem::FLOAT64_T, 3));

    const std::set<synnax::channel::Key> index_keys;
    const auto start = x::telem::TimeStamp(1000);
    const auto end = x::telem::TimeStamp(4000);
    constexpr size_t n_read = 3;
    constexpr size_t offset = 0;

    driver::task::common::generate_index_data(
        fr,
        index_keys,
        start,
        end,
        n_read,
        offset
    );
    EXPECT_EQ(fr.size(), 1);
}

/// @brief it should generate inclusive timestamps including end point.
TEST(TestCommonReadTask, testGenerateIndexDataInclusive) {
    x::telem::Frame fr;
    fr.reserve(2);
    fr.emplace(1, x::telem::Series(x::telem::FLOAT64_T, 3)); // Data channel
    fr.emplace(2, x::telem::Series(x::telem::TIMESTAMP_T, 3)); // Index channel

    const std::set<synnax::channel::Key> index_keys = {2};
    const auto start = x::telem::TimeStamp(1000);
    const auto end = x::telem::TimeStamp(3000);
    constexpr size_t n_read = 3;
    constexpr size_t offset = 1;
    constexpr bool inclusive = true;

    driver::task::common::generate_index_data(
        fr,
        index_keys,
        start,
        end,
        n_read,
        offset,
        inclusive
    );

    // Check inclusive spacing (end point included in equal intervals)
    EXPECT_EQ(fr.series->at(1).at<x::telem::TimeStamp>(0), x::telem::TimeStamp(1000));
    EXPECT_EQ(fr.series->at(1).at<x::telem::TimeStamp>(1), x::telem::TimeStamp(2000));
    EXPECT_EQ(fr.series->at(1).at<x::telem::TimeStamp>(2), x::telem::TimeStamp(3000));
}
