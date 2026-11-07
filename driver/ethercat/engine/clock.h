// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

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
}
