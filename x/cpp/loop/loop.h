// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include "x/cpp/telem/telem.h"
#include "glog/logging.h"
#include "x/cpp/breaker/breaker.h"

using hs_clock = std::chrono::high_resolution_clock;
using nanos = std::chrono::nanoseconds;

namespace loop {
const telem::TimeSpan HIGH_RES_THRESHOLD = telem::Rate(200).period();
const telem::TimeSpan MEDIUM_RES_THRESHOLD = telem::Rate(20).period();

const uint64_t RESOLUTION = (100 * telem::MICROSECOND).value;

/// @brief fine grain sleep function (using Welford's online algorithm)
/// @param dur the duration to sleep for in nanoseconds
inline void preciseSleep(const telem::TimeSpan &dur) {
    const auto ns = dur.chrono();
    const auto end = hs_clock::now() + ns;
    // static because variance in sleep duration is measured across each call
    // to compute a more accurate sleep time for the machine running the code
    const uint64_t nanoseconds = ns.count();
    static uint64_t estimate = RESOLUTION * 10; // overestimate innitially
    static uint64_t mean = RESOLUTION * 10;
    static uint64_t M2 = 0;
    static uint64_t count = 1;
    // use the welford's online algorithm to sleep for most of the time
    // updating the estimate as we go
    while (nanoseconds > estimate) {
        auto start = hs_clock::now();
        std::this_thread::sleep_for(std::chrono::nanoseconds(RESOLUTION));
        const auto end = hs_clock::now();
        const auto elapsed = std::chrono::duration_cast<nanos>(end - start).count();
        const uint64_t delta = elapsed - mean;
        mean += delta / count;
        M2 += delta * (elapsed - mean);
        estimate = mean + 1 * std::sqrt(M2 / count);
        count++;
    }
    // busy wait for the last bit to ensure we sleep for the correct duration
    while (end > hs_clock::now());
}

class Timer {
public:
    Timer() = default;

    explicit Timer(
        const telem::TimeSpan &interval
    ) : interval(interval), last(std::chrono::high_resolution_clock::now()) {
    }

    explicit Timer(
        const telem::Rate &rate
    ) : interval(rate.period()), last(std::chrono::high_resolution_clock::now()) {
    }

    telem::TimeSpan elapsed(std::chrono::high_resolution_clock::time_point now) {
        if (!last_set) {
            last_set = true;
            return telem::TimeSpan(0);
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
        if (this->highRate()) preciseSleep(remaining);
        else std::this_thread::sleep_for(remaining.chrono());
        last = hs_clock::now();
        return {telem::TimeSpan(elapsed), true};
    }

    std::pair<telem::TimeSpan, bool> wait(breaker::Breaker &breaker) {
        const auto now = hs_clock::now();
        const auto elapsed = this->elapsed(now);
        if (elapsed > interval) {
            last = now;
            return {telem::TimeSpan(elapsed), false};
        }
        const auto remaining = interval - elapsed;;
        if (this->highRate()) preciseSleep(remaining);
        else if (this->mediumRate())
            std::this_thread::sleep_for(remaining.chrono());
        else breaker.wait_for(remaining);
        last = hs_clock::now();
        return {telem::TimeSpan(elapsed), true};
    }

private:
    [[nodiscard]] bool highRate() const { return interval < HIGH_RES_THRESHOLD; }

    [[nodiscard]] bool mediumRate() const {
        return interval < MEDIUM_RES_THRESHOLD;
    }

    telem::TimeSpan interval{};
    bool last_set = false;
    std::chrono::time_point<std::chrono::high_resolution_clock> last;
};
}
