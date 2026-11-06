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
#include <cmath>
#include <cstdint>

#include "x/cpp/telem/telem.h"

namespace driver::ethercat::engine {
/// @brief holds the bus cycle at a fixed phase of the distributed clock. Each frame
/// passes the reference clock half a period after a multiple of the period from origin,
/// as far as possible from the SYNC0 events.
class PhaseLock {
    /// @brief the share of the phase error that each cycle corrects.
    static constexpr double KP = 0.05;
    /// @brief the share of the summed phase error that each cycle corrects. It cancels
    /// the drift between the host and DC clocks.
    static constexpr double KI = 0.0001;
    x::telem::TimeSpan period;
    x::telem::TimeStamp origin;
    x::telem::TimeSpan error;
    double integral = 0;

public:
    PhaseLock(const x::telem::TimeSpan period, const x::telem::TimeStamp origin):
        period(period), origin(origin) {}

    /// @brief returns how far to move the start of the next cycle, at most a tenth of
    /// the period either way. A positive value delays it.
    /// @param dc the DC time at which the latest frame passed the reference clock.
    x::telem::TimeSpan update(const x::telem::TimeStamp dc) {
        const auto p = this->period.nanoseconds();
        const auto phase = ((dc - this->origin).nanoseconds() % p + p) % p;
        this->error = x::telem::TimeSpan(phase - p / 2);
        const auto e = static_cast<double>(this->error.nanoseconds());
        const auto limit = static_cast<double>(p) / 10;
        const auto shift = -(KP * e + KI * (this->integral + e));
        if (std::abs(shift) < limit) this->integral += e;
        return x::telem::TimeSpan(
            static_cast<int64_t>(std::clamp(shift, -limit, limit))
        );
    }

    /// @brief true when the latest frame passed the reference clock within a quarter
    /// period of the target phase.
    [[nodiscard]] bool locked() const { return this->error.abs() <= this->period / 4; }
};
}
