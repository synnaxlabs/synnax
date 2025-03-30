// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <random>

/// external
#include "gtest/gtest.h"

/// internal
#include "driver/task/common/sample_clock.h"

/// @brief it should correctly use the system clock to time samples.
TEST(TestSampleClock, testSoftwareTimedSampleClock) {
    auto clock = common::SoftwareTimedSampleClock(telem::HZ * 250);
    auto now = telem::TimeStamp::now();
    breaker::Breaker b;
    const auto start = clock.wait(b);
    EXPECT_GE(start, now);
    now = telem::TimeStamp::now();
    const auto end = clock.end();
    ASSERT_GE(end, now);
}

/// @brief it should correctly rely on steady sample spacing to time samples.
TEST(TestSampleClock, testHardwareTimedSampleClockNominal) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0,
        .k_i = 0,
        .k_d = 0
    });
    breaker::Breaker b;

    auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    now_v = telem::SECOND * 1;
    auto end = clock.end();
    ASSERT_EQ(end, telem::SECOND * 1);

    start = clock.wait(b);
    ASSERT_EQ(start, telem::SECOND * 1);
    now_v = telem::SECOND * 2;
    end = clock.end();
    ASSERT_EQ(end, telem::SECOND * 2);
}

TEST(TestSampleClock, testHardwareTimedSampleClockNowIsLater) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    constexpr double k_p = 0.1;
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = k_p,
        .k_i = 0,
        .k_d = 0
    });
    breaker::Breaker b;

    auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    now_v = telem::SECOND * 1;
    auto end = clock.end();
    ASSERT_EQ(end, telem::SECOND * 1);

    start = clock.wait(b);
    ASSERT_EQ(start, telem::SECOND * 1);

    const auto skew = telem::MILLISECOND * 250;
    now_v = telem::SECOND * 2 + skew;
    end = clock.end();
    ASSERT_EQ(
        telem::TimeSpan(end.nanoseconds()),
        telem::SECOND * 2 + skew * k_p
    );
}

TEST(TestSampleClock, testHardwareTimedSampleClockReset) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = telem::SECOND * 5;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0,
        .k_i = 0,
        .k_d = 0
    });
    breaker::Breaker b;

    // First cycle
    auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    now_v += telem::SECOND * 1;
    auto end = clock.end();

    // Reset clock
    clock.reset();

    // Verify reset state
    start = clock.wait(b);
    ASSERT_EQ(start, now_v); // Should use new current time after reset
    now_v += telem::SECOND * 1;
    end = clock.end();
    ASSERT_EQ(end, now_v);
}

TEST(TestSampleClock, testHardwareTimedSampleClockPIDCorrection) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0.5,
        .k_i = 0.1,
        .k_d = 0.1
    });
    breaker::Breaker b;

    // First sample - establish baseline
    const auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);

    // Simulate system running slower than expected
    now_v = telem::SECOND * 1 + telem::MILLISECOND * 100; // 100ms delay
    const auto end = clock.end();

    // The PID controller should attempt to correct for the delay
    // The exact value depends on the PID parameters, but it should be less than
    // the actual system time to compensate for the delay
    ASSERT_LT(end, telem::TimeStamp(now_v.nanoseconds()));
}

TEST(TestSampleClock, testHardwareTimedSampleClockConsecutiveCycles) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = 0,
        .k_i = 0,
        .k_d = 0
    });
    breaker::Breaker b;

    // Test multiple consecutive cycles
    for (int i = 0; i < 3; i++) {
        auto start = clock.wait(b);
        ASSERT_EQ(start, now_v);
        now_v += telem::SECOND * 1; // Advance time by exactly one period
        auto end = clock.end();
        ASSERT_EQ(end, now_v);

        // Verify that the next start time matches the previous end time
        auto next_start = clock.wait(b);
        ASSERT_EQ(next_start, end);
    }
}

TEST(TestSampleClock, testHardwareTimedSampleClockMaxBackCorrection) {
    const auto sample_rate = telem::HZ * 2;
    const auto stream_rate = telem::HZ * 1;
    auto now_v = 0 * telem::SECOND;
    auto now_f = [&now_v] { return telem::TimeStamp(now_v); };

    // Set a large P gain to ensure correction would exceed max if unconstrained
    constexpr double k_p = 2.0;
    constexpr double max_back_correction_factor = 0.1; // 10% of period
    auto clock = common::HardwareTimedSampleClock({
        .now = now_f,
        .sample_rate = sample_rate,
        .stream_rate = stream_rate,
        .k_p = k_p,
        .k_i = 0,
        .k_d = 0,
        .max_back_correction_factor = max_back_correction_factor
    });
    breaker::Breaker b;

    const auto start = clock.wait(b);
    ASSERT_EQ(start, now_v);
    /// now is way way earlier than end.
    now_v = telem::MILLISECOND * 500;
    const auto end = clock.end();
    const auto expected_time = telem::TimeStamp(telem::MILLISECOND * 900);
    ASSERT_EQ(end, expected_time);
}

