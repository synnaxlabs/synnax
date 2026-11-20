// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <cstdint>
#include <numeric>
#include <utility>
#include <vector>

#include "driver/ethercat/dc/dc.h"

namespace driver::ethercat::dc {
namespace {
/// @brief returns the fraction with the smallest denominator in [lo, hi], as a
/// numerator and denominator. Requires 0 < lo <= hi.
std::pair<double, double> simplest_fraction(const double lo, const double hi) {
    if (std::ceil(lo) <= hi) return {std::ceil(lo), 1};
    const double whole = std::floor(lo);
    const auto [num, den] = simplest_fraction(1 / (hi - whole), 1 / (lo - whole));
    return {(whole * num) + den, num};
}
}

double
grid(const x::telem::TimeSpan period, const std::vector<x::telem::TimeSpan> &cycles) {
    int64_t gcd = 0;
    for (const auto &c: cycles)
        gcd = std::gcd(gcd, c.nanoseconds());
    // The period of a rate such as 300 Hz is rounded to whole nanoseconds, so it can
    // miss a multiple of a SYNC0 cycle by a nanosecond. gcd / n divides the period
    // within a nanosecond when period / gcd is within 1 / gcd of a fraction m / n.
    const auto p = static_cast<double>(period.nanoseconds());
    const auto g = static_cast<double>(gcd);
    return g / simplest_fraction((p - 1) / g, (p + 1) / g).second;
}
}
