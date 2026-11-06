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
/// passes the reference clock midway between SYNC0 events: half a grid after an event,
/// where the grid is the largest span that divides the SYNC0 cycle and the period. No
/// phase is farther from every SYNC0 event.
class PhaseLock {
    /// @brief the share of the phase error that each cycle corrects.
    static constexpr double KP = 0.05;
    /// @brief the share of the summed phase error that each cycle corrects. It cancels
    /// the drift between the host and DC clocks.
    static constexpr double KI = 0.0001;
    /// @brief the largest remainder in nanoseconds that still divides. The period of a
    /// rate such as 300 Hz is not a whole number of nanoseconds, and a SYNC0 cycle made
    /// from that period is rounded.
    static constexpr double TOLERANCE = 1;
    double period;
    int64_t sync0;
    double grid;
    x::telem::TimeStamp event;
    double error = 0;
    double integral = 0;
    bool held = false;

    /// @brief returns the largest span that divides sync0 a whole number of times and
    /// divides period within TOLERANCE.
    static double common_grid(const double period, const double sync0) {
        double a = std::max(period, sync0);
        double b = std::min(period, sync0);
        while (true) {
            const double r = std::fmod(a, b);
            if (r <= TOLERANCE || b - r <= TOLERANCE) break;
            a = b;
            b = r;
        }
        return sync0 / std::round(sync0 / b);
    }

public:
    /// @param rate the bus rate.
    /// @param sync0 the SYNC0 cycle. Must be positive.
    /// @param event the time of a SYNC0 event.
    PhaseLock(
        const x::telem::Rate rate,
        const x::telem::TimeSpan sync0,
        const x::telem::TimeStamp event
    ):
        period(static_cast<double>(x::telem::SECOND.nanoseconds()) / rate.hz()),
        sync0(sync0.nanoseconds()),
        grid(common_grid(this->period, static_cast<double>(this->sync0))),
        event(event) {}

    /// @brief returns how far to move the start of the next cycle, at most a tenth of
    /// the period either way. A positive value delays it.
    /// @param dc the DC time at which the latest frame passed the reference clock.
    x::telem::TimeSpan update(const x::telem::TimeStamp dc) {
        const auto since = ((dc - this->event).nanoseconds() % this->sync0 +
                            this->sync0) %
                           this->sync0;
        this->error = std::fmod(static_cast<double>(since), this->grid) -
                      this->grid / 2;
        const auto limit = this->period / 10;
        const auto shift = -(KP * this->error + KI * (this->integral + this->error));
        if (std::abs(shift) < limit) this->integral += this->error;
        if (this->locked()) this->held = true;
        return x::telem::TimeSpan(
            static_cast<int64_t>(std::clamp(shift, -limit, limit))
        );
    }

    /// @brief true when the latest frame passed the reference clock within a quarter
    /// grid of the target phase.
    [[nodiscard]] bool locked() const {
        return std::abs(this->error) <= this->grid / 4;
    }

    /// @brief true when the lock held at an earlier frame and the latest frame is out
    /// of it.
    [[nodiscard]] bool lost() const { return this->held && !this->locked(); }
};
}