struct PIDTestParams {
    telem::Rate sample_rate;
    telem::Rate stream_rate;
    double k_p;
    double k_i;
    double k_d;
    // Timing error patterns
    telem::TimeSpan constant_offset; // Constant timing offset
    telem::TimeSpan jitter; // Random timing variation
};

class HardwareTimedSampleClockPIDTest : public testing::TestWithParam<PIDTestParams> {
protected:
    telem::TimeSpan current_time = 0 * telem::SECOND;
    std::default_random_engine rng;

    [[nodiscard]] telem::TimeStamp now_func() const {
        return telem::TimeStamp(current_time);
    }

    // Simulates system time with error patterns
    void advance_system_time(const telem::TimeSpan expected_advance) {
        const auto &params = GetParam();

        auto actual_advance = expected_advance +
                              telem::TimeSpan(
                                  params.constant_offset * telem::MILLISECOND);

        if (params.jitter > 0) {
            std::normal_distribution<double> jitter(0, static_cast<double>(params.jitter.nanoseconds()));
            actual_advance += telem::TimeSpan(jitter(rng));
        }

        current_time += actual_advance;
    }
};

TEST_P(HardwareTimedSampleClockPIDTest, ConvergenceTest) {
    const auto &params = GetParam();

    auto clock = common::HardwareTimedSampleClock({
        .now = [this] { return this->now_func(); },
        .sample_rate = params.sample_rate,
        .stream_rate = params.stream_rate,
        .k_p = params.k_p,
        .k_i = params.k_i,
        .k_d = params.k_d
    });

    breaker::Breaker b;
    std::vector<telem::TimeSpan> timing_errors;
    constexpr int n_cycles = 50; // Test over 50 cycles to observe convergence

    // Run the clock for multiple cycles
    for (int i = 0; i < n_cycles; i++) {
        const auto start = clock.wait(b);
        const auto expected_period = params.stream_rate.period();

        // Simulate system time advancing with error
        advance_system_time(expected_period);

        const auto end = clock.end();

        // Calculate timing error (difference between expected and actual period)
        if (i > 0) {
            const auto actual_period = end - start;
            timing_errors.push_back(actual_period - expected_period);
        }
    }

    std::vector<int64_t> timing_errors_ns(timing_errors.size());
    for (size_t i = 0; i < timing_errors.size(); ++i)
        timing_errors_ns[i] = timing_errors[i].nanoseconds();

    // Analyze results
    // 1. Check if errors converge (later errors should be smaller)
    const auto early_avg_error = std::accumulate(
                                     timing_errors_ns.begin(),
                                     timing_errors_ns.begin() + 10,
                                     0.0) / 10.0;

    const auto late_avg_error = std::accumulate(
                                    timing_errors_ns.end() - 10,
                                    timing_errors_ns.end(),
                                    0.0) / 10.0;

    // System should improve over time
    EXPECT_LT(late_avg_error, early_avg_error);

    // 2. Check maximum error in steady state (last 20 samples)
    const auto max_steady_error = telem::TimeSpan(*std::max_element(
            timing_errors_ns.end() - 20,
            timing_errors_ns.end())
    );

    // Maximum steady-state error should be reasonable (e.g., < 5% of period)
    EXPECT_LT(max_steady_error, params.stream_rate.period() * 0.05);
}

// Define test parameters
INSTANTIATE_TEST_SUITE_P(
    PIDTests,
    HardwareTimedSampleClockPIDTest,
    testing::Values(
        // Test case 1: Fast stream rate, conservative PID
        PIDTestParams{
        .sample_rate = telem::HZ * 1000,
        .stream_rate = telem::HZ * 100,
        .k_p = 0.1,
        .k_i = 0.01,
        .k_d = 0.001,
        .constant_offset = telem::MILLISECOND * 1,
        .jitter = telem::MICROSECOND * 500
        },
        // Test case 2: Slow stream rate, aggressive PID
        PIDTestParams{
        .sample_rate = telem::HZ * 100,
        .stream_rate = telem::HZ * 10,
        .k_p = 0.5,
        .k_i = 0.1,
        .k_d = 0.05,
        .constant_offset = telem::MILLISECOND * 5,
        .jitter = telem::MILLISECOND * 2
        }
    )
);
