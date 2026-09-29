// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <utility>

#include "x/cpp/errors/errors.h"

namespace driver::can {
/// @brief the bit timing register ranges of a CAN controller. tseg1 includes the
/// propagation segment.
struct TimingLimits {
    /// @brief the smallest time segment 1 in time quanta.
    std::uint32_t tseg1_min = 1;
    /// @brief the largest time segment 1 in time quanta.
    std::uint32_t tseg1_max = 16;
    /// @brief the smallest time segment 2 in time quanta.
    std::uint32_t tseg2_min = 1;
    /// @brief the largest time segment 2 in time quanta.
    std::uint32_t tseg2_max = 8;
    /// @brief the largest synchronization jump width in time quanta.
    std::uint32_t sjw_max = 4;
    /// @brief the smallest bitrate prescaler.
    std::uint32_t brp_min = 1;
    /// @brief the largest bitrate prescaler.
    std::uint32_t brp_max = 64;
    /// @brief the step between valid prescalers.
    std::uint32_t brp_inc = 1;
};

/// @brief the bit timing register values for one bitrate.
struct Timing {
    /// @brief the bitrate prescaler.
    std::uint32_t brp = 0;
    /// @brief time segment 1 in time quanta, including the propagation segment.
    std::uint32_t tseg1 = 0;
    /// @brief time segment 2 in time quanta.
    std::uint32_t tseg2 = 0;
    /// @brief the synchronization jump width in time quanta.
    std::uint32_t sjw = 0;
};

/// @brief computes register values that produce a bitrate exactly, with the sample
/// point CiA 301 recommends: 87.5% up to 500 kbit/s, 80% up to 800 kbit/s, and 75%
/// above. It prefers the smallest prescaler, which gives the finest time quanta.
/// @param clock_hz the controller clock in hertz.
/// @param bitrate the bitrate in bits per second.
/// @param limits the controller's register ranges.
/// @returns CONFIG_ERROR when no register values produce the bitrate exactly.
[[nodiscard]] std::pair<Timing, x::errors::Error> compute_timing(
    std::uint32_t clock_hz,
    std::uint32_t bitrate,
    const TimingLimits &limits
);
}
