// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/time_wheel.h"

#include <algorithm>
#include <numeric>

namespace arc {

TimeWheel::TimeWheel(uint64_t base_period_ns)
    : base_period_ns_(base_period_ns), last_tick_(telem::TimeStamp::now()) {}

uint64_t TimeWheel::calculate_base_period(const std::vector<uint64_t> &periods,
                                           uint64_t min_period_ns) {
    if (periods.empty()) {
        return min_period_ns;
    }

    // Calculate GCD of all periods using std::gcd
    uint64_t gcd_result = periods[0];
    for (size_t i = 1; i < periods.size(); i++) {
        gcd_result = std::gcd(gcd_result, periods[i]);
    }

    // Clamp to minimum to prevent sub-millisecond ticks
    return std::max(gcd_result, min_period_ns);
}

bool TimeWheel::should_tick() {
    auto now = telem::TimeStamp::now();
    auto elapsed = now - last_tick_;

    if (elapsed.nanoseconds() >= static_cast<int64_t>(base_period_ns_)) {
        last_tick_ = now;
        return true;
    }

    return false;
}

}  // namespace arc
