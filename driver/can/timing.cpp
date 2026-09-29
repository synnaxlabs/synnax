// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdlib>
#include <optional>
#include <string>

#include "driver/can/can.h"
#include "driver/can/timing.h"

namespace driver::can {
namespace {
/// @returns the recommended sample point of a bitrate in tenths of a percent.
std::uint32_t sample_point(const std::uint32_t bitrate) {
    if (bitrate <= 500000) return 875;
    if (bitrate <= 800000) return 800;
    return 750;
}

/// @returns register values that split total time quanta near the sample point, or
/// nullopt when the limits allow no split.
std::optional<Timing> split(
    const std::uint32_t brp,
    const std::uint32_t total,
    const std::uint32_t target,
    const TimingLimits &limits
) {
    const std::uint32_t sampled = (total * target + 500) / 1000;
    std::uint32_t tseg2 = total - std::max<std::uint32_t>(sampled, 2);
    tseg2 = std::clamp(tseg2, limits.tseg2_min, limits.tseg2_max);
    std::uint32_t tseg1 = total - 1 - tseg2;
    if (tseg1 < limits.tseg1_min || tseg1 > limits.tseg1_max) {
        tseg1 = std::clamp(tseg1, limits.tseg1_min, limits.tseg1_max);
        tseg2 = total - 1 - tseg1;
        if (tseg2 < limits.tseg2_min || tseg2 > limits.tseg2_max) return std::nullopt;
    }
    return Timing{
        .brp = brp,
        .tseg1 = tseg1,
        .tseg2 = tseg2,
        .sjw = std::min<std::uint32_t>(tseg2, limits.sjw_max),
    };
}
}

std::pair<Timing, x::errors::Error> compute_timing(
    const std::uint32_t clock_hz,
    const std::uint32_t bitrate,
    const TimingLimits &limits
) {
    const auto target = sample_point(bitrate);
    const auto step = std::max<std::uint32_t>(limits.brp_inc, 1);
    const std::uint32_t min_total = 1 + limits.tseg1_min + limits.tseg2_min;
    const std::uint32_t max_total = 1 + limits.tseg1_max + limits.tseg2_max;
    std::optional<Timing> best;
    std::uint32_t best_error = 0;
    for (std::uint32_t brp = std::max<std::uint32_t>(limits.brp_min, 1);
         bitrate > 0 && brp <= limits.brp_max;
         brp += step) {
        const std::uint64_t per_bit = static_cast<std::uint64_t>(brp) * bitrate;
        if (clock_hz % per_bit != 0) continue;
        const auto total = static_cast<std::uint32_t>(clock_hz / per_bit);
        if (total < min_total || total > max_total) continue;
        const auto timing = split(brp, total, target, limits);
        if (!timing.has_value()) continue;
        const auto actual = (1 + timing->tseg1) * 1000 / total;
        const auto error = static_cast<std::uint32_t>(
            std::abs(static_cast<int>(actual) - static_cast<int>(target))
        );
        if (best.has_value() && error >= best_error) continue;
        best = timing;
        best_error = error;
    }
    if (!best.has_value())
        return {
            {},
            {CONFIG_ERROR,
             "the adapter cannot produce a bitrate of " + std::to_string(bitrate) +
                 " bit/s from its " + std::to_string(clock_hz) + " Hz clock"}
        };
    return {*best, x::errors::NIL};
}
}
