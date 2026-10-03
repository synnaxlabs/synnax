// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <chrono>
#include <cmath>
#include <thread>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/sleeper.h"
#include "x/cpp/telem/telem.h"

using hs_clock = std::chrono::steady_clock;
using nanos = std::chrono::nanoseconds;

namespace x::loop {
/// @brief Threshold below which high-resolution timing is used.
const telem::TimeSpan HIGH_RES_THRESHOLD = telem::Rate(200).period();
/// @brief Threshold below which medium-resolution timing is used.
const telem::TimeSpan MEDIUM_RES_THRESHOLD = telem::Rate(20).period();
/// @brief Span of one sleep step of a precise sleep.
const telem::TimeSpan RESOLUTION = (100 * telem::MICROSECOND);

class Timer {
public:
    Timer() = default;

    explicit Timer(const telem::TimeSpan &interval):
        interval(interval), last(hs_clock::now()) {}

    explicit Timer(const telem::Rate &rate):
        interval(rate.period()), last(hs_clock::now()) {}

    telem::TimeSpan elapsed(const hs_clock::time_point now) {
        if (!last_set) {
            last_set = true;
            return telem::TimeSpan::ZERO();
        }
        const auto elapsed = now - last;
        return telem::TimeSpan(elapsed);
    }

    std::pair<telem::TimeSpan, bool> wait() {
        const auto now = hs_clock::now();
        const auto elapsed = this->elapsed(now);
        if (elapsed > interval) {
            last = now;
            return {telem::TimeSpan(elapsed), false};
        }
        const auto remaining = interval - elapsed;
        const auto deadline = now + remaining.chrono();
        if (this->high_rate())
            this->precise_sleep(remaining);
        else
            std::this_thread::sleep_for(remaining.chrono());
        // Anchor to the deadline so a late wake shortens the next sleep instead of
        // stretching every period. An early wake keeps the wake time.
        last = std::min(deadline, hs_clock::now());
        return {telem::TimeSpan(elapsed), true};
    }

    std::pair<telem::TimeSpan, bool> wait(breaker::Breaker &breaker) {
        const auto now = hs_clock::now();
        const auto elapsed = this->elapsed(now);
        if (elapsed > interval) {
            last = now;
            return {telem::TimeSpan(elapsed), false};
        }
        const auto remaining = interval - elapsed;
        const auto deadline = now + remaining.chrono();
        if (this->high_rate())
            this->precise_sleep(remaining);
        else if (this->medium_rate())
            std::this_thread::sleep_for(remaining.chrono());
        else
            breaker.wait_for(remaining);
        last = std::min(deadline, hs_clock::now());
        return {telem::TimeSpan(elapsed), true};
    }

    /// @brief Fine-grained sleep using Welford's online algorithm for calibration.
    void precise_sleep(const telem::TimeSpan &dur) {
        this->sleep_while(dur, [] { return true; });
    }

    /// @brief Fine-grained sleep that ends early when the breaker stops.
    /// @returns false if the breaker stopped before or during the sleep.
    bool precise_sleep(const telem::TimeSpan &dur, const breaker::Breaker &breaker) {
        this->sleep_while(dur, [&breaker] { return breaker.running(); });
        return breaker.running();
    }

private:
    /// @brief sleeps precisely for dur, and ends early once running returns false.
    template<typename Running>
    void sleep_while(const telem::TimeSpan &dur, const Running &running) {
        const auto end = hs_clock::now() + dur.chrono();
        while (true) {
            if (!running()) return;
            const auto start = hs_clock::now();
            if (telem::TimeSpan(end - start) <= sleep_estimate_) break;
            this->sleeper_.sleep(RESOLUTION);
            const auto curr_end = hs_clock::now();
            const auto elapsed_ns = std::chrono::duration_cast<nanos>(curr_end - start)
                                        .count();
            const telem::TimeSpan delta = elapsed_ns - sleep_mean_;
            sleep_mean_ += delta / sleep_count_;
            sleep_M2_ += delta * (elapsed_ns - sleep_mean_);
            sleep_estimate_ = sleep_mean_ +
                              std::sqrt((sleep_M2_ / sleep_count_).nanoseconds());
            sleep_count_++;
        }
        while (end > hs_clock::now())
            if (!running()) return;
    }

    [[nodiscard]] bool high_rate() const { return interval < HIGH_RES_THRESHOLD; }

    [[nodiscard]] bool medium_rate() const { return interval < MEDIUM_RES_THRESHOLD; }

    telem::TimeSpan interval{};
    bool last_set = false;
    hs_clock::time_point last;
    /// @brief sleeps each step of a precise sleep.
    Sleeper sleeper_;

    /// @brief Welford's algorithm state: the most one step is expected to take. It
    /// starts at one step, so a sleep longer than one step measures a step.
    telem::TimeSpan sleep_estimate_ = RESOLUTION;
    /// @brief Welford's algorithm state: running mean of step durations.
    telem::TimeSpan sleep_mean_ = RESOLUTION;
    /// @brief Welford's algorithm state: sum of squared deviations.
    telem::TimeSpan sleep_M2_ = telem::TimeSpan::ZERO();
    /// @brief Welford's algorithm state: sample count.
    int64_t sleep_count_ = 1;
};
}
