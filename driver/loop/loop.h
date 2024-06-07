// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include "client/cpp/synnax.h"
#include "glog/logging.h"

using hs_clock = std::chrono::high_resolution_clock;
using nanos = std::chrono::nanoseconds;

namespace loop {
const synnax::TimeSpan HIGH_RES_THRESHOLD = synnax::Rate(20).period();

inline void sleep(const std::chrono::nanoseconds ns) {
    const auto end = hs_clock::now() + ns;
    while (end > hs_clock::now());
}

const uint64_t RESOLUTION = (100 * synnax::MICROSECOND).value;

inline void preciseSleep(const std::chrono::nanoseconds ns) {
    const auto end = hs_clock::now() + ns;
    const uint64_t nanoseconds = ns.count();
    static uint64_t estimate = RESOLUTION * 10; // overestimate innitially
    static uint64_t mean = RESOLUTION * 10;
    static uint64_t M2 = 0;
    static uint64_t count = 1;
    while (nanoseconds > estimate) {
        auto start = hs_clock::now();
        sleep(std::chrono::nanoseconds(RESOLUTION));
        const auto end = hs_clock::now();
        const auto elapsed = std::chrono::duration_cast<nanos>(end - start).count();
        const uint64_t delta = elapsed - mean;
        mean += delta / count;
        M2 += delta * (elapsed - mean);
        estimate = mean + 1 * std::sqrt(M2 / count);
        count++;
    }
    while (end > hs_clock::now());
}

class Timer {
public:
    Timer() = default;

    explicit Timer(
        const synnax::TimeSpan &interval
    ): interval(interval), last(std::chrono::high_resolution_clock::now()) {
    }

    explicit Timer(
        const synnax::Rate &rate
    ): interval(rate.period()), last(std::chrono::high_resolution_clock::now()) {
    }

    std::pair<std::chrono::nanoseconds, bool> wait(breaker::Breaker &breaker) {
        const auto now = hs_clock::now();
        const auto elapsed = now - last;
        const auto interval_nanos = interval.nanoseconds();
        if (elapsed > interval_nanos) {
            last = now;
            return {elapsed, false};
        }
        const auto remaining = nanos(interval_nanos - elapsed);
        if (this->highRate()) preciseSleep(remaining);
        else breaker.waitFor(remaining);
        last = hs_clock::now();
        return {elapsed, true};
    }
private:
    [[nodiscard]] bool highRate() const { return interval < HIGH_RES_THRESHOLD; }
    synnax::TimeSpan interval{};
    std::chrono::time_point<std::chrono::high_resolution_clock> last;
};
}
