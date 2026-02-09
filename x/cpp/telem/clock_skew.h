// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>

#include "x/cpp/telem/telem.h"

namespace x::telem {
/// @brief ClockSkewCalculator calculates and tracks clock skew between two systems
/// using a midpoint synchronization algorithm. This is useful for distributed
/// systems where clock synchronization is critical.
class ClockSkewCalculator {
public:
    /// @brief function to get the current timestamp, defaults to
    /// TimeStamp::now
    std::function<TimeStamp()> now = TimeStamp::now;
    /// @brief timestamp when the most recent measurement started
    TimeStamp local_start_t = TimeStamp(0);
    /// @brief running sum of all measured clock skews
    TimeSpan accumulated_skew = TimeSpan::ZERO();
    /// @brief number of measurements taken
    std::uint64_t n = 0;

    /// @brief default constructor
    ClockSkewCalculator() = default;

    /// @brief constructor with custom time source
    /// @param now Function that returns the current timestamp
    explicit ClockSkewCalculator(const std::function<TimeStamp()> &now): now(now) {}

    /// @brief starts a new clock skew measurement
    void start() { this->local_start_t = this->now(); }

    /// @brief completes a clock skew measurement
    /// @param remote_midpoint_t The timestamp from the remote system to compare
    /// against Uses the midpoint method: local_midpoint = start + (end - start)/2
    /// The skew is then calculated as: local_midpoint - remote_midpoint
    void end(const TimeStamp &remote_midpoint_t) {
        const auto local_end_t = this->now();
        const auto this_midpoint_t = this->local_start_t +
                                     (local_end_t - this->local_start_t) / 2;
        const auto skew = this_midpoint_t - remote_midpoint_t;
        this->accumulated_skew += skew;
        this->n++;
    }

    /// @brief returns the average clock skew across all measurements
    /// @return TimeSpan representing the average clock skew
    TimeSpan skew() const {
        if (this->n == 0) return TimeSpan::ZERO();
        return this->accumulated_skew / this->n;
    }

    /// @brief checks if the absolute value of the average clock skew exceeds a
    /// threshold
    /// @param threshold The maximum acceptable clock skew
    /// @return true if the absolute skew exceeds the threshold, false otherwise
    bool exceeds(const TimeSpan &threshold) const { return skew().abs() > threshold; }
};
}
