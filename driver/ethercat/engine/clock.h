// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <utility>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

namespace driver::ethercat::engine {
/// @brief maps distributed clock times onto the host clock. Mapped times keep the low
/// jitter of the DC clock and follow the rate of the host clock.
class Clock {
    /// @brief a residual larger than this re-anchors the offset instead of filtering
    /// it.
    inline static const x::telem::TimeSpan STEP = x::telem::MILLISECOND * 100;
    /// @brief the number of updates over which the offset follows the host clock.
    static constexpr int64_t GAIN = 1000;
    x::telem::TimeSpan offset;
    bool anchored = false;

public:
    /// @brief returns the host time of an event at the given DC time.
    /// @param dc the DC time of the event.
    /// @param host the host time of the same event, read with jitter.
    x::telem::TimeStamp
    map(const x::telem::TimeStamp dc, const x::telem::TimeStamp host) {
        const auto residual = host - dc - this->offset;
        if (!this->anchored || residual.abs() > STEP) {
            this->offset += residual;
            this->anchored = true;
        } else
            this->offset += residual / GAIN;
        return dc + this->offset;
    }
};

/// @brief the host clocks the engine reads: steady time for the bus cycle and wall time
/// for frame stamps. A default HostClock reads the host.
class HostClock {
    x::telem::TimeStamp wall_start;
    hs_clock::time_point steady_start;
    std::optional<x::loop::Clock> steady;

public:
    HostClock() = default;

    /// @brief reads both times from source. The wall time starts at the host's wall
    /// time and then moves with source.
    explicit HostClock(x::loop::Clock source):
        wall_start(x::telem::TimeStamp::now()),
        steady_start(source.now()),
        steady(std::move(source)) {}

    /// @brief returns a timer for the bus cycle that waits on this clock.
    [[nodiscard]] x::loop::Timer timer(const x::telem::TimeSpan &interval) const {
        if (!this->steady) return x::loop::Timer(interval);
        return {interval, *this->steady};
    }

    /// @brief returns the wall time.
    [[nodiscard]] x::telem::TimeStamp now() const {
        if (!this->steady) return x::telem::TimeStamp::now();
        return this->wall_start +
               x::telem::TimeSpan(this->steady->now() - this->steady_start);
    }
};
}
