// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <vector>

#include "x/cpp/telem/telem.h"

namespace arc {

/// @brief Simple timer for interval-based execution triggering.
///
/// The TimeWheel determines WHEN to call scheduler->next() based on a
/// GCD-computed base period from all intervals in the graph. It does NOT
/// check individual interval nodes - those self-check their periods.
///
/// This matches the runtime main loop pattern where we call next() on either:
/// - Time trigger: base period has elapsed
/// - Data trigger: new channel data arrived
class TimeWheel {
    uint64_t base_period_ns_;      ///< GCD of all interval periods (nanoseconds)
    telem::TimeStamp last_tick_;   ///< Last tick timestamp

public:
    /// @brief Construct TimeWheel with a base period.
    /// @param base_period_ns Base tick period in nanoseconds.
    explicit TimeWheel(uint64_t base_period_ns);

    /// @brief Calculate GCD-based base period from interval periods.
    ///
    /// Computes the greatest common divisor of all interval periods to
    /// determine the optimal base tick rate. This minimizes wasted cycles
    /// while ensuring all intervals can fire at their configured periods.
    ///
    /// @param periods Vector of interval periods in nanoseconds.
    /// @param min_period_ns Minimum allowed period (default 10ms).
    /// @return GCD of all periods, clamped to minimum.
    ///
    /// Example: periods=[100ms, 250ms, 1s] → GCD=50ms
    ///          TimeWheel ticks every 50ms:
    ///          - 100ms interval checks every 2 ticks
    ///          - 250ms interval checks every 5 ticks
    ///          - 1s interval checks every 20 ticks
    static uint64_t calculate_base_period(
        const std::vector<uint64_t> &periods,
        uint64_t min_period_ns = 10'000'000  // 10ms default minimum
    );

    /// @brief Check if base period has elapsed since last tick.
    ///
    /// This method is called in the runtime main loop to determine if
    /// scheduler->next() should be called for time-based execution.
    ///
    /// @return true if base period has elapsed (time trigger).
    /// @note RT-safe: Simple timestamp comparison, no allocations.
    bool should_tick();

    /// @brief Get base period in nanoseconds.
    /// @return Base tick period.
    uint64_t base_period_ns() const { return base_period_ns_; }
};

}  // namespace arc
